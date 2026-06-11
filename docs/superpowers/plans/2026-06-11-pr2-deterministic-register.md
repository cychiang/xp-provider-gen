# PR 2: Deterministic register.go — Implementation Plan

**Goal:** Replace the parse-and-merge updater subsystem with two deterministic generators that render `apis/register.go` and `internal/controller/register.go` in full from the PROJECT resource list. Delete `api_registration_updater.go`, `template_updater.go`, and `file_parser.go`.

**Architecture:** Two `machinery.Template` builders precompute their entries in Go (so dedup is explicit) and render a fixed template body:
- `APIRegisterGenerator` — entries are unique **(group, version)** pairs (two kinds in one GV share a scheme builder), with the base `providerv1alpha1` always first.
- `ControllerRegisterGenerator` — entries are **per-kind** controller packages, with the base `config` (`config.Setup`) always first; managed kinds use `<kind>.SetupGated`.

`create api` builds the list from `config.GetResources()` + the resource being created (not yet in config at Scaffold time), deduped. Init keeps seeding the base via its existing static templates; the generators overwrite deterministically once resources exist. Also propagate pipeline errors from both PostScaffolds (completes PR1's fail-loudly).

**Key facts (from code):**
- `config.GetResources() ([]resource.Resource, error)`; new resource added in PostScaffold, so include `*p.resource` explicitly.
- API alias scheme = `group+version` (e.g. `samplev1`); controller package = `strings.ToLower(kind)`.
- Only `createapi.go` references the deleted types; no test references; `file_parser.go` is used only by the two updaters.

### Tasks
1. **TDD** `register_generators_test.go`: dedup-by-GV for API (2 kinds same GV → base + 1 GV); per-kind for controller (base + 2); render contains expected imports/registrations and no duplicates.
2. Implement `register_generators.go` (the two generators + `uniqueGroupVersions` / `controllerPackages` helpers).
3. Rewire `createapi.go` Scaffold: replace the two updaters with the two generators built from `GetResources()+resource`; propagate the PostScaffold pipeline error.
4. Propagate the PostScaffold pipeline error in `init.go`.
5. Delete `api_registration_updater.go`, `template_updater.go`, `file_parser.go`.
6. `make test`, `golangci-lint`, and full `./scripts/e2e-test.sh` (the two-kinds-same-GV case is the regression guard).

### Self-review
Covers spec §3.2 (deterministic register.go, delete parse-and-merge) and the PostScaffold half of §3.6 (fail loudly). Init's static register seed templates are retained (still single-source: the parse-and-merge is gone). Hyphenated-group import aliases are pre-existing behavior, out of scope.
