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
	"os"
	"os/exec"
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
		config:      config,
		boilerplate: templates.DefaultBoilerplate(),
	}
}

// Scaffold scaffolds the Crossplane provider project structure
func (s *InitScaffolder) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project structure...\n")

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(s.boilerplate),
	)

	// Create template factory for true Factory Pattern template creation
	factory := templates.NewFactory(s.config).(*templates.CrossplaneTemplateFactory)

	// Get all templates using the factory pattern
	goMod, _ := factory.GoMod()
	makefile, _ := factory.Makefile()
	readme, _ := factory.README()
	gitIgnore, _ := factory.GitIgnore()
	mainGo, _ := factory.MainGo()
	apisTemplate, _ := factory.CreateInitTemplate(templates.APIsTemplateType)
	generateGo, _ := factory.CreateInitTemplate(templates.GenerateGoTemplateType)
	boilerplate, _ := factory.CreateInitTemplate(templates.BoilerplateTemplateType)
	providerConfigTypes, _ := factory.ProviderConfigTypes()
	providerConfigRegister, _ := factory.ProviderConfigRegister()
	crossplanePackage, _ := factory.CrossplanePackage()
	configController, _ := factory.ConfigController()
	controllerRegister, _ := factory.ControllerRegister()
	clusterDockerfile, _ := factory.ClusterDockerfile()
	clusterMakefile, _ := factory.ClusterMakefile()
	versionGo, _ := factory.VersionGo()
	license, _ := factory.License()
	docGo, _ := factory.DocGo()

	// Execute scaffolding with proper Factory Pattern
	if err := scaffold.Execute(
		// Basic project structure
		goMod,
		makefile,
		readme,
		gitIgnore,
		mainGo,
		apisTemplate,
		generateGo,
		boilerplate,

		// ProviderConfig APIs
		providerConfigTypes,
		providerConfigRegister,
		docGo,

		// Package and controllers
		crossplanePackage,
		configController,
		controllerRegister,

		// Cluster resources for Docker image building
		clusterDockerfile,
		clusterMakefile,

		// Version information
		versionGo,

		// Static files
		license,
	); err != nil {
		return fmt.Errorf("error scaffolding Crossplane provider project: %w", err)
	}

	fmt.Printf("Crossplane provider project scaffolded successfully!\n")

	// Automate post-init steps
	if err := s.runPostInitSteps(); err != nil {
		fmt.Printf("Warning: Some post-init steps failed: %v\n", err)
		// Don't return error - project is still created successfully
	}

	return nil
}

// runPostInitSteps runs the automated steps after project scaffolding
func (s *InitScaffolder) runPostInitSteps() error {
	fmt.Printf("Running automated setup steps...\n")

	// Step 1: Run make submodules
	fmt.Printf("  1. Setting up build system (make submodules)...\n")
	if err := s.runCommand("make", "submodules"); err != nil {
		return fmt.Errorf("failed to run make submodules: %w", err)
	}

	// Step 2: Run make generate
	fmt.Printf("  2. Generating code (make generate)...\n")
	if err := s.runCommand("make", "generate"); err != nil {
		fmt.Printf("    Warning: make generate failed: %v (you can run it manually later)\n", err)
	}

	// Step 3: Run make reviewable
	fmt.Printf("  3. Running quality checks (make reviewable)...\n")
	if err := s.runCommand("make", "reviewable"); err != nil {
		fmt.Printf("    Warning: make reviewable failed: %v (you can run it manually later)\n", err)
	}

	// Step 4: Create initial commit
	fmt.Printf("  4. Creating initial commit...\n")
	if err := s.createInitialCommit(); err != nil {
		fmt.Printf("    Warning: initial commit failed: %v (you can commit manually later)\n", err)
	}

	fmt.Printf("Automated setup completed successfully!\n")
	return nil
}

// runCommand executes a command in the current directory
func (s *InitScaffolder) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// createInitialCommit creates an initial git commit
func (s *InitScaffolder) createInitialCommit() error {
	// Add all files
	if err := s.runCommand("git", "add", "."); err != nil {
		return fmt.Errorf("failed to git add: %w", err)
	}

	// Create initial commit
	commitMessage := fmt.Sprintf(`Initial Crossplane provider project

This project provides a Crossplane provider for %s resources.
Generated project includes CRDs, controllers, and package configuration.
Ready for implementing custom resource management logic.`, s.extractProviderName())

	if err := s.runCommand("git", "commit", "-m", commitMessage); err != nil {
		return fmt.Errorf("failed to git commit: %w", err)
	}

	return nil
}

// extractProviderName extracts provider name from config
func (s *InitScaffolder) extractProviderName() string {
	if s.config.GetRepository() != "" {
		parts := strings.Split(s.config.GetRepository(), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return "crossplane-provider"
}
