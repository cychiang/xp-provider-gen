/*
Copyright 2025 The Kubernetes Authors.

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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &TemplateUpdater{}

// TemplateUpdater updates the template.go file to include new controller imports and setup calls
type TemplateUpdater struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
	machinery.RepositoryMixin
	machinery.BoilerplateMixin

	// Force overwrite existing content
	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *TemplateUpdater) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("internal", "controller", "template.go")
	}

	f.TemplateBody = templateUpdaterTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const templateUpdaterTemplate = `{{ .Boilerplate }}

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"{{ .Repo }}/internal/controller/config"
{{- if .Resource.Group }}
	"{{ .Repo }}/internal/controller/{{ .Resource.Group }}/{{ lower .Resource.Kind }}"
{{- else }}
	"{{ .Repo }}/internal/controller/{{ lower .Resource.Kind }}"
{{- end }}
)

// Setup creates all Template controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		{{ lower .Resource.Kind }}.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
`