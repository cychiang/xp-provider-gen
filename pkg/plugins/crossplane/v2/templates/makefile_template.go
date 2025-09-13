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

// Makefile creates Makefile template  
func Makefile(cfg config.Config) machinery.Template {
	return SimpleFile(cfg, "Makefile", makefileTemplate)
}

const makefileTemplate = `# Makefile for {{ .ProviderName }}

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
PLATFORMS ?= linux_amd64 linux_arm64 darwin_amd64 darwin_arm64

# Include Crossplane build makelib
include build/makelib/common.mk
include build/makelib/output.mk

# Default target
all: build

# Build the provider
build:
	@echo "Building {{ .ProviderName }}..."
	$(GO_OUT_DIR)/{{ .ProviderName }}

# Generate code
generate:
	@echo "Generating code..."
	go generate ./...

.PHONY: all build generate`