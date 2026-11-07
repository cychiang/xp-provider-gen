# Architecture

`xp-provider-gen` is a CLI built on the Kubebuilder v4 plugin system. It scaffolds a Crossplane
provider from embedded templates, runs a post-scaffold pipeline (git, module tidy, code
generation), and later refreshes the tool-owned part of that provider with `update`, leaving
the author's code alone.

This page holds the contracts and invariants that stay true when a function is renamed. To
learn what a function does, read its package. The decisions behind these contracts are linked
in [section 8](#8-decision-records).

## 1. Layers

```
cmd/xp-provider-gen/            CLI entry: the Kubebuilder CLI plus the update and create-test commands
pkg/plugins/crossplane/v2/      Plugin: the init, create api, update and create-test commands
  core/                         Side-effecting building blocks with no template knowledge
  templates/engine/             Template discovery, the generators, and Render
  automation/                   Post-scaffold pipeline: steps and git operations
  validation/                   Input validation
pkg/templates/                  Embedded template filesystem, one root per flavor
pkg/versions/                   dependencies.yaml: the pins a generated provider gets
pkg/version/                    This CLI's own version, distinct from pkg/versions/
test/e2e/                       The e2e stage harness for both flavors
```

`core` imports nothing else from this module. `templates/engine`, `automation` and
`validation` import `core` and never each other. Run
`go list -f '{{join .Imports "\n"}}' ./pkg/plugins/crossplane/v2/<package>` to check.

## 2. Commands

One rule holds for every command: **a render never writes to the provider directory.** A
command renders the full result into memory with `engine.Render`, then writes it through
`core.Apply`, which decides per file from what is already on disk (section 6). A failed render
leaves the directory untouched.

- **`init`** validates its inputs, renders the init scope, applies it, and runs the init
  pipeline. Kubebuilder saves PROJECT right after the scaffold step, before the pipeline runs.
- **`create api`** adds the resource to the config, renders the kind's templates, and
  regenerates the register files from every resource (section 3). While the history is still
  the tool's scaffold, the commit folds into the `Initial commit`, which carries the
  `xp-provider-gen-scaffold` trailer, so a fresh provider has one commit. After the author
  commits, later runs add their own commits.
- **`update`** is its own cobra command, because Kubebuilder's plugin interface has no update
  hook. Section 7 is its contract.
- **`create-test`** renders a chainsaw skeleton to `test/behavior/<name>/chainsaw-test.yaml`
  and never overwrites one.

PROJECT records the flavor (native or upjet). The flavor picks the template root, the
generators, the validator and the pipelines.

## 3. Template engine

- **Templates are data.** Discovery walks the embedded filesystem once. A path with `GROUP`,
  `VERSION` or `KIND` placeholders is per-kind, and every other path is init. Every path lands
  in one of the two, so a template cannot be silently dropped, and adding a file needs no
  registration ([templates.md](templates.md) is the contributor guide). A walk error panics,
  because the filesystem is embedded and an error is a build defect.
- **Slices, not maps.** Nothing looks a template up by name, and a derived key could collide
  and drop a file (`Makefile.tmpl` exists in two directories).
- **A template's machinery action is `SkipFile`, except in the in-memory render `update`
  uses.** That render sets `OverwriteFile` on a template carrying the header. The generators
  set their own action: `OverwriteFile` for the register files, `docs/ownership.md` and the
  upjet resources file, `SkipFile` for `go.mod`. No render writes to disk on its own, because
  `core.Apply` decides each write.
- **Generated files are rendered in full, never parsed and merged.** The register files are a
  pure function of the resource list, and `go.mod` is rendered from the dependency manifest.
  Generator bodies live in `pkg/templates/generators/`, outside the discovered roots.
- **`docs/ownership.md` is built from the headers the ownership gate reads.** It walks the
  template filesystem and maps each path as `core.GenerateOutputPath` does, so the published
  contract cannot drift from the enforced one.
- **`engine.Render` is the one assembly.** Its three scopes (init, kind, project) are the only
  place templates and generators combine, for both flavors.

## 4. Automation pipeline

A pipeline is a sequence of steps that runs after the scaffold.

- **Every step is required.** A failure aborts the run. There is no warn-and-continue.
- **The commit is last**, so the tree is clean and committed when the command returns.
- **Each command picks its pipeline in one place per flavor**, and `update` picks its finalize
  pipeline the same way.
- **An upjet project does not compile until `make generate` has run.** Its init pipeline skips
  tidy, generate and reviewable, and runs `go mod download` for the `go.sum` entries of the
  generator's own tools. Its create-api pipeline only folds the commit.

## 5. Validation and the dependency manifest

- **Validation** enforces Kubebuilder and Kubernetes naming rules and wraps a failure with a
  fix-it hint. `update` runs it again over PROJECT, because that file may have been edited
  by hand.
- **`pkg/versions/dependencies.yaml` is the one source** for a generated provider's pins. It
  is rendered into `go.mod`, tracked by a Renovate custom manager, and applied to an existing
  provider by `update`.

## 6. Ownership contract (the upgrade foundation)

A file is tool-owned if and only if it carries `// Code generated by xp-provider-gen. DO NOT EDIT.`
within its first 1024 bytes (`headerScanLimit`, `core/ownership.go`). The match is on a
substring, so an author-written file that quotes the line becomes tool-owned and `update` overwrites it.

| Bucket | Files | On `update` |
|--------|-------|-------------|
| Tool-owned (header) | `<kind>/wiring.go`, `internal/provider/connector.go`, all `register.go`, `config.go`, `main.go`, `doc.go`, `generate.go`, `groupversion_info.go`, `version.go`, `docs/ownership.md`, `hack/xp-provider-gen.mk`, `cluster/local/integration_tests.sh` | overwritten; removed if this render no longer produces it |
| Codegen-owned | `zz_generated.*`, CRDs | regenerated by `make generate` |
| User-owned (no header) | `<kind>/external.go`, `internal/provider/client.go`, `internal/provider/options.go`, `*_types.go` | never touched |
| Seed-once (no header) | `go.mod`, `crossplane.yaml`, Makefile, Dockerfile, README, `AGENTS.md` | created once, never re-touched |

User-owned and seed-once are one mechanism. Both are headerless, and `DecideWrite` treats them
the same. The difference is intent: the first are files we expect the author to edit, and the
second are files we decline to manage.

`core.DecideWrite` returns seed if the file is absent, overwrite if the file on disk is
tool-owned, and skip otherwise. The golden test `templates/engine/ownership_test.go` asserts
every template's bucket, so a new template fails until someone decides its owner. Without it, a
tool-owned template that forgot its header would be user-owned, and `update` would never
refresh it. `go.mod` is seed-once, and `update` changes it in two ways only: it raises the
`go` directive to `go_version` and never lowers it (`godirective.go` parses with `modfile`,
keeps everything else, and writes the file), and it runs `go get` for each pinned dependency.

**Every write and delete passes `core.CheckContained`.** Rendered paths come from PROJECT, so
a hand-edited PROJECT must not turn `update` into an arbitrary-file-write primitive.
`CheckContained` rejects an absolute path, a path that climbs out of the project and the
empty path (`filepath.IsLocal`). `core.Apply` calls it before it touches a file, and so do
orphan removal and `--adopt`. A new code path that writes or deletes must call it too.

The generated Makefile follows the same line. The user-owned `Makefile` holds the project
variables and `include hack/xp-provider-gen.mk`. The tool-owned fragment holds the build
pipeline, so a pipeline fix reaches existing providers on `update`.

## 7. The `update` command

`update` brings a provider's tool-owned core to the running generator and leaves the result
uncommitted for review.

1. **The tree must be clean**, so the whole effect shows as one `git diff`.
2. **PROJECT is loaded once and gates the rest.** An unrecognized `flavor:`, or `flavor: upjet`
   with no `upjet:` block, is a corrupt PROJECT and is refused, not defaulted to native. Only an
   empty flavor means native. `checkNotDowngrade` refuses a generator older than the clean
   release that last stamped PROJECT, and `checkSameGenerator` applies the same ordering to
   `create api`. Both compare only `vX.Y.Z` strings with `semver.Compare`, so a development
   build is never judged older.
3. **Render** the flavor's full set into memory.
4. **Apply `--terraform-provider-version`**, if given, before reconcile. It rewrites the
   Terraform provider pin, and a native provider refuses it.
5. **Reconcile** through `core.Apply`. Tool-owned files are overwritten, user-owned files are
   skipped, and a file already equal to the render counts as `Unchanged` and is not written. An
   upjet project never seeds a missing user-owned file, because some need init-time Terraform
   settings that PROJECT does not keep. `update` lists them instead.
6. **Remove orphans.** A file is a candidate only if git tracks it (`git ls-files`, so an
   ignored file is never one), it carries the header, and no template renders it now.
7. **Raise the go directive, then bump dependencies** from the manifest. The directive moves to
   `go_version` only if it is lower. A higher one is the author's and stays.
8. **Finalize.** Native runs tidy, generate and reviewable. Upjet runs generate first, because
   tidy cannot resolve the packages that generation produces. Then `update` stamps the
   generator version into PROJECT.

`update --adopt` retrofits a provider that predates the contract. It adds the header to the
recognized tool-owned files and stamps PROJECT. It never adopts a user-owned file.

## 8. Decision records

- [Render and apply](design/render-apply.md) is the proposal behind one render path and one
  write rule. It is implemented.
- [docs/plans/](plans/) holds one record per program: what it decided, what it declined, and
  the limits it accepted. Read the relevant one before reversing a decision it covers.

## 9. Seams (the modular layout)

A seam is a tool-owned caller, a user-owned stub, and one documented name. Adding a seam later
is mechanical, not a redesign.

| Seam | Tool-owned caller | User-owned stub | Names |
|------|-------------------|-----------------|-------|
| External logic | `<kind>/wiring.go` | `<kind>/external.go` | `NewExternal` |
| API client | `internal/provider/connector.go` | `internal/provider/client.go` | `Client`, `NewClient` |
| Reconciler options | `<kind>/wiring.go` | `<kind>/external.go` | `ReconcilerOptions` |
| Provider options | `cmd/provider/main.go` | `internal/provider/options.go` | `Flags`, `Configure` |

**Only those six names are frozen.** Tool-owned signatures such as `Connector`,
`ClientConfig` and `clientConfig` may change in any release without being a breaking change.
That freedom is why the plumbing moved to the tool side.

The connector is shared across kinds because `ProviderConfigUsageTracker.Track` takes
`resource.ModernManaged`. Asserting that type once makes `Connect` kind-agnostic. The per-kind
part is the injected `func(*Client) managed.ExternalClient` factory.

Per-kind `wiring.go` stays flat and explicit instead of hiding behind a generic. DRY governs
hand-maintained sources of truth, not generated output: one template that produces N files
is one source of truth, and `update` regenerates all N. The connector differed because it
lived in a user-owned file, which can never be regenerated.
