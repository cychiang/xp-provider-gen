# PR 5: `update` Command — Implementation Plan

**Goal:** `xp-provider-gen update` refreshes the tool-owned core of an existing provider
(registration, controller wiring/`setup.go`, `main.go`, `config.go`, framework deps) to the
current generator/version, without touching user logic, leaving the result as a reviewable diff.

**Integration:** kubebuilder's plugin interface has no update hook, so `update` is a plain
cobra command added via `cli.WithExtraCommands(...)` in `main.go`. The logic lives in package
`v2` (`update.go`) so it can reuse the factory, generators, scaffolder, and gate.

**Engine — render to memfs, reconcile to disk through the gate:**
1. **Precondition (drift protection):** the working tree must be clean (`git status
   --porcelain` empty); else abort with "commit or stash first". This, plus leaving the
   result uncommitted, makes the diff the review/safety surface (spec §3.5) without needing
   content hashes.
2. **Load** the project config from `PROJECT` via `yaml.New(osFS).Load()`.
3. **Render** into an in-memory FS (`machinery.Filesystem{FS: afero.NewMemMapFs()}`):
   - init templates + static templates + register generators (full resource list);
   - per resource: its API templates (`factory.GetAPITemplates(WithResource(res))`).
4. **Reconcile** memfs → disk: for each rendered path, read the on-disk file and apply
   `core.DecideWrite(exists, onDiskContent)`:
   - `Overwrite` (on-disk is tool-owned/headered) → write the new content;
   - `Skip` (on-disk has no header — `controller.go`, `*_types.go`, `go.mod`, `crossplane.yaml`,
     Makefile) → leave it;
   - `Seed` (absent) → write.
   The gate reads the **on-disk** header, so user files are protected regardless of what the
   (stub) memfs render contains.
5. **Dependencies:** `go get <module>@<version>` for each `versions.GoModDependencies()` entry
   (never overwrites go.mod).
6. **Finalize:** `go mod tidy`, `make generate`, `make reviewable`. **No commit.**
7. Print a summary (overwritten / seeded / skipped counts) and remind to review `git diff`.

### Tasks
1. TDD `update_reconcile_test.go`: reconcile writes tool-owned (headered) memfs files over
   disk, skips headerless disk files, seeds absent ones (drive `reconcile` with two afero FSs).
2. Implement `update.go`: `NewUpdateCommand() *cobra.Command` + `runUpdate()` + `reconcile()`.
3. Wire `cli.WithExtraCommands(crossplanev2.NewUpdateCommand())` in `main.go`.
4. e2e: after create, hand-edit `controller.go`, run `update`, assert (a) the edit survives,
   (b) a tool-owned file (`setup.go`) is refreshed, (c) tree was clean precondition enforced.

### Deferred (noted, not in this PR)
- **Version-stamp provenance / staleness warnings** (spec §3.5 item 3) — small follow-up; the
  core update works without it.
- **Hash-based drift abort** — replaced by the clean-tree precondition + git-diff review, which
  fits the chosen header-ownership model (no per-file hashes to maintain).
