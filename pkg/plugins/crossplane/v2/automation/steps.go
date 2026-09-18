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
	"io/fs"
	"os"
	"strings"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// Step is a single unit of post-scaffold automation. Every step is required:
// a failure aborts the pipeline (see Pipeline.Run).
type Step interface {
	Name() string
	Execute() error
}

// stepNameInitialCommit is the display name of the commit step.
const stepNameInitialCommit = "Create initial commit"

type GitInitStep struct {
	git *GitOperations
}

func NewGitInitStep(config *core.PluginConfig) *GitInitStep {
	return &GitInitStep{git: NewGitOperations(config)}
}

func (s *GitInitStep) Name() string {
	return "Initialize git repository"
}

func (s *GitInitStep) Execute() error {
	return s.git.Init(context.Background())
}

// stepNameFoldCommit is the display name of the fold-commit step (see
// GitOperations.CommitOrAmendScaffold).
const stepNameFoldCommit = "Commit changes (fold into initial scaffold if applicable)"

// GitCommitStep commits pending changes: a plain commit, or, when fold is
// set, folded into the initial scaffold commit while the provider is still
// in initial setup (see GitOperations.CommitOrAmendScaffold).
type GitCommitStep struct {
	git     *GitOperations
	message string
	fold    bool
}

func NewGitCommitStep(config *core.PluginConfig, message string) *GitCommitStep {
	return &GitCommitStep{git: NewGitOperations(config), message: message}
}

// NewGitFoldCommitStep builds a commit step that folds into the initial
// scaffold commit while applicable.
func NewGitFoldCommitStep(config *core.PluginConfig, message string) *GitCommitStep {
	return &GitCommitStep{git: NewGitOperations(config), message: message, fold: true}
}

func (s *GitCommitStep) Name() string {
	if s.fold {
		return stepNameFoldCommit
	}
	return stepNameInitialCommit
}

func (s *GitCommitStep) Execute() error {
	if s.fold {
		return s.git.CommitOrAmendScaffold(context.Background(), s.message)
	}
	return s.git.CreateCommit(context.Background(), s.message)
}

type GitSubmoduleStep struct {
	git  *GitOperations
	url  string
	path string
}

func NewGitSubmoduleStep(config *core.PluginConfig) *GitSubmoduleStep {
	return &GitSubmoduleStep{
		git:  NewGitOperations(config),
		url:  config.Git.BuildSubmoduleURL,
		path: "build",
	}
}

func (s *GitSubmoduleStep) Name() string {
	return fmt.Sprintf("Add build submodule from %s", s.url)
}

func (s *GitSubmoduleStep) Execute() error {
	return s.git.AddSubmodule(context.Background(), s.url, s.path)
}

type MakeStep struct {
	target string
}

// ExecutableBitStep marks scaffolded shell scripts executable: kubebuilder's
// machinery writes every file 0644, but scripts are exec'd directly (uptest
// runs test/setup.sh), so the bit must be set — and committed — at scaffold
// time. It applies the same core.FileMode rule `update` uses when it writes
// files, rather than a per-flavor list of paths that would need editing
// whenever a scaffold gains a script.
type ExecutableBitStep struct {
	root string
}

// NewExecutableBitStep builds the chmod step for a scaffolded project root.
func NewExecutableBitStep(root string) *ExecutableBitStep {
	if root == "" {
		root = "."
	}
	return &ExecutableBitStep{root: root}
}

func (s *ExecutableBitStep) Name() string {
	return "Mark scaffolded scripts executable"
}

func (s *ExecutableBitStep) Execute() error {
	// os.Root confines every operation below to the project directory, so a
	// symlink planted mid-walk cannot redirect a chmod outside it.
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return fmt.Errorf("opening project root %s: %w", s.root, err)
	}
	defer root.Close()

	return fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if core.FileMode(path) != core.ScriptMode {
			return nil
		}
		if err := root.Chmod(path, core.ScriptMode); err != nil {
			return fmt.Errorf("chmod +x %s: %w", path, err)
		}
		return nil
	})
}

func NewMakeStep(target string) *MakeStep {
	return &MakeStep{target: target}
}

func (s *MakeStep) Name() string {
	return fmt.Sprintf("Run make %s", s.target)
}

func (s *MakeStep) Execute() error {
	return core.NewCommandRunner("").Run(context.Background(), "make", s.target)
}

// GoModDownloadStep populates go.sum for the declared dependencies. An upjet
// project cannot be tidied at init — cmd/provider imports packages `make
// generate` has not produced yet — but the generation tools still need their
// checksums, and download works from go.mod alone.
type GoModDownloadStep struct{}

// NewGoModDownloadStep builds the dependency download step.
func NewGoModDownloadStep() *GoModDownloadStep {
	return &GoModDownloadStep{}
}

func (s *GoModDownloadStep) Name() string {
	return "Download dependencies (go mod download)"
}

func (s *GoModDownloadStep) Execute() error {
	return core.NewCommandRunner("").Run(context.Background(), "go", "mod", "download")
}

type GoModTidyStep struct{}

func NewGoModTidyStep() *GoModTidyStep {
	return &GoModTidyStep{}
}

func (s *GoModTidyStep) Name() string {
	return "Tidy dependencies (go mod tidy)"
}

func (s *GoModTidyStep) Execute() error {
	return core.NewCommandRunner("").Run(context.Background(), "go", "mod", "tidy")
}

// StreamingCommandStep runs a command whose output the user watches live:
// update's finalize steps take minutes, and Run's buffered output would leave
// the terminal silent until they finish.
type StreamingCommandStep struct {
	name string
	args []string
}

// NewStreamingCommandStep builds a step that runs name with args, streaming its
// output to the terminal.
func NewStreamingCommandStep(name string, args ...string) *StreamingCommandStep {
	return &StreamingCommandStep{name: name, args: args}
}

func (s *StreamingCommandStep) Name() string {
	return "Run " + strings.Join(append([]string{s.name}, s.args...), " ")
}

func (s *StreamingCommandStep) Execute() error {
	return core.NewCommandRunner("").RunStreaming(context.Background(), os.Stdout, os.Stderr, s.name, s.args...)
}
