# Design: one render path and one apply rule for init, create api and update

- **Status:** Proposed — implementation waits for the maintainer to accept the behavior
  changes in [§4](#4-behavior-changes) and to answer [§7](#7-open-questions).
- **Date:** 2026-09-16
- **Author:** Chuan-Yen Chiang
- **Scope:** `pkg/plugins/crossplane/v2/` (`templates/engine/`, `scaffold/`, `createapi.go`,
  `update.go`). It follows the architecture review's R2 ("Render/Apply").
  The review accepted unified assembly but deferred the apply-policy layer until
  someone wrote this design.

All `file:line` references point at the state PR #161 merges into main (after
Tasks 8, 10, 11, 12 and 13). Kubebuilder references are to module `sigs.k8s.io/kubebuilder/v4@v4.15.0`
(the version in `go.mod`). Line numbers there change whenever that dependency is bumped.

## 1. Summary

Today, three pieces of code decide *what* to render, and two different rules decide *whether
a file is written*. This document proposes:

```go
// package engine — assembly only; it writes nothing but the filesystem it is handed.
func Render(dst machinery.Filesystem, cfg config.Config, flavor core.Flavor,
    upjet *core.UpjetSettings, resources []resource.Resource, scope Scope, opts ...Option) error

// package scaffold — the only code that decides what reaches the user's tree.
func Apply(src, dst afero.Fs, policy Policy) (Result, error)
```

- **Render** is the only assembly. It runs `machinery.Scaffold.Execute` for one scope onto the
  filesystem it is given: an in-memory one under this proposal, the way `update` already does.
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
| `update.go:331-370` (`renderToMemFS`) | `GetInitTemplates(WithUpjet(meta.Upjet))` + `CoreGeneratorsFor` (`:339-352`) + `GetAPITemplates(WithForce(true), WithResource, WithUpjet)` for every kind (`:354-368`) | An in-memory filesystem, reconciled onto disk later (`:227`) |
| `templates/engine/render_test.go:77-135` (`renderProject`) | A test copy of init followed by `create api` for each kind | In-memory filesystem |

Task 8 removed the duplicated *choices*: `CoreGeneratorsFor` at `templates/engine/assembly.go:67-72`
and `DependenciesFor` at `templates/engine/gomod_generator.go:56-61`. The *sequences* are still
repeated. Each new per-flavor input still has to be threaded through every site by hand: Task 10
threaded `WithUpjet(meta.Upjet)` through `renderToMemFS` as the third copy (`update.go:339`, `:356`).
The render test also exercises its own copy of the sequence, not the production one.

### 2.2 Two write rules

**Rule A — machinery `IfExistsAction`, set per template.** It is used by `init`, `create api` and `create-test`.
`Scaffold.Execute` builds every file model first (`machinery/scaffold.go:129-166`), then `writeFile`
(`machinery/scaffold.go:511-551`) checks each file's own action:

| Action (`machinery/file.go:23-30`) | Declared by |
|---|---|
| `SkipFile` (zero value) | every discovered template by default. Also `--force` downgrades to it for user-owned bodies (`templates/engine/builders.go:103-107`, Task 11), and `NewGoModGenerator` (`templates/engine/gomod_generator.go:66`) uses it. |
| `OverwriteFile` | `--force` on tool-owned templates (`builders.go:93-99` → `product_base.go:92-99`), and the generators: `register_generators.go:128,156`, `upjet_generator.go:69`, `ownership_doc_generator.go:117` |
| `Error` | `ChainsawTestGenerator` (`chainsaw_generator.go:67`), used by `create-test` |

Machinery writes every file with mode `0644` (`machinery/scaffold.go:47`). Scripts therefore get their executable bit
afterwards, from `ExecutableBitStep` (`automation/steps.go:127-176`), which is wired into both init pipelines
(`automation/pipeline.go:55,70`).

**Rule B — `core.DecideWrite`, decided by what is on disk.** It is used by `update`.
`reconcile`/`applyFile` (`update.go:376-459`) pick an action from the *existing file's* header
(`core/ownership.go:58-67`): seed if absent, overwrite if headered, skip otherwise. They reject
paths outside the project (`update.go:406-411`) and write `core.FileMode(rel)` (`update.go:448`,
`core/filemode.go:34-39`).

The two rules disagree exactly where it matters:

| Situation | Rule A (`create api` today) | Rule B (`update`) |
|---|---|---|
| A kind's `wiring.go` exists but is stale, and `create api` runs again without `--force` | skipped (stale file stays) | overwritten |
| `apis/register.go` exists and the author removed its header | overwritten (generator declares `OverwriteFile`) | skipped (now user-owned) |
| A user-owned file the author deleted | `create api` does not touch other kinds | re-seeded for native; for upjet only recorded as unseeded (C7, `update.go:440`) |
| Script mode | `0644`, fixed later by an init pipeline step | `0755`, written directly |

The ownership contract (`docs/architecture.md` §6) is stated in terms of Rule B. Rule A only
approximates it. For generators, `IfExistsAction` also doubles as ownership *metadata*: the ownership doc
classifies a generator as tool-owned by `GetIfExistsAction() == machinery.OverwriteFile`
(`ownership_doc_generator.go:94-98`).

## 3. Proposal

### 3.1 Render

```go
// package engine — next to the factory, the generators, and the tests and golden maps
// that already exercise them. Render needs no afero write path of its own.
type Scope int

const (
    ScopeInit    Scope = iota // init templates + generators(no resources) + go.mod seed
    ScopeKind                 // the created kind's per-kind templates (last in resources) + generators(all resources)
    ScopeProject              // init templates + generators(all resources) + every kind's per-kind templates
)

// Render assembles one scope and executes it onto dst: one machinery scaffold for the
// init templates and generators, then one per kind, as renderToMemFS does today.
func Render(dst machinery.Filesystem, cfg config.Config, flavor core.Flavor,
    upjet *core.UpjetSettings, resources []resource.Resource, scope Scope, opts ...Option) error
```

`ScopeKind` is what `create api` renders today. Option B (§5) needs it, and so does the
[Q1](#7-open-questions) fallback, so it is not speculative. `opts` reuses the engine's existing
`Option` (`WithForce`), so option B can keep passing the user's `--force`.

- `resources` is passed explicitly. `create api` must include the kind it is creating, and
  `PostScaffold` persists that kind only later (`createapi.go:165-169`, `:188-194`).
- `upjet` carries what the caller has. For `init` that is the full settings. For `create api` and
  `update` it is what PROJECT persists (`core/upjet.go:76`, `TerraformResourcePrefix`). The Terraform
  CLI version is not carried at all: `Configure` derives it from `pkg/versions`
  (`product_base.go:72-73`), so every render has it. For the kind
  being created, `create api` adds its `TerraformResource`, as it does today (`createapi.go:145-150`).
  Under `ScopeProject` the same settings reach every kind, so other kinds' `config/KIND/config.go`
  renders with the new kind's Terraform resource. That output is never written: it is user-owned and
  outside `inKind` (§3.3). This holds only while no *tool-owned* per-kind upjet template reads
  `.TerraformResource`; a test in step 4 of §6 pins it.
- Under Apply (option C) every caller passes a fresh memfs, and per-kind templates are rendered
  `WithForce(true)`, as `update.go:356` does today. Inside a fresh memfs the flag only decides
  whether a second kind in the same group re-renders `groupversion_info.go` or skips it. The output
  is identical either way, so force stops being a user-facing decision.
- Formatting stays where it is. `doTemplate` runs `imports.Process` on every `.go` output
  (`machinery/scaffold.go:232-239`), so unparsable Go fails inside Render, before anything
  touches the user's tree.
- No new context type and no flavor profile. Render is the body of `renderToMemFS` with a scope switch
  and a destination parameter.

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

`SeedOnly` is simply the policy that never overwrites; `OwnershipGate` is the one that does. Apply
lives in package `scaffold`, the write side, which `v2` already imports ([Q8](#7-open-questions)).
Apply is `reconcile`/`applyFile` moved out of `update.go`, and keeps what they already guarantee:
`checkContained`, `core.DecideWrite` as the base decision, and `core.FileMode` for the written mode.
It absorbs Task 10's `seedUserOwned` flag (`update.go:376`, `:416`, `:440`) instead of adding a
second copy: that flag becomes the `SeedUserOwned` predicate.

| On disk | Rendered | `SeedOnly` | `OwnershipGate(p)` |
|---|---|---|---|
| absent | tool-owned | seed | seed |
| absent | user-owned | seed | seed iff `p(rel)`, otherwise record as unseeded |
| exists, headered | any | skip | overwrite |
| exists, headerless | any | skip | skip |

### 3.3 Commands

Every command except `create-test` renders onto a fresh memfs and then applies it to `dst`. "Injected FS" below is the
`FS` field (an `afero.Fs`) of the `machinery.Filesystem` Kubebuilder passes to `Scaffold`.

| Command | Render | Apply | `dst` |
|---|---|---|---|
| `init` | `ScopeInit` | `SeedOnly` | Injected FS |
| `create api` | `ScopeProject`, after a flavor-aware `validateProject` (§4.1.7) | `OwnershipGate(inKind)`: seed user-owned only for paths of the kind being created | Injected FS |
| `update` | `ScopeProject` | `OwnershipGate(always)` for native; `OwnershipGate(never)` for upjet (C7) | `afero.NewOsFs()`, as `update.go:227` does today |
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
   every `groupversion_info.go`, Task 13's `hack/xp-provider-gen.mk`, and for upjet `config/provider.go`
   and friends, to the running generator's templates. Today only `update` does this.
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
   upjet templates read init-only fields: `project/Makefile.tmpl` (`TerraformProvider`,
   `TerraformProviderName`, `TerraformProviderRepo`, `TerraformProviderVersion`, `TerraformDocsPath`),
   `project/README.md.tmpl`, `AGENTS.md.tmpl`, `examples/providerconfig/providerconfig.yaml.tmpl`.
   Tool-owned upjet templates read only values every render has: `.Domain`, `.NamespacedDomain`,
   `.Repo`, `.ProviderName`, `.TerraformResourcePrefix`, and — since Task 13 moved the Terraform CLI
   version into the tool-owned `hack/xp-provider-gen.mk.tmpl` — `.TerraformVersion`, which `Configure`
   derives from `pkg/versions` rather than from the caller (`product_base.go:69-74`). Refreshing them
   from persisted settings is therefore safe. Task 1 made `NamespacedDomain` derive correctly in that case.
5. **`--force` becomes a no-op** (§3.4).
6. **Scripts are written `0755` directly** instead of being chmod-ed by a pipeline step (§4.4).
7. **`create api` becomes coupled to every kind already in PROJECT.**
   - Today `PreScaffold` validates only the new resource (`createapi.go:94-129`, `ValidateResource` at `:107`), and
     `Scaffold` renders only that kind's templates (`createapi.go:152-157`). Existing kinds reach the
     generators only as import lines in the registration files (`:165-173`).
   - Under `ScopeProject`, `create api B` renders every existing kind's full template set. So any existing
     kind makes `create api B` fail if its hand-edited PROJECT entry is invalid, if one of its templates fails
     to render, or if one of its output paths fails Apply's `checkContained`. Before, the command would have succeeded.
   - `update` already guards this by validating PROJECT before rendering (`update.go:95`, `validateProject` at
     `:274-303`). `create api` must run the same check, **flavor-aware**: upjet kinds need
     `validation.NewValidatorAllowingReservedKinds()`, the rule `create api` already applies to
     the new kind (`createapi.go:102-106`) and Task 10 gave `update` (`update.go:279-283`). It must run before
     `Render(ScopeProject)`, so that a broken PROJECT fails with "PROJECT is not usable: …" before anything is written.

### 4.2 `init`

- Scaffolding into an empty directory produces the same files, because `SeedOnly` and `SkipFile` are
  equivalent when nothing exists. Scaffold-diff proves this.
- In a non-empty directory, generator outputs that already exist (`apis/register.go`,
  `docs/ownership.md`, …) are now skipped. Today machinery overwrites them.

### 4.3 `update`

- No change beyond what Task 10 already shipped: the upjet flavor (`update.go:331-370`) and the C7
  no-seed rule (`update.go:227`, `:440`). Both are expressed as `Render(ScopeProject)` plus a
  `SeedUserOwned` predicate. `update` keeps its render scope, and `go.mod` stays out of it:
  `ScopeProject` does not include the go.mod seeder, just as `update.go:339-352` does not.

### 4.4 Cross-cutting

- **`machinery.Error` (create-test).** `create-test` is one generator on one path, not an assembly
  site, so it keeps calling `machinery.Scaffold` with `Error` (`chainsaw_generator.go:67`).
  Apply gets no `ErrorIfExists` policy, because nothing else would use it.
- **Formatting (machinery gofmt/imports).** Nothing moves. Render is `Scaffold.Execute` into a memfs,
  and formatting already happens there (`machinery/scaffold.go:232-239`). Apply copies bytes.
- **Kubebuilder's injected `machinery.Filesystem` is honored, not bypassed.**
  - Kubebuilder passes its filesystem to `Scaffold(fs)` (`plugin/subcommand.go:60`, `cli/cmd_helpers.go:503`).
  - The default is `afero.NewOsFs()` (`cli/cli.go:140`); tests can replace it with `cli.WithFilesystem` (`cli/options.go:166`).
  - Apply writes to the injected `machinery.Filesystem`'s `FS` (an `afero.Fs`), so the injection keeps working. `cmd/xp-provider-gen/main.go:98-106` does not override it.
  - What is bypassed is only machinery's *write step* (`machinery/scaffold.go:511-551`): its `IfExistsAction` switch and its fixed `0644` mode.
- **Path containment.** Apply's `checkContained` (`update.go:406-411`) now also guards `init` and
  `create api`, where today only `update` runs it. This is hardening, not a change for valid
  projects. It is also one of the ways an existing kind can fail `create api` (§4.1.7).
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
| **A. Status quo** | Keep three assembly sites and two write rules. | No work, no behavior change. | Every new per-flavor input is threaded by hand three times, and the render test checks a copy of the sequence rather than the production one. `create api` never refreshes stale tool-owned files. `--force` stays subtle. Task 10 already added the third upjet-aware copy. |
| **B. Share assembly only** | `engine.Render` (§3.1) with the caller's destination. `init` and `create api` render straight onto the injected filesystem with machinery's actions (`ScopeInit`, and `ScopeKind` + `WithForce(p.Force)`); `update` renders `ScopeProject` into memfs and reconciles as today. | Behavior-preserving and provable by scaffold-diff. S: Render lives in `templates/engine` beside `render_test.go`, its fixtures (`newTestConfig`, `fixtureResources`) and the golden maps (`ownership_test.go`), so the existing render tests switch to the production function with no fixture moves. One place threads `WithUpjet`, force and generators. | Two write rules remain, and so do the §2.2 disagreements. |
| **C. Full Render + Apply** | This proposal. | One write rule for `init`, `create api` and `update`, which is the ownership contract (`create-test` stays on machinery's `Error`, §4.4). `create api` keeps projects fresh. `--force` disappears. Script modes are handled in one place. | L. Changes `create api` behavior (§4.1), including its new dependence on every existing kind being valid (§4.1.7), and needs the §7 decisions. Adds a memfs round-trip to `init` and `create api` (negligible: `update` already does it). |

**Recommendation: B now, C after acceptance.** B is step 1 of C's migration (§6), so nothing is
wasted if C is accepted, and B stands on its own if it is not. This matches the review's split
("統一組裝值得做; Apply policy 延後").

## 6. Migration plan

**Prerequisites:** Tasks 10, 12 and 13, all merged by PR #161. Task 10 reworked `renderToMemFS`
and added the C7 seeding flag, Task 12 reworked finalize, and Task 13 added the tool-owned
`hack/xp-provider-gen.mk` that Apply must refresh. Apply absorbs their result; it does not fork it.

For every step, the "test first" item must exist and pass on the base commit before the refactor starts.
Scaffold-diff means `scaffold-diff.sh <base> <head>` exits 0 for both flavors.

| # | PR (type) | Change | Test first | Acceptance |
|---|---|---|---|---|
| 1 | `refactor:` | Add `engine.Render` (option B). `scaffold/init.go` calls `Render(fs, …, ScopeInit)`, `create api` calls `Render(fs, …, ScopeKind, WithForce(p.Force))`, and `renderToMemFS` calls `Render(mem, …, ScopeProject, WithForce(true))`. | `TestRenderAllTemplates` (`render_test.go:200`), `TestRenderUpjetPinsTerraformVersion` (`:263`) and `TestRenderUpjetGeneratorsWithoutResources` (`:279`) already cover both flavors, in the same package. In the same PR, `renderProject` (`:77-135`) becomes `Render(ScopeInit)` followed by `Render(ScopeKind)` per fixture kind (the production init + `create api` sequence), keeping its fixtures and the golden maps where they are. | Scaffold-diff identical; `make test` green. |
| 2 | `refactor:` | Extract `Apply` + `Policy` from `update.go:376-459` into `scaffold`; `update` calls `Apply(mem, OsFs, OwnershipGate(...))`. | `TestReconcile` (`update_test.go:238`), `TestReconcile_NestedSeed` (`:387`) and `TestReconcile_UpjetDoesNotSeedUserOwned` (`:199`) move with the code. Add a table test for §3.2's decision table, including the predicate and modes. | Scaffold-diff identical (init and `create api` untouched); the controller's `make e2e-upgrade` green. |
| 3 | `refactor:` | `init` → `Render(mem, …, ScopeInit)` + `Apply(SeedOnly)` into the injected FS. | Apply into an empty memfs gives the same path set as `renderProject`'s init stage; `.sh` files land `0755`, others `0644`. | Scaffold-diff identical, **plus a mode comparison** (`find . -type f -perm -u+x` in both trees, since `diff -r` ignores modes); `make e2e-native` and `make e2e-upjet` green (controller). |
| 4 | `feat:` (behavior change) | `create api` → flavor-aware `validateProject` (shared with `update`, §4.1.7) → `Render(mem, …, ScopeProject)` + `Apply(OwnershipGate(inKind))`; `--force` prints a deprecation notice. | (a) `init` → `create api A` → append a marker to A's `wiring.go` → delete A's `examples/<group>/<kind>.yaml` → `create api B`. Assert the marker is gone, A's `external.go` is unchanged, and the example is **not** re-seeded. (b) Re-running `create api A` without `--force` refreshes A's `wiring.go`. (c) Upjet: with persisted-only settings, delete `Makefile` and `AGENTS.md` → `create api` → neither is seeded, `hack/xp-provider-gen.mk` is refreshed and still pins the Terraform CLI version, no tool-owned output has an empty quoted Terraform value, and no tool-owned per-kind upjet template reads `.TerraformResource`. (d) A PROJECT whose *existing* kind is invalid (e.g. a hand-edited group with `..`) → `create api B` fails with "PROJECT is not usable" and writes nothing; on an upjet provider, an existing reserved-word kind is accepted. | Scaffold-diff identical. A fresh tree has nothing stale, so identical output is expected, and tests (a)–(d) are the real acceptance. The release note lists §4.1. |
| 5 | `refactor:` | Clean up per §7: remove `--force` (or keep it), remove `ExecutableBitStep` from the init pipelines (`automation/pipeline.go:55,70`) if Apply's modes cover it, and optionally switch the ownership doc to `IsToolOwned(body)`. | Step 3's mode test; `TestInitPipelines_ShareLeadingStepsAndFinalStep` updated deliberately. | Scaffold-diff identical + mode comparison; both e2e scripts green. |

Steps 1–3 are behavior-preserving and can merge whether or not step 4 is accepted.

## 7. Open questions

1. **Refresh scope of `create api`.** Accept that `create api` refreshes other kinds' and init-level
   tool-owned files (§4.1.1)? Accepting also means `create api` fails whenever any existing kind in
   PROJECT fails validation, rendering or path containment (§4.1.7). If not, `create api` uses
   `ScopeKind` with the same gate: it still fixes the stale re-run (§4.1.2), never touches other
   kinds, and stays independent of them.
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
7. **`ExecutableBitStep`.** Remove it from the init pipelines once Apply writes modes? (`update --adopt`
   already uses `core.FileMode`, `update.go:184`.)
8. **Package home for Apply.** Render belongs in `templates/engine` (§3.1). For Apply, the choice is
   `scaffold` (proposed: the write side, already imported by `v2`, and it keeps `core` free of afero)
   or `core`, next to `DecideWrite` and `FileMode`, which would add afero to `core`'s imports.

---

See [architecture.md](../architecture.md) for the current structure. §6 of that document covers the
ownership contract and §7 covers `update`.
