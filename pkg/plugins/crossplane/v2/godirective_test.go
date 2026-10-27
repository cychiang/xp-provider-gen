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

package v2

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"golang.org/x/mod/modfile"
)

// testGoVersion stands in for the manifest's go_version; lowerPatch is an
// older generator's directive, higherPatch and higherMinor ones an author set
// above it.
const (
	testGoVersion = "1.26.10"
	lowerPatch    = "1.26.8"
	higherPatch   = "1.26.11"
	higherMinor   = "1.27.0"
)

const goModRequires = `
require (
	github.com/crossplane/crossplane-runtime/v2 v2.4.3
	example.com/authors/own v1.2.3
)
`

type raiseGoDirectiveCase struct {
	name       string
	directive  string // "" writes no go line
	toolchain  string
	wantRaised bool
	wantFrom   string
	wantGo     string
}

func TestRaiseGoDirective(t *testing.T) {
	tests := []raiseGoDirectiveCase{
		{name: "lower patch is raised", directive: lowerPatch, wantRaised: true, wantFrom: lowerPatch, wantGo: testGoVersion},
		{name: "two-component directive is raised", directive: "1.26", wantRaised: true, wantFrom: "1.26", wantGo: testGoVersion},
		{name: "older minor is raised", directive: "1.25.3", wantRaised: true, wantFrom: "1.25.3", wantGo: testGoVersion},
		{name: "missing directive is raised", directive: "", wantRaised: true, wantFrom: "", wantGo: testGoVersion},
		{name: "equal is kept", directive: testGoVersion, wantRaised: false, wantFrom: testGoVersion, wantGo: testGoVersion},
		{name: "author's higher patch is kept", directive: higherPatch, wantRaised: false, wantFrom: higherPatch, wantGo: higherPatch},
		{name: "author's higher minor is kept", directive: higherMinor, wantRaised: false, wantFrom: higherMinor, wantGo: higherMinor},
		{
			name: "author's toolchain line survives a raise", directive: lowerPatch, toolchain: "go1.27.3",
			wantRaised: true, wantFrom: lowerPatch, wantGo: testGoVersion,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "module github.com/example/provider-x\n\n"
			if tt.directive != "" {
				content += "go " + tt.directive + "\n"
			}
			if tt.toolchain != "" {
				content += "\ntoolchain " + tt.toolchain + "\n"
			}
			content += goModRequires

			updated, from, raised, err := raiseGoDirective([]byte(content), testGoVersion)
			if err != nil {
				t.Fatalf("raiseGoDirective: %v", err)
			}
			if raised != tt.wantRaised || from != tt.wantFrom {
				t.Errorf("raised, from = %v, %q; want %v, %q", raised, from, tt.wantRaised, tt.wantFrom)
			}
			if !raised && string(updated) != content {
				t.Errorf("unraised content changed:\n%s", updated)
			}
			checkRaisedGoMod(t, tt, updated)
		})
	}
}

// checkRaisedGoMod asserts the go.mod raiseGoDirective returned: the wanted
// directive, the author's requires, and the toolchain line exactly as it was.
func checkRaisedGoMod(t *testing.T, tt raiseGoDirectiveCase, updated []byte) {
	t.Helper()
	f, err := modfile.Parse("go.mod", updated, nil)
	if err != nil {
		t.Fatalf("parsing result: %v\n%s", err, updated)
	}
	if f.Go == nil || f.Go.Version != tt.wantGo {
		t.Errorf("go directive = %v; want %s", f.Go, tt.wantGo)
	}
	if len(f.Require) != 2 {
		t.Errorf("requires = %d; want the author's 2 kept", len(f.Require))
	}
	if tt.toolchain != "" && (f.Toolchain == nil || f.Toolchain.Name != tt.toolchain) {
		t.Errorf("toolchain = %v; want %s kept", f.Toolchain, tt.toolchain)
	}
	if tt.toolchain == "" && f.Toolchain != nil {
		t.Errorf("a toolchain line was added: %s", f.Toolchain.Name)
	}
}

func TestRaiseGoDirective_RejectsUnparseableGoMod(t *testing.T) {
	_, _, _, err := raiseGoDirective([]byte("module\ngo not-a-version\n"), testGoVersion)
	if err == nil || !strings.Contains(err.Error(), "parsing go.mod") {
		t.Fatalf("err = %v; want a parsing go.mod error", err)
	}
}

func TestApplyGoDirective(t *testing.T) {
	t.Run("raises and writes", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		if err := afero.WriteFile(fs, "go.mod", []byte("module m\n\ngo 1.26.8\n"+goModRequires), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := applyGoDirective(fs, testGoVersion); err != nil {
			t.Fatalf("applyGoDirective: %v", err)
		}
		got, _ := afero.ReadFile(fs, "go.mod")
		if !strings.Contains(string(got), "\ngo 1.26.10\n") || !strings.Contains(string(got), "example.com/authors/own v1.2.3") {
			t.Errorf("go.mod after raise:\n%s", got)
		}
	})
	t.Run("leaves a higher directive byte-identical", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		original := "module m\n\ngo 1.27.0\n" + goModRequires
		if err := afero.WriteFile(fs, "go.mod", []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := applyGoDirective(fs, testGoVersion); err != nil {
			t.Fatalf("applyGoDirective: %v", err)
		}
		if got, _ := afero.ReadFile(fs, "go.mod"); string(got) != original {
			t.Errorf("go.mod changed:\n%s", got)
		}
	})
	t.Run("missing go.mod is an error", func(t *testing.T) {
		if err := applyGoDirective(afero.NewMemMapFs(), testGoVersion); err == nil || !strings.Contains(err.Error(), "reading go.mod") {
			t.Fatalf("err = %v; want a reading go.mod error", err)
		}
	})
}
