# Architecture

`xp-provider-gen` is a CLI built on the **Kubebuilder v4 plugin system**. It scaffolds a
complete Crossplane provider from embedded templates, runs a post-scaffold automation
pipeline (git, module tidy, code generation), and can later **`update`** an existing
provider's tool-owned core without touching the user's business logic.

The code is organized into clearly separated layers:

```
cmd/xp-provider-gen/            CLI entry point (Kubebuilder CLI + the `update` command)
pkg/plugins/crossplane/v2/
├── plugin.go, init.go,         Plugin layer — subcommands (init, create api)
│   createapi.go, update.go     + the update / update --adopt command
├── core/                       Reusable building blocks (git, exec, config, ownership gate)
├── templates/engine/           Template discovery + deterministic generators
├── automation/                 Post-scaffold pipeline (steps + git operations)
└── validation/                 Input validation (domain, repo, group/version/kind)
pkg/templates/                  Embedded template filesystem (go:embed) + loader
pkg/versions/                   Dependency manifest (single source of truth for generated go.mod)
```

## 1. Entry point & command flow

`cmd/xp-provider-gen/main.go` constructs a Kubebuilder CLI, registers the Crossplane plugin,
and adds the standalone `update` command (Kubebuilder's plugin interface has no update hook):

```go
cli.New(
    cli.WithCommandName("crossplane-provider-gen"),
    cli.WithPlugins(&crossplanev2.Plugin{}),
    cli.WithDefaultPlugins(cfgv3.Version, &crossplanev2.Plugin{}),
    cli.WithExtraCommands(crossplanev2.NewUpdateCommand()),
)
```

Kubebuilder routes `init` and `create api` to the plugin's subcommands, each driven through
the standard lifecycle: `BindFlags` → `InjectConfig` → `PreScaffold` → `Scaffold` →
`PostScaffold`. `update` is driven by its own `cobra` command.

## 2. Plugin layer (`pkg/plugins/crossplane/v2/`)

- **`plugin.go`** — `Plugin` implements Kubebuilder's `plugin.Full`, advertises config v3 /
  plugin v2, returns the init and create-api subcommands.
- **`init.go`** — binds `--domain`, `--repo`, `--git-name`, `--git-email`; validates inputs;
  resolves git author (CLI flags > system git config > defaults); scaffolds the init + static
  templates, the register generators, and the go.mod generator; saves PROJECT; runs the init
  pipeline. Propagates pipeline errors (fails loudly).
- **`createapi.go`** — injects the Kubebuilder resource model with Crossplane defaults;
  validates the resource; renders the resource's API templates and **regenerates the register
  files deterministically** from `GetResources()` + the new resource; persists to PROJECT;
  runs the API-commit pipeline.
- **`update.go`** — the `update` / `update --adopt` command. See §7.
- **`config.go`** — alias to `core.PluginConfig`; `NewPluginConfig()` seeds defaults.
- **`errors_compat.go`** — thin shim delegating to the `validation` package.

## 3. Core layer (`pkg/plugins/crossplane/v2/core/`)

Reusable, side-effecting building blocks with no template knowledge:

- **`command_runner.go`** — `CommandRunner` wraps `exec.CommandContext` with a working dir.
- **`git_runner.go`** — `GitCommandRunner`: `Init`, `Add`, `Commit`/`CommitWithAuthor`,
  `GetUserName/Email`, `AddSubmodule`.
- **`config.go`** — `PluginConfig` (domain, repo prefix, git author, flags); `GenerateDefaultRepo()`.
- **`project.go`** — `ProjectFile` wraps Kubebuilder config; `Save()` and `AddResource()`.
- **`provider.go`** — `ExtractProviderName` / `ExtractProjectName` helpers.
- **`template_path_processor.go`** — maps a template path to an output path (strips `files/`
  and `.tmpl`, applies `{GROUP}`/`{VERSION}`/`{KIND}`/`{IMAGENAME}`).
- **`ownership.go`** — the **ownership gate**: `GeneratedHeader`, `IsToolOwned(content)`, and
  `DecideWrite(exists, existing) → Seed | Overwrite | Skip`. This is the rule that lets `update`
  refresh tool files while never clobbering user files (§6).

## 4. Template engine (`pkg/plugins/crossplane/v2/templates/engine/`)

The engine turns embedded `.tmpl` files into Kubebuilder template products. Its defining
trait is **auto-discovery**: templates are found by walking the embedded filesystem at factory
init, not registered by hand.

- **Discovery** — `autodiscovery.go` classifies each template into `InitCategory` /
  `APICategory` / `StaticCategory` by path pattern; `loader.go` reads template bodies.
- **Factory** — `factory.go` (`CrossplaneTemplateFactory`) registers discovered templates and
  exposes `Create*Template` / `Get*Templates`.
- **Strategy pattern** — `builders.go` defines `BuildStrategy` with `Init`/`API`/`Static`
  implementations; `BaseTemplateBuilder` delegates to the chosen strategy.
- **Products** — `product_base.go` (`BaseTemplateProduct`) embeds Kubebuilder machinery mixins;
  `product_generic.go` (`GenericTemplateProduct`) loads any discovered template's body.
- **Deterministic generators** — instead of parsing and merging existing files, the register
  and go.mod files are rendered **in full** from the project state:
  - `register_generators.go` — `APIRegisterGenerator` (renders `apis/register.go` from the
    unique group/versions) and `ControllerRegisterGenerator` (renders
    `internal/controller/register.go` per kind). Output is a pure function of the resource list.
  - `gomod_generator.go` — `GoModGenerator` renders `go.mod` from the dependency manifest
    (seed-once; never overwritten — see §6/§8).
  - `ownership_doc_generator.go` — `OwnershipDocGenerator` renders `docs/ownership.md` by
    walking the template FS and reading the same headers the ownership gate reads, so the
    published contract cannot drift from the enforced one. It walks the FS directly and uses
    `GenerateOutputPath` (not `GetOutputPath`, which keeps the `project/` prefix).
  - `assembly.go` — `AsBuilders` and `CoreGenerators` helpers shared by init, create, and update.

## 5. Automation pipeline (`pkg/plugins/crossplane/v2/automation/`)

A sequential chain of steps run after scaffolding. **Every step is required** — a failure
aborts (no warn-and-continue) — and the **commit is last**, so the tree is left clean and
fully committed.

- **`steps.go`** — `Step` interface (`Name`, `Execute`); steps: `GitInitStep`, `GitCommitStep`,
  `GitSubmoduleStep`, `MakeStep(target)`, `GoModTidyStep`.
- **`pipeline.go`** — `NewInitPipeline()` runs git init → submodule → `make submodules` →
  `go mod tidy` → `make generate` → `make reviewable` → **commit**; `NewAPICommitPipeline()`
  runs `make generate` → **commit**. `Run()` aborts on the first failure.
- **`git.go`** — `GitOperations`: idempotent `Init`, `CreateCommit`, idempotent `AddSubmodule`.

## 6. Ownership contract (the upgrade foundation)

A file is **tool-owned** iff it carries `// Code generated by xp-provider-gen. DO NOT EDIT.`:

| Bucket | Files | On `update` |
|--------|-------|-------------|
| Tool-owned (header) | `<kind>/wiring.go`, `internal/provider/connector.go`, all `register.go`, `config.go`, `main.go`, `doc.go`, `generate.go`, `groupversion_info.go`, `version.go`, `docs/ownership.md` | overwritten |
| Codegen-owned | `zz_generated.*`, CRDs | regenerated by `make generate` |
| User-owned (no header) | `<kind>/external.go`, `internal/provider/client.go`, `internal/provider/options.go`, `*_types.go` | never touched |
| Seed-once (no header) | `go.mod`, `crossplane.yaml`, Makefile, Dockerfile, README, `AGENTS.md` | created once, never re-touched |

"User-owned" and "seed-once" are the same mechanism, not two: both are headerless,
and `DecideWrite` treats them identically. The distinction is intent — user-owned
files are ones we expect to be edited, seed-once files are ones we merely decline
to manage.

A golden test (`templates/engine/ownership_test.go`) asserts every template's
bucket by walking the embedded FS, so adding a template forces a deliberate
ownership decision. Without it, forgetting the header on a new tool-owned template
would silently make it user-owned and `update` would never refresh it.

`core.DecideWrite` enforces this: seed if absent, overwrite if the on-disk file is tool-owned,
otherwise skip. `go.mod` is seed-once; its framework versions are bumped via `go get`, never by
overwrite.

## 7. The `update` command (`update.go`)

`update` refreshes an existing provider's tool-owned core to the current generator:

1. **Precondition** — the working tree must be clean (drift protection); the result is left
   uncommitted for review via `git diff`.
2. **Render** the full template set into an in-memory FS (`afero.NewMemMapFs`).
3. **Reconcile** onto disk through `core.DecideWrite` (tool files overwritten, user files
   skipped, new files seeded).
4. **Bump dependencies** from the manifest via `go get` (go.mod's own requires preserved).
5. `go mod tidy` / `make generate` / `make reviewable`; stamp the generator version into PROJECT.

**`update --adopt`** retrofits a provider generated before the contract existed: it writes the
header onto recognized tool-owned files (so plain `update` can manage them) and stamps
provenance. User files are never adopted.

## 8. Validation, templates & the dependency manifest

- **Validation** (`validation/`) — `validator.go` enforces Kubebuilder/Kubernetes conventions;
  `errors.go` provides `PluginError` with hint helpers.
- **Templates** (`pkg/templates/`) — `loader.go` embeds the `.tmpl` tree via `go:embed`. To add
  scaffolding, add a `.tmpl` file; discovery picks it up. Tool-owned templates include the
  generated header; user-owned ones (`external.go`, `client.go`, `options.go`, `*_types.go`)
  do not. The golden ownership test enforces this — see §6.
- **Dependency manifest** (`pkg/versions/`) — `dependencies.yaml` is the single source of truth
  for the generated provider's direct dependency versions, plus the `GoVersion` constant. It is
  rendered into `go.mod`, tracked by a Renovate custom manager, and applied to existing
  providers by `update`.

## 9. Seams (the modular layout)

A **seam** is three things: a tool-owned caller, a user-owned stub, and one
documented name. That is the whole extension mechanism — adding a seam later is
mechanical, not a redesign.

| Seam | Tool-owned caller | User-owned stub | Names |
|------|-------------------|-----------------|-------|
| External logic | `<kind>/wiring.go` | `<kind>/external.go` | `NewExternal` |
| API client | `internal/provider/connector.go` | `internal/provider/client.go` | `Client`, `NewClient` |
| Reconciler options | `<kind>/wiring.go` | `<kind>/external.go` | `ReconcilerOptions` |
| Provider options | `cmd/provider/main.go` | `internal/provider/options.go` | `Flags`, `Configure` |

**Only those six names are frozen.** Tool-owned signatures — `Connector`,
`ClientConfig`, `clientConfig` — may change in any release without being a breaking
change, which is the point of moving the plumbing tool-side.

The connector is shared rather than per-kind because `ProviderConfigUsageTracker.Track`
takes `resource.ModernManaged`: the concrete-kind assertion the old per-kind
`controller.go` performed existed only to satisfy it, so asserting `ModernManaged`
once makes the whole of `Connect` kind-agnostic. The per-kind part is the injected
`func(*Client) managed.ExternalClient` factory.

Per-kind `wiring.go` stays flat and explicit rather than hidden behind a generic.
DRY governs hand-maintained sources of truth, not generated output — one template
producing N files is one source of truth, and `update` regenerates all N for free.
The connector was different precisely because it lived in a user-owned file and so
could never be regenerated.

## Design patterns at a glance

| Pattern | Where |
|---------|-------|
| Plugin architecture | Kubebuilder v4 plugin (`plugin.go`) + `WithExtraCommands` for `update` |
| Auto-discovery | `autodiscovery.go` + `factory.go` |
| Deterministic generation | register/go.mod generators (no parse-and-merge) |
| Ownership gate | `core.DecideWrite` (header-based) |
| Render-then-reconcile | `update` renders to memfs, reconciles to disk via the gate |
| Pipeline / chain | `Pipeline` + `Step` |
| Embedded FS | `go:embed` template tree |

## Command flow summary

**`init`** → validate → scaffold init/static templates + register & go.mod generators → save
PROJECT → init pipeline (git init/submodule, `make submodules`, tidy, generate, reviewable,
commit).

**`create api`** → inject & validate resource → render API templates + **regenerate register
files** from all resources → `AddResource` to PROJECT → API-commit pipeline (generate, commit).
While the history is still just the tool's scaffold (the `Initial commit` carries the
`xp-provider-gen-scaffold` trailer and the user hasn't committed yet), the commit **folds into
that `Initial commit`** via `--amend`, so a freshly scaffolded provider has a single commit;
once the user commits their own work, later `create api` runs add separate commits.

**`update`** → require clean tree → render to memfs → reconcile via the ownership gate → bump
deps via `go get` → tidy/generate/reviewable → stamp provenance (no commit; review the diff).

**`update --adopt`** → require clean tree → render to memfs → add the header to recognized
tool-owned on-disk files → stamp provenance (no commit).
