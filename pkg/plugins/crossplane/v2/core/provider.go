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
	"path"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
)

const defaultProviderName = "provider-example"

func ExtractProviderName(repo string) string {
	if repo == "" {
		return defaultProviderName
	}

	return path.Base(repo)
}

func ExtractProjectName(cfg config.Config) string {
	name := cfg.GetProjectName()
	if name != "" {
		return name
	}

	return ExtractProviderName(cfg.GetRepository())
}

// APIImportPath returns the Go import path a resource's API group/version
// package is generated at: <repo>/apis/<group>/<version>.
func APIImportPath(repo, group, version string) string {
	return repo + "/apis/" + group + "/" + version
}
