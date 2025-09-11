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

package v2

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PluginConfig holds centralized configuration and defaults for the Crossplane plugin
type PluginConfig struct {
	// Plugin metadata
	Name    string
	Version string
	
	// Default values for flags
	Defaults DefaultValues
	
	// Validation rules
	Validation ValidationRules
	
	// Git configuration
	Git GitConfig
	
	// Templates configuration
	Templates TemplateConfig
}

// DefaultValues holds default values for command flags
type DefaultValues struct {
	// Init command defaults
	Domain           string
	RepoPrefix       string
	Owner            string
	
	// CreateAPI command defaults
	GenerateClient   bool
	Force            bool
}

// ValidationRules holds validation configuration
type ValidationRules struct {
	// Currently no validation rules needed for simplified provider
}

// GitConfig holds git-related configuration
type GitConfig struct {
	BuildSubmoduleURL string
	DefaultAuthor     string
	DefaultEmail      string
}

// TemplateConfig holds template-related configuration
type TemplateConfig struct {
	GoVersion         string
	CrossplaneRuntime string
	KubernetesVersion string
}

// NewPluginConfig creates a new PluginConfig with sensible defaults
func NewPluginConfig() *PluginConfig {
	return &PluginConfig{
		Name:    pluginName,
		Version: "v1.0.0",
		
		Defaults: DefaultValues{
			Domain:           "",
			RepoPrefix:       "github.com/crossplane-contrib",
			Owner:            "",
			GenerateClient:   true,
			Force:            false,
		},
		
		Validation: ValidationRules{
			// No validation rules needed for simplified provider
		},
		
		Git: GitConfig{
			BuildSubmoduleURL: "https://github.com/crossplane/build",
			DefaultAuthor:     "Crossplane Provider Generator",
			DefaultEmail:      "noreply@crossplane.io",
		},
		
		Templates: TemplateConfig{
			GoVersion:         "1.24",
			CrossplaneRuntime: "v2.0.0",
			KubernetesVersion: "0.31.0",
		},
	}
}

// GenerateDefaultRepo creates a default repository name based on current directory
func (c *PluginConfig) GenerateDefaultRepo() string {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("%s/provider-example", c.Defaults.RepoPrefix)
	}
	
	// Get the directory name
	dirName := filepath.Base(wd)
	
	// Clean up the directory name to be valid for go modules
	dirName = strings.ToLower(dirName)
	dirName = strings.ReplaceAll(dirName, "_", "-")
	
	// If directory doesn't start with "provider-", add it
	if !strings.HasPrefix(dirName, "provider-") {
		if strings.HasPrefix(dirName, "crossplane-") {
			// Replace crossplane- with provider-
			dirName = strings.Replace(dirName, "crossplane-", "provider-", 1)
		} else {
			// Add provider- prefix
			dirName = "provider-" + dirName
		}
	}
	
	// Generate repository name following Crossplane convention
	return fmt.Sprintf("%s/%s", c.Defaults.RepoPrefix, dirName)
}

// Simplified validation - no provider type validation needed

// GetBoilerplate returns the standard license boilerplate
func (c *PluginConfig) GetBoilerplate() string {
	return `/*
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
*/`
}

// GetDefaultAuthor returns the default git commit author string
func (c *PluginConfig) GetDefaultAuthor() string {
	return fmt.Sprintf("%s <%s>", c.Git.DefaultAuthor, c.Git.DefaultEmail)
}

// GetFlagHelp returns standardized help text for flags
func (c *PluginConfig) GetFlagHelp() FlagHelp {
	return FlagHelp{
		Domain: "domain for API groups",
		Repo: fmt.Sprintf("name to use for go module (e.g., github.com/user/repo, defaults to %s/provider-<dirname>)", 
			c.Defaults.RepoPrefix),
		Owner:          "owner to add to the copyright",
		GenerateClient: "generate external client interface",
		Force:          "overwrite existing files if they exist",
	}
}

// FlagHelp holds help text for various flags
type FlagHelp struct {
	Domain         string
	Repo           string
	Owner          string
	GenerateClient string
	Force          string
}