---
name: xp-provider-gen
description: Scaffold, configure, and maintain Crossplane providers with xp-provider-gen, a Go CLI generating native (hand-written reconcilers) or upjet (Terraform-wrapped) provider projects via init, create api, create-test, and update. Use when initializing a new Crossplane provider, adding a managed resource kind, wrapping a Terraform provider with upjet, writing a chainsaw behavior test, upgrading an existing generated provider, or contributing to the xp-provider-gen generator itself.
---

<!-- Portable skill: any Markdown-reading agent, any client. Frontmatter carries only name
     and description (no client-specific keys). The body below names no tools and no agent
     features — only shell commands, when to run them, and what to check. Keep it that way. -->

Two audiences, two sections: **A** uses `xp-provider-gen` to build a Crossplane provider.
**B** contributes to the generator's own source.

## Install

Cross-client, user-level: copy or symlink this directory to `~/.agents/skills/xp-provider-gen`.
Individual clients also scan their own native directory — check your client's docs for its own
convention (for example, Claude Code additionally reads project-level `.claude/skills/` and
user-level `~/.claude/skills/`). Commands below assume `xp-provider-gen` is on `PATH`; otherwise
substitute its actual path (e.g. `./bin/xp-provider-gen`).

## A. Build a provider

Flavor is chosen once, at `init`, and recorded in PROJECT for every later command to read:
**native** (you write the reconcile logic) or **upjet** (`--upjet`: types and controllers are
generated from a wrapped Terraform provider's schema, you write configuration). Each entry below
names which flavor(s) it applies to.

### `init` — native

- **What**: scaffolds a complete native provider project — ProviderConfig APIs, package
  metadata, build-submodule wiring, Go module and controller structure.
- **When**: starting a new provider where you write the external-client/reconcile logic
  yourself.
- **Command**:
  ```bash
  xp-provider-gen init --domain=example.com --repo=github.com/example/provider-acme
  ```
- **Produces**: runs 8 steps automatically — git init, mark scripts executable, add the
  `crossplane/build` submodule, `make submodules`, `go mod tidy`, `make generate`,
  `make reviewable`, then one commit. Check: `--domain` is required (`init --help`); anything
  you scaffold before your own first commit (e.g. `create api`) folds into that same commit
  rather than creating a new one.

### `init --upjet` — upjet

- **What**: scaffolds an upjet provider — a Terraform provider's schema drives generated types
  and controllers; you write per-resource configuration instead of reconcile logic.
- **When**: wrapping an existing Terraform provider rather than hand-writing an external client.
- **Command**:
  ```bash
  xp-provider-gen init --domain=example.com --repo=github.com/example/provider-k8s \
    --upjet --terraform-provider=hashicorp/kubernetes --terraform-provider-version=2.38.0
  ```
- **Produces**: a **6**-step pipeline, deliberately shorter than native's — git init, scripts
  executable, add the build submodule, `make submodules`, `go mod download` (not `tidy`),
  commit. No `make generate`, no `make reviewable`: the scaffold does not compile yet, because
  `cmd/provider` imports API/controller packages that only `make generate` produces from the
  Terraform schema. Optional coordinates: `--terraform-provider-repo` (defaults to the
  provider's hashicorp GitHub repo), `--terraform-provider-docs-path` (default
  `docs/resources`), `--terraform-version` (default `1.5.7`). Check: **TERRAFORM_VERSION must
  stay below `1.6.0`** — Terraform 1.6+ is BSL licensed and the generated Makefile's
  `check-terraform-version` target refuses it.

### `create api` — native

- **What**: scaffolds a managed resource API — CRD, Parameters/Observation structs, controller,
  external-client interface, and registers it in the controller manager.
- **When**: adding a resource kind to a native provider.
- **Command**:
  ```bash
  xp-provider-gen create api --group=storage --version=v1alpha1 --kind=Bucket
  ```
- **Produces**: runs `make generate`, folds into the scaffold commit until your first own
  commit. Check: a second kind in an already-scaffolded group/version does **not** need
  `--force`; `--force` is only for re-scaffolding files that already exist.

### `create api` — upjet

- **What**: same scaffold, plus wiring one Terraform resource type into the provider's
  per-kind configuration.
- **When**: adding a resource kind to an upjet provider.
- **Command**:
  ```bash
  xp-provider-gen create api --group=core --version=v1alpha1 --kind=ConfigMap \
    --terraform-resource=kubernetes_config_map
  ```
- **Produces**: `config/<kind>/config.go`, and regenerates `config/zz_resources.go` (the
  aggregator listing every configured resource) in full; folds into the scaffold commit until
  your first own commit, same as native. Run `make generate` next to actually generate the
  types and controller from the Terraform schema (network — fetches the schema and docs).

### `create-test`

- **What**: scaffolds a chainsaw behavior-test skeleton under `test/behavior/<name>/`, applying
  one managed resource and asserting it becomes Ready. Reads the kind's
  `examples/<group>/<kind>.yaml` for the manifest shape — same command, same output, on
  either flavor.
- **When**: pinning real controller behavior (error paths, drift, pause) for a kind, once that
  kind has an example manifest. `create api` seeds one automatically on native; on upjet it
  cannot (nothing about a valid `spec.forProvider` is knowable before `make generate` runs) —
  write the example first, or `create-test` refuses with the exact path it looked for.
- **Command**:
  ```bash
  xp-provider-gen create-test --name=bucket-pause --kind=Bucket
  ```
- **Produces**: `test/behavior/bucket-pause/chainsaw-test.yaml` with TODO assertions to fill
  in; run with `make test-behavior` or as part of `make e2e`. Check: `--kind` is only optional
  when the project has exactly one kind; with more than one it prompts on a TTY or errors
  asking for both flags non-interactively.

### `update`

- **What**: regenerates the tool-owned core of an existing provider — registration, controller
  wiring, `main.go`, config, framework dependency versions — without touching your business
  logic.
- **When**: pulling in generator fixes/improvements after upgrading `xp-provider-gen`, on a
  native provider.
- **Command**:
  ```bash
  xp-provider-gen update
  ```
- **Produces**: requires a clean working tree; leaves the result uncommitted for `git diff`
  review, then your own commit. On a mid-run failure the error names the exact revert step.
  Check: **refuses upjet projects outright** — its per-kind Terraform coordinates aren't
  persisted in PROJECT yet, so there's nothing safe to re-render from (`update --help`).

### `update --adopt`

- **What**: one-time retrofit for a provider generated before the ownership contract existed —
  stamps provenance and writes the generated header onto recognized tool-owned files.
- **When**: bringing an old, pre-contract provider under `update`'s management.
- **Command**:
  ```bash
  xp-provider-gen update --adopt
  ```
- **Produces**: exits after stamping headers; commit that diff, then run plain `update`
  afterward to actually refresh the files.

### `version` / `completion`

`xp-provider-gen version` prints the build version. `xp-provider-gen completion
{bash,zsh,fish,powershell}` prints a shell completion script to source.

### Ownership: what's yours, what isn't

Every tool-owned generated file carries a `// Code generated by xp-provider-gen. DO NOT EDIT.`
header and is overwritten by `update`; everything else is yours forever — see the generated
`docs/ownership.md` **inside your own provider** for the exact, always-accurate list.

### Ignore these

`alpha` (refuses outright — not supported), `edit`, `create webhook`, and the
`--plugins`/`--project-version` flags are Kubebuilder-framework leftovers, not part of this
tool's supported surface.

### The generated provider's own loop — native

- **`make generate`** — what: runs code generation (deepcopy, CRDs). when: after editing types.
  command: `make generate`. produces: regenerated `apis/**/zz_*` and `package/crds/`.
- **`make build`** — what: builds the provider binary and image (from the `crossplane/build`
  submodule). when: after generate. command: `make build`. produces:
  `_output/bin/<os>_<arch>/provider` — not `bin/`, which is this generator's own output dir.
- **`make reviewable`** — what: `generate` + `lint` + `test`, from `crossplane/build`'s
  `common.mk` — a different Makefile from this generator's own `reviewable` gate, and does not
  include gosec or mod-tidy. when: before every commit. command: `make reviewable`. produces:
  green output, or the one failing step to fix.
- **`make test-integration`** — what: fast loop — runs the controller from source against a
  throwaway, auto-removed kind cluster. when: iterating on reconcile logic without a full
  package build. command: `make test-integration`. produces: cluster
  `<provider>-integration`, removed when the run ends.
- **`make dev` / `make dev-clean`** — what: creates (or deletes) a throwaway kind cluster with
  CRDs installed and the provider running from source. when: fast local iteration. command:
  `make dev`. produces: kind cluster `<provider>-dev`.
- **`make run`** — what: runs the provider binary out-of-cluster, against your current
  `kubectl` context, with `--debug`. when: quick local runs without kind at all. command:
  `make run`. produces: the controller manager running in your terminal.
- **`test/setup.sh`** — what: seeded, user-owned uptest setup script — waits for the provider
  package to become Healthy, then applies `examples/provider/config.yaml`. when: before
  `make test-behavior` / `make e2e` against a live cluster. command:
  `KUBECTL=kubectl ./test/setup.sh`. produces: ProviderConfig + credentials applied.
- **`make test-behavior`** / **`make e2e`** — what: runs the chainsaw suite under
  `test/behavior/`, or the full package-build-deploy-uptest-chainsaw cycle. when: after
  `create-test`, or for release confidence. command: `make e2e`. produces: a dedicated
  `<provider>-e2e` kind cluster left running (`make e2e-clean` removes it).

### The generated provider's own loop — upjet

- **`make generate`** — what: fetches the Terraform provider's schema and docs (network), then
  runs upjet's code generation. when: after `create api` adds a resource. command:
  `make generate`. produces: types/controllers under `apis/{cluster,namespaced}/**` and
  `internal/controller/{cluster,namespaced}/**`.
- **`make build`** — what: builds the provider binary and an image with Terraform itself and
  the mirrored provider plugin baked in (no network needed at pod runtime). when: after
  generate. command: `make build`. produces: image `<registry>/<provider>-<arch>`; the
  container's `ENV` satisfies `cmd/provider`'s required flags with nothing extra to configure.
- **`make local-deploy`** — what: builds, creates a dedicated kind cluster, installs
  Crossplane, deploys the provider package. when: first real run of an upjet provider. command:
  `make local-deploy`. produces: cluster `<provider>-e2e`; switches your `kubectl` context to it.
- **`make run`** — what: runs the provider binary out-of-cluster, against your current
  `kubectl` context, with `--debug`. when: quick local runs without kind at all — the project
  Makefile's own `export TERRAFORM_VERSION ?=`/etc. (no image needed) already satisfy
  `cmd/provider`'s required flags. command: `make run`. produces: the controller manager
  running in your terminal.
- **`test/setup.sh`** — what: seeded, user-owned — waits for the provider package to become
  Healthy, then applies `examples/providerconfig/providerconfig.yaml`. when: right after
  `local-deploy`, before applying your own managed resources. command:
  `KUBECTL=kubectl ./test/setup.sh`. produces: ProviderConfig + Secret applied. Check:
  **your managed resource and this ProviderConfig/Secret must share a namespace** — the
  namespaced credential lookup requires it
  ([details](https://github.com/cychiang/xp-provider-gen/blob/main/docs/upjet-provider.md#6-worked-example-managing-a-configmap-with-hashicorpkubernetes)).
- **`make test-behavior`** / **`make e2e`** — what: runs the chainsaw suite under
  `test/behavior/` (skips, not fails, if that directory doesn't exist yet — `create-test` seeds
  it), or the full package-build-deploy-uptest-chainsaw cycle against every file under
  `examples/*/*.yaml` except the ProviderConfig example — no `UPTEST_EXAMPLE_LIST` to set, same
  wildcard convention as native. when: after `create-test`, or for release confidence. command:
  `make e2e`. produces: a dedicated `<provider>-e2e` kind cluster left running (`make e2e-clean`
  removes it). Check: **a fresh scaffold's `examples/` has nothing for `make e2e` to test** —
  `create api` cannot seed a real example for upjet (nothing about a valid `spec.forProvider` is
  knowable before `make generate` runs) — write one first; the scraped
  `examples-generated/**` manifests are not applyable as-is (unresolved Terraform
  interpolations like `${file(...)}`/`${filebase64(...)}`)
  ([details](https://github.com/cychiang/xp-provider-gen/blob/main/docs/upjet-provider.md#5-build-and-deploy)).

## B. Develop `xp-provider-gen` itself

Not scaffolding a provider — contributing to this generator's own code? Start with
[AGENTS.md](https://github.com/cychiang/xp-provider-gen/blob/main/AGENTS.md) in the repo root:
it has the make targets (`build`, `test`, `lint`, `reviewable`), the two e2e surfaces (native
and upjet), and the working agreements. Templates live under `pkg/templates/files/` (native)
and `pkg/templates/upjet/` (upjet) and are auto-discovered — adding a `.tmpl` file wires it in;
there is no registration step.

The one habit AGENTS.md assumes but doesn't spell out: **prove every template change with an
old-vs-new scaffold diff**, never by reading the template. `go build`'s package argument must
be a directory inside the current module, so build the old side from its own checkout — a git
worktree is the cheapest way to get one. Both `init` runs hit the network and take a minute or
two (submodule clone, `go mod` work).

```bash
git worktree add /tmp/xpgen-old-src <pre-change-ref>   # e.g. a commit or branch before your edit
( cd /tmp/xpgen-old-src && go build -o /tmp/xpgen-old ./cmd/xp-provider-gen )
go build -o /tmp/xpgen-new ./cmd/xp-provider-gen
mkdir -p /tmp/old /tmp/new
( cd /tmp/old && /tmp/xpgen-old init --domain=example.com --repo=github.com/x/p1 )
( cd /tmp/new && /tmp/xpgen-new init --domain=example.com --repo=github.com/x/p1 )
diff -ru /tmp/old /tmp/new   # must show only the lines you intended to change
git worktree remove /tmp/xpgen-old-src
```
