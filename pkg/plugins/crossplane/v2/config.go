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

type PluginConfig struct {
	Name     string
	Version  string
	Defaults DefaultValues
	Git      GitConfig
}

type DefaultValues struct {
	Domain         string
	RepoPrefix     string
	Owner          string
	GenerateClient bool
	Force          bool
}

type GitConfig struct {
	BuildSubmoduleURL string
	DefaultAuthor     string
	DefaultEmail      string
}

func NewPluginConfig() *PluginConfig {
	return &PluginConfig{
		Name:    pluginName,
		Version: "v1.0.0",
		
		Defaults: DefaultValues{
			Domain:         "",
			RepoPrefix:     "github.com/crossplane-contrib",
			Owner:          "",
			GenerateClient: true,
			Force:          false,
		},
		
		Git: GitConfig{
			BuildSubmoduleURL: "https://github.com/crossplane/build",
			DefaultAuthor:     "Crossplane Provider Generator",
			DefaultEmail:      "noreply@crossplane.io",
		},
	}
}

func (c *PluginConfig) GenerateDefaultRepo() string {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Sprintf("%s/provider-example", c.Defaults.RepoPrefix)
	}
	
	dirName := filepath.Base(wd)
	dirName = strings.ToLower(dirName)
	dirName = strings.ReplaceAll(dirName, "_", "-")
	
	if !strings.HasPrefix(dirName, "provider-") {
		if strings.HasPrefix(dirName, "crossplane-") {
			dirName = strings.Replace(dirName, "crossplane-", "provider-", 1)
		} else {
			dirName = "provider-" + dirName
		}
	}
	
	return fmt.Sprintf("%s/%s", c.Defaults.RepoPrefix, dirName)
}

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

func (c *PluginConfig) GetDefaultAuthor() string {
	return fmt.Sprintf("%s <%s>", c.Git.DefaultAuthor, c.Git.DefaultEmail)
}