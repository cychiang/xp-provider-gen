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
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// BaseTemplateProduct provides common functionality for all template products
type BaseTemplateProduct struct {
	machinery.TemplateMixin
	machinery.DomainMixin
	machinery.RepositoryMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	templateType TemplateType
	ProviderName string
	Force        bool
	customData   map[string]interface{}
}

// NewBaseTemplateProduct creates a new base template product
func NewBaseTemplateProduct(templateType TemplateType) *BaseTemplateProduct {
	return &BaseTemplateProduct{
		templateType: templateType,
		customData:   make(map[string]interface{}),
	}
}

// GetTemplateType returns the template type
func (t *BaseTemplateProduct) GetTemplateType() TemplateType {
	return t.templateType
}

// Configure sets up the template with configuration
func (t *BaseTemplateProduct) Configure(cfg config.Config) error {
	if cfg != nil {
		t.Domain = cfg.GetDomain()
		t.DomainMixin = machinery.DomainMixin{Domain: t.Domain}
		t.Repo = cfg.GetRepository()
		t.RepositoryMixin = machinery.RepositoryMixin{Repo: t.Repo}
	}

	if t.ProviderName == "" && t.Repo != "" {
		t.ProviderName = extractProviderName(t.Repo)
	}

	// Set default boilerplate
	t.Boilerplate = DefaultBoilerplate()
	t.BoilerplateMixin = machinery.BoilerplateMixin{Boilerplate: t.Boilerplate}

	return nil
}

// SetResource sets the resource for API templates
func (t *BaseTemplateProduct) SetResource(res *resource.Resource) error {
	if res != nil {
		t.Resource = res
		t.ResourceMixin = machinery.ResourceMixin{Resource: t.Resource}
	}
	return nil
}

// SetCustomData sets custom data for the template
func (t *BaseTemplateProduct) SetCustomData(data map[string]interface{}) {
	t.customData = data
}

// GetCustomData returns custom data
func (t *BaseTemplateProduct) GetCustomData() map[string]interface{} {
	return t.customData
}

// SetForce sets the force flag
func (t *BaseTemplateProduct) SetForce(force bool) {
	t.Force = force
	if force {
		t.IfExistsAction = machinery.OverwriteFile
	} else {
		t.IfExistsAction = machinery.Error
	}
}

// GetBase returns the base template product for accessing common functionality
func (t *BaseTemplateProduct) GetBase() *BaseTemplateProduct {
	return t
}

// extractProviderName extracts provider name from repository URL
func extractProviderName(repo string) string {
	if repo == "" {
		return "provider-example"
	}

	parts := strings.Split(repo, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "provider-example"
}
