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
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

// wantOwnership is the golden ownership map: every template's output path
// mapped to whether the tool owns it (true) or the user does (false).
//
// Tool-owned files carry the generated header and are overwritten by `update`.
// User-owned files have no header and are never touched once created.
//
// Adding a template REQUIRES adding it here. That is deliberate: a new
// tool-owned template that forgets the header would otherwise be silently
// user-owned, and `update` would never refresh it.
var wantOwnership = map[string]bool{
	"apis/doc.go":      true,
	"apis/generate.go": true,
	"apis/GROUP/VERSION/groupversion_info.go":     true,
	"apis/GROUP/VERSION/KIND_types.go":            false,
	"apis/v1alpha1/register.go":                   true,
	"apis/v1alpha1/types.go":                      false,
	"cluster/images/IMAGENAME/Dockerfile":         false,
	"cluster/images/IMAGENAME/Makefile":           false,
	"cluster/local/integration_tests.sh":          true,
	"cmd/provider/main.go":                        true,
	"examples/GROUP/KIND.yaml":                    false,
	"examples/provider/config.yaml":               false,
	"hack/boilerplate.go.txt":                     false,
	"internal/controller/config/config.go":        true,
	"internal/controller/KIND/external.go":        false,
	"internal/controller/KIND/wiring.go":          true,
	"internal/provider/connector.go":              true,
	"internal/provider/client.go":                 false,
	"internal/provider/options.go":                false,
	"internal/version/version.go":                 true,
	"test/setup.sh":                               false,
	"test/README.md":                              false,
	"test/e2e/KIND-lifecycle.yaml":                false,
	"test/behavior/KIND-pause/chainsaw-test.yaml": false,
	"AGENTS.md":               false,
	"LICENSE":                 false,
	"package/crossplane.yaml": false,
	".gitignore":              false,
	"Makefile":                false,
	"OWNERS.md":               false,
	"README.md":               false,
}

// wantOwnershipUpjet is the golden ownership map for the upjet flavor. Same
// contract as wantOwnership: the upjet plumbing (generator entrypoint, the
// generate chain, ProviderConfig controllers) is tool-owned and refreshed by
// `update`; everything the provider author configures — per-resource config,
// the credentials seam, the ProviderConfig types — is theirs forever.
var wantOwnershipUpjet = map[string]bool{
	"LICENSE":                                                 false,
	"apis/cluster/v1alpha1/doc.go":                            true,
	"apis/cluster/v1alpha1/register.go":                       true,
	"apis/cluster/v1beta1/doc.go":                             true,
	"apis/cluster/v1beta1/register.go":                        true,
	"apis/cluster/v1beta1/types.go":                           false,
	"apis/generate.go":                                        true,
	"apis/namespaced/v1alpha1/doc.go":                         true,
	"apis/namespaced/v1alpha1/register.go":                    true,
	"apis/namespaced/v1beta1/doc.go":                          true,
	"apis/namespaced/v1beta1/register.go":                     true,
	"apis/namespaced/v1beta1/types.go":                        false,
	"cluster/images/IMAGENAME/Dockerfile":                     false,
	"cluster/images/IMAGENAME/Makefile":                       false,
	"cmd/generator/main.go":                                   true,
	"cmd/provider/main.go":                                    true,
	"config/KIND/config.go":                                   false,
	"config/provider.go":                                      true,
	"examples/providerconfig/providerconfig.yaml":             false,
	"hack/boilerplate.go.txt":                                 false,
	"internal/clients/clients.go":                             false,
	"internal/clients/resolve.go":                             true,
	"internal/controller/cluster/providerconfig/config.go":    true,
	"internal/controller/doc.go":                              true,
	"internal/controller/namespaced/providerconfig/config.go": true,
	"internal/features/features.go":                           true,
	"internal/version/version.go":                             true,
	"package/crossplane.yaml":                                 false,
	".gitignore":                                              false,
	"Makefile":                                                false,
}

// enumerateTemplates walks the embedded template filesystem and returns each
// template's output path mapped to its tool-owned status.
//
// It walks templates.TemplateFS directly rather than using any name-keyed map:
// template base names are NOT unique (Makefile.tmpl exists at both project/ and
// cluster/images/IMAGENAME/), so keying by base name silently drops a file.
//
// GenerateOutputPath is what maps a template to the path it actually occupies
// in a generated provider — notably stripping the "project/" prefix that puts
// project/Makefile.tmpl at the provider's root.
func enumerateTemplates(t *testing.T, root string) map[string]bool {
	t.Helper()

	got := map[string]bool{}

	err := fs.WalkDir(templates.TemplateFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !core.IsTemplateFile(path) {
			return nil
		}
		body, err := templates.TemplateFS.ReadFile(path)
		if err != nil {
			return err
		}
		// nil replacements: keep GROUP/VERSION/KIND/IMAGENAME as stable map keys.
		out := filepath.ToSlash(core.GenerateOutputPath(path, nil))
		if _, dup := got[out]; dup {
			t.Fatalf("two templates produce the same output path %q", out)
		}
		got[out] = core.IsToolOwned(body)
		return nil
	})
	if err != nil {
		t.Fatalf("walking template FS: %v", err)
	}
	return got
}

func TestTemplateOwnership(t *testing.T) {
	for root, want := range map[string]map[string]bool{"files": wantOwnership, "upjet": wantOwnershipUpjet} {
		t.Run(root, func(t *testing.T) { assertOwnership(t, root, want) })
	}
}

func assertOwnership(t *testing.T, root string, wantOwnership map[string]bool) {
	got := enumerateTemplates(t, root)

	for path, wantTool := range wantOwnership {
		gotTool, ok := got[path]
		if !ok {
			t.Errorf("template %q is in the golden map but no longer exists", path)
			continue
		}
		if gotTool != wantTool {
			t.Errorf("template %q: ownership is tool=%v, golden says tool=%v\n"+
				"  if this change is intended, update wantOwnership;\n"+
				"  if not, a tool-owned template is missing the %q header",
				path, gotTool, wantTool, core.GeneratedHeader)
		}
	}

	for path := range got {
		if _, ok := wantOwnership[path]; !ok {
			t.Errorf("template %q is new and has no ownership decision;\n"+
				"  add it to wantOwnership (true = tool-owned, needs the generated header)", path)
		}
	}
}

// TestTemplateCountMatchesGolden guards the enumeration itself: any keying that
// is lossy (base names ignore directories) could silently drop a template, so
// this asserts the golden map covers exactly the template files on disk.
func TestTemplateCountMatchesGolden(t *testing.T) {
	for root, want := range map[string]map[string]bool{"files": wantOwnership, "upjet": wantOwnershipUpjet} {
		t.Run(root, func(t *testing.T) { assertCount(t, root, want) })
	}
}

func assertCount(t *testing.T, root string, wantOwnership map[string]bool) {
	var files int
	err := fs.WalkDir(templates.TemplateFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".tmpl") {
			files++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking template FS: %v", err)
	}
	if files != len(wantOwnership) {
		t.Errorf("found %d template files but the golden map has %d entries", files, len(wantOwnership))
	}
}
