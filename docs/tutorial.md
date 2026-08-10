# Tutorial: build a provider and run it locally

Build **provider-local**, a small but complete Crossplane provider, in about thirty
minutes — scaffold it, implement real reconcile logic, run it against a local
[kind](https://kind.sigs.k8s.io/) cluster, and take a generator upgrade at the end.
Every command and code block in this page has been run exactly as shown.

The provider you build manages `Bucket` resources whose external system is the
cluster itself: each Bucket materializes as a **ConfigMap**. That keeps the whole
tutorial local — no cloud account, no credentials — while exercising every seam a
real provider uses. When you build against a real API later, only the contents of
the seam files change; the shape stays identical.

For the reference companion to this walkthrough — what each seam is for, the names
you must not rename, the full `update` contract — see the
[provider guide](provider-guide.md).

## Prerequisites

- Go (the version in this repo's `go.mod`)
- [kind](https://kind.sigs.k8s.io/) and `kubectl` (Docker running, for kind)
- `xp-provider-gen` built from this repo: `make build` → `bin/xp-provider-gen`

## 1. Scaffold

```bash
mkdir provider-local && cd provider-local
xp-provider-gen init --domain=example.com --repo=github.com/example/provider-local
xp-provider-gen create api --group=storage --version=v1alpha1 --kind=Bucket
```

You now have a buildable provider with a single `Initial commit`. The files you
will touch — and the only ones you need to — are:

| File | You'll make it |
|---|---|
| `apis/storage/v1alpha1/bucket_types.go` | define what a Bucket is |
| `apis/v1alpha1/types.go` | give the ProviderConfig a `namespace` setting |
| `internal/provider/client.go` | build the provider's client |
| `internal/controller/bucket/external.go` | implement observe/create/update/delete |

Everything else — the connector, credential handling, controller wiring,
`main.go` — is generated and refreshed by `update`. The generated
`docs/ownership.md` in your project lists exactly which file is which.

## 2. Define your API

Replace the two TODO structs in `apis/storage/v1alpha1/bucket_types.go`:

```go
// BucketParameters are the configurable fields of a Bucket.
// +kubebuilder:object:generate=true
type BucketParameters struct {
	// Data is written verbatim to the bucket's backing ConfigMap.
	// +optional
	Data map[string]string `json:"data,omitempty"`
}

// BucketObservation are the observable fields of a Bucket.
// +kubebuilder:object:generate=true
type BucketObservation struct {
	// Keys is the number of keys currently in the backing ConfigMap.
	Keys int `json:"keys,omitempty"`
}
```

Then regenerate the CRDs and deepcopy code:

```bash
make generate
```

## 3. Extend the ProviderConfig

Provider-wide settings live on the ProviderConfig, and its spec is yours. Add a
field to `ProviderConfigSpec` in `apis/v1alpha1/types.go`:

```go
type ProviderConfigSpec struct {
	// Credentials required to authenticate to this provider.
	Credentials ProviderCredentials `json:"credentials"`

	// Namespace is where this provider writes the ConfigMaps that back
	// Buckets. Defaults to "default".
	// +optional
	Namespace string `json:"namespace,omitempty"`
}
```

Run `make generate` again. The connector passes the whole spec through, so the
field arrives in your client with no further plumbing.

## 4. Build your client

`internal/provider/client.go` turns a resolved ProviderConfig into whatever your
API needs. For provider-local that is the cluster client the connector already
injects (`cfg.Kube`) plus your new namespace field:

```go
package provider

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client is everything External needs to manage buckets: a cluster client
// and the namespace the backing ConfigMaps live in.
type Client struct {
	Kube      client.Client
	Namespace string
}

// NewClient builds the provider's client from the resolved ProviderConfig.
// This provider manages resources inside the cluster itself, so it uses the
// manager's own client (cfg.Kube) and no external credentials — the
// ProviderConfig's credentials source is "None".
func NewClient(_ context.Context, cfg ClientConfig) (*Client, error) {
	ns := cfg.Spec.Namespace
	if ns == "" {
		ns = "default"
	}
	return &Client{Kube: cfg.Kube, Namespace: ns}, nil
}
```

A real provider would parse `cfg.Credentials` here and build an SDK client — see
the [provider guide](provider-guide.md#the-api-client) for that variant. Keep the
names `Client` and `NewClient`: generated code calls them.

## 5. Implement the external client

`internal/controller/bucket/external.go` is where your business logic lives. By the
time any method runs, the connector has resolved the ProviderConfig and handed you
a ready `*provider.Client`. Replace the scaffolded methods (keep the license
header, and keep `Disconnect` as generated):

```go
package bucket

import (
	"context"
	"maps"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/example/provider-local/apis/storage/v1alpha1"
	"github.com/example/provider-local/internal/provider"
)

const errNotBucket = "managed resource is not a Bucket custom resource"

// External implements the observe/create/update/delete logic for Bucket.
// Each Bucket is backed by a ConfigMap in the namespace the ProviderConfig
// names; the ConfigMap's name is the Bucket's external name.
type External struct {
	client *provider.Client
}

// NewExternal builds the Bucket external client. The connector calls it
// once the ProviderConfig has been resolved into a *provider.Client.
func NewExternal(c *provider.Client) *External {
	return &External{client: c}
}

// ReconcilerOptions returns extra reconciler options for Bucket. Return
// nil when you need none.
func ReconcilerOptions(_ ctrl.Manager, _ controller.Options) ([]managed.ReconcilerOption, error) {
	return nil, nil
}

func (e *External) configMap(cr *v1alpha1.Bucket) types.NamespacedName {
	return types.NamespacedName{Name: meta.GetExternalName(cr), Namespace: e.client.Namespace}
}

func (e *External) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotBucket)
	}
	if meta.GetExternalName(cr) == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cm := &corev1.ConfigMap{}
	if err := e.client.Kube.Get(ctx, e.configMap(cr), cm); err != nil {
		if kerrors.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get ConfigMap")
	}

	cr.Status.AtProvider.Keys = len(cm.Data)
	cr.Status.SetConditions(xpv2.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: maps.Equal(cm.Data, cr.Spec.ForProvider.Data),
	}, nil
}

func (e *External) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotBucket)
	}
	cr.Status.SetConditions(xpv2.Creating())
	meta.SetExternalName(cr, cr.GetName())

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cr.GetName(), Namespace: e.client.Namespace},
		Data:       cr.Spec.ForProvider.Data,
	}
	if err := e.client.Kube.Create(ctx, cm); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create ConfigMap")
	}
	return managed.ExternalCreation{}, nil
}

func (e *External) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotBucket)
	}

	cm := &corev1.ConfigMap{}
	if err := e.client.Kube.Get(ctx, e.configMap(cr), cm); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot get ConfigMap")
	}
	cm.Data = cr.Spec.ForProvider.Data
	if err := e.client.Kube.Update(ctx, cm); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update ConfigMap")
	}
	return managed.ExternalUpdate{}, nil
}

func (e *External) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Bucket)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotBucket)
	}
	cr.Status.SetConditions(xpv2.Deleting())

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: meta.GetExternalName(cr), Namespace: e.client.Namespace},
	}
	if err := e.client.Kube.Delete(ctx, cm); err != nil && !kerrors.IsNotFound(err) {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete ConfigMap")
	}
	return managed.ExternalDelete{}, nil
}

func (e *External) Disconnect(_ context.Context) error {
	return nil
}
```

The pattern to notice: `Observe` decides everything. It reports whether the external
resource exists and whether it matches the spec; the managed-resource reconciler
then calls `Create`, `Update`, or `Delete` for you. You never call them yourself.

## 6. Unit-test it

Your logic takes a `*provider.Client`, so tests inject a fake cluster client —
no cluster needed. Create `internal/controller/bucket/external_test.go`:

```go
package bucket

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	"github.com/example/provider-local/apis/storage/v1alpha1"
	"github.com/example/provider-local/internal/provider"
)

func bucket(name string, data map[string]string) *v1alpha1.Bucket {
	b := &v1alpha1.Bucket{ObjectMeta: metav1.ObjectMeta{Name: name}}
	b.Spec.ForProvider.Data = data
	meta.SetExternalName(b, name)
	return b
}

func TestBucketLifecycle(t *testing.T) {
	ctx := context.Background()
	kube := fake.NewClientBuilder().Build()
	e := NewExternal(&provider.Client{Kube: kube, Namespace: "default"})
	b := bucket("demo", map[string]string{"owner": "team-a"})

	obs, err := e.Observe(ctx, b)
	if err != nil || obs.ResourceExists {
		t.Fatalf("Observe before create: exists=%v err=%v", obs.ResourceExists, err)
	}

	if _, err := e.Create(ctx, b); err != nil {
		t.Fatalf("Create: %v", err)
	}
	obs, err = e.Observe(ctx, b)
	if err != nil || !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Fatalf("Observe after create: %+v err=%v", obs, err)
	}

	b.Spec.ForProvider.Data = map[string]string{"owner": "team-b"}
	obs, _ = e.Observe(ctx, b)
	if obs.ResourceUpToDate {
		t.Fatal("Observe after spec change: want ResourceUpToDate=false")
	}
	if _, err := e.Update(ctx, b); err != nil {
		t.Fatalf("Update: %v", err)
	}
	cm := &corev1.ConfigMap{}
	if err := kube.Get(ctx, e.configMap(b), cm); err != nil || cm.Data["owner"] != "team-b" {
		t.Fatalf("ConfigMap after update: %v err=%v", cm.Data, err)
	}

	if _, err := e.Delete(ctx, b); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	obs, _ = e.Observe(ctx, b)
	if obs.ResourceExists {
		t.Fatal("Observe after delete: want ResourceExists=false")
	}
}
```

Verify everything before running it for real:

```bash
go test ./...
make reviewable
```

## 7. Run it on kind

The generated Makefile has a target that creates a kind cluster, installs your
CRDs, and runs the controller locally (Ctrl-C to stop, and it needs a terminal of
its own):

```bash
make dev
```

Wait for `Starting workers ... controllerKind: Bucket` in the log. In a second
terminal, point a ProviderConfig at your namespace — no Secret, because
credentials come from your kubeconfig via the manager (`source: None`). Replace
`examples/provider/config.yaml` with:

```yaml
apiVersion: example.com/v1alpha1
kind: ProviderConfig
metadata:
  name: example
  namespace: default
spec:
  credentials:
    source: None
  namespace: default
```

and `examples/storage/bucket.yaml` (generated by `make generate`) with:

```yaml
apiVersion: storage.example.com/v1alpha1
kind: Bucket
metadata:
  name: demo
  namespace: default
spec:
  forProvider:
    data:
      owner: team-a
      tier: standard
  providerConfigRef:
    name: example
    kind: ProviderConfig
```

Apply both and watch your provider work:

```bash
kubectl apply -f examples/provider/config.yaml
kubectl apply -f examples/storage/bucket.yaml
kubectl get bucket
```

```
NAME   READY   SYNCED   EXTERNAL-NAME   AGE
demo   True    True     demo            8s
```

```bash
kubectl get configmap demo -o jsonpath='{.data}'
```

```
{"owner":"team-a","tier":"standard"}
```

Change `owner: team-a` to `owner: team-b` in the Bucket manifest, re-apply, and the
ConfigMap follows; `kubectl get bucket demo -o jsonpath='{.status.atProvider.keys}'`
shows your observation field (`2`). Delete the Bucket and the ConfigMap goes with it:

```bash
kubectl delete -f examples/storage/bucket.yaml
kubectl get configmap demo
# Error from server (NotFound): configmaps "demo" not found
```

That is the full managed-resource lifecycle, reconciled by code you wrote in two
files. Tear the cluster down when you're done:

```bash
make dev-clean
```

## 8. Take an upgrade

When a new `xp-provider-gen` ships, refreshing the generated core is one command —
and it cannot touch the files you edited in this tutorial:

```bash
git status            # must be clean
xp-provider-gen update
git diff              # tool-owned files + dependency bumps only
make reviewable
git commit -m "chore: update provider core"
```

The [provider guide](provider-guide.md#4-upgrading) covers what to expect in that
diff and how `--adopt` works for providers that predate the ownership contract.

## Where to go next

- Run your provider's own test suites: `make e2e` reconciles your example
  resources on a throwaway kind cluster, and `make e2e-package` proves the
  production install path (xpkg → Crossplane → Healthy; needs Docker + Helm).
- Add a CLI flag or tune controller options in `internal/provider/options.go` —
  see [CLI flags](provider-guide.md#cli-flags).
- Point `NewClient` at a real API: parse `cfg.Credentials`, keep the same shape.
- Read `docs/ownership.md` and `AGENTS.md` inside your generated provider — the
  always-accurate map of which files are yours.
