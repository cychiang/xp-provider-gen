# Maintainability and full e2e: what the evidence supports

A plan, written before its first unit shipped. Its units land as separate PRs, so the git
log says which are done. It says which architecture changes make `xp-provider-gen` cheaper
to maintain and extend for both flavors, which areas should stay as they are, and what the
maintainer decided. The box-by-box execution checklist lives outside the repository.

Baseline is `main` at `28ecb40`. Measurements were taken on 2026-11-04 and 2026-11-05.
"Measured" means run or read from a real artifact. "Predicted" means inferred from a
replay on paper, to be checked when the PR exists.

## The conclusion

The generator's core is sound and should stay. Package dependencies have no cycles,
files are small, and the render-apply work holds. Three things cost more than they
should:

- **Proof.** The e2e can report success without proving what it names.
- **Facts with several homes.** Versions live in places the manifest does not control.
- **A CLI wrapper we fight.** This is the weakest case of the three, and it is optional.

One assumption did not survive the evidence. Commit counts first suggested CI was the
biggest maintenance cost. 51 of the 56 `ci` commits in the last 120 are Renovate
action bumps, not human work, and no CI run has failed since 2026-09-24.

## Three defects found on the way

All three were reproducible on `main` at `28ecb40`.

| Defect | Evidence |
|---|---|
| `init` exits 1 on host Go 1.27.3. golangci-lint 2.13.2, built with go1.27.0, cannot type-check the newer standard library. `update` runs the same lint. CI hides it, because it installs the pinned Go. | Native `init` fails with `could not import reflect`. `GOTOOLCHAIN=go1.26.10 make reviewable` exits 0, changing only that variable. |
| After that failed `init` nothing is committed, yet `create api` proceeds and commits the whole tree as `Add Widget managed resource`. | Reproduced by running the two commands in order. |
| `xp-provider-gen edit` inside a project panics with a nil dereference in Kubebuilder's `cli/cmd_helpers.go`. | Reproduced. The skill already calls `edit` unsupported. |

This corrects a project belief. The earlier finding was that a toolchain pin cannot cap
a newer host, and that bumping golangci-lint is the only lever. That holds for the
`toolchain` directive in `go.mod`. It does not hold for the `GOTOOLCHAIN` environment
variable, which does cap it. Exporting it from the tool-owned fragment needs no pin in
any user-owned file:

```make
export GOTOOLCHAIN ?= go1.26.10+auto
```

With that line and nothing else changed, on host Go 1.27.3, native `init`, `create api`
and `make reviewable`, and upjet `make generate` and `make reviewable`, all exit 0.
Removing the line returns upjet `make lint` to exit 2. Because of `?=`, any ambient
`GOTOOLCHAIN` wins, including `local` and `auto`, and a developer who exports either
gets the old failure back.

## What the current e2e proves

| Stage | Native | Upjet |
|---|---|---|
| `init`, `create api`, generate, build | Proven in CI | Proven in CI |
| Build xpkg, install in kind, Healthy | Proven in full CI. Skips with exit 0 if the daemon is missing. | Same |
| `Ready` and `Synced` | Proven, but against a stub that compares two fields of its own resource and sets `Available` itself | Proven against a real ConfigMap through Terraform |
| Update a managed resource | Pause test only. Uptest logs `Skipping update step`. | Proven |
| Delete, external object gone | Not proven, because no external object exists | Proven |
| Update from a released binary | Not proven. `e2e-upgrade.sh` mutates local templates and fakes `VERSION`. | Not proven |

Two checks cannot fail what they name. `e2e-upgrade.sh:295` chains `make generate && make
lint && log_success`, so a lint failure passes under `set -e`. `e2e-native.sh:366` greps
`plugins:`, which `init` already wrote. Separately, a skipped run prints
`All native E2E tests completed successfully!`.

## Full e2e, defined

Full means this declared list, in order, for each flavor. A required stage that does not
pass fails the run. In full mode a missing prerequisite is a failure before the first
stage. Fast mode exists, and reports `VERDICT PARTIAL` with each stage it did not run.

| Stage | Proves |
|---|---|
| `init`, `create-api` | Run by the binary under test or by a real released binary |
| `generate` | Native `make generate`, or upjet's real Terraform schema pipeline |
| `user-logic` | Overlays author code, tidies, lints, commits |
| `update-to-head` | Release cases only. User-owned bytes unchanged, go directive and PROJECT stamp correct. |
| `update-noop`, `create-api-refresh`, `adopt` | Head only. The per-PR `update` coverage `e2e-native.sh` has today. |
| `stock-e2e` | Head only. The unmodified scaffold's `make e2e`, the author's entry point. |
| `build-xpkg`, `install-in-kind`, `provider-config` | Package installs and the provider is Healthy |
| `mr-ready-synced` | `Ready` and `Synced`, and the external value matches, read around the provider |
| `mr-update`, `drift-corrected` | External value follows a spec patch, and a hand edit is restored |
| `mr-delete`, `external-gone` | The resource is gone, then the external object is gone |
| `create-test` | A scaffolded chainsaw test passes |

The cells are `{native, upjet} x {head, newest release}` (amended 2026-11-07; the record
first listed v0.2.0 and v0.3.0, see decision four).

Native needs an honest external system without changing what authors receive. The
harness overlays the user-owned `client.go` and `external.go` in the scaffold under test,
with a ConfigMap client built from `ClientConfig.Credentials`. That exercises the
credential path in `connector.go`, keeps state outside the managed resource, and shares
one `kubectl` check with upjet. The shipped stub stays a stub.

This ran. A 246-line prototype and a 138-line overlay passed every native stage from
head, v0.2.0 and v0.3.0, in 4.5 to 6.5 minutes on a warm cache. A no-op `Update` failed
`mr-update`. A `Delete` that lies passed `mr-delete` and failed `external-gone`. Upjet
providers from real v0.2.0 and v0.3.0 binaries took `update` and `make build` from `main`
plus the P1 line, so `update` delivers the fix to existing providers.

Not yet run: any upjet stage in the new harness, `--mode`, preflight, `--matrix`, and all
CI timings. A real limit: a scaffold from an old binary pulls today's `build` head, and
head refreshed only one tool-owned file, so upgrade stages exercise a small delta today.

## Scenarios

| Scenario | Today (measured) | After (predicted) | How to check |
|---|---|---|---|
| Add a template | Template and golden map. Three tests name the fix. | Unchanged. This already works. | Repeat the experiment |
| Change plumbing in both flavors | Two trees, five identical files guarded | Unchanged, minus pin lines | `git grep -c CROSSPLANE_VERSION pkg/templates` |
| Absorb a breaking upstream release | #105 touched 4 templates, #187 touched 6 files | #105 unchanged, #187 falls to 1 file | Replay on a scratch branch |
| Bump a pin | #187 6 files, #189 7, #191 6 | One manifest line for provider pins | Replay, and count `renovate.json` managers |
| Change a seam signature | No mechanism, no detection | No mechanism. Release cells detect the break. | Rename `NewExternal`, and `--from=v0.3.0` must fail |
| Change `update` behavior | #192 5 files, no wiring test possible | #192 still 5 files, with a wiring test | Remove the `applyGoDirective` call, and a test must fail |
| Know whether a flavor works | Read three scripts, and a skip reads as success | `--list` and one verdict line | Stop Docker, and full mode must exit 1 |
| Upgrade from an older release | Zero real releases | Four flavor and release pairs | Run the cells |

Replays are predictions until each PR runs them. #187's count of one file also assumes
P9 lands and that a new Renovate rule covers the new manifest keys.

## The units

| PR | Change | Breaks for existing providers |
|---|---|---|
| P1 | `GOTOOLCHAIN` in both fragments, plus a job on `stable` Go so it cannot return unseen | `update` rewrites the fragment. Offline hosts need `GOTOOLCHAIN=local`. |
| P2 | `create api` refuses a repository with no commit. `edit` and `create webhook` refuse cleanly. | Nothing valid |
| P3 | Today's scripts fail instead of skip in full runs. Fix the two blind checks. If the harness is Go, it stops there, since P5 to P7 and P11 replace the rest. | Nothing |
| P4 | `runUpdate` and siblings take `afero.Fs`, so wiring is testable | Nothing |
| P5 | Harness and native lifecycle. Delete `e2e-native.sh`. | Nothing |
| P6 | Real releases for native. Delete `e2e-upgrade.sh`. | Nothing |
| P7 | Upjet in the harness. Delete remaining shell and `hack/envtest-provider-check`. | Nothing |
| P8 | One home for provider pins. Delete C10. | `update` refreshes literals it owns. Native angryjet stops floating. |
| P9 | Decouple generator Go from provider Go, reversing #191. Droppable. | Nothing |
| P10 | Pin `build` for new providers, report drift on `update`, and assert it in the harness | New providers get a pinned `build` |
| P11 | Triggers fail toward running, and the matrix is derived. Delete C8. | Nothing. CI cost rises. |
| P12 | Cut docs that restate code | Nothing |
| P13 | Plain cobra for `cli.New`. Droppable. | `--plugins`, `--project-version`, `edit` and `create webhook` disappear. PROJECT is unchanged. |

P1 comes first because later verification needs it on any host with a newer Go.

## Left alone, on purpose

- **The two template trees.** Identical changed lines in the six both-tree commits were 2
  of 2, 0 of 9, 0 of 7, 78 of 283, 6 of 70 and 0 of 2. A shared base would add precedence
  rules to fix a duplication that mostly is not one.
- **`machinery`.** An 85-line executor matched output byte for byte. About 100 saved lines
  move no scenario, so R8 stays rejected. Dropping `cli.New` is a separate decision no
  record covers: R8 is scoped to `machinery`.
- **Flavor Profile, typed render context, seam migration engine.** No new evidence.
- **`ownership_doc_generator.go:98`** stays a known second reader of ownership. The golden
  test covers it.
- **The shipped `external.go` stub.** Making it real is a product decision every author
  would inherit.

## Decisions for the maintainer

The maintainer answered on 2026-11-06. Take each recommendation below. Generated
providers have no users yet, so a change may break them, and no change needs a shim or a
migration path for that reason alone. The ten live lanes per PR stay.

1. **Breaking changes.** Three are on the table: removing the inert flags and commands in
   P13, `update` moving a provider's `build` gitlink, and offline hosts opting out of the
   toolchain line. Recommendation: take P13 last, have `update` report drift only, and
   document the opt-out.
2. **Scope.** Is a third flavor or another extension axis in scope? Recommendation: no.
   Nothing here designs for one.
3. **Upjet's live test target.** Recommendation: keep `hashicorp/kubernetes` against the
   kind cluster, with no cloud credentials.
4. **CI cost.** Today's medians are 442 seconds native and upgrade, and 598 seconds upjet.
   Recommendation: head cells per PR, release cells daily and before a release, and a set
   budget for P11's per-PR rule.

   Amended 2026-11-07 by the operator. The "daily" above had already become weekly by the
   operator's order, and the release cells now change further. Upgrades test the newest
   published release in each flavor (`--from=latest`), on the weekly schedule and on
   `workflow_dispatch`, and not on pull requests. The operator's words: "You can skip
   upgrade from v0.1.0 to v0.3.0 or anything like that, you dont't need to consider any
   backward compatiable just make sure the new architecture and implementation is easy to
   maintain in the future", and then "will the upgrade path becomes much easy to test and
   measurable?" Older releases are not tested, no list of releases is kept by hand, and
   each run can write a JSON report (`--report`) of stage times and upgrade counts, which
   makes the path measurable. A tag that needs a toolchain override is the one exception
   the harness lists (v0.3.0 needs `go1.26.8`).
5. **Harness language.** Recommendation: Go, because the three blind checks above are
   shell-shaped failures, and `scripts/check-go-version` is the precedent. Stage names, the
   verdict rule and exit codes survive any choice. If the answer is shell, P3 becomes the
   lasting fix and P5 to P7 shrink.
6. **The on-hold `xp-provider-kit` rename.** Recommendation: it does not affect this work.
7. **How much ships, and in what order.** Recommendation: release P1 and P2 at once as a
   patch, since the first defect stops `init` on a current Go. Treat P9 and P13 as optional.
   P9 reverses #191, which deliberately made the Go pins bump together, so taking it means
   accepting that reversal.

## Done means

All of these hold, shown by their output:

- Scenario replays show the predicted reductions, or the plan says why not.
- Scaffold diffs for both flavors show only declared changes.
- A provider from the newest published release upgrades cleanly in both flavors, run weekly
  and on dispatch rather than on pull requests (amended 2026-11-07), and the run reports
  what it compared and what differed.
- Both flavors reach `Ready` and `Synced` in kind and clean up the external object.
- A deliberately broken template fails the e2e, and every new check has been break-tested.
- `make reviewable` passes, and CI passes on a clean runner with zero skipped required
  stages.
