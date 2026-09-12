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

// TestCommandRunner_Run_ErrorSurfacesChildOutput pins A7: a failed init/create-api
// step used to report a bare "exit status N" with the child's diagnostics
// discarded, which cost a manual re-run to diagnose (e.g. a golangci-lint/Go
// version mismatch buried in "make reviewable" output). The error text must
// contain what the child actually printed.
func TestCommandRunner_Run_ErrorSurfacesChildOutput(t *testing.T) {
	err := NewCommandRunner("").Run(context.Background(), "git", "--this-flag-does-not-exist")
	if err == nil {
		t.Fatal("expected an error for an unknown git flag")
	}
	if !strings.Contains(err.Error(), "unknown option") {
		t.Errorf("error does not surface child output: %v", err)
	}
}

// TestCommandRunner_RunWithOutput_ErrorSurfacesStderr mirrors the above for
// RunWithOutput, whose stdout return value is used as data (e.g. `git status
// --porcelain`), so only stderr is folded into the error, not stdout.
func TestCommandRunner_RunWithOutput_ErrorSurfacesStderr(t *testing.T) {
	_, err := NewCommandRunner("").RunWithOutput(context.Background(), "git", "--this-flag-does-not-exist")
	if err == nil {
		t.Fatal("expected an error for an unknown git flag")
	}
	if !strings.Contains(err.Error(), "unknown option") {
		t.Errorf("error does not surface child stderr: %v", err)
	}
}

// TestCommandRunner_Run_RejectsDisallowedCommand pins the allowlist invariant
// CommandRunner and GitCommandRunner both rely on.
func TestCommandRunner_Run_RejectsDisallowedCommand(t *testing.T) {
	err := NewCommandRunner("").Run(context.Background(), "rm", "-rf", "/")
	if err == nil || !strings.Contains(err.Error(), "not an allowed command") {
		t.Fatalf("Run(\"rm\", ...) = %v, want a refusal naming the allowlist", err)
	}
}
