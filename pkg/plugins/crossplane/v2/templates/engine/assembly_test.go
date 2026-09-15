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
	"slices"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// TestCoreGeneratorsFor pins that the one flavor switch hands each flavor its
// own generator set: upjet has no native register.go, native has no upjet
// resource aggregator.
func TestCoreGeneratorsFor(t *testing.T) {
	const (
		apisRegister   = "apis/register.go"
		upjetResources = "config/zz_resources.go"
	)
	tests := []struct {
		flavor      core.Flavor
		wantPath    string
		notWantPath string
	}{
		{flavor: core.FlavorNative, wantPath: apisRegister, notWantPath: upjetResources},
		{flavor: core.FlavorUpjet, wantPath: upjetResources, notWantPath: apisRegister},
	}
	for _, tt := range tests {
		t.Run(string(tt.flavor), func(t *testing.T) {
			var paths []string
			for _, b := range CoreGeneratorsFor(tt.flavor, newTestConfig(t), nil) {
				tmpl, ok := b.(machinery.Template)
				if !ok {
					t.Fatalf("builder %T is not a machinery.Template", b)
				}
				if err := tmpl.SetTemplateDefaults(); err != nil {
					t.Fatalf("SetTemplateDefaults on %T: %v", b, err)
				}
				paths = append(paths, tmpl.GetPath())
			}
			if !slices.Contains(paths, tt.wantPath) {
				t.Errorf("generator paths = %v, want %q among them", paths, tt.wantPath)
			}
			if slices.Contains(paths, tt.notWantPath) {
				t.Errorf("generator paths = %v, must not contain %q", paths, tt.notWantPath)
			}
		})
	}
}
