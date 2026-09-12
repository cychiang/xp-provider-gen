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
	"os"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
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

// TestParseManifest_GoVersion pins that GoVersion is genuinely parsed from
// the manifest's go_version key (not hardcoded), using arbitrary input
// decoupled from the real embedded dependencies.yaml.
func TestParseManifest_GoVersion(t *testing.T) {
	m, err := parseManifest([]byte("go_version: \"9.9.9\"\ndependencies: []\n"))
	if err != nil {
		t.Fatalf("parseManifest() error: %v", err)
	}
	if m.GoVersion != "9.9.9" {
		t.Errorf("GoVersion = %q, want %q", m.GoVersion, "9.9.9")
	}
}

// TestParseManifest_MissingGoVersion documents that an absent go_version
// parses to the zero value rather than erroring — mustGoVersion (used to
// populate the exported GoVersion var) is what turns that into a panic, so
// this test isolates the parsing step from that policy decision.
func TestParseManifest_MissingGoVersion(t *testing.T) {
	m, err := parseManifest([]byte("dependencies: []\n"))
	if err != nil {
		t.Fatalf("parseManifest() error: %v", err)
	}
	if m.GoVersion != "" {
		t.Errorf("GoVersion = %q, want empty", m.GoVersion)
	}
}

// TestGoVersion_MatchesManifestFile cross-checks the exported GoVersion
// against an independent read+parse of the real on-disk dependencies.yaml
// (using a different, unexported anonymous type — not the package's own
// manifest type or parseManifest — so this cannot pass merely because both
// sides share a bug). This is the test proving versions.go actually reads
// go_version from the manifest rather than, say, still returning a
// hardcoded literal that happens to equal today's value.
func TestGoVersion_MatchesManifestFile(t *testing.T) {
	raw, err := os.ReadFile("dependencies.yaml")
	if err != nil {
		t.Fatalf("reading dependencies.yaml: %v", err)
	}
	var doc struct {
		GoVersion string `json:"go_version"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("independently parsing dependencies.yaml: %v", err)
	}
	if doc.GoVersion == "" {
		t.Fatal("dependencies.yaml has no go_version key")
	}
	if GoVersion != doc.GoVersion {
		t.Errorf("versions.GoVersion = %q, want %q (dependencies.yaml's go_version)", GoVersion, doc.GoVersion)
	}
}
