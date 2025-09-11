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
	"os/exec"
)

type GitUtils struct {
	config *PluginConfig
}

func NewGitUtils(config *PluginConfig) *GitUtils {
	return &GitUtils{config: config}
}

func (g *GitUtils) InitRepo() error {
	if _, err := os.Stat(".git"); err == nil {
		return nil
	}
	
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}
	
	return nil
}

func (g *GitUtils) CreateInitialCommit() error {
	cmd := exec.Command("git", "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files to git: %w", err)
	}
	
	commitMsg := "Initial commit\n\nScaffolded Crossplane provider project"
	authorFlag := fmt.Sprintf("--author=%s", g.config.GetDefaultAuthor())
	
	cmd = exec.Command("git", "commit", "-m", commitMsg, authorFlag)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "commit", "-m", commitMsg)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create initial commit: %w", err)
		}
	}
	
	return nil
}

func (g *GitUtils) AddBuildSubmodule() error {
	if _, err := os.Stat("build"); err == nil {
		return nil
	}
	
	cmd := exec.Command("git", "submodule", "add", g.config.Git.BuildSubmoduleURL, "build")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add build submodule: %w", err)
	}
	
	fmt.Printf("Added build submodule from %s\n", g.config.Git.BuildSubmoduleURL)
	return nil
}

