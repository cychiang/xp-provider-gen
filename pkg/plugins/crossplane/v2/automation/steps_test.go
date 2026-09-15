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

package automation

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestExecutableBitStep pins that the chmod step applies core.FileMode: scripts
// become executable, other files keep the mode they were written with, and
// nothing under .git is touched.
func TestExecutableBitStep(t *testing.T) {
	tests := []struct {
		path string
		want fs.FileMode
	}{
		{path: "test/setup.sh", want: 0o755},
		{path: "Makefile", want: 0o600},
		{path: ".git/hooks/pre-commit.sh", want: 0o600},
	}

	root := t.TempDir()
	for _, tt := range tests {
		p := filepath.Join(root, tt.path)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	if err := NewExecutableBitStep(root).Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			info, err := os.Stat(filepath.Join(root, tt.path))
			if err != nil {
				t.Fatalf("Stat: %v", err)
			}
			if got := info.Mode().Perm(); got != tt.want {
				t.Errorf("mode = %#o, want %#o", got, tt.want)
			}
		})
	}
}
