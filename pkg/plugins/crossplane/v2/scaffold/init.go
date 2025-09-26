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

package scaffold

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/templates/engine"
)

type InitScaffolder struct {
	config config.Config
}

func NewInitScaffolder(config config.Config) *InitScaffolder {
	return &InitScaffolder{
		config: config,
	}
}

func (s *InitScaffolder) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project structure...\n")

	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(engine.DefaultBoilerplate()),
	)

	factory := engine.NewFactory(s.config)

	initTemplates, err := factory.GetInitTemplates()
	if err != nil {
		return fmt.Errorf("failed to get init templates: %w", err)
	}

	staticTemplates, err := factory.GetStaticTemplates()
	if err != nil {
		return fmt.Errorf("failed to get static templates: %w", err)
	}

	allTemplates := make([]machinery.Builder, 0, len(initTemplates)+len(staticTemplates))
	for _, tmpl := range initTemplates {
		allTemplates = append(allTemplates, tmpl)
	}
	for _, tmpl := range staticTemplates {
		allTemplates = append(allTemplates, tmpl)
	}

	if err := scaffold.Execute(allTemplates...); err != nil {
		return fmt.Errorf("error scaffolding Crossplane provider project: %w", err)
	}

	fmt.Printf("Crossplane provider project scaffolded successfully!\n")

	s.runPostInitSteps()

	return nil
}

func (s *InitScaffolder) setupGitAndSubmodule() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if err := s.runCommand("git", "init"); err != nil {
			return fmt.Errorf("failed to git init: %w", err)
		}
	}
	return nil
}

func (s *InitScaffolder) addBuildSubmodule() error {
	return s.setupBuildSubmodule()
}

func (s *InitScaffolder) setupBuildSubmodule() error {
	// Check if build directory exists
	if _, err := os.Stat("build"); os.IsNotExist(err) {
		return s.addNewBuildSubmodule()
	}

	return s.initializeExistingBuildSubmodule()
}

func (s *InitScaffolder) addNewBuildSubmodule() error {
	buildSubmoduleURL := "https://github.com/crossplane/build"
	if err := s.runCommand("git", "submodule", "add", buildSubmoduleURL, "build"); err != nil {
		return fmt.Errorf("failed to add build submodule: %w", err)
	}
	fmt.Printf("Added build submodule from %s\n", buildSubmoduleURL)

	if err := s.runCommand("git", "submodule", "update", "--init", "--recursive"); err != nil {
		return fmt.Errorf("failed to initialize build submodule: %w", err)
	}
	fmt.Printf("Initialized build submodule content\n")
	return nil
}

func (s *InitScaffolder) initializeExistingBuildSubmodule() error {
	if _, err := os.Stat("build/.git"); os.IsNotExist(err) {
		if err := s.runCommand("git", "submodule", "update", "--init", "--recursive"); err != nil {
			return fmt.Errorf("failed to initialize existing build submodule: %w", err)
		}
		fmt.Printf("Initialized existing build submodule content\n")
	}
	return nil
}

func (s *InitScaffolder) runPostInitSteps() {
	fmt.Printf("Running automated setup steps...\n")

	fmt.Printf("  1. Setting up git repository...\n")
	if err := s.setupGitAndSubmodule(); err != nil {
		fmt.Printf("    Warning: git setup failed: %v\n", err)
	}

	fmt.Printf("  2. Creating initial commit...\n")
	if err := s.createInitialCommit(); err != nil {
		fmt.Printf("    Warning: initial commit failed: %v (you can commit manually later)\n", err)
	}

	fmt.Printf("  3. Adding build submodule...\n")
	if err := s.addBuildSubmodule(); err != nil {
		fmt.Printf("    Warning: build submodule setup failed: %v\n", err)
	}

	fmt.Printf("  4. Setting up build system (make submodules)...\n")
	if err := s.runCommand("make", "submodules"); err != nil {
		fmt.Printf("    Warning: make submodules failed: %v (you can run it manually later)\n", err)
	}

	if err := s.verifyBuildSystem(); err != nil {
		fmt.Printf("    Warning: Build system not fully ready: %v\n", err)
	}

	fmt.Printf("  5. Downloading dependencies (go mod tidy)...\n")
	if err := s.runCommand("go", "mod", "tidy"); err != nil {
		fmt.Printf("    Warning: go mod tidy failed: %v (you can run it manually later)\n", err)
	}

	fmt.Printf("  6. Generating code (make generate)...\n")
	if err := s.runCommand("make", "generate"); err != nil {
		fmt.Printf("    Warning: make generate failed: %v (you can run it manually later)\n", err)
	}

	fmt.Printf("  7. Running quality checks (make reviewable)...\n")
	if err := s.runCommand("make", "reviewable"); err != nil {
		fmt.Printf("    Warning: make reviewable failed: %v (you can run it manually later)\n", err)
	}

	fmt.Printf("Automated setup completed successfully!\n")
}

func (s *InitScaffolder) runCommand(name string, args ...string) error {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, name, args...)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	cmd.Dir = cwd

	cmd.Env = os.Environ()

	if name == "make" {
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("    Command failed: %s %v\n", name, args)
			fmt.Printf("    Output: %s\n", string(output))
		} else {
			fmt.Print(string(output))
		}
		return err
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *InitScaffolder) verifyBuildSystem() error {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "make", "help")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("make help failed: %w", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "generate") {
		return fmt.Errorf("generate target not available")
	}
	if !strings.Contains(outputStr, "reviewable") {
		return fmt.Errorf("reviewable target not available")
	}

	return nil
}

func (s *InitScaffolder) createInitialCommit() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if err := s.runCommand("git", "init"); err != nil {
			return fmt.Errorf("failed to git init: %w", err)
		}
	}

	// Best effort git config setup - ignore errors as they're not critical
	_ = s.runCommand("git", "config", "user.email", "crossplane-provider-gen@example.com")
	_ = s.runCommand("git", "config", "user.name", "Crossplane Provider Generator")

	cwd, _ := os.Getwd()
	_ = s.runCommand("git", "config", "--global", "--add", "safe.directory", cwd)

	if err := s.runCommand("git", "add", "."); err != nil {
		return fmt.Errorf("failed to git add: %w", err)
	}

	commitMessage := fmt.Sprintf(`Initial Crossplane provider project

This project provides a Crossplane provider for %s resources.
Generated project includes CRDs, controllers, and package configuration.
Ready for implementing custom resource management logic.`, s.extractProviderName())

	if err := s.runCommand("git", "commit", "-m", commitMessage); err != nil {
		return fmt.Errorf("failed to git commit: %w", err)
	}

	return nil
}

func (s *InitScaffolder) extractProviderName() string {
	name := core.ExtractProviderName(s.config.GetRepository())
	if name == "provider-example" {
		return "crossplane-provider"
	}
	return name
}
