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

package automation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

type GitOperations struct {
	config *core.PluginConfig
}

func NewGitOperations(config *core.PluginConfig) *GitOperations {
	return &GitOperations{config: config}
}

func (g *GitOperations) Init(ctx context.Context) error {
	if _, err := os.Stat(".git"); err == nil {
		return nil
	}

	cmd := exec.CommandContext(ctx, "git", "init")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	return nil
}

func (g *GitOperations) CreateCommit(ctx context.Context, message, author string) error {
	cmd := exec.CommandContext(ctx, "git", "add", ".")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files to git: %w", err)
	}

	// Use -F - to read message from stdin instead of passing as argument
	authorFlag := fmt.Sprintf("--author=%s", author)
	cmd = exec.CommandContext(ctx, "git", "commit", "-F", "-", authorFlag)
	cmd.Stdin = strings.NewReader(message)
	if err := cmd.Run(); err != nil {
		// Fallback without author flag
		cmd = exec.CommandContext(ctx, "git", "commit", "-F", "-")
		cmd.Stdin = strings.NewReader(message)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create commit: %w", err)
		}
	}

	return nil
}

func (g *GitOperations) AddSubmodule(ctx context.Context, url, path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	cmd := exec.CommandContext(ctx, "git", "submodule", "add", url, path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add submodule: %w", err)
	}

	return nil
}
