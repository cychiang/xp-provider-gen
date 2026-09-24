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

// Tests for reconcile.go: the ownership-gated copy, orphan removal, tracked
// file listing, and the result-printing summary.
package v2

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// TestReconcile_UpjetDoesNotSeedUserOwned pins that a user-owned file missing
// on disk is seeded for a native project, but not for an upjet one:
// some upjet user-owned templates need init-time Terraform settings PROJECT
// does not keep, so seeding would write empty values, and none are recreated.
// Tool-owned files are seeded either way.
func TestReconcile_UpjetDoesNotSeedUserOwned(t *testing.T) {
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
		{name: string(core.FlavorUpjet), seed: false, wantSeeded: []string{headeredPath}, wantUnseeded: []string{headerlessPath}},
		{name: string(core.FlavorNative), seed: true, wantSeeded: []string{headeredPath, headerlessPath}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := afero.NewMemMapFs(), afero.NewMemMapFs()
			_ = afero.WriteFile(src, headeredPath, []byte(core.GeneratedHeader+"\npackage config\n"), 0o644)
			_ = afero.WriteFile(src, headerlessPath, []byte("kind: ProviderConfig\n"), 0o644)

			result, err := reconcile(src, dst, tt.seed)
			if err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			if !slices.Equal(result.seeded, tt.wantSeeded) {
				t.Errorf("seeded = %v, want %v", result.seeded, tt.wantSeeded)
			}
			if !slices.Equal(result.unseeded, tt.wantUnseeded) {
				t.Errorf("unseeded = %v, want %v", result.unseeded, tt.wantUnseeded)
			}
			for _, p := range tt.wantUnseeded {
				if ok, _ := afero.Exists(dst, p); ok {
					t.Errorf("%s was written despite not being seeded", p)
				}
			}
		})
	}
}

func TestReconcile(t *testing.T) {
	const headered = core.GeneratedHeader + "\npackage foo\n// new\n"
	const oldHeadered = core.GeneratedHeader + "\npackage foo\n// old\n"
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
	_ = afero.WriteFile(src, "apis/register.go", []byte(headered), 0o644)

	result, err := reconcile(src, dst, true)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	got, _ := afero.ReadFile(dst, "internal/controller/mytype/setup.go")
	if string(got) != headered {
		t.Errorf("tool-owned setup.go = %q, want overwritten with new content", got)
	}
	got, _ = afero.ReadFile(dst, "internal/controller/mytype/controller.go")
	if string(got) != userEdited {
		t.Errorf("user-owned controller.go = %q, want preserved", got)
	}
	got, _ = afero.ReadFile(dst, "apis/register.go")
	if string(got) != headered {
		t.Errorf("new register.go = %q, want seeded", got)
	}

	assertContains(t, "overwritten", result.overwritten, "internal/controller/mytype/setup.go")
	assertContains(t, "skipped", result.skipped, "internal/controller/mytype/controller.go")
	assertContains(t, "seeded", result.seeded, "apis/register.go")
}

// TestReconcile_NestedSeed verifies a new file in a directory that does not yet
// exist on disk is created (MkdirAll path).
func TestReconcile_NestedSeed(t *testing.T) {
	src := afero.NewMemMapFs()
	dst := afero.NewMemMapFs()
	content := core.GeneratedHeader + "\npackage v1\n"
	_ = afero.WriteFile(src, "apis/newgroup/v1/groupversion_info.go", []byte(content), 0o644)

	if _, err := reconcile(src, dst, true); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got, err := afero.ReadFile(dst, "apis/newgroup/v1/groupversion_info.go")
	if err != nil || string(got) != content {
		t.Errorf("nested seed = %q (err %v), want the rendered content", got, err)
	}
}

// Paths TestRemoveOrphans, TestTrackedFiles and TestReconcileResultPrint key
// off.
const (
	orphanFileA    = "a.go"
	orphanFileB    = "b.go"
	orphanFileUser = "u.go"
	orphanDir      = "build" // stands in for a submodule gitlink entry
)

// removeOrphansCase is one TestRemoveOrphans table row.
type removeOrphansCase struct {
	name        string
	tracked     []string
	srcFiles    map[string][]byte
	dstFiles    map[string][]byte
	dstDirs     []string
	wantRemoved []string
	wantErr     bool
	wantPresent []string
}

// TestRemoveOrphans pins removeOrphans's deletion gate. dst is always a real
// on-disk filesystem (afero.NewBasePathFs over t.TempDir()), never MemMapFs:
// MemMapFs returns a nil error when reading a directory, so the
// submodule-gitlink case below would pass even against a wrong
// implementation that never checks fi.IsDir(). src may be MemMapFs — it is
// only ever read.
func TestRemoveOrphans(t *testing.T) {
	headered := []byte(core.GeneratedHeader + "\npackage foo\n")
	headerless := []byte("package foo\n// mine\n")

	tests := []removeOrphansCase{
		{
			name:        "removes an orphaned tool-owned file",
			tracked:     []string{orphanFileA},
			dstFiles:    map[string][]byte{orphanFileA: headered},
			wantRemoved: []string{orphanFileA},
		},
		{
			name:        "keeps a rendered tool-owned file",
			tracked:     []string{orphanFileA},
			srcFiles:    map[string][]byte{orphanFileA: headered},
			dstFiles:    map[string][]byte{orphanFileA: headered},
			wantPresent: []string{orphanFileA},
		},
		{
			name:        "keeps a user-owned file",
			tracked:     []string{orphanFileUser},
			dstFiles:    map[string][]byte{orphanFileUser: headerless},
			wantPresent: []string{orphanFileUser},
		},
		{
			name:    "ignores a tracked file missing on disk",
			tracked: []string{"gone.go"},
		},
		{
			name:    "skips a tracked directory entry (submodule gitlink)",
			tracked: []string{orphanDir},
			dstDirs: []string{orphanDir},
		},
		{
			name:    "rejects a path outside the project",
			tracked: []string{"../x.go"},
			wantErr: true,
		},
		{
			name:        "removes several, in the order given",
			tracked:     []string{orphanFileB, orphanFileA},
			dstFiles:    map[string][]byte{orphanFileB: headered, orphanFileA: headered},
			wantRemoved: []string{orphanFileB, orphanFileA},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { runRemoveOrphansCase(t, tt) })
	}
}

// runRemoveOrphansCase seeds dst (a real on-disk FS) and src (in-memory) per
// tt, calls removeOrphans, and checks its return value plus dst's resulting
// state. Split out of TestRemoveOrphans to keep both functions' cyclomatic
// complexity low.
func runRemoveOrphansCase(t *testing.T, tt removeOrphansCase) {
	t.Helper()
	dst := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	for _, dir := range tt.dstDirs {
		if err := dst.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("seeding dst dir %s: %v", dir, err)
		}
	}
	for path, content := range tt.dstFiles {
		if err := afero.WriteFile(dst, path, content, 0o644); err != nil {
			t.Fatalf("seeding dst file %s: %v", path, err)
		}
	}
	src := afero.NewMemMapFs()
	for path, content := range tt.srcFiles {
		if err := afero.WriteFile(src, path, content, 0o644); err != nil {
			t.Fatalf("seeding src file %s: %v", path, err)
		}
	}

	got, err := removeOrphans(tt.tracked, src, dst)
	if tt.wantErr {
		if err == nil {
			t.Fatal("removeOrphans() error = nil, want an error")
		}
		return
	}
	if err != nil {
		t.Fatalf("removeOrphans() error = %v", err)
	}
	if !slices.Equal(got, tt.wantRemoved) {
		t.Errorf("removeOrphans() = %v, want %v", got, tt.wantRemoved)
	}
	for _, p := range tt.wantRemoved {
		if ok, _ := afero.Exists(dst, p); ok {
			t.Errorf("%s should have been removed", p)
		}
	}
	for _, p := range tt.wantPresent {
		if ok, _ := afero.Exists(dst, p); !ok {
			t.Errorf("%s should still exist", p)
		}
	}
}

// TestTrackedFiles proves the whole point of trackedFiles existing: a file
// git ignores never appears in its output, however deletion-eligible it
// otherwise looks (tracked file, present on disk, carries the header). None
// of TestRemoveOrphans's cases can prove this — their `tracked` slice is fed
// by the test itself, not read from git — so this shells out to real git,
// the same style command_runner_test.go already uses.
func TestTrackedFiles(t *testing.T) {
	dir := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=xp-provider-gen-test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=xp-provider-gen-test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	runGit("init", "-q")
	if err := os.WriteFile(filepath.Join(dir, orphanFileA), []byte("package a\n"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", orphanFileA, err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("_output/\n"), 0o600); err != nil {
		t.Fatalf("writing .gitignore: %v", err)
	}
	runGit("add", orphanFileA, ".gitignore")
	runGit("commit", "-q", "-m", "init")

	if err := os.MkdirAll(filepath.Join(dir, "_output"), 0o755); err != nil {
		t.Fatalf("mkdir _output: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "_output", "copied.go"),
		[]byte(core.GeneratedHeader+"\npackage x\n"), 0o600); err != nil {
		t.Fatalf("writing _output/copied.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "untracked.go"), []byte("package u\n"), 0o600); err != nil {
		t.Fatalf("writing untracked.go: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	got, err := trackedFiles(t.Context())
	if err != nil {
		t.Fatalf("trackedFiles: %v", err)
	}
	want := []string{".gitignore", orphanFileA}
	if !slices.Equal(got, want) {
		t.Errorf("trackedFiles() = %v, want %v (must not include gitignored _output/copied.go or untracked untracked.go)", got, want)
	}
}

// TestReconcileResultPrint pins the summary line and the per-removal lines
// reconcileResult.print writes, and the neutral "Updating from generator ...
// to ..." line that appears whenever lastVersion is known and differs from
// curVersion — regardless of whether anything was removed, and never
// phrased as a warning: a comparable *upgrade* (v0.1.0 -> v0.2.0) prints it
// too, and must never say "older" (checkNotDowngrade, not print, is what
// refuses an actual downgrade, before print ever runs).
func TestReconcileResultPrint(t *testing.T) {
	const (
		updatingMarker  = "Updating from generator"
		removedOneCount = "removed 1"
		removedALine    = "  removed " + orphanFileA + ": "
	)
	oneRemoved := reconcileResult{removed: []string{orphanFileA}}

	tests := []struct {
		name         string
		result       reconcileResult
		lastVersion  string
		curVersion   string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "no removal, same version",
			result:       reconcileResult{},
			lastVersion:  testGenV020,
			curVersion:   testGenV020,
			wantContains: []string{"removed 0"},
			wantAbsent:   []string{"  removed ", updatingMarker},
		},
		{
			name:         "removes one, same version",
			result:       oneRemoved,
			lastVersion:  testGenV020,
			curVersion:   testGenV020,
			wantContains: []string{removedOneCount, removedALine},
			wantAbsent:   []string{updatingMarker},
		},
		{
			name:        "upgrade with a removal prints a neutral note, never 'older'",
			result:      oneRemoved,
			lastVersion: testGenV010,
			curVersion:  testGenV020,
			wantContains: []string{
				removedOneCount, removedALine, "Updating from generator v0.1.0 to v0.2.0.",
			},
			wantAbsent: []string{"older"},
		},
		{
			name:        "lastVersion equals curVersion prints no note",
			result:      reconcileResult{},
			lastVersion: testGenV020,
			curVersion:  testGenV020,
			wantAbsent:  []string{updatingMarker},
		},
		{
			name:         "lastVersion empty (never updated before) prints no note",
			result:       oneRemoved,
			lastVersion:  "",
			curVersion:   testGenV020,
			wantContains: []string{removedOneCount},
			wantAbsent:   []string{updatingMarker},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.result.print(&buf, tt.lastVersion, tt.curVersion)
			got := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("print() = %q, want it to contain %q", got, want)
				}
			}
			for _, notWant := range tt.wantAbsent {
				if strings.Contains(got, notWant) {
					t.Errorf("print() = %q, want it to NOT contain %q", got, notWant)
				}
			}
		})
	}
}
