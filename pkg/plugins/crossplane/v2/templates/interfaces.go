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
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// TemplateType represents the type of template to create
type TemplateType string

const (
	// Init template types
	GoModTemplateType            TemplateType = "gomod"
	MakefileTemplateType         TemplateType = "makefile"
	READMETemplateType           TemplateType = "readme"
	GitIgnoreTemplateType        TemplateType = "gitignore"
	MainGoTemplateType           TemplateType = "maingo"
	APIsTemplateType             TemplateType = "apis"
	GenerateGoTemplateType       TemplateType = "generatego"
	BoilerplateTemplateType      TemplateType = "boilerplate"
	ProviderConfigTypesType      TemplateType = "providerconfigtypes"
	ProviderConfigRegisterType   TemplateType = "providerconfigregister"
	CrossplanePackageType        TemplateType = "crossplanepackage"
	ConfigControllerType         TemplateType = "configcontroller"
	ControllerRegisterType       TemplateType = "controllerregister"
	VersionGoType                TemplateType = "versiongo"
	ClusterDockerfileType        TemplateType = "clusterdockerfile"
	ClusterMakefileType          TemplateType = "clustermakefile"
	LicenseType                  TemplateType = "license"

	// API template types
	APITypesTemplateType         TemplateType = "apitypes"
	APIGroupTemplateType         TemplateType = "apigroup"
	ControllerTemplateType       TemplateType = "controller"
)

// TemplateProduct defines the interface for all template products
type TemplateProduct interface {
	machinery.Template
	GetTemplateType() TemplateType
	Configure(cfg config.Config) error
	SetResource(res *resource.Resource) error
}

// TemplateFactory defines the abstract factory interface
type TemplateFactory interface {
	CreateInitTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error)
	CreateAPITemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error)
	CreateStaticTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error)
	GetSupportedTypes() []TemplateType
}

// TemplateBuilder builds specific template types
type TemplateBuilder interface {
	Build(cfg config.Config, opts ...Option) (TemplateProduct, error)
	GetTemplateType() TemplateType
}

// Option provides configuration options for template creation
type Option func(*TemplateOptions)

// TemplateOptions holds configuration options for template creation
type TemplateOptions struct {
	Force        bool
	Resource     *resource.Resource
	CustomData   map[string]interface{}
}

// Option functions
func WithForce(force bool) Option {
	return func(opts *TemplateOptions) {
		opts.Force = force
	}
}

func WithResource(resource *resource.Resource) Option {
	return func(opts *TemplateOptions) {
		opts.Resource = resource
	}
}

func WithCustomData(data map[string]interface{}) Option {
	return func(opts *TemplateOptions) {
		opts.CustomData = data
	}
}