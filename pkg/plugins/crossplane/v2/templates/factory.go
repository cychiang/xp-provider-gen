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
)

// TemplateFactory creates templates with minimal configuration
// This is the main entry point for creating all template types
type TemplateFactory struct {
	config config.Config
}

// NewFactory creates a new template factory
func NewFactory(cfg config.Config) *TemplateFactory {
	return &TemplateFactory{config: cfg}
}

// Init Template Methods - delegates to individual template functions

// GoMod creates go.mod template
func (f *TemplateFactory) GoMod() machinery.Template {
	return GoMod(f.config)
}

// Makefile creates Makefile template
func (f *TemplateFactory) Makefile() machinery.Template {
	return Makefile(f.config)
}

// GitIgnore creates .gitignore template
func (f *TemplateFactory) GitIgnore() machinery.Template {
	return GitIgnore(f.config)
}

// README creates README.md template
func (f *TemplateFactory) README() machinery.Template {
	return README(f.config)
}

// MainGo creates cmd/provider/main.go template
func (f *TemplateFactory) MainGo() machinery.Template {
	return MainGo(f.config)
}

// ProviderConfigTypes creates ProviderConfig types
func (f *TemplateFactory) ProviderConfigTypes() machinery.Template {
	return ProviderConfigTypes(f.config)
}

// ProviderConfigRegister creates ProviderConfig registration
func (f *TemplateFactory) ProviderConfigRegister() machinery.Template {
	return ProviderConfigRegister(f.config)
}

// CrossplanePackage creates package/crossplane.yaml
func (f *TemplateFactory) CrossplanePackage() machinery.Template {
	return CrossplanePackage(f.config)
}

// ConfigController creates config controller
func (f *TemplateFactory) ConfigController() machinery.Template {
	return ConfigController(f.config)
}

// ControllerRegister creates controller registration file
func (f *TemplateFactory) ControllerRegister() machinery.Template {
	return ControllerRegister(f.config)
}

// VersionGo creates version management
func (f *TemplateFactory) VersionGo() machinery.Template {
	return VersionGo(f.config)
}

// ClusterDockerfile creates cluster Dockerfile
func (f *TemplateFactory) ClusterDockerfile() machinery.Template {
	return ClusterDockerfile(f.config)
}

// ClusterMakefile creates cluster Makefile
func (f *TemplateFactory) ClusterMakefile() machinery.Template {
	return ClusterMakefile(f.config)
}

// License creates LICENSE file
func (f *TemplateFactory) License() machinery.Template {
	return License(f.config)
}

// API Template Methods - delegates to API template functions

// APITypes creates API types file for managed resource
func (f *TemplateFactory) APITypes(force bool) machinery.Template {
	return APITypes(f.config, force)
}

// APIGroup creates API group registration
func (f *TemplateFactory) APIGroup() machinery.Template {
	return APIGroup(f.config)
}

// Controller creates controller implementation
func (f *TemplateFactory) Controller(force bool) machinery.Template {
	return Controller(f.config, force)
}
