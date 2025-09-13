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

// CrossplanePackage creates package/crossplane.yaml
func CrossplanePackage(cfg config.Config) machinery.Template {
	return SimpleFile(cfg, "package/crossplane.yaml", crossplanePackageTemplate)
}

const crossplanePackageTemplate = `apiVersion: meta.pkg.crossplane.io/v1alpha1
kind: Provider
metadata:
  name: {{ .ProviderName }}
  annotations:
    meta.crossplane.io/maintainer: {{ .ProviderName }} Maintainers
    meta.crossplane.io/source: {{ .Repo }}
    meta.crossplane.io/license: Apache-2.0
    meta.crossplane.io/description: |
      {{ .ProviderName }} is a Crossplane provider for managing cloud resources.
spec:
  controller:
    image: {{ .Repo }}:latest
  crossplane:
    version: ">=v1.14.0-0"
  dependsOn:
    - provider: xpkg.upbound.io/crossplane-contrib/provider-kubernetes
      version: ">=v0.4.0"`
