# Upjet Provider Support — Implementation Plan

**Spec:** `docs/superpowers/specs/2026-08-19-upjet-provider-support-design.md`
**Branch:** `feat/upjet-provider`

## Global constraints
- Never modify `pkg/templates/files/**` (native flavor) except where a shared helper demands it.
- Follow the existing patterns: auto-discovered templates, deterministic generators for
  anything derived from the resource list, golden ownership test entry per template.
- Every new template needs a `wantOwnership` entry; tool-owned ones carry the DO NOT EDIT header.

### Task 1 — Flavor plumbing (engine + PROJECT)
- `core`: `Flavor` type (`native`, `upjet`), stored in the plugin config alongside provenance.
- `engine.NewFactory(cfg)` → `engine.NewFactoryForFlavor(cfg, flavor)`; `discoverTemplates`
  walks `files` or `upjet` accordingly. Keep `NewFactory` as the native-flavor wrapper so
  no existing call site changes.
- Golden ownership test walks both roots.

### Task 2 — Upjet templates (`pkg/templates/upjet/**`)
Port the hand-written surface from upjet-provider-template, parameterised:
`config/provider.go`, `config/external_name.go`, `cmd/generator/main.go`,
`internal/clients/PROVIDER.go`, `apis/generate.go`, `apis/{cluster,namespaced}/v1beta1`
(+v1alpha1 register/doc), `internal/{features,version}`, `cmd/provider/main.go`,
`Makefile` (with `TERRAFORM_*`), `package/crossplane.yaml`, `cluster/images/...`,
`examples/providerconfig`, `.gitignore`, `hack/boilerplate.go.txt`, `README.md`, `AGENTS.md`.

### Task 3 — `init --upjet`
Flags: `--upjet`, `--terraform-provider` (required with --upjet), `--terraform-provider-version`,
`--terraform-provider-repo`, `--terraform-provider-docs-path`, `--terraform-version`.
Validate the provider is `org/name`; persist flavor + coordinates in PROJECT; render.

### Task 4 — `create api` on an upjet project
`--terraform-resource` required. Writes `config/<scope>/<group>/config.go` per resource and
re-renders `config/external_name.go` from all resources (deterministic generator).
Seeds the chainsaw behavior test as the native path does.

### Task 5 — Docs
`docs/upjet-provider.md` (the flavor's guide), plus updates to README, CLAUDE.md,
provider-guide (pointer), templates.md (second root), architecture (flavor concept).

### Task 6 — Verification
- `make reviewable`, golden test both roots.
- `make e2e-test` (native) green — the flavor split is non-breaking.
- `scripts/e2e-upjet.sh` + `make e2e-upjet`: scaffold hashicorp/kubernetes provider,
  `create api` for `kubernetes_secret`, run `make generate` (real upjet pipeline),
  assert zz_* types + CRDs + controllers exist and the provider builds.
