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

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

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

// TestBuildTemplate_DerivesNamespacedDomainFromPersistedSettings pins that
// settings read back from PROJECT carry no NamespacedDomain (json:"-"), and
// applying them must not wipe the value Configure derives from the domain.
func TestBuildTemplate_DerivesNamespacedDomainFromPersistedSettings(t *testing.T) {
	persisted := &core.UpjetSettings{TerraformResourcePrefix: testTerraformProviderName}
	info := AnalyzeTemplatePath("upjet/config/provider.go.tmpl")

	product, err := BuildTemplate(newTestConfig(t), info, WithUpjet(persisted))
	if err != nil {
		t.Fatalf("BuildTemplate: %v", err)
	}
	got := product.NamespacedDomain
	if want := core.NamespacedDomain("example.com"); got != want {
		t.Errorf("NamespacedDomain = %q, want %q", got, want)
	}
}

// TestBuildTemplate_ForceOnlyOverwritesToolOwned pins that --force refreshes
// only templates carrying the generated header; user-owned ones stay SkipFile.
func TestBuildTemplate_ForceOnlyOverwritesToolOwned(t *testing.T) {
	cfg := newTestConfig(t)
	res := fixtureResources(cfg.GetRepository(), cfg.GetDomain())[0]

	tests := []struct {
		template string
		force    bool
		want     machinery.IfExistsAction
	}{
		{"files/internal/controller/KIND/wiring.go.tmpl", true, machinery.OverwriteFile},
		{"files/internal/controller/KIND/external.go.tmpl", true, machinery.SkipFile},
		{"files/apis/GROUP/VERSION/KIND_types.go.tmpl", true, machinery.SkipFile},
		{"files/internal/controller/KIND/wiring.go.tmpl", false, machinery.SkipFile},
		{"files/internal/controller/KIND/external.go.tmpl", false, machinery.SkipFile},
		{"files/apis/GROUP/VERSION/KIND_types.go.tmpl", false, machinery.SkipFile},
	}
	for _, tt := range tests {
		product, err := BuildTemplate(cfg, AnalyzeTemplatePath(tt.template), WithResource(&res), WithForce(tt.force))
		if err != nil {
			t.Fatalf("BuildTemplate(%s): %v", tt.template, err)
		}
		if got := product.GetIfExistsAction(); got != tt.want {
			t.Errorf("%s force=%v: IfExistsAction = %v, want %v", tt.template, tt.force, got, tt.want)
		}
	}
}

// TestCreateAPIForce_KeepsUserOwnedFiles re-runs create api's per-kind render
// with --force over an existing kind: the tool-owned wiring.go is rewritten,
// the user's external.go and types file are left exactly as they were.
func TestCreateAPIForce_KeepsUserOwnedFiles(t *testing.T) {
	const (
		mine     = "// mine\n"
		wiring   = "internal/controller/bucket/wiring.go"
		external = "internal/controller/bucket/external.go"
		types    = "apis/storage/v1alpha1/bucket_types.go"
	)
	mem := afero.NewMemMapFs()
	for _, path := range []string{wiring, external, types} {
		if err := afero.WriteFile(mem, path, []byte(mine), 0o600); err != nil {
			t.Fatalf("seeding %s: %v", path, err)
		}
	}

	cfg := newTestConfig(t)
	res := fixtureResources(cfg.GetRepository(), cfg.GetDomain())[0]
	products, err := NewFactoryForFlavor(cfg, core.FlavorNative).GetAPITemplates(WithForce(true), WithResource(&res))
	if err != nil {
		t.Fatalf("GetAPITemplates: %v", err)
	}
	scaffold := machinery.NewScaffold(machinery.Filesystem{FS: mem},
		machinery.WithConfig(cfg),
		machinery.WithBoilerplate(DefaultBoilerplate()),
		machinery.WithResource(&res),
	)
	if err := scaffold.Execute(AsBuilders(products)...); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	for _, path := range []string{external, types} {
		if got, _ := afero.ReadFile(mem, path); string(got) != mine {
			t.Errorf("%s was overwritten by --force", path)
		}
	}
	if got, _ := afero.ReadFile(mem, wiring); !core.IsToolOwned(got) {
		t.Errorf("%s was not rewritten by --force", wiring)
	}
}
