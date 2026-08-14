# Uptest + Chainsaw E2E — Design

- **Date:** 2026-08-15
- **Status:** Approved (design discussed and accepted in session)
- **Scope:** Generated providers adopt upstream provider-template's uptest+chainsaw
  e2e suite; a new `create-test` command scaffolds chainsaw behavior tests
  interactively; the upstream `ClusterProviderConfigUsage` fix is ported into the
  templates. Base: the `refactor/template-management` branch (PR #120) — the
  generator-body convention lives there.
- **Reference:** `crossplane/provider-template@ef72f04` (`test/` tree, `uptest.mk`
  wiring, hook structure) — re-read 2026-08-15 including the latest changes.

## 1. Goals (user's words, restated)

1. Generated providers use **uptest** (resource lifecycle) and **chainsaw**
   (controller behavior) for e2e.
2. A bootstrapped project can run resource e2e out of the box — scaffold-level,
   not full implementations.
3. The tool helps developers add chainsaw test cases, prompting interactively
   when information is missing.

## 2. Generated provider changes

### test/ tree (all user-owned — tests belong to the developer; seeded once)

```
test/
├── setup.sh                          # uptest setup: wait provider Healthy, apply examples/provider/config.yaml
├── README.md                         # structure, how to run, create-test pointer
├── e2e/
│   └── KIND-lifecycle.yaml           # per kind (API-category template): two MRs
│                                     #   (ProviderConfig + ClusterProviderConfig refs),
│                                     #   uptest annotations: timeout 120, conditions Ready,Synced.
│                                     #   No hooks — scaffold-level; README shows how to add them.
└── behavior/
    └── KIND-pause/chainsaw-test.yaml # per kind: seed chainsaw test — apply MR → Ready →
                                      #   annotate crossplane.io/paused → assert ReconcilePaused →
                                      #   unpause → Ready. The pattern to copy.
```

### Makefile.tmpl

- `CROSSPLANE_VERSION ?= 2.3.4` (matches upstream), include `controlplane.mk`
  and `uptest.mk` (already in the build submodule; `local.xpkg.mk` is already
  included).
- `UPTEST_LOCAL_DEPLOY_TARGET = local.xpkg.deploy.provider.$(PROJECT_NAME)`.
- `UPTEST_INPUT_MANIFESTS = $(wildcard test/e2e/*-lifecycle.yaml)` — the
  user-owned Makefile never needs editing when kinds are added.
- **Safety deviation from upstream:** `KIND_CLUSTER_NAME ?= $(PROJECT_NAME)-e2e`.
  Upstream inherits `local-dev` and `make e2e` starts with `controlplane.down`,
  which would delete an unrelated personal cluster of that name.
- **Behavior wiring deviation:** upstream runs chainsaw from inside a per-kind
  uptest post-assert hook (their README calls uptest's vocabulary the limiter).
  We instead add a first-class target and append it to `e2e`'s prerequisites
  (serial make runs prerequisites in list order, same mechanism uptest.mk itself
  relies on):

  ```make
  test-behavior: $(CHAINSAW) $(KUBECTL)
  	@$(CHAINSAW) test test/behavior --parallel 1 --quiet || $(FAIL)
  e2e: test-behavior
  ```

  Per-kind lifecycle files stay hook-free; `make test-behavior` also runs alone
  against an existing cluster.
- `make e2e` is now the uptest flow; `e2e.run: test-integration` unhooked;
  `test-integration` stays as the fast from-source loop; **`e2e-package` and
  `cluster/local/package_tests.sh` are removed** (the uptest flow deploys from
  the local xpkg — packaging is covered).

### Upstream fix ported (b07ef7f)

Our templates carry the bug upstream just fixed: a `ClusterProviderConfigUsage`
type that nothing ever creates, so the cluster-config reconciler counts zero
users and in-use `ClusterProviderConfig`s can be deleted. Port:

- `apis/v1alpha1/types.go.tmpl` — remove the `ClusterProviderConfigUsage` type.
- `apis/v1alpha1/register.go.tmpl` — remove its scheme registration.
- `internal/controller/config/config.go.tmpl` — point the cluster reconciler's
  usage kinds at namespaced `ProviderConfigUsage` (with upstream's explanatory
  comment).

`config.go` is tool-owned, so existing providers receive the reconciler fix via
`update`; `types.go` is user-owned, so stale (harmless, unused) type definitions
remain there — documented in the provider guide's upgrade notes.

## 3. `xp-provider-gen create-test`

Root-level cobra command (same pattern as `update`):

- `--name <test-name>` → `test/behavior/<name>/chainsaw-test.yaml`;
  `--kind <Kind>` → the kind the skeleton applies/asserts.
- Missing flag + TTY → interactive prompt; kind is a numbered pick-list from
  `PROJECT`'s resources (auto-selected when exactly one; error when none:
  "run create api first"). Missing flag + no TTY → error (CI-safe).
- Renders a chainsaw skeleton (body in `pkg/templates/generators/chainsaw_test.yaml.tmpl`):
  apply one MR of the kind → assert `Ready` → a clearly marked TODO block for
  real assertions. Never overwrites an existing file.
- Output is user-owned by definition (no header). It is created on demand, so it
  is not part of `CoreGenerators` and not listed in the generated ownership doc.

## 4. Tool-side verification changes

- Golden ownership map: `+ test/setup.sh, test/README.md, test/e2e/KIND-lifecycle.yaml,
  test/behavior/KIND-pause/chainsaw-test.yaml` (all user-owned);
  `− cluster/local/package_tests.sh`.
- `scripts/assert-layout.sh`: base layout + per-kind lists gain the test files;
  `package_tests.sh` removed.
- `scripts/e2e-test.sh`: Step E still runs the scaffold's `make e2e` — now the
  full uptest+chainsaw flow (slower; that is goal 2's proof). A new sub-step runs
  `create-test --name smoke-test --kind MyType` non-interactively and asserts the
  chainsaw file lands.
- `upgrade-sim`: unchanged mechanics; its provider now contains the test tree
  (user-owned files must survive `update` — covered by the existing untouched-file
  assertions pattern).

## 5. Documentation (all of it)

- `docs/testing.md` — harness description: Step E now uptest+chainsaw; the
  create-test sub-step.
- `docs/tutorial.md` — "Run it on kind" gains the e2e story: `make e2e`
  (uptest+chainsaw), `make test-integration` (fast loop), `make test-behavior`;
  `create-test` walkthrough; `e2e-package` references removed.
- `docs/provider-guide.md` — testing section: the three targets, the test/ tree,
  the ownership of test files, upgrade note about the stale CPCU type in old
  providers.
- `README.md` — commands table gains `create-test`; e2e description updated.
- `pkg/templates/files/AGENTS.md.tmpl` — "where your code goes" gains test/.
- `test/README.md` (template) — the in-provider explanation (structure, targets,
  adding tests, hooks pointer to upstream docs).
- `docs/templates.md` — unchanged (flow already covers new templates); verify only.

## 6. Non-goals

- No porting of upstream's five behavior tests beyond the per-kind pause seed.
- No uptest hook scaffolding (README documents the upstream pattern instead).
- No changes to `make dev`, `make test-integration` internals, or the update
  contract.

## 7. Verification

1. Repo gate (`fmt vet lint test`) green.
2. `make upgrade-sim` green (test tree survives update).
3. Full `make e2e-test` green — Step E now proves: bootstrap → `make e2e` runs
   uptest lifecycle for both kinds + chainsaw behavior suite on a kind cluster,
   from the locally built package.
4. In the scaffold: `create-test` non-interactive scaffolds a valid chainsaw file;
   interactive path exercised manually once.
5. PR checks green (stacked on PR #120's branch).
