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
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

var templateFS embed.FS = templates.TemplateFS

// TemplateLoader loads templates from the embedded filesystem
type TemplateLoader struct {
	fs embed.FS
}

// NewTemplateLoader creates a new template loader
func NewTemplateLoader() *TemplateLoader {
	return &TemplateLoader{
		fs: templateFS,
	}
}

// LoadTemplate loads a template by its name/path
func (tl *TemplateLoader) LoadTemplate(templatePath string) (string, error) {
	// Convert template path to filesystem path
	fsPath := path.Join("files", templatePath)

	content, err := tl.fs.ReadFile(fsPath)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", templatePath, err)
	}

	return string(content), nil
}

// ListTemplates returns all available templates
func (tl *TemplateLoader) ListTemplates() ([]string, error) {
	var templates []string

	err := fs.WalkDir(tl.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(path, ".tmpl") {
			// Remove the base path and .tmpl extension for cleaner names
			templateName := strings.TrimPrefix(path, "files/")
			templateName = strings.TrimSuffix(templateName, ".tmpl")
			templates = append(templates, templateName)
		}

		return nil
	})

	return templates, err
}

// TemplateExists checks if a template exists
func (tl *TemplateLoader) TemplateExists(templatePath string) bool {
	fsPath := path.Join("files", templatePath)
	_, err := tl.fs.ReadFile(fsPath)
	return err == nil
}
