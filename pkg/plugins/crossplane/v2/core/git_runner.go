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

package core

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitCommandRunner provides secure git command execution.
type GitCommandRunner struct {
	workDir string
}

// NewGitCommandRunner creates a new git command runner.
func NewGitCommandRunner(workDir string) *GitCommandRunner {
	return &GitCommandRunner{workDir: workDir}
}

// RunCommand executes a git command with the provided arguments.
func (g *GitCommandRunner) RunCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if g.workDir != "" {
		cmd.Dir = g.workDir
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command failed: %w", err)
	}
	return nil
}

// RunCommandWithOutput executes a git command and returns its output.
func (g *GitCommandRunner) RunCommandWithOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if g.workDir != "" {
		cmd.Dir = g.workDir
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// RunCommandWithStdin executes a git command with stdin input.
func (g *GitCommandRunner) RunCommandWithStdin(ctx context.Context, stdin string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if g.workDir != "" {
		cmd.Dir = g.workDir
	}
	cmd.Stdin = strings.NewReader(stdin)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command failed: %w", err)
	}
	return nil
}

// Init initializes a git repository.
func (g *GitCommandRunner) Init(ctx context.Context) error {
	return g.RunCommand(ctx, "init")
}

// Add stages files for commit.
func (g *GitCommandRunner) Add(ctx context.Context, files ...string) error {
	args := append([]string{"add"}, files...)
	return g.RunCommand(ctx, args...)
}

// Commit creates a commit with the provided message.
func (g *GitCommandRunner) Commit(ctx context.Context, message string) error {
	return g.RunCommandWithStdin(ctx, message, "commit", "-F", "-")
}

// GetUserName retrieves the git user.name from system config.
func (g *GitCommandRunner) GetUserName(ctx context.Context) (string, error) {
	return g.RunCommandWithOutput(ctx, "config", "--get", "user.name")
}

// GetUserEmail retrieves the git user.email from system config.
func (g *GitCommandRunner) GetUserEmail(ctx context.Context) (string, error) {
	return g.RunCommandWithOutput(ctx, "config", "--get", "user.email")
}

// GetSystemAuthor retrieves the system git author in "Name <email>" format.
func (g *GitCommandRunner) GetSystemAuthor(ctx context.Context) (string, error) {
	name, err := g.GetUserName(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get user name: %w", err)
	}

	email, err := g.GetUserEmail(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get user email: %w", err)
	}

	return fmt.Sprintf("%s <%s>", name, email), nil
}

// CommitWithSystemAuthor creates a commit using system git configuration.
func (g *GitCommandRunner) CommitWithSystemAuthor(ctx context.Context, message string) error {
	// Git will automatically use system config for author if not specified
	return g.RunCommandWithStdin(ctx, message, "commit", "-F", "-")
}

// CommitWithAuthor creates a commit with the provided message and author.
func (g *GitCommandRunner) CommitWithAuthor(ctx context.Context, message, author string) error {
	authorFlag := fmt.Sprintf("--author=%s", author)
	return g.RunCommandWithStdin(ctx, message, "commit", "-F", "-", authorFlag)
}

// AddSubmodule adds a git submodule.
func (g *GitCommandRunner) AddSubmodule(ctx context.Context, url, path string) error {
	return g.RunCommand(ctx, "submodule", "add", url, path)
}
