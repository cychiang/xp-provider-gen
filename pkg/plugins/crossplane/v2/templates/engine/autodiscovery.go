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
	"strings"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

type TemplateCategory string

const (
	InitCategory   TemplateCategory = "init"
	APICategory    TemplateCategory = "api"
	StaticCategory TemplateCategory = "static"
)

type TemplateInfo struct {
	Name      string
	Path      string
	Category  TemplateCategory
	OutputDir string
}

func AnalyzeTemplatePath(path string) TemplateInfo {
	processor := core.NewTemplatePathProcessor()

	cleanPath := processor.CleanTemplatePath(path)
	name := processor.GetTemplateBaseName(path)
	outputPath := processor.GetOutputPath(path)

	category := determineCategory(cleanPath)

	return TemplateInfo{
		Name:      name,
		Path:      path,
		Category:  category,
		OutputDir: outputPath,
	}
}

// determineCategory infers when a template renders from its path placeholders:
// GROUP/VERSION/KIND mean the output path depends on a resource, so the
// template renders per kind at `create api`. IMAGENAME (the provider name) and
// placeholder-free paths render once at `init`. LICENSE is static.
func determineCategory(path string) TemplateCategory {
	processor := core.NewTemplatePathProcessor()

	if processor.PathHasPattern(path, []string{placeholderGroup, placeholderVersion, placeholderKind}) {
		return APICategory
	}

	if processor.PathHasPattern(path, []string{"LICENSE"}) {
		return StaticCategory
	}

	return InitCategory
}

func (t TemplateInfo) GenerateTemplateType() TemplateType {
	processor := core.NewTemplatePathProcessor()
	parts := processor.SplitPathComponents(t.OutputDir)
	var name string

	for _, part := range parts {
		words := strings.FieldsFunc(part, func(c rune) bool {
			return c == '-' || c == '_' || c == '.'
		})

		for _, word := range words {
			if len(word) > 0 {
				name += strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
			}
		}
	}

	return TemplateType(name + "Type")
}
