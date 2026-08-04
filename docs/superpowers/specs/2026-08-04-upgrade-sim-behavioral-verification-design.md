# Upgrade-Sim Behavioral Verification — Design

- **Date:** 2026-08-04
- **Status:** Approved (design); ready for implementation planning
- **Scope:** Extend `scripts/upgrade-sim.sh` so the upgrade proof covers *behavior*,
  not just compilation. No CLI changes, no new make targets, no new dependencies.
- **Builds on:** [2026-07-31 Modular Provider Layout](2026-07-31-modular-provider-layout-design.md),
  which established the seam contract and the upgrade simulation.

## 1. Problem

`make upgrade-sim` proves two of the three properties an upgrade must preserve:

1. **Build works** — the upgraded provider passes `make reviewable` and `make build`. ✅
2. **Business logic remains** — user-owned files are untouched and their content survives. ✅
3. **Behavior is unchanged** — nothing executes the user logic before and after the
   upgrade. ❌ A regression that compiles (a seam called with different semantics, a
   flag that no longer reaches `provider.Flags`) would pass today's sim.

## 2. Decision

Verify behavior with **user-owned unit tests plus a flag reachability check**, both run
before and after the upgrade. The tests encode expected behavior at the seams; passing
identically on both sides proves the upgrade changed plumbing, not semantics.

Rejected alternatives:

- **Full runtime (envtest/kind):** strongest proof, but slow and adds heavy
  dependencies for a simulation script. Disproportionate to the risk being tested.
- **Artifact comparison only:** compares CRDs/help output but never executes code
  paths; a semantic seam regression would still slip through.

## 3. Design

### 3.1 Behavioral tests (written in step 2, alongside the user logic)

The sim writes test files the way a real provider author would — user-owned, next to
the logic they test, exercising exactly the seams the sim already fills in:

**`internal/provider/client_test.go`** (package `provider`, so it can read the
package-level `region` var):

| Case | Asserts |
|------|---------|
| `NewClient` with empty `Endpoint` | returns an error |
| `NewClient` with `Endpoint` + credentials | client carries the endpoint, the token, and the current `--region` value |
| `Flags` on a fresh `kingpin.New` app, then `Parse(["--region=eu-west-1"])` | the package var holds `eu-west-1` |
| `Configure` with a valid region | `PollInterval` is doubled, no error |
| `Configure` with an empty region | returns an error |

**`internal/controller/instance/external_test.go`:**

| Case | Asserts |
|------|---------|
| `ReconcilerOptions(nil, o)` | returns an error |
| `NewExternal(&provider.Client{…}).Observe(ctx, &Instance{…})` | user observe path runs; returns a `managed.ExternalObservation`, no error |
| `Observe` with a non-`Instance` managed resource (`crossplane-runtime`'s `resource/fake.Managed` — a nil interface would panic in `meta.WasDeleted` before the type assertion) | returns the `errNotInstance` error |

Tests are table-driven and small; they exist to pin seam semantics, not to be a real
provider's test suite.

### 3.2 Baseline (step 2, before the simulated v2)

After committing the user logic:

- `go test ./...` must pass — reported as an explicit `PASS`/`FAIL` verdict line
  (never `&& echo` alone).
- Build the provider binary and assert `--region` appears in its `--help` output:
  proof the user's `Flags` seam is reachable from the tool-owned `main.go` at runtime.

A baseline failure aborts the sim — it would mean the test harness itself is broken,
and post-upgrade results would be meaningless.

### 3.3 Post-upgrade (new step, after today's build check)

With the *same, unmodified* test files:

- `go test ./...` must pass again.
- Rebuild the binary; `--region` must still appear in `--help` — the v2 `main.go`
  still calls `provider.Flags`.

### 3.4 Verdict and docs

The final verdict block gains three lines: tests pass before, tests pass after, user
flag reachable after upgrade. Any failure sets the existing `FAIL` flag so the script
exits non-zero. `docs/testing.md`'s upgrade-sim section gets one sentence describing
the behavioral verification.

## 4. Non-goals

- No envtest, kind, or any runtime cluster.
- No changes to `scripts/e2e-test.sh`, the Makefile targets, or the CLI.
- No test-output diffing between runs — "same tests pass" is the behavioral assertion;
  comparing verbose output would couple the sim to Go test formatting.

## 5. Risks

- **Template drift:** the tests assert against user logic the sim itself writes, so
  they can only break if the sim's own step 2 changes — self-contained by design.
- **Help-text coupling:** the flag check greps only for `--region`, not the full help
  text, so legitimate v2 changes to tool-owned help output cannot false-positive.
