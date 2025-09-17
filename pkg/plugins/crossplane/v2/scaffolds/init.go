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

	// Initialize git and build submodule BEFORE automated setup
	if err := s.setupGitAndSubmodule(); err != nil {
		fmt.Printf("Warning: Git setup failed: %v\n", err)
		// Continue anyway - user can set up manually
	}

	// Automate post-init steps
	if err := s.runPostInitSteps(); err != nil {
		fmt.Printf("Warning: Some post-init steps failed: %v\n", err)
		// Don't return error - project is still created successfully
	}

	return nil
}

// setupGitAndSubmodule initializes git repository and build submodule
func (s *InitScaffolder) setupGitAndSubmodule() error {
	// Initialize git repository if not already initialized
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if err := s.runCommand("git", "init"); err != nil {
			return fmt.Errorf("failed to git init: %w", err)
		}
	}

	// Add build submodule if build directory doesn't exist as a submodule
	if _, err := os.Stat("build"); os.IsNotExist(err) {
		buildSubmoduleURL := "https://github.com/crossplane/build"
		if err := s.runCommand("git", "submodule", "add", buildSubmoduleURL, "build"); err != nil {
			return fmt.Errorf("failed to add build submodule: %w", err)
		}
		fmt.Printf("Added build submodule from %s\n", buildSubmoduleURL)
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

	// Verify build system is ready by checking if make targets are available
	if err := s.verifyBuildSystem(); err != nil {
		fmt.Printf("    Warning: Build system not fully ready: %v\n", err)
	}

	// Step 2: Run go mod tidy
	fmt.Printf("  2. Downloading dependencies (go mod tidy)...\n")
	if err := s.runCommand("go", "mod", "tidy"); err != nil {
		fmt.Printf("    Warning: go mod tidy failed: %v (you can run it manually later)\n", err)
	}

	// Step 3: Run make generate
	fmt.Printf("  3. Generating code (make generate)...\n")
	if err := s.runCommand("make", "generate"); err != nil {
		fmt.Printf("    Warning: make generate failed: %v (you can run it manually later)\n", err)
	}

	// Step 4: Run make reviewable
	fmt.Printf("  4. Running quality checks (make reviewable)...\n")
	if err := s.runCommand("make", "reviewable"); err != nil {
		fmt.Printf("    Warning: make reviewable failed: %v (you can run it manually later)\n", err)
	}

	// Step 5: Create initial commit
	fmt.Printf("  5. Creating initial commit...\n")
	if err := s.createInitialCommit(); err != nil {
		fmt.Printf("    Warning: initial commit failed: %v (you can commit manually later)\n", err)
	}

	fmt.Printf("Automated setup completed successfully!\n")
	return nil
}

// runCommand executes a command in the current directory
func (s *InitScaffolder) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	// Set working directory explicitly
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	cmd.Dir = cwd

	// Preserve environment variables
	cmd.Env = os.Environ()

	// For make commands, use combined output to capture errors better
	if name == "make" {
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("    Command failed: %s %v\n", name, args)
			fmt.Printf("    Output: %s\n", string(output))
		} else {
			// Print successful output
			fmt.Print(string(output))
		}
		return err
	}

	// For other commands, use normal stdout/stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// verifyBuildSystem checks if the build system is ready by testing make targets
func (s *InitScaffolder) verifyBuildSystem() error {
	// Try to run make help to see if targets are available
	cmd := exec.Command("make", "help")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("make help failed: %w", err)
	}

	// Check if generate and reviewable targets are available
	outputStr := string(output)
	if !strings.Contains(outputStr, "generate") {
		return fmt.Errorf("generate target not available")
	}
	if !strings.Contains(outputStr, "reviewable") {
		return fmt.Errorf("reviewable target not available")
	}

	return nil
}

// createInitialCommit creates an initial git commit
func (s *InitScaffolder) createInitialCommit() error {
	// Initialize git repository if not already initialized
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if err := s.runCommand("git", "init"); err != nil {
			return fmt.Errorf("failed to git init: %w", err)
		}
	}

	// Set git config if not set (needed for temp directories)
	if err := s.runCommand("git", "config", "user.email", "crossplane-provider-gen@example.com"); err != nil {
		// Ignore error, user might have global config
	}
	if err := s.runCommand("git", "config", "user.name", "Crossplane Provider Generator"); err != nil {
		// Ignore error, user might have global config
	}

	// Add safe directory to handle permission issues in temp directories
	cwd, _ := os.Getwd()
	if err := s.runCommand("git", "config", "--global", "--add", "safe.directory", cwd); err != nil {
		// Ignore error, this is just to handle temp directory issues
	}

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
