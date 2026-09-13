# Building an upjet provider

An **upjet** provider wraps a Terraform provider: its API types and controllers
are *generated* from that provider's schema, so instead of writing reconcile
logic you write configuration saying which Terraform resources to expose and
how. Use this flavor when a good Terraform provider already exists for your API.

The two flavors differ in **where the truth about a kind lives**, and
everything else — what you write, what `make generate` does, what adding a
kind means — follows from that. Here the truth is external: a Terraform
provider's schema. You name a resource ("expose `kubernetes_config_map`");
upjet's own generator produces the API types, the controller and the CRDs
from that schema, so `make generate` **is** the per-kind generation step, not
a finalize step. In the native flavor the truth is internal — your own Go
types and hand-written reconcile logic — and `make generate` only produces
derived artifacts (deepcopy, CRDs) from what you already wrote by hand; that
is [the provider guide](provider-guide.md)'s subject, and where to go if
you'd rather design the API and write the reconcile logic yourself.

## 1. Scaffold

```bash
xp-provider-gen init \
  --domain=example.com --repo=github.com/you/provider-k8s \
  --upjet \
  --terraform-provider=hashicorp/kubernetes \
  --terraform-provider-version=2.38.0
```

`--terraform-provider` and `--terraform-provider-version` are required. The docs
repository is guessed from the provider name and can be overridden with
`--terraform-provider-repo` / `--terraform-provider-docs-path`;
`--terraform-version` pins the Terraform CLI used to read the schema.

Unlike a native provider, a freshly scaffolded upjet project **does not compile
yet** — `cmd/provider` imports the API and controller packages that generation
produces. That is expected; step 3 fixes it.

## 2. Add the resources you need

```bash
xp-provider-gen create api \
  --group=core --version=v1alpha1 --kind=Secret \
  --terraform-resource=kubernetes_secret
```

This writes `config/secret/config.go` — where you tune the kind, its API group
and its external-name strategy — and regenerates `config/zz_resources.go`, which
tells upjet which Terraform resources to generate. Repeat per resource; the
include list stays in step automatically.

## 3. Generate

```bash
make generate
```

That downloads Terraform, reads the provider schema into `config/schema.json`,
scrapes the provider's docs into `config/provider-metadata.yaml`, then runs the
upjet pipeline to produce API types (`zz_*_types.go`), controllers
(`zz_controller.go`), CRDs and example manifests. It needs network access and
takes a few minutes the first time.

Re-run it after every `create api` and after changing anything under `config/`.

## 4. Map your credentials

`internal/clients/clients.go` is yours: it turns a ProviderConfig into the
Terraform setup the generated controllers run with. The credentials Secret holds
the Terraform provider's own configuration as JSON, and is passed straight
through — reshape it there if your provider expects something different.

Unlike the native flavor, an upjet provider scaffolds **two** ProviderConfig
scopes: `apis/namespaced/v1beta1` (a namespaced `ProviderConfig` plus a
cluster-scoped `ClusterProviderConfig`) and `apis/cluster/v1beta1` (a
cluster-scoped `ProviderConfig`). Upjet's own generator pipeline expects a
cluster and a namespaced provider config and generates both trees — see
[architecture.md](architecture.md). Both use the full `CommonCredentialSelectors`:
a cross-namespace `secretRef`, plus `Filesystem` and `Environment` sources.
`examples/providerconfig/providerconfig.yaml` ships one of each; whichever scope
a request resolves through, `internal/clients/resolve.go` (tool-owned) turns it
into what `clients.go` receives.

## 5. Build and deploy

```bash
make build
```

The image bakes in what the controllers need at runtime: Terraform itself, the
Terraform provider binary (mirrored into Terraform's filesystem-mirror layout,
so no network access is needed at runtime), and a `.terraformrc` pointing at
that mirror (`cluster/images/<provider>/terraformrc.hcl` — yours, seeded once).
`cmd/provider`'s required flags (`--terraform-version`,
`--terraform-provider-source`, `--terraform-provider-version`) are satisfied
from the image's own `ENV`, so the pod starts with nothing extra to configure.

```bash
make local-deploy   # build, kind cluster, install Crossplane, deploy the provider
```

`local-deploy` creates a dedicated `<provider>-e2e` kind cluster (`KIND_CLUSTER_NAME`,
same convention as the native flavor) rather than reusing whatever cluster you
already have, and switches your `kubectl` context to it. Apply the
scaffolded credentials Secret and ProviderConfig with `test/setup.sh`
(yours, seeded once — waits for the provider package to become Healthy, then
applies `examples/providerconfig/providerconfig.yaml`):

```bash
KUBECTL=kubectl ./test/setup.sh
```

`make e2e` runs the same deploy, then upstream's uptest lifecycle flow against
every file under `examples/*/*.yaml` (except the ProviderConfig example) — no
`UPTEST_EXAMPLE_LIST` to set, the same wildcard convention the native flavor
uses. The catch: `create api` cannot seed a real example for upjet — nothing
about a valid `spec.forProvider` is knowable before `make generate` produces
the schema-derived types — so a fresh scaffold's `examples/` has nothing for
`make e2e` to test until you write one: copy
`examples-generated/namespaced/<group>/<version>/<kind>.yaml` (upjet's own
scraped-doc example) to `examples/<group>/<kind>.yaml` and resolve any `${...}`
Terraform interpolations (e.g. `file()`, `filebase64()`) first — the scraped
copy does not resolve them for you, and applying it as-is silently produces
garbage field values, not an error (see the worked example below, or
`test/README.md` inside your scaffold). `make test-behavior` and `make e2e-clean`
now exist too; `make dev`, `make dev-clean` and `make test-integration` are
still native-only — they run the controller from source, which an upjet
controller cannot do without Terraform, the provider plugin and the
`TERRAFORM_*` env the image bakes in.

## 6. Worked example: managing a ConfigMap with hashicorp/kubernetes

```bash
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-k8s \
  --upjet --terraform-provider=hashicorp/kubernetes --terraform-provider-version=2.38.0
xp-provider-gen create api --group=core --version=v1alpha1 --kind=ConfigMap \
  --terraform-resource=kubernetes_config_map
make generate
make build
make local-deploy
KUBECTL=kubectl ./test/setup.sh
```

Write `examples/core/configmap.yaml` — this is the one file `make e2e` and
`create-test` will both use once it exists, so put it in place now rather than
applying a throwaway manifest:

```yaml
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
```

`core.example.m.com` is `<group>.<namespaced domain>` — `example.com` becomes
`example.m.com` for the namespaced API group (see [§4](#4-map-your-credentials)).
**The MR and its ProviderConfig/Secret must share a namespace** —
`internal/clients/resolve.go`'s namespaced lookup requires it; the scaffold's
own `examples/providerconfig/providerconfig.yaml` places both in
`crossplane-system`, so the manifest above does too.

Apply it and Crossplane creates a real `ConfigMap` named `example` in the
`default` namespace with `data: {hello: world}`:

```bash
kubectl apply -f examples/core/configmap.yaml
```

The external name is the Terraform ID `kubernetes_config_map` uses —
`namespace/name` — so `crossplane.io/external-name` becomes `default/example`.
Editing `spec.forProvider.data` and re-applying updates the real object in
place through Terraform's own update path (no delete-and-recreate); deleting
the MR deletes the ConfigMap. Once this file exists, `make e2e` picks it up by
wildcard with no further wiring, and `xp-provider-gen create-test --kind=ConfigMap`
derives a chainsaw behavior test from the same file.

No credentials are needed for this example: the scaffolded Secret ships
`credentials: "{}"`, which `hashicorp/kubernetes` reads as an empty provider
block and falls through to in-cluster ServiceAccount detection — and
Crossplane's own system RBAC already grants that ServiceAccount access to
ConfigMaps. This is specific to `hashicorp/kubernetes`; a provider wrapping a
cloud API needs real credentials in that Secret.

## 7. Bumping the Terraform provider

Edit `TERRAFORM_PROVIDER_VERSION` in the Makefile, clear the previous version's
Terraform lock, then regenerate:

```bash
rm -rf .work/terraform
make generate
```

Skip the `rm -rf` and the schema step fails outright — Terraform's own lock
file still pins the old version:

```
Error: Failed to query available provider packages
Could not retrieve the list of available versions for provider hashicorp/kubernetes: locked
provider registry.terraform.io/hashicorp/kubernetes 2.38.0 does not match configured version
constraint 3.0.0; must use terraform init -upgrade to allow selection of new versions
```

Three things to expect once generation succeeds:

- **Most kinds won't change.** A bump only reshapes a kind if that resource's
  own Terraform schema changed between the two versions. `hashicorp/kubernetes`
  2.38.0 → 3.0.0 — a full major bump — changes 16 of its 82 resources;
  `kubernetes_config_map` and `kubernetes_secret`, this doc's own examples, are
  not among them and regenerate byte-identical. If you bump and `git diff`
  shows nothing for your kind, the tool is working correctly — the schema
  simply didn't change for that resource.
- **`TERRAFORM_NATIVE_PROVIDER_BINARY` is a second version string**, set once
  at `init` and never re-derived — it still names the old release's binary
  filename after a Makefile-only bump. Update it too
  (`terraform-provider-<name>_v<version>_x5`); nothing reads it incorrectly
  today (Terraform's filesystem-mirror install matches by directory, not
  filename), but a stale value there is one more thing to explain later.
- **A resource the new version drops breaks the build, not just "stops
  updating."** If a Terraform resource type you've configured
  (`create api --terraform-resource=...`) disappears from the bumped schema,
  `make generate` deletes upjet's own generated files for it cleanly but
  leaves your `config/<kind>/config.go` and its entry in
  `config/zz_resources.go` pointing at a Go type that no longer exists —
  `angryjet: undefined: <Kind>`. There is no `remove api`; recover by hand:
  delete `config/<kind>/`, drop its import, `.Configure` call and
  `.TerraformResource` include-list entry from `config/zz_resources.go`, then
  regenerate.

`PROJECT`'s `terraform_provider_version` is stamped once at `init` and nothing
reads it back afterward, so a Makefile-only bump leaves it stale — cosmetic
today, not a functional bug, but don't trust it to answer "what version is
this provider actually built against."

## What you own, and what the tool does

| Yours | Tool's |
|---|---|
| `config/<kind>/config.go` — per-resource configuration | `config/provider.go`, `config/zz_resources.go` |
| `internal/clients/clients.go` — credentials → Terraform setup | `internal/clients/resolve.go` |
| `apis/*/v1beta1/types.go` — ProviderConfig spec | the `go:generate` chain, generator entrypoint, ProviderConfig controllers |

Tool-owned files carry the `DO NOT EDIT` header, but `xp-provider-gen update`
does not support the upjet flavor yet — it refuses to run on one because its
render path is hard-wired to the native template set, not because anything is
missing from PROJECT. That gap is real but infrequent: tool-owned upjet files
do occasionally need a fix that only a newer generator carries (for example, a
wrong hard-coded API group in `apis/*/register.go`), and until `update` learns
this flavor the only way to pick one up is to regenerate a fresh scaffold and
port your own files over by hand. This is a different, rarer need than
[bumping the Terraform provider](#7-bumping-the-terraform-provider) — that one
`update` was never going to help with anyway, since it only ever touches
tool-owned files and the provider version lives in the user-owned Makefile.
Everything without the header is yours forever, and everything upjet itself
generates (`zz_*`) is reproduced by `make generate` and should not be edited
either.

An upjet scaffold also ships `AGENTS.md`, `README.md`, `OWNERS.md` and a `test/`
tree (`test/setup.sh`, `test/README.md`) — all yours, seeded once, same as the
native flavor. `docs/ownership.md` lists the exact, always-accurate split.

## Where to look next

- [upjet's resource configuration guide](https://github.com/crossplane/upjet/blob/main/docs/configuring-a-resource.md)
  — external names, references, sensitive fields, late initialization.
- [docs/templates.md](templates.md) — how this flavor's templates are organised
  if you want to change what a scaffold contains.
