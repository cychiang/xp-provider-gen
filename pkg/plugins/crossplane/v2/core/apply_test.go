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
	"io/fs"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

// TestApplyDecisionTable pins Apply's write decision for every on-disk x
// rendered combination, with seedUserOwned both ways, and the mode of whatever
// lands on disk. The path is a script so a seeded file proves the executable
// bit comes from the write layer; an overwritten file keeps the mode it had.
func TestApplyDecisionTable(t *testing.T) {
	const (
		rel        = "test/setup.sh"
		headered   = GeneratedHeader + "\n#!/bin/sh\n# new\n"
		oldHeader  = GeneratedHeader + "\n#!/bin/sh\n# old\n"
		headerless = "#!/bin/sh\n# user\n"
		rendered   = "#!/bin/sh\n# stub\n"
		existingFM = fs.FileMode(0o600) // distinguishes "kept" from "rewritten"
	)
	tests := []struct {
		name     string
		onDisk   *string // nil: absent
		render   string
		seedUser bool
		want     outcome
		wantBody string // "" : absent on disk
		wantMode fs.FileMode
	}{
		{name: "absent, rendered with header, seed", render: headered, seedUser: true, want: seeded, wantBody: headered, wantMode: FileMode(rel)},
		{name: "absent, rendered with header, no seed", render: headered, seedUser: false, want: seeded, wantBody: headered, wantMode: FileMode(rel)},
		{name: "absent, rendered headerless, seed", render: rendered, seedUser: true, want: seeded, wantBody: rendered, wantMode: FileMode(rel)},
		{name: "absent, rendered headerless, no seed", render: rendered, seedUser: false, want: unseeded},
		{name: "exists with header, seed", onDisk: ptr(oldHeader), render: headered, seedUser: true, want: overwritten, wantBody: headered, wantMode: existingFM},
		{name: "exists with header, no seed", onDisk: ptr(oldHeader), render: headered, seedUser: false, want: overwritten, wantBody: headered, wantMode: existingFM},
		{name: "exists with header, identical content, seed", onDisk: ptr(headered), render: headered, seedUser: true, want: unchanged, wantBody: headered, wantMode: existingFM},
		{name: "exists with header, identical content, no seed", onDisk: ptr(headered), render: headered, seedUser: false, want: unchanged, wantBody: headered, wantMode: existingFM},
		{name: "exists headerless, seed", onDisk: ptr(headerless), render: headered, seedUser: true, want: skipped, wantBody: headerless, wantMode: existingFM},
		{name: "exists headerless, no seed", onDisk: ptr(headerless), render: headered, seedUser: false, want: skipped, wantBody: headerless, wantMode: existingFM},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, mem := afero.NewMemMapFs(), afero.NewMemMapFs()
			if err := afero.WriteFile(src, rel, []byte(tt.render), 0o644); err != nil {
				t.Fatal(err)
			}
			if tt.onDisk != nil {
				if err := afero.WriteFile(mem, rel, []byte(*tt.onDisk), existingFM); err != nil {
					t.Fatal(err)
				}
			}
			dst := &writeCountingFs{Fs: mem}

			result, err := Apply(src, dst, tt.seedUser)
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}

			assertOutcome(t, result, rel, tt.want)
			wantWrite := tt.want == seeded || tt.want == overwritten
			if (dst.writes > 0) != wantWrite {
				t.Errorf("write opens = %d, want a write: %v", dst.writes, wantWrite)
			}

			info, statErr := dst.Stat(rel)
			if tt.wantBody == "" {
				if statErr == nil {
					t.Fatalf("%s was written, want absent", rel)
				}
				return
			}
			if statErr != nil {
				t.Fatalf("stat %s: %v", rel, statErr)
			}
			if got, _ := afero.ReadFile(dst, rel); string(got) != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
			if info.Mode().Perm() != tt.wantMode {
				t.Errorf("mode = %v, want %v", info.Mode().Perm(), tt.wantMode)
			}
		})
	}
}

// writeCountingFs counts the opens that can write, so a test can prove Apply
// did not touch a file at all (identical bytes would hide a rewrite).
type writeCountingFs struct {
	afero.Fs
	writes int
}

func (f *writeCountingFs) OpenFile(name string, flag int, perm fs.FileMode) (afero.File, error) {
	if flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_APPEND) != 0 {
		f.writes++
	}
	return f.Fs.OpenFile(name, flag, perm)
}

func ptr[T any](v T) *T { return &v }

// TestApply_UpjetDoesNotSeedUserOwned pins that a user-owned file missing
// on disk is seeded for a native project, but not for an upjet one:
// some upjet user-owned templates need init-time Terraform settings PROJECT
// does not keep, so seeding would write empty values, and none are recreated.
// Tool-owned files are seeded either way.
func TestApply_UpjetDoesNotSeedUserOwned(t *testing.T) {
	const (
		headeredPath   = "config/provider.go"
		headerlessPath = "examples/providerconfig/providerconfig.yaml"
	)
	tests := []struct {
		name         string
		seed         bool
		wantSeeded   []string
		wantUnseeded []string
	}{
		{name: string(FlavorUpjet), seed: false, wantSeeded: []string{headeredPath}, wantUnseeded: []string{headerlessPath}},
		{name: string(FlavorNative), seed: true, wantSeeded: []string{headeredPath, headerlessPath}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := afero.NewMemMapFs(), afero.NewMemMapFs()
			_ = afero.WriteFile(src, headeredPath, []byte(GeneratedHeader+"\npackage config\n"), 0o644)
			_ = afero.WriteFile(src, headerlessPath, []byte("kind: ProviderConfig\n"), 0o644)

			result, err := Apply(src, dst, tt.seed)
			if err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if !slices.Equal(result.Seeded, tt.wantSeeded) {
				t.Errorf("seeded = %v, want %v", result.Seeded, tt.wantSeeded)
			}
			if !slices.Equal(result.Unseeded, tt.wantUnseeded) {
				t.Errorf("unseeded = %v, want %v", result.Unseeded, tt.wantUnseeded)
			}
			for _, p := range tt.wantUnseeded {
				if ok, _ := afero.Exists(dst, p); ok {
					t.Errorf("%s was written despite not being seeded", p)
				}
			}
		})
	}
}

func TestApply(t *testing.T) {
	const headered = GeneratedHeader + "\npackage foo\n// new\n"
	const oldHeadered = GeneratedHeader + "\npackage foo\n// old\n"
	const userEdited = "package foo\n// my hand-written logic\n"

	src := afero.NewMemMapFs()
	dst := afero.NewMemMapFs()

	// Tool-owned file present on disk (old) -> overwritten.
	_ = afero.WriteFile(src, "internal/controller/mytype/setup.go", []byte(headered), 0o644)
	_ = afero.WriteFile(dst, "internal/controller/mytype/setup.go", []byte(oldHeadered), 0o644)

	// User-owned file present on disk (edited) -> skipped (the render is just a stub).
	_ = afero.WriteFile(src, "internal/controller/mytype/controller.go", []byte("package foo\n// stub\n"), 0o644)
	_ = afero.WriteFile(dst, "internal/controller/mytype/controller.go", []byte(userEdited), 0o644)

	// New tool-owned file absent on disk -> seeded.
	_ = afero.WriteFile(src, apisRegisterPath, []byte(headered), 0o644)

	result, err := Apply(src, dst, true)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, _ := afero.ReadFile(dst, "internal/controller/mytype/setup.go")
	if string(got) != headered {
		t.Errorf("tool-owned setup.go = %q, want overwritten with new content", got)
	}
	got, _ = afero.ReadFile(dst, "internal/controller/mytype/controller.go")
	if string(got) != userEdited {
		t.Errorf("user-owned controller.go = %q, want preserved", got)
	}
	got, _ = afero.ReadFile(dst, apisRegisterPath)
	if string(got) != headered {
		t.Errorf("new register.go = %q, want seeded", got)
	}

	assertContains(t, "overwritten", result.Overwritten, "internal/controller/mytype/setup.go")
	assertContains(t, "skipped", result.Skipped, "internal/controller/mytype/controller.go")
	assertContains(t, "seeded", result.Seeded, apisRegisterPath)
}

// TestApply_NestedSeed verifies a new file in a directory that does not yet
// exist on disk is created (MkdirAll path).
func TestApply_NestedSeed(t *testing.T) {
	src := afero.NewMemMapFs()
	dst := afero.NewMemMapFs()
	content := GeneratedHeader + "\npackage v1\n"
	_ = afero.WriteFile(src, "apis/newgroup/v1/groupversion_info.go", []byte(content), 0o644)

	if _, err := Apply(src, dst, true); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	got, err := afero.ReadFile(dst, "apis/newgroup/v1/groupversion_info.go")
	if err != nil || string(got) != content {
		t.Errorf("nested seed = %q (err %v), want the rendered content", got, err)
	}
}

// TestApplyFileRefusesEscapes pins the containment gate: rendered paths derive
// from PROJECT (group/version/kind are substituted into template paths), so a
// hand-edited or attacker-supplied PROJECT must never make `update` write
// outside the project directory.
func TestApplyFileRefusesEscapes(t *testing.T) {
	escapes := []string{
		"../escape.txt",
		"../../../../tmp/escape.txt",
		"/tmp/absolute-escape.txt",
		"apis/../../escape.txt",
		"", // an empty path must not resolve to the working directory
	}
	if runtime.GOOS == "windows" {
		// Backslash is a separator on Windows only; elsewhere these are just
		// unusual file names and are legitimately allowed.
		escapes = append(escapes,
			`..\escape.txt`,
			`C:\absolute-escape.txt`,
			`apis\..\..\escape.txt`,
		)
	}
	for _, rel := range escapes {
		t.Run(rel, func(t *testing.T) {
			src, dst := afero.NewMemMapFs(), afero.NewMemMapFs()
			if err := afero.WriteFile(src, "rendered", []byte("payload"), 0o644); err != nil {
				t.Fatalf("seeding src: %v", err)
			}

			decision, err := applyFile(src, dst, "rendered", rel, true)
			if err == nil {
				t.Fatalf("applyFile(%q) was allowed; want refusal", rel)
			}
			if !strings.Contains(err.Error(), "refusing to write outside the project") {
				t.Fatalf("unexpected error for %q: %v", rel, err)
			}
			if decision != Skip {
				t.Errorf("decision = %v, want Skip", decision)
			}
			// Skip for the empty path: it resolves to the filesystem root,
			// which always exists and says nothing about a write.
			if rel != "" {
				if exists, _ := afero.Exists(dst, rel); exists {
					t.Errorf("%q was written despite refusal", rel)
				}
			}
		})
	}
}

// TestApplyFileAllowsProjectPaths guards against the check being so strict it
// breaks normal rendering.
func TestApplyFileAllowsProjectPaths(t *testing.T) {
	src, dst := afero.NewMemMapFs(), afero.NewMemMapFs()
	if err := afero.WriteFile(src, "rendered", []byte("payload"), 0o644); err != nil {
		t.Fatalf("seeding src: %v", err)
	}
	for _, rel := range []string{"go.mod", apisRegisterPath, "internal/controller/thing/wiring.go"} {
		if _, err := applyFile(src, dst, "rendered", rel, true); err != nil {
			t.Fatalf("applyFile(%q) refused a legitimate path: %v", rel, err)
		}
		if exists, _ := afero.Exists(dst, rel); !exists {
			t.Errorf("%q was not written", rel)
		}
	}
}

const apisRegisterPath = "apis/register.go"

func assertContains(t *testing.T, label string, list []string, want string) {
	t.Helper()
	if !slices.Contains(list, want) {
		t.Errorf("%s = %v, want it to contain %q", label, list, want)
	}
}

type outcome int

const (
	seeded outcome = iota
	overwritten
	skipped
	unseeded
	unchanged
)

// outcomeNames labels each outcome in failure messages.
var outcomeNames = map[outcome]string{
	seeded: "seeded", overwritten: "overwritten", skipped: "skipped", unseeded: "unseeded", unchanged: "unchanged",
}

// assertOutcome checks rel is in exactly the one result list want names.
func assertOutcome(t *testing.T, result ApplyResult, rel string, want outcome) {
	t.Helper()
	lists := map[outcome][]string{
		seeded: result.Seeded, overwritten: result.Overwritten,
		skipped: result.Skipped, unseeded: result.Unseeded, unchanged: result.Unchanged,
	}
	for o, list := range lists {
		wantLen := 0
		if o == want {
			wantLen = 1
		}
		if len(list) != wantLen || (wantLen == 1 && list[0] != rel) {
			t.Errorf("%s list = %v, want %d entry for %s", outcomeNames[o], list, wantLen, rel)
		}
	}
}
