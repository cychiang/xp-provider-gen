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
)

// CommandRunner provides secure command execution.
type CommandRunner struct {
	workDir string
}

// NewCommandRunner creates a new command runner.
func NewCommandRunner(workDir string) *CommandRunner {
	return &CommandRunner{workDir: workDir}
}

// Run executes a command with the provided arguments.
func (c *CommandRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s command failed: %w", name, err)
	}
	return nil
}

// RunWithOutput executes a command and returns its output.
func (c *CommandRunner) RunWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s command failed: %w", name, err)
	}
	return string(output), nil
}
