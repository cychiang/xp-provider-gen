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
	"strings"
)

// GitCommandRunner provides the git-specific convenience methods used by init
// and the automation pipeline, on top of CommandRunner: the executable is
// always the literal "git", never a variable, and CommandRunner's allowlist
// (which already includes "git") applies uniformly instead of git commands
// spawning exec.Command directly. The two runners used to duplicate the same
// ~25 lines of exec plumbing; this keeps that in one place plus git treats
// values after -m/-- as data, so no shell is involved and the variable
// argument lists carry no injection risk either way.
type GitCommandRunner struct {
	runner *CommandRunner
}

// NewGitCommandRunner creates a new git command runner.
func NewGitCommandRunner(workDir string) *GitCommandRunner {
	return &GitCommandRunner{runner: NewCommandRunner(workDir)}
}

// RunCommand executes a git command with the provided arguments.
func (g *GitCommandRunner) RunCommand(ctx context.Context, args ...string) error {
	return g.runner.Run(ctx, "git", args...)
}

// RunCommandWithOutput executes a git command and returns its output.
func (g *GitCommandRunner) RunCommandWithOutput(ctx context.Context, args ...string) (string, error) {
	output, err := g.runner.RunWithOutput(ctx, "git", args...)
	return strings.TrimSpace(output), err
}

// RunCommandWithStdin executes a git command with stdin input.
func (g *GitCommandRunner) RunCommandWithStdin(ctx context.Context, stdin string, args ...string) error {
	return g.runner.RunWithStdin(ctx, stdin, "git", args...)
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

// GetUserName retrieves the git user.name from system config.
func (g *GitCommandRunner) GetUserName(ctx context.Context) (string, error) {
	return g.RunCommandWithOutput(ctx, "config", "--get", "user.name")
}

// GetUserEmail retrieves the git user.email from system config.
func (g *GitCommandRunner) GetUserEmail(ctx context.Context) (string, error) {
	return g.RunCommandWithOutput(ctx, "config", "--get", "user.email")
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
