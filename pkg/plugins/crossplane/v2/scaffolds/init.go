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

package scaffolds

import (
	"fmt"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/internal/templates"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/internal/templates/apis"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/internal/templates/controllers"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/internal/templates/hack"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/internal/templates/pkg"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/internal/templates/providerconfig"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/internal/templates/version"
)


// InitScaffolder scaffolds the project structure for a Crossplane provider
type InitScaffolder struct {
	config      config.Config
	boilerplate string
}

// NewInitScaffolder returns a new InitScaffolder for Crossplane provider projects
func NewInitScaffolder(config config.Config) *InitScaffolder {
	return &InitScaffolder{
		config: config,
		boilerplate: `/*
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
*/`,
	}
}

// Scaffold scaffolds the Crossplane provider project structure
func (s *InitScaffolder) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project structure...\n")

	// Extract provider name from repository
	var providerName string
	if s.config.GetRepository() != "" {
		parts := strings.Split(s.config.GetRepository(), "/")
		if len(parts) > 0 {
			providerName = parts[len(parts)-1]
		}
	}
	if providerName == "" {
		providerName = "provider-example"
	}

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(s.boilerplate),
	)

	// Execute scaffolding templates for Crossplane provider
	if err := scaffold.Execute(
		// Basic project structure
		&templates.GoMod{},
		&templates.Main{},
		&templates.Makefile{},
		&templates.Dockerfile{},
		&templates.ReadMe{},
		&templates.GitIgnore{},
		&templates.GitModules{},
		&templates.ControllerTemplate{
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
		},
		
		// APIs registration
		&apis.APIs{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
			ProviderName: providerName,
		},
		
		// Code generation files
		&apis.Generate{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
		},
		&hack.Boilerplate{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
		},
		
		// ProviderConfig APIs - critical for Crossplane providers
		&providerconfig.ProviderConfigTypes{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
			Domain: s.config.GetDomain(),
			ProviderName: providerName,
		},
		&providerconfig.ProviderConfigRegister{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
			Domain: s.config.GetDomain(),
			ProviderName: providerName,
		},
		&providerconfig.ProviderConfigDoc{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
			Domain: s.config.GetDomain(),
			ProviderName: providerName,
		},
		
		// Package metadata for Crossplane
		&pkg.CrossplanePackage{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
			ProviderName: providerName,
		},
		
		// Config controller
		&controllers.ConfigController{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
			ProviderName: providerName,
		},
		
		// Version management
		&version.Version{
			TemplateMixin: machinery.TemplateMixin{},
			DomainMixin: machinery.DomainMixin{Domain: s.config.GetDomain()},
			RepositoryMixin: machinery.RepositoryMixin{Repo: s.config.GetRepository()},
		},
	); err != nil {
		return fmt.Errorf("error scaffolding Crossplane provider project: %w", err)
	}

	fmt.Printf("Crossplane provider project scaffolded successfully!\n")
	return nil
}