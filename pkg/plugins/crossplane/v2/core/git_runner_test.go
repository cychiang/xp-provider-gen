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
	"strings"
	"testing"
)

// TestGitCommandRunner_DelegatesToCommandRunner pins C5/A7 together:
// GitCommandRunner used to duplicate CommandRunner's exec plumbing with its
// own silent cmd.Run()/cmd.Output() calls, so a failing git step also
// reported a bare exit status. Now that it delegates, git failures are
// self-diagnosing the same way make/go failures are.
func TestGitCommandRunner_DelegatesToCommandRunner(t *testing.T) {
	g := NewGitCommandRunner("")

	err := g.RunCommand(context.Background(), "--this-flag-does-not-exist")
	if err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("RunCommand() error = %v, want it to surface git's stderr", err)
	}

	_, err = g.RunCommandWithOutput(context.Background(), "--this-flag-does-not-exist")
	if err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("RunCommandWithOutput() error = %v, want it to surface git's stderr", err)
	}
}

// TestGitCommandRunner_RunCommandWithOutput_TrimsWhitespace guards the
// TrimSpace behavior callers (e.g. GetUserName) rely on.
func TestGitCommandRunner_RunCommandWithOutput_TrimsWhitespace(t *testing.T) {
	g := NewGitCommandRunner("")
	out, err := g.RunCommandWithOutput(context.Background(), "--version")
	if err != nil {
		t.Fatalf("RunCommandWithOutput(--version): %v", err)
	}
	if out != strings.TrimSpace(out) {
		t.Errorf("output not trimmed: %q", out)
	}
	if !strings.HasPrefix(out, "git version") {
		t.Errorf("unexpected output: %q", out)
	}
}
