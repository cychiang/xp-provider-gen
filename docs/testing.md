# Testing

This page tells you how to run the test suites and how to read their output. What each
end-to-end stage proves is in the stage's own code and error messages, and `--list` prints
the stage names. Which workflow runs which suite, and when, is in
[.github/WORKFLOWS.md](../.github/WORKFLOWS.md).

## Unit tests

```bash
make test         # go test -v -race ./...
make coverage     # writes coverage/coverage.html
```

Tests sit next to the code in `*_test.go` and are table-driven. The golden ownership test,
`pkg/plugins/crossplane/v2/templates/engine/ownership_test.go`, fails when you add a template
until you add it to `wantOwnership`. That is deliberate (see
[architecture.md](architecture.md#6-ownership-contract-the-upgrade-foundation)).

Run the end-to-end suite below whenever you change templates, the engine, or the automation
pipeline. Unit tests alone do not catch broken generated output.

## End-to-end harness

`test/e2e` is a Go program that runs a declared list of stages, in order, against a provider
scaffolded by `bin/xp-provider-gen`. A stage returns an error or it passed. There is no skip
result, and the run stops at the first failure.

```bash
make e2e-native                                  # build, then a full native run
make e2e-upjet                                   # build, then a full upjet run
make e2e-native E2E_ARGS=--mode=fast             # only the stages that need no Docker
make e2e-native E2E_ARGS=--from=latest           # upgrade from the newest published release
make e2e-upjet E2E_ARGS=--from=latest            # the same for an upjet provider
make e2e-native E2E_ARGS=--report=report.json    # also write the run as JSON
```

The upjet run needs network access, because it downloads Terraform and a provider schema, and
it takes several minutes. Run `make e2e-native E2E_ARGS=--from=latest` and
`make e2e-upjet E2E_ARGS=--from=latest` before you ship a generator change that affects
`update`. They are the only runs where tool-owned files receive a real earlier generator's
output.

### Read the list

`go run ./test/e2e --list --flavor=native` prints one line per stage, in run order. This excerpt shows three of them:

```
init                 needs=[] release-only=false
stock-clone          needs=[docker] release-only=false
update-to-head       needs=[] release-only=true
```

`needs=[docker]` marks a stage that fast mode leaves out. `release-only=true` marks a stage
that runs only with `--from`, and a head run skips it. Add `--from=latest` to list only the
stages that apply to an upgrade run, which also drops the head-only stages. `--from=latest`
still calls `gh` to resolve the newest release, so it needs `gh` and network access even
for `--list`. `--list` runs nothing else and writes no report. Use `--flavor=upjet` for the other flavor.

### Read the verdict

A run ends in one `VERDICT` line.

| Verdict | Meaning | Exit code |
|---|---|---|
| `COMPLETE` | Every stage ran and passed. | 0 |
| `PARTIAL` | Fast mode left out the stages that need Docker, and the line names each one. | 0 |
| `FAILED` | A stage failed. The line names it, and each later stage prints `NOT-RUN`. | 1 |

Only a full run can end in `COMPLETE`. Full mode checks that Docker answers before its first
stage, and exits 1 if it does not. An unknown `--mode`, or a plan with no stage to run, also
exits 1 before anything starts.

### Flags

| Flag | Effect |
|---|---|
| `--flavor=native\|upjet` | The flavor to test. `make` passes it for you. |
| `--mode=full\|fast` | `full` is the default. |
| `--from=head\|latest\|<tag>` | The binary that runs `init` and `create api`. `head` is the default. `latest` is the newest published release, which `gh release list` resolves without pre-releases or drafts. A tag such as `v0.3.0` names another release. Older releases are not tested in CI. |
| `--report=<file>` | Write the run as JSON, also when a stage fails. |
| `--logs=<dir>` | Keep per-stage logs there. The default is a new temporary directory. |
| `--overlay=<dir>` | Lay these files over the scaffold. The default is `test/e2e/testdata/<flavor>`. |
| `--bin=<path>` | The binary under test. The default is `bin/xp-provider-gen`. |

### Isolation

Each run makes its own work directory, log directory and kind cluster name, and sets a
private `KUBECONFIG` on every subprocess. It never reads or writes `~/.kube/config`, and
parallel runs do not collide. The run deletes its cluster on exit, also on failure. A failed
run keeps its work directory and prints the path.

The overlay in `test/e2e/testdata/<flavor>/` gives the scaffold an external system to talk to.
It lives in the harness and not in the shipped template, so the stub that authors receive stays
a stub.

### Upgrade runs

With `--from`, the release's own binary does `init` and `create api`, and the binary under test
does everything after. The `update-to-head` stage plants a stale go directive, runs `update`,
and takes four measurements before it fails on any of them.

The hand-kept part is `toolchains` in `test/e2e/plan.go`. A tag listed there (today `v0.3.0`)
runs its `init` and `create api` under that `GOTOOLCHAIN`, printed at the start of the run, so
the scaffold is the one the release produced whatever Go the host has. Every other command runs
with `GOTOOLCHAIN` unset. A release whose make fragment carries the toolchain line should need no entry. No such
release has been cut yet to confirm it.

### Read the report

`--report=<file>` writes this JSON, and the CI upgrade jobs upload it as `report.json`:

```json
{
  "schema": 1,
  "flavor": "upjet",
  "from": "v0.3.0",
  "head": "v0.3.0-19-g723b828",
  "verdict": "COMPLETE",
  "stages": [{"name": "init", "status": "pass", "seconds": 1.7}],
  "upgrade": {
    "tool_owned_compared": 21, "tool_owned_differing": 0,
    "user_owned_changed": 0, "generated_changed": 0
  }
}
```

`from` is empty for a head run. A stage `status` is `pass`, `fail` or `not-run`, and `seconds`
is 0 for a stage that did not run. `schema` changes when a field is renamed or removed.

`upgrade` appears only when `update-to-head` ran its measurements, so its absence on an
upgrade run means the stage failed before it. A count is `null`, not `0`, when its check
errored before it could count, so a measured zero is never confused with a check that did not
finish. The run also prints one line, for example
`UPGRADE tool_owned 21 compared 0 differing, user_owned 0 changed, generated 0 changed` for upjet
(a native run compares 15), with `n/a` for a `null` count. A run that fails before its first stage, say because `gh` could not
fetch the release, writes a `FAILED` report with no stages.

| Count | Meaning |
|---|---|
| `tool_owned_compared` | The tool-owned files (those with the generated header) in a fresh scaffold from the binary under test, made in a second directory with `generate` and `reviewable` stubbed out. |
| `tool_owned_differing` | How many of those the updated tree lacks or holds with other bytes, plus tool-owned files it holds that a fresh scaffold does not. Zero means `update` produced what a fresh scaffold has. |
| `user_owned_changed` | Tracked files without the header that `update` removed, changed or added. `PROJECT`, `go.mod` and `go.sum` are expected to move and are left out, as is generated output. Zero means `update` left the author's files alone. |
| `generated_changed` | Files that the provider's own `make generate` changes, adds or removes after `update`. Zero means `update` left no stale generated output. |

### Limits

Fast mode, for either flavor, does not prove that the generated controllers start. The upjet
`provider-starts` stage builds the provider, prints `--help` and reaches the startup precheck without a panic,
but it starts no controller. Full mode deploys the provider with `install-in-kind` and shows a
resource reconciling in `mr-ready-synced`, and both need Docker.

## Release-note rules

`hack/cliff-golden.sh` is a golden test for `cliff.toml`. It builds a fixture repository with a
known history and asserts the next version, the grouped release notes, the ASCII normalization,
and that a range with nothing releasable leaves the version unchanged. It needs `git-cliff` on
`PATH` and runs as a step of every `release.yml` run. Run it locally with
`hack/cliff-golden.sh`.

## Test a change by hand

To see what a provider author sees, build the binary and follow the [tutorial](tutorial.md)
for a native provider or the [upjet guide](upjet-provider.md) for an upjet one. Scaffold into a
directory outside this repository, never inside it.
