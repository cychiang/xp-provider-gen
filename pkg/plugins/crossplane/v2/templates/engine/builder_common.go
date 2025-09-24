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
)

func findTemplateInfoByCategory(category TemplateCategory, templateType TemplateType) (TemplateInfo, error) {
	var foundInfo TemplateInfo
	found := false

	err := walkTemplateFS("files", func(path string, isDir bool) error {
		if isDir || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		if info.Category == category && TemplateType(info.GenerateTemplateType()) == templateType {
			foundInfo = info
			found = true
		}
		return nil
	})

	if err != nil {
		return TemplateInfo{}, err
	}

	if !found {
		return TemplateInfo{}, fmt.Errorf("template not found for type: %s", templateType)
	}

	return foundInfo, nil
}

func parseOptions(opts []Option) *TemplateOptions {
	options := &TemplateOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

func configureTemplateProduct(product TemplateProduct, cfg config.Config, options *TemplateOptions) error {
	if err := product.Configure(cfg); err != nil {
		return fmt.Errorf("failed to configure template: %w", err)
	}

	if options.Resource != nil {
		if err := product.SetResource(options.Resource); err != nil {
			return fmt.Errorf("failed to set resource: %w", err)
		}
	}

	if genericProduct, ok := product.(*GenericTemplateProduct); ok {
		base := genericProduct.GetBase()
		if options.Force {
			base.SetForce(options.Force)
		}
		if options.CustomData != nil {
			base.SetCustomData(options.CustomData)
		}
	}

	if err := product.SetTemplateDefaults(); err != nil {
		return fmt.Errorf("failed to set template defaults: %w", err)
	}

	return nil
}

func createTemplateProduct(templateType TemplateType, info TemplateInfo, replacements map[string]string) TemplateProduct {
	outputPath := generateOutputPath(info, replacements)
	templatePath := strings.TrimPrefix(info.Path, "files/")
	return NewGenericTemplateProduct(templateType, outputPath, templatePath)
}