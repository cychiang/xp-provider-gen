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

package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/cychiang/xp-provider-gen/pkg/version"
)

// runRoot executes the root command with the given command line and returns
// what it wrote to its output.
func runRoot(cmdline string) (string, error) {
	var out bytes.Buffer
	root := newRootCommand()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(strings.Fields(cmdline))
	err := root.Execute()
	return out.String(), err
}

// TestUnknownCommandsFail pins that the Kubebuilder commands this generator
// never had are unknown commands, each with and without --help and through
// `help`, never a help page that succeeds.
func TestUnknownCommandsFail(t *testing.T) {
	for _, cmdline := range []string{
		"edit", "edit --help", "help edit",
		"alpha generate", "alpha generate --help", "help alpha generate",
		"create webhook", "create webhook --help", "help create webhook",
	} {
		t.Run(cmdline, func(t *testing.T) {
			out, err := runRoot(cmdline)
			if err == nil || !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("Execute = %v, want an unknown command error", err)
			}
			if strings.Contains(out, "Usage:") {
				t.Errorf("printed help:\n%s", out)
			}
		})
	}
}

// TestKeptCommandsHaveHelp pins that the commands still offered answer --help.
func TestKeptCommandsHaveHelp(t *testing.T) {
	for _, cmdline := range []string{
		"--help", "init --help", "create --help", "create api --help",
		"create-test --help", "update --help", "help create api",
	} {
		t.Run(cmdline, func(t *testing.T) {
			out, err := runRoot(cmdline)
			if err != nil || !strings.Contains(out, "Usage:") {
				t.Fatalf("Execute = %v, output %q, want usage and no error", err, out)
			}
		})
	}
}

// TestCreateRejectsUnknownFlag pins that a flag create does not have is named
// as a flag, not as an unknown command.
func TestCreateRejectsUnknownFlag(t *testing.T) {
	if _, err := runRoot("create --bogus"); err == nil || !strings.Contains(err.Error(), "unknown flag: --bogus") {
		t.Errorf("Execute = %v, want an unknown flag error", err)
	}
}

// TestRemovedFlagsFail pins that the Kubebuilder plugin-selection flags are gone.
func TestRemovedFlagsFail(t *testing.T) {
	for _, flag := range []string{"--plugins=go/v4", "--project-version=3"} {
		if _, err := runRoot("init " + flag); err == nil || !strings.Contains(err.Error(), "unknown flag") {
			t.Errorf("init %s: %v, want an unknown flag error", flag, err)
		}
	}
}

// TestVersionPrintsToStdout pins that version writes to the process's stdout,
// where test/e2e's versionOf reads it. It leaves the command's writers unset
// and redirects the real streams: cobra's cmd.Println falls back to stderr,
// which a SetOut buffer would hide.
func TestVersionPrintsToStdout(t *testing.T) {
	stdout, stderr := captureStreams(t, func() {
		root := newRootCommand()
		root.SetArgs([]string{"version"})
		if err := root.Execute(); err != nil {
			t.Errorf("Execute: %v", err)
		}
	})
	if want := version.Get().Short() + "\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// captureStreams runs fn with os.Stdout and os.Stderr redirected to pipes and
// returns what it wrote to each.
func captureStreams(t *testing.T, fn func()) (string, string) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = origOut, origErr }()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = outW, errW
	fn()
	os.Stdout, os.Stderr = origOut, origErr
	_ = outW.Close()
	_ = errW.Close()

	read := func(r *os.File) string {
		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	return read(outR), read(errR)
}
