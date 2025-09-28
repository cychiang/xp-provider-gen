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
	"path/filepath"
	"strings"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/templates"
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

func DiscoverTemplates() (map[string]TemplateInfo, error) {
	templates := make(map[string]TemplateInfo)

	processor := core.NewTemplatePathProcessor()

	err := walkTemplateFS("files", func(path string, isDir bool) error {
		if isDir || !processor.IsTemplateFile(path) {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		templates[info.Name] = info

		return nil
	})

	return templates, err
}

func walkTemplateFS(root string, fn func(path string, isDir bool) error) error {
	entries, err := templates.TemplateFS.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())

		if err := fn(path, entry.IsDir()); err != nil {
			return err
		}

		if entry.IsDir() {
			if err := walkTemplateFS(path, fn); err != nil {
				return err
			}
		}
	}

	return nil
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

func determineCategory(path string) TemplateCategory {
	processor := core.NewTemplatePathProcessor()

	apiPatterns := []string{
		"apis/GROUP/VERSION/",
		"internal/controller/KIND/",
		"examples/GROUP/",
	}

	staticPatterns := []string{
		"LICENSE",
	}

	if processor.PathHasPattern(path, apiPatterns) {
		return APICategory
	}

	if processor.PathHasPattern(path, staticPatterns) {
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
