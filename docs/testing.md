# Testing

Two layers: fast Go unit tests, and a full end-to-end scaffold test.

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

## End-to-end test

`scripts/e2e-test.sh` (run via `make e2e-test`) exercises the real generator workflow against
a throwaway project in `/tmp/provider-template`:

1. Build the binary and prepare a clean temp directory.
2. `init` a provider project; verify the base structure; **assert the working tree is clean**
   (generate-then-commit leaves nothing uncommitted).
3. `create api` twice (same group/version, different kinds); verify the generated types,
   controllers, CRDs, and examples; **assert the tree is clean again**.
4. **Ownership contract:** assert tool-owned files (`register.go`, `wiring.go`,
   `internal/provider/connector.go`, `main.go`, `config.go`, `docs/ownership.md`) carry the
   `DO NOT EDIT` header, and that user files (`external.go`, `internal/provider/client.go`,
   `internal/provider/options.go`, `*_types.go`, `apis/v1alpha1/types.go`, `AGENTS.md`) do not.
5. **`update` (the upgrade guarantee):** append a marker to **all three** user-owned seam
   files and commit, run `update`, then assert (a) every marker survives, (b) `wiring.go`,
   `connector.go` and `docs/ownership.md` are refreshed with headers intact, (c) seed-once
   `AGENTS.md` is untouched, (d) `update` refuses a dirty tree.
6. **`update --adopt`:** strip the header from `wiring.go` (simulate a pre-contract provider),
   run `update --adopt`, then assert the header is restored and PROJECT gains the provenance stamp.
7. **create-test:** scaffold a chainsaw behavior test non-interactively and assert the file
   lands — and that an existing test is never overwritten.
8. **The generated provider's own e2e:** run `make e2e` inside the scaffold — the full
   uptest + chainsaw flow: build the xpkg, stand up a dedicated kind control plane with
   Crossplane installed, deploy the provider from the local package, run every kind's
   uptest lifecycle (create → Ready/Synced → delete), then the chainsaw behavior suite
   (the seeded pause tests). Skipped with a warning when no Docker daemon is available.
9. Verify the provider builds.

## Upgrade-path simulation (`make upgrade-sim`)

`scripts/upgrade-sim.sh` covers a gap the e2e cannot: e2e Step U runs `update` with
the **same** generator, so tool-owned files come out byte-identical and it can only
prove that user files survive — never that tool-owned files actually receive a new
generator's changes.

The simulation scaffolds a provider, writes **real** logic into every user-owned seam
(an HTTP client reading a user-added `ProviderConfigSpec` field, a `--region` flag
with validation, custom `ReconcilerOptions` and observe logic) plus unit tests that
pin that behavior, commits it, then
mutates the tool-owned templates to stand in for a new generator version, rebuilds,
and runs `update`. It asserts:

- no user-owned file appears in the update diff,
- both tool-owned files received the simulated change,
- every piece of user logic is still present,
- the upgraded provider still passes `make reviewable` and `make build`,
- the behavioral tests pass before **and after** the upgrade — same tests, same
  results, so the upgrade changed plumbing, not semantics,
- the user's `--region` flag still appears in the rebuilt binary's `--help`.

It restores the templates it mutated. **Run it before shipping a framework bump.**

On **success** the temp project is left in place for inspection (the next run recreates it).
On **failure** the script removes the incomplete directory and exits non-zero.

```bash
make e2e-test            # build + run
./scripts/e2e-test.sh -h # usage
```

Run the e2e test whenever you change templates, the template engine, or the automation
pipeline — unit tests alone do not catch broken generated output.

## In CI

Both layers run on every push/PR (see [.github/WORKFLOWS.md](../.github/WORKFLOWS.md)):
`test.yml` runs unit tests with coverage and the e2e workflow; `lint.yml` and `ci.yml` add
linting, gosec, and Trivy scanning.
