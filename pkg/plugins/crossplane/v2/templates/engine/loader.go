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

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

var templateFS embed.FS = templates.TemplateFS

// TemplateLoader loads templates from the embedded filesystem.
type TemplateLoader struct {
	fs embed.FS
}

// NewTemplateLoader creates a new template loader.
func NewTemplateLoader() *TemplateLoader {
	return &TemplateLoader{
		fs: templateFS,
	}
}

// LoadTemplate loads a template by its name/path.
func (tl *TemplateLoader) LoadTemplate(templatePath string) (string, error) {
	processor := core.NewTemplatePathProcessor()

	// Convert template path to filesystem path
	fsPath := processor.ConvertToFilesystemPath(templatePath)

	content, err := tl.fs.ReadFile(fsPath)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", templatePath, err)
	}

	return string(content), nil
}
