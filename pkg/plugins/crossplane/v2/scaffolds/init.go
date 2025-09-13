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

	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/templates"
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

	// Create template factory for ultra-simple template creation
	factory := templates.NewFactory(s.config)
	
	// Execute scaffolding - dramatically simplified from 80+ lines to ~15 lines!
	if err := scaffold.Execute(
		// Basic project structure
		factory.GoMod(),
		factory.Makefile(),
		factory.README(),
		factory.GitIgnore(),
		factory.MainGo(),
		
		// ProviderConfig APIs  
		factory.ProviderConfigTypes(),
		factory.ProviderConfigRegister(),
		
		// Package and controllers
		factory.CrossplanePackage(),
		factory.ConfigController(),
		factory.ControllerRegister(),
		factory.VersionGo(),
		
		// Cluster build configuration
		factory.ClusterDockerfile(),
		factory.ClusterMakefile(),
	); err != nil {
		return fmt.Errorf("error scaffolding Crossplane provider project: %w", err)
	}

	fmt.Printf("Crossplane provider project scaffolded successfully!\n")
	return nil
}