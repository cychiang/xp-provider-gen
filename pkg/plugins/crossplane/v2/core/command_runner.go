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
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// allowedCommands are the only executables this tool may spawn. The generator
// never runs user-supplied programs; keeping the set closed makes that an
// enforced invariant rather than a convention, and makes the #nosec G204
// annotations below provably true.
var allowedCommands = map[string]bool{"git": true, "go": true, "make": true}

// checkCommand rejects any executable outside allowedCommands.
func checkCommand(name string) error {
	if !allowedCommands[name] {
		return fmt.Errorf("refusing to run %q: not an allowed command", name)
	}
	return nil
}

// CommandRunner provides secure command execution.
type CommandRunner struct {
	workDir string
}

// NewCommandRunner creates a new command runner.
func NewCommandRunner(workDir string) *CommandRunner {
	return &CommandRunner{workDir: workDir}
}

// Run executes a command with the provided arguments. On failure, the child's
// combined stdout/stderr is attached to the returned error: without it, every
// caller sees only "exit status N" and has to re-run the command by hand to
// find out why (e.g. a golangci-lint/Go version mismatch buried in "make
// reviewable" output).
func (c *CommandRunner) Run(ctx context.Context, name string, args ...string) error {
	if err := checkCommand(name); err != nil {
		return err
	}
	// No shell is involved and name is allowlisted above; args are literals or
	// repo-controlled data (make targets, dependency coordinates).
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- allowlisted command, no shell
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s command failed: %w\n%s", name, err, output)
	}
	return nil
}

// RunWithOutput executes a command and returns its stdout. On failure the
// child's stderr is attached to the returned error for the same reason as Run.
func (c *CommandRunner) RunWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	if err := checkCommand(name); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- allowlisted command, no shell
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s command failed: %w\n%s", name, err, stderr.Bytes())
	}
	return string(output), nil
}

// RunWithStdin executes a command, feeding it stdin. On failure the child's
// combined stdout/stderr is attached to the returned error for the same
// reason as Run.
func (c *CommandRunner) RunWithStdin(ctx context.Context, stdin, name string, args ...string) error {
	if err := checkCommand(name); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- allowlisted command, no shell
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}
	cmd.Stdin = strings.NewReader(stdin)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s command failed: %w\n%s", name, err, output)
	}
	return nil
}
