# Uptest + Chainsaw E2E Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generated providers ship uptest+chainsaw e2e (lifecycle per kind + seed
behavior test), `make e2e` becomes the uptest flow, `e2e-package` is retired, a new
`create-test` command scaffolds chainsaw tests interactively, and the upstream
ClusterProviderConfigUsage fix is ported.

**Architecture:** Everything template-shaped rides the drop-a-file pipeline (per-kind
files use the KIND placeholder → API category automatically). The chainsaw skeleton for
`create-test` is a generator body under `pkg/templates/generators/`. The command is a
root-level cobra command registered via `cli.WithExtraCommands` like `update`.

**Tech Stack:** Go (cobra, kubebuilder config store), uptest.mk/controlplane.mk from the
build submodule, chainsaw, bash harnesses.

**Spec:** `docs/superpowers/specs/2026-08-15-uptest-chainsaw-e2e-design.md`
**Reference:** `/Users/cychiang/Projects/github.com/crossplane/provider-template` @ ef72f04

## Global Constraints

- Branch `feat/uptest-chainsaw-e2e`, stacked on `refactor/template-management`.
- `KIND_CLUSTER_NAME ?= $(PROJECT_NAME)-e2e` — never inherit `local-dev`.
- `CROSSPLANE_VERSION ?= 2.3.4`.
- All `test/` files user-owned (no generated header). Golden map updated same commit.
- Template variables available: `{{ .Repo }}`, `{{ .ProviderName }}`, `{{ .Domain }}`,
  `{{ .Resource.Kind }}`, `{{ .Resource.Group }}`, `{{ .Resource.Version }}`,
  `{{ .Boilerplate }}`; MR apiVersion form: `{{ .Resource.Group }}.{{ .Domain }}/{{ .Resource.Version }}`
  (copied from `examples/GROUP/KIND.yaml.tmpl`).
- Verify each template-affecting task with `make test` (golden) before moving on.

---

### Task 1: Port upstream b07ef7f (ClusterProviderConfigUsage removal)

**Files:** Modify `pkg/templates/files/apis/v1alpha1/types.go.tmpl`,
`pkg/templates/files/apis/v1alpha1/register.go.tmpl`,
`pkg/templates/files/internal/controller/config/config.go.tmpl`.

- [ ] **Step 1:** In `types.go.tmpl`: delete the `ClusterProviderConfigUsage` and
  `ClusterProviderConfigUsageList` type blocks (incl. their kubebuilder markers and the
  interface assertion vars naming them); replace the `ProviderConfigUsage` doc comment
  with upstream's ("…indicates that a resource is using a ProviderConfig or a
  ClusterProviderConfig. There is deliberately no cluster scoped usage type…").
- [ ] **Step 2:** In `register.go.tmpl`: delete the ClusterProviderConfigUsage type-metadata
  var block and its `SchemeBuilder.Register` entry.
- [ ] **Step 3:** In `config.go.tmpl`: in `setupClusterProviderConfig`, switch
  `Usage`/`UsageList` to `ProviderConfigUsageGroupVersionKind` /
  `ProviderConfigUsageListGroupVersionKind`, with upstream's four-line comment.
- [ ] **Step 4:** `make test` (golden unchanged — same files) and scaffold-compile check:
  `make build && rm -rf /tmp/pcu-check && mkdir /tmp/pcu-check && cd /tmp/pcu-check &&
  <repo>/bin/xp-provider-gen init --domain=example.com --repo=github.com/example/pcu &&
  <repo>/bin/xp-provider-gen create api --group=s --version=v1alpha1 --kind=Thing &&
  make reviewable` → green; `grep -r ClusterProviderConfigUsage apis/` → empty.
- [ ] **Step 5:** Commit `fix(templates): track ClusterProviderConfig usage via namespaced ProviderConfigUsage`.

### Task 2: test/ tree templates

**Files:** Create `pkg/templates/files/test/setup.sh.tmpl`,
`pkg/templates/files/test/README.md.tmpl`,
`pkg/templates/files/test/e2e/KIND-lifecycle.yaml.tmpl`,
`pkg/templates/files/test/behavior/KIND-pause/chainsaw-test.yaml.tmpl`.
Modify `pkg/plugins/crossplane/v2/templates/engine/ownership_test.go` (golden +4, all `false`).

- [ ] **Step 1:** `setup.sh.tmpl` (init category — no placeholder in path):

```bash
#!/usr/bin/env bash
set -euo pipefail

# uptest setup: runs once before the lifecycle tests.
echo "Waiting for provider to become healthy..."
${KUBECTL} wait provider.pkg {{ .ProviderName }} \
  --for=condition=Healthy \
  --timeout=180s

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "Creating ProviderConfig and credentials from examples..."
${KUBECTL} apply -f "${PROJECT_ROOT}/examples/provider/config.yaml"

echo "Setup complete."
```

- [ ] **Step 2:** `KIND-lifecycle.yaml.tmpl` — two MRs (ProviderConfig + ClusterProviderConfig
  refs), names `e2e-<kind>-lifecycle` / `e2e-<kind>-lifecycle-cluster`, annotations
  `uptest.upbound.io/timeout: "120"` and `uptest.upbound.io/conditions: "Ready,Synced"`,
  spec copied from the example template's forProvider. No hooks.
- [ ] **Step 3:** `KIND-pause/chainsaw-test.yaml.tmpl` — adapt the reference repo's
  `test/behavior/pause/chainsaw-test.yaml` **assertion-for-assertion** (its chainsaw
  array/condition semantics are debugged; do not invent): replace `MyType`→`{{ .Resource.Kind }}`,
  apiVersion→the standard form, resource names→`behavior-<kind>-pause`, field stays
  `configurableField` (scaffold kinds have it). Keep upstream's step structure
  (Ready → pause → assert ReconcilePaused → unpause → Ready) and timeouts.
- [ ] **Step 4:** `README.md.tmpl` — structure tree, the three targets
  (`e2e` / `test-integration` / `test-behavior`), "add a test with
  `xp-provider-gen create-test`", pointer to uptest annotation docs + upstream
  provider-template for hook patterns.
- [ ] **Step 5:** `make test` → golden fails listing exactly the 4 new paths → add them
  (`false`) → `make test` green. Commit
  `feat(templates): scaffold uptest lifecycle and chainsaw behavior tests`.

### Task 3: Makefile.tmpl uptest wiring; retire e2e-package

**Files:** Modify `pkg/templates/files/project/Makefile.tmpl`;
delete `pkg/templates/files/cluster/local/package_tests.sh.tmpl`; golden −1;
modify `scripts/assert-layout.sh` (remove package_tests.sh; add test/ files, per-kind
lifecycle + pause paths).

- [ ] **Step 1:** In Makefile.tmpl after the xpkg block insert:

```make
# ====================================================================================
# Setup Uptest

CROSSPLANE_VERSION ?= 2.3.4
-include build/makelib/controlplane.mk

# A dedicated cluster name so e2e's controlplane.down can never delete an
# unrelated kind cluster (the build system default is "local-dev").
KIND_CLUSTER_NAME ?= $(PROJECT_NAME)-e2e

UPTEST_LOCAL_DEPLOY_TARGET = local.xpkg.deploy.provider.$(PROJECT_NAME)
UPTEST_INPUT_MANIFESTS = $(wildcard test/e2e/*-lifecycle.yaml)
-include build/makelib/uptest.mk

# Controller behavior tests (chainsaw). Appended to e2e's prerequisites so a
# serial make runs them after uptest; also usable alone against a live cluster.
test-behavior: $(CHAINSAW) $(KUBECTL)
	@$(INFO) running chainsaw behavior tests
	@$(CHAINSAW) test test/behavior --parallel 1 --quiet || $(FAIL)
	@$(OK) chainsaw behavior tests passed
e2e: test-behavior
```

  (`local.xpkg.mk` is already included above.) Remove the `e2e.run: test-integration`
  line and the whole `e2e-package` target; drop `e2e-package` from `.PHONY` and help;
  add `test-behavior` to both; update the help text for `e2e`.
- [ ] **Step 2:** Delete `package_tests.sh.tmpl`; remove its golden entry; in
  `assert-layout.sh` swap `cluster/local/package_tests.sh` for `test/setup.sh` +
  `test/README.md`, and add per-kind `test/e2e/${kind_lower}-lifecycle.yaml` +
  `test/behavior/${kind_lower}-pause/chainsaw-test.yaml`.
- [ ] **Step 3:** `make test` green; scaffold in /tmp/pcu-check again and check
  `make -n e2e` shows controlplane+deploy+uptest+test-behavior order and
  `grep KIND_CLUSTER_NAME Makefile` shows the -e2e suffix. Commit
  `feat(templates)!: make e2e the uptest+chainsaw flow; retire e2e-package`.

### Task 4: create-test command

**Files:** Create `pkg/plugins/crossplane/v2/createtest.go`,
`pkg/plugins/crossplane/v2/createtest_test.go`,
`pkg/templates/generators/chainsaw_test.yaml.tmpl`;
modify `cmd/xp-provider-gen/main.go` (register), engine: export a small
`NewChainsawTestGenerator(name string, res resource.Resource, domain string)`
in a new `pkg/plugins/crossplane/v2/templates/engine/chainsaw_generator.go`.

- [ ] **Step 1:** Generator body `chainsaw_test.yaml.tmpl` (no ownership header — user file):

```yaml
# Chainsaw behavior test scaffolded by xp-provider-gen create-test.
# Docs: https://kyverno.github.io/chainsaw/
apiVersion: chainsaw.kyverno.io/v1alpha1
kind: Test
metadata:
  name: {{ .TestName }}
spec:
  timeouts:
    apply: 1m
    assert: 2m
    delete: 2m
  steps:
    - name: {{ .Resource.Kind }} becomes Ready
      try:
        - apply:
            resource:
              apiVersion: {{ .Resource.Group }}.{{ .Domain }}/{{ .Resource.Version }}
              kind: {{ .Resource.Kind }}
              metadata:
                name: behavior-{{ .TestName }}
                namespace: default
              spec:
                forProvider:
                  configurableField: "{{ .TestName }}"
                providerConfigRef:
                  name: example
                  kind: ProviderConfig
        - assert:
            resource:
              apiVersion: {{ .Resource.Group }}.{{ .Domain }}/{{ .Resource.Version }}
              kind: {{ .Resource.Kind }}
              metadata:
                name: behavior-{{ .TestName }}
                namespace: default
              status:
                (conditions[?type == 'Ready']):
                  - status: "True"
    # TODO: this scaffold only proves the resource reconciles. Replace or extend
    # the steps below with the behavior you actually want to pin — error paths,
    # drift, pause, config resolution. See test/behavior/ for a worked example.
```

- [ ] **Step 2:** `chainsaw_generator.go`: machinery.Template with TemplateMixin,
  `TestName string`, `Resource resource.Resource`, `Domain string`; Path =
  `test/behavior/<TestName>/chainsaw-test.yaml`; `IfExistsAction = machinery.Error`
  (never overwrite); body from `templates.GeneratorBody("chainsaw_test.yaml.tmpl")`.
- [ ] **Step 3:** `createtest.go` — `NewCreateTestCommand() *cobra.Command`, flags
  `--name`, `--kind`. Flow: load PROJECT via yaml store (same as update.go's `prepare`
  pattern, minus clean-tree check); resources := cfg.GetResources() filtered to
  Group != "". Resolution: kind flag → match (case-insensitive) or error listing kinds;
  no flag + 1 kind → use it; no flag + >1 + TTY → numbered stdin prompt; no TTY → error.
  Name: flag, else TTY prompt; validate `^[a-z0-9][a-z0-9-]*$`; no TTY → error.
  TTY check: `(os.Stdin state via os.Stdin.Stat() & os.ModeCharDevice) != 0` — no new deps.
  Render via machinery.NewScaffold(os FS) executing the generator; print the created path
  and a next-steps line (`make test-behavior`).
- [ ] **Step 4:** Unit tests: kind resolution table (flag match, case-insensitive, single
  default, ambiguous error, none error), name validation, and generator render → contains
  kind + test name, no DO NOT EDIT header, path correct.
- [ ] **Step 5:** Register in main.go: `cli.WithExtraCommands(crossplanev2.NewUpdateCommand(), crossplanev2.NewCreateTestCommand())`.
- [ ] **Step 6:** `make test` green; manual: in /tmp/pcu-check run
  `bin/xp-provider-gen create-test --name smoke --kind Thing` → file lands; rerun → error
  (exists); `create-test --name x` with 1 kind → kind auto-picked. Commit
  `feat: add create-test command scaffolding chainsaw behavior tests`.

### Task 5: Harness + e2e-test.sh

**Files:** Modify `scripts/e2e-test.sh`.

- [ ] **Step 1:** After the second `create api` (Step 5 region) add a create-test check:

```bash
    log_info "Scaffolding a chainsaw test with create-test..."
    if "$BINARY_PATH" create-test --name smoke-test --kind "$KIND1"; then
        verify_files_exist "create-test output" "test/behavior/smoke-test/chainsaw-test.yaml"
    else
        log_error "create-test failed"; exit 1
    fi
```

  (Before the single-commit assertion? No — create-test output is uncommitted; place it
  AFTER the clean-tree + single-commit assertions and remove the file + `git clean` it
  before the lifecycle copy, so the pristine-scaffold guarantees hold. Simplest: run it
  in the LIFECYCLE_DIR copy instead, right after `cd "$LIFECYCLE_DIR"`.)
  → Implement in LIFECYCLE_DIR; add a summary line `create-test scaffolds a chainsaw test: PASSED`.
- [ ] **Step 2:** Step E stays `make e2e` (now uptest+chainsaw). Update its header text to
  "Generated provider's own e2e (uptest + chainsaw)" and the docker-unavailable skip
  message. Run `bash -n scripts/e2e-test.sh`.
- [ ] **Step 3:** Commit `test(e2e): cover create-test; Step E now runs uptest+chainsaw`.

### Task 6: Documentation sweep

**Files:** Modify `docs/testing.md`, `docs/tutorial.md`, `docs/provider-guide.md`,
`README.md`, `pkg/templates/files/AGENTS.md.tmpl`. (`docs/templates.md` verify-only.)

- [ ] **Step 1:** testing.md — e2e section: Step E now uptest+chainsaw; create-test
  sub-step; remove `e2e-package` mentions (the packaging path is inside `make e2e` now).
- [ ] **Step 2:** tutorial.md — replace the `make e2e`/`e2e-package` bullet in "Where to go
  next" with: `make e2e` (uptest lifecycle + chainsaw behavior from the local package),
  `make test-integration` (fast from-source), `make test-behavior`, and a one-line
  `create-test` example.
- [ ] **Step 3:** provider-guide.md — new "Testing your provider" subsection: the test/
  tree (user-owned), three targets, create-test, and an upgrade note: providers
  generated before this change keep a harmless stale `ClusterProviderConfigUsage`
  type in user-owned `types.go` (delete it by hand or regenerate).
- [ ] **Step 4:** README.md — commands table row for `create-test`; update the e2e bullet.
- [ ] **Step 5:** AGENTS.md.tmpl — "Where your code goes" table gains
  `test/e2e/`, `test/behavior/` rows.
- [ ] **Step 6:** Commit `docs: uptest+chainsaw e2e and create-test across all guides`.

### Task 7: Full verification + PR

- [ ] **Step 1:** `for t in fmt vet lint test; do … PASS/FAIL; done` → 4 PASS.
- [ ] **Step 2:** `make upgrade-sim` → green (test tree survives update untouched).
- [ ] **Step 3:** `make e2e-test` → all steps green, Step E = real uptest+chainsaw run
  (both kinds' lifecycles + pause behavior tests on the throwaway kind cluster,
  provider deployed from local xpkg).
- [ ] **Step 4:** Kind-cluster hygiene check: `kind get clusters` → only `local-dev`
  (user's) remains.
- [ ] **Step 5:** Push `-u`; open PR with base `refactor/template-management`; report.
