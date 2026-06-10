# Architecture

`xp-provider-gen` is a CLI built on the **Kubebuilder v4 plugin system**. It scaffolds a
complete Crossplane provider project from embedded templates and then runs a post-scaffold
automation pipeline (git init, module tidy, code generation).

The code is organized into clearly separated layers:

```
cmd/xp-provider-gen/            CLI entry point (Kubebuilder CLI wiring)
pkg/plugins/crossplane/v2/
├── plugin.go, init.go,         Plugin layer — subcommands (init, create api)
│   createapi.go, config.go
├── core/                       Reusable building blocks (git, exec, config, parsing)
├── templates/engine/           Template discovery, building, and registration updaters
├── automation/                 Post-scaffold pipeline (steps + git operations)
└── validation/                 Input validation (domain, repo, group/version/kind)
pkg/templates/                  Embedded template filesystem (go:embed) + loader
```

## 1. Entry point & command flow

`cmd/xp-provider-gen/main.go` constructs a Kubebuilder CLI and registers the Crossplane
plugin:

```go
cli.New(
    cli.WithCommandName("crossplane-provider-gen"),
    cli.WithPlugins(&crossplanev2.Plugin{}),
    cli.WithDefaultPlugins(cfgv3.Version, &crossplanev2.Plugin{}),
)
```

Kubebuilder routes `init` and `create api` to the plugin's subcommands, each driven through
the standard lifecycle: `BindFlags` → `InjectConfig` → `PreScaffold` → `Scaffold` → `PostScaffold`.

## 2. Plugin layer (`pkg/plugins/crossplane/v2/`)

- **`plugin.go`** — `Plugin` implements Kubebuilder's `plugin.Full`, advertises config v3 /
  plugin v2, and returns the init and create-api subcommands.
- **`init.go`** — binds `--domain`, `--repo`, `--git-name`, `--git-email`; validates inputs;
  resolves git author (CLI flags > system git config > defaults); scaffolds init + static
  templates; saves the PROJECT file; runs the init automation pipeline.
- **`createapi.go`** — injects the Kubebuilder resource model with Crossplane defaults;
  validates the resource; pulls API templates from the factory; chains in the registration
  updaters; persists the resource to PROJECT; runs the API-commit pipeline.
- **`config.go`** — alias to `core.PluginConfig`; `NewPluginConfig()` seeds defaults.
- **`errors_compat.go`** — thin shim delegating to the `validation` package.

## 3. Core layer (`pkg/plugins/crossplane/v2/core/`)

Reusable, side-effecting building blocks with no template knowledge:

- **`command_runner.go`** — `CommandRunner` wraps `exec.CommandContext` with a working dir.
- **`git_runner.go`** — `GitCommandRunner`: `Init`, `Add`, `Commit`/`CommitWithAuthor`,
  `GetUserName/Email`, `AddSubmodule`.
- **`config.go`** — `PluginConfig` (domain, repo prefix, git author, `force`/`generateClient`
  flags); `GenerateDefaultRepo()` names a provider `provider-{dirname}`.
- **`file_parser.go`** — configurable, regex/section-marker Go-source parser with a fluent
  `FileParserBuilder`; used by the updaters to read existing imports/registrations.
- **`project.go`** — `ProjectFile` wraps Kubebuilder config; `Save()` and `AddResource()`.
- **`provider.go`** — `ExtractProviderName` / `ExtractProjectName` helpers.
- **`template_path_processor.go`** — maps a template path to an output path: strips `files/`
  and `.tmpl`, applies `{GROUP}`/`{VERSION}`/`{KIND}`/`{IMAGENAME}` replacements, special-cases
  the `project/` prefix for root-level files.

## 4. Template engine (`pkg/plugins/crossplane/v2/templates/engine/`)

The engine turns embedded `.tmpl` files into Kubebuilder template products. Its defining
trait is **auto-discovery**: templates are found by walking the embedded filesystem at
factory init, not registered by hand.

- **Discovery** — `autodiscovery.go` walks the FS and classifies each template into
  `InitCategory`, `APICategory`, or `StaticCategory` by path pattern. `loader.go`
  (`TemplateLoader`) reads template bodies from the embedded FS.
- **Factory** — `factory.go` (`CrossplaneTemplateFactory`) registers discovered templates
  into `initRegistry` / `apiRegistry` / `staticRegistry` and exposes `Create*Template` and
  `Get*Templates`.
- **Strategy pattern** — `builders.go` defines `BuildStrategy` (`GetCategory`,
  `ValidateOptions`, `GenerateReplacements`) with `Init` / `API` / `Static` implementations;
  `BaseTemplateBuilder` delegates to the chosen strategy. `placeholderImageName` and the
  `GROUP`/`VERSION`/`KIND` keys are the replacement variables.
- **Products** — `product_base.go` (`BaseTemplateProduct`) embeds Kubebuilder machinery
  mixins (Template/Domain/Repository/Boilerplate/Resource); `product_generic.go`
  (`GenericTemplateProduct`) loads any discovered template's body in `SetTemplateDefaults()`.
- **Updaters** — `api_registration_updater.go` rewrites `apis/register.go`, and
  `template_updater.go` rewrites `internal/controller/register.go`, merging new imports and
  registrations with existing ones (de-duplicated) so adding an API wires itself in.

## 5. Automation pipeline (`pkg/plugins/crossplane/v2/automation/`)

A simple chain-of-steps run after scaffolding:

- **`steps.go`** — `Step` interface (`Name`, `Execute`, `IsRequired`); concrete steps:
  `GitInitStep`, `GitCommitStep`, `GitSubmoduleStep`, `MakeStep(target)`, `GoModTidyStep`.
- **`pipeline.go`** — `NewInitPipeline()` runs git init → commit → submodule → `make
  submodules` → `go mod tidy` → `make generate` → `make reviewable`; `NewAPICommitPipeline()`
  runs git commit → `make generate`. `Run()` is sequential, fails fast on required steps and
  warns on optional ones.
- **`git.go`** — `GitOperations` over `GitCommandRunner`: idempotent `Init`, `CreateCommit`
  (system git config or explicit author), idempotent `AddSubmodule`.

## 6. Validation (`pkg/plugins/crossplane/v2/validation/`)

`validator.go` enforces Kubebuilder/Kubernetes conventions: domain (DNS name), repository
(Go module path, warns if not `provider-*`), group (DNS-1123, ≤63), version
(`v\d+(alpha\d+|beta\d+)?`), kind (PascalCase, ≤63, not a reserved name). `errors.go`
provides `PluginError` with an `ErrorBuilder` and context-specific hint helpers.

## 7. Templates (`pkg/templates/`)

`loader.go` embeds the template tree:

```go
//go:embed files files/project/.gitignore.tmpl
var TemplateFS embed.FS
```

Static-path templates (`apis/`, `cmd/provider/`, `package/`, `project/`, `hack/`, etc.) are
init/static; path-variable templates (`apis/GROUP/VERSION/`, `internal/controller/KIND/`,
`examples/GROUP/`) are discovered per API. Kubebuilder's machinery renders Go-template
actions (`{{ .Boilerplate }}`, `{{ .Resource.Kind }}`) when writing each file.

**To add scaffolding, add a `.tmpl` file under `pkg/templates/files/`** — discovery and
rendering pick it up automatically; no registration code is needed.

## Design patterns at a glance

| Pattern | Where |
|---------|-------|
| Plugin architecture | Kubebuilder v4 plugin (`plugin.go`) |
| Auto-discovery | `autodiscovery.go` + `factory.go` (FS walk, not static registration) |
| Strategy | `BuildStrategy` (init / API / static) |
| Factory | `CrossplaneTemplateFactory` |
| Builder (fluent) | `FileParserBuilder`, `ErrorBuilder` |
| Pipeline / chain | `Pipeline` + `Step` |
| Mixin composition | Kubebuilder machinery mixins in `BaseTemplateProduct` |
| Embedded FS | `go:embed` template tree |

## Command flow summary

**`init`** → validate inputs → scaffold init + static templates → save PROJECT → init
pipeline (git init/commit/submodule, `make submodules`, `go mod tidy`, `make generate`,
`make reviewable`).

**`create api`** → inject & validate resource → build API templates (`{GROUP}`/`{VERSION}`/
`{KIND}`) → append registration updaters → scaffold → `AddResource` to PROJECT → API-commit
pipeline (git commit, `make generate`).
