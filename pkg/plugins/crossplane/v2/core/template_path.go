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

package core

import (
	"path/filepath"
	"strings"
)

// Template paths are pure string transforms with no state to carry, so these
// are plain functions: embedded template path in, generated-provider path out.

// IsTemplateFile reports whether a path in the embedded FS is a template.
func IsTemplateFile(path string) bool {
	return strings.HasSuffix(path, ".tmpl")
}

// CleanTemplatePath removes the "files/" prefix from a template path.
func CleanTemplatePath(path string) string {
	return strings.TrimPrefix(path, "files/")
}

// ConvertToFilesystemPath converts a template path back to its location within
// the embedded template FS.
func ConvertToFilesystemPath(templatePath string) string {
	// Ensure we always use forward slashes for embedded filesystem paths.
	cleanPath := strings.ReplaceAll(templatePath, "\\", "/")
	return filepath.Join("files", cleanPath)
}

// GenerateOutputPath converts a template path to its final path inside a
// generated provider: the "files/" prefix and ".tmpl" suffix are dropped, the
// special "project/" prefix maps to the provider root, and any placeholder
// segments (GROUP, VERSION, KIND, IMAGENAME) are substituted.
func GenerateOutputPath(templatePath string, replacements map[string]string) string {
	outputPath := strings.TrimSuffix(CleanTemplatePath(templatePath), ".tmpl")
	outputPath = strings.TrimPrefix(outputPath, "project/")

	for placeholder, value := range replacements {
		outputPath = strings.ReplaceAll(outputPath, placeholder, value)
	}
	return outputPath
}

// PathHasPattern reports whether a path contains any of the given patterns.
func PathHasPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}
