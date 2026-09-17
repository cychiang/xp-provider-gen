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

package core

import (
	"io/fs"
	"testing"
)

func TestFileMode(t *testing.T) {
	tests := []struct {
		path string
		want fs.FileMode
	}{
		{path: "test/setup.sh", want: 0o755},
		{path: "cluster/local/integration_tests.sh", want: 0o755},
		{path: "Makefile", want: 0o644},
		{path: "internal/x.go", want: 0o644},
		{path: "script.sh.bak", want: 0o644},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := FileMode(tt.path); got != tt.want {
				t.Errorf("FileMode(%q) = %#o, want %#o", tt.path, got, tt.want)
			}
		})
	}
}
