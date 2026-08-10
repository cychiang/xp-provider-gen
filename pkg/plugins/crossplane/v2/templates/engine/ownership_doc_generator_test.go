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
)

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
