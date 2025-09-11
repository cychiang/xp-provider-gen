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

package hack

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Boilerplate{}

// Boilerplate scaffolds the hack/boilerplate.go.txt file for code generation
type Boilerplate struct {
	machinery.TemplateMixin
	machinery.DomainMixin
	machinery.RepositoryMixin
}

// SetTemplateDefaults implements file.Template
func (f *Boilerplate) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("hack", "boilerplate.go.txt")
	}

	f.TemplateBody = boilerplateTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const boilerplateTemplate = `/*
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