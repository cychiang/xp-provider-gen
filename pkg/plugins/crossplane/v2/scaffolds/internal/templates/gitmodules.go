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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &GitModules{}

// GitModules scaffolds a .gitmodules file for Crossplane build system submodule
type GitModules struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *GitModules) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(".gitmodules")
	}

	f.TemplateBody = gitModulesTemplate

	return nil
}

const gitModulesTemplate = `[submodule "build"]
	path = build
	url = https://github.com/crossplane/build
`