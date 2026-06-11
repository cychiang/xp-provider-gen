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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/versions"
)

func TestGoModGenerator_RendersManifest(t *testing.T) {
	deps := []versions.Dependency{
		{Module: "github.com/crossplane/crossplane-runtime/v2", Version: "v2.0.0"},
		{Module: "k8s.io/apimachinery", Version: "v0.33.3"},
	}
	g := NewGoModGenerator(testRepo, deps)

	out := render(t, g)
	for _, want := range []string{
		"module " + testRepo,
		"go " + versions.GoVersion,
		"tool sigs.k8s.io/controller-tools/cmd/controller-gen",
		"github.com/crossplane/crossplane-runtime/v2 v2.0.0",
		"k8s.io/apimachinery v0.33.3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("go.mod output missing %q\n%s", want, out)
		}
	}
	// go.mod is user-owned: it must NOT carry the generated header.
	if strings.Contains(out, "DO NOT EDIT") {
		t.Error("go.mod must not carry the tool-owned header (it is seed-once/user-owned)")
	}
}

func TestGoModGenerator_SeedsOnce(t *testing.T) {
	g := NewGoModGenerator(testRepo, nil)
	if err := g.SetTemplateDefaults(); err != nil {
		t.Fatal(err)
	}
	if g.Path != "go.mod" {
		t.Errorf("Path = %q, want go.mod", g.Path)
	}
	if g.IfExistsAction != machinery.SkipFile {
		t.Errorf("IfExistsAction = %v, want SkipFile (seed-once)", g.IfExistsAction)
	}
}
