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
	"slices"
	"testing"

	"github.com/cychiang/xp-provider-gen/pkg/versions"
)

func TestDependenciesToGet(t *testing.T) {
	const pinned = "v0.9.9"
	goMod := []byte(`module example.com/p

go 1.26

require (
	k8s.io/apiextensions-apiserver v0.9.9
	k8s.io/client-go v0.9.0
	github.com/google/go-cmp v0.7.0 // indirect
)
`)
	deps := []versions.Dependency{
		{Module: "k8s.io/apiextensions-apiserver", Version: pinned}, // current: skipped
		{Module: "k8s.io/client-go", Version: pinned},               // behind: kept
		{Module: "github.com/google/go-cmp", Version: "v0.7.0"},     // current, indirect: skipped
		{Module: "k8s.io/apimachinery", Version: pinned},            // absent: kept
	}
	got, err := dependenciesToGet(goMod, deps)
	if err != nil {
		t.Fatal(err)
	}
	if want := []versions.Dependency{deps[1], deps[3]}; !slices.Equal(got, want) {
		t.Errorf("dependenciesToGet = %v, want %v", got, want)
	}
}
