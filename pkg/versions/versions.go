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
	"sort"

	"sigs.k8s.io/yaml"
)

//go:embed dependencies.yaml
var dependenciesYAML []byte

// Dependency is one direct module requirement in a generated provider's go.mod.
type Dependency struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

// manifest is the shape of dependencies.yaml: the single source of truth for
// both the framework/Kubernetes dependency versions and the Go version a
// generated provider targets.
type manifest struct {
	GoVersion         string       `json:"go_version"`
	Dependencies      []Dependency `json:"dependencies"`
	UpjetDependencies []Dependency `json:"upjet_dependencies"`
}

// parseManifest decodes raw dependencies.yaml content. Taking raw bytes
// (rather than reading the embedded var directly) keeps this testable
// against arbitrary YAML, independent of the real embedded file.
func parseManifest(raw []byte) (manifest, error) {
	var m manifest
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return manifest{}, fmt.Errorf("parse dependencies manifest: %w", err)
	}
	return m, nil
}

// GoVersion is the Go language version generated providers target (the
// go.mod `go` directive), parsed from the embedded manifest's go_version key
// — the single place this number is set. This repo's own go.mod `go`
// directive and the Dockerfile's golang base image tag both have to match it
// literally (neither can be computed), which scripts/check-go-version
// enforces in CI instead of re-deriving them.
var GoVersion = mustGoVersion()

// mustGoVersion parses GoVersion out of the embedded manifest. The manifest
// is embedded at compile time and repo-controlled (not user input), so a
// parse failure or a missing key is a build defect, not a runtime data
// problem — this panics rather than threading an error through every one of
// GoVersion's callers, the same reasoning the template engine uses for its
// own embedded-FS failures (see e.g. templates/engine/factory.go).
func mustGoVersion() string {
	m, err := parseManifest(dependenciesYAML)
	if err != nil {
		panic(fmt.Errorf("parsing embedded dependencies manifest: %w", err))
	}
	if m.GoVersion == "" {
		panic("pkg/versions/dependencies.yaml: go_version is required")
	}
	return m.GoVersion
}

// GoModDependencies returns the direct dependencies a generated provider's
// go.mod should declare, parsed from the embedded manifest.
func GoModDependencies() ([]Dependency, error) {
	m, err := parseManifest(dependenciesYAML)
	if err != nil {
		return nil, err
	}
	return m.Dependencies, nil
}

// UpjetGoModDependencies returns the dependencies an upjet-flavored provider
// declares: the shared set plus upjet's own, sorted so go.mod renders stably.
func UpjetGoModDependencies() ([]Dependency, error) {
	m, err := parseManifest(dependenciesYAML)
	if err != nil {
		return nil, err
	}
	deps := append(m.Dependencies, m.UpjetDependencies...) //nolint:gocritic // deliberate copy
	sort.Slice(deps, func(i, j int) bool { return deps[i].Module < deps[j].Module })
	return deps, nil
}
