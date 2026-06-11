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

// Package versions is the single source of truth for the dependency versions a
// generated provider targets. The set is rendered into the provider's go.mod and
// applied to existing providers by `xp-provider-gen update`.
package versions

import (
	_ "embed"
	"fmt"

	"sigs.k8s.io/yaml"
)

// GoVersion is the Go language version generated providers target (the go.mod
// `go` directive). Bumping it is a deliberate toolchain decision.
const GoVersion = "1.26.0"

//go:embed dependencies.yaml
var dependenciesYAML []byte

// Dependency is one direct module requirement in a generated provider's go.mod.
type Dependency struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

type manifest struct {
	Dependencies []Dependency `json:"dependencies"`
}

// GoModDependencies returns the direct dependencies a generated provider's
// go.mod should declare, parsed from the embedded manifest.
func GoModDependencies() ([]Dependency, error) {
	var m manifest
	if err := yaml.Unmarshal(dependenciesYAML, &m); err != nil {
		return nil, fmt.Errorf("parse dependencies manifest: %w", err)
	}
	return m.Dependencies, nil
}
