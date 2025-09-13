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
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// VersionGo creates version management
func VersionGo(cfg config.Config) machinery.Template {
	return SimpleFile(cfg, "internal/version/version.go", versionGoTemplate)
}

const versionGoTemplate = `package version

import "runtime/debug"

// Version is the version of {{ .ProviderName }}.
var Version string

// GetVersion returns the version of {{ .ProviderName }}.
func GetVersion() string {
	if Version != "" {
		return Version
	}

	// Fallback to build info if version is not set
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return "unknown"
}`
