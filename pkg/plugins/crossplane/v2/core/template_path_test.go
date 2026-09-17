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

import "testing"

func TestCleanTemplatePath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "files/Makefile.tmpl", want: "Makefile.tmpl"},
		{path: "upjet/config/provider.go.tmpl", want: "config/provider.go.tmpl"},
		{path: "generators/gomod.tmpl", want: "generators/gomod.tmpl"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := CleanTemplatePath(tt.path); got != tt.want {
				t.Errorf("CleanTemplatePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
