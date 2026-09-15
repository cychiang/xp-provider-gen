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

package core

import (
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// testProviderName is the Terraform provider name half reused across this
// file's fully-populated UpjetSettings fixtures.
const testProviderName = "kubernetes"

// TestUpjetSettings_PersistsOnlyResourcePrefix pins the ownership decision:
// of UpjetSettings' fields, only TerraformResourcePrefix is read back after
// init (createapi.go, to validate --terraform-resource on every later
// `create api` call) and so is the only one PROJECT should carry. Every
// other field here is either a render-time-only input (baked into the
// generated Makefile at init and never re-read), or moved to the tool
// (TerraformVersion, now sourced from pkg/versions). A fully populated
// struct must still marshal down to just the one field: this fails the
// moment someone adds a json tag back to a field nothing reads, or forgets
// json:"-" on a new field that shouldn't persist.
func TestUpjetSettings_PersistsOnlyResourcePrefix(t *testing.T) {
	full := UpjetSettings{
		TerraformProvider:        "hashicorp/kubernetes",
		TerraformProviderName:    testProviderName,
		TerraformProviderVersion: "2.38.0",
		TerraformProviderRepo:    "https://github.com/hashicorp/terraform-provider-kubernetes",
		TerraformDocsPath:        "docs/resources",
		TerraformVersion:         "1.5.7",
		TerraformResourcePrefix:  testProviderName,
		NamespacedDomain:         "example.m.com",
		TerraformResource:        "kubernetes_secret",
	}

	out, err := yaml.Marshal(full)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got := strings.TrimSpace(string(out))
	want := "terraform_resource_prefix: kubernetes"
	if got != want {
		t.Errorf("UpjetSettings marshaled to %q, want exactly %q (only terraform_resource_prefix persists)", got, want)
	}
}

// TestUpjetSettings_RenderFieldsSurviveGoLevelUse guards the other half of
// the contract: dropping the json tag must not have dropped the field. Every
// value below still has to be an ordinary readable/settable Go field, since
// scaffold rendering (WithUpjet) populates and reads all of them at
// `init`/`create api` time — only PROJECT persistence was cut, not the
// render-time data flow.
func TestUpjetSettings_RenderFieldsSurviveGoLevelUse(t *testing.T) {
	s := UpjetSettings{
		TerraformProvider:        "hashicorp/kubernetes",
		TerraformProviderName:    testProviderName,
		TerraformProviderVersion: "2.38.0",
		TerraformProviderRepo:    "https://example.com/repo",
		TerraformDocsPath:        "docs/resources",
		TerraformVersion:         "1.5.7",
		TerraformResourcePrefix:  testProviderName,
		NamespacedDomain:         "example.m.com",
		TerraformResource:        "kubernetes_secret",
	}
	for name, got := range map[string]string{
		"TerraformProvider":        s.TerraformProvider,
		"TerraformProviderName":    s.TerraformProviderName,
		"TerraformProviderVersion": s.TerraformProviderVersion,
		"TerraformProviderRepo":    s.TerraformProviderRepo,
		"TerraformDocsPath":        s.TerraformDocsPath,
		"TerraformVersion":         s.TerraformVersion,
		"TerraformResourcePrefix":  s.TerraformResourcePrefix,
		"NamespacedDomain":         s.NamespacedDomain,
		"TerraformResource":        s.TerraformResource,
	} {
		if got == "" {
			t.Errorf("%s is empty; render-time fields must remain ordinary Go fields even when excluded from persistence", name)
		}
	}
}
