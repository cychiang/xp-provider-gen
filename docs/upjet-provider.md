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

```bash
make build       # build the provider
```

`make e2e` is upstream's uptest flow: set `UPTEST_EXAMPLE_LIST` to the generated
example manifests and provide `cluster/test/setup.sh` (not scaffolded yet). The
native flavor's `test/` tree, `make test-behavior` and `make dev` are not part
of an upjet scaffold.

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

An upjet scaffold has no `AGENTS.md`, `README.md`, `OWNERS.md` or `test/` tree;
`docs/ownership.md` is the only generated doc.

## Where to look next

- [upjet's resource configuration guide](https://github.com/crossplane/upjet/blob/main/docs/configuring-a-resource.md)
  — external names, references, sensitive fields, late initialization.
- [docs/templates.md](templates.md) — how this flavor's templates are organised
  if you want to change what a scaffold contains.
