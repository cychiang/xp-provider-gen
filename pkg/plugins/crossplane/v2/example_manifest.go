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
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/yaml"
)

// specIndent is how many spaces create-test's chainsaw skeleton
// (generators/chainsaw_test.yaml.tmpl) indents the resource spec it splices
// in under the apply step's `spec:` key.
const specIndent = 16

// exampleManifest is the shape create-test reads out of
// examples/<group>/<kind>.yaml — the one file both flavors already treat as
// "a valid manifest for this kind" (native seeds it at `create api`; an
// upjet author writes it after `make generate`). Only the three fields a
// chainsaw step needs are decoded; everything else in the file is ignored.
// apiVersion/metadata/spec are the Kubernetes API's own field names, fixed
// by that spec, not by this tool's snake_case convention for its own config.
//
//nolint:tagliatelle // Kubernetes API field names, not ours to rename
type exampleManifest struct {
	APIVersion string `json:"apiVersion"`
	Metadata   struct {
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec map[string]interface{} `json:"spec"`
}

// examplePath returns where create-test looks for a kind's example
// manifest — the same path `create api` seeds it at (native) or tells the
// author to write it at (upjet): examples/<group>/<kind>.yaml, both
// lowercased.
func examplePath(res resource.Resource) string {
	return filepath.Join("examples", strings.ToLower(res.Group), strings.ToLower(res.Kind)+".yaml")
}

// generatedExamplePath returns where upjet's own `make generate` writes a
// kind's scraped example, for the hint that tells an upjet author where to
// copy examplePath's target from. upjet writes these file names lowercased
// (verified against a real generated scaffold), so — unlike examplePath,
// which is this tool's own convention — this is not a free choice: getting
// the case wrong here produces a path that does not exist on a
// case-sensitive filesystem. Used from both the create-test refusal message
// and create api's upjet next-steps text, so the format lives in one place.
func generatedExamplePath(res resource.Resource) string {
	return fmt.Sprintf("examples-generated/namespaced/%s/%s/%s.yaml",
		strings.ToLower(res.Group), res.Version, strings.ToLower(res.Kind))
}

// loadExampleManifest reads examples/<group>/<kind>.yaml and extracts the
// apiVersion, namespace and spec (pre-indented, ready to splice under the
// chainsaw skeleton's spec: key) create-test needs. This is the one thing
// that makes create-test work identically for both flavors: whichever
// flavor produced the example, the shape read here is the same, so there is
// no flavor branch in create-test at all.
func loadExampleManifest(fs afero.Fs, res resource.Resource) (string, string, string, error) {
	path := examplePath(res)
	raw, err := afero.ReadFile(fs, path)
	if err != nil {
		return "", "", "", fmt.Errorf(
			"no example manifest at %s: create-test derives the test from it, so it must exist first\n"+
				"  native: 'create api' already seeded one there — fill in its spec\n"+
				"  upjet: run 'make generate', then copy %s "+
				"there and fix any Terraform interpolations (${...})",
			path, generatedExamplePath(res))
	}

	var manifest exampleManifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return "", "", "", fmt.Errorf("parsing %s: %w", path, err)
	}
	if manifest.APIVersion == "" {
		return "", "", "", fmt.Errorf("%s has no apiVersion", path)
	}
	if manifest.Metadata.Namespace == "" {
		return "", "", "", fmt.Errorf("%s has no metadata.namespace", path)
	}
	if len(manifest.Spec) == 0 {
		return "", "", "", fmt.Errorf("%s has no spec — create-test needs a real manifest, not a skeleton", path)
	}

	specBytes, err := yaml.Marshal(manifest.Spec)
	if err != nil {
		return "", "", "", fmt.Errorf("re-marshaling spec from %s: %w", path, err)
	}
	return manifest.APIVersion, manifest.Metadata.Namespace, indentYAML(specBytes, specIndent), nil
}

// indentYAML prefixes every non-empty line of a rendered YAML block with n
// spaces, so it can be spliced under a fixed key in the chainsaw skeleton.
func indentYAML(raw []byte, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
