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
	"slices"
	"sort"

	"sigs.k8s.io/yaml"
)

//go:embed dependencies.yaml
var dependenciesYAML []byte

// embedded is dependencies.yaml, parsed once at package init. It is embedded
// at compile time and repo-controlled (not user input), so a parse failure
// here is a build defect, not a runtime data problem — mustParse panics
// rather than threading an error through every reader below, the same
// reasoning the template engine uses for its own embedded-FS failures (see
// e.g. templates/engine/factory.go).
var embedded = mustParse(dependenciesYAML)

func mustParse(raw []byte) manifest {
	m, err := parseManifest(raw)
	if err != nil {
		panic(fmt.Errorf("parsing embedded dependencies manifest: %w", err))
	}
	return m
}

// Dependency is one direct module requirement in a generated provider's go.mod.
type Dependency struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

// manifest is the shape of dependencies.yaml: the single source of truth for
// the framework/Kubernetes dependency versions, the Go version, and the
// Terraform CLI version a generated provider targets.
type manifest struct {
	GoVersion         string       `json:"go_version"`
	TerraformVersion  string       `json:"terraform_version"`
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
// go.mod `go` directive), read from the embedded manifest's go_version key —
// the single place this number is set. This repo's own go.mod `go` directive
// and the Dockerfile's golang base image tag both have to match it literally
// (neither can be computed), which scripts/check-go-version enforces in CI
// instead of re-deriving them.
var GoVersion = requireField(embedded.GoVersion, "go_version")

// TerraformVersion is the Terraform CLI version an upjet-flavored provider
// uses to read its wrapped provider's schema, read from the embedded
// manifest's terraform_version key. This is a tool decision, not the
// author's: the whole upjet ecosystem is pinned below Terraform 1.6 because
// that version is BSL-licensed, a licensing constraint rather than a
// compatibility one, but not the author's call either way — there is
// deliberately no CLI flag for it.
var TerraformVersion = requireField(embedded.TerraformVersion, "terraform_version")

// requireField panics if a required manifest key is missing — a build
// defect, since the manifest is repo-controlled, not runtime input.
func requireField(value, key string) string {
	if value == "" {
		panic("pkg/versions/dependencies.yaml: " + key + " is required")
	}
	return value
}

// GoModDependencies returns the direct dependencies a generated provider's
// go.mod should declare. Cloned so a caller (or UpjetGoModDependencies,
// below) can never mutate the package-level embedded.Dependencies backing
// array out from under the other.
func GoModDependencies() []Dependency {
	return slices.Clone(embedded.Dependencies)
}

// UpjetGoModDependencies returns the dependencies an upjet-flavored provider
// declares: the shared set plus upjet's own, sorted so go.mod renders stably.
// slices.Concat always allocates a new backing array — unlike append, which
// only allocates when the first slice's capacity is exhausted, silently
// aliasing (and corrupting, once sorted) embedded.Dependencies otherwise.
func UpjetGoModDependencies() []Dependency {
	deps := slices.Concat(embedded.Dependencies, embedded.UpjetDependencies)
	sort.Slice(deps, func(i, j int) bool { return deps[i].Module < deps[j].Module })
	return deps
}
