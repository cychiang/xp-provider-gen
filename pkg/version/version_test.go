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

package version

import "testing"

func TestInfoShort(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "bare version gets v-prefixed", version: "1.2.3", want: "v1.2.3"},
		{name: "already v-prefixed version is not doubled", version: "v0.1.0", want: "v0.1.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Info{Version: tt.version}
			if got := i.Short(); got != tt.want {
				t.Errorf("Short() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestResolveVersion pins the go-install fallback: a plain `go install
// pkg@vX.Y.Z` never runs the Makefile's ldflags, so Version stays "dev" and
// resolveVersion must fall back to the module version the Go toolchain
// records in build info.
func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name      string
		ldflags   string
		buildInfo string
		want      string
	}{
		{name: "ldflags-injected version wins", ldflags: "v0.1.0", buildInfo: "x", want: "v0.1.0"},
		{name: "dev falls back to build info module version", ldflags: "dev", buildInfo: "v0.1.0", want: "v0.1.0"},
		{name: "dev with (devel) build info stays dev", ldflags: "dev", buildInfo: "(devel)", want: "dev"},
		{name: "dev with empty build info stays dev", ldflags: "dev", buildInfo: "", want: "dev"},
		{
			name:      "dev falls back to a pseudo-version",
			ldflags:   "dev",
			buildInfo: "v0.0.0-20260923200331-c7b96e644331",
			want:      "v0.0.0-20260923200331-c7b96e644331",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveVersion(tt.ldflags, tt.buildInfo); got != tt.want {
				t.Errorf("resolveVersion(%q, %q) = %q, want %q", tt.ldflags, tt.buildInfo, got, tt.want)
			}
		})
	}
}
