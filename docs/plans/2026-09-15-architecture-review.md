# Architecture review backlog: what shipped and how

This document records how `xp-provider-gen` implemented the 2026-09-15
architecture review's 14-item backlog on the `refactor/architecture-review`
branch, and what was deliberately left out. It exists so a later reader does
not have to reconstruct the reasoning from commit messages and session
transcripts.

- **Branch:** `refactor/architecture-review`
- **Baseline:** `386b534` (branch state immediately before backlog item #1's
  commit)
- **Result:** `440efd2` (branch HEAD after all 14 items and their review
  follow-ups)

## The 14 backlog items

Each row gives the acceptance criterion exactly as written in the review's
implementation plan (Chinese, verbatim), a one-sentence English summary, the
commit that implements it, and the outcome.

| # | 驗收條件（原文） | Summary | Commit | Result |
|---|---|---|---|---|
| 1 | 傳入 NamespacedDomain 為空的 `WithUpjet` 時，產品值等於 `core.NamespacedDomain(domain)`；init 的 scaffold diff 為零；`make test` 綠。 | (`configureProduct` now applies upjet settings before `Configure`, so the domain-derived `NamespacedDomain` survives instead of being overwritten by the empty persisted value.) | `c065ec4` | Done |
| 2 | 空值仍視為 native；`flavor: bogus` 時回傳錯誤；update 拒絕執行。 | (An unrecognized `flavor:` value in `PROJECT` is now a load error instead of silently falling back to native.) | `f8ae717` | Done (follow-up `8f6c4a6`: pinned the refusal message wording and added the missing license header) |
| 3 | 檔案移除；`grep -r TEMPLATE_README` 沒有結果。 | (Deleted the stale `TEMPLATE_README.md`, which described a template system that no longer exists.) | `f8af4d6` | Done |
| 4 | init 與 API 樣板渲染進 memfs 沒有錯誤；路徑集合等於 golden map 加上 generator 路徑；upjet 的 `config/provider.go` root group 正確；刻意製造 `.Kindd` typo 時測試失敗。 | (Added a render smoke test that executes every template of both flavors into an in-memory filesystem, closing the gap where `make test` never rendered a template.) | `1f97552` | Done (follow-up `9ad9f6d`: anchored the test to the new flavor selectors and added zero-resource upjet coverage) |
| 5 | 5 個路徑在兩個 root 間 byte-identical；改其中一份測試就失敗。 | (Added a guard test pinning the five templates both flavor roots must keep byte-identical.) | `d15ebeb` | Done |
| 6 | upjet 的 `docs/ownership.md` 在 user-owned 欄列出 `go.mod`；測試涵蓋 `upjet_resources.go.tmpl`。 | (Fixed the upjet ownership doc generator to list `go.mod` as user-owned, matching native.) | `a3b3d56` | Done |
| 7 | 只剩 `TemplateFS` 與 `GeneratorBody` 兩個入口；#4 綠。 | (Deleted the `TemplateLoader` wrapper; template bodies are read directly from the embedded FS.) | `62822c7` | Done (follow-up `67c0a3c`: moved a stray doc sentence about where template bodies are read to the correct bullet) |
| 8 | generator 與依賴集的選擇各只剩一處；`templateRoots` 不再寫死；#4 綠；scaffold diff 為零。 | (Converged flavor selection: `CoreGeneratorsFor` and `DependenciesFor` are now the single choice points, and `core.Flavors` derives both `Valid()` and the template roots.) | `f6c10aa` | Done (follow-up `9199fcd`: pinned the known-flavor list and routed the render test through the new selectors) |
| 9 | 兩處都呼叫同一個 helper；既有測試綠。 | (`core.FileMode` is now the single rule for "`.sh` means 0755", used by both `update.go` and the post-scaffold chmod step.) | `8d62cc5` | Done |
| 10 | tool-owned 檔被刷新且 diff 合理；user-owned 檔不會以空的 Terraform 值 seed（缺資料時跳過或報錯）；使用 upjet 依賴集；e2e 覆蓋一次 update。 | (`xp-provider-gen update` now supports upjet projects: it refreshes tool-owned files, bumps the upjet dependency set, and — per policy C7 below — does not seed a missing user-owned file with empty Terraform values.) | `57b09c4` | Done (follow-up `19efef1`: corrected the update flow and seeding-policy description in docs and help text) |
| 11 | 對既有 kind 執行 `create api --force` 只覆寫帶 header 的檔案；`external.go` 與 `*_types.go` 不變；有測試。 | (Behavior change: `--force` now only overwrites files carrying the generated-code header; user-owned per-kind files such as `external.go` and `*_types.go` are never overwritten.) | `15bf8b4` | Done |
| 12 | 移除 `runVisible`；新 step 走 allowlist；`make reviewable` 仍即時輸出；失敗時仍附 revertAdvice。 | (Replaced `update.go`'s ad hoc `runVisible` with a streaming automation step that goes through the same command allowlist as every other step.) | `4a242b3` | Done (follow-up `85d26ec`: fixed a failed streaming command's message being duplicated with a trailing space) |
| 13 | 新 provider 的 Makefile 只剩變數與 include；update 會刷新 `.mk`；兩條 e2e 綠；手動 migration 文件經 upgrade-sim 驗證。 | (Split the generated Makefile into a short user-owned shell — project variables plus one `include` — and a tool-owned `hack/xp-provider-gen.mk` fragment that `update` can refresh.) | `024bc8e` | Done |
| 14 | 先寫設計文件，確認 create api 的行為變化可以接受後再動工。 (Delivered: the design doc covering sections 1–7, reviewed by arch-reviewer; implementation awaits maintainer acceptance.) | (Delivered `docs/design/render-apply.md`, proposing a unified `Render`/`Apply` path; implementation is intentionally deferred pending maintainer sign-off.) | `a23b2ab` | Done, design only (follow-up `361d251`: placed `Render` in `engine` and priced `create api`'s coupling to every kind) |

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
- **scaffold-diff** — a development-time check: build the old and new
  `xp-provider-gen` binaries, scaffold the same provider with each, and diff
  the two output trees. It proves a change is behavior-preserving (or shows
  exactly what changed) for refactor-type items. This script lives in the
  session's scratch directory, not in this repository.

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
(`11228a5`, `2c1773b`) and are already resolved at `440efd2`. The completion
work that produced this document addressed six more:

- **M3** — `engine.CoreGenerators`, `engine.UpjetCoreGenerators`,
  `automation.NewUpdateFinalizePipeline`, `automation.NewUpjetUpdateFinalizePipeline`
  had no caller outside their own package; unexported to make the `…For`
  selector the only public entry point.
- **M4** — whether `core.Flavor.TemplateRoot()`'s two-way switch should be
  derived from a single table alongside `Valid()`, or left as a documented
  two-place edit. Decided by the completion work's own KISS-first tiebreak:
  see that change's commit message for which one and why.
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

**M8, M10, M11, M12 remain open**, per the final review's own triage
(`sdd/final-review.md` §5, "Can stay"): they are cosmetic (missing `t.Run`
subtests, a missing comment explaining two deliberate exec styles), or
coverage-only and true-by-construction (a test asserting an already-guaranteed
invariant, a test depending on `go`'s exit code for an unknown subcommand).
None represent a correctness gap.

## 已知缺陷（backlog）

**`create api --force` on an unchanged kind fails with a git error.** The
commit step after scaffolding does not pass `--allow-empty`
(`automation/git.go`, `core/git_runner.go`), so when `--force` regenerates a
kind whose output is byte-identical to what is already on disk, `git commit`
exits non-zero with "nothing to commit" — even though the scaffolding itself
succeeded completely. This was found while planning the completion work that
produced this document. **Fixed in this delivery by the completion plan's
Task 5b** (`fix: let create api --force succeed when the scaffold is
unchanged`); confirm that commit is present on this branch before treating
the defect as closed.
