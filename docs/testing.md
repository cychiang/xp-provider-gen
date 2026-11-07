# Testing

This page covers the **automated** suites: fast Go unit tests, a full end-to-end scaffold
test, an upgrade from each published release, and an upjet-flavor e2e — four layers, all run by `make`
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
the first failure. The runner is `plan.go`, the stages are `stages.go`, and `plan_test.go`
tests the unknown-mode and empty-run rejections, the full-mode rule, the fast-mode verdict, the exact set of stages fast mode runs and the first-failure stop. `stages_test.go` tests the chainsaw-report check, which needs a passed `lifecycle` testcase (`go test ./test/e2e/`).

| Stage | Proves |
|---|---|
| `init`, `create-api` | The binary scaffolds a provider with two kinds (`Widget`, `Gadget`) and `scripts/assert-layout.sh` finds every expected file. `init` stamps PROJECT with the generator `version:`, and leaves a clean tree, so `create api` cannot commit an `init` leftover. |
| `generate` | `make submodules generate reviewable` pass on the pristine scaffold. CRDs and examples exist, the ownership contract holds (tool-owned files carry the `DO NOT EDIT` header, user-owned files do not), the tree is clean, and `init` plus `create api` folded into one `Initial commit`. |
| `user-logic` | The overlay replaces the user-owned `client.go`, `external.go` and `options.go`, adds a `--region` flag read by `ReconcilerOptions`, a user field on `ProviderConfigSpec` and two user unit tests, then `go mod tidy`, `make generate lint` and a commit. The ProviderConfig CRD must carry the new field, the user tests must pass, and the provider's `--help` must list `--region`. |
| `update-to-head` | Release cases only (`--from=<tag>`). `update` by the binary under test leaves user-owned bytes alone, moves the go directive and the PROJECT stamp, keeps the ownership contract, regenerates the CRD with the user field, keeps the user tests passing and `--region` in `--help`, and leaves every tool-owned file byte-identical to a fresh scaffold from the same binary. |
| `refuse-old-generator` | Release cases only. The release binary's `update` and `update --adopt` refuse the updated tree, and `create api` refuses it in both directions where the release has that check. |
| `orphan-removal` | `update` removes and names a tracked file that carries the generated header but that no template renders, and leaves a gitignored one alone. |
| `update-noop` | A same-generator `update` reports `removed 0`, `Refreshed 0` and a non-zero `Unchanged N`, leaves the seam files byte-identical and the tree clean, refuses a dirty tree, and rejects `--terraform-provider-version` naming the flavor. |
| `create-api-refresh` | Re-running `create api` regenerates a tool-owned file, keeps a user edit, commits nothing the second time, and rejects `--force`. |
| `adopt` | Plants a sentinel PROJECT `version:` and strips a header, then asserts `update --adopt` restores both. |
| `stock-e2e` | The scaffold exactly as generated, in a clone of its first commit, passes its own `make e2e`, with the head's `create-test` lifecycle test added first, so the one run covers uptest, the pause tests and the test `create-test` wrote; the stage then fails unless `junit.xml` names that test. Uptest skips its update step on the stock scaffold: it updates only a resource annotated `meta.upbound.io/example-id`, and allows one such root per run, while `make e2e` runs every kind's example in one run. |
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
`VERDICT COMPLETE`. Other flags: `--logs=<dir>` keeps per-stage logs there, `--from=<tag>`
scaffolds with that published release and adds `update-to-head`.

**Isolation.** Each run makes its own work directory, log directory and kind cluster name, and
sets a private `KUBECONFIG` on every subprocess, so it never reads or writes `~/.kube/config` and
parallel runs cannot collide. It deletes its cluster on exit, including on failure. A failed run
keeps its work directory and prints the path.

The shell helpers `assert_create_api_refreshes`, `assert_adopt_restores` and
`assert_update_refuses_dirty` live in `scripts/lib.sh`, still shared with `e2e-upjet.sh`.

```bash
make e2e-native                                  # build + full run
make e2e-native E2E_ARGS=--mode=fast             # no Docker needed
go run ./test/e2e --list --flavor=native         # the declared stages
```

Run the e2e test whenever you change templates, the template engine, or the automation
pipeline: unit tests alone do not catch broken generated output.

## Upgrade from a released binary (`--from=<tag>`)

`make e2e-native E2E_ARGS=--from=v0.3.0` runs the native stages with the published release's own
binary doing `init` and `create api`, and the binary under test doing everything after. That is
the one run in which tool-owned files receive a real generator's changes, which the
same-generator `update-noop` cannot show. The `update-to-head` stage is added; the rest are the
stages above.

`Releases` in `test/e2e/plan.go` is the list of tags the harness knows, each with its
`GOTOOLCHAIN`. A run first asks `gh release list` for the newest release and fails if
`Releases` lacks it, so a new release needs a line there before any run passes. The archive and
checksum file are downloaded with `gh` into new directories under the run's work directory;
the checksum is verified before the binary is unpacked, and the binary runs only after.

A released binary's `init` and `create api`, and only those, run under the release's declared
`GOTOOLCHAIN` (`go1.26.8`, that release's `go_version`), printed at the start of each run, so the
scaffold is the one the release produced whatever Go the host has. Every other command runs with
`GOTOOLCHAIN` unset, even when the caller exports one, as `actions/setup-go` does with `local`.
A flavor and release pair that cannot run is declared `NotApplicable` in `Releases` with a
reason; the run prints `VERDICT NOT-APPLICABLE` and the reason, and exits 0.

`update-to-head` plants a stale go directive, then runs the binary under test's `update`
and asserts the user-owned seams (the overlay files, the `ProviderConfigSpec` field in
`types.go` and the user unit tests) are byte-identical, the go directive is the `go_version` in
`pkg/versions/dependencies.yaml`, the PROJECT stamp is the binary's own `version`, and the
ownership contract holds. It reruns the user tests and the `--region` check that `user-logic`
ran before the update. It then scaffolds the same project (same flags, `Widget` and `Gadget`)
with the binary under test in a second directory and requires every file carrying the
`Code generated by xp-provider-gen` header, in its first 1024 bytes, to exist in both trees with
the same bytes, and the two sets to be equal. The set must be non-empty and hold `connector.go`,
a `wiring.go` and the make fragment. No file is excluded: PROJECT and the other user-owned files
carry no header and are not in the set. The stages after it build, deploy and reconcile the
updated provider, so one that does not come up `Ready` fails there. `orphan-removal` tracks a
file that carries the generated header but that no template renders, and asserts `update`
removes and names it, leaves a gitignored file with the same header alone, and changes nothing
else.

The `E2E (upgrade from <tag>)` jobs in `e2e-native-full.yml` run one release each, with a
20-minute limit (a cell took 14 to 18 minutes on a runner; the fresh head scaffold `update-to-head` makes for its comparison runs `init`, with generate and reviewable, and is about two of them). **Run it before shipping a generator bump.**

`refuse-old-generator` runs the release binary against the updated tree. A released binary
orders only two clean release versions, and the head build's version is a git-describe string or a
bare commit hash, so against the real stamp every release accepts `update`, `update --adopt` and
`create api`. The stage therefore stamps PROJECT by hand with `v9.9.9`, commits, and asserts
`update` and `update --adopt` refuse with "is older than the one that last updated this project"
and change nothing. Where `Releases` declares `CreateAPIGuard` (v0.2.0 and v0.3.0), `create api` is
also refused at `v9.9.9` ("is older than") and at `v0.0.1` ("Run ... update first"). v0.1.0 predates
the `create api` check and accepts, so the stage does not ask it. The stamp is reset afterwards.
`TestCheckNotDowngrade` and `TestCheckSameGenerator` in
`pkg/plugins/crossplane/v2/downgrade_test.go` cover the head generator's side.

## Upjet-flavor e2e (`make e2e-upjet`)

`scripts/e2e-upjet.sh` proves the other flavor end to end: scaffold a provider
wrapping `hashicorp/kubernetes`, configure `kubernetes_secret` with `create api`,
then run the **real** upjet pipeline — `make generate` downloads Terraform, reads
the provider schema, scrapes the provider's docs and generates API types,
controllers, scheme registration and CRDs — and finally build the result. It
then runs `update --terraform-provider-version=2.38.0` once on the generated
provider, asserting a stale tool-owned file is refreshed, the Makefile no longer
pins the old `2.37.1`, and a deleted user-owned file is not re-seeded (the log
also reports `removed 0`, since the same-generator run orphans nothing).

Step 8, **"the generated provider starts, not just builds,"** goes past a bare
build: it runs the provider binary's scheme registration against an unreachable
API server (asserting no panic), then builds and runs
`hack/envtest-provider-check` — its own Go module — against a real, ephemeral
`envtest` API server to prove the generated controllers actually start and
register, not merely compile.

It then covers three git-automation paths native's e2e already had (the `create api` refresh)
or that upjet previously lacked (`--adopt`, the dirty-tree refusal), none of
which need Docker:

- **`create api` refresh:** mark a tool-owned file (`config/zz_resources.go`) and a user-owned one
  (`config/secret/config.go`), commit, then re-run `create api` (with
  `--terraform-resource`, required on every upjet `create api` call) and assert that the
  tool-owned marker is regenerated while the user-owned one survives.
- **`create api` with nothing to change:** run it again and assert it still exits 0, reports
  the no-change skip, and adds no commit (git.go's `stageAndCheck` skips the commit when
  nothing is staged).
- **`--adopt`:** strip `config/provider.go`'s header to simulate a pre-contract provider,
  run `update --adopt`, and assert it reports adopting exactly one file and that
  `git diff --name-only` contains `config/provider.go` and, optionally, `PROJECT` (adopt
  stamps a generator version there, but step 6's own `update` may already have stamped
  the same value).
- **Dirty-tree refusal:** leave an uncommitted change and assert `update` refuses it,
  citing the working tree.

It also configures a `kubernetes_config_map` resource, writes its
`docs/upjet-provider.md` worked example (with `uptest.upbound.io/*` annotations), and runs
`xp-provider-gen create-test` — the scaffolding half of the native e2e's `create-test` coverage
(it does not re-run to check the refusal to overwrite). With a Docker daemon available (skipped with a warning otherwise, or
with `E2E_SKIP_DOCKER` set; `E2E_REQUIRE_DOCKER=1` fails the run instead), it then:

- runs the **generated provider's own `make e2e`**: the uptest lifecycle (create →
  Ready/Synced → import → delete) plus the `test-behavior` chainsaw hook running the suite
  `create-test` just scaffolded, asserting `junit.xml` shows it ran;
- on that same still-live cluster, exercises the one lifecycle path uptest's own run never
  covers — UPDATE, since the worked example carries no `uptest.upbound.io/update-parameter`
  annotation: apply the example directly and assert the real ConfigMap's `data.hello`;
- patch `spec.forProvider.data` and assert the real object follows;
- delete it and confirm it's gone.

That is the only test that proves the config files this tool scaffolds satisfy
upjet's contract; a unit test cannot, because the contract is upjet's generator.
It needs network access and takes several minutes, so it is a separate target
rather than part of `make e2e-native`.

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
