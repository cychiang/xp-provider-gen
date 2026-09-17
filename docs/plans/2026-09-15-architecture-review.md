# Architecture review backlog: what shipped and how

This document records how `xp-provider-gen` implemented the 2026-09-15
architecture review's 14-item backlog on the `refactor/architecture-review`
branch, and what was deliberately left out. It exists so a later reader does
not have to reconstruct the reasoning from commit messages and session
transcripts.

- **Branch:** `refactor/architecture-review`
- **Baseline:** `386b534` (branch state immediately before backlog item #1's
  commit; this SHA is on `main`)
- **Result:** the 14-item backlog, its review follow-ups, and a second
  completion round (see "Second batch" below), merged into `main` as a
  single squash commit by PR #161.

All commits named below were made on this branch. PR #161 squash-merges all
of them into one commit on `main`, so their SHAs do not resolve there — each
table cites the commit's subject line instead, which stays a valid index
after the squash.

## The 14 backlog items

Each row gives the acceptance criterion exactly as written in the review's
implementation plan (Chinese, verbatim), a one-sentence English summary, the
implementing commit's subject, and the outcome.

| # | 驗收條件（原文） | Summary | Commit | Result |
|---|---|---|---|---|
| 1 | 傳入 NamespacedDomain 為空的 `WithUpjet` 時，產品值等於 `core.NamespacedDomain(domain)`；init 的 scaffold diff 為零；`make test` 綠。 | (`configureProduct` now applies upjet settings before `Configure`, so the domain-derived `NamespacedDomain` survives instead of being overwritten by the empty persisted value.) | `fix: derive the upjet namespaced domain after applying persisted settings` | Done |
| 2 | 空值仍視為 native；`flavor: bogus` 時回傳錯誤；update 拒絕執行。 | (An unrecognized `flavor:` value in `PROJECT` is now a load error instead of silently falling back to native.) | `fix: refuse a PROJECT whose flavor this generator does not know` | Done (follow-up `test: pin the unknown-flavor refusal message and add the license header`: pinned the refusal message wording and added the missing license header) |
| 3 | 檔案移除；`grep -r TEMPLATE_README` 沒有結果。 | (Deleted the stale `TEMPLATE_README.md`, which described a template system that no longer exists.) | `docs: remove the template README that describes a system that no longer exists` | Done |
| 4 | init 與 API 樣板渲染進 memfs 沒有錯誤；路徑集合等於 golden map 加上 generator 路徑；upjet 的 `config/provider.go` root group 正確；刻意製造 `.Kindd` typo 時測試失敗。 | (Added a render smoke test that executes every template of both flavors into an in-memory filesystem, closing the gap where `make test` never rendered a template.) | `test: render every template of both flavors in make test` | Done (follow-up `test: anchor the render test and cover the zero-resource upjet generators`: anchored the test to the new flavor selectors and added zero-resource upjet coverage) |
| 5 | 5 個路徑在兩個 root 間 byte-identical；改其中一份測試就失敗。 | (Added a guard test pinning the five templates both flavor roots must keep byte-identical.) | `test: keep the templates both flavors copy verbatim identical` | Done |
| 6 | upjet 的 `docs/ownership.md` 在 user-owned 欄列出 `go.mod`；測試涵蓋 `upjet_resources.go.tmpl`。 | (Fixed the upjet ownership doc generator to list `go.mod` as user-owned, matching native.) | `fix: list go.mod in the upjet ownership doc` | Done |
| 7 | 只剩 `TemplateFS` 與 `GeneratorBody` 兩個入口；#4 綠。 | (Deleted the `TemplateLoader` wrapper; template bodies are read directly from the embedded FS.) | `refactor: read template bodies straight from the embedded FS` | Done (follow-up `docs: say once, in the right bullet, where template bodies are read`: moved a stray doc sentence about where template bodies are read to the correct bullet) |
| 8 | generator 與依賴集的選擇各只剩一處；`templateRoots` 不再寫死；#4 綠；scaffold diff 為零。 | (Converged flavor selection: `CoreGeneratorsFor` and `DependenciesFor` are now the single choice points, and `core.Flavors` derives both `Valid()` and the template roots.) | `refactor: choose each flavor's generators and dependencies in one place` | Done (follow-up `refactor: pin the known-flavor list and route the render test through the selectors`: pinned the known-flavor list and routed the render test through the new selectors) |
| 9 | 兩處都呼叫同一個 helper；既有測試綠。 | (`core.FileMode` is now the single rule for "`.sh` means 0755", used by both `update.go` and the post-scaffold chmod step.) | `refactor: give the script permission rule one home` | Done |
| 10 | tool-owned 檔被刷新且 diff 合理；user-owned 檔不會以空的 Terraform 值 seed（缺資料時跳過或報錯）；使用 upjet 依賴集；e2e 覆蓋一次 update。 | (`xp-provider-gen update` now supports upjet projects: it refreshes tool-owned files, bumps the upjet dependency set, and — per policy C7 below — does not seed a missing user-owned file with empty Terraform values.) | `feat: let update refresh upjet providers` | Done (follow-up `fix: state update's upjet flow and seeding policy accurately`: corrected the update flow and seeding-policy description in docs and help text) |
| 11 | 對既有 kind 執行 `create api --force` 只覆寫帶 header 的檔案；`external.go` 與 `*_types.go` 不變；有測試。 | (Behavior change: `--force` now only overwrites files carrying the generated-code header; user-owned per-kind files such as `external.go` and `*_types.go` are never overwritten.) | `fix: never let create api --force overwrite user-owned files` | Done |
| 12 | 移除 `runVisible`；新 step 走 allowlist；`make reviewable` 仍即時輸出；失敗時仍附 revertAdvice。 | (Replaced `update.go`'s ad hoc `runVisible` with a streaming automation step that goes through the same command allowlist as every other step.) | `refactor: run update's finalize steps through the allowlisted automation pipeline` | Done (follow-up `fix: name a failed streaming command once, without a trailing space`: fixed a failed streaming command's message being duplicated with a trailing space) |
| 13 | 新 provider 的 Makefile 只剩變數與 include；update 會刷新 `.mk`；兩條 e2e 綠；手動 migration 文件經 upgrade-sim 驗證。 | (Split the generated Makefile into a short user-owned shell — project variables plus one `include` — and a tool-owned `hack/xp-provider-gen.mk` fragment that `update` can refresh.) | `feat: move the generated Makefile's tool pipeline into a refreshable fragment` | Done |
| 14 | 先寫設計文件，確認 create api 的行為變化可以接受後再動工。 (Delivered: the design doc covering sections 1–7, reviewed; implementation awaits maintainer acceptance.) | (Delivered `docs/design/render-apply.md`, proposing a unified `Render`/`Apply` path; implementation is intentionally deferred pending maintainer sign-off.) | `docs: propose one render and apply path for init, create api and update` | Done, design only (follow-up `docs: put Render in engine and price create api's coupling to every kind`: placed `Render` in `engine` and priced `create api`'s coupling to every kind) |

## 驗證方式

What each verification command proves about this branch:

- **`make reviewable`** — `mod-tidy`, `fmt`, `vet`, `golangci-lint`, `gosec`,
  `go test -race ./...`. Proves the code compiles, is idiomatic, has no known
  security lint findings, and every unit test (including the new render smoke
  test, item #4) passes under the race detector.
- **`make e2e-test`** — scaffolds a native-flavor provider with the built
  binary, builds it, and runs the generated provider's own `make e2e`
  (uptest + chainsaw). Proves the native flavor produces a real, buildable,
  deployable provider, end to end.
- **`make e2e-upjet`** — scaffolds an upjet-flavor provider, runs the upjet
  code generator against a real Terraform provider, builds it, and runs it
  (requires network). Proves the upjet flavor is deployable end to end, not
  just that its templates render.
- **`make upgrade-sim`** — simulates bumping the generator version against a
  scaffolded provider that already carries user-added logic. Proves `update`
  preserves user-owned code and correctly refreshes tool-owned files across a
  version bump — the scenario item #13's Makefile split exists to support.
- **scaffold-diff** — a development-time check, not a repository script:
  build the old and new `xp-provider-gen` binaries, scaffold the same
  provider with each, and diff the two output trees (ignoring blank lines and
  comments). It proves a change is behavior-preserving (or shows exactly what
  changed) for refactor-type items.

## 第二批：e2e 覆蓋與缺陷修正

A second completion round landed after this document was first written,
carrying its own task numbers:

| Task | Summary | Commit(s) |
|---|---|---|
| 2 | Added `docs/manual-testing.md`, a step-by-step manual test guide for both flavors. | `docs: add a step-by-step manual test guide for both flavors` |
| 3 | `scripts/e2e-upjet.sh` now runs the generated upjet provider's own `make e2e` (uptest + chainsaw) instead of a hand-written ConfigMap create/update/delete lifecycle, and separately covers the UPDATE reconcile that uptest's own example doesn't exercise. | `test: run the generated upjet provider's own e2e harness`, `fix: cover upjet's UPDATE reconcile and drop the CHAINSAW_ARGS workaround` |
| 4 | Both flavors' e2e scripts gained `create api --force` coverage, and upjet gained `update --adopt` and dirty-tree-refusal coverage (native already had `--adopt`); see also M9 below. | `test: cover --force, --adopt and the dirty-tree refusal on both flavors`, `fix: make the adopt-diff assertion stamp-order-independent, verify 5b's no-op path` |
| 5b | Fixed the "known defect" below: `create api --force` no longer fails with a git error when the scaffold is unchanged. | `fix: let create api --force succeed when the scaffold is unchanged` |
| 7 | Deduplicated the upjet resource aggregator so re-running `create api --force` against an existing kind no longer emits the kind twice (duplicate `Configure` call and include pattern). | `fix: never list an upjet resource twice in the aggregator` |

Also fixed in this round, without its own task number: the `create-test`
chainsaw skeleton (`chainsaw_test.yaml.tmpl`) was missing a `cleanup`
timeout and defaulted to chainsaw's own 30s — too short for an upjet
provider's Terraform-CLI-backed delete confirmation (measured at ~36s). Set
to 2m, matching the skeleton's own delete timeout (`fix: give the
create-test chainsaw skeleton its own cleanup timeout`).

## 當時做的判斷

- **Policy C7 (seeding a missing user-owned file on `update`).** Decided by
  the user on 2026-09-15: on `update`, a rendered user-owned file that does
  not exist on disk is still seeded for **native** projects (unchanged
  behavior), and is **not** seeded for **upjet** projects — because seeding it
  would require Terraform-specific settings only known at `create api` time,
  and writing it with empty values would be actively wrong. An upjet `update`
  lists those paths as skipped instead of guessing.
- **Five items the review considered and rejected — not implemented:**
  - R6 — a typed `RenderContext` (kept the existing template-context shape).
  - R7.3 — renaming `pkg/templates/files/` (the directory name predates the
    upjet flavor and renaming it was judged not worth the churn).
  - R7.5 — templating the "next steps" text printed after scaffolding.
  - R8 — moving off Kubebuilder's `machinery` package entirely.
  - A full `Flavor Profile` struct (a heavier abstraction than the
    `CoreGeneratorsFor`/`DependenciesFor` selector functions item #8 shipped).
- **Item #14 ships a design document only.** The review's own acceptance
  criterion for #14 says to write the design first and get the resulting
  `create api` behavior change accepted before implementing it
  (「先寫設計文件，確認 create api 的行為變化可以接受後再動工」). This branch
  delivers `docs/design/render-apply.md`; the unified `Render`/`Apply` path it
  proposes is a follow-up gated on maintainer acceptance, not implemented
  here.

## 延後未做

The final review found 12 Minor items after the 14 backlog items landed. Two
(M1, M2) were folded into the branch's own pre-completion fix commits
(`refactor: choose the flavor's validator in one place`, `docs: state the
migration's override rule instead of listing variables`) and are already
resolved on this branch. The completion work that produced this document
addressed six more:

- **M3** — `engine.CoreGenerators`, `engine.UpjetCoreGenerators`,
  `automation.NewUpdateFinalizePipeline`, `automation.NewUpjetUpdateFinalizePipeline`
  had no caller outside their own package; unexported to make the `…For`
  selector the only public entry point.
- **M4** — whether `core.Flavor.TemplateRoot()`'s two-way switch should be
  derived from a single table alongside `Valid()`, or left as a documented
  two-place edit. Left as the switch: a lookup table is not simpler than an
  `if` over two flavors, and `Flavors`' doc comment already tells the next
  editor to update both `Valid()` and `TemplateRoot()`.
- **M5** — both generated Makefile fragments called themselves "this
  Makefile", which misleads a reader given they are `include`d, not edited
  directly; corrected to describe them as a fragment.
- **M6** — `scripts/assert-layout.sh` named a caller that no longer invokes
  it; corrected to the current caller.
- **M7** — `.github/workflows/lint.yml` claimed "Renovate bumps both" the Go
  module and the golangci-lint version; verified against Renovate PR #142
  (`e48b299`, which only touched `lint.yml`) that this was wrong and
  `docs/development.md` was the accurate side; corrected `lint.yml`.
- **M9** — the upjet flavor had no `--force` or `--adopt` e2e coverage
  (native had `--adopt` but not `--force` either); both flavors now have
  `--force` coverage, and upjet gained `--adopt` and dirty-tree-refusal
  coverage.

**M8, M10, M11, M12 remain open**: they are cosmetic (missing `t.Run`
subtests, a missing comment explaining two deliberate exec styles), or
coverage-only and true-by-construction (a test asserting an already-guaranteed
invariant, a test depending on `go`'s exit code for an unknown subcommand).
None represent a correctness gap.

## 已知缺陷（backlog，已修）

**`create api --force` on an unchanged kind used to fail with a git error.**
The commit step after scaffolding did not pass `--allow-empty`
(`automation/git.go`, `core/git_runner.go`), so when `--force` regenerated a
kind whose output was byte-identical to what was already on disk, `git
commit` exited non-zero with "nothing to commit" — even though the
scaffolding itself succeeded completely. This was found while planning the
completion work that produced this document. Fixed by
`fix: let create api --force succeed when the scaffold is unchanged`: both
commit paths now stage first and skip the commit when nothing changed.
