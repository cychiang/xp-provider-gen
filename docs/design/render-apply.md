# Design: one render path and one apply rule for init, create api and update

- **Status:** Proposed — implementation waits for the maintainer to accept the behavior
  changes in [§4](#4-behavior-changes) and to answer [§7](#7-open-questions).
- **Date:** 2026-09-16
- **Author:** Chuan-Yen Chiang
- **Scope:** `pkg/plugins/crossplane/v2/` (`scaffold/`, `createapi.go`, `update.go`,
  `templates/engine/`). It follows the architecture review's R2 ("Render/Apply").
  The review accepted unified assembly but deferred the apply-policy layer until
  someone wrote this design.

All `file:line` references point at the `refactor/architecture-review` branch after
Tasks 8 and 11. Kubebuilder references are to module `sigs.k8s.io/kubebuilder/v4@v4.15.0`
(the version in `go.mod`). Line numbers there change whenever that dependency is bumped.

## 1. Summary

Today, three pieces of code decide *what* to render, and two different rules decide *whether
a file is written*. This document proposes:

```go
// package scaffold
func Render(cfg config.Config, flavor core.Flavor, upjet *core.UpjetSettings,
    resources []resource.Resource, scope Scope) (afero.Fs, error)

func Apply(src, dst afero.Fs, policy Policy) (Result, error)
```

- **Render** is the only assembly. It runs `machinery.Scaffold.Execute` into an in-memory
  filesystem, the way `update` already does.
- **Apply** is the only rule for writing to the user's tree. It is the ownership gate
  `update` already uses, with two policies: `SeedOnly` for `init` and `OwnershipGate` for `create api` and `update`.

The recommended order is **Render first, Apply second**. Render alone is behavior-preserving
and can be proven with scaffold-diff. Apply changes what `create api` writes, so it lands only
after [§7](#7-open-questions) is answered.

## 2. Problem

### 2.1 Three assembly sites (plus one test mirror)

| Site | What it assembles | Where it writes |
|------|-------------------|-----------------|
| `scaffold/init.go:41-75` | `GetInitTemplates(WithUpjet(s.upjet))` + `CoreGeneratorsFor(flavor, cfg, nil)` + `NewGoModGenerator(repo, deps)` (`:49-66`) | Kubebuilder's injected `machinery.Filesystem` (`:44-47`, `:68`) |
| `createapi.go:131-186` | `GetAPITemplates(WithForce, WithResource, WithUpjet)` for the new kind (`:152-157`) + `CoreGeneratorsFor` over existing kinds plus the new one (`:165-173`) | Injected filesystem (`:137-141`, `:176`) |
| `update.go:350-388` (`renderToMemFS`) | `GetInitTemplates()` + `CoreGeneratorsFor` (`:358-371`) + `GetAPITemplates(WithForce(true), WithResource)` for every kind (`:373-386`) | An in-memory filesystem, reconciled onto disk later (`:249`) |
| `templates/engine/render_test.go:73-138` (`renderProject`) | A test copy of init followed by `create api` for each kind | In-memory filesystem |

Task 8 removed the duplicated *choices*: `CoreGeneratorsFor` at `templates/engine/assembly.go:67-72`
and `DependenciesFor` at `templates/engine/gomod_generator.go:56-61`. The *sequences* are still
repeated. Each new per-flavor input still has to be threaded through every site by hand; Task 10
will thread `WithUpjet(meta.Upjet)` through `renderToMemFS` a third time. The render test also
exercises its own copy of the sequence, not the production one.

### 2.2 Two write rules

**Rule A — machinery `IfExistsAction`, set per template.** It is used by `init`, `create api` and `create-test`.
`Scaffold.Execute` builds every file model first (`machinery/scaffold.go:129-166`), then `writeFile`
(`machinery/scaffold.go:511-551`) checks each file's own action:

| Action (`machinery/file.go:23-30`) | Declared by |
|---|---|
| `SkipFile` (zero value) | every discovered template by default. Also `--force` downgrades to it for user-owned bodies (`templates/engine/builders.go:103-107`, Task 11), and `NewGoModGenerator` (`templates/engine/gomod_generator.go:66`) uses it. |
| `OverwriteFile` | `--force` on tool-owned templates (`builders.go:93-99` → `product_base.go:86-91`), and the generators: `register_generators.go:128,156`, `upjet_generator.go:69`, `ownership_doc_generator.go:117` |
| `Error` | `ChainsawTestGenerator` (`chainsaw_generator.go:67`), used by `create-test` |

Machinery writes every file with mode `0644` (`machinery/scaffold.go:47`). Scripts therefore get their executable bit
afterwards, from `ExecutableBitStep` (`automation/steps.go:126-175`), which is wired into both init pipelines
(`automation/pipeline.go:55,70`).

**Rule B — `core.DecideWrite`, decided by what is on disk.** It is used by `update`.
`reconcile`/`applyFile` (`update.go:393-457`) pick an action from the *existing file's* header
(`core/ownership.go:58-67`): seed if absent, overwrite if headered, skip otherwise. They reject
paths outside the project (`update.go:419-424`) and write `core.FileMode(rel)` (`update.go:456`,
`core/filemode.go:34-39`).

The two rules disagree exactly where it matters:

| Situation | Rule A (`create api` today) | Rule B (`update`) |
|---|---|---|
| A kind's `wiring.go` exists but is stale, and `create api` runs again without `--force` | skipped (stale file stays) | overwritten |
| `apis/register.go` exists and the author removed its header | overwritten (generator declares `OverwriteFile`) | skipped (now user-owned) |
| A user-owned file the author deleted | `create api` does not touch other kinds | re-seeded (the C7 trade-off; Task 10 turns it off for upjet) |
| Script mode | `0644`, fixed later by an init pipeline step | `0755`, written directly |

The ownership contract (`docs/architecture.md` §6) is stated in terms of Rule B. Rule A only
approximates it. For generators, `IfExistsAction` also doubles as ownership *metadata*: the ownership doc
classifies a generator as tool-owned by `GetIfExistsAction() == machinery.OverwriteFile`
(`ownership_doc_generator.go:94-98`).

## 3. Proposal

### 3.1 Render

```go
// package scaffold — it already imports engine and core, and v2 already imports it.
type Scope int

const (
    ScopeInit    Scope = iota // init templates + generators(no resources) + go.mod seed
    ScopeKind                 // the created kind's per-kind templates (last in resources) + generators(all resources)
    ScopeProject              // init templates + generators(all resources) + every kind's per-kind templates
)

// Render assembles and renders one scope into a fresh in-memory filesystem.
func Render(cfg config.Config, flavor core.Flavor, upjet *core.UpjetSettings,
    resources []resource.Resource, scope Scope) (afero.Fs, error)
```

- `resources` is passed explicitly. `create api` must include the kind it is creating, and
  `PostScaffold` persists that kind only later (`createapi.go:165-169`, `:188-194`).
- `upjet` carries what the caller has. For `init` that is the full settings. For `create api` and
  `update` it is what PROJECT persists (`core/upjet.go:74`, `TerraformResourcePrefix`). For the kind
  being created, `create api` adds its `TerraformResource`, as it does today (`createapi.go:145-150`).
  Under `ScopeProject` the same settings reach every kind, so other kinds' `config/KIND/config.go`
  renders with the new kind's Terraform resource. That output is never written: it is user-owned and
  outside `inKind` (§3.3). This holds only while no *tool-owned* per-kind upjet template reads
  `.TerraformResource`; a test in step 4 of §6 pins it.
- Per-kind templates are always rendered `WithForce(true)`, as `update.go:374` does today. Inside a
  fresh memfs the flag only decides whether a second kind in the same group re-renders
  `groupversion_info.go` or skips it. The output is identical either way, so force stops being a
  user-facing decision.
- Formatting stays where it is. `doTemplate` runs `imports.Process` on every `.go` output
  (`machinery/scaffold.go:232-239`), so unparsable Go fails inside Render, before anything
  touches the user's tree.
- No new context type and no flavor profile. Render is the body of `renderToMemFS` with a scope switch.

### 3.2 Apply

```go
type Policy struct {
    // OverwriteToolOwned overwrites an existing file that carries the generated header.
    OverwriteToolOwned bool
    // SeedUserOwned reports whether a rendered headerless file that is absent
    // from dst is written. Rendered tool-owned files are always seeded when absent.
    SeedUserOwned func(rel string) bool
}

var SeedOnly = Policy{OverwriteToolOwned: false, SeedUserOwned: always}

func OwnershipGate(seedUserOwned func(rel string) bool) Policy {
    return Policy{OverwriteToolOwned: true, SeedUserOwned: seedUserOwned}
}

func Apply(src, dst afero.Fs, policy Policy) (Result, error) // Result: overwritten, seeded, skipped, unseeded
```

Apply is `reconcile`/`applyFile` moved out of `update.go`, and keeps what they already guarantee:
`checkContained`, `core.DecideWrite` as the base decision, and `core.FileMode` for the written mode.
It absorbs Task 10's `seedUserOwned` flag instead of adding a second copy: that flag becomes the
`SeedUserOwned` predicate.

| On disk | Rendered | `SeedOnly` | `OwnershipGate(p)` |
|---|---|---|---|
| absent | tool-owned | seed | seed |
| absent | user-owned | seed | seed iff `p(rel)`, otherwise record as unseeded |
| exists, headered | any | skip | overwrite |
| exists, headerless | any | skip | skip |

### 3.3 Commands

| Command | Render | Apply | `dst` |
|---|---|---|---|
| `init` | `ScopeInit` | `SeedOnly` | `fs.FS` from `Scaffold(fs machinery.Filesystem)` |
| `create api` | `ScopeProject` | `OwnershipGate(inKind)`: seed user-owned only for paths of the kind being created | `fs.FS` |
| `update` | `ScopeProject` | `OwnershipGate(always)` for native; `OwnershipGate(never)` for upjet (C7, Task 10) | `afero.NewOsFs()`, as `update.go:249` does today |
| `create-test` | unchanged: one generator on `machinery.Scaffold` (`createtest.go:90-100`) | unchanged (`machinery.Error`) | `afero.NewOsFs()` |

`inKind` is the set of output paths of `GetAPITemplates(WithResource(new))`. Those paths are known
without rendering, because `configureProduct` already resolves each product's path. Seeding only
those paths is the key choice for `create api`: its own new user-owned files are seeded with
complete data (including `--terraform-resource`), and it never re-creates files the author deleted
elsewhere, in either flavor.

`ScopeProject` for `create api` resolves the contradiction in the review (finding A7). Rendering only
"kind + generators" cannot refresh anything else, so a `create api` that should refresh other kinds'
tool-owned files must render the project. `ScopeKind` remains the fallback if [Q1](#7-open-questions) is
answered "no".

### 3.4 What `--force` means

After Task 11, `--force` means "also overwrite *tool-owned* files of this kind". Under
`OwnershipGate` every existing tool-owned file is overwritten anyway, so `--force` has no effect
left. The proposal keeps the flag for one release as a no-op that prints a deprecation notice,
then removes it ([Q3](#7-open-questions)).

## 4. Behavior changes

### 4.1 `create api`

1. **Refreshes other kinds' and init-level tool-owned files.** After a generator upgrade,
   `create api B` rewrites A's `wiring.go`, `cmd/provider/main.go`, `internal/provider/connector.go`,
   every `groupversion_info.go`, and for upjet `config/provider.go` and friends, to the running
   generator's templates. Today only `update` does this.
   - **Consequence 1 — version skew.** `create api` does not run `go get`, so refreshed files can need
     framework versions the project does not have yet. The native API-commit pipeline then fails at `make generate`
     (`automation/pipeline.go:99-110`). See [Q2](#7-open-questions).
   - **Consequence 2 — commits.** The fold commit stages the whole tree (`automation/git.go:98-110`), so these
     unrelated refreshes land in the "Add B managed resource" commit, or are amended into the
     Initial commit.
2. **Re-running `create api` for an existing kind refreshes its tool-owned files without `--force`.**
   Today they are skipped (`SkipFile`) unless `--force` is given. User-owned files stay untouched in both cases.
3. **A generator output whose on-disk copy lost its header stops being regenerated.** For example,
   a hand-edited `apis/register.go` with the header removed is skipped instead of overwritten.
   The new kind is then not registered and the build breaks later. Today machinery overwrites it
   unconditionally. See [Q4](#7-open-questions). `update` already behaves this way.
4. **User-owned files outside the kind being created are never seeded** (the `inKind` predicate).
   This matches today. It is listed because the unguarded alternative, `OwnershipGate(always)`, would
   move the C7 trade-off into `create api`:
   - native would re-create deleted files such as `test/behavior/<kind>-pause/`;
   - upjet would seed user-owned init files with **empty Terraform values**.

   The upjet case is the real hazard. `create api` has only persisted settings, and these user-owned
   upjet templates read init-only fields: `project/Makefile.tmpl` (`TerraformProvider*`,
   `TerraformDocsPath`, `TerraformVersion`), `project/README.md.tmpl`, `AGENTS.md.tmpl`,
   `examples/providerconfig/providerconfig.yaml.tmpl`. Tool-owned upjet templates read only
   `.Domain`, `.NamespacedDomain`, `.Repo`, `.ProviderName` and `.TerraformResourcePrefix`, so
   refreshing them from persisted settings is safe. Task 1 made `NamespacedDomain` derive
   correctly in that case.
5. **`--force` becomes a no-op** (§3.4).
6. **Scripts are written `0755` directly** instead of being chmod-ed by a pipeline step (§4.4).

### 4.2 `init`

- Scaffolding into an empty directory produces the same files, because `SeedOnly` and `SkipFile` are
  equivalent when nothing exists. Scaffold-diff proves this.
- In a non-empty directory, generator outputs that already exist (`apis/register.go`,
  `docs/ownership.md`, …) are now skipped. Today machinery overwrites them.

### 4.3 `update`

- No change beyond what Task 10 already introduces: the upjet flavor, and the C7 no-seed rule for upjet.
  `update` keeps its render scope, and `go.mod` stays out of it: `ScopeProject` does not include the
  go.mod seeder, just as `update.go:358-371` does not.

### 4.4 Cross-cutting

- **`machinery.Error` (create-test).** `create-test` is one generator on one path, not an assembly
  site, so it keeps calling `machinery.Scaffold` with `Error` (`chainsaw_generator.go:67`).
  Apply gets no `ErrorIfExists` policy, because nothing else would use it.
- **Formatting (machinery gofmt/imports).** Nothing moves. Render is `Scaffold.Execute` into a memfs,
  and formatting already happens there (`machinery/scaffold.go:232-239`). Apply copies bytes.
- **Kubebuilder's injected `machinery.Filesystem` is honored, not bypassed.**
  - Kubebuilder passes its filesystem to `Scaffold(fs)` (`plugin/subcommand.go:60`, `cli/cmd_helpers.go:503`).
  - The default is `afero.NewOsFs()` (`cli/cli.go:140`); tests can replace it with `cli.WithFilesystem` (`cli/options.go:166`).
  - Apply writes to `fs.FS`, so the injection keeps working. `cmd/xp-provider-gen/main.go:98-106` does not override it.
  - What is bypassed is only machinery's *write step* (`machinery/scaffold.go:511-551`): its `IfExistsAction` switch and its fixed `0644` mode.
- **File modes.** Apply writes `core.FileMode(rel)`, so `ExecutableBitStep` becomes redundant for init
  and `create api` ([Q7](#7-open-questions)). Scaffold-diff runs `diff -r`, which ignores modes, so the
  step that moves init onto Apply needs an explicit mode check.
- **Ownership doc.** It keeps classifying generators by `IfExistsAction`
  (`ownership_doc_generator.go:94-98`). Generators must keep declaring `OverwriteFile` even though Apply
  ignores the action, or the doc switches to `core.IsToolOwned(body)`, which is what the golden
  test already uses for templates.

## 5. Alternatives considered

| Option | What it is | Benefit | Cost |
|---|---|---|---|
| **A. Status quo** | Keep three assembly sites and two write rules. | No work, no behavior change. | Every new per-flavor input is threaded by hand three times, and the render test checks a copy of the sequence rather than the production one. `create api` never refreshes stale tool-owned files. `--force` stays subtle. Task 10 adds a third upjet-aware copy. |
| **B. Share assembly only** | One function returns the builders for a scope. `init` and `create api` still `Execute` onto the injected filesystem with machinery's actions; `update` still executes into memfs and reconciles. | Behavior-preserving and provable by scaffold-diff (S–M). One place threads `WithUpjet`, force and generators. `render_test.go` exercises production assembly instead of a copy. | Two write rules remain, and so do the §2.2 disagreements. |
| **C. Full Render + Apply** | This proposal. | One write rule, which is the ownership contract. `create api` keeps projects fresh. `--force` disappears. Script modes are handled in one place. | L. Changes `create api` behavior (§4.1) and needs the §7 decisions. Adds a memfs round-trip to `init` and `create api` (negligible: `update` already does it). |

**Recommendation: B now, C after acceptance.** B is step 1 of C's migration (§6), so nothing is
wasted if C is accepted, and B stands on its own if it is not. This matches the review's split
("統一組裝值得做; Apply policy 延後").

## 6. Migration plan

**Prerequisites:** Tasks 10, 12 and 13 are merged. All three touch `update.go`: Task 10 reworks
`renderToMemFS` and adds the C7 seeding flag, Task 12 reworks finalize, and Task 13 adds a tool-owned
Makefile fragment that Apply must seed. Apply absorbs their result; it does not fork it.

For every step, the "test first" item must exist and pass on the base commit before the refactor starts.
Scaffold-diff means `scaffold-diff.sh <base> <head>` exits 0 for both flavors.

| # | PR (type) | Change | Test first | Acceptance |
|---|---|---|---|---|
| 1 | `refactor:` | Extract the assembly into the `scaffold` package (option B): `init`, `create api`, `renderToMemFS` and `render_test.go`'s `renderProject` all call it. | `TestRenderAllTemplates` (`render_test.go:203`) and `TestRenderUpjetGeneratorsWithoutResources` (`:265`) already cover both flavors. They live in package `engine`, which `scaffold` imports, so in the same PR move them to package `scaffold` and have them call the production function (an `engine` test cannot import `scaffold` without an import cycle). | Scaffold-diff identical; `make test` green. |
| 2 | `refactor:` | Extract `Apply` + `Policy` from `update.go:393-457` into `scaffold`; `update` calls `Apply(mem, OsFs, OwnershipGate(...))`. | `TestReconcile` (`update_test.go:86`), `TestReconcile_NestedSeed` (`:235`) and Task 10's upjet no-seed test move with the code. Add a table test for §3.2's decision table, including the predicate and modes. | Scaffold-diff identical (init and `create api` untouched); the controller's `make upgrade-sim` green. |
| 3 | `refactor:` | `init` → `Render(ScopeInit)` + `Apply(SeedOnly)` into `fs.FS`. | Apply into an empty memfs gives the same path set as `renderProject`'s init stage; `.sh` files land `0755`, others `0644`. | Scaffold-diff identical, **plus a mode comparison** (`find . -type f -perm -u+x` in both trees, since `diff -r` ignores modes); `make e2e-test` and `make e2e-upjet` green (controller). |
| 4 | `feat:` (behavior change) | `create api` → `Render(ScopeProject)` + `Apply(OwnershipGate(inKind))`; `--force` prints a deprecation notice. | (a) `init` → `create api A` → append a marker to A's `wiring.go` → delete A's `examples/<group>/<kind>.yaml` → `create api B`. Assert the marker is gone, A's `external.go` is unchanged, and the example is **not** re-seeded. (b) Re-running `create api A` without `--force` refreshes A's `wiring.go`. (c) Upjet: with persisted-only settings, delete `Makefile` and `AGENTS.md` → `create api` → neither is seeded, no tool-owned output has an empty quoted Terraform value, and no tool-owned per-kind upjet template reads `.TerraformResource`. | Scaffold-diff identical. A fresh tree has nothing stale, so identical output is expected, and tests (a)–(c) are the real acceptance. The release note lists §4.1. |
| 5 | `refactor:` | Clean up per §7: remove `--force` (or keep it), remove `ExecutableBitStep` from the init pipelines (`automation/pipeline.go:55,70`) if Apply's modes cover it, and optionally switch the ownership doc to `IsToolOwned(body)`. | Step 3's mode test; `TestInitPipelines_ShareLeadingStepsAndFinalStep` updated deliberately. | Scaffold-diff identical + mode comparison; both e2e scripts green. |

Steps 1–3 are behavior-preserving and can merge whether or not step 4 is accepted.

## 7. Open questions

1. **Refresh scope of `create api`.** Accept that `create api` refreshes other kinds' and init-level
   tool-owned files (§4.1.1)? If not, `create api` uses `ScopeKind` with the same gate: it still fixes the
   stale re-run (§4.1.2) but never touches other kinds.
2. **Version skew.** When the running generator is newer than the version stamped in PROJECT
   (`projectmeta.go:33`, `Version`), should `create api`:
   - (a) refresh anyway;
   - (b) refuse, and tell the author to run `update` first; or
   - (c) apply the gate only to the kind being created (`ScopeKind`) until `update` has run?
3. **`--force`.** No-op with a deprecation notice for one release, then removal (proposed)? Or removal
   now? Or a new meaning, e.g. "also re-seed this kind's deleted user-owned files"?
4. **Generator outputs without a header.** Skip silently, as `update` does? Warn in the `Result`
   summary? Or always overwrite the deterministic generator outputs regardless of on-disk
   ownership (a special case in Apply)?
5. **User-owned seeding in `create api`.** Only the new kind's paths (proposed), or native re-seeding of
   every deleted user-owned file, like `update`?
6. **Idempotent writes.** Should Apply skip byte-identical overwrites, so that mtimes stay untouched and
   "Refreshed N file(s)" counts only real changes?
7. **`ExecutableBitStep`.** Remove it once Apply writes modes? `update --adopt` still writes `0644`
   (`update.go:206`) and would need the same rule.
8. **Package home.** `scaffold` (proposed, since `v2` already imports it) or `templates/engine`? Engine
   has no afero write path today, and `scaffold` would be a thin package.

---

See [architecture.md](../architecture.md) for the current structure. §6 of that document covers the
ownership contract and §7 covers `update`.
