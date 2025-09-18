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

package templates

import (
	"fmt"
	"io/fs"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
)

type CrossplaneTemplateFactory struct {
	config         config.Config
	initRegistry   map[TemplateType]TemplateBuilder
	apiRegistry    map[TemplateType]TemplateBuilder
	staticRegistry map[TemplateType]TemplateBuilder
}

func NewFactory(cfg config.Config) TemplateFactory {
	factory := &CrossplaneTemplateFactory{
		config:         cfg,
		initRegistry:   make(map[TemplateType]TemplateBuilder),
		apiRegistry:    make(map[TemplateType]TemplateBuilder),
		staticRegistry: make(map[TemplateType]TemplateBuilder),
	}

	factory.discoverAndRegisterTemplates()
	return factory
}

func (f *CrossplaneTemplateFactory) discoverAndRegisterTemplates() {
	fs.WalkDir(templateFS, "scaffolds", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		templateType := TemplateType(info.GenerateTemplateType())

		switch info.Category {
		case "init":
			f.initRegistry[templateType] = NewInitTemplateBuilder(templateType)
		case "api":
			f.apiRegistry[templateType] = NewAPITemplateBuilder(templateType)
		case "static":
			f.staticRegistry[templateType] = NewStaticTemplateBuilder(templateType)
		}

		return nil
	})
}
func (f *CrossplaneTemplateFactory) CreateInitTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.initRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported init template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build init template %s: %w", templateType, err)
	}

	return product, nil
}

func (f *CrossplaneTemplateFactory) CreateAPITemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.apiRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported API template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build API template %s: %w", templateType, err)
	}

	return product, nil
}

func (f *CrossplaneTemplateFactory) CreateStaticTemplate(templateType TemplateType, opts ...Option) (TemplateProduct, error) {
	builder, exists := f.staticRegistry[templateType]
	if !exists {
		return nil, fmt.Errorf("unsupported static template type: %s", templateType)
	}

	product, err := builder.Build(f.config, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to build static template %s: %w", templateType, err)
	}

	return product, nil
}

func (f *CrossplaneTemplateFactory) GetSupportedTypes() []TemplateType {
	var types []TemplateType

	for templateType := range f.initRegistry {
		types = append(types, templateType)
	}

	for templateType := range f.apiRegistry {
		types = append(types, templateType)
	}

	for templateType := range f.staticRegistry {
		types = append(types, templateType)
	}

	return types
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

func (f *CrossplaneTemplateFactory) GetStaticTemplates(opts ...Option) ([]TemplateProduct, error) {
	var templates []TemplateProduct

	for templateType, builder := range f.staticRegistry {
		product, err := builder.Build(f.config, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to build static template %s: %w", templateType, err)
		}
		templates = append(templates, product)
	}

	return templates, nil
}


