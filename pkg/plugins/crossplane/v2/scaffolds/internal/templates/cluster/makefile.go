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

package cluster

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &ClusterMakefile{}

// ClusterMakefile scaffolds the cluster/images/provider/Makefile
type ClusterMakefile struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
	
	// ProviderName is the name extracted from the repository
	ProviderName string
}

// SetTemplateDefaults implements machinery.Template
func (f *ClusterMakefile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("cluster", "images", f.ProviderName, "Makefile")
	}

	f.TemplateBody = clusterMakefileTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

const clusterMakefileTemplate = `# ====================================================================================
# Setup Project

include ../../../build/makelib/common.mk

# ====================================================================================
#  Options

include ../../../build/makelib/imagelight.mk

# ====================================================================================
# Targets

img.build:
	@$(INFO) docker build $(IMAGE)
	@$(MAKE) BUILD_ARGS="--load" img.build.shared
	@$(OK) docker build $(IMAGE)

img.publish:
	@$(INFO) Skipping image publish for $(IMAGE)
	@echo Publish is deferred to xpkg machinery
	@$(OK) Image publish skipped for $(IMAGE)

img.build.shared:
	@cp Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -r $(OUTPUT_DIR)/bin/ $(IMAGE_TEMP_DIR)/bin || $(FAIL)
	@docker buildx build $(BUILD_ARGS) \
		--platform $(IMAGE_PLATFORMS) \
		-t $(IMAGE) \
		$(IMAGE_TEMP_DIR) || $(FAIL)

img.promote:
	@$(INFO) Skipping image promotion from $(FROM_IMAGE) to $(TO_IMAGE)
	@echo Promote is deferred to xpkg machinery
	@$(OK) Image promotion skipped for $(FROM_IMAGE) to $(TO_IMAGE)
`