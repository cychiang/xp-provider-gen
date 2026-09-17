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

import (
	"strings"
	"testing"
)

// TestInfoString pins the CLI's own name in the long-form version string.
// This binary was renamed from crossplane-provider-gen to xp-provider-gen
// (see cmd/xp-provider-gen/main.go's commandName), and String() must not
// drift back to the old name.
func TestInfoString(t *testing.T) {
	i := Info{
		Version:   "v1.2.3",
		GitCommit: "abc123",
		BuildDate: "2026-01-01T00:00:00Z",
		GoVersion: "go1.26.8",
		Platform:  "linux/amd64",
	}
	got := i.String()
	if !strings.HasPrefix(got, "xp-provider-gen version ") {
		t.Errorf("String() = %q, want it to start with %q", got, "xp-provider-gen version ")
	}
	if strings.Contains(got, "crossplane-provider-gen") {
		t.Errorf("String() = %q, contains the retired name %q", got, "crossplane-provider-gen")
	}
	for _, want := range []string{i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.Platform} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, want it to contain %q", got, want)
		}
	}
}

func TestInfoShort(t *testing.T) {
	i := Info{Version: "1.2.3"}
	if got, want := i.Short(), "v1.2.3"; got != want {
		t.Errorf("Short() = %q, want %q", got, want)
	}
}
