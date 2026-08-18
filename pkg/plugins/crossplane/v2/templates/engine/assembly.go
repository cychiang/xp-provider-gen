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

// AsBuilders adapts a slice of TemplateProduct to the []machinery.Builder that
// machinery.Scaffold.Execute expects (Go cannot implicitly convert the slices).
func AsBuilders(products []TemplateProduct) []machinery.Builder {
	builders := make([]machinery.Builder, 0, len(products))
	for _, p := range products {
		builders = append(builders, p)
	}
	return builders
}

// CoreGenerators returns every deterministically generated tool-owned file:
// the two registration files and the ownership doc. They are always emitted
// together so init, create api and update cannot drift from one another.
// UpjetCoreGenerators returns the deterministically generated files for an
// upjet project: the resource aggregator plus the ownership doc. Upjet itself
// generates the scheme and controller registration (zz_setup.go), so the
// native register generators have no counterpart here.
func UpjetCoreGenerators(cfg config.Config, resources []resource.Resource) []machinery.Builder {
	res := NewUpjetResourcesGenerator(cfg.GetRepository(), resources)
	return []machinery.Builder{res, NewOwnershipDocGenerator(res)}
}

func CoreGenerators(cfg config.Config, resources []resource.Resource) []machinery.Builder {
	repo := cfg.GetRepository()
	providerName := core.ExtractProviderName(repo)
	api := NewAPIRegisterGenerator(repo, providerName, resources)
	controller := NewControllerRegisterGenerator(repo, providerName, resources)
	// The go.mod seeder is wired separately by init (it needs the dependency
	// manifest); a zero-dep instance supplies its path and ownership here.
	doc := NewOwnershipDocGenerator(api, controller, NewGoModGenerator(repo, nil))
	return []machinery.Builder{api, controller, doc}
}
