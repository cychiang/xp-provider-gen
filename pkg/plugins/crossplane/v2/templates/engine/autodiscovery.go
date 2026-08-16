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
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// TemplateCategory says when a discovered template renders.
type TemplateCategory string

const (
	// InitCategory templates render once, at `init`.
	InitCategory TemplateCategory = "init"
	// APICategory templates render per managed resource kind, at `create api`.
	APICategory TemplateCategory = "api"
)

// TemplateInfo is one discovered template: its path in the embedded FS and
// when it renders. The output path is derived from Path at render time, once
// the placeholder values are known.
type TemplateInfo struct {
	Path     string
	Category TemplateCategory
}

// AnalyzeTemplatePath classifies one embedded template path.
func AnalyzeTemplatePath(path string) TemplateInfo {
	return TemplateInfo{Path: path, Category: determineCategory(path)}
}

// determineCategory infers when a template renders from its path placeholders:
// GROUP/VERSION/KIND mean the output path depends on a resource, so the
// template renders per kind at `create api`. IMAGENAME (the provider name) and
// placeholder-free paths render once at `init`.
func determineCategory(path string) TemplateCategory {
	if core.PathHasPattern(path, []string{placeholderGroup, placeholderVersion, placeholderKind}) {
		return APICategory
	}
	return InitCategory
}
