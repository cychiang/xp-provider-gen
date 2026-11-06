# Testing

This page covers the **automated** suites: fast Go unit tests, a full end-to-end scaffold
test, an upgrade e2e, and an upjet-flavor e2e — four layers, all run by `make`
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
| `user-logic` | The overlay replaces the user-owned `client.go`, `external.go` and `options.go`, then `go mod tidy`, `make lint` and a commit. |
| `update-to-head` | Release cases only (`--from=<released binary>`). `update` by the binary under test leaves user-owned bytes alone and moves the PROJECT stamp. |
| `update-noop` | A same-generator `update` reports `removed 0`, `Refreshed 0` and a non-zero `Unchanged N`, leaves the seam files byte-identical and the tree clean, refuses a dirty tree, and rejects `--terraform-provider-version` naming the flavor. |
| `create-api-refresh` | Re-running `create api` regenerates a tool-owned file, keeps a user edit, commits nothing the second time, and rejects `--force`. |
| `adopt` | Plants a sentinel PROJECT `version:` and strips a header, then asserts `update --adopt` restores both. |
| `stock-e2e` | The scaffold exactly as generated, in a clone of its first commit, passes its own `make e2e`, with the head's `create-test` lifecycle test added first, so the one run covers uptest, the pause tests and the test `create-test` wrote; the stage then fails unless `junit.xml` names that test. Uptest skips its update step on the stock scaffold: it updates only a resource annotated `meta.upbound.io/example-id`, and allows one such root per run, while `make e2e` runs every kind's example in one run. |
| `build-xpkg`, `install-in-kind`, `provider-config` | The overlay provider's package builds, installs into a new kind cluster and is Healthy, (deployed with `--poll=10s` through a harness-only DeploymentRuntimeConfig, so drift shows in seconds; the stage fails unless the provider's deployment carries that argument), and a ProviderConfig holds the cluster's own kubeconfig as credentials. |
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
`VERDICT COMPLETE`. Other flags: `--logs=<dir>` keeps per-stage logs there, `--from=<binary>`
scaffolds with a released binary and adds `update-to-head`.

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

## Upgrade e2e (`make e2e-upgrade`)

`scripts/e2e-upgrade.sh` covers a gap the e2e cannot: the `update-noop` stage of the native e2e runs `update` with
the **same** generator, so tool-owned files come out byte-identical and it can only
prove that user-owned files survive — never that tool-owned files actually receive a new
generator's changes.

The upgrade e2e scaffolds a provider, writes **real** logic into every user-owned seam
(an HTTP client reading a user-added `ProviderConfigSpec` field, a `--region` flag
with validation, custom `ReconcilerOptions` and observe logic) plus unit tests that
pin that behavior, commits it, then
mutates the tool-owned templates to stand in for a new generator version, rebuilds,
and runs `update`. It asserts:

- no user-owned file appears in the update diff,
- all three tool-owned files (`internal/provider/connector.go`,
  `internal/controller/instance/wiring.go`, `hack/xp-provider-gen.mk`) received the
  simulated change,
- every piece of user logic is still present,
- the upgraded provider still generates, lints and builds (`make generate`, `make lint`,
  `make build`),
- the behavioral tests pass before **and after** the upgrade — same tests, same
  results, so the upgrade changed plumbing, not semantics,
- the user's `--region` flag still appears in the rebuilt binary's `--help`.

Three more steps run after that upgrade:

- **Orphan removal:** a tool-owned template (`cluster/local/integration_tests.sh`) is removed
  from the simulated new generator version; `update` deletes the now-orphaned tracked file on
  disk and its log reports `removed 1` and names the removed path.
- **Downgrade refusal:** with the project stamped by a simulated newer generator version,
  running an older one is refused on both `update` and `update --adopt` — the error says the
  running generator **is older than** the one that last updated the project, names both
  versions, and changes nothing.
- **create api refusal:** on the same v0.2.0-stamped project, `create api` is refused by an older
  (v0.1.0) and by a newer (v0.3.0) generator — the older one is told to run the recorded
  generator, the newer one to run `update` first — and neither changes anything.

It restores the templates it mutated. **Run it before shipping a generator bump.**

`/tmp/xpg-e2e-upgrade`, its temp project, is left in place after each run, whether it succeeded or
failed, for inspection; the next run removes and recreates it before scaffolding.

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
