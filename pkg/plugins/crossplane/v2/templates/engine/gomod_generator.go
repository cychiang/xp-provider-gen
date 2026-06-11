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
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/versions"
)

// GoModGenerator seeds a generated provider's go.mod from the dependency
// manifest. go.mod is user-owned (no header, seeded once): `update` bumps the
// framework versions via `go get`, never by overwriting the file, so users keep
// their own requires.
type GoModGenerator struct {
	machinery.TemplateMixin

	Repo         string
	GoVersion    string
	Dependencies []versions.Dependency
}

var _ machinery.Template = &GoModGenerator{}

// NewGoModGenerator builds the go.mod seed generator for the given module path.
func NewGoModGenerator(repo string, deps []versions.Dependency) *GoModGenerator {
	return &GoModGenerator{
		Repo:         repo,
		GoVersion:    versions.GoVersion,
		Dependencies: deps,
	}
}

func (f *GoModGenerator) SetTemplateDefaults() error {
	f.Path = "go.mod"
	// Seed once: never clobber a provider's go.mod (it holds user requires).
	f.IfExistsAction = machinery.SkipFile
	f.TemplateBody = goModTemplate
	return nil
}

const goModTemplate = `module {{ .Repo }}

go {{ .GoVersion }}

tool sigs.k8s.io/controller-tools/cmd/controller-gen

tool github.com/crossplane/crossplane-tools/cmd/angryjet

require (
{{- range .Dependencies }}
	{{ .Module }} {{ .Version }}
{{- end }}
)
`
