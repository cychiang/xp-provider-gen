# Provider Generation & Upgradability — Implementation Roadmap

> **For agentic workers:** Each PR below is implemented as its own branch → `/simplify` → `/review` → green CI → squash-merge to `main`. Plans for PRs 2–7 are written just-in-time (in this directory) immediately before each is implemented, so each reflects the merged state of prior PRs. PR 1 is fully detailed in `2026-06-11-pr1-pipeline-reliability.md`.

**Source spec:** `docs/superpowers/specs/2026-06-11-provider-generation-upgradability-design.md`

**Goal:** Clean, reliable, idempotent provider generation, plus a safe `update` command that refreshes core components of existing providers without touching user logic.

## PR sequence

Ordered by dependency and risk (low-risk/high-value first):

| PR | Title | Delivers | Depends on |
|----|-------|----------|------------|
| **1** | Pipeline reliability | generate-then-commit ordering; all steps required; clean-tree assertion in e2e | — |
| **2** | Deterministic register.go | regenerate `apis/register.go` + `internal/controller/register.go` from PROJECT; delete `file_parser.go` + the two updaters | 1 |
| **3** | Ownership headers + overwrite gate | `DO NOT EDIT` header on tool-owned templates; seed/overwrite/skip gate; user files protected | 2 |
| **4** | Dependency manifest | `pkg/versions/dependencies.yaml`; render `go.mod.tmpl` from it; Renovate custom manager | — (parallel to 2/3) |
| **5** | `update` command + drift | new `update` subcommand; version/provenance stamp in PROJECT; drift detection; applies dependency manifest via `go get` | 3, 4 |
| **6** | `update --adopt` migration | one-shot adoption for existing providers (stamp + write headers) | 5 |
| **7** | Docs review & consolidation | reconcile `docs/**` + README with the new architecture; remove "needs regeneration" caveat | 2–6 |

Each PR produces working, tested software on its own and is independently mergeable.

## Quality gates (every PR)

1. TDD where unit-testable; extend `scripts/e2e-test.sh` for behavior that needs the full workflow.
2. `make reviewable` green locally.
3. `/simplify` then `/code-review` on the diff; address findings.
4. CI green (lint, gosec, trivy, tests, e2e, builds).
5. Squash-merge with a conventional-commit title.

## Open decisions carried from the spec (§7)

- `setup.go` ownership (default: tool-owned) — resolved in PR 3 when headers are applied.
- `crossplane.yaml` structural-vs-prose split — resolved in PR 3.
