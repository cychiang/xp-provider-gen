# Testing

This page covers the **automated** suites: fast Go unit tests, a full end-to-end scaffold
test, an upgrade from the newest published release, and an upjet-flavor e2e — four layers, all run by `make`
targets and wired into CI — plus a golden test for the release pipeline's git-cliff rules.
For a **manual** walkthrough of the same surfaces — reproducing a bug report,
sanity-checking a change by hand, or seeing what a provider author actually experiences —
see [docs/manual-testing.md](manual-testing.md).

## Unit tests

Standard `go test`, table-driven, run with the race detector:

```bash
make test         # go test -v -race ./...
make coverage     # writes coverage/coverage.html
```

Tests live next to the code (`*_test.go`). Pattern:

```go
tests := []struct {
    name    string
    input   string
    wantErr bool
}{
    {name: "valid", input: "example.com", wantErr: false},
    {name: "empty", input: "", wantErr: true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        err := validator.ValidateDomain(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("ValidateDomain() error = %v, wantErr %v", err, tt.wantErr)
        }
    })
}
```

### The golden ownership test

`pkg/plugins/crossplane/v2/templates/engine/ownership_test.go` walks the embedded
template FS and asserts each template's ownership bucket against a golden map, plus
that the map covers exactly the files on disk.

**Adding a template fails this test until you add it to `wantOwnership`.** That is
deliberate: a tool-owned template that forgets the header would otherwise be silently
user-owned, and `update` would never refresh it — a bug nobody would notice for months.

Walk the FS directly rather than keying by template name: base names are not unique
(`Makefile.tmpl` exists at both `project/` and `cluster/images/IMAGENAME/`), so a
name-keyed map silently drops a file.

Reuse shared literals via constants (keeps tests DRY and satisfies `goconst`).

## Native-flavor e2e (`make e2e-native`)

`test/e2e` is a Go program that runs a declared list of stages, in order, against a native
provider scaffolded by `bin/xp-provider-gen`. `go run ./test/e2e --list --flavor=native` prints
the list. A stage returns an error or it passed; there is no skip result, and the run stops at
the first failure. One stage, `stock-e2e`, is a background stage: it starts after `user-logic` and runs beside the update stages that follow it. It works on a clone that the foreground stage `stock-clone` makes before it starts, so nothing it reads changes under it. It shares one thing with `install-in-kind`, the run's `KIND_CLUSTER_NAME`, and that is safe only because the run joins it at `install-in-kind` and its `e2e-clean` deletes the cluster first; the run's cleanup deletes whatever is left. `build-xpkg` runs beside it too, so its outcome is reported at `install-in-kind` as well. The runner is `plan.go`, the one stage list both flavors run is `Flavor.Stages` in `flavor.go`, the stages are `stages.go` and `upjet.go`, and `plan_test.go`
tests the unknown-mode and empty-run rejections, the full-mode rule, the fast-mode verdict, the exact set of stages fast mode runs and the first-failure stop. `stages_test.go` tests the chainsaw-report check, which needs a passed `lifecycle` testcase (`go test ./test/e2e/`).

| Stage | Proves |
|---|---|
| `init`, `create-api` | The binary scaffolds a provider with two kinds (`Widget`, `Gadget`) and every file in the layout lists of `layout.go` exists. `init` stamps PROJECT with the generator `version:`, and leaves a clean tree, so `create api` cannot commit an `init` leftover. |
| `generate` | `make submodules generate reviewable` pass on the pristine scaffold; release cases run `make submodules generate` only, since reviewable of an old scaffold proves nothing about this build. CRDs and examples exist, the ownership contract holds (tool-owned files carry the `DO NOT EDIT` header, user-owned files do not), the tree is clean, and `init` plus `create api` folded into one `Initial commit`. |
| `user-logic` | The overlay replaces the user-owned `client.go`, `external.go` and `options.go`, adds a `--region` flag read by `ReconcilerOptions`, a user field on `ProviderConfigSpec` and two user unit tests, then `go mod tidy`, `make generate lint` (release cases: `make generate`, because a release's make fragment pins no toolchain and its linter cannot read a newer host Go's export data; `update` writes the pin and its own reviewable lints the overlay) and a commit. The ProviderConfig CRD must carry the new field, the user tests must pass, and the provider's `--help` must list `--region`. |
| `update-to-head` | Release cases only (`--from=latest`). `update` by the binary under test leaves user-owned bytes alone, moves the go directive and the PROJECT stamp, keeps the ownership contract, keeps the user tests passing and `--region` in `--help`, leaves every tool-owned file byte-identical to a fresh scaffold from the same binary, and a following `make generate` changes nothing. |
| `refuse-old-generator` | Release cases only. The release binary's `update` and `update --adopt` refuse the updated tree, and `create api` refuses it in both directions. |
| `orphan-removal` | `update` removes and names a tracked file that carries the generated header but that no template renders, and leaves a gitignored one alone. |
| `update-noop` | A same-generator `update` reports `removed 0`, `Refreshed 0` and a non-zero `Unchanged N`, leaves the seam files byte-identical and the tree clean, refuses a dirty tree, and rejects `--terraform-provider-version` naming the flavor. |
| `create-api-refresh` | Re-running `create api` regenerates a tool-owned file, keeps a user edit, commits nothing the second time, and rejects `--force`. |
| `adopt` | Plants a sentinel PROJECT `version:` and strips a header, then asserts `update --adopt` restores both. |
| `stock-clone` | Head cases only. Clones the scaffold and checks out the flavor's stock commit, in the foreground, so the background `stock-e2e` never reads the provider directory the stages beside it change. |
| `stock-e2e` | Background stage, head cases only: in a release case it would test the old release's scaffold, which the head cell already covers for this build, and the upgraded tree is deployed and reconciled by `install-in-kind` and the `mr-` stages. The scaffold exactly as generated, in a clone of its first commit, passes its own `make e2e`, with the head's `create-test` lifecycle test added first, so the one run covers uptest, the pause tests and the test `create-test` wrote; the stage then fails unless `junit.xml` names that test. Uptest skips its update step on the stock scaffold: it updates only a resource annotated `meta.upbound.io/example-id`, and allows one such root per run, while `make e2e` runs every kind's example in one run. |
| `build-xpkg`, `install-in-kind`, `provider-config` | The overlay provider's package builds, installs into a new kind cluster and is Healthy, (deployed with `--poll=10s` through a harness-only DeploymentRuntimeConfig, so drift shows in seconds; the stage fails unless the provider's deployment carries that argument), and a ProviderConfig holds the cluster's own kubeconfig as credentials and keeps the user-added `endpoint` field instead of pruning it. |
| `mr-ready-synced` | The `Widget` example is `Ready` and `Synced`, and a ConfigMap, the external object, holds its value. |
| `mr-update`, `drift-corrected` | The external value follows a spec patch, and a hand edit of the external object is restored. |
| `mr-delete`, `external-gone` | The managed resource is deleted, then the external object is gone. |
| `create-test` | `create-test` writes a chainsaw test and refuses to overwrite it. Needs no Docker, so fast mode runs it. |

The overlay in `test/e2e/testdata/native/` gives the scaffold an honest external system: a
ConfigMap client built from `ClientConfig.Credentials`. It lives in the harness, not in the
shipped template, so the stub authors receive stays a stub. `--overlay=<dir>` points the run at
other files, which is how the break tests below were made.

**Modes.** `make e2e-native` runs full mode: it checks that Docker answers before the first stage and
exits 1 if not. `make e2e-native E2E_ARGS=--mode=fast` runs only the stages that need no Docker and
ends in `VERDICT PARTIAL`, naming each stage it did not run. An unknown `--mode`, or a plan with no stage to run, exits 1 before anything starts. Only a full run ends in
`VERDICT COMPLETE`. Other flags: `--logs=<dir>` keeps per-stage logs there, `--from=latest` (or a release tag)
scaffolds with that published release, adds `update-to-head` and `refuse-old-generator`, and drops `stock-e2e`; `--report=<file>` writes the JSON report; `--list` with `--from` prints the stages that apply.

**Isolation.** Each run makes its own work directory, log directory and kind cluster name, and
sets a private `KUBECONFIG` on every subprocess, so it never reads or writes `~/.kube/config` and
parallel runs cannot collide. It deletes its cluster on exit, including on failure. A failed run
keeps its work directory and prints the path.

```bash
make e2e-native                                  # build + full run
make e2e-native E2E_ARGS=--mode=fast             # no Docker needed
go run ./test/e2e --list --flavor=native         # the declared stages
```

Run the e2e test whenever you change templates, the template engine, or the automation
pipeline: unit tests alone do not catch broken generated output.

## Upgrade from a released binary (`--from=latest`)

`make e2e-native E2E_ARGS=--from=latest` runs the native stages with the newest published
release's own binary doing `init` and `create api`, and the binary under test doing everything
after. `make e2e-upjet E2E_ARGS=--from=latest` does the same for an upjet provider; its stages
are in the upjet section below. That is the one run in which tool-owned files receive a real
generator's changes, which the same-generator `update-noop` cannot show. The `update-to-head` and
`refuse-old-generator` stages are added and `stock-e2e` is dropped; the rest are the stages above.

`latest` is resolved by `gh release list`, which leaves out pre-releases and drafts, so no list
of releases is kept by hand when a release is cut. `--from=<tag>` names another published release and is
the way to look at an older one; older releases are not tested in CI, and nothing here keeps
them working. The only per-release fact the harness holds, and the one thing here that is kept by hand, is
`toolchains` in `test/e2e/plan.go`: a tag listed there (today `v0.3.0`, `go1.26.8`) has its `init` and `create api`, and only those,
run under that `GOTOOLCHAIN`, printed at the start of each run, so the scaffold is the one the
release produced whatever Go the host has. A tag not listed runs with no override. A release whose scaffold carries the toolchain pin needs
no entry: the head templates do (`export GOTOOLCHAIN ?= go<version>+auto` in
`pkg/templates/files/hack/xp-provider-gen.mk.tmpl` and `pkg/templates/upjet/hack/xp-provider-gen.mk.tmpl`),
so by that reading the next release needs nothing; no such release has been cut to show it. Every other
command runs with `GOTOOLCHAIN` unset, even when the caller exports one, as `actions/setup-go`
does with `local`. The archive and checksum file are downloaded with `gh` into new directories
under the run's work directory; the checksum is verified before the binary is unpacked, and the
binary runs only after.

`update-to-head` plants a stale go directive, then runs the binary under test's `update`
and takes four measurements before it fails on any of them:

| Count | What it is |
|---|---|
| `tool_owned_compared`, `tool_owned_differing` | The binary under test scaffolds the same project (same flags) in a second directory, with `generate` and `reviewable` stubbed out. Every file carrying the `Code generated by xp-provider-gen` header, in its first 1024 bytes, must exist in both trees with the same bytes, and the two sets must be equal. The set must be non-empty and hold `connector.go`, a `wiring.go` and the make fragment. `compared` is the fresh scaffold's tool-owned files; `differing` is those the updated tree lacks or holds with other bytes, plus tool-owned files it holds that a fresh scaffold does not. |
| `user_owned_changed` | Every tracked file with no generator header must be byte-identical across the `update`, except `PROJECT` (the version stamp), `go.mod` (the go directive moves to head's pin) and `go.sum` (follows `go.mod`); generated output is left out by name (below). A new user-owned file is covered without being listed, and one `update` adds counts. |
| `generated_changed` | After `update`, the provider's own `make generate` runs and every file it changes, adds or removes counts. The byte comparison above leaves generated output out (any `zz_*` file, `package/crds/`, `examples-generated/`, and upjet's `config/schema.json` and `config/provider-metadata.yaml`), so a stale generated file that still compiles and lints would otherwise pass. Whatever `update` left stale, generate rewrites, and the stage fails naming the file. |

The stage also asserts the go directive is the `go_version` in `pkg/versions/dependencies.yaml`,
the PROJECT stamp is the binary's own `version`, the ownership contract holds, the user-owned
seams (the overlay files, the `ProviderConfigSpec` field in `types.go` and the user unit tests)
still work, and the user tests and the `--region` check that `user-logic` ran pass again. The
stages after it build, deploy and reconcile the updated provider, so one that does not come up
`Ready` fails there. The generate check costs one `make generate`. It runs after the stamp, go directive
and ownership reads, because generate's tidy could otherwise rewrite go.mod and hide a directive
`update` left behind.

`orphan-removal` tracks a file that carries the generated header but that no template renders,
and asserts `update` removes and names it, leaves a gitignored file with the same header alone,
and changes nothing else.

**The report.** `--report=<file>` writes the run as JSON, also when a stage fails, and the CI
jobs upload it with the logs as `report.json`. The shape (`schema` is 1):

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

`from` is empty for a head run. A stage's `status` is `pass`, `fail` or `not-run`, and
`seconds` is 0 for one that did not run. `upgrade` is present only when `update-to-head` ran its
measurements, so a missing one on an upgrade run means the stage failed before it. A count is
`null`, not `0`, when its check errored before it could count (say `make generate` failed), so a
measured zero is never confused with a check that did not finish. The run also prints one line,
for example `UPGRADE tool_owned 21 compared 0 differing, user_owned 0 changed, generated 0 changed`,
with `n/a` for a count that is `null`. `--list` writes no report. A run that
failed before any stage, say because `gh` could not fetch the release, writes a `FAILED` report
with no stages.

The `E2E (upgrade from the newest release)` job in `e2e-native-full.yml` and the `E2E (upjet,
upgrade from the newest release)` job in `e2e-upjet.yml` run on the weekly schedule and on
`workflow_dispatch`, never on a pull request, with a 20-minute limit that only cancels a hung
run. A release cell runs `generate` without `reviewable` and skips `stock-e2e`, and the fresh
head scaffold that `update-to-head` compares against runs with `generate` and `reviewable`
stubbed out. **Run one before shipping a generator bump:** `gh workflow run e2e-native-full.yml`
and `gh workflow run e2e-upjet.yml`.

`refuse-old-generator` runs the release binary against the updated tree. A released binary
orders only two clean release versions, and the head build's version is a git-describe string or a
bare commit hash, so against the real stamp the release accepts `update`, `update --adopt` and
`create api`. The stage therefore stamps PROJECT by hand with `v9.9.9`, commits, and asserts
`update` and `update --adopt` refuse with "is older than the one that last updated this project"
and change nothing, and that `create api` is refused at `v9.9.9` ("is older than") and at
`v0.0.1` ("Run ... update first"). The stamp is reset afterwards.
`TestCheckNotDowngrade` and `TestCheckSameGenerator` in
`pkg/plugins/crossplane/v2/downgrade_test.go` cover the head generator's side.

## Upjet-flavor e2e (`make e2e-upjet`)

`go run ./test/e2e --list --flavor=upjet` prints the stages. The upjet flavor runs the same
harness, with the same `Stage`, `Case`, `Plan` and verdict rules as native, and runs the
same stage list as native (`Flavor.Stages`), with four stages whose work differs (`init`, `create-api`, `generate`, `user-logic`) and two of its own after `create-test`, `tf-bump` and `provider-starts` (`Flavor.Extra`).
The target is a provider that wraps `hashicorp/kubernetes` at 2.37.1, so the external system is
the kind cluster itself and no cloud credentials are involved.

Where an upjet stage differs from the native one of the same name:

| Stage | Proves |
|---|---|
| `init`, `create-api` | `init --upjet` scaffolds the files `upjetBaseLayout` lists, pins the `build` submodule at `build_ref` (head), wires the Terraform provider into the Makefile and leaves a clean tree. `create api` for `Secret` (`kubernetes_secret`) writes `config/secret/config.go` and wires it into `config/zz_resources.go`. |
| `generate` | The real upjet pipeline (`make submodules generate`): Terraform is downloaded, the provider schema read, its docs scraped, and the types, controllers, scheme registration, setup and CRDs written from the real schema. Then `create api` for `ConfigMap` (`kubernetes_config_map`) and a second `make generate`, the incremental generate an author runs for a new kind, which must write the second kind's types, controller and CRD. `go build ./...` passes, the ownership contract holds, and the output is committed as `Generate provider`. The scaffold's `go.generate` tidies go.mod after generating, as the build submodule's own recipe does, so the first generate leaves it as the next one would (before, `dario.cat/mergo` stayed indirect until the second generate, and a same-generator `update` then changed go.mod). |
| `user-logic` | The overlay adds a user-owned `internal/clients/user.go` and test, wires the scaffold's `clients.go` to call it, and adds the `examples/core/configmap.yaml` that uptest and `create-test` read. The user test passes and the provider builds. |
| `update-to-head` | As native. The user-owned files not rewritten by `update` stay byte-identical, and the tool-owned files match a fresh upjet scaffold. |
| `update-noop` | As native, with the user-owned probe `config/secret/config.go`. The `--terraform-provider-version` refusal is native only. |
| `create-api-refresh`, `adopt` | `create api` regenerates `config/zz_resources.go` and keeps an edit in `config/secret/config.go`; `update --adopt` restores `config/provider.go`'s header, reports `Adopted 1 tool-owned file(s)` and touches only that file and PROJECT. |
| `tf-bump` | Makes `config/provider.go` and `hack/xp-provider-gen.mk` stale and deletes `examples/providerconfig/providerconfig.yaml`, then `update --terraform-provider-version=2.38.0`. The Makefile is the old one with 2.37.1 replaced by 2.38.0 and nothing else, with the new provider binary pinned; the docs cache `.work/hashicorp/kubernetes@2.37.1` still exists next to `@2.38.0`, the unversioned directory is gone, the new checkout is at `v2.38.0` and `.work/terraform@2.38.0` exists; both stale files are refreshed, the deleted example is not re-seeded and the report says so; the other user-owned files are byte-identical. |
| `provider-starts` | `make go.build` produces a provider that prints `--help`, and run against an API server that is not there it reaches `SafeStart precheck failed` without a panic. |
| `stock-e2e` | Background stage, head cases only: the generated provider's own `make e2e` (uptest create, Ready and Synced, import, delete, then the chainsaw suite `create-test` wrote), on a clone of the `user logic` commit. |
| `build-xpkg`, `install-in-kind`, `provider-config` | The package builds, installs into a new kind cluster and is Healthy with `--poll=10s`; `examples/providerconfig/providerconfig.yaml` is applied. |
| `mr-ready-synced`, `mr-update`, `drift-corrected` | The `ConfigMap` example is Ready and Synced and a real ConfigMap `default/example` holds `hello: world`; a spec patch changes it to `updated-value`; a hand edit is restored. |
| `mr-delete`, `external-gone` | The managed resource is deleted, then a `kubectl get` of the real ConfigMap finds nothing. |

The file ownership checks of an upjet `update` leave out, by name, what `make generate` writes
and `update` re-runs: `zz_*` files (upjet's types, controllers, registration and setup),
`package/crds/`, `examples-generated/`, and `config/schema.json` and `config/provider-metadata.yaml`
(upjet's scrape of the schema and docs). PROJECT, go.mod and go.sum are the only other files an
`update` changes (checked for v0.3.0); `tf-bump` also allows the Makefile, as
above.

Fast mode runs the stages that need no Docker, through `provider-starts`, and ends in `VERDICT
PARTIAL`. `provider-starts` proves the provider builds, prints `--help` and reaches its startup
precheck without a panic; it does not start a controller. That proof, which the old shell script
gave without Docker through `hack/envtest-provider-check/`, now needs Docker, so fast mode no
longer has it. The generated controllers are shown to start against a real API server by the cluster
stages: `install-in-kind` waits for the provider to be Healthy and then for its log to show a
started controller for every generated kind and for ProviderConfig, and `mr-ready-synced`
shows one reconciling.

The upgrade cell (`--from=latest`) runs the same stages as native's: the release's binary does
`init` and `create api`, `generate` runs with the release's make fragment, and head's `update` is
checked by `update-to-head`, including the `generated_changed` count above.

It needs network access and takes several minutes, so it is a separate target rather than part of
`make e2e-native`.

## Release-note rules (`hack/cliff-golden.sh`)

`hack/cliff-golden.sh` is a golden test for `cliff.toml`'s git-cliff rules: it builds a
fixture repo with a known commit history and asserts the exact next version, the exact
grouped release notes (Breaking changes / Features / Bug fixes / Dependency updates), that
the six typographic characters `cliff.toml` normalizes come out as ASCII, and that a range
with nothing releasable leaves the version unchanged. It requires `git-cliff` on `PATH` and
runs as a step of every `release.yml` run — dry-run and real alike — so a rule change that
breaks version bumping, note grouping, or ASCII normalization turns the release run red
before GoReleaser ever executes. Run it locally the same way:

```bash
hack/cliff-golden.sh
```

## In CI

Which workflow runs which of the layers above, and on what trigger, is
[.github/WORKFLOWS.md](../.github/WORKFLOWS.md)'s job, not this page's — see its `Workflows`
table.

