# Building an upjet provider

An **upjet** provider wraps a Terraform provider: its API types and controllers
are *generated* from that provider's schema, so instead of writing reconcile
logic you write configuration saying which Terraform resources to expose and
how. Use this flavor when a good Terraform provider already exists for your API.

Prefer writing the reconcile logic yourself? That is the default flavor — see
[the provider guide](provider-guide.md).

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
scaffolded credentials Secret and ProviderConfig with `cluster/test/setup.sh`
(yours, seeded once — waits for the provider package to become Healthy, then
applies `examples/providerconfig/providerconfig.yaml`):

```bash
KUBECTL=kubectl ./cluster/test/setup.sh
```

`make e2e` runs the same deploy plus upstream's uptest lifecycle flow, but still
needs `UPTEST_EXAMPLE_LIST` set by hand to real, applyable example manifests —
the scraped ones under `examples-generated/` can contain unresolved Terraform
interpolations (e.g. `${file(...)}`) and may need hand-editing first. The
native flavor's `test/` tree, `make test-behavior` and `make dev` are still not
part of an upjet scaffold.

## 6. Worked example: managing a ConfigMap with hashicorp/kubernetes

```bash
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-k8s \
  --upjet --terraform-provider=hashicorp/kubernetes --terraform-provider-version=2.38.0
xp-provider-gen create api --group=core --version=v1alpha1 --kind=ConfigMap \
  --terraform-resource=kubernetes_config_map
make generate
make build
make local-deploy
KUBECTL=kubectl ./cluster/test/setup.sh
```

Then apply a `ConfigMap` managed resource:

```yaml
apiVersion: core.example.m.com/v1alpha1
kind: ConfigMap
metadata:
  name: e2e-configmap
  namespace: crossplane-system
spec:
  forProvider:
    metadata:
      - name: e2e-configmap
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
`crossplane-system`, so the MR above does too.

Applying it creates a real `ConfigMap` named `e2e-configmap` in the `default`
namespace with `data: {hello: world}`. The external name is the Terraform ID
`kubernetes_config_map` uses — `namespace/name` — so
`crossplane.io/external-name` becomes `default/e2e-configmap`. Editing
`spec.forProvider.data` and re-applying updates the real object in place
through Terraform's own update path (no delete-and-recreate); deleting the MR
deletes the ConfigMap.

No credentials are needed for this example: the scaffolded Secret ships
`credentials: "{}"`, which `hashicorp/kubernetes` reads as an empty provider
block and falls through to in-cluster ServiceAccount detection — and
Crossplane's own system RBAC already grants that ServiceAccount access to
ConfigMaps. This is specific to `hashicorp/kubernetes`; a provider wrapping a
cloud API needs real credentials in that Secret.

## What you own, and what the tool does

| Yours | Tool's |
|---|---|
| `config/<kind>/config.go` — per-resource configuration | `config/provider.go`, `config/zz_resources.go` |
| `internal/clients/clients.go` — credentials → Terraform setup | `internal/clients/resolve.go` |
| `apis/*/v1beta1/types.go` — ProviderConfig spec | the `go:generate` chain, generator entrypoint, ProviderConfig controllers |

Tool-owned files carry the `DO NOT EDIT` header, but `xp-provider-gen update`
does not support the upjet flavor yet — it refuses to run on one, because the
per-kind Terraform coordinates it would need to safely re-render tool-owned
files live only in `config/<kind>/config.go`, not in PROJECT. Regenerate a
fresh scaffold to pick up an updated tool-owned file for now. Everything
without the header is yours forever, and everything upjet itself generates
(`zz_*`) is reproduced by `make generate` and should not be edited either.

An upjet scaffold has no `AGENTS.md`, `README.md`, `OWNERS.md` or `test/` tree
— only `cluster/test/setup.sh`, for uptest's `--setup-script`. `docs/ownership.md`
is the only generated doc.

## Where to look next

- [upjet's resource configuration guide](https://github.com/crossplane/upjet/blob/main/docs/configuring-a-resource.md)
  — external names, references, sensitive fields, late initialization.
- [docs/templates.md](templates.md) — how this flavor's templates are organised
  if you want to change what a scaffold contains.
