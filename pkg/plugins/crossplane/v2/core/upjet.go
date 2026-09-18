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
// itself has no default — it is what makes the provider specific. There is
// no default Terraform CLI version here: that is a tool decision, sourced
// from pkg/versions.TerraformVersion, not an author-configurable default.
const (
	DefaultTerraformDocsPath  = "docs/resources"
	defaultProviderRepoPrefix = "https://github.com/hashicorp/terraform-provider-"
)

// UpjetSettings are the Terraform coordinates an upjet provider is generated
// from — the full set a render pass needs, whether or not each field is
// worth persisting in PROJECT afterward. Rendered into the generated
// Makefile and provider config at `init` (and, for TerraformResourcePrefix,
// read again by `create api` on every later call).
//
// Only TerraformResourcePrefix carries a json tag: it is the one field
// anything reads back out of PROJECT after init (createapi.go, to validate
// --terraform-resource). The rest are render-time-only — baked as literals
// into the generated Makefile once at init and never re-read from PROJECT by
// this tool again — so persisting them would just be a second, driftable
// copy of a value that already lives durably in the Makefile the moment
// `init` finishes. json:"-" keeps each one an ordinary Go field (still set
// and read during rendering) while keeping it out of the PROJECT file.
type UpjetSettings struct {
	// TerraformProvider is the Terraform registry source, e.g.
	// "hashicorp/kubernetes". Render-time only: baked into the Makefile's
	// TERRAFORM_PROVIDER_SOURCE at init.
	TerraformProvider string `json:"-"`
	// TerraformProviderName is the source's name half, e.g. "kubernetes".
	// Render-time only: baked into TERRAFORM_PROVIDER_DOWNLOAD_NAME and
	// TERRAFORM_NATIVE_PROVIDER_BINARY at init.
	TerraformProviderName string `json:"-"`
	// TerraformProviderVersion is the provider version, e.g. "2.38.0".
	// Render-time only: baked into TERRAFORM_PROVIDER_VERSION and
	// TERRAFORM_NATIVE_PROVIDER_BINARY at init.
	TerraformProviderVersion string `json:"-"`
	// TerraformProviderRepo is the git repository its docs are scraped from.
	// Render-time only: baked into TERRAFORM_PROVIDER_REPO at init.
	TerraformProviderRepo string `json:"-"`
	// TerraformDocsPath is where resource docs live in that repository.
	// Render-time only: baked into TERRAFORM_DOCS_PATH at init.
	TerraformDocsPath string `json:"-"`
	// TerraformVersion is the Terraform CLI version used to read the schema,
	// sourced from pkg/versions.TerraformVersion (the tool's call, not the
	// author's — see that package for why). Callers leave it empty: every
	// render fills it in (engine.BaseTemplateProduct.Configure) and writes it
	// to TERRAFORM_VERSION in the tool-owned hack/xp-provider-gen.mk, so
	// `update` keeps it current.
	TerraformVersion string `json:"-"`
	// TerraformResourcePrefix is the resource name prefix, e.g. "kubernetes"
	// for kubernetes_secret. The one field `create api` reads back out of
	// PROJECT on every later call, to validate --terraform-resource.
	TerraformResourcePrefix string `json:"terraform_resource_prefix,omitempty"`
	// NamespacedDomain is the API group upjet uses for namespaced resources.
	// Render-time only, and always re-derivable from Domain (persisted
	// separately in PROJECT) via NamespacedDomain(domain) below —
	// BaseTemplateProduct.Configure already recomputes it whenever it's
	// empty, so there is nothing here that needs a second, stale copy in
	// PROJECT.
	NamespacedDomain string `json:"-"`
	// TerraformResource is the Terraform resource a single kind maps to. Only
	// set when rendering per-resource templates — excluded from persistence:
	// it is per-kind and PROJECT does not track per-kind coordinates, so a
	// value here is always transient and must never round-trip through PROJECT.
	TerraformResource string `json:"-"`
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
