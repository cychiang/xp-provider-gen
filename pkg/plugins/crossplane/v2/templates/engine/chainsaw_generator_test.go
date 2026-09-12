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
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// TestChainsawTestGenerator_RendersFromExample pins B3's fix end to end at
// the template layer: the rendered chainsaw-test.yaml must carry whatever
// apiVersion/namespace/spec create-test read out of the project's example
// manifest, not a hard-coded native-shaped body. One rendering path serves
// both flavors — the test data below is deliberately upjet-shaped
// (crossplane-system namespace, a "data" field, no "configurableField") to
// prove nothing here assumes native.
func TestChainsawTestGenerator_RendersFromExample(t *testing.T) {
	res := resource.Resource{GVK: resource.GVK{Group: "core", Version: "v1beta1", Kind: "Secret"}}
	spec := "                forProvider:\n" +
		"                  data:\n" +
		"                    key: dmFsdWU=\n" +
		"                providerConfigRef:\n" +
		"                  kind: ProviderConfig\n" +
		"                  name: default"

	g := NewChainsawTestGenerator("drift-check", res, "core.example.m.com/v1beta1", "crossplane-system", spec)
	out := render(t, g)

	for _, want := range []string{
		"name: drift-check",
		"apiVersion: core.example.m.com/v1beta1",
		"kind: Secret",
		"name: behavior-drift-check",
		"namespace: crossplane-system",
		"data:",
		"key: dmFsdWU=",
		"providerConfigRef:",
		"name: default",
		"((conditions[?type == 'Ready'])[0]):",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing %q:\n%s", want, out)
		}
	}
	// The old hard-coded native-only literals must be gone.
	for _, unwanted := range []string{"configurableField", "namespace: default", "QualifiedGroup"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("rendered output still contains hard-coded %q:\n%s", unwanted, out)
		}
	}
}

func TestChainsawTestGenerator_SetTemplateDefaults(t *testing.T) {
	res := resource.Resource{GVK: resource.GVK{Group: "core", Version: "v1beta1", Kind: "Secret"}}
	g := NewChainsawTestGenerator("drift-check", res, "core.example.com/v1beta1", "default", "                forProvider: {}")
	if err := g.SetTemplateDefaults(); err != nil {
		t.Fatalf("SetTemplateDefaults: %v", err)
	}
	if want := "test/behavior/drift-check/chainsaw-test.yaml"; g.Path != want {
		t.Errorf("Path = %q, want %q", g.Path, want)
	}
}
