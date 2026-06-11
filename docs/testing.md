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

Reuse shared literals via constants (keeps tests DRY and satisfies `goconst`).

## End-to-end test

`scripts/e2e-test.sh` (run via `make e2e-test`) exercises the real generator workflow against
a throwaway project in `/tmp/provider-template`:

1. Build the binary and prepare a clean temp directory.
2. `init` a provider project; verify the base structure; **assert the working tree is clean**
   (generate-then-commit leaves nothing uncommitted).
3. `create api` twice (same group/version, different kinds); verify the generated types,
   controllers, CRDs, and examples; **assert the tree is clean again**.
4. **Ownership contract:** assert tool-owned files (`register.go`, `setup.go`, `main.go`,
   `config.go`) carry the `DO NOT EDIT` header and user files (`controller.go`, `*_types.go`)
   do not.
5. **`update`:** hand-edit a `controller.go` and commit, run `update`, then assert (a) the edit
   survives, (b) `setup.go` is refreshed (header intact), (c) `update` refuses a dirty tree.
6. **`update --adopt`:** strip the header from `setup.go` (simulate a pre-contract provider),
   run `update --adopt`, then assert the header is restored and PROJECT gains the provenance stamp.
7. Verify the provider builds.

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
