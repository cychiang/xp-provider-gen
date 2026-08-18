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

import "strings"

// Default Terraform coordinates for an upjet provider. Only the provider
// itself has no default — it is what makes the provider specific.
const (
	DefaultTerraformVersion   = "1.5.7"
	DefaultTerraformDocsPath  = "docs/resources"
	defaultProviderRepoPrefix = "https://github.com/hashicorp/terraform-provider-"
)

// UpjetSettings are the Terraform coordinates an upjet provider is generated
// from. They are recorded in PROJECT at init and rendered into the generated
// Makefile and provider config.
type UpjetSettings struct {
	// TerraformProvider is the Terraform registry source, e.g.
	// "hashicorp/kubernetes".
	TerraformProvider string
	// TerraformProviderName is the source's name half, e.g. "kubernetes".
	TerraformProviderName string
	// TerraformProviderVersion is the provider version, e.g. "2.38.0".
	TerraformProviderVersion string
	// TerraformProviderRepo is the git repository its docs are scraped from.
	TerraformProviderRepo string
	// TerraformDocsPath is where resource docs live in that repository.
	TerraformDocsPath string
	// TerraformVersion is the Terraform CLI version used to read the schema.
	TerraformVersion string
	// TerraformResourcePrefix is the resource name prefix, e.g. "kubernetes"
	// for kubernetes_secret.
	TerraformResourcePrefix string
	// NamespacedDomain is the API group upjet uses for namespaced resources.
	NamespacedDomain string
	// TerraformResource is the Terraform resource a single kind maps to. Only
	// set when rendering per-resource templates.
	TerraformResource string
}

// ProviderNameFromSource returns the name half of "org/name".
func ProviderNameFromSource(source string) string {
	if _, name, ok := strings.Cut(source, "/"); ok {
		return name
	}
	return source
}

// DefaultProviderRepo guesses the git repository holding a Terraform
// provider's docs from its registry source.
func DefaultProviderRepo(source string) string {
	return defaultProviderRepoPrefix + ProviderNameFromSource(source)
}

// NamespacedDomain inserts the "m" label upjet uses for namespaced API groups:
// example.com becomes example.m.com.
func NamespacedDomain(domain string) string {
	first, rest, ok := strings.Cut(domain, ".")
	if !ok {
		return domain
	}
	return first + ".m." + rest
}
