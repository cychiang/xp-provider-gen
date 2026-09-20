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

package validation

import "testing"

// TestValidateTerraformProviderVersion pins the whitelist: the value is
// interpolated into a JSON string in main.tf.json and into make variable
// assignments in the Makefile, so '"', '#', '$', '\' and newlines all have
// to be impossible, not merely the values a blacklist happened to think of.
func TestValidateTerraformProviderVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{name: "accepts a plain semver", version: "2.38.0", wantErr: false},
		{name: "accepts a prerelease suffix", version: "2.38.0-beta.1", wantErr: false},
		{name: "accepts build metadata", version: "2.38.0+build.5", wantErr: false},
		{name: "rejects empty", version: "", wantErr: true},
		{name: "rejects a two-segment version", version: "2.38", wantErr: true},
		{name: "rejects a leading v", version: "v2.38.0", wantErr: true},
		{name: "rejects a double quote", version: `2.38.0"`, wantErr: true},
		{name: "rejects a dollar sign", version: "2.38.0$(rm -rf /)", wantErr: true},
		{name: "rejects a hash", version: "2.38.0#comment", wantErr: true},
		{name: "rejects a backslash", version: `2.38.0\`, wantErr: true},
		{name: "rejects an embedded newline", version: "2.38.0\nEVIL=1", wantErr: true},
		{name: "rejects a path traversal", version: "../../etc", wantErr: true},
		{name: "rejects a trailing space", version: "2.38.0 ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTerraformProviderVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTerraformProviderVersion(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
			}
		})
	}
}
