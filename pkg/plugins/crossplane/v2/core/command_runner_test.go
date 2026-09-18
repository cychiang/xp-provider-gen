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
	"strings"
	"testing"
)

// TestCommandRunner_Run_ErrorSurfacesChildOutput pins that a failed
// init/create-api step's error text contains what the child actually
// printed (e.g. a golangci-lint/Go version mismatch buried in "make
// reviewable" output), not a bare "exit status N" that costs a manual
// re-run to diagnose.
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

// TestCommandRunner_RunStreaming_RejectsDisallowedCommand pins that the
// streaming path, which update's long-running steps use, enforces the same
// allowlist as Run.
func TestCommandRunner_RunStreaming_RejectsDisallowedCommand(t *testing.T) {
	var out bytes.Buffer
	err := NewCommandRunner("").RunStreaming(context.Background(), &out, &out, "sh", "-c", "true")
	if err == nil || !strings.Contains(err.Error(), "not an allowed command") {
		t.Fatalf("RunStreaming(\"sh\", ...) = %v, want a refusal naming the allowlist", err)
	}
}

// TestCommandRunner_RunStreaming_StreamsOutput pins that the child's output
// reaches the caller's writer rather than being buffered away.
func TestCommandRunner_RunStreaming_StreamsOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := NewCommandRunner("").RunStreaming(context.Background(), &stdout, &stderr, "go", "version"); err != nil {
		t.Fatalf("RunStreaming(go version): %v", err)
	}
	if !strings.Contains(stdout.String(), "go version") {
		t.Errorf("stdout = %q, want it to contain %q", stdout.String(), "go version")
	}
}

// TestCommandRunner_RunStreaming_ErrorFormat pins that RunStreaming's error
// names nothing itself: callers already name the command (Pipeline.Run
// prefixes the step name, update names the module), so repeating it here
// produced "Run make reviewable failed: make reviewable: exit status 2".
func TestCommandRunner_RunStreaming_ErrorFormat(t *testing.T) {
	tests := []struct {
		desc string
		args []string
		want string
	}{
		{desc: "with arguments", args: []string{"bogus-subcommand"}, want: "exit status 2"},
		{desc: "without arguments", want: "exit status 2"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var out bytes.Buffer
			err := NewCommandRunner("").RunStreaming(context.Background(), &out, &out, "go", tt.args...)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("RunStreaming(go %v) error = %q, want %q", tt.args, err, tt.want)
			}
		})
	}
}
