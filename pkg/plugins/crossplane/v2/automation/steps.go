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

type Step interface {
	Name() string
	Execute() error
	IsRequired() bool
}

type GitInitStep struct {
	git      *GitOperations
	required bool
}

func NewGitInitStep(config *core.PluginConfig) *GitInitStep {
	return &GitInitStep{
		git:      NewGitOperations(config),
		required: false,
	}
}

func (s *GitInitStep) Name() string {
	return "Initialize git repository"
}

func (s *GitInitStep) Execute() error {
	return s.git.Init(context.Background())
}

func (s *GitInitStep) IsRequired() bool {
	return s.required
}

type GitCommitStep struct {
	git      *GitOperations
	message  string
	author   string
	required bool
}

func NewGitCommitStep(config *core.PluginConfig, message string) *GitCommitStep {
	return &GitCommitStep{
		git:      NewGitOperations(config),
		message:  message,
		author:   "", // Empty to use system git config, fallback to default in CreateCommit
		required: false,
	}
}

func (s *GitCommitStep) Name() string {
	return "Create initial commit"
}

func (s *GitCommitStep) Execute() error {
	return s.git.CreateCommit(context.Background(), s.message, s.author)
}

func (s *GitCommitStep) IsRequired() bool {
	return s.required
}

type GitSubmoduleStep struct {
	git      *GitOperations
	url      string
	path     string
	required bool
}

func NewGitSubmoduleStep(config *core.PluginConfig) *GitSubmoduleStep {
	return &GitSubmoduleStep{
		git:      NewGitOperations(config),
		url:      config.Git.BuildSubmoduleURL,
		path:     "build",
		required: false,
	}
}

func (s *GitSubmoduleStep) Name() string {
	return fmt.Sprintf("Add build submodule from %s", s.url)
}

func (s *GitSubmoduleStep) Execute() error {
	return s.git.AddSubmodule(context.Background(), s.url, s.path)
}

func (s *GitSubmoduleStep) IsRequired() bool {
	return s.required
}

type MakeStep struct {
	target   string
	required bool
}

func NewMakeStep(target string, required bool) *MakeStep {
	return &MakeStep{
		target:   target,
		required: required,
	}
}

func (s *MakeStep) Name() string {
	return fmt.Sprintf("Run make %s", s.target)
}

func (s *MakeStep) Execute() error {
	return core.NewCommandRunner("").Run(context.Background(), "make", s.target)
}

func (s *MakeStep) IsRequired() bool {
	return s.required
}

type GoModTidyStep struct {
	required bool
}

func NewGoModTidyStep(required bool) *GoModTidyStep {
	return &GoModTidyStep{
		required: required,
	}
}

func (s *GoModTidyStep) Name() string {
	return "Download dependencies (go mod tidy)"
}

func (s *GoModTidyStep) Execute() error {
	return core.NewCommandRunner("").Run(context.Background(), "go", "mod", "tidy")
}

func (s *GoModTidyStep) IsRequired() bool {
	return s.required
}
