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

type GitCommitStep struct {
	git     *GitOperations
	message string
	author  string
}

func NewGitCommitStep(config *core.PluginConfig, message string) *GitCommitStep {
	return &GitCommitStep{
		git:     NewGitOperations(config),
		message: message,
		author:  "", // Empty to use system git config, fallback to default in CreateCommit
	}
}

func (s *GitCommitStep) Name() string {
	return stepNameInitialCommit
}

func (s *GitCommitStep) Execute() error {
	return s.git.CreateCommit(context.Background(), s.message, s.author)
}

// GitFoldCommitStep commits, folding into the initial scaffold commit while the
// provider is still in initial setup (see GitOperations.CommitOrAmendScaffold).
type GitFoldCommitStep struct {
	git     *GitOperations
	message string
	author  string
}

func NewGitFoldCommitStep(config *core.PluginConfig, message string) *GitFoldCommitStep {
	return &GitFoldCommitStep{
		git:     NewGitOperations(config),
		message: message,
		author:  "",
	}
}

func (s *GitFoldCommitStep) Name() string {
	return "Commit changes (fold into initial scaffold if applicable)"
}

func (s *GitFoldCommitStep) Execute() error {
	return s.git.CommitOrAmendScaffold(context.Background(), s.message, s.author)
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

func NewMakeStep(target string) *MakeStep {
	return &MakeStep{target: target}
}

func (s *MakeStep) Name() string {
	return fmt.Sprintf("Run make %s", s.target)
}

func (s *MakeStep) Execute() error {
	return core.NewCommandRunner("").Run(context.Background(), "make", s.target)
}

type GoModTidyStep struct{}

func NewGoModTidyStep() *GoModTidyStep {
	return &GoModTidyStep{}
}

func (s *GoModTidyStep) Name() string {
	return "Download dependencies (go mod tidy)"
}

func (s *GoModTidyStep) Execute() error {
	return core.NewCommandRunner("").Run(context.Background(), "go", "mod", "tidy")
}
