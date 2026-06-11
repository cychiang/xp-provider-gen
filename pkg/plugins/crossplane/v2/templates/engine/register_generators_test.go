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
	"bytes"
	"strings"
	"testing"
	"text/template"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

const testRepo = "github.com/example/provider-test"

// twoKindsSameGroupVersion mirrors the e2e: two kinds sharing sample/v1.
func twoKindsSameGroupVersion() []resource.Resource {
	return []resource.Resource{
		{GVK: resource.GVK{Group: "sample", Version: "v1", Kind: "MyType"}},
		{GVK: resource.GVK{Group: "sample", Version: "v1", Kind: "MyValue"}},
	}
}

func render(t *testing.T, b machinery.Template) string {
	t.Helper()
	if err := b.SetTemplateDefaults(); err != nil {
		t.Fatalf("SetTemplateDefaults: %v", err)
	}
	tmpl, err := template.New("t").Parse(b.GetBody())
	if err != nil {
		t.Fatalf("parse body: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, b); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return buf.String()
}

func TestAPIRegisterGenerator_DedupsByGroupVersion(t *testing.T) {
	g := NewAPIRegisterGenerator(testRepo, "provider-test", twoKindsSameGroupVersion())

	// Base providerv1alpha1 + exactly one sample/v1 (two kinds collapse to one GV).
	if len(g.Groups) != 2 {
		t.Fatalf("Groups = %d, want 2 (base + sample/v1)", len(g.Groups))
	}
	if g.Groups[0].Alias != "providerv1alpha1" {
		t.Errorf("Groups[0].Alias = %q, want providerv1alpha1", g.Groups[0].Alias)
	}
	if g.Groups[1].Alias != "samplev1" || g.Groups[1].Path != testRepo+"/apis/sample/v1" {
		t.Errorf("Groups[1] = %+v, want samplev1 -> %s/apis/sample/v1", g.Groups[1], testRepo)
	}

	out := render(t, g)
	if n := strings.Count(out, `samplev1 "`); n != 1 {
		t.Errorf("samplev1 import count = %d, want 1\n%s", n, out)
	}
	if n := strings.Count(out, "samplev1.SchemeBuilder.AddToScheme"); n != 1 {
		t.Errorf("samplev1 registration count = %d, want 1", n)
	}
	if !strings.Contains(out, `providerv1alpha1 "`+testRepo+`/apis/v1alpha1"`) {
		t.Errorf("missing base providerv1alpha1 import\n%s", out)
	}
}

func TestControllerRegisterGenerator_PerKind(t *testing.T) {
	g := NewControllerRegisterGenerator(testRepo, "provider-test", twoKindsSameGroupVersion())

	// Base config + one controller per kind.
	if len(g.Controllers) != 3 {
		t.Fatalf("Controllers = %d, want 3 (config + mytype + myvalue)", len(g.Controllers))
	}

	out := render(t, g)
	for _, want := range []string{
		"config.Setup,",
		"mytype.SetupGated,",
		"myvalue.SetupGated,",
		`"` + testRepo + `/internal/controller/mytype"`,
		`"` + testRepo + `/internal/controller/myvalue"`,
		`"` + testRepo + `/internal/controller/config"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}
