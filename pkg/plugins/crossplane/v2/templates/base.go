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
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// CrossplaneTemplate provides a base template that reduces boilerplate
// while maintaining full kubebuilder ecosystem compatibility
type CrossplaneTemplate struct {
	machinery.TemplateMixin
	machinery.DomainMixin  
	machinery.RepositoryMixin
	
	// Crossplane-specific common fields
	ProviderName string
	Force        bool
}

// ConfigureDefaults sets up common template defaults from kubebuilder config
// This method follows kubebuilder patterns and works with the machinery framework
func (t *CrossplaneTemplate) ConfigureDefaults(cfg config.Config) {
	// Set domain from kubebuilder config
	if t.Domain == "" && cfg != nil {
		t.Domain = cfg.GetDomain()
		t.DomainMixin = machinery.DomainMixin{Domain: t.Domain}
	}
	
	// Set repository from kubebuilder config  
	if t.Repo == "" && cfg != nil {
		t.Repo = cfg.GetRepository()
		t.RepositoryMixin = machinery.RepositoryMixin{Repo: t.Repo}
	}
	
	// Extract provider name following Crossplane conventions
	if t.ProviderName == "" && t.Repo != "" {
		t.ProviderName = extractProviderName(t.Repo)
	}
}

// SetTemplatePath sets the template path with kubebuilder-compatible substitution
func (t *CrossplaneTemplate) SetTemplatePath(path string) {
	t.Path = path
}

// SetTemplateBody sets the template body content
func (t *CrossplaneTemplate) SetTemplateBody(body string) {
	t.TemplateBody = body
}

// SetIfExistsAction sets the action to take if file exists - kubebuilder standard
func (t *CrossplaneTemplate) SetIfExistsAction(action machinery.IfExistsAction) {
	t.IfExistsAction = action
}

// extractProviderName extracts provider name from repository URL
// Follows Crossplane naming conventions while being flexible
func extractProviderName(repo string) string {
	if repo == "" {
		return "provider-example"
	}
	
	parts := strings.Split(repo, "/")
	if len(parts) == 0 {
		return "provider-example"
	}
	
	// Take the last part as the provider name
	providerName := parts[len(parts)-1]
	
	// If it doesn't start with "provider-", that's okay - be flexible
	// But we'll use it as-is to maintain user intent
	return providerName
}

// InitTemplate provides base functionality for init command templates
type InitTemplate struct {
	CrossplaneTemplate
}

// NewInitTemplate creates a new init template with defaults
func NewInitTemplate(cfg config.Config) *InitTemplate {
	t := &InitTemplate{}
	t.ConfigureDefaults(cfg)
	return t
}

// APITemplate provides base functionality for create api command templates  
type APITemplate struct {
	CrossplaneTemplate
	machinery.ResourceMixin
}

// NewAPITemplate creates a new API template with defaults
func NewAPITemplate(cfg config.Config) *APITemplate {
	t := &APITemplate{}
	t.ConfigureDefaults(cfg)
	return t
}