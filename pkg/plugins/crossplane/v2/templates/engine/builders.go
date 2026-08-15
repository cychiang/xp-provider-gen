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

// BaseTemplateBuilder builds one discovered template. It keeps the TemplateInfo
// found during discovery: Build needs the template's path, and re-deriving it
// would mean walking the embedded FS again for every template built.
type BaseTemplateBuilder struct {
	templateType TemplateType
	info         TemplateInfo
}

// NewBaseTemplateBuilder creates a builder for an already-discovered template.
func NewBaseTemplateBuilder(templateType TemplateType, info TemplateInfo) TemplateBuilder {
	return &BaseTemplateBuilder{templateType: templateType, info: info}
}

func (b *BaseTemplateBuilder) GetTemplateType() TemplateType {
	return b.templateType
}

func (b *BaseTemplateBuilder) Build(cfg config.Config, opts ...Option) (TemplateProduct, error) {
	options := parseOptions(opts)

	// Per-kind templates cannot render without the kind.
	if b.info.Category == APICategory && options.Resource == nil {
		return nil, fmt.Errorf("resource is required for API template %s", b.info.Path)
	}

	info := b.info
	replacements := replacementsFor(cfg, options)

	// Create and configure template product
	product := createTemplateProduct(b.templateType, info, replacements)

	if err := configureTemplateProduct(product, cfg, options); err != nil {
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
