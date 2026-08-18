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

package v2

import (
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// TestApplyFileRefusesEscapes pins the containment gate: rendered paths derive
// from PROJECT (group/version/kind are substituted into template paths), so a
// hand-edited or attacker-supplied PROJECT must never make `update` write
// outside the project directory.
func TestApplyFileRefusesEscapes(t *testing.T) {
	escapes := []string{
		"../escape.txt",
		"../../../../tmp/escape.txt",
		"/tmp/absolute-escape.txt",
		"apis/../../escape.txt",
		"", // an empty path must not resolve to the working directory
	}
	if runtime.GOOS == "windows" {
		// Backslash is a separator on Windows only; elsewhere these are just
		// unusual file names and are legitimately allowed.
		escapes = append(escapes,
			`..\escape.txt`,
			`C:\absolute-escape.txt`,
			`apis\..\..\escape.txt`,
		)
	}
	for _, rel := range escapes {
		t.Run(rel, func(t *testing.T) {
			src, dst := afero.NewMemMapFs(), afero.NewMemMapFs()
			if err := afero.WriteFile(src, "rendered", []byte("payload"), 0o644); err != nil {
				t.Fatalf("seeding src: %v", err)
			}

			decision, err := applyFile(src, dst, "rendered", rel)
			if err == nil {
				t.Fatalf("applyFile(%q) was allowed; want refusal", rel)
			}
			if !strings.Contains(err.Error(), "refusing to write outside the project") {
				t.Fatalf("unexpected error for %q: %v", rel, err)
			}
			if decision != core.Skip {
				t.Errorf("decision = %v, want Skip", decision)
			}
			// Skip for the empty path: it resolves to the filesystem root,
			// which always exists and says nothing about a write.
			if rel != "" {
				if exists, _ := afero.Exists(dst, rel); exists {
					t.Errorf("%q was written despite refusal", rel)
				}
			}
		})
	}
}

// TestApplyFileAllowsProjectPaths guards against the check being so strict it
// breaks normal rendering.
func TestApplyFileAllowsProjectPaths(t *testing.T) {
	src, dst := afero.NewMemMapFs(), afero.NewMemMapFs()
	if err := afero.WriteFile(src, "rendered", []byte("payload"), 0o644); err != nil {
		t.Fatalf("seeding src: %v", err)
	}
	for _, rel := range []string{"go.mod", "apis/register.go", "internal/controller/thing/wiring.go"} {
		if _, err := applyFile(src, dst, "rendered", rel); err != nil {
			t.Fatalf("applyFile(%q) refused a legitimate path: %v", rel, err)
		}
		if exists, _ := afero.Exists(dst, rel); !exists {
			t.Errorf("%q was not written", rel)
		}
	}
}
