# Modular Provider Layout — Design

- **Date:** 2026-07-31
- **Status:** Approved (design); ready for implementation planning
- **Scope:** Phase A of a two-phase program. Restructure the generated provider so
  crossplane-runtime plumbing is tool-owned and business logic is user-owned, behind a
  small set of named seams.
- **Builds on:** [2026-06-11 Provider Generation & Upgradability](2026-06-11-provider-generation-upgradability-design.md),
  which established the ownership header, the `update` command, and the dependency manifest.
- **Not in this spec:** Phase B (detect/list available upgrades, apply breaking changes).
  It gets its own spec once A lands.

## 1. Problem

`update` can refresh any file carrying the `DO NOT EDIT` header, and skips every file
without it. That contract works — but the *placement* of the line is wrong.

`internal/controller/<kind>/controller.go` is user-owned, correctly, because it holds
business logic. It also holds ~90 lines of pure crossplane-runtime plumbing:

- the `connector` struct and its `Connect` method,
- the `ProviderConfig` / `ClusterProviderConfig` resolution switch,
- `resource.CommonCredentialExtractor`,
- `usage.Track` and the concrete-kind type assertion that exists only to satisfy it.

Three consequences follow:

1. **Runtime changes cannot be delivered.** When crossplane-runtime changes ProviderConfig
   resolution or credential handling, `update` is structurally forbidden from touching the
   file that contains it. Every provider hand-patches it.
2. **The cost scales with kind count.** The plumbing is copied into every kind directory. A
   20-kind provider hand-patches 20 identical copies.
3. **There is nowhere to put provider-wide code.** Client construction is `newNoOpService`
   returning `interface{}`, buried per kind. Provider CLI flags cannot be added at all —
   `cmd/provider/main.go` is tool-owned and overwritten, so a user-added flag is lost on the
   next `update`.

No amount of upgrade tooling fixes this; the layout has to change first. That is why this
phase precedes Phase B.

## 2. Decisions

| Decision | Choice |
|----------|--------|
| Separation model | Unchanged — file-level ownership via the `DO NOT EDIT` header |
| Seam mechanism | Tool-owned caller + user-owned stub + one documented name |
| Connector | One shared, tool-owned `Connector` with the per-kind factory **injected** |
| Naming | Role-based, clean slate; crossplane vocabulary kept only where it earns its place |
| Compatibility | **Clean break.** No migration machinery, no dual-layout support |
| Provider config | Users extend `ProviderConfigSpec`; the spec reaches `NewClient` via `ClientConfig` |

**Clean break rationale.** There is no meaningful install base. Writing old-layout detection
for an empty population is speculative code the repo's KISS rule forbids. Providers generated
before this change are regenerated; that is one README line, not a subsystem.

## 3. Design

### 3.1 Target layout

| File | Owner | Contents |
|------|-------|----------|
| `cmd/provider/main.go` | tool | manager wiring; calls `provider.Flags`, `provider.Configure` |
| `internal/provider/connector.go` | tool | `Connector`, `Connect`, unexported `clientConfig()`, `ClientConfig` |
| `internal/provider/client.go` | **user** | `Client`, `NewClient` |
| `internal/provider/options.go` | **user** | `Flags`, `Configure` |
| `internal/controller/<kind>/wiring.go` | tool | `SetupGated`, `Setup`, reconciler construction |
| `internal/controller/<kind>/external.go` | **user** | `External`, `NewExternal`, the five methods, `ReconcilerOptions` |
| `apis/v1alpha1/types.go` | **user** | `ProviderConfigSpec` — extend freely |
| `apis/<grp>/<ver>/<kind>_types.go` | **user** | unchanged |
| `docs/ownership.md` | tool | generated ownership map (§6) |
| `AGENTS.md` | seed-once | user's own agent guidance; links to `docs/ownership.md` |

`controller.go` and `setup.go` are replaced by `external.go` and `wiring.go`. **Per-kind file
count is unchanged at two** — one the user owns, one the tool owns.

### 3.2 The seam contract

Six user-owned names the tool-owned code calls. Renaming any of them breaks the build; nothing
else in the generated tree is a stable contract.

| Name | Signature | Fallible |
|------|-----------|----------|
| `provider.Client` | user's type | — |
| `provider.NewClient` | `(ctx context.Context, cfg ClientConfig) (*Client, error)` | yes — real I/O |
| `provider.Flags` | `(app *kingpin.Application)` | no — registration cannot fail |
| `provider.Configure` | `(o *controller.Options) error` | yes — validation |
| `NewExternal` | `(c *provider.Client) *External` | no — struct literal |
| `ReconcilerOptions` | `(mgr ctrl.Manager, o controller.Options) ([]managed.ReconcilerOption, error)` | yes — fallible setup |

Error returns are principled, not uniform: a function returns `error` only where it can
genuinely fail.

**Why `ReconcilerOptions` and `Configure` must return errors.** `SetupGated` registers the
controller on the CRD readiness gate; the gate invokes `Setup` *after* the manager is running,
so a failure there has no caller to return to and becomes a `panic`. A hook that cannot report
failure forces users to panic themselves, producing an opaque stack trace from inside a gate
callback. Returning an error lets `Setup` wrap it with the kind name, so the operator sees
`cannot setup MyType controller: <message>`.

**Why `ReconcilerOptions` takes `(mgr, o)`.** Every useful reconciler option depends on one or
the other — `WithLogger` and `WithPollInterval` need `o`, initializers and connection
publishers need `mgr.GetClient()`. A no-argument hook could only return options that depend on
nothing, which is almost none of them.

**Why `ReconcilerOptions` ships now rather than later.** `wiring.go` is tool-owned, so adding
the call site later is free — but the *user's* `external.go` would then lack the function and
fail to compile. Adding it later is a breaking change for every provider; adding it now costs
four lines in a stub.

### 3.3 The injectable connector

`internal/provider/connector.go`, tool-owned, one copy per provider:

```go
// ClientConfig is everything the connector resolved from the ProviderConfig.
// Add fields to ProviderConfigSpec in apis/v1alpha1/types.go and read them
// here via Spec — nothing in between needs to change.
type ClientConfig struct {
	Spec        v1alpha1.ProviderConfigSpec // your spec, extend it freely
	Credentials []byte                      // extracted per Spec.Credentials
	Kube        client.Client               // for any lookup the tool did not do for you
}

// Connector resolves a managed resource's ProviderConfig into a Client, then
// hands it to a per-kind factory to build the ExternalClient.
type Connector struct {
	kube     client.Client
	usage    *resource.ProviderConfigUsageTracker
	external func(*Client) managed.ExternalClient // injected per kind
}

func NewConnector(mgr ctrl.Manager, external func(*Client) managed.ExternalClient) *Connector

func (c *Connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	m, ok := mg.(resource.ModernManaged)   // kind-agnostic — see below
	...
	c.usage.Track(ctx, m)
	cfg, err := c.clientConfig(ctx, m)     // unexported: PC/ClusterPC switch + extractor
	client, err := NewClient(ctx, cfg)     // user seam
	return c.external(client), nil         // user seam
}
```

Per-kind `wiring.go` injects the factory in one line:

```go
managed.WithExternalConnector(provider.NewConnector(mgr,
	func(c *provider.Client) managed.ExternalClient { return NewExternal(c) }))
```

**Why the whole connector is shared, not just credential extraction.** `Track` takes
`resource.ModernManaged`, not `resource.Managed` (verified against crossplane-runtime v2.3.3).
The concrete-kind type assertion in today's template exists *only* to satisfy it, so asserting
`ModernManaged` once makes steps 1–4 of `Connect` entirely kind-agnostic. Extracting only
credentials would have left the tracker, the assertion and the connector struct duplicated per
kind. `ProviderConfig` and `ClusterProviderConfig` share a single `ProviderConfigSpec` type, so
one value carries either.

**Accepted trade:** the `ModernManaged` assertion is a runtime check where the concrete-kind
assertion was closer to compile-time. In practice `managed.NewReconciler` is already bound to
the kind via `resource.ManagedKind(...)`, so the concrete assertion was a redundant safety net.

**Signature stability.** `clientConfig` is unexported and `ClientConfig` is tool-owned, so both
can change in a future crossplane-runtime bump without being a breaking change to any provider.
Only the six names in §3.2 are frozen.

### 3.4 Custom credentials need no new seam

The `Source` enum already includes `None`, for which `CommonCredentialExtractor` returns
`nil, nil`. A provider using pod identity, IRSA or any bespoke scheme sets `source: None`,
ignores `cfg.Credentials`, and builds its client however it likes — with `cfg.Spec` for its own
config fields and `cfg.Kube` for any lookup it needs. This is documented, not built.

`ClientConfig.Kube` is the weakest field in the design and was nearly cut. It is kept because
it is the difference between "custom credential handling is always possible" and "you are
blocked until the next generator release". One field on an existing struct is not complexity;
a dedicated seam for it would have been.

### 3.5 Provider options

`main.go` calls `provider.Flags(app)` before `app.Parse`, and
`kingpin.FatalIfError(provider.Configure(&o), ...)` after the standard options are assembled.

`client.go` and `options.go` are the same package and both user-owned, so a flag registered in
`Flags` can be stored in a package variable and read directly by `NewClient`. Flags reach
client construction with no plumbing — a consequence of putting provider-wide concerns in one
package rather than splitting `clients/` and `options/`.

### 3.6 What deliberately does not change

- **`SetupGated` and `Setup` both stay exported, with current behavior.** `SetupGated`
  registers the kind's GVK on the `customresourcesgate` readiness gate so controllers do not
  start before their CRDs are established. The name is load-bearing documentation of that
  deferral, and unexporting `Setup` would remove an escape hatch users cannot restore, since
  `wiring.go` is tool-owned.
- **Per-kind `wiring.go` stays flat and explicit.** Sharing its ~60 lines behind a generic
  `SetupController[T, L](...)` was considered and rejected: DRY governs hand-maintained sources
  of truth, not generated output. One template producing N files *is* one source of truth — the
  same reason `zz_generated.deepcopy.go` is not a DRY violation. Since `update` regenerates all
  N copies for free, a seven-parameter generic would trade readable, debuggable code for nothing.
  The connector was different precisely because it lived in a **user-owned** file and therefore
  could never be regenerated.
- **The ownership gate, `update`, and the dependency manifest** are untouched.

## 4. Ownership map

Unchanged rule: a file is tool-owned iff it carries
`// Code generated by xp-provider-gen. DO NOT EDIT.` New files slot into the existing buckets.

| Bucket | New members |
|--------|-------------|
| Tool-owned | `internal/provider/connector.go`, `internal/controller/<kind>/wiring.go`, `docs/ownership.md` |
| User-owned | `internal/provider/client.go`, `internal/provider/options.go`, `internal/controller/<kind>/external.go` |
| Seed-once | `AGENTS.md` |

`update` seeds absent files, so a provider gains new tool-owned files automatically.

**"User-owned" and "seed-once" are the same mechanism**, not two. Both are headerless, and
`core.DecideWrite` treats them identically: seed if absent, skip if present. The distinction is
only intent — user-owned files are ones we expect to be edited, seed-once files are ones we
merely decline to manage. No new gate behavior is introduced by this design.

**Header in a Markdown file.** `docs/ownership.md` is the first tool-owned non-Go file.
`core.IsToolOwned` does a substring match over the first 1024 bytes, so the marker must appear
literally. Wrap it in an HTML comment — `<!-- // Code generated by xp-provider-gen. DO NOT EDIT. -->` —
which satisfies the match while rendering invisibly.

## 5. Testing

**Unit — golden ownership test.** Enumerate every template found by
`templates/engine/autodiscovery.go` and assert its bucket. Anchoring to discovery rather than a
hand-maintained list means the test cannot drift from reality: adding a template forces a
deliberate ownership decision plus a one-line test change. This closes a real gap — today,
forgetting the header on a new tool-owned template silently makes it user-owned, so `update`
never refreshes it and nothing notices.

**e2e — extend `scripts/e2e-test.sh`,** reusing the existing Step U pattern rather than adding
a harness:

1. After `create api`, assert the new files exist:
   `internal/provider/{client,options,connector}.go`,
   `internal/controller/<kind>/{external,wiring}.go`, `docs/ownership.md`, `AGENTS.md`.
2. Assert headers: present on `connector.go`, `wiring.go`, `docs/ownership.md`; absent on
   `client.go`, `options.go`, `external.go`, `AGENTS.md`.
3. In Step U, append a marker to **all three** user-owned seam files, commit, run `update`,
   assert all three markers survive and `connector.go` / `wiring.go` still carry the header.
4. `make reviewable` and `make build` must pass throughout — which is also what proves the six
   seam names actually link.

## 6. Documentation

**`docs/ownership.md` — tool-owned, generated** by a small deterministic generator following
the existing `register_generators.go` / `gomod_generator.go` pattern, rendered from template
discovery. Because it is generated from the same data the golden test asserts against, the
published contract cannot drift from the enforced one.

**`AGENTS.md` — seed-once,** linking to `docs/ownership.md`. Splitting these matters: a
tool-owned `AGENTS.md` would be overwritten by `update`, destroying the user's own agent
guidance the first time they customized it. The generated half stays fresh; the user's half
stays theirs.

**Repo-side:** `docs/architecture.md` gains the new layout and the seam recipe (tool-owned
caller + user-owned stub + one documented name), so adding seam #5 is mechanical. `README.md`
gains one line stating that providers generated before this version must be regenerated.
Neither restates the ownership map — both link to it.

## 7. Out of scope (YAGNI)

- **Phase B** — upgrade detection/listing and breaking-change application.
- **Controller-builder customization** (extra `Watches`, custom predicates). Rarer than
  reconciler options, and the seam recipe makes adding it later a mechanical change.
- **Build and packaging seams** (Makefile, Dockerfile, `crossplane.yaml`). Still seed-once.
- **A generator-side config file** (`.xp-provider-gen.yaml`). Speculative configuration.
- **Old-layout detection or migration.** See §2.

## 8. Components affected

- `pkg/templates/files/internal/provider/{connector,client,options}.go.tmpl` — new.
- `pkg/templates/files/internal/controller/KIND/{wiring,external}.go.tmpl` — replace
  `setup.go.tmpl` and `controller.go.tmpl`.
- `pkg/templates/files/cmd/provider/main.go.tmpl` — call `Flags` before parse, `Configure` after.
- `pkg/templates/files/AGENTS.md.tmpl` — new, no header.
- `pkg/plugins/crossplane/v2/templates/engine/` — ownership-doc generator; verify
  `autodiscovery.go` classifies `internal/provider/*` as init/static templates.
- Golden ownership test (new).
- `scripts/e2e-test.sh` — assertions in §5.
- `docs/architecture.md`, `README.md`.

## 9. Risks

| Risk | Mitigation |
|------|------------|
| `autodiscovery.go` misclassifies the new `internal/provider/*` paths | Verify classification first; the golden ownership test makes any misclassification a test failure rather than a silent bug |
| `ModernManaged` assertion fails at runtime for an unusual type | The scaffold only generates modern managed resources; `managed.NewReconciler` is already kind-bound |
| Ownership-doc generator adds a moving part | It follows an established pattern in the same package and is covered by the golden test |
