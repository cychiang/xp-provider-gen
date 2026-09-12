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
	"bytes"
	"reflect"
	"strings"
	"testing"
	"text/template"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/yaml"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/templates/engine"
)

const testKindSecret = "Secret"

func testExampleResource() resource.Resource {
	return resource.Resource{GVK: resource.GVK{Group: "core", Version: testVersionBeta, Kind: testKindSecret}}
}

// TestLoadExampleManifest pins B3's fix: create-test derives its skeleton
// from examples/<group>/<kind>.yaml instead of hard-coding a native-shaped
// body, so the same loader must work unmodified for a native-shaped example
// (namespace "default", providerConfigRef "example") and an upjet-shaped one
// (namespace "crossplane-system", providerConfigRef "default") alike — no
// flavor branch anywhere in this code.
func TestLoadExampleManifest(t *testing.T) {
	res := testExampleResource()

	tests := []struct {
		name        string
		example     string // file content; omit the file entirely if empty and missingFile is set
		missingFile bool
		wantErr     string
		wantAPI     string
		wantNS      string
		wantSpecHas []string // substrings the rendered spec block must contain
	}{
		{
			name:        "example missing",
			missingFile: true,
			wantErr:     "no example manifest at examples/core/secret.yaml",
		},
		{
			name:    "example malformed YAML",
			example: "apiVersion: core.example.com/v1beta1\n  bad indent: [\n",
			wantErr: "parsing examples/core/secret.yaml",
		},
		{
			name: "example present but no spec",
			example: `apiVersion: core.example.com/v1beta1
kind: Secret
metadata:
  name: example
  namespace: default
`,
			wantErr: "examples/core/secret.yaml has no spec",
		},
		{
			name: "native happy path",
			example: `apiVersion: core.example.com/v1beta1
kind: Secret
metadata:
  name: example
  namespace: default
spec:
  forProvider:
    configurableField: test
  providerConfigRef:
    name: example
    kind: ProviderConfig
`,
			wantAPI:     "core.example.com/v1beta1",
			wantNS:      "default",
			wantSpecHas: []string{"forProvider:", "configurableField: test", "providerConfigRef:", "name: example"},
		},
		{
			name: "upjet happy path",
			example: `apiVersion: core.example.m.com/v1beta1
kind: Secret
metadata:
  name: example
  namespace: crossplane-system
spec:
  forProvider:
    data:
      key: dmFsdWU=
  providerConfigRef:
    name: default
    kind: ProviderConfig
`,
			wantAPI:     "core.example.m.com/v1beta1",
			wantNS:      "crossplane-system",
			wantSpecHas: []string{"forProvider:", "data:", "key: dmFsdWU=", "providerConfigRef:", "name: default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if !tt.missingFile {
				if err := afero.WriteFile(fs, "examples/core/secret.yaml", []byte(tt.example), 0o644); err != nil {
					t.Fatalf("seeding example: %v", err)
				}
			}

			apiVersion, namespace, specYAML, err := loadExampleManifest(fs, res)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("loadExampleManifest() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadExampleManifest() unexpected error: %v", err)
			}
			if apiVersion != tt.wantAPI {
				t.Errorf("apiVersion = %q, want %q", apiVersion, tt.wantAPI)
			}
			if namespace != tt.wantNS {
				t.Errorf("namespace = %q, want %q", namespace, tt.wantNS)
			}
			for _, want := range tt.wantSpecHas {
				if !strings.Contains(specYAML, want) {
					t.Errorf("spec YAML missing %q:\n%s", want, specYAML)
				}
			}
			// Every line of the rendered spec must be indented to splice
			// correctly under the chainsaw skeleton's `spec:` key.
			for _, line := range strings.Split(specYAML, "\n") {
				if line != "" && !strings.HasPrefix(line, strings.Repeat(" ", specIndent)) {
					t.Errorf("spec line not indented by %d spaces: %q", specIndent, line)
				}
			}
		})
	}
}

// TestExamplePath pins the path create-test looks for, which must match
// where 'create api' seeds (native) or tells the author to write (upjet) the
// same file — group and kind lowercased, matching the existing convention in
// createapi.go's next-steps text and files/examples/GROUP/KIND.yaml.tmpl.
func TestExamplePath(t *testing.T) {
	res := resource.Resource{GVK: resource.GVK{Group: "Core", Version: testVersionBeta, Kind: testKindSecret}}
	if got, want := examplePath(res), "examples/core/secret.yaml"; got != want {
		t.Errorf("examplePath() = %q, want %q", got, want)
	}
}

// TestGeneratedExamplePath pins review round 2 item 2: upjet's `make
// generate` writes example file names lowercased (verified against a real
// generated scaffold — examples-generated/namespaced/core/v1alpha1/secret.yaml,
// never .../Secret.yaml), so a PascalCase resource.Resource.Kind (as
// kubebuilder always gives it, e.g. "Secret") must not leak into the hinted
// path — on a case-sensitive filesystem that path would not exist.
func TestGeneratedExamplePath(t *testing.T) {
	res := resource.Resource{GVK: resource.GVK{Group: "Core", Version: testVersion, Kind: "ConfigMap"}}
	want := "examples-generated/namespaced/core/v1alpha1/configmap.yaml"
	if got := generatedExamplePath(res); got != want {
		t.Errorf("generatedExamplePath() = %q, want %q", got, want)
	}
}

// TestChainsawSkeleton_SpecRoundTrips pins the coupling between specIndent
// (this package) and generators/chainsaw_test.yaml.tmpl's `spec:` depth
// (package engine) with a YAML round trip on the FULLY RENDERED output,
// rather than comparing against the same specIndent constant that produced
// the string (that comparison is circular: it can't detect the template and
// the constant drifting apart, since both indentation depths are only ever
// checked against themselves).
//
// The round-trip test in package v2 rather than exporting anything from
// engine: indentYAML/specIndent already live here, ChainsawTestGenerator is
// already imported here (createtest.go), and package v2 has no test-only
// dependents of its own to keep this out of — so this is the one package
// where both halves of the coupling are visible without changing either
// package's public surface.
//
// If generators/chainsaw_test.yaml.tmpl's `spec:` line moves to (or past)
// this package's specIndent, the spliced block stops being a child of
// `spec:` in YAML terms (sibling or shallower indentation breaks the
// parent/child relationship), so `resource.spec` decodes as empty/absent and
// the deep-equal below fails — pinning exactly the coupling worth pinning.
func TestChainsawSkeleton_SpecRoundTrips(t *testing.T) {
	res := resource.Resource{GVK: resource.GVK{Group: "core", Version: testVersionBeta, Kind: testKindSecret}}

	// Values are deliberately not plain Go ints/floats: sigs.k8s.io/yaml
	// round-trips through encoding/json, and comparing against the original
	// Go map (rather than a re-decoded one) would risk spurious int/float64
	// mismatches that have nothing to do with the indentation coupling this
	// test exists to catch.
	wantSpec := map[string]interface{}{
		"forProvider": map[string]interface{}{
			"data": map[string]interface{}{
				"key": "dmFsdWU=",
			},
			"tags": []interface{}{"a", "b"},
		},
		"providerConfigRef": map[string]interface{}{
			"name": "default",
			"kind": "ProviderConfig",
		},
	}

	specBytes, err := yaml.Marshal(wantSpec)
	if err != nil {
		t.Fatalf("marshaling wantSpec: %v", err)
	}
	// The real helper, not a hand-rolled indent — this is what
	// loadExampleManifest actually feeds the generator.
	specYAML := indentYAML(specBytes, specIndent)

	gen := engine.NewChainsawTestGenerator("roundtrip", res, "core.example.m.com/v1beta1", "crossplane-system", specYAML)
	rendered := renderTemplate(t, gen)

	var doc struct {
		Spec struct {
			Steps []struct {
				Try []struct {
					Apply struct {
						Resource struct {
							Spec map[string]interface{} `json:"spec"`
						} `json:"resource"`
					} `json:"apply"`
				} `json:"try"`
			} `json:"steps"`
		} `json:"spec"`
	}
	if err := yaml.Unmarshal([]byte(rendered), &doc); err != nil {
		t.Fatalf("rendered chainsaw test is not valid YAML: %v\n%s", err, rendered)
	}
	if len(doc.Spec.Steps) == 0 || len(doc.Spec.Steps[0].Try) == 0 {
		t.Fatalf("rendered document has no apply step to check:\n%s", rendered)
	}

	gotSpec := doc.Spec.Steps[0].Try[0].Apply.Resource.Spec
	if !reflect.DeepEqual(gotSpec, wantSpec) {
		t.Errorf("spliced spec does not round-trip through the rendered template.\n got:  %#v\nwant: %#v\nrendered:\n%s",
			gotSpec, wantSpec, rendered)
	}
}

// renderTemplate parses and executes a machinery.Template's body the same
// way templates/engine's own render(t, b) test helper does; duplicated here
// (not exported from engine) because this is the one place in package v2
// that needs it, for the reason explained on TestChainsawSkeleton_SpecRoundTrips.
func renderTemplate(t *testing.T, gen *engine.ChainsawTestGenerator) string {
	t.Helper()
	if err := gen.SetTemplateDefaults(); err != nil {
		t.Fatalf("SetTemplateDefaults: %v", err)
	}
	tmpl, err := template.New("t").Parse(gen.GetBody())
	if err != nil {
		t.Fatalf("parse body: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, gen); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}
