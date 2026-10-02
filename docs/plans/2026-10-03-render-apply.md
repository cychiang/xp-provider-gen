# Program: one render path, one write rule (render-apply)

What shipped in #176–#181, what was deliberately left out, and what the review gate
caught. The proposal is [`docs/design/render-apply.md`](../design/render-apply.md); this
record says what was decided. Written so a later reader can tell a decision from an
accident.

## What shipped

| PR | Change |
|---|---|
| #176 | `engine.Render(…, scope, target)` with `ScopeInit`, `ScopeKind` and `ScopeProject` replaces the three places that assembled templates and generators (init, create api, update) and the render test's private copy. `scaffold/` is gone. Output unchanged. |
| #177 | `create api` refuses when PROJECT's generator version and the running binary are different clean releases: a newer binary is told to run `update` first. |
| #178 | `update`'s ownership-gated write moves to `core.Apply`; path containment to `core.CheckContained`, one copy. Output unchanged. |
| #179 | `init` renders in memory and writes through `core.Apply`. `ExecutableBitStep` is gone: Apply writes scripts 0755 itself. |
| #180 | `core.Apply` does not rewrite a tool-owned file whose bytes already match; `update` prints an `Unchanged N` line, and `update --verbose` lists every file with its reason. |
| #181 | `create api` renders the target kind in memory and writes through `core.Apply`. `--force` is deprecated and has no effect. |

All three commands that write a project now share one rule: a file carrying the
generated header is refreshed, one without it is the author's and is kept, a missing
one is created.

## Decisions, and what they cost

### `create api` renders only the target kind (Q1)

Refreshing the whole project from `create api` was rejected. Deleting orphans cannot
run there: `create api` has no clean-tree precondition and its fold commit is
`git add .`, so a wrong deletion would be committed. Without deletion a whole-project
refresh is only half of `update`. #177's version check is what makes the narrow scope
sufficient: with the same generator, adding a kind cannot make any other file stale.
Accepted limit: a development build (`dev`, git-describe, `-dirty`) is never compared,
the same blind spot as the downgrade check.

### The target kind is named, not found by position

`ScopeKind` takes the resource explicitly. Kubebuilder's `AddResource` does not move an
existing kind to the end, so "the last resource" re-renders the wrong kind when
`create api` is re-run for the first of two.

### Seeding needs no path predicate (Q5)

The proposal wanted `create api` to seed user-owned files only under the new kind's
paths. `ScopeKind` renders nothing else, so a plain `seedUserOwned = true` gives the
same result. Accepted: re-running `create api` for an existing kind recreates a
user-owned file its author deleted — as it always has.

### Deletion stays in `update`, outside `Apply` (Q9)

Orphan removal is only safe against a whole-project render and a clean tree. Measured
on a two-kind project: judged against one kind's render, 11 of 16 headered files would
have been deleted.

### `init` follows the same rule as `update`

The first plan seeded only (skip anything that exists), which cannot answer "is this
existing file current?". With the update rule, every tool-owned file is current after
`init`. In a non-empty directory without PROJECT that is a visible change: a headered
template output is now refreshed (it used to be skipped), and a headerless generator
output is kept and listed (it used to be overwritten, against the contract). `go.mod`
is kept as before.

### `--force` is a no-op for one minor (Q3)

`create api` now refreshes tool-owned files without it, so the flag has nothing left to
do. It is kept, hidden and marked deprecated (pflag prints the notice on stderr),
because v0.1.0 shipped with it and scripts pass it. One case changes meaning: a
tool-owned file whose author removed the header used to be overwritten by `--force`,
because the old code looked at the template's header, not the file's. It is now kept.

### A headerless registration file is skipped with a warning (Q4)

If an author removed the header from `apis/register.go` or
`internal/controller/register.go` (upjet: `config/zz_resources.go`), `create api` no
longer overwrites it; it prints a warning that the new kind was not registered there.
The build fails later otherwise, far from the cause.

### Identical files are not rewritten (Q6)

`Refreshed` in `update`'s summary now counts files whose content changed; the rest are
`Unchanged`. Counts by default, file lists under `--verbose` — a freshly scaffolded
native project already has 17 unchanged files.

### `Apply` lives in `core`

The proposal placed it in `scaffold`; #176 removed that package. `core` already held
`DecideWrite`, `IsToolOwned` and `FileMode`, and `v2` imports `core`, not the reverse.

## Deliberately not done

- **Removing `--force`.** Deprecated now; removal waits one minor.
- **Changing modes on unchanged or refreshed files.** An existing script keeps the mode
  it has: `WriteFile` applies permissions only when it creates a file, so only created
  files get `core.FileMode`.
- **Making `e2e-upgrade.sh` skip without Docker.** Its build step needs a daemon; when
  none runs the failure reads "upgraded provider does not build", which is not a code
  bug. The other two e2e scripts skip.

## What the review gate caught

Plans went back to review until a round returned no blocker and no should-fix: 4
rounds for Phase 1/1b and 3 for Phase 2. Each PR then went through break tests by the
controller and code review by a second model. The findings worth not repeating:

| Where | What |
|---|---|
| Plan | `ScopeKind` keyed on "the last resource" would re-render the wrong kind (the Kubebuilder ordering above). |
| Plan | The render test compared only path sets: dropping every generator from `ScopeKind` stayed green, and dropping them from `ScopeProject` stayed green on native — in `update` that would have deleted both `register.go` files as orphans. |
| Plan | The scaffold-diff script reported "identical, exit 0" when both scaffolds failed. |
| Plan | Removing `ExecutableBitStep` was expected to leave stdout identical; the pipeline prints numbered steps, so it could not. |
| Plan | `create api`'s new refusal would have gained an unrelated "Suggestions" block if its wording ever contained `version`: `createAPIHints` matches substrings of the cause. #177 pins it with a test. |
| #179 | Deleting the `Kept` heading left the new test green, and no unit test noticed `init` no longer seeding user-owned files. |
| #179 | The release note claimed a failed write leaves no half-written tree; only rendering is all-in-memory, writes are per file. |
| #180 | The e2e assertion `grep -q 'Unchanged'` could never fail: the line is printed even at zero. |
| #181 | A `create api` error hint still told users to pass `--force`. |
