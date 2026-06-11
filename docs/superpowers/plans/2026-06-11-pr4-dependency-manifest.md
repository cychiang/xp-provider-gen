# PR 4: Dependency Manifest — Implementation Plan

**Goal:** Make the generated provider's dependency versions a single, Renovate-tracked
source of truth instead of invisible literals in `go.mod.tmpl`.

**Architecture:**
- `pkg/versions/dependencies.yaml` — the direct deps a generated provider declares.
- `pkg/versions/versions.go` — embeds the manifest, exposes `GoModDependencies()` and
  the `GoVersion` constant.
- `engine.GoModGenerator` — renders `go.mod` from the manifest (seed-once, no header,
  `IfExistsAction = SkipFile`; go.mod is user-owned — `update` bumps framework versions
  via `go get`, never by overwrite). Replaces the static `go.mod.tmpl`.
- `renovate.json` custom (regex) manager on `dependencies.yaml` with the `go` datasource,
  so every entry gets its own upstream bump PR gated by CI + e2e.

### Tasks
1. TDD `pkg/versions` (manifest parses; includes crossplane-runtime).
2. TDD `GoModGenerator` (renders module/go/require; no header; seeds once).
3. Wire the generator into init scaffolding; delete `go.mod.tmpl`.
4. Add the Renovate custom manager; validate JSON + regex captures all entries.
5. e2e: generated go.mod builds and `go mod verify` passes (already in the script).

### Notes
- `tool` directives (controller-gen, angryjet) stay unversioned in the template; `go mod
  tidy` resolves them. Only the explicit direct requires are manifested (YAGNI: transitive
  deps are Renovate/tidy's job).
- PR5's `update` reads this same manifest to apply framework bumps to existing providers.
