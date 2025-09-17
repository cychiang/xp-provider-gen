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

type InitScaffolder struct {
	config      config.Config
	boilerplate string
}

func NewInitScaffolder(config config.Config) *InitScaffolder {
	return &InitScaffolder{
		config:      config,
		boilerplate: templates.DefaultBoilerplate(),
	}
}

func (s *InitScaffolder) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project structure...\n")

	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(s.boilerplate),
	)

	factory := templates.NewFactory(s.config).(*templates.CrossplaneTemplateFactory)

	goMod, err := factory.GoMod()
	if err != nil {
		return fmt.Errorf("failed to create GoMod template: %w", err)
	}
	makefile, err := factory.Makefile()
	if err != nil {
		return fmt.Errorf("failed to create Makefile template: %w", err)
	}
	readme, err := factory.README()
	if err != nil {
		return fmt.Errorf("failed to create README template: %w", err)
	}
	gitIgnore, err := factory.GitIgnore()
	if err != nil {
		return fmt.Errorf("failed to create GitIgnore template: %w", err)
	}
	mainGo, err := factory.MainGo()
	if err != nil {
		return fmt.Errorf("failed to create MainGo template: %w", err)
	}
	apisTemplateType, err := factory.FindTemplateTypeByPath("apis")
	if err != nil {
		return fmt.Errorf("failed to find APIs template: %w", err)
	}
	apisTemplate, err := factory.CreateInitTemplate(apisTemplateType)
	if err != nil {
		return fmt.Errorf("failed to create APIs template: %w", err)
	}
	generateGoType, err := factory.FindTemplateTypeByPath("generatego")
	if err != nil {
		return fmt.Errorf("failed to find GenerateGo template: %w", err)
	}
	generateGo, err := factory.CreateInitTemplate(generateGoType)
	if err != nil {
		return fmt.Errorf("failed to create GenerateGo template: %w", err)
	}
	boilerplateType, err := factory.FindTemplateTypeByPath("boilerplate")
	if err != nil {
		return fmt.Errorf("failed to find Boilerplate template: %w", err)
	}
	boilerplate, err := factory.CreateInitTemplate(boilerplateType)
	if err != nil {
		return fmt.Errorf("failed to create Boilerplate template: %w", err)
	}
	providerConfigTypes, err := factory.ProviderConfigTypes()
	if err != nil {
		return fmt.Errorf("failed to create ProviderConfigTypes template: %w", err)
	}
	providerConfigRegister, err := factory.ProviderConfigRegister()
	if err != nil {
		return fmt.Errorf("failed to create ProviderConfigRegister template: %w", err)
	}
	crossplanePackage, err := factory.CrossplanePackage()
	if err != nil {
		return fmt.Errorf("failed to create CrossplanePackage template: %w", err)
	}
	configController, err := factory.ConfigController()
	if err != nil {
		return fmt.Errorf("failed to create ConfigController template: %w", err)
	}
	controllerRegister, err := factory.ControllerRegister()
	if err != nil {
		return fmt.Errorf("failed to create ControllerRegister template: %w", err)
	}
	clusterDockerfile, err := factory.ClusterDockerfile()
	if err != nil {
		return fmt.Errorf("failed to create ClusterDockerfile template: %w", err)
	}
	clusterMakefile, err := factory.ClusterMakefile()
	if err != nil {
		return fmt.Errorf("failed to create ClusterMakefile template: %w", err)
	}
	versionGo, err := factory.VersionGo()
	if err != nil {
		return fmt.Errorf("failed to create VersionGo template: %w", err)
	}
	license, err := factory.License()
	if err != nil {
		return fmt.Errorf("failed to create License template: %w", err)
	}
	docGo, err := factory.DocGo()
	if err != nil {
		return fmt.Errorf("failed to create DocGo template: %w", err)
	}
	examplesProviderConfig, err := factory.ExamplesProviderConfig()
	if err != nil {
		return fmt.Errorf("failed to create ExamplesProviderConfig template: %w", err)
	}

	if err := scaffold.Execute(
		goMod,
		makefile,
		readme,
		gitIgnore,
		mainGo,
		apisTemplate,
		generateGo,
		boilerplate,

		providerConfigTypes,
		providerConfigRegister,
		docGo,

		crossplanePackage,
		configController,
		controllerRegister,

		clusterDockerfile,
		clusterMakefile,

		versionGo,

		license,

		examplesProviderConfig,
	); err != nil {
		return fmt.Errorf("error scaffolding Crossplane provider project: %w", err)
	}

	fmt.Printf("Crossplane provider project scaffolded successfully!\n")

	if err := s.setupGitAndSubmodule(); err != nil {
		fmt.Printf("Warning: Git setup failed: %v\n", err)
	}

	if err := s.runPostInitSteps(); err != nil {
		fmt.Printf("Warning: Some post-init steps failed: %v\n", err)
	}

	return nil
}

func (s *InitScaffolder) setupGitAndSubmodule() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		if err := s.runCommand("git", "init"); err != nil {
			return fmt.Errorf("failed to git init: %w", err)
		}
	}

	if _, err := os.Stat("build"); os.IsNotExist(err) {
		buildSubmoduleURL := "https://github.com/crossplane/build"
		if err := s.runCommand("git", "submodule", "add", buildSubmoduleURL, "build"); err != nil {
			return fmt.Errorf("failed to add build submodule: %w", err)
		}
		fmt.Printf("Added build submodule from %s\n", buildSubmoduleURL)
	}

	return nil
}

func (s *InitScaffolder) runPostInitSteps() error {
	fmt.Printf("Running automated setup steps...\n")

	fmt.Printf("  1. Creating initial commit...\n")
	if err := s.createInitialCommit(); err != nil {
		fmt.Printf("    Warning: initial commit failed: %v (you can commit manually later)\n", err)
	}

	fmt.Printf("  2. Setting up build system (make submodules)...\n")
	if err := s.runCommand("make", "submodules"); err != nil {
		fmt.Printf("    Warning: make submodules failed: %v (you can run it manually later)\n", err)
	}

	if err := s.verifyBuildSystem(); err != nil {
		fmt.Printf("    Warning: Build system not fully ready: %v\n", err)
	}

	fmt.Printf("  3. Downloading dependencies (go mod tidy)...\n")
	if err := s.runCommand("go", "mod", "tidy"); err != nil {
		fmt.Printf("    Warning: go mod tidy failed: %v (you can run it manually later)\n", err)
	}

	fmt.Printf("  4. Generating code (make generate)...\n")
	if err := s.runCommand("make", "generate"); err != nil {
		fmt.Printf("    Warning: make generate failed: %v (you can run it manually later)\n", err)
	}

	fmt.Printf("  5. Running quality checks (make reviewable)...\n")
	if err := s.runCommand("make", "reviewable"); err != nil {
		fmt.Printf("    Warning: make reviewable failed: %v (you can run it manually later)\n", err)
	}

	fmt.Printf("Automated setup completed successfully!\n")
	return nil
}

func (s *InitScaffolder) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
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
	cmd := exec.Command("make", "help")
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

	if err := s.runCommand("git", "config", "user.email", "crossplane-provider-gen@example.com"); err != nil {
	}
	if err := s.runCommand("git", "config", "user.name", "Crossplane Provider Generator"); err != nil {
	}

	cwd, _ := os.Getwd()
	if err := s.runCommand("git", "config", "--global", "--add", "safe.directory", cwd); err != nil {
	}

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
	if s.config.GetRepository() != "" {
		parts := strings.Split(s.config.GetRepository(), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return "crossplane-provider"
}
