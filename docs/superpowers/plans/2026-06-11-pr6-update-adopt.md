# PR 6: `update --adopt` + provenance — Implementation Plan

**Goal:** Let an existing/older provider (generated before the ownership contract) opt into
the upgrade workflow: stamp the generator version into PROJECT and write the
`DO NOT EDIT` header onto its recognized tool-owned files, so plain `update` then works.
Also stamp provenance on every `update` (delivers the PR5-deferred version stamp).

**Architecture (reuses PR5):**
- `update --adopt`:
  1. clean-tree precondition + load PROJECT;
  2. render the template set into memfs (`renderToMemFS`);
  3. for each rendered file that is tool-owned (`core.IsToolOwned`), if the on-disk file
     exists and lacks the header, insert it before the `package` clause;
  4. stamp `{version}` into PROJECT under the plugin key via `EncodePluginConfig` + save;
  5. leave the result uncommitted (review as a diff); print what was adopted.
- `runUpdate` also stamps provenance at the end (so the recorded version stays current).

**Key APIs:** `cfg.EncodePluginConfig(pluginName, provenance{Version})`, `version.Get().Version`,
`store.Save()`. Header insertion lands within the first 1024 bytes (before `package`), where
`IsToolOwned` scans.

### Tasks
1. TDD `insertGeneratedHeader` (inserts before package; idempotent if already present) and
   `adoptHeaders` (adds header to headerless tool files, leaves user files + already-headered).
2. Implement: `--adopt` flag, `runAdopt`, `adoptHeaders`, `insertGeneratedHeader`,
   `stampProvenance`; call `stampProvenance` from `runUpdate` too.
3. `go mod tidy`; build/test/lint.
4. e2e: strip the header from a tool file (simulate an old provider), run `update --adopt`,
   assert the header is restored and PROJECT gains the version stamp; then `update` succeeds.

### Notes
- Adoption only marks EXISTING tool files; new tool files are seeded by a subsequent `update`.
- User files (controller.go, *_types.go) are never adopted (their memfs render lacks the header).
