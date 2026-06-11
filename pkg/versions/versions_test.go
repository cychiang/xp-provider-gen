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

package versions

import (
	"strings"
	"testing"
)

func TestGoModDependencies(t *testing.T) {
	deps, err := GoModDependencies()
	if err != nil {
		t.Fatalf("GoModDependencies() error: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("expected at least one dependency")
	}

	// crossplane-runtime is the load-bearing dependency the upgrade story tracks.
	var found bool
	for _, d := range deps {
		if d.Module == "" || d.Version == "" {
			t.Errorf("incomplete dependency entry: %+v", d)
		}
		if !strings.HasPrefix(d.Version, "v") {
			t.Errorf("version %q for %q should start with 'v'", d.Version, d.Module)
		}
		if d.Module == "github.com/crossplane/crossplane-runtime/v2" {
			found = true
		}
	}
	if !found {
		t.Error("manifest must include crossplane-runtime/v2")
	}
}
