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

	err := walkTemplateFS("files", func(path string, isDir bool) error {
		if isDir || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		info := AnalyzeTemplatePath(path)
		templates[info.Name] = info

		return nil
	})

	return templates, err
}

func walkTemplateFS(root string, fn func(path string, isDir bool) error) error {
	entries, err := templateFS.ReadDir(root)
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
	cleanPath := strings.TrimPrefix(path, "files/")
	name := strings.TrimSuffix(filepath.Base(cleanPath), ".tmpl")
	outputPath := strings.TrimSuffix(cleanPath, ".tmpl")

	category := determineCategory(cleanPath)

	return TemplateInfo{
		Name:      name,
		Path:      path,
		Category:  category,
		OutputDir: outputPath,
	}
}

func determineCategory(path string) TemplateCategory {
	apiPatterns := []string{
		"apis/GROUP/VERSION/",
		"internal/controller/KIND/",
		"examples/GROUP/",
	}

	staticPatterns := []string{
		"LICENSE",
	}

	for _, pattern := range apiPatterns {
		if strings.Contains(path, pattern) {
			return APICategory
		}
	}

	for _, pattern := range staticPatterns {
		if strings.Contains(path, pattern) {
			return StaticCategory
		}
	}

	return InitCategory
}

func (t TemplateInfo) GenerateTemplateType() TemplateType {
	parts := strings.Split(t.OutputDir, "/")
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

