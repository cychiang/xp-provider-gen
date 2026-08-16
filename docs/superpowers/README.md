# Design records

Historical design documents and implementation plans, kept for the reasoning behind
decisions — **not** as a description of how the tool works today.

**Do not read these as current behavior.** They were written before the work landed and
were not revised afterwards. For what is true now:

| You want | Read |
|---|---|
| How the generator is built | [architecture.md](../architecture.md) |
| What provider authors write, and upgrading | [provider-guide.md](../provider-guide.md) |
| Adding or changing templates | [templates.md](../templates.md) |
| Testing | [testing.md](../testing.md) |
| The ownership contract for a specific provider | `docs/ownership.md` **inside that provider** |

## What shipped

Every spec and plan below is implemented and merged; the layout each describes has
since moved on in places.

| Design | Shipped as |
|---|---|
| [Provider generation & upgradability](specs/2026-06-11-provider-generation-upgradability-design.md) | the ownership header, `update`, and the dependency manifest (PRs 1–6) |
| [Modular provider layout](specs/2026-07-31-modular-provider-layout-design.md) | the six-seam layout: `external.go`/`wiring.go` per kind, `internal/provider` |
| [Upgrade-sim behavioral verification](specs/2026-08-04-upgrade-sim-behavioral-verification-design.md) | `scripts/upgrade-sim.sh` / `make upgrade-sim` |
| [Developer tutorial](specs/2026-08-05-developer-tutorial-design.md) | [tutorial.md](../tutorial.md) |
| [uptest + chainsaw e2e](specs/2026-08-15-uptest-chainsaw-e2e-design.md) | the generated `test/` tree and `create-test` |

## Known divergences

Where these documents no longer match the tool:

- **ProviderConfig is namespaced-only.** Several documents describe a cluster-scoped
  `ClusterProviderConfig` alongside the namespaced one. That was removed: managed
  resources are namespaced, so a cluster-scoped config only widened who could reach
  whose credentials. Credentials now use a `LocalSecretKeySelector`, and the
  `Filesystem` / `Environment` credential sources are no longer offered. See
  [provider-guide.md](../provider-guide.md#5-upgrading).
- **Phase B (detect/apply available upgrades)** is referenced as future work in the
  upgradability and modular-layout specs. It has not been designed or built.
