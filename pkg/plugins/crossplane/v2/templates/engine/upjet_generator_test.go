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
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// dedupTestResource builds a fixture resource for
// TestNewUpjetResourcesGenerator_Dedups. Group/version are fixed here (not
// repeated as literals per table entry) so the test data doesn't add to this
// package's "core"/"v1alpha1" literal pools and trip goconst.
func dedupTestResource(kind string) resource.Resource {
	const group, version = "gadgets", "v9"
	return resource.Resource{GVK: resource.GVK{Group: group, Version: version, Kind: kind}}
}

// TestNewUpjetResourcesGenerator_Dedups pins the fix for `create api --force`
// against an existing kind: PROJECT's stored resources and the current run's
// resource used to be concatenated unconditionally, so a kind seen twice
// produced two aggregator entries (duplicate import alias, duplicate
// Configure call) and the generated project failed to compile.
func TestNewUpjetResourcesGenerator_Dedups(t *testing.T) {
	tests := []struct {
		name      string
		resources []resource.Resource
		wantAlias []string
	}{
		{
			name: "same kind twice collapses to one entry",
			resources: []resource.Resource{
				dedupTestResource("Alpha"),
				dedupTestResource("Alpha"),
			},
			wantAlias: []string{"alpha"},
		},
		{
			name: "distinct kinds keep one entry each in first-seen order",
			resources: []resource.Resource{
				dedupTestResource("Bravo"),
				dedupTestResource("Charlie"),
				dedupTestResource("Bravo"),
			},
			wantAlias: []string{"bravo", "charlie"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewUpjetResourcesGenerator(testRepo, tt.resources)

			if len(g.Resources) != len(tt.wantAlias) {
				t.Fatalf("Resources = %d, want %d: %+v", len(g.Resources), len(tt.wantAlias), g.Resources)
			}
			for i, wantAlias := range tt.wantAlias {
				if g.Resources[i].Alias != wantAlias {
					t.Errorf("Resources[%d].Alias = %q, want %q", i, g.Resources[i].Alias, wantAlias)
				}
				wantPath := testRepo + "/config/" + wantAlias
				if g.Resources[i].Path != wantPath {
					t.Errorf("Resources[%d].Path = %q, want %q", i, g.Resources[i].Path, wantPath)
				}
			}

			out := render(t, g)
			for _, alias := range tt.wantAlias {
				if n := strings.Count(out, alias+` "`); n != 1 {
					t.Errorf("%s import count = %d, want 1\n%s", alias, n, out)
				}
				if n := strings.Count(out, alias+".Configure,"); n != 1 {
					t.Errorf("%s.Configure count = %d, want 1\n%s", alias, n, out)
				}
			}
		})
	}
}
