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
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/versions"
)

// fixtureResources mirrors what createAPISubcommand.InjectResource fills in.
// Two kinds in different groups and versions, so per-kind and per-group
// templates each render more than once.
func fixtureResources(repo, domain string) []resource.Resource {
	mk := func(group, version, kind string) resource.Resource {
		return resource.Resource{
			GVK:        resource.GVK{Group: group, Version: version, Kind: kind, Domain: domain},
			Path:       fmt.Sprintf("%s/apis/%s/%s", repo, group, version),
			API:        &resource.API{CRDVersion: "v1", Namespaced: true},
			Controller: true,
		}
	}
	return []resource.Resource{mk("storage", "v1alpha1", "Bucket"), mk("compute", "v1beta1", "Instance")}
}

// The Terraform provider the upjet fixtures wrap.
const (
	testTerraformProviderName = "kubernetes"
	testTerraformProvider     = "hashicorp/" + testTerraformProviderName
)

// fixtureUpjetSettings are the settings init would persist for a Terraform
// provider. NamespacedDomain is deliberately absent: rendering must derive it.
func fixtureUpjetSettings() *core.UpjetSettings {
	return &core.UpjetSettings{
		TerraformProvider:        testTerraformProvider,
		TerraformProviderName:    testTerraformProviderName,
		TerraformProviderVersion: "2.38.0",
		TerraformProviderRepo:    core.DefaultProviderRepo(testTerraformProvider),
		TerraformDocsPath:        core.DefaultTerraformDocsPath,
		TerraformVersion:         versions.TerraformVersion,
		TerraformResourcePrefix:  testTerraformProviderName,
	}
}

// renderProject renders a whole project of one flavor into memory the way
// `init` followed by one `create api` per fixture resource does: every init
// template and generator, then every per-kind template. Machinery formats Go
// output, so a template producing unparsable Go fails here too.
func renderProject(t *testing.T, flavor core.Flavor) afero.Fs {
	t.Helper()

	mem := afero.NewMemMapFs()
	cfg := newTestConfig(t)
	res := fixtureResources(cfg.GetRepository(), cfg.GetDomain())
	factory := NewFactoryForFlavor(cfg, flavor)

	var settings *core.UpjetSettings
	if flavor == core.FlavorUpjet {
		settings = fixtureUpjetSettings()
	}

	initTemplates, err := factory.GetInitTemplates(WithUpjet(settings))
	if err != nil {
		t.Fatalf("%s: building init templates: %v", flavor, err)
	}
	builders := AsBuilders(initTemplates)
	var deps []versions.Dependency
	if flavor == core.FlavorUpjet {
		builders = append(builders, UpjetCoreGenerators(cfg, res)...)
		deps, err = versions.UpjetGoModDependencies()
	} else {
		builders = append(builders, CoreGenerators(cfg, res)...)
		deps, err = versions.GoModDependencies()
	}
	if err != nil {
		t.Fatalf("%s: loading go.mod dependencies: %v", flavor, err)
	}
	builders = append(builders, NewGoModGenerator(cfg.GetRepository(), deps))

	initScaffold := machinery.NewScaffold(machinery.Filesystem{FS: mem},
		machinery.WithConfig(cfg),
		machinery.WithBoilerplate(DefaultBoilerplate()),
	)
	if err := initScaffold.Execute(builders...); err != nil {
		t.Fatalf("%s: rendering init templates: %v", flavor, err)
	}

	for i := range res {
		r := &res[i]
		// Mirror create api: per-kind templates get only what PROJECT persists
		// (the resource prefix) plus --terraform-resource, never init's full
		// settings, so they render here exactly as empty as in production.
		var perKind *core.UpjetSettings
		if settings != nil {
			perKind = &core.UpjetSettings{
				TerraformResourcePrefix: settings.TerraformResourcePrefix,
				TerraformResource:       testTerraformProviderName + "_" + strings.ToLower(r.Kind),
			}
		}
		apiTemplates, err := factory.GetAPITemplates(WithResource(r), WithUpjet(perKind))
		if err != nil {
			t.Fatalf("%s: building API templates for %s: %v", flavor, r.Kind, err)
		}
		apiScaffold := machinery.NewScaffold(machinery.Filesystem{FS: mem},
			machinery.WithConfig(cfg),
			machinery.WithBoilerplate(DefaultBoilerplate()),
			machinery.WithResource(r),
		)
		if err := apiScaffold.Execute(AsBuilders(apiTemplates)...); err != nil {
			t.Fatalf("%s: rendering API templates for %s: %v", flavor, r.Kind, err)
		}
	}
	return mem
}

// expectedPaths expands a golden ownership map into the concrete paths a
// rendered project holds: IMAGENAME becomes the project name, and templates
// with resource placeholders appear once per fixture resource.
func expectedPaths(t *testing.T, golden map[string]bool, generators []string) map[string]bool {
	t.Helper()

	cfg := newTestConfig(t)
	res := fixtureResources(cfg.GetRepository(), cfg.GetDomain())
	projectName := core.ExtractProjectName(cfg)

	want := map[string]bool{}
	for _, path := range generators {
		want[path] = true
	}
	for key := range golden {
		path := strings.ReplaceAll(key, placeholderImageName, projectName)
		if !core.PathHasPattern(path, []string{placeholderGroup, placeholderVersion, placeholderKind}) {
			want[path] = true
			continue
		}
		for _, r := range res {
			want[strings.NewReplacer(
				placeholderGroup, strings.ToLower(r.Group),
				placeholderVersion, r.Version,
				placeholderKind, strings.ToLower(r.Kind),
			).Replace(path)] = true
		}
	}
	return want
}

// renderedPaths lists every file in the rendered filesystem.
func renderedPaths(t *testing.T, mem afero.Fs) map[string]bool {
	t.Helper()

	got := map[string]bool{}
	err := afero.Walk(mem, ".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			got[path] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking rendered filesystem: %v", err)
	}
	return got
}

// missingFrom returns the sorted keys of a that are not in b.
func missingFrom(a, b map[string]bool) []string {
	var out []string
	for path := range a {
		if !b[path] {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func TestRenderAllTemplates(t *testing.T) {
	tests := []struct {
		flavor     core.Flavor
		golden     map[string]bool
		generators []string
	}{
		{
			flavor: core.FlavorNative,
			golden: wantOwnership,
			generators: []string{
				"apis/register.go", "internal/controller/register.go", goModPath, ownershipDocPath,
			},
		},
		{
			flavor:     core.FlavorUpjet,
			golden:     wantOwnershipUpjet,
			generators: []string{upjetResourcesPath, goModPath, ownershipDocPath},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.flavor), func(t *testing.T) {
			got := renderedPaths(t, renderProject(t, tt.flavor))
			want := expectedPaths(t, tt.golden, tt.generators)

			// A literal anchor: want expands IMAGENAME with the same function
			// production uses, so a regression there would move both sides.
			const anchor = "cluster/images/provider-test/Dockerfile"
			if !got[anchor] {
				t.Errorf("%s not rendered", anchor)
			}

			if missing := missingFrom(want, got); len(missing) > 0 {
				t.Errorf("expected paths not rendered:\n  %s", strings.Join(missing, "\n  "))
			}
			if unexpected := missingFrom(got, want); len(unexpected) > 0 {
				t.Errorf("rendered paths not expected:\n  %s", strings.Join(unexpected, "\n  "))
			}
		})
	}
}

// TestRenderUpjetDerivesNamespacedDomain checks, on real rendered output, that
// the tool-owned files using the namespaced API group get the derived value
// even though the settings passed in carry none.
func TestRenderUpjetDerivesNamespacedDomain(t *testing.T) {
	const want = `"example.m.com"`
	mem := renderProject(t, core.FlavorUpjet)

	for _, path := range []string{"config/provider.go", "apis/namespaced/v1beta1/register.go"} {
		body, err := afero.ReadFile(mem, path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if !strings.Contains(string(body), want) {
			t.Errorf("%s does not contain %s", path, want)
		}
	}
}

// TestRenderUpjetGeneratorsWithoutResources renders the generators the way
// init does, before any kind exists: a fresh upjet provider ships the
// zero-resource config/zz_resources.go.
func TestRenderUpjetGeneratorsWithoutResources(t *testing.T) {
	mem := afero.NewMemMapFs()
	cfg := newTestConfig(t)
	scaffold := machinery.NewScaffold(machinery.Filesystem{FS: mem},
		machinery.WithConfig(cfg),
		machinery.WithBoilerplate(DefaultBoilerplate()),
	)
	if err := scaffold.Execute(UpjetCoreGenerators(cfg, nil)...); err != nil {
		t.Fatalf("rendering upjet generators without resources: %v", err)
	}
	if ok, err := afero.Exists(mem, upjetResourcesPath); err != nil || !ok {
		t.Errorf("%s not rendered (exists=%v, err=%v)", upjetResourcesPath, ok, err)
	}
}
