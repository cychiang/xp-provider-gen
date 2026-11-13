# Building and upgrading a provider

How to scaffold a Crossplane provider with `xp-provider-gen`, write your logic in
the right places, and take framework upgrades without losing that logic.

This covers the **native** flavor, where you design the API and write the
reconcile logic yourself. Wrapping a Terraform provider instead? See
[docs/upjet-provider.md](upjet-provider.md), which explains why the two
flavors differ and what that means for adding a kind.

Prefer a walkthrough? [The tutorial](tutorial.md) builds a working provider
end-to-end on a local kind cluster.

If you only read one thing: **files carrying a generated-code header are rewritten
by `xp-provider-gen update`; everything else is yours forever.** Your provider ships
a generated `docs/ownership.md` listing exactly which is which.

## 1. Scaffold

```bash
xp-provider-gen init --domain=example.com --repo=github.com/you/provider-acme
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
```

`init` creates the project, wires the crossplane build submodule pinned at the commit
this generator version records as its `build_ref`, runs code generation, and leaves a
single clean commit. `create api` adds a kind and folds
into that commit until you make one of your own.

Both leave a clean working tree. If yours is dirty afterwards, that is a bug worth
reporting. Both commit as your git identity. `init --git-name` and `--git-email` replace
`user.name` and `user.email` for the commit `init` makes. A `create api` amend keeps that author
and records your git identity as the committer. A name or email that neither the flags nor
git's `user.name` and `user.email` supply falls back to
`xp-provider-gen <xp-provider-gen@example.com>`. Neither writes an identity into `.git/config`.

### The Go toolchain

`hack/xp-provider-gen.mk` exports `GOTOOLCHAIN ?= go<go_version>+auto` above the Go
build includes, where `<go_version>` is the release in `pkg/versions/dependencies.yaml`.
It caps `make` at that release, so a newer Go on your host, which the pinned
golangci-lint cannot type-check, does not break `make lint` or `make reviewable` with
`could not import`. The `go.mod` `go` directive stays free of a `toolchain` line.

Because of `?=`, any `GOTOOLCHAIN` already in your environment wins over that line,
including `auto`, which brings the failure back on a host whose Go is newer than the
linter.

**Work offline.** The `+auto` suffix lets Go download `go<go_version>` when it is not
installed, and an offline host cannot. To opt out of the line, install that Go release,
put it first on your `PATH`, and run `export GOTOOLCHAIN=local` before `make`. With
`GOTOOLCHAIN=local`, a host Go newer than the pin makes `make lint` fail with `typecheck`,
the failure the line prevents. A host Go older than the `go` directive fails earlier, with
`go.mod requires go >= <version>`.

`actions/setup-go` exports `GOTOOLCHAIN=local`, so a CI job needs its installed Go to match
the pin. Install it with `go-version-file: go.mod`, or clear the variable before `make`.

## 2. What you actually write

| You want to | Edit |
|---|---|
| Implement observe/create/update/delete | `internal/controller/<kind>/external.go` |
| Add spec/status fields to a kind | `apis/<group>/<version>/<kind>_types.go` |
| Build the API client from credentials | `internal/provider/client.go` |
| Add CLI flags, adjust controller options | `internal/provider/options.go` |
| Add settings to the ProviderConfig | `apis/v1alpha1/types.go` |

Everything else — the connector, ProviderConfig resolution, credential extraction,
controller registration, `main.go` — is generated and maintained for you.

### The external client

`external.go` holds only your business logic. The connector has already resolved
credentials and built your client by the time it is called:

```go
type External struct {
	client *provider.Client
}

func (e *External) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Instance)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotInstance)
	}

	got, err := e.client.GetInstance(ctx, meta.GetExternalName(cr))
	if kerrors.IsNotFound(err) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.Status.SetConditions(xpv2.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: got.Size == cr.Spec.ForProvider.Size,
	}, nil
}
```

### The API client

`client.go` turns a resolved ProviderConfig into whatever your API needs:

```go
type Client struct {
	http *http.Client
	base string
}

func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	var creds struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(cfg.Credentials, &creds); err != nil {
		return nil, errors.Wrap(err, "cannot parse credentials")
	}
	return &Client{http: newAuthedClient(creds.Token), base: cfg.Spec.Endpoint}, nil
}
```

Note `cfg.Spec.Endpoint` — that field does not exist until you add it.

### Adding settings to the ProviderConfig

`apis/v1alpha1/types.go` is yours. Add a field:

```go
type ProviderConfigSpec struct {
	Credentials ProviderCredentials `json:"credentials"`

	// Endpoint is the API base URL.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
}
```

Run `make generate`, and it is available as `cfg.Spec.Endpoint` in `NewClient`.
Nothing between the two needs changing — the connector passes the whole spec
through.

### Where credentials come from

A ProviderConfig is namespaced, and so are its credentials: `secretRef` names a
Secret in **the ProviderConfig's own namespace** (it has no `namespace` field).
That is deliberate — it keeps one tenant's config from reaching another's
Secrets. The available sources are `Secret`, `None` and `InjectedIdentity`.

For pod identity, IRSA or workload identity, set the credentials source to
`None`. `cfg.Credentials` is then empty, and you build the client from ambient
identity. `cfg.Kube` is available for any lookup the generator did not do for
you.

### CLI flags

`options.go` and `client.go` are the same package, so a flag reaches client
construction with no plumbing:

```go
var region = new(string)

func Flags(app *kingpin.Application) {
	region = app.Flag("region", "Target region.").Envar("REGION").Default("us-east-1").String()
}

func Configure(o *controller.Options) error {
	if *region == "" {
		return errors.New("region must not be empty")
	}
	return nil
}
```

`Flags` runs before parsing; `Configure` runs after, and returning an error aborts
startup with a clear message instead of failing later inside a controller.

### Per-kind reconciler options

`wiring.go` is generated, but it calls `ReconcilerOptions` from your `external.go`,
so per-kind tuning survives updates:

```go
func ReconcilerOptions(mgr ctrl.Manager, o controller.Options) ([]managed.ReconcilerOption, error) {
	return []managed.ReconcilerOption{
		managed.WithInitializers(managed.NewNameAsExternalName(mgr.GetClient())),
	}, nil
}
```

Return an error rather than panicking: controllers start from inside a CRD-readiness
gate callback, where a panic surfaces with no context.

## 3. Names you must not rename

Generated code calls these six by name (the full seam contract is in
[architecture.md §9](architecture.md#9-seams-the-modular-layout)). Renaming any of them breaks the build:

| Name | File |
|---|---|
| `Client`, `NewClient` | `internal/provider/client.go` |
| `Flags`, `Configure` | `internal/provider/options.go` |
| `NewExternal`, `ReconcilerOptions` | `internal/controller/<kind>/external.go` |

You can change anything else about them — add fields to `Client`, add helpers, split
files in the package. Only the names and signatures above are fixed.

## 4. Testing your provider

The scaffold seeds a complete e2e suite — `test/` plus a per-kind example under
`examples/` — all user-owned, seeded once and never touched by `update`:

- `examples/<group>/<kind>.yaml` per kind — both a usage example and
  [uptest](https://github.com/crossplane/uptest)'s lifecycle input via its
  `uptest.upbound.io/*` annotations: create → assert `Ready,Synced` → delete.
  Picked up by wildcard; adding kinds needs no wiring.
- `test/behavior/<kind>-pause/` per kind — a seed
  [chainsaw](https://kyverno.github.io/chainsaw/) test proving `crossplane.io/paused`
  actually stops reconciliation. Copy its pattern, or scaffold new tests with
  `xp-provider-gen create-test`.

| Command | What it proves | kind cluster | Lifecycle |
|---|---|---|---|
| `make e2e` | the full story: xpkg builds, Crossplane installs it, every kind's lifecycle passes, behavior tests pass | `<provider>-e2e` | recreated per run, **left running**; `make e2e-clean` removes it |
| `make test-integration` | fast loop: your reconcile logic, controller run from source | `<provider>-integration` | created and removed per run |
| `make test-behavior` | just the chainsaw tests | whatever kubectl points at | untouched |
| `make dev` / `dev-clean` | interactive development | `<provider>-dev` | explicit create/delete |

One name per purpose, printed in every message — none of these can ever touch a
cluster the project didn't create. `make e2e` and `make dev` do switch your
current kubectl context while running.

## 5. Upgrading

This is the payoff. When a new `xp-provider-gen` ships — a crossplane-runtime bump,
a fix to the controller wiring, a new framework feature:

```bash
git status                 # must be clean; update refuses otherwise
xp-provider-gen update
git diff                   # review
make reviewable
git add -A && git commit -m "chore: update provider core"
```

When `update` changes nothing, the last command reports `nothing to commit`.

After upgrading the generator, run `update` before `create api`: `create api` refuses when the
generator differs from the one that last updated the project.

`update` refuses outright if PROJECT declares an unknown `flavor:` value, or `flavor: upjet`
with no `upjet:` settings block. It refuses a PROJECT with no plugin block or an empty `flavor:` the same way.

`update` also refuses when the generator running is an older clean
release than the one that last updated this project, naming both versions; to downgrade
deliberately, edit `version:` under the `crossplane.go.kubebuilder.io` plugin in PROJECT,
commit it, then run `update` again.

`update` does six things:

1. Regenerates every tool-owned file from the current templates. A file whose content
   already matches is not rewritten: the summary counts it as `Unchanged`, and only files
   whose content actually changed count as `Refreshed`.
2. Seeds any file that is new in this version.
3. **Skips every user-owned file**, whether or not it has changed.
4. Removes any tracked, tool-owned file this version no longer produces — a
   template this generator retired since you last updated.
5. Raises `go.mod`'s `go` directive to the generator's Go version if yours is lower
   (a higher one you set is kept), then bumps the framework dependency versions via
   `go get`, leaving your own requires alone.
6. Runs `go mod tidy`, `make generate` and `make reviewable`, and refuses if any of
   them fail — this is why the diff also contains regenerated `zz_generated.*` and CRDs.

`update` prints counts by default; `update --verbose` also lists each file under
`Refreshed`, `Unchanged`, `Added` and `Kept` (user-owned) headings.

`update` never moves the `build` submodule. When yours is at a different commit than
`build_ref`, it prints one line naming both commits and the commands that move it; the
commit is yours to take or not.

It stops there deliberately — no commit — so `git diff` is your review surface.

**What you should see in that diff:** tool-owned files (including the build pipeline in
`hack/xp-provider-gen.mk`), `go.mod` / `go.sum` version lines, regenerated
`zz_generated.*` and CRDs, `PROJECT` (stamped with the generator version that just ran), and
occasionally a deleted tool-owned file the summary names.

**What you should never see:** `external.go`, `client.go`, `options.go`, any
`*_types.go`, `AGENTS.md`, or your `Makefile`. If one appears, that is a bug in the generator, not
something to work around — please report it with the diff. Never paste the generated-file
header line into a file of your own, even in a comment: step 3 and step 4 both key off that
one substring, so a file that merely quotes it reads as tool-owned and can be overwritten or
deleted the same as a real generated file.

If a tool-owned file lost its header by accident, `update` keeps it as yours, and the lint in
its closing `make reviewable` covers it like any file you wrote. To get the generated version
back instead, delete the file and commit the deletion, since `update` refuses a dirty working
tree. Then run `update`, which seeds the file again with its header.

If a step fails midway, the error message spells out the exact revert steps: `git reset
--hard` to restore modified tool-owned files, plus `rm -rf` for any file this run seeded (those
are untracked, so `git reset --hard` alone leaves them behind). That is why the clean-tree
precondition exists.

## 6. Where to look next

- `docs/ownership.md` **inside your provider** — the generated, always-accurate list
  of which files are yours.
- `AGENTS.md` inside your provider — a short orientation for humans and coding
  agents. It is yours; add project-specific guidance to it.
- [docs/architecture.md](architecture.md) — how the generator itself is built.
