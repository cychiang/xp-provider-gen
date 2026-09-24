# Program: update.go split, release automation, downgrade refusal, documentation

What shipped in #170–#173, what was deliberately left out, and what the review gate
caught. Written so a later reader can tell a decision from an accident.

## What shipped

| PR | Change |
|---|---|
| #170 | `update.go` (732 lines) split by responsibility into `update.go`, `reconcile.go`, `adopt.go`, `tfversion.go`, tests to match. Pure move. |
| #171 | `update` and `update --adopt` refuse when an older clean release runs on a project a newer one stamped. `init` stamps the generator version. The "may mean this binary is older" hint, which fired on every legitimate upgrade that removed a file, is gone. |
| #172 | Releases: one dry-run-by-default button running git-cliff (version and notes) and GoReleaser (build, archives, checksums, GitHub Release), plus a PR-title check. Replaces a hand-written `release.yml` that had never run. |
| #173 | Every document made true again against the code; duplicates cut to one canonical copy; AGENTS.md rewritten; check-consistency gains C9, which fails on a broken markdown link or anchor. |

## Decisions, and what they cost

### The split is proven, not asserted

A line-multiset diff was the first proof and it was blind to two statements swapping
places inside a function. The proof that shipped compares one record per top-level
declaration, doc comment included, and fails closed: an earlier version, run under
macOS's `/bin/bash` 3.2, fed `diff` a broken brace expansion and still printed a
passing `0`. The positive proof is the three e2e suites; scaffold-diff never runs
`update`, so it is only the negative check.

### Only clean releases are ordered

`semver.Compare("v0.2.0-5-gabc1234", "v0.2.0")` is `-1`: git-describe output is valid
semver but a prerelease, so comparing it would refuse a maintainer's own build. And
`semver.IsValid` accepts `v0.2.0-dirty`. So both sides must match `^v\d+\.\d+\.\d+$` or
nothing is compared. There is no `--allow-downgrade` flag: the refusal prints the
three-step exit (edit PROJECT's `version:`, commit, update), which leaves the downgrade in
git history. Accepted blind spot: a project stamped by a development build is never
protected against an older release.

### Release tooling is off the shelf

The maintainer asked for the most widely used open-source tools rather than a custom
one; a hand-written `hack/release` had been designed and reviewed, and was discarded
before any code existed. git-cliff was chosen over release-please because it gives a
real button (release-please's trigger is merging a bot's PR, and it commits a
CHANGELOG.md) and reaches `v0.1.0` with no special case; semantic-release fixes the
first version at 1.0.0. GoReleaser does the building. The existing docker job stays,
because GoReleaser's image builder wants a Dockerfile that only copies prebuilt
binaries and ours is shared with CI and Renovate.

### The first release note is one line

Grouping the whole history would list about 47 lines. The first release body is
`Initial release.`; notes are grouped from the second release on.

### English is enforced twice, and approximately

A regex cannot detect a language. The PR check allows printable ASCII plus six
typographic characters (the repo's own titles use em dashes); `cliff.toml` normalizes
those six and the release job refuses notes containing any other non-ASCII.

### Documents are cut only by rule

Content is removed only when it is wrong, fully duplicated (unique parts moved first),
or a one-time migration recipe for a population recorded as zero — and only if links
are fixed in the same change. No whole document met that bar; the pass trimmed and
merged instead.

## Deliberately not done

- **Publishing `v0.1.0`.** Everything up to the tag has run: a dry run on GitHub built
  all five platforms and both image architectures and published nothing. Cutting the
  first release is the maintainer's call.
- **Making the PR-title check required.** Without a required status check it is
  advisory: a red check can still be merged, and a squash title edited in the merge
  dialog is never checked. That is a repository setting, left to the maintainer.
- **`docs/design/render-apply.md`.** The maintainer is deciding its fate separately;
  it was not edited, moved or counted as stale.

## What the review gate caught

Plans went back to review until a round returned no blocker and no should-fix; that
took 4 rounds for the release design, 3 for the custom-tool plan it replaced, 3 for the
open-source version, 3 for the split, 2 for the downgrade plan, and 3 for the docs.
The findings worth not repeating:

| Where | What |
|---|---|
| Downgrade design | The existing hint fired on every legitimate upgrade that removed a file — the feature's main use. No e2e saw it, because the upgrade test rebuilt from the same commit. |
| Release design | `github.event.inputs.dry_run` is a string; `"false" == false` is `NaN`, so a publish step gated on it would be skipped silently with the run green. |
| Release design | With notes computed by one tool twice — before the tag and on the tag — "the previous tag" became the tag itself and the body came out empty. |
| Release (OSS) | Notes written to the workspace make GoReleaser's real path fail `errors/dirty`, and `--snapshot` skips that check — the dry run could never show it. |
| Release (OSS) | `changelog.disable: true` skips the pipe that loads `--release-notes`; every body would have been empty. Found by the implementer, confirmed in GoReleaser's source. |
| Release (OSS) | Without `release.target_commitish`, GitHub creates the tag at the branch HEAD, not at the commit that was built. |
| Release (OSS) | A git-cliff `body =` rule never sees `BREAKING CHANGE:`; git-cliff parses it as a footer. |
| Split proof | The fail-open brace expansion under `/bin/bash` 3.2 above. |
| Downgrade plan | `update --adopt` appended "run 'git reset --hard'" to every error, including a dirty tree where nothing had changed — and it would have contradicted the refusal's own "nothing to revert". |
| e2e | `e2e-upjet.sh` asserted PROJECT carried a version with `^ *version:`, which matched the top-level `version: "3"` and could never fail. |
| Acceptance | Two break tests went red from a compile error, not from the test — a red that proves nothing. |
