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
	"fmt"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

// upjetResourcesPath is the generated aggregator that tells upjet which
// Terraform resources to generate and how.
const upjetResourcesPath = "config/zz_resources.go"

// upjetResource is one per-resource config package in the aggregator.
type upjetResource struct {
	Alias string // import alias, e.g. secret
	Path  string // import path of the per-resource config package
}

// UpjetResourcesGenerator renders config/zz_resources.go: the include list and
// the configurator list, both derived from the project's resources.
//
// It is regenerated in full on every create api — the same deterministic
// approach the native flavor uses for register.go — so a second resource can
// never leave the first one half-wired.
type UpjetResourcesGenerator struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin

	Resources []upjetResource
}

var _ machinery.Template = &UpjetResourcesGenerator{}

// NewUpjetResourcesGenerator builds the aggregator for the given resources.
func NewUpjetResourcesGenerator(repo string, resources []resource.Resource) *UpjetResourcesGenerator {
	g := &UpjetResourcesGenerator{}
	for _, res := range ManagedResources(resources) {
		pkg := strings.ToLower(res.Kind)
		g.Resources = append(g.Resources, upjetResource{
			Alias: pkg,
			Path:  fmt.Sprintf("%s/config/%s", repo, pkg),
		})
	}
	return g
}

func (f *UpjetResourcesGenerator) SetTemplateDefaults() error {
	f.Path = upjetResourcesPath
	f.IfExistsAction = machinery.OverwriteFile
	f.TemplateBody = templates.GeneratorBody("upjet_resources.go.tmpl")
	return nil
}
