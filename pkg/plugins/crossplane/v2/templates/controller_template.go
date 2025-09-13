package templates

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func Controller(cfg config.Config, force bool) machinery.Template {
	t := NewAPITemplate(cfg, "internal/controller/%[group]/%[kind]/%[kind].go", crossplaneControllerTemplate)
	if force {
		t.SetAction(machinery.OverwriteFile)
	}
	return t
}

const crossplaneControllerTemplate = `{{ .Boilerplate }}

package {{ lower .Resource.Kind }}

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"{{ .Repo }}/apis/v1alpha1"
	"{{ .Repo }}/apis/{{ .Resource.Group }}/{{ .Resource.Version }}"
)

// Setup adds a controller that reconciles {{ .Resource.Kind }} managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName({{ .Resource.Version }}.{{ .Resource.Kind }}GroupVersionKind.GroupKind())

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), v1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind({{ .Resource.Version }}.{{ .Resource.Kind }}GroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1alpha1.ProviderConfigUsage{}),
			newServiceFn: {{ lower .Resource.Kind }}.NewService,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&{{ .Resource.Version }}.{{ .Resource.Kind }}{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// TODO: Implement the external connector and service client
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(ctx context.Context, cfg {{ lower .Resource.Kind }}.Config) ({{ lower .Resource.Kind }}.Service, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	// TODO: Implement connection logic to external service
	return &external{}, nil
}

type external struct {
	// TODO: Add external service client fields
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	// TODO: Implement observation logic
	return managed.ExternalObservation{}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	// TODO: Implement creation logic
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// TODO: Implement update logic
	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	// TODO: Implement deletion logic
	return nil
}`