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

var _ machinery.Template = &ControllerTemplate{}

// ControllerTemplate scaffolds the register.go file for controller registration
type ControllerTemplate struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
	
	// ProviderName is the name extracted from the repository
	ProviderName string
}

// SetTemplateDefaults implements machinery.Template
func (f *ControllerTemplate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("internal", "controller", "register.go")
	}

	f.TemplateBody = controllerTemplateTemplate

	return nil
}

const controllerTemplateTemplate = `{{ .Boilerplate }}

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"{{ .Repo }}/internal/controller/config"
)

// Setup creates all {{ .ProviderName }} controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		config.Setup,
		// TODO: Add your managed resource controller setup functions here
		// This section will be populated when you create managed resources using:
		// crossplane-provider-gen create api --group=<group> --version=<version> --kind=<kind>
		//
		// Example:
		// mytype.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}
`