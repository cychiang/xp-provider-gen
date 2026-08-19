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
	"fmt"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// Path placeholder tokens: uppercase segments in template paths replaced at
// render time. GROUP/VERSION/KIND mark per-kind templates; IMAGENAME marks the
// provider name (init-phase).
const (
	placeholderGroup     = "GROUP"
	placeholderVersion   = "VERSION"
	placeholderKind      = "KIND"
	placeholderImageName = "IMAGENAME"
)

// BuildTemplate turns one discovered template into a renderable product: it
// resolves the output path's placeholders and loads the template body.
func BuildTemplate(cfg config.Config, info TemplateInfo, opts ...Option) (TemplateProduct, error) {
	options := &TemplateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Per-kind templates cannot render without the kind.
	if info.Category == APICategory && options.Resource == nil {
		return nil, fmt.Errorf("resource is required for API template %s", info.Path)
	}

	product := NewGenericTemplateProduct(
		core.GenerateOutputPath(info.Path, replacementsFor(cfg, options)),
		info.Path,
	)
	if err := configureProduct(product, cfg, options); err != nil {
		return nil, err
	}
	return product, nil
}

// replacementsFor returns the path placeholder substitutions for one template.
// The resource placeholders appear only when a resource is in play — which is
// exactly what separates a per-kind render from an init-time one.
func replacementsFor(cfg config.Config, options *TemplateOptions) map[string]string {
	replacements := map[string]string{
		placeholderImageName: core.ExtractProjectName(cfg),
	}
	if options.Resource != nil {
		replacements[placeholderGroup] = strings.ToLower(options.Resource.Group)
		replacements[placeholderVersion] = options.Resource.Version
		replacements[placeholderKind] = strings.ToLower(options.Resource.Kind)
	}
	return replacements
}

// configureProduct applies the project config, resource and force flag, then
// loads the template body.
func configureProduct(product *GenericTemplateProduct, cfg config.Config, options *TemplateOptions) error {
	if err := product.Configure(cfg); err != nil {
		return fmt.Errorf("failed to configure template: %w", err)
	}
	if options.Upjet != nil {
		product.UpjetSettings = *options.Upjet
	}
	if options.Resource != nil {
		if err := product.SetResource(options.Resource); err != nil {
			return fmt.Errorf("failed to set resource: %w", err)
		}
	}
	if options.Force {
		// Without --force the zero value (machinery.SkipFile) applies, which is
		// what a second `create api` in an existing group/version needs: the
		// group-scoped templates (groupversion_info.go) are already on disk and
		// must be left alone rather than erroring the whole command.
		product.SetForce(true)
	}
	if err := product.SetTemplateDefaults(); err != nil {
		return fmt.Errorf("failed to set template defaults: %w", err)
	}
	return nil
}
