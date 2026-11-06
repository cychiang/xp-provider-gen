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
	"strings"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// ScaffoldCommitTrailer marks the initial scaffold commit. While HEAD carries it
// (i.e. the user hasn't committed their own work yet), `create api` folds into
// that commit instead of adding a new one, so a freshly scaffolded provider has a
// single "Initial commit".
const ScaffoldCommitTrailer = "xp-provider-gen-scaffold: true"

// BuildSubmodulePath is where a provider keeps the crossplane/build submodule.
const BuildSubmodulePath = "build"

// gitlinkMode is the index mode of a submodule entry.
const gitlinkMode = "160000"

// noChangesMessage is printed when a commit step finds nothing staged, e.g.
// `create api` rerun against an already up-to-date scaffold. Skipping
// avoids git's "nothing to commit" failure and an empty commit in history.
const noChangesMessage = "No changes to commit; scaffold is already up to date."

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

func (g *GitOperations) CreateCommit(ctx context.Context, message string) error {
	changed, err := g.stageAndCheck(ctx)
	if err != nil || !changed {
		return err
	}
	return g.commit(ctx, message)
}

// commit creates a commit using the project's local git config (set during Init).
func (g *GitOperations) commit(ctx context.Context, message string) error {
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
func (g *GitOperations) CommitOrAmendScaffold(ctx context.Context, message string) error {
	changed, err := g.stageAndCheck(ctx)
	if err != nil || !changed {
		return err
	}
	if g.headIsScaffold(ctx) {
		// Keep the Initial commit's message and author; just add the new files.
		return g.runner.RunCommand(ctx, "commit", "--amend", "--no-edit")
	}
	return g.commit(ctx, message)
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

// stageAndCheck stages the working tree (`git add .`) and reports whether
// anything changed relative to HEAD. When nothing changed it prints
// noChangesMessage so callers can skip the commit instead of failing on git's
// "nothing to commit" — the case that broke `create api` reruns
// against an already up-to-date scaffold.
func (g *GitOperations) stageAndCheck(ctx context.Context) (bool, error) {
	if err := g.runner.Add(ctx, "."); err != nil {
		return false, err
	}
	out, err := g.runner.RunCommandWithOutput(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	if out == "" {
		fmt.Println(noChangesMessage)
		return false, nil
	}
	return true, nil
}

// AddSubmodule puts the submodule at path in place and pins it at ref, staging
// the gitlink there. The pin is staged, not left to the commit step, because
// `git submodule update` (what `make submodules` runs next) checks the
// submodule out at the index's recorded commit: left at the clone's HEAD, it
// would move the checkout straight back. It converges from any interrupted
// state, and a ref the submodule does not have fails naming the ref.
func (g *GitOperations) AddSubmodule(ctx context.Context, url, path, ref string) error {
	if err := g.ensureSubmoduleCheckout(ctx, url, path); err != nil {
		return err
	}
	if err := g.runner.RunCommand(ctx, "-C", path, "checkout", "--detach", ref); err != nil {
		return fmt.Errorf("checking out %s at build_ref %s: %w", path, ref, err)
	}
	return g.runner.Add(ctx, path)
}

// ensureSubmoduleCheckout leaves a submodule checkout at path, adding the
// submodule when path is absent and initializing it when only registered. A
// path that exists but cannot become a checkout fails, since running git
// there would act on the parent repository.
func (g *GitOperations) ensureSubmoduleCheckout(ctx context.Context, url, path string) error {
	if _, err := os.Stat(path + "/.git"); err == nil {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		if err := g.runner.AddSubmodule(ctx, url, path); err != nil {
			return err
		}
	}
	if err := g.runner.RunCommand(ctx, "submodule", "update", "--init", "--recursive"); err != nil {
		return err
	}
	if _, err := os.Stat(path + "/.git"); err != nil {
		return fmt.Errorf("%s exists but is not a git submodule checkout; remove or move it and rerun", path)
	}
	return nil
}

// BuildDrift reports how the build submodule's recorded commit differs from
// ref, or "" when it matches or the project has no such submodule. It compares
// the index's gitlink with ref and touches neither the submodule nor the
// network, so the commit need not exist locally.
func (g *GitOperations) BuildDrift(ctx context.Context, path, ref string) (string, error) {
	out, err := g.runner.RunCommandWithOutput(ctx, "ls-files", "--stage", "--", path)
	if err != nil {
		return "", err
	}
	mode, sha, ok := strings.Cut(out, " ")
	if !ok || mode != gitlinkMode {
		return "", nil
	}
	sha, _, _ = strings.Cut(sha, " ")
	if sha == ref {
		return "", nil
	}
	return fmt.Sprintf("%s is at %s; this generator pins %s. update does not move it: "+
		"git -C %s fetch origin %s && git -C %s checkout %s && git add %s",
		path, sha, ref, path, ref, path, ref, path), nil
}
