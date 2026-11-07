package widget

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/xpg-e2e/provider-acme/apis/sample/v1alpha1"
	"github.com/xpg-e2e/provider-acme/internal/provider"
)

const (
	errNotWidget = "managed resource is not a Widget custom resource"
	errNoRegion  = "--region must not be empty"
	externalNS   = "default"
	dataKey      = "value"
)

// External manages one ConfigMap per Widget through the client the
// ProviderConfig's credentials built.
type External struct {
	client *provider.Client
}

// NewExternal builds the Widget external client.
func NewExternal(c *provider.Client) *External {
	return &External{client: c}
}

// ReconcilerOptions tags the reconciler's logger with the --region flag.
func ReconcilerOptions(_ ctrl.Manager, o controller.Options) ([]managed.ReconcilerOption, error) {
	if *provider.Region == "" {
		return nil, errors.New(errNoRegion)
	}
	return []managed.ReconcilerOption{managed.WithLogger(o.Logger.WithValues("region", *provider.Region))}, nil
}

func (e *External) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Widget)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWidget)
	}
	name := meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{}, nil
	}
	cm, err := e.client.Kube.CoreV1().ConfigMaps(externalNS).Get(ctx, name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return managed.ExternalObservation{}, nil
	}
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "getting external configmap")
	}
	cr.Status.AtProvider.ConfigurableField = cm.Data[dataKey]
	cr.Status.SetConditions(xpv2.Available())
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: cm.Data[dataKey] == cr.Spec.ForProvider.ConfigurableField,
	}, nil
}

func (e *External) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Widget)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWidget)
	}
	cr.Status.SetConditions(xpv2.Creating())
	name := "widget-" + cr.GetName()
	meta.SetExternalName(cr, name)
	_, err := e.client.Kube.CoreV1().ConfigMaps(externalNS).Create(ctx, configMap(name, cr), metav1.CreateOptions{})
	return managed.ExternalCreation{}, errors.Wrap(err, "creating external configmap")
}

func (e *External) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Widget)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWidget)
	}
	_, err := e.client.Kube.CoreV1().ConfigMaps(externalNS).Update(ctx, configMap(meta.GetExternalName(cr), cr), metav1.UpdateOptions{})
	return managed.ExternalUpdate{}, errors.Wrap(err, "updating external configmap")
}

func (e *External) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Widget)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWidget)
	}
	cr.Status.SetConditions(xpv2.Deleting())
	err := e.client.Kube.CoreV1().ConfigMaps(externalNS).Delete(ctx, meta.GetExternalName(cr), metav1.DeleteOptions{})
	if kerrors.IsNotFound(err) {
		return managed.ExternalDelete{}, nil
	}
	return managed.ExternalDelete{}, errors.Wrap(err, "deleting external configmap")
}

func (e *External) Disconnect(_ context.Context) error {
	return nil
}

func configMap(name string, cr *v1alpha1.Widget) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: externalNS},
		Data:       map[string]string{dataKey: cr.Spec.ForProvider.ConfigurableField},
	}
}
