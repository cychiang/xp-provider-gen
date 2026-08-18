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

package engine

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// BaseTemplateProduct provides common functionality for all template products.
type BaseTemplateProduct struct {
	machinery.TemplateMixin
	machinery.DomainMixin
	machinery.RepositoryMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	ProviderName string
	Force        bool
}

// NewBaseTemplateProduct creates a new base template product.
func NewBaseTemplateProduct() *BaseTemplateProduct {
	return &BaseTemplateProduct{}
}

// Configure sets up the template with configuration.
func (t *BaseTemplateProduct) Configure(cfg config.Config) error {
	if cfg != nil {
		t.Domain = cfg.GetDomain()
		t.DomainMixin = machinery.DomainMixin{Domain: t.Domain}
		t.Repo = cfg.GetRepository()
		t.RepositoryMixin = machinery.RepositoryMixin{Repo: t.Repo}
	}

	if t.ProviderName == "" && t.Repo != "" {
		t.ProviderName = core.ExtractProviderName(t.Repo)
	}

	// Set default boilerplate
	t.Boilerplate = DefaultBoilerplate()
	t.BoilerplateMixin = machinery.BoilerplateMixin{Boilerplate: t.Boilerplate}

	return nil
}

// SetResource sets the resource for API templates.
func (t *BaseTemplateProduct) SetResource(res *resource.Resource) error {
	if res != nil {
		t.Resource = res
		t.ResourceMixin = machinery.ResourceMixin{Resource: t.Resource}
	}
	return nil
}

// SetForce makes the template overwrite an existing file. It is only called
// for --force; the zero-value action (machinery.SkipFile) is the default.
func (t *BaseTemplateProduct) SetForce(force bool) {
	t.Force = force
	if force {
		t.IfExistsAction = machinery.OverwriteFile
	}
}
