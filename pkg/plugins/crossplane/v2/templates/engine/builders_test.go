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

package engine

import (
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

func newTestConfig(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.New(cfgv3.Version)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}
	if err := cfg.SetDomain("example.com"); err != nil {
		t.Fatalf("SetDomain: %v", err)
	}
	if err := cfg.SetRepository(testRepo); err != nil {
		t.Fatalf("SetRepository: %v", err)
	}
	return cfg
}

// TestBuildTemplate_DerivesNamespacedDomainFromPersistedSettings pins C1:
// settings read back from PROJECT carry no NamespacedDomain (json:"-"), and
// applying them must not wipe the value Configure derives from the domain.
func TestBuildTemplate_DerivesNamespacedDomainFromPersistedSettings(t *testing.T) {
	persisted := &core.UpjetSettings{TerraformResourcePrefix: testTerraformProviderName}
	info := AnalyzeTemplatePath("upjet/config/provider.go.tmpl")

	product, err := BuildTemplate(newTestConfig(t), info, WithUpjet(persisted))
	if err != nil {
		t.Fatalf("BuildTemplate: %v", err)
	}
	got := product.(*GenericTemplateProduct).NamespacedDomain
	if want := core.NamespacedDomain("example.com"); got != want {
		t.Errorf("NamespacedDomain = %q, want %q", got, want)
	}
}
