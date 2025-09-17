/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package templates

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// APITypesTemplateProduct implements API types template
type APITypesTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *APITypesTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = fmt.Sprintf("apis/%s/%s/%s_types.go",
			t.Resource.Group, t.Resource.Version, strings.ToLower(t.Resource.Kind))
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("apis/group/version/types.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = apiTypesTemplate
	}
	return nil
}

const apiTypesTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

// {{ .Resource.Kind }}Parameters are the configurable fields of a {{ .Resource.Kind }}.
type {{ .Resource.Kind }}Parameters struct {
	// TODO: Add your configurable fields here
	Region string ` + "`" + `json:"region"` + "`" + `
}

// {{ .Resource.Kind }}Observation are the observable fields of a {{ .Resource.Kind }}.
type {{ .Resource.Kind }}Observation struct {
	// TODO: Add your observable fields here
	Status string ` + "`" + `json:"status,omitempty"` + "`" + `
}

// A {{ .Resource.Kind }}Spec defines the desired state of a {{ .Resource.Kind }}.
type {{ .Resource.Kind }}Spec struct {
	xpv2.ManagedResourceSpec ` + "`" + `json:",inline"` + "`" + `
	ForProvider              {{ .Resource.Kind }}Parameters ` + "`" + `json:"forProvider"` + "`" + `
}

// A {{ .Resource.Kind }}Status represents the observed state of a {{ .Resource.Kind }}.
type {{ .Resource.Kind }}Status struct {
	xpv1.ResourceStatus ` + "`" + `json:",inline"` + "`" + `
	AtProvider          {{ .Resource.Kind }}Observation ` + "`" + `json:"atProvider,omitempty"` + "`" + `
}

// +kubebuilder:object:root=true

// A {{ .Resource.Kind }} is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,{{ .ProviderName | lower }}},shortName={{ .Resource.Kind | lower }}
type {{ .Resource.Kind }} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{ .Resource.Kind }}Spec   ` + "`" + `json:"spec"` + "`" + `
	Status {{ .Resource.Kind }}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

// +kubebuilder:object:root=true

// {{ .Resource.Kind }}List contains a list of {{ .Resource.Kind }}
type {{ .Resource.Kind }}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{ .Resource.Kind }} ` + "`" + `json:"items"` + "`" + `
}

// {{ .Resource.Kind }} type metadata.
var (
	{{ .Resource.Kind }}Kind             = reflect.TypeOf({{ .Resource.Kind }}{}).Name()
	{{ .Resource.Kind }}GroupKind        = schema.GroupKind{Group: Group, Kind: {{ .Resource.Kind }}Kind}.String()
	{{ .Resource.Kind }}KindAPIVersion   = {{ .Resource.Kind }}Kind + "." + CRDGroupVersion.String()
	{{ .Resource.Kind }}GroupVersionKind = CRDGroupVersion.WithKind({{ .Resource.Kind }}Kind)
)

func init() {
	SchemeBuilder.Register(&{{ .Resource.Kind }}{}, &{{ .Resource.Kind }}List{})
}`

// APIGroupTemplateProduct implements API group registration template
type APIGroupTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *APIGroupTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = fmt.Sprintf("apis/%s/%s/groupversion_info.go",
			t.Resource.Group, t.Resource.Version)
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("apis/group/version/groupversion_info.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = apiGroupTemplate
	}
	return nil
}

const apiGroupTemplate = `{{ .Boilerplate }}

// Package {{ .Resource.Version }} contains API Schema definitions for the {{ .Resource.Group }} {{ .Resource.Version }} API group
// +kubebuilder:object:generate=true
// +groupName={{ .Resource.QualifiedGroup }}
// +versionName={{ .Resource.Version }}
package {{ .Resource.Version }}

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "{{ .Resource.QualifiedGroup }}"
	Version = "{{ .Resource.Version }}"
)

var (
	// CRDGroupVersion is the API Group Version used to register the objects
	CRDGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: CRDGroupVersion}
)`

// ControllerTemplateProduct implements controller template
type ControllerTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *ControllerTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("internal", "controller", t.Resource.Group,
			strings.ToLower(t.Resource.Kind), strings.ToLower(t.Resource.Kind)+".go")
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("internal/controller/kind/controller.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = controllerTemplate
	}
	return nil
}

const controllerTemplate = `{{ .Boilerplate }}

package {{ .Resource.Kind | lower }}

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	"{{ .Repo }}/apis/{{ .Resource.Group }}/{{ .Resource.Version }}"
	apisv1alpha1 "{{ .Repo }}/apis/v1alpha1"
)

const (
	errNot{{ .Resource.Kind }}    = "managed resource is not a {{ .Resource.Kind }} custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCPC       = "cannot get ClusterProviderConfig"
	errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// A NoOpService does nothing.
type NoOpService struct{}

var (
	newNoOpService = func(_ []byte) (interface{}, error) { return &NoOpService{}, nil }
)

// Setup adds a controller that reconciles {{ .Resource.Kind }} managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName({{ .Resource.Version }}.{{ .Resource.Kind }}GroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newNoOpService}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &{{ .Resource.Version }}.{{ .Resource.Kind }}List{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind {{ .Resource.Version }}.{{ .Resource.Kind }}List")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind({{ .Resource.Version }}.{{ .Resource.Kind }}GroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&{{ .Resource.Version }}.{{ .Resource.Kind }}{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        *resource.ProviderConfigUsageTracker
	newServiceFn func(creds []byte) (interface{}, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*{{ .Resource.Version }}.{{ .Resource.Kind }})
	if !ok {
		return nil, errors.New(errNot{{ .Resource.Kind }})
	}

	if err := c.usage.Track(ctx, cr); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	var cd apisv1alpha1.ProviderCredentials

	// Switch to ModernManaged resource to get ProviderConfigRef
	m := mg.(resource.ModernManaged)
	ref := m.GetProviderConfigReference()

	switch ref.Kind {
	case "ProviderConfig":
		pc := &apisv1alpha1.ProviderConfig{}
		if err := c.kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: m.GetNamespace()}, pc); err != nil {
			return nil, errors.Wrap(err, errGetPC)
		}
		cd = pc.Spec.Credentials
	case "ClusterProviderConfig":
		cpc := &apisv1alpha1.ClusterProviderConfig{}
		if err := c.kube.Get(ctx, types.NamespacedName{Name: ref.Name}, cpc); err != nil {
			return nil, errors.Wrap(err, errGetCPC)
		}
		cd = cpc.Spec.Credentials
	default:
		return nil, errors.Errorf("unsupported provider config kind: %s", ref.Kind)
	}

	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := c.newServiceFn(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service interface{}
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*{{ .Resource.Version }}.{{ .Resource.Kind }})
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNot{{ .Resource.Kind }})
	}

	// These fmt statements should be removed in the real implementation.
	fmt.Printf("Observing: %+v", cr)

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*{{ .Resource.Version }}.{{ .Resource.Kind }})
	cr.Status.SetConditions(xpv1.Creating())
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNot{{ .Resource.Kind }})
	}

	fmt.Printf("Creating: %+v", cr)

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*{{ .Resource.Version }}.{{ .Resource.Kind }})
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNot{{ .Resource.Kind }})
	}

	fmt.Printf("Updating: %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*{{ .Resource.Version }}.{{ .Resource.Kind }})
	cr.Status.SetConditions(xpv1.Deleting())
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNot{{ .Resource.Kind }})
	}

	fmt.Printf("Deleting: %+v", cr)

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
`

// ExamplesManagedResourceTemplateProduct implements managed resource examples
type ExamplesManagedResourceTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *ExamplesManagedResourceTemplateProduct) GetPath() string {
	return fmt.Sprintf("examples/%s/%s.yaml",
		strings.ToLower(t.Resource.Group),
		strings.ToLower(t.Resource.Kind))
}

func (t *ExamplesManagedResourceTemplateProduct) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

func (t *ExamplesManagedResourceTemplateProduct) SetTemplateDefaults() error {
	// Load from scaffolds file
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("examples/group/kind.yaml.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = managedResourceExampleTemplate
	}
	return nil
}

const managedResourceExampleTemplate = `apiVersion: {{ .Resource.Group }}.{{ .Domain }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  name: example
  namespace: default
spec:
  forProvider:
    # TODO: Update with your managed resource's configurable fields
    # Example field for demonstration:
    # configurableField: test
  providerConfigRef:
    name: example
    kind: ProviderConfig
---
apiVersion: {{ .Resource.Group }}.{{ .Domain }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  name: cluster-example
  namespace: default
spec:
  forProvider:
    # TODO: Update with your managed resource's configurable fields
    # Example field for demonstration:
    # configurableField: test
  providerConfigRef:
    name: example
    kind: ClusterProviderConfig
`