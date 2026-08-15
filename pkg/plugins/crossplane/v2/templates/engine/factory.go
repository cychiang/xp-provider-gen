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

type CrossplaneTemplateFactory struct {
	config       config.Config
	initRegistry map[TemplateType]TemplateBuilder
	apiRegistry  map[TemplateType]TemplateBuilder
}

func NewFactory(cfg config.Config) TemplateFactory {
	factory := &CrossplaneTemplateFactory{
		config:       cfg,
		initRegistry: make(map[TemplateType]TemplateBuilder),
		apiRegistry:  make(map[TemplateType]TemplateBuilder),
	}

	factory.discoverAndRegisterTemplates()
	return factory
}

func (f *CrossplaneTemplateFactory) discoverAndRegisterTemplates() {
	processor := core.NewTemplatePathProcessor()

	err := fs.WalkDir(templates.TemplateFS, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !processor.IsTemplateFile(path) {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		templateType := info.GenerateTemplateType()

		switch info.Category {
		case InitCategory:
			f.initRegistry[templateType] = NewBaseTemplateBuilder(templateType, info)
		case APICategory:
			f.apiRegistry[templateType] = NewBaseTemplateBuilder(templateType, info)
		default:
			return fmt.Errorf("template %q has no category", path)
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
	var templates []TemplateProduct

	for templateType, builder := range f.initRegistry {
		product, err := builder.Build(f.config, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to build init template %s: %w", templateType, err)
		}
		templates = append(templates, product)
	}

	return templates, nil
}

func (f *CrossplaneTemplateFactory) GetAPITemplates(opts ...Option) ([]TemplateProduct, error) {
	var templates []TemplateProduct

	for templateType, builder := range f.apiRegistry {
		product, err := builder.Build(f.config, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to build API template %s: %w", templateType, err)
		}
		templates = append(templates, product)
	}

	return templates, nil
}
