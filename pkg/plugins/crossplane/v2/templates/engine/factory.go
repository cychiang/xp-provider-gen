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
	"io/fs"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

// CrossplaneTemplateFactory holds the templates discovered in the embedded FS,
// split by when they render. The two lists are only ever iterated, so they are
// slices: nothing looks a template up by name.
type CrossplaneTemplateFactory struct {
	config        config.Config
	root          string
	initTemplates []TemplateInfo
	apiTemplates  []TemplateInfo
}

// NewFactory returns a factory over the native flavor's templates.
func NewFactory(cfg config.Config) TemplateFactory {
	return NewFactoryForFlavor(cfg, core.FlavorNative)
}

// NewFactoryForFlavor returns a factory over the given flavor's template tree.
func NewFactoryForFlavor(cfg config.Config, flavor core.Flavor) TemplateFactory {
	factory := &CrossplaneTemplateFactory{config: cfg, root: flavor.TemplateRoot()}
	factory.discoverTemplates()
	return factory
}

func (f *CrossplaneTemplateFactory) discoverTemplates() {
	err := fs.WalkDir(templates.TemplateFS, f.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !core.IsTemplateFile(path) {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		if info.Category == APICategory {
			f.apiTemplates = append(f.apiTemplates, info)
		} else {
			f.initTemplates = append(f.initTemplates, info)
		}
		return nil
	})
	if err != nil {
		// The template FS is embedded at compile time, so a discovery failure
		// is a build defect — fail loudly rather than scaffold incompletely.
		panic(fmt.Errorf("discovering templates: %w", err))
	}
}

func (f *CrossplaneTemplateFactory) GetInitTemplates(opts ...Option) ([]TemplateProduct, error) {
	return f.build(f.initTemplates, opts)
}

func (f *CrossplaneTemplateFactory) GetAPITemplates(opts ...Option) ([]TemplateProduct, error) {
	return f.build(f.apiTemplates, opts)
}

// build renders each discovered template into a product.
func (f *CrossplaneTemplateFactory) build(infos []TemplateInfo, opts []Option) ([]TemplateProduct, error) {
	products := make([]TemplateProduct, 0, len(infos))
	for _, info := range infos {
		product, err := BuildTemplate(f.config, info, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to build template %s: %w", info.Path, err)
		}
		products = append(products, product)
	}
	return products, nil
}
