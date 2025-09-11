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

package v1

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitUtils provides git-related utility functions
type GitUtils struct {
	config *PluginConfig
}

// NewGitUtils creates a new GitUtils instance
func NewGitUtils(config *PluginConfig) *GitUtils {
	return &GitUtils{config: config}
}

// InitRepo initializes a git repository if one doesn't exist
func (g *GitUtils) InitRepo() error {
	// Check if .git directory already exists
	if _, err := os.Stat(".git"); err == nil {
		// Git repo already exists
		return nil
	}
	
	// Initialize git repository
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}
	
	return nil
}

// CreateInitialCommit adds all files and creates an initial commit
func (g *GitUtils) CreateInitialCommit() error {
	// Add all scaffolded files to git
	cmd := exec.Command("git", "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files to git: %w", err)
	}
	
	// Create initial commit with fallback author if none configured
	commitMsg := "Initial commit\n\nScaffolded Crossplane provider project"
	authorFlag := fmt.Sprintf("--author=%s", g.config.GetDefaultAuthor())
	
	cmd = exec.Command("git", "commit", "-m", commitMsg, authorFlag)
	if err := cmd.Run(); err != nil {
		// Try without author override in case user has git configured
		cmd = exec.Command("git", "commit", "-m", commitMsg)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create initial commit: %w", err)
		}
	}
	
	return nil
}

// AddBuildSubmodule adds the build submodule from crossplane/build
func (g *GitUtils) AddBuildSubmodule() error {
	// Check if build directory already exists
	if _, err := os.Stat("build"); err == nil {
		// Build directory already exists, skip
		return nil
	}
	
	// Add the build submodule
	cmd := exec.Command("git", "submodule", "add", g.config.Git.BuildSubmoduleURL, "build")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add build submodule: %w", err)
	}
	
	fmt.Printf("Added build submodule from %s\n", g.config.Git.BuildSubmoduleURL)
	return nil
}

// StringUtils provides string manipulation utilities
type StringUtils struct{}

// NewStringUtils creates a new StringUtils instance
func NewStringUtils() *StringUtils {
	return &StringUtils{}
}

// ToLowerKebab converts a string to lowercase kebab-case
func (s *StringUtils) ToLowerKebab(input string) string {
	result := strings.ToLower(input)
	result = strings.ReplaceAll(result, "_", "-")
	result = strings.ReplaceAll(result, " ", "-")
	return result
}

// EnsureProviderPrefix ensures a string has the "provider-" prefix
func (s *StringUtils) EnsureProviderPrefix(input string) string {
	if strings.HasPrefix(input, "provider-") {
		return input
	}
	
	if strings.HasPrefix(input, "crossplane-") {
		// Replace crossplane- with provider-
		return strings.Replace(input, "crossplane-", "provider-", 1)
	}
	
	// Add provider- prefix
	return "provider-" + input
}

// ValidationUtils provides validation utilities
type ValidationUtils struct {
	config *PluginConfig
}

// NewValidationUtils creates a new ValidationUtils instance
func NewValidationUtils(config *PluginConfig) *ValidationUtils {
	return &ValidationUtils{config: config}
}

// Simplified validation - no provider type validation needed

// IsEmptyString checks if a string is empty or whitespace-only
func (v *ValidationUtils) IsEmptyString(s string) bool {
	return strings.TrimSpace(s) == ""
}

// MetadataUtils provides utilities for plugin metadata
type MetadataUtils struct {
	config *PluginConfig
}

// NewMetadataUtils creates a new MetadataUtils instance
func NewMetadataUtils(config *PluginConfig) *MetadataUtils {
	return &MetadataUtils{config: config}
}

// FormatExamples formats command examples with proper indentation
func (m *MetadataUtils) FormatExamples(commandName string, examples []string) string {
	var formatted strings.Builder
	for i, example := range examples {
		if i > 0 {
			formatted.WriteString("\n\n")
		}
		formatted.WriteString(fmt.Sprintf("  %s", 
			strings.ReplaceAll(example, "%[1]s", commandName)))
	}
	return formatted.String()
}

// GetInitExamples returns standardized examples for init command
func (m *MetadataUtils) GetInitExamples(commandName string) string {
	examples := []string{
		"# Initialize a basic Crossplane provider project",
		"%[1]s init --domain=example.com --repo=github.com/example/provider-example",
	}
	return m.FormatExamples(commandName, examples)
}

// GetCreateAPIExamples returns standardized examples for create api command
func (m *MetadataUtils) GetCreateAPIExamples(commandName string) string {
	pluginFlag := fmt.Sprintf("--plugins=%s", m.config.Name)
	examples := []string{
		"# Create a basic managed resource",
		fmt.Sprintf("%%[1]s create api %s --group=compute --version=v1alpha1 --kind=Instance", pluginFlag),
		"",
		"# Create a storage resource", 
		fmt.Sprintf("%%[1]s create api %s --group=storage --version=v1beta1 --kind=Bucket", pluginFlag),
		"",
		"# Create a cluster-scoped resource",
		fmt.Sprintf("%%[1]s create api %s --group=network --version=v1alpha1 --kind=VPC \\", pluginFlag),
		"  --namespaced=false",
	}
	return m.FormatExamples(commandName, examples)
}