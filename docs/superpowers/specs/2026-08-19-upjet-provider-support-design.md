# Upjet-based Provider Support — Design

- **Date:** 2026-08-19
- **Status:** Approved (user asked to preserve the design, then implement directly)
- **References:** `crossplane/upjet` and `crossplane/upjet-provider-template` (read 2026-08-19)

## 1. Problem

`xp-provider-gen` scaffolds native providers: the author writes reconcile logic by
hand in `external.go`. An **upjet** provider is a different animal — its API types and
controllers are *generated from a Terraform provider's schema*, and the author instead
writes small **configuration** files that tell upjet which Terraform resources to
expose and how. Today the tool cannot produce one at all.

## 2. What an upjet provider actually is

From `upjet-provider-template`: 28 files are generated (`zz_*`), 23 are hand-written.
Only the hand-written surface is ours to scaffold:

| Hand-written (we scaffold) | Purpose |
|---|---|
| `config/provider.go` | central upjet config: schema, resource prefix, root group, include list |
| `config/external_name.go` | per-Terraform-resource external-name strategy + include list |
| `config/<scope>/<group>/config.go` | per-resource configurator (`Kind`, overrides) |
| `cmd/generator/main.go` | runs `pipeline.Run(cluster, namespaced, rootDir)` |
| `internal/clients/<provider>.go` | ProviderConfig → `terraform.Setup` (the credentials seam) |
| `apis/generate.go` | the `go:generate` chain: scraper → generator → controller-gen → angryjet |
| `apis/{cluster,namespaced}/v1beta1/*` | ProviderConfig types (both scopes, as upjet expects) |
| `Makefile` | `TERRAFORM_*` variables, schema download, `generate.init` |
| `internal/{features,version}`, `cmd/provider/main.go`, package/cluster/hack files | standard |

Generation requires, **before** `make generate` runs: `config/schema.json` (produced by
`terraform providers schema -json` for the configured provider) and
`config/provider-metadata.yaml` (produced by upjet's scraper from the Terraform
provider's docs). The generated Makefile already wires both as `generate.init`
prerequisites — our scaffold must preserve that contract exactly.

## 3. Decisions

**D1 — Flavor = template root.** `pkg/templates/files/**` stays the native flavor,
untouched. Upjet templates live in a new root `pkg/templates/upjet/**`. The factory
already walks a single hardcoded root; it gains that root as a parameter. Nothing about
the native path changes, so this cannot break existing providers.

**D2 — The flavor is persisted in PROJECT**, in the plugin config block that already
carries provenance. `create api` and `update` read it, so a user never repeats the flag.

**D3 — `init --upjet` takes the Terraform provider coordinates** (`--terraform-provider`,
`--terraform-provider-version`, `--terraform-provider-repo`, `--terraform-provider-docs-path`,
`--terraform-version`), and renders them into the Makefile and `internal/clients`. All
have sensible defaults; only `--terraform-provider` is required with `--upjet`.

**D4 — `create api` on an upjet project configures a resource, it does not write types.**
`--terraform-resource=kubernetes_secret` is required; group/version/kind keep their
existing meaning. It writes the per-resource configurator and re-renders
`config/external_name.go` from the full resource list — the same deterministic
"regenerate from PROJECT" pattern the native path uses for `register.go`, so adding a
second resource cannot drift. It also seeds a chainsaw behavior test, exactly as the
native path does.

**D5 — Ownership contract is unchanged.** Config files the author edits are user-owned;
the generator entrypoint, the go:generate chain and the register/setup files are
tool-owned. The golden ownership test covers both roots.

**D6 — Both scopes.** Upjet's `pipeline.Run` takes a cluster *and* a namespaced provider
and generates both trees. We follow upjet rather than imposing the native path's
namespaced-only decision; that decision governed our own templates, not upjet's pipeline.

## 3b. The bootstrap problem (found while implementing)

A provider generated from upstream's template always has upjet's output already
committed. A *freshly scaffolded* one does not, and three things follow that the
design has to handle explicitly:

1. **The project does not compile until `make generate` runs.** `cmd/provider`
   imports `apis/{cluster,namespaced}` and `internal/controller/{cluster,namespaced}`,
   packages upjet creates. So the init pipeline stops before building, and the
   next-steps text says to generate first.
2. **Generation must be scoped to `./apis/...`.** The build submodule generates
   over every package in `GO_SUBDIRS`, which includes `cmd` — unloadable before
   the first generation. Overriding `generate.run` does not work, because the
   submodule declares `generate.run: go.generate` and make merges prerequisites;
   the recipe-bearing `go.generate` is what has to be replaced.
3. **`go.sum` must cover the generator tools.** They sit behind the `generate`
   build tag, so a plain `go mod download` never fetches them; `go mod tidy -e`
   does, and tolerates the packages that do not exist yet.

## 4. Non-goals

- No vendoring of Terraform or the Terraform provider — the generated Makefile downloads
  them, as upstream does.
- No attempt to run `make generate` inside `init`: it needs network and minutes. The
  scaffold prints the next command.
- No changes to the native flavor's templates, commands, or ownership rules.

## 5. Verification

1. Repo gate (`make reviewable`) and the golden ownership test over both roots.
2. `make e2e-test` (native path) stays green — proof the flavor split broke nothing.
3. **Upjet e2e** (new, `scripts/e2e-upjet.sh`): scaffold a provider for
   `hashicorp/kubernetes`, add `kubernetes_secret` via `create api`, then run the real
   upjet pipeline — `make generate` — and assert it produces `zz_*` types, CRDs and
   controllers, and that the provider builds. That exercises every config file we
   scaffold against the actual upjet generator, which is the thing worth proving.
