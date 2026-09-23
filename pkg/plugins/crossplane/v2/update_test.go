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

// Tests for update.go: the shared test config/meta helpers (also used by
// reconcile_test.go, adopt_test.go and tfversion_test.go), project loading
// and validation, template rendering, revert advice, and the flag-wiring
// test that has to go through cmd.Execute().
package v2

import (
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/versions"
)

// The group and version every update test resource lives in.
const (
	updateTestGroup   = "core"
	updateTestVersion = "v1alpha1"
)

// newUpdateTestConfig builds a PROJECT config carrying the given plugin block
// and resources, filled in the way createAPISubcommand.InjectResource does.
func newUpdateTestConfig(t *testing.T, meta projectMeta, kinds ...string) config.Config {
	t.Helper()
	cfg, err := config.New(cfgv3.Version)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}
	if err := cfg.SetDomain(testDomain); err != nil {
		t.Fatalf("SetDomain: %v", err)
	}
	if err := cfg.SetRepository("github.com/example/provider-test"); err != nil {
		t.Fatalf("SetRepository: %v", err)
	}
	if err := cfg.EncodePluginConfig(pluginName, meta); err != nil {
		t.Fatalf("EncodePluginConfig: %v", err)
	}
	for _, kind := range kinds {
		res := resource.Resource{
			GVK:        resource.GVK{Group: updateTestGroup, Version: updateTestVersion, Kind: kind, Domain: cfg.GetDomain()},
			Path:       cfg.GetRepository() + "/apis/" + updateTestGroup + "/" + updateTestVersion,
			API:        &resource.API{CRDVersion: "v1", Namespaced: true},
			Controller: true,
		}
		if err := cfg.AddResource(res); err != nil {
			t.Fatalf("AddResource: %v", err)
		}
	}
	return cfg
}

// upjetTestMeta is the plugin block `init --upjet` persists: only the
// resource prefix, never the render-time Terraform settings.
func upjetTestMeta() projectMeta {
	return projectMeta{Flavor: core.FlavorUpjet, Upjet: &core.UpjetSettings{TerraformResourcePrefix: "kubernetes"}}
}

// TestValidateProject pins that update gates PROJECT the way the other
// commands do: an unusable plugin block (such as an unknown flavor) is
// refused, and an upjet project may use kinds that shadow core Kubernetes
// names, as `create api` allows, while a native one may not.
func TestValidateProject(t *testing.T) {
	tests := []struct {
		name       string
		meta       projectMeta
		wantFlavor core.Flavor
		wantErr    string
	}{
		{name: string(core.FlavorNative), meta: projectMeta{Flavor: core.FlavorNative}, wantErr: "reserved"},
		{name: "unstamped reads as native", meta: projectMeta{}, wantErr: "reserved"},
		{name: "upjet allows reserved kinds", meta: upjetTestMeta(), wantFlavor: core.FlavorUpjet},
		{
			name:    "unknown flavor is refused",
			meta:    projectMeta{Flavor: unknownTestFlavor},
			wantErr: `PROJECT is not usable: PROJECT declares unknown flavor "` + string(unknownTestFlavor) + `"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := validateProject(newUpdateTestConfig(t, tt.meta, "Secret"))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("validateProject() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateProject() error = %v", err)
			}
			if meta.Flavor != tt.wantFlavor {
				t.Errorf("validateProject() flavor = %q, want %q", meta.Flavor, tt.wantFlavor)
			}
		})
	}
}

// TestRenderToMemFS pins that update renders each project with its own
// flavor's template set: an upjet project gets its plumbing (with the
// namespaced domain derived, since PROJECT does not persist it) and none of
// the native-only files, and a native project keeps rendering its own.
func TestRenderToMemFS(t *testing.T) {
	tests := []struct {
		name        string
		meta        projectMeta
		wantPaths   []string
		absentPaths []string
	}{
		{
			name:      string(core.FlavorNative),
			meta:      projectMeta{Flavor: core.FlavorNative},
			wantPaths: []string{nativeConnectorPath, makeFragmentPath},
		},
		{
			name:        string(core.FlavorUpjet),
			meta:        upjetTestMeta(),
			wantPaths:   []string{upjetProviderConfigPath, "config/zz_resources.go", makeFragmentPath},
			absentPaths: []string{nativeConnectorPath},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := afero.NewMemMapFs()
			cfg := newUpdateTestConfig(t, tt.meta, "Widget")
			if err := renderToMemFS(cfg, tt.meta, machinery.Filesystem{FS: mem}); err != nil {
				t.Fatalf("renderToMemFS: %v", err)
			}
			for _, p := range tt.wantPaths {
				if ok, _ := afero.Exists(mem, p); !ok {
					t.Errorf("rendered project is missing %s", p)
				}
			}
			for _, p := range tt.absentPaths {
				if ok, _ := afero.Exists(mem, p); ok {
					t.Errorf("rendered project contains %s, which belongs to another flavor", p)
				}
			}
		})
	}

	t.Run("upjet renders with PROJECT's settings and derives the namespaced domain", func(t *testing.T) {
		mem := afero.NewMemMapFs()
		meta := upjetTestMeta()
		if err := renderToMemFS(newUpdateTestConfig(t, meta, "Widget"), meta, machinery.Filesystem{FS: mem}); err != nil {
			t.Fatalf("renderToMemFS: %v", err)
		}
		got, err := afero.ReadFile(mem, upjetProviderConfigPath)
		if err != nil {
			t.Fatalf("reading %s: %v", upjetProviderConfigPath, err)
		}
		for _, want := range []string{`resourcePrefix = "kubernetes"`, `"example.m.com"`} {
			if !strings.Contains(string(got), want) {
				t.Errorf("%s does not contain %s:\n%s", upjetProviderConfigPath, want, got)
			}
		}

		// PROJECT keeps no Terraform CLI version: the tool-owned make fragment
		// must still pin the tool's, never an empty value.
		fragment, err := afero.ReadFile(mem, makeFragmentPath)
		if err != nil {
			t.Fatalf("reading %s: %v", makeFragmentPath, err)
		}
		if want := "export TERRAFORM_VERSION ?= " + versions.TerraformVersion + "\n"; !strings.Contains(string(fragment), want) {
			t.Errorf("%s does not contain %q:\n%s", makeFragmentPath, want, fragment)
		}
	})
}

// Paths the render tests key off.
const (
	nativeConnectorPath     = "internal/provider/connector.go"
	upjetProviderConfigPath = "config/provider.go"
	makeFragmentPath        = "hack/xp-provider-gen.mk"
)

// TestRevertAdvice pins that 'git reset --hard' alone does not undo a failed
// update once reconcile has seeded new (untracked) files — they are left
// behind while the advice implies the tree is clean. The advice must name
// them explicitly instead of blanket-suggesting 'git clean -fd', which would
// also delete unrelated untracked work already in the repo.
func TestRevertAdvice(t *testing.T) {
	if got := revertAdvice(nil); !strings.Contains(got, "git reset --hard") {
		t.Errorf("no seeded files: advice = %q, want it to mention git reset --hard", got)
	}
	if strings.Contains(revertAdvice(nil), "clean") {
		t.Errorf("no seeded files: advice must not suggest git clean: %q", revertAdvice(nil))
	}

	seeded := []string{"apis/register.go", "internal/provider/"}
	got := revertAdvice(seeded)
	for _, want := range seeded {
		if !strings.Contains(got, want) {
			t.Errorf("advice = %q, want it to name seeded path %q", got, want)
		}
	}
	if strings.Contains(got, "clean -fd") {
		t.Errorf("advice must not suggest git clean -fd (would delete unrelated untracked work): %q", got)
	}
}

// TestUpdateCommand_RejectsTerraformVersionFlagBeforeRunning pins that RunE
// actually calls checkTerraformVersionFlag, not just that the function
// rejects in isolation (TestUpdateFlagRejects calls it directly, so it can't
// tell whether RunE is still wired to it). Going through cmd.Execute() with a
// real flag value proves the wiring; the env var makes this reject before
// runUpdate ever touches git or the filesystem, so it's safe to run as a unit
// test with no fixture project.
func TestUpdateCommand_RejectsTerraformVersionFlagBeforeRunning(t *testing.T) {
	t.Setenv("TERRAFORM_PROVIDER_VERSION", "2.37.1")

	cmd := NewUpdateCommand()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--terraform-provider-version=2.38.0"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() = nil, want an error when TERRAFORM_PROVIDER_VERSION is set in the environment")
	}
	if !strings.Contains(err.Error(), "TERRAFORM_PROVIDER_VERSION is set in the environment") {
		t.Errorf("err = %q, want it to name the env-var rejection (RunE must call checkTerraformVersionFlag)", err)
	}
}

func assertContains(t *testing.T, label string, list []string, want string) {
	t.Helper()
	if !slices.Contains(list, want) {
		t.Errorf("%s = %v, want to contain %q", label, list, want)
	}
}
