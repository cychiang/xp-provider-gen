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
	"os"
	"strings"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// ScaffoldCommitTrailer marks the initial scaffold commit. While HEAD carries it
// (i.e. the user hasn't committed their own work yet), `create api` folds into
// that commit instead of adding a new one, so a freshly scaffolded provider has a
// single "Initial commit".
const ScaffoldCommitTrailer = "xp-provider-gen-scaffold: true"

type GitOperations struct {
	config *core.PluginConfig
	runner *core.GitCommandRunner
}

func NewGitOperations(config *core.PluginConfig) *GitOperations {
	return &GitOperations{
		config: config,
		runner: core.NewGitCommandRunner(""),
	}
}

func (g *GitOperations) Init(ctx context.Context) error {
	if _, err := os.Stat(".git"); err == nil {
		return nil
	}

	if err := g.runner.Init(ctx); err != nil {
		return err
	}

	// Set project-local git config for consistency
	return g.configureProjectGit(ctx)
}

// configureProjectGit sets git config in the project's .git/config.
func (g *GitOperations) configureProjectGit(ctx context.Context) error {
	name := g.config.Git.Author
	email := g.config.Git.Email

	if name != "" && email != "" {
		if err := g.runner.RunCommand(ctx, "config", "user.name", name); err != nil {
			return err
		}
		if err := g.runner.RunCommand(ctx, "config", "user.email", email); err != nil {
			return err
		}
	}
	return nil
}

func (g *GitOperations) CreateCommit(ctx context.Context, message, author string) error {
	if err := g.runner.Add(ctx, "."); err != nil {
		return err
	}

	// If explicit author provided via CLI, use it
	if author != "" {
		return g.runner.CommitWithAuthor(ctx, message, author)
	}

	// Use project's local git config (set during Init)
	return g.runner.CommitWithSystemAuthor(ctx, message)
}

// CommitOrAmendScaffold folds the staged change into the existing scaffold commit
// while HEAD still carries the scaffold trailer (the provider is still in initial
// setup); otherwise it creates a new commit. This keeps a freshly scaffolded
// provider at one "Initial commit" until the user makes their own commit.
//
// The fold rewrites HEAD via --amend and stages the whole working tree, so it
// assumes the intended workflow: scaffold the provider fully (init + create api)
// before pushing or making your own commit, and don't run create-api with
// unrelated unstaged edits you don't want folded into the Initial commit. Once
// you commit your own work, create-api stops folding and adds separate commits.
func (g *GitOperations) CommitOrAmendScaffold(ctx context.Context, message, author string) error {
	if err := g.runner.Add(ctx, "."); err != nil {
		return err
	}
	if g.headIsScaffold(ctx) {
		// Keep the Initial commit's message and author; just add the new files.
		return g.runner.RunCommand(ctx, "commit", "--amend", "--no-edit")
	}
	if author != "" {
		return g.runner.CommitWithAuthor(ctx, message, author)
	}
	return g.runner.CommitWithSystemAuthor(ctx, message)
}

// headIsScaffold reports whether the current HEAD commit is the tool's scaffold
// commit (carries the trailer and the user hasn't committed since).
func (g *GitOperations) headIsScaffold(ctx context.Context) bool {
	out, err := g.runner.RunCommandWithOutput(ctx, "log", "-1", "--pretty=%B")
	if err != nil {
		return false // no commits yet — make a normal commit
	}
	return strings.Contains(out, ScaffoldCommitTrailer)
}

func (g *GitOperations) AddSubmodule(ctx context.Context, url, path string) error {
	if _, err := os.Stat(path); err == nil {
		// Directory exists, check if it's a submodule
		if _, err := os.Stat(path + "/.git"); err == nil {
			return nil // Already initialized
		}
		// Directory exists but not initialized as submodule
		return g.runner.RunCommand(ctx, "submodule", "update", "--init", "--recursive")
	}

	// Add new submodule
	if err := g.runner.AddSubmodule(ctx, url, path); err != nil {
		return err
	}

	// Initialize the submodule
	return g.runner.RunCommand(ctx, "submodule", "update", "--init", "--recursive")
}

// CreateCommitWithSystemConfig creates a commit using only system git configuration.
func (g *GitOperations) CreateCommitWithSystemConfig(ctx context.Context, message string) error {
	if err := g.runner.Add(ctx, "."); err != nil {
		return err
	}

	return g.runner.CommitWithSystemAuthor(ctx, message)
}

// GetSystemAuthor retrieves the current git user configuration.
func (g *GitOperations) GetSystemAuthor(ctx context.Context) (string, error) {
	return g.runner.GetSystemAuthor(ctx)
}
