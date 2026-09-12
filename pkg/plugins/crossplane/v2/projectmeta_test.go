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
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// TestLoadProjectMeta pins A8: a PROJECT that declares flavor: upjet but
// carries no upjet: block used to reach a nil-pointer dereference in
// createapi.go (meta.Upjet.TerraformResourcePrefix). loadProjectMeta must
// reject that shape itself rather than swallow the decode error.
func TestLoadProjectMeta(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, cfg config.Config)
		wantErr    string
		wantFlavor core.Flavor
	}{
		{
			name:       "no plugin block defaults to native",
			setup:      func(*testing.T, config.Config) {},
			wantFlavor: core.FlavorNative,
		},
		{
			name: "upjet flavor with a matching upjet block",
			setup: func(t *testing.T, cfg config.Config) {
				t.Helper()
				err := cfg.EncodePluginConfig(pluginName, projectMeta{
					Flavor: core.FlavorUpjet,
					Upjet:  &core.UpjetSettings{TerraformProvider: "hashicorp/kubernetes"},
				})
				if err != nil {
					t.Fatalf("EncodePluginConfig: %v", err)
				}
			},
			wantFlavor: core.FlavorUpjet,
		},
		{
			name: "upjet flavor with no upjet block is invalid",
			setup: func(t *testing.T, cfg config.Config) {
				t.Helper()
				if err := cfg.EncodePluginConfig(pluginName, map[string]any{"flavor": "upjet"}); err != nil {
					t.Fatalf("EncodePluginConfig: %v", err)
				}
			},
			wantErr: "upjet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.New(cfgv3.Version)
			if err != nil {
				t.Fatalf("config.New: %v", err)
			}
			tt.setup(t, cfg)

			meta, err := loadProjectMeta(cfg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadProjectMeta() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadProjectMeta() unexpected error: %v", err)
			}
			if meta.Flavor != tt.wantFlavor {
				t.Errorf("Flavor = %q, want %q", meta.Flavor, tt.wantFlavor)
			}
		})
	}
}
