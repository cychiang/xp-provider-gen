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

// BaseTemplate provides the foundation for all Crossplane templates
// This eliminates ~80% of the boilerplate code across all templates
type BaseTemplate struct {
	machinery.TemplateMixin
	machinery.DomainMixin
	machinery.RepositoryMixin
	machinery.BoilerplateMixin
	
	// Common Crossplane fields
	ProviderName string
	Force        bool
}

// Configure sets up all common template defaults from kubebuilder config
func (t *BaseTemplate) Configure(cfg config.Config) {
	if cfg != nil {
		t.Domain = cfg.GetDomain()
		t.DomainMixin = machinery.DomainMixin{Domain: t.Domain}
		t.Repo = cfg.GetRepository()
		t.RepositoryMixin = machinery.RepositoryMixin{Repo: t.Repo}
	}
	
	if t.ProviderName == "" && t.Repo != "" {
		t.ProviderName = extractProviderName(t.Repo)
	}
	
	// Configure boilerplate
	t.Boilerplate = `/*
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
*/`
	
	// Set sensible defaults
	if t.IfExistsAction == 0 {
		t.IfExistsAction = machinery.OverwriteFile
	}
}

// SetPath sets the template file path
func (t *BaseTemplate) SetPath(path string) {
	t.Path = path
}

// SetBody sets the template content
func (t *BaseTemplate) SetBody(body string) {
	t.TemplateBody = body
}

// SetAction sets the action to take if file exists
func (t *BaseTemplate) SetAction(action machinery.IfExistsAction) {
	t.IfExistsAction = action
}

// InitTemplate specializes BaseTemplate for init command templates
type InitTemplate struct {
	BaseTemplate
}

// NewInitTemplate creates an init template with proper configuration
func NewInitTemplate(cfg config.Config, path, body string) *InitTemplate {
	t := &InitTemplate{}
	t.Configure(cfg)
	t.SetPath(path)
	t.SetBody(body)
	return t
}

// SetTemplateDefaults implements machinery.Template
func (t *InitTemplate) SetTemplateDefaults() error {
	return nil
}

// APITemplate specializes BaseTemplate for API templates (has resource info)  
type APITemplate struct {
	BaseTemplate
	machinery.ResourceMixin
}

// NewAPITemplate creates an API template with resource configuration
func NewAPITemplate(cfg config.Config, path, body string) *APITemplate {
	t := &APITemplate{}
	t.Configure(cfg)
	t.SetPath(path)
	t.SetBody(body)
	return t
}

// SetTemplateDefaults implements machinery.Template
func (t *APITemplate) SetTemplateDefaults() error {
	if t.Resource != nil {
		t.Path = t.Resource.Replacer().Replace(t.Path)
	}
	return nil
}

// StaticTemplate for simple templates that don't change
type StaticTemplate struct {
	BaseTemplate
}

// NewStaticTemplate creates a static template (like .gitignore, LICENSE)
func NewStaticTemplate(cfg config.Config, path, body string) *StaticTemplate {
	t := &StaticTemplate{}
	t.Configure(cfg)
	t.SetPath(path)
	t.SetBody(body)
	t.SetAction(machinery.SkipFile) // Don't overwrite static files
	return t
}

// SetTemplateDefaults implements machinery.Template
func (t *StaticTemplate) SetTemplateDefaults() error {
	return nil
}

// ConditionalTemplate for templates that may or may not be generated
type ConditionalTemplate struct {
	BaseTemplate
	Condition func(config.Config) bool
}

// NewConditionalTemplate creates a template that's only generated if condition is true
func NewConditionalTemplate(cfg config.Config, path, body string, condition func(config.Config) bool) *ConditionalTemplate {
	t := &ConditionalTemplate{
		Condition: condition,
	}
	t.Configure(cfg)
	t.SetPath(path)
	t.SetBody(body)
	return t
}

// SetTemplateDefaults implements machinery.Template
func (t *ConditionalTemplate) SetTemplateDefaults() error {
	return nil
}

// ShouldGenerate returns whether this conditional template should be generated
func (t *ConditionalTemplate) ShouldGenerate(cfg config.Config) bool {
	if t.Condition == nil {
		return true
	}
	return t.Condition(cfg)
}

// VersionedTemplate for templates that vary by version
type VersionedTemplate struct {
	BaseTemplate
	MinVersion string
	MaxVersion string
}

// Utility functions

// extractProviderName extracts provider name from repository URL
func extractProviderName(repo string) string {
	if repo == "" {
		return "provider-example"
	}
	
	parts := strings.Split(repo, "/")
	if len(parts) == 0 {
		return "provider-example"
	}
	
	return parts[len(parts)-1]
}

// Factory Functions - these make template creation super simple

// SimpleFile creates a basic file template
func SimpleFile(cfg config.Config, path, body string) machinery.Template {
	return NewInitTemplate(cfg, path, body)
}

// StaticFile creates a static file that won't be overwritten
func StaticFile(cfg config.Config, path, body string) machinery.Template {
	return NewStaticTemplate(cfg, path, body)
}

// APIFile creates an API-related file template  
func APIFile(cfg config.Config, path, body string) machinery.Template {
	return NewAPITemplate(cfg, path, body)
}

// ConditionalFile creates a file only if condition is met
func ConditionalFile(cfg config.Config, path, body string, condition func(config.Config) bool) machinery.Template {
	return NewConditionalTemplate(cfg, path, body, condition)
}