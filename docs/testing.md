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
2. `init` a provider project; verify the base structure (Makefile, go.mod, apis/, cmd/provider/,
   internal/controller/).
3. Run `make submodules`, `make generate`, `make reviewable` on the generated project.
4. `create api` twice (same group/version, different kinds); verify the generated types and
   controllers.
5. Re-run the build targets; verify generated CRDs and examples.
6. Verify the provider builds.

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
