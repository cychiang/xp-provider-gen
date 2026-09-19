# Consistency pass: what changed, and what the gate cannot see

This document records the 2026-09-19 naming-and-consistency pass, what was
deliberately left out, and — most useful to a later reader — the blind spots of
the gate it left behind. It exists so nobody has to rediscover those from
commit messages and session transcripts.

- **Branch:** `refactor/naming-consistency`
- **Baseline:** `d296571` (on `main`)
- **Result:** merged into `main` as a single squash commit by PR #164.
- **Follow-up:** a second PR closed three blind spots this one's review had
  recorded (C4's non-strict-mode facets, C1's spelling variants and file names,
  C7's line-broken strings) and gave the CLI's own name a single home in
  `pkg/version.CommandName`. The gate described below is that later state.

Commits named below were made on that branch; the squash means their SHAs do
not resolve on `main`, so each is cited by subject line.

## Why

`upgrade-sim` did not simulate anything. It is the most end-to-end script in
the repo — real scaffold, real user logic, real `update`, real build and
behavior tests. The only simulated thing is the *new generator version*.

A name that misdescribes its script is not only cosmetic: two of the three e2e
scripts had names that hid their kinship (`e2e-test` / `upgrade-sim` /
`e2e-upjet`), and they were not maintained as kin either — three skeletons,
scratch paths written per script, three stage-output formats, and CI that
invoked some through `make` and some directly.

An audit found 8 must-fix and 13 worth-fixing items across scripts, Makefile,
workflows, docs and Go messages.

## What shipped

| Batch | What | Commit subjects |
|---|---|---|
| 1 | `git mv` the two odd-named scripts into the `e2e-native` / `e2e-upgrade` / `e2e-upjet` family, with their Makefile targets, `.PHONY` entries, `make help` rows, workflow `paths:` filters and job names; add `hack/check-workflow-paths.py` | `refactor: rename e2e scripts into the native/upjet/upgrade family` · `docs: match the native e2e heading's shape to its upjet/upgrade siblings` · `fix: drop unpinned pip install in lint.yml, list check-workflow-paths.py in AGENTS.md` |
| 1b | 36 hard-coded `/tmp` paths in 13 shapes become `/tmp/xpg-e2e-<name>`, with logs and template backups in a sibling `-aux/` directory | `refactor: unify e2e scripts' /tmp paths under an xpg-e2e- prefix` |
| 2 | Doc numbers matched against the code; `ci.yml` → `security.yml`; CI invokes the scripts through `make` | `docs: align architecture/testing/WORKFLOWS docs with the code they describe` · `ci: rename ci.yml to security.yml, call e2e scripts via make targets` |
| 3a | One script skeleton (`set -euo pipefail`, `lib.sh`, one `step_header`/`section_header` pair); five terminology pairs collapsed | `refactor: unify e2e script skeleton, harden e2e-upgrade against strict mode` · `ci: drop redundant Build binary steps, sync run: comments, scope SARIF categories` · `docs:` and `refactor:` wording commits · `fix: harden e2e script failure paths and finish terminology cleanup` |
| 3b | Error strings normalized to the gerund form 32 other error strings already used; `"PROJECT is not usable"` written once instead of four times; one success line per command instead of two | `refactor: dedupe repeated "PROJECT is not usable" error string` · `refactor: print one success line per command, not two` · `fix: normalize error strings to gerund form, not infinitive` |
| 5 | `hack/check-consistency.sh`, wired into `make reviewable` and `lint.yml` | `ci: add hack/check-consistency.sh as the repo's consistency gate` · `fix: close the consistency gate's fail-open and widen its checks` |

Three names moved, and their old spellings are gone everywhere except this
directory:

| Before | After |
|---|---|
| `scripts/e2e-test.sh` · `make e2e-test` | `scripts/e2e-native.sh` · `make e2e-native` |
| `scripts/upgrade-sim.sh` · `make upgrade-sim` | `scripts/e2e-upgrade.sh` · `make e2e-upgrade` |
| `.github/workflows/ci.yml` (named `CI`, ran only gosec and Trivy) | `.github/workflows/security.yml` (named `Security`) |

`set -euo pipefail` was a behavior change, not a style choice: unguarded `$1`
aborts a no-argument run; `cmd | grep -q` turns a match into a failure when
`grep` exits first and the writer takes SIGPIPE; and one failure-diagnostic
path aborted before it could report. Those were found by static scan, by a
minimal reproduction during review, and by actually running the scripts —
one each.

## Deliberately not done

- **Unifying the three test-naming conventions across 99 tests** — cost
  exceeds the benefit.
- **Renaming workflow files wholesale** — breaks Actions history and the
  muscle memory for `gh workflow run`. Only `ci.yml`, whose name was wrong,
  was renamed.
- **`!` vs `.` on the three terminal success messages** — three messages, three
  different sentence shapes; there is no majority to converge on.
- **`docs/plans/` itself** — a historical record, left as written. The gate
  exempts this path, which also means a plan written later is never checked.

## The gate, and what it cannot see

`make check-consistency` (also a dependency of `make reviewable`, and a step in
`lint.yml`) runs eight checks. Each one turns red on its own violation and only
its own; that was verified by breaking each in turn. A check that *cannot
search* — a malformed pattern, an unreadable file — reports failure rather than
passing, because the first two review rounds found exactly that fail-open.

What it does not catch. This list is the point of this document: each entry was
found by deliberately breaking the check, so treat it as the gate's contract
rather than as a list of bugs.

- **C7's bare-verb list is a denylist, not a rule.** It holds the bare form of
  every gerund the repo already opens an error string with, plus `failed to` /
  `unable to` / `could not`. A verb nobody has used yet passes; extend the list
  when one shows up. C7 also only scans `*.go` — never `*.go.tmpl`, which is the
  generated project's code with its own conventions — and only `fmt.Errorf` /
  `errors.New`, not `t.Errorf` test messages.
- **C1 blocks more file names than it should.** Matching `e2e[-_]?test` also
  rejects the Go-idiomatic `*_e2e_test.go`, and the standalone-word `sim` rule
  rejects `sim.go`. Neither exists today; narrow the pattern when one is really
  needed. `e2e test` with a space is deliberately *not* a legacy name — fifteen
  lines of ordinary English say it.
- **C2's regex only sees simple target names** — not ones containing `.` or
  `%`, and not multi-target rules. The Makefile has none today.
- **C3 only sees targets carrying a `## ` description.** A target without one is
  invisible to both `make help` and this check. `lint-install` is the only such
  target, deliberately.
- **C5 does not follow indirection.** `T=/tmp; "$T/x"` is invisible. Its job is
  to keep `rm -rf` away from anything unprefixed, not to audit every path.
- **C6 cannot see a term broken across lines.**
- **C7 cannot see a string the opening parenthesis does not immediately
  precede** — a trailing `// comment` or a blank line between them hides it.
  gofmt produces neither.
- **C2 and C3 abort instead of reporting.** If the `Makefile` is missing,
  malformed, or its `help` recipe fails, the script dies where it stands rather
  than printing `FAIL`. Still fail-closed, and deliberately not wrapped: a
  broken `Makefile` means `make check-consistency` could not have started
  either.
- **`git grep --untracked` sees your local scratch files.** An untracked
  `notes.md` containing a retired term turns `make reviewable` red locally. CI
  is unaffected, and the message names the file.
- **`docs/plans/` is exempt** — including this document, and including any plan
  written later.

Two properties it does hold, which cost several review rounds to get right and
are worth preserving in any new check:

- **Every check is held to an expected value, not to agreement among peers.**
  All three e2e scripts sliding back to `set -e` together must fail, not pass.
- **A check that cannot run reports failure** — every check but the two
  `Makefile`-reading ones above. `git grep` exits 0 for hits, 1 for none, and
  anything else for "could not search"; conflating those three made a broken
  pattern look green three times, in three different checks, each caught only
  by deliberately breaking it.

## Verification

- `make reviewable` (which now includes `make check-consistency`) — green.
- `make e2e-native`, `make e2e-upgrade`, `make e2e-upjet` — all exit 0 under the
  new strict mode, including each generated provider's own uptest + chainsaw
  suite against a real kind cluster.
- **scaffold-diff** — a development-time check, not a repository script: build
  the baseline and branch binaries, scaffold with each, and compare three
  surfaces — the generated tree, the git history it creates, and what it
  prints. Tree and history were byte-identical for both flavors; stdout
  differed by exactly the duplicate success lines this pass removed. The
  generator's output did not change.
- PR #164's 17 GitHub Actions checks — none failing.

## One manual step this left behind

A SARIF upload with no `category:` gets one derived from the workflow file path
plus the job name. Renaming `ci.yml` to `security.yml` therefore stranded the
old key `.github/workflows/ci.yml:security`, and the two configurations under
it — one per tool, gosec and Trivy — have to be deleted by hand under Security →
Code scanning → Tool status. Both uploads now pass an explicit `category:`
(`gosec`, `trivy`), which no longer contains the path, so a future rename will
not repeat this.
