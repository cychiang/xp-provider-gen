# Manual testing guide

The automated suites (`make test`, `make e2e-test`, `make e2e-upjet`, `make upgrade-sim`) cover
regressions in CI. This guide is for the opposite case: a human walking through the same
surfaces by hand — to sanity-check a change before it ships, to reproduce a bug report, or to
see what a provider author actually experiences. Each scenario below is copy-pasteable except
for the scratch directory name, which is yours to pick (this guide uses `/tmp/xpg-manual-*`).

None of this touches the automated scripts' fixed paths or kind cluster names, so it is safe to
run alongside CI or another `make e2e-test`/`make e2e-upjet` run on the same machine — see
[Prerequisites](#prerequisites) for the one rule that keeps them from colliding.

## Prerequisites

| Requirement | Check |
|---|---|
| Go (the version this repo targets, from `pkg/versions/dependencies.yaml`'s `go_version`) | `go version` |
| Docker (scenarios B and the live half of A; not needed for C's ownership checks, D, or E) | `docker info` |
| Network access (upjet scenarios download Terraform and a provider schema; native `init`/`create api` fetch Go modules) | `curl -sI https://proxy.golang.org >/dev/null && echo ok` |

**Cluster names.** Every scaffolded provider's `make e2e` creates a kind cluster named
`<PROJECT_NAME>-e2e`, where `PROJECT_NAME` is derived from `--repo`. The automated scripts use
fixed repos (`github.com/example/provider-template`, `github.com/example/provider-k8s`), so
picking any other `--repo` for these manual runs — as this guide does — already avoids a name
collision. Nothing further to configure.

Build the binary once before starting:

```bash
cd /path/to/xp-provider-gen   # this repo
make build                    # writes bin/xp-provider-gen
export BIN="$(pwd)/bin/xp-provider-gen"
```

Every command below assumes `$BIN` is set to that path.

## Scenario A — native flavor, full flow

~15 minutes. Needs Docker.

```bash
mkdir -p /tmp/xpg-manual-a && cd /tmp/xpg-manual-a
"$BIN" init --domain=example.com --repo=github.com/example/provider-manual-a
"$BIN" create api --group=sample --version=v1 --kind=Widget
"$BIN" create api --group=sample --version=v1 --kind=Gadget
make generate
make build
"$BIN" create-test --name widget-basic --kind Widget
make e2e
```

`make e2e` builds the xpkg, stands up a dedicated kind cluster (`provider-manual-a-e2e`),
installs Crossplane, deploys the provider, runs uptest's create → Ready/Synced → delete
lifecycle for every example under `examples/*/*.yaml`, then `make test-behavior` (chainsaw),
which includes the `widget-basic` test just scaffolded.

**Expected result:**
- [ ] `init` and both `create api` runs exit 0 and leave a clean git tree (`git status`)
- [ ] `make generate` produces CRDs under `package/crds/` for both kinds
- [ ] `make build` succeeds
- [ ] `create-test` writes `test/behavior/widget-basic/chainsaw-test.yaml`
- [ ] `make e2e` passes — uptest reports every example Ready/Synced, then chainsaw is green
- [ ] `junit.xml` in the project root mentions `widget-basic`

**Cleanup:**
```bash
make e2e-clean            # deletes the provider-manual-a-e2e kind cluster
cd / && rm -rf /tmp/xpg-manual-a
```

## Scenario B — upjet flavor, full flow

~30 minutes. Needs Docker and network (downloads Terraform and the `hashicorp/kubernetes`
provider schema).

```bash
mkdir -p /tmp/xpg-manual-b && cd /tmp/xpg-manual-b
"$BIN" init --domain=example.com --repo=github.com/example/provider-manual-b \
  --upjet --terraform-provider=hashicorp/kubernetes --terraform-provider-version=2.38.0
"$BIN" create api --group=core --version=v1alpha1 --kind=ConfigMap \
  --terraform-resource=kubernetes_config_map
make generate
```

Write `examples/core/configmap.yaml` — this is the file `make e2e` and `create-test` both use,
copied verbatim from the worked example in
[docs/upjet-provider.md](upjet-provider.md#6-worked-example-managing-a-configmap-with-hashicorpkubernetes)
(the `uptest.upbound.io/*` annotations are what makes it the `make e2e` lifecycle input, not just
a manifest you'd `kubectl apply` by hand):

```bash
cat > examples/core/configmap.yaml <<'EOF'
# uptest.upbound.io/* annotations make this the make e2e lifecycle input too.
apiVersion: core.example.m.com/v1alpha1
kind: ConfigMap
metadata:
  name: example
  namespace: crossplane-system
  annotations:
    uptest.upbound.io/timeout: "120"
    uptest.upbound.io/conditions: "Ready,Synced"
spec:
  forProvider:
    metadata:
      - name: example
        namespace: default
    data:
      hello: world
  providerConfigRef:
    name: default
    kind: ProviderConfig
EOF
"$BIN" create-test --name configmap-basic --kind ConfigMap
make e2e
```

`make e2e` here is the whole story: it builds, brings up its own kind control plane
(`controlplane.down` then `controlplane.up`), deploys the provider, and uptest itself runs
`test/setup.sh` to apply the ProviderConfig and credentials before driving the ConfigMap example
through create → Ready/Synced → delete. There is no separate manual `make local-deploy` or
`KUBECTL=kubectl ./test/setup.sh` step in this flow — those exist for a slower, more inspectable
alternative (deploy once, then apply examples by hand), not for this end-to-end path.

It is fine to stop the moment the kind cluster comes up — a full `make e2e` run is not required
to have exercised the scaffold→generate→example→create-test path, which is the part specific to
this tool. If you do stop it early (`Ctrl-C` or `kill` the `make` process), tear the cluster down
with the **Cleanup** commands below before moving on.

**Expected result:**
- [ ] `init` scaffolds the upjet layout; `Makefile` references `hashicorp/kubernetes`
- [ ] `create api --terraform-resource=kubernetes_config_map` writes `config/configmap/config.go`
      and wires it into `config/zz_resources.go`
- [ ] `make generate` produces `apis/**/zz_configmap_types.go`, a controller, and a CRD from the
      real Terraform schema
- [ ] `create-test` writes `test/behavior/configmap-basic/chainsaw-test.yaml`
- [ ] `make e2e` creates a kind cluster named `provider-manual-b-e2e` and (if left to finish)
      reports the ConfigMap example Ready/Synced, then a green chainsaw run

**Cleanup:**
```bash
make e2e-clean || kind delete cluster --name=provider-manual-b-e2e
cd / && rm -rf /tmp/xpg-manual-b
```

## Scenario C — update and ownership

Two short sub-sections, one per flavor. No Docker needed.

### C.1 — native

```bash
mkdir -p /tmp/xpg-manual-c-native && cd /tmp/xpg-manual-c-native
"$BIN" init --domain=example.com --repo=github.com/example/provider-manual-c-native
"$BIN" create api --group=sample --version=v1 --kind=Widget
printf '\n// hand-edit: tool-owned\n' >> internal/controller/widget/wiring.go
printf '\n// hand-edit: user-owned\n' >> internal/controller/widget/external.go
git add -A && git commit -m "simulate local edits to a tool-owned and a user-owned file"
"$BIN" update
git diff --stat
```

**Expected result:**
- [ ] `update` exits 0
- [ ] `git diff` shows `internal/controller/widget/wiring.go` reverted to the generated content
      (the hand-edit is gone — `update` refreshes tool-owned files unconditionally)
- [ ] `git diff` does **not** touch `internal/controller/widget/external.go` (the hand-edit
      is still there — `grep -c 'hand-edit: user-owned' internal/controller/widget/external.go`
      prints `1`)

### C.2 — upjet

```bash
mkdir -p /tmp/xpg-manual-c-upjet && cd /tmp/xpg-manual-c-upjet
"$BIN" init --domain=example.com --repo=github.com/example/provider-manual-c-upjet \
  --upjet --terraform-provider=hashicorp/kubernetes --terraform-provider-version=2.38.0
"$BIN" create api --group=core --version=v1alpha1 --kind=Secret --terraform-resource=kubernetes_secret
printf '\n// hand-edit: tool-owned\n' >> config/provider.go
printf '\n// hand-edit: user-owned\n' >> config/secret/config.go
git add -A && git commit -m "simulate local edits to a tool-owned and a user-owned file"
"$BIN" update
git diff --stat
```

**Expected result:**
- [ ] `git diff` shows `config/provider.go` reverted; `config/secret/config.go` untouched
      (same shape as C.1)

Then, on the same directory, the two upjet-only behaviors:

```bash
# A deleted user-owned file is never recreated — update lists it instead.
rm examples/providerconfig/providerconfig.yaml
git add -A && git commit -m "simulate deleting a user-owned example"
"$BIN" update | grep 'Not seeded'
git checkout HEAD~1 -- examples/providerconfig/providerconfig.yaml
git add -A && git commit -m "restore the example"

# A dirty tree is refused outright, on either flavor.
printf '\n// dirty\n' >> config/provider.go
"$BIN" update; echo "exit: $?"
git checkout -- config/provider.go
```

**Expected result:**
- [ ] The first `update` prints a line containing `Not seeded` and
      `examples/providerconfig/providerconfig.yaml`, and does not recreate the file
- [ ] The second `update` exits non-zero with a message that mentions the working tree

**Cleanup:**
```bash
cd / && rm -rf /tmp/xpg-manual-c-native /tmp/xpg-manual-c-upjet
```

## Scenario D — `--force` and `--adopt`

No Docker needed. `--force` re-runs `create api` for a kind that already exists; the check
that matters is that it wins back the tool-owned file (even if you broke it) without touching
your logic. Pollute both files **before** running `--force`, so the assertion is not vacuous.

### D.1 — native `--force`

```bash
mkdir -p /tmp/xpg-manual-d-native && cd /tmp/xpg-manual-d-native
"$BIN" init --domain=example.com --repo=github.com/example/provider-manual-d-native
"$BIN" create api --group=mygroup --version=v1alpha1 --kind=MyType
printf '\n// force-test: tool-owned marker\n' >> internal/controller/mytype/wiring.go
printf '\n// force-test: user-owned marker\n' >> internal/controller/mytype/external.go
git add -A && git commit -m "simulate drift before --force"
"$BIN" create api --group=mygroup --version=v1alpha1 --kind=MyType --force
echo "exit: $?"
grep -c 'force-test: tool-owned marker' internal/controller/mytype/wiring.go || true
grep -c 'force-test: user-owned marker' internal/controller/mytype/external.go
```

**Expected result:**
- [ ] `create api --force` exits 0
- [ ] the tool-owned marker in `wiring.go` is gone (grep count `0`)
- [ ] the user-owned marker in `external.go` is still there (grep count `1`)

### D.2 — upjet `--force`

```bash
mkdir -p /tmp/xpg-manual-d-upjet && cd /tmp/xpg-manual-d-upjet
"$BIN" init --domain=example.com --repo=github.com/example/provider-manual-d-upjet \
  --upjet --terraform-provider=hashicorp/kubernetes --terraform-provider-version=2.38.0
"$BIN" create api --group=core --version=v1alpha1 --kind=Secret --terraform-resource=kubernetes_secret
printf '\n// force-test: tool-owned marker\n' >> config/zz_resources.go
printf '\n// force-test: user-owned marker\n' >> config/secret/config.go
git add -A && git commit -m "simulate drift before --force"
"$BIN" create api --group=core --version=v1alpha1 --kind=Secret \
  --terraform-resource=kubernetes_secret --force
echo "exit: $?"
grep -c 'force-test: tool-owned marker' config/zz_resources.go || true
grep -c 'force-test: user-owned marker' config/secret/config.go
```

`--terraform-resource` is required on every upjet `create api`, `--force` included — without it
the command fails fast with a `missing flag` error before touching anything.

**Expected result:**
- [ ] `create api --force` exits 0
- [ ] the tool-owned marker in `config/zz_resources.go` is gone (grep count `0`)
- [ ] the user-owned marker in `config/secret/config.go` is still there (grep count `1`)

### D.3 — upjet `--adopt`

Continue in the same `/tmp/xpg-manual-d-upjet` directory. `--adopt` retrofits a provider that
predates the ownership contract: it adds the generated header back to tool-owned files it
recognizes, and stamps the generator version into `PROJECT`.

```bash
git add -A && git commit -m "commit the --force result" || true
grep -v 'Code generated by xp-provider-gen' config/provider.go > /tmp/provider.go.tmp \
  && mv /tmp/provider.go.tmp config/provider.go
git add -A && git commit -m "simulate a pre-contract provider by stripping the header"
"$BIN" update --adopt
git diff --name-only HEAD
grep -c 'DO NOT EDIT' config/provider.go
```

**Expected result:**
- [ ] stdout contains `Adopted 1 tool-owned file(s)`
- [ ] `config/provider.go` has its `DO NOT EDIT` header back (grep count `1`)
- [ ] `git diff --name-only` lists `config/provider.go`; it may also list `PROJECT` — `--adopt`
      stamps the generator's version there the first time a provider is adopted or updated,
      so expect that file too unless a prior `update`/`--adopt` already stamped it

**Cleanup:**
```bash
cd / && rm -rf /tmp/xpg-manual-d-native /tmp/xpg-manual-d-upjet
```

## Scenario E — moving an old single-file Makefile onto the refreshable pipeline

Providers scaffolded before the `hack/xp-provider-gen.mk` split keep their whole build
pipeline in a user-owned `Makefile`, which `update` never touches. This walks through
migrating one by hand, following
[docs/provider-guide.md](provider-guide.md#moving-an-older-makefile-onto-the-refreshable-pipeline).

You need a binary built from before the split to produce that old layout. `main` no longer
works for this — the split landed in [#161](https://github.com/cychiang/xp-provider-gen/pull/161)
and has been on `main` since, so a binary built from `main` produces the new layout, not the
old one. Pin to the commit right before that merge instead:
`386b534` (`43e1217^`, the parent of the merge commit). The only reliable way to get a binary
from an arbitrary commit without disturbing your current checkout is a second worktree — do
**not** use `git stash` for this: it does not switch branches, and this repo's stash stack is
shared with every other worktree on the machine. A worktree at a specific commit (detached
HEAD) can coexist with any branch checkout, including one already on `main`, so this never
collides with another worktree on the machine:

```bash
git worktree add /tmp/xpg-old 386b534
make -C /tmp/xpg-old build
OLD_BIN=/tmp/xpg-old/bin/xp-provider-gen

mkdir -p /tmp/xpg-manual-e && cd /tmp/xpg-manual-e
"$OLD_BIN" init --domain=example.com --repo=github.com/example/provider-manual-e
ls hack/ 2>/dev/null || echo "(no hack/ dir — this is the old single-Makefile layout)"
```

Now run `update` from **this** branch's binary — the one built at the top of this guide:

```bash
"$BIN" update
ls hack/xp-provider-gen.mk
git diff --stat Makefile   # expect no output: update never touches the old Makefile
```

**Expected result (before migrating):**
- [ ] the freshly-scaffolded provider has no `hack/` directory (or none containing
      `xp-provider-gen.mk`)
- [ ] `update` seeds `hack/xp-provider-gen.mk` and exits 0
- [ ] `Makefile` is byte-for-byte unchanged (`git diff --stat Makefile` prints nothing)

Migrate `Makefile` by hand: keep the project variables, drop everything else, and include the
seeded fragment.

```bash
grep -nE '^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*(\+=|\?=|:=|=)' hack/xp-provider-gen.mk
```

Check that output against the table in
[docs/provider-guide.md](provider-guide.md#moving-an-older-makefile-onto-the-refreshable-pipeline)
for any variable you had customized (a fresh scaffold has none, so the replacement below is
exactly `PROJECT_NAME`, `PROJECT_REPO`, `PLATFORMS`, plus the include):

```bash
cat > Makefile <<'EOF'
# ====================================================================================
# Setup Project
PROJECT_NAME := provider-manual-e
PROJECT_REPO := github.com/example/provider-manual-e

PLATFORMS ?= linux_amd64 linux_arm64

include hack/xp-provider-gen.mk
EOF
make generate
make build
```

**Expected result (after migrating):**
- [ ] `make generate` exits 0
- [ ] `make build` exits 0 (produces a local xpkg under `_output/`)

**Cleanup:**
```bash
cd / && rm -rf /tmp/xpg-manual-e
git -C /path/to/xp-provider-gen worktree remove /tmp/xpg-old
```
