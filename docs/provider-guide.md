# Building and upgrading a provider

How to scaffold a Crossplane provider with `xp-provider-gen`, write your logic in
the right places, and take framework upgrades without losing that logic.

If you only read one thing: **files carrying a generated-code header are rewritten
by `xp-provider-gen update`; everything else is yours forever.** Your provider ships
a generated `docs/ownership.md` listing exactly which is which.

## 1. Scaffold

```bash
xp-provider-gen init --domain=example.com --repo=github.com/you/provider-acme
xp-provider-gen create api --group=compute --version=v1alpha1 --kind=Instance
```

`init` creates the project, wires the crossplane build submodule, runs code
generation, and leaves a single clean commit. `create api` adds a kind and folds
into that commit until you make one of your own.

Both leave a clean working tree. If yours is dirty afterwards, that is a bug worth
reporting.

## 2. What you actually write

Two files per kind, plus two provider-wide files. That is the whole surface.

| You want to | Edit |
|---|---|
| Implement observe/create/update/delete | `internal/controller/<kind>/external.go` |
| Add spec/status fields to a kind | `apis/<group>/<version>/<kind>_types.go` |
| Build the API client from credentials | `internal/provider/client.go` |
| Add CLI flags, adjust controller options | `internal/provider/options.go` |
| Add settings to the ProviderConfig | `apis/v1alpha1/types.go` |

Everything else — the connector, ProviderConfig resolution, credential extraction,
controller registration, `main.go` — is generated and maintained for you.

### The external client

`external.go` holds only your business logic. The connector has already resolved
credentials and built your client by the time it is called:

```go
type External struct {
	client *provider.Client
}

func (e *External) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Instance)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotInstance)
	}

	got, err := e.client.GetInstance(ctx, meta.GetExternalName(cr))
	if kerrors.IsNotFound(err) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.Status.SetConditions(xpv2.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: got.Size == cr.Spec.ForProvider.Size,
	}, nil
}
```

### The API client

`client.go` turns a resolved ProviderConfig into whatever your API needs:

```go
type Client struct {
	http *http.Client
	base string
}

func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	var creds struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(cfg.Credentials, &creds); err != nil {
		return nil, errors.Wrap(err, "cannot parse credentials")
	}
	return &Client{http: newAuthedClient(creds.Token), base: cfg.Spec.Endpoint}, nil
}
```

Note `cfg.Spec.Endpoint` — that field does not exist until you add it.

### Adding settings to the ProviderConfig

`apis/v1alpha1/types.go` is yours. Add a field:

```go
type ProviderConfigSpec struct {
	Credentials ProviderCredentials `json:"credentials"`

	// Endpoint is the API base URL.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
}
```

Run `make generate`, and it is available as `cfg.Spec.Endpoint` in `NewClient`.
Nothing between the two needs changing — the connector passes the whole spec
through. `ProviderConfig` and `ClusterProviderConfig` share this type, so the field
works for both.

### Credentials you resolve yourself

For pod identity, IRSA or workload identity, set the ProviderConfig's credentials
source to `None`. `cfg.Credentials` is then empty, and you build the client from
ambient identity. `cfg.Kube` is available for any lookup the generator did not do
for you.

### CLI flags

`options.go` and `client.go` are the same package, so a flag reaches client
construction with no plumbing:

```go
var region = new(string)

func Flags(app *kingpin.Application) {
	region = app.Flag("region", "Target region.").Envar("REGION").Default("us-east-1").String()
}

func Configure(o *controller.Options) error {
	if *region == "" {
		return errors.New("region must not be empty")
	}
	return nil
}
```

`Flags` runs before parsing; `Configure` runs after, and returning an error aborts
startup with a clear message instead of failing later inside a controller.

### Per-kind reconciler options

`wiring.go` is generated, but it calls `ReconcilerOptions` from your `external.go`,
so per-kind tuning survives updates:

```go
func ReconcilerOptions(mgr ctrl.Manager, o controller.Options) ([]managed.ReconcilerOption, error) {
	return []managed.ReconcilerOption{
		managed.WithInitializers(managed.NewNameAsExternalName(mgr.GetClient())),
	}, nil
}
```

Return an error rather than panicking: controllers start from inside a CRD-readiness
gate callback, where a panic surfaces with no context.

## 3. Names you must not rename

Generated code calls these six by name. Renaming any of them breaks the build:

| Name | File |
|---|---|
| `Client`, `NewClient` | `internal/provider/client.go` |
| `Flags`, `Configure` | `internal/provider/options.go` |
| `NewExternal`, `ReconcilerOptions` | `internal/controller/<kind>/external.go` |

You can change anything else about them — add fields to `Client`, add helpers, split
files in the package. Only the names and signatures above are fixed.

## 4. Upgrading

This is the payoff. When a new `xp-provider-gen` ships — a crossplane-runtime bump,
a fix to the controller wiring, a new framework feature:

```bash
git status                 # must be clean; update refuses otherwise
xp-provider-gen update
git diff                   # review
make reviewable
git commit -m "chore: update provider core"
```

`update` does four things:

1. Regenerates every tool-owned file from the current templates.
2. Seeds any file that is new in this version.
3. **Skips every user-owned file**, whether or not it has changed.
4. Bumps the framework dependency versions in `go.mod` via `go get`, leaving your
   own requires alone.

It stops there deliberately — no commit — so `git diff` is your review surface.

**What you should see in that diff:** tool-owned files, `go.mod` / `go.sum` version
lines, regenerated `zz_generated.*` and CRDs.

**What you should never see:** `external.go`, `client.go`, `options.go`, any
`*_types.go`, or `AGENTS.md`. If one appears, that is a bug in the generator, not
something to work around — please report it with the diff.

If a step fails midway, `git reset --hard` returns you to where you started. That is
why the clean-tree precondition exists.

### Adopting an older provider

A provider generated before the ownership contract existed has no headers, so
`update` cannot tell tool files from yours. Run this once:

```bash
xp-provider-gen update --adopt   # adds headers to recognised tool-owned files
git diff && git commit -m "chore: adopt xp-provider-gen ownership contract"
xp-provider-gen update           # now refreshes them normally
```

Providers generated before the modular layout (`external.go` / `wiring.go` /
`internal/provider`) must be regenerated instead — there is no in-place migration
for that change.

## 5. Where to look next

- `docs/ownership.md` **inside your provider** — the generated, always-accurate list
  of which files are yours.
- `AGENTS.md` inside your provider — a short orientation for humans and coding
  agents. It is yours; add project-specific guidance to it.
- [docs/architecture.md](architecture.md) — how the generator itself is built.
