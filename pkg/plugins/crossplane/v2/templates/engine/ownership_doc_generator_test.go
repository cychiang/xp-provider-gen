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
	"slices"
	"testing"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

// TestGeneratorBodiesCarryOwnershipHeader pins the header literal in the
// generator body files: it used to be spliced in from core.GeneratedHeader at
// compile time, but the bodies now live as files under pkg/templates/generators,
// so drift there would silently stop `update` from refreshing these files.
func TestGeneratorBodiesCarryOwnershipHeader(t *testing.T) {
	for _, name := range []string{
		"apis_register.go.tmpl",
		"controller_register.go.tmpl",
		"ownership_doc.md.tmpl",
	} {
		if !core.IsToolOwned([]byte(templates.GeneratorBody(name))) {
			t.Errorf("generator body %q lost the %q header — update would stop refreshing its output",
				name, core.GeneratedHeader)
		}
	}
	if core.IsToolOwned([]byte(templates.GeneratorBody("gomod.tmpl"))) {
		t.Error("gomod.tmpl must NOT carry the generated header: go.mod is seeded once and user-owned")
	}
}

// TestOwnershipDocClassifiesGeneratorOutputs verifies the doc's entries for
// generator-emitted files (invisible to the template-FS walk) are derived from
// the generators themselves: OverwriteFile lands tool-owned, SkipFile (go.mod,
// seeded once) lands user-owned.
func TestOwnershipDocClassifiesGeneratorOutputs(t *testing.T) {
	g := NewOwnershipDocGenerator(
		NewAPIRegisterGenerator(testRepo, "provider-test", nil),
		NewControllerRegisterGenerator(testRepo, "provider-test", nil),
		NewGoModGenerator(testRepo, nil),
	)

	for _, want := range []string{"apis/register.go", "internal/controller/register.go", ownershipDocPath} {
		if !slices.Contains(g.ToolOwned, want) {
			t.Errorf("tool-owned bucket is missing %q; got %v", want, g.ToolOwned)
		}
	}
	if !slices.Contains(g.UserOwned, goModPath) {
		t.Errorf("user-owned bucket is missing %q; got %v", goModPath, g.UserOwned)
	}
}
