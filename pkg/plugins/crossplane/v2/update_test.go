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
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/version"
	"github.com/cychiang/xp-provider-gen/pkg/versions"
)

// The group and version every update test resource lives in.
const (
	updateTestGroup   = "core"
	updateTestVersion = "v1alpha1"
)

// newUpdateTestConfig builds a PROJECT config carrying the given plugin block
// and resources, filled in the way createAPISubcommand.InjectResource does.
func newUpdateTestConfig(t *testing.T, meta projectMeta, kinds ...string) config.Config {
	t.Helper()
	cfg, err := config.New(cfgv3.Version)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}
	if err := cfg.SetDomain(testDomain); err != nil {
		t.Fatalf("SetDomain: %v", err)
	}
	if err := cfg.SetRepository("github.com/example/provider-test"); err != nil {
		t.Fatalf("SetRepository: %v", err)
	}
	if err := cfg.EncodePluginConfig(pluginName, meta); err != nil {
		t.Fatalf("EncodePluginConfig: %v", err)
	}
	for _, kind := range kinds {
		res := resource.Resource{
			GVK:        resource.GVK{Group: updateTestGroup, Version: updateTestVersion, Kind: kind, Domain: cfg.GetDomain()},
			Path:       cfg.GetRepository() + "/apis/" + updateTestGroup + "/" + updateTestVersion,
			API:        &resource.API{CRDVersion: "v1", Namespaced: true},
			Controller: true,
		}
		if err := cfg.AddResource(res); err != nil {
			t.Fatalf("AddResource: %v", err)
		}
	}
	return cfg
}

// upjetTestMeta is the plugin block `init --upjet` persists: only the
// resource prefix, never the render-time Terraform settings.
func upjetTestMeta() projectMeta {
	return projectMeta{Flavor: core.FlavorUpjet, Upjet: &core.UpjetSettings{TerraformResourcePrefix: "kubernetes"}}
}

// TestValidateProject pins that update gates PROJECT the way the other
// commands do: an unusable plugin block (such as an unknown flavor) is
// refused, and an upjet project may use kinds that shadow core Kubernetes
// names, as `create api` allows, while a native one may not.
func TestValidateProject(t *testing.T) {
	tests := []struct {
		name       string
		meta       projectMeta
		wantFlavor core.Flavor
		wantErr    string
	}{
		{name: string(core.FlavorNative), meta: projectMeta{Flavor: core.FlavorNative}, wantErr: "reserved"},
		{name: "unstamped reads as native", meta: projectMeta{}, wantErr: "reserved"},
		{name: "upjet allows reserved kinds", meta: upjetTestMeta(), wantFlavor: core.FlavorUpjet},
		{
			name:    "unknown flavor is refused",
			meta:    projectMeta{Flavor: unknownTestFlavor},
			wantErr: `PROJECT is not usable: PROJECT declares unknown flavor "` + string(unknownTestFlavor) + `"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := validateProject(newUpdateTestConfig(t, tt.meta, "Secret"))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("validateProject() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateProject() error = %v", err)
			}
			if meta.Flavor != tt.wantFlavor {
				t.Errorf("validateProject() flavor = %q, want %q", meta.Flavor, tt.wantFlavor)
			}
		})
	}
}

// TestRenderToMemFS pins that update renders each project with its own
// flavor's template set: an upjet project gets its plumbing (with the
// namespaced domain derived, since PROJECT does not persist it) and none of
// the native-only files, and a native project keeps rendering its own.
func TestRenderToMemFS(t *testing.T) {
	tests := []struct {
		name        string
		meta        projectMeta
		wantPaths   []string
		absentPaths []string
	}{
		{
			name:      string(core.FlavorNative),
			meta:      projectMeta{Flavor: core.FlavorNative},
			wantPaths: []string{nativeConnectorPath, makeFragmentPath},
		},
		{
			name:        string(core.FlavorUpjet),
			meta:        upjetTestMeta(),
			wantPaths:   []string{upjetProviderConfigPath, "config/zz_resources.go", makeFragmentPath},
			absentPaths: []string{nativeConnectorPath},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := afero.NewMemMapFs()
			cfg := newUpdateTestConfig(t, tt.meta, "Widget")
			if err := renderToMemFS(cfg, tt.meta, machinery.Filesystem{FS: mem}); err != nil {
				t.Fatalf("renderToMemFS: %v", err)
			}
			for _, p := range tt.wantPaths {
				if ok, _ := afero.Exists(mem, p); !ok {
					t.Errorf("rendered project is missing %s", p)
				}
			}
			for _, p := range tt.absentPaths {
				if ok, _ := afero.Exists(mem, p); ok {
					t.Errorf("rendered project contains %s, which belongs to another flavor", p)
				}
			}
		})
	}

	t.Run("upjet renders with PROJECT's settings and derives the namespaced domain", func(t *testing.T) {
		mem := afero.NewMemMapFs()
		meta := upjetTestMeta()
		if err := renderToMemFS(newUpdateTestConfig(t, meta, "Widget"), meta, machinery.Filesystem{FS: mem}); err != nil {
			t.Fatalf("renderToMemFS: %v", err)
		}
		got, err := afero.ReadFile(mem, upjetProviderConfigPath)
		if err != nil {
			t.Fatalf("reading %s: %v", upjetProviderConfigPath, err)
		}
		for _, want := range []string{`resourcePrefix = "kubernetes"`, `"example.m.com"`} {
			if !strings.Contains(string(got), want) {
				t.Errorf("%s does not contain %s:\n%s", upjetProviderConfigPath, want, got)
			}
		}

		// PROJECT keeps no Terraform CLI version: the tool-owned make fragment
		// must still pin the tool's, never an empty value.
		fragment, err := afero.ReadFile(mem, makeFragmentPath)
		if err != nil {
			t.Fatalf("reading %s: %v", makeFragmentPath, err)
		}
		if want := "export TERRAFORM_VERSION ?= " + versions.TerraformVersion + "\n"; !strings.Contains(string(fragment), want) {
			t.Errorf("%s does not contain %q:\n%s", makeFragmentPath, want, fragment)
		}
	})
}

// Paths the render tests key off.
const (
	nativeConnectorPath     = "internal/provider/connector.go"
	upjetProviderConfigPath = "config/provider.go"
	makeFragmentPath        = "hack/xp-provider-gen.mk"
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

func TestInsertGeneratedHeader(t *testing.T) {
	withLicense := "/*\nCopyright\n*/\n\npackage foo\n\nfunc F() {}\n"
	got := string(insertGeneratedHeader([]byte(withLicense)))
	if !core.IsToolOwned([]byte(got)) {
		t.Errorf("header not detected after insertion:\n%s", got)
	}
	if !strings.Contains(got, core.GeneratedHeader+"\n\npackage foo") {
		t.Errorf("header should sit just before the package clause:\n%s", got)
	}
	// Idempotent: a file that already has the header is unchanged.
	if again := string(insertGeneratedHeader([]byte(got))); again != got {
		t.Errorf("insertGeneratedHeader not idempotent:\n%s", again)
	}
}

// TestInsertGeneratedHeader_Shebang pins that a script's shebang line stays
// on line 1: a naive insertion before "\npackage " (or at byte 0 when there
// is none) would land the header ABOVE it, breaking it ("//: No such file or
// directory"). The header must sit after the shebang instead.
func TestInsertGeneratedHeader_Shebang(t *testing.T) {
	script := "#!/usr/bin/env bash\nset -euo pipefail\necho hi\n"
	got := string(insertGeneratedHeader([]byte(script)))

	if !strings.HasPrefix(got, "#!/usr/bin/env bash\n") {
		t.Fatalf("shebang must stay on line 1:\n%s", got)
	}
	if !core.IsToolOwned([]byte(got)) {
		t.Errorf("header not detected after insertion:\n%s", got)
	}
	if !strings.Contains(got, "set -euo pipefail") {
		t.Errorf("script body must be preserved:\n%s", got)
	}
	// Idempotent: a shebang script that already has the header is unchanged.
	if again := string(insertGeneratedHeader([]byte(got))); again != got {
		t.Errorf("insertGeneratedHeader not idempotent on a shebang script:\n%s", again)
	}
}

// TestAdoptFile_AdoptsShebangScripts pins the corrected A9 fix at the call
// site that matters: update --adopt must stamp the header on a pre-contract
// shell script too, not skip it. Skipping it would leave the on-disk file
// permanently headerless, which DecideWrite reads as user-owned forever —
// every later plain `update` would then also Skip it, so it would never be
// refreshed. insertGeneratedHeader's shebang branch keeps the interpreter
// line intact while still making the file correctly tool-owned.
func TestAdoptFile_AdoptsShebangScripts(t *testing.T) {
	src, dst := afero.NewMemMapFs(), afero.NewMemMapFs()

	rendered := "#!/usr/bin/env bash\n# " + core.GeneratedHeader + "\necho hi\n"
	onDisk := "#!/usr/bin/env bash\necho hi\n"
	_ = afero.WriteFile(src, "cluster/local/integration_tests.sh", []byte(rendered), 0o644)
	_ = afero.WriteFile(dst, "cluster/local/integration_tests.sh", []byte(onDisk), 0o644)

	_, adopted, err := adoptFile(src, dst, "cluster/local/integration_tests.sh")
	if err != nil {
		t.Fatalf("adoptFile: %v", err)
	}
	if !adopted {
		t.Fatal("adoptFile did not adopt a tool-owned shell script")
	}
	got, _ := afero.ReadFile(dst, "cluster/local/integration_tests.sh")
	if !strings.HasPrefix(string(got), "#!/usr/bin/env bash\n") {
		t.Fatalf("shebang must stay on line 1:\n%s", got)
	}
	if !core.IsToolOwned(got) {
		t.Errorf("adopted script is not recognized as tool-owned:\n%s", got)
	}
	if !strings.Contains(string(got), "echo hi") {
		t.Errorf("script body must be preserved:\n%s", got)
	}
	// DecideWrite must now treat it as tool-owned, so a later plain `update`
	// will refresh it instead of skipping it forever.
	if decision := core.DecideWrite(true, got); decision != core.Overwrite {
		t.Errorf("DecideWrite after adopt = %v, want Overwrite (must not be stuck Skip forever)", decision)
	}
}

func TestAdoptHeaders(t *testing.T) {
	src := afero.NewMemMapFs()
	dst := afero.NewMemMapFs()

	// Tool-owned render + an on-disk copy lacking the header (an "old" provider).
	_ = afero.WriteFile(src, "internal/controller/mytype/setup.go", []byte(core.GeneratedHeader+"\npackage mytype\n"), 0o644)
	_ = afero.WriteFile(dst, "internal/controller/mytype/setup.go", []byte("package mytype\n\nfunc Setup() {}\n"), 0o644)

	// User-owned render (no header) + on-disk user-owned file — must NOT be adopted.
	_ = afero.WriteFile(src, "internal/controller/mytype/controller.go", []byte("package mytype\n// stub\n"), 0o644)
	_ = afero.WriteFile(dst, "internal/controller/mytype/controller.go", []byte("package mytype\n// my logic\n"), 0o644)

	adopted, err := adoptHeaders(src, dst)
	if err != nil {
		t.Fatalf("adoptHeaders: %v", err)
	}

	setup, _ := afero.ReadFile(dst, "internal/controller/mytype/setup.go")
	if !core.IsToolOwned(setup) {
		t.Errorf("tool-owned setup.go should have been adopted (header added):\n%s", setup)
	}
	ctrl, _ := afero.ReadFile(dst, "internal/controller/mytype/controller.go")
	if core.IsToolOwned(ctrl) {
		t.Error("user-owned controller.go must not be adopted")
	}
	assertContains(t, "adopted", adopted, "internal/controller/mytype/setup.go")
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

// TestRevertAdvice pins that 'git reset --hard' alone does not undo a failed
// update once reconcile has seeded new (untracked) files — they are left
// behind while the advice implies the tree is clean. The advice must name
// them explicitly instead of blanket-suggesting 'git clean -fd', which would
// also delete unrelated untracked work already in the repo.
func TestRevertAdvice(t *testing.T) {
	if got := revertAdvice(nil); !strings.Contains(got, "git reset --hard") {
		t.Errorf("no seeded files: advice = %q, want it to mention git reset --hard", got)
	}
	if strings.Contains(revertAdvice(nil), "clean") {
		t.Errorf("no seeded files: advice must not suggest git clean: %q", revertAdvice(nil))
	}

	seeded := []string{"apis/register.go", "internal/provider/"}
	got := revertAdvice(seeded)
	for _, want := range seeded {
		if !strings.Contains(got, want) {
			t.Errorf("advice = %q, want it to name seeded path %q", got, want)
		}
	}
	if strings.Contains(got, "clean -fd") {
		t.Errorf("advice must not suggest git clean -fd (would delete unrelated untracked work): %q", got)
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
// reconcileResult.print writes, and the version-drift note that appears only
// when a removal happened and lastVersion names an older generator than the
// one running now: an unexpected removal is exactly the case where knowing
// "this binary is older than the one that last touched PROJECT" matters.
func TestReconcileResultPrint(t *testing.T) {
	const (
		noteMarker      = "note:"
		removedOneCount = "removed 1"
		removedALine    = "  removed " + orphanFileA + ": "
	)
	oneRemoved := reconcileResult{removed: []string{orphanFileA}}

	tests := []struct {
		name         string
		result       reconcileResult
		lastVersion  string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "no removal",
			result:       reconcileResult{},
			lastVersion:  version.Get().Version,
			wantContains: []string{"removed 0"},
			wantAbsent:   []string{"  removed ", noteMarker},
		},
		{
			name:         "removes one, same version",
			result:       oneRemoved,
			lastVersion:  version.Get().Version,
			wantContains: []string{removedOneCount, removedALine},
			wantAbsent:   []string{noteMarker},
		},
		{
			name:         "removes one, different version",
			result:       oneRemoved,
			lastVersion:  "old",
			wantContains: []string{removedOneCount, removedALine, noteMarker + " PROJECT was last updated by generator old"},
		},
		{
			name:         "removes one, lastVersion is empty",
			result:       oneRemoved,
			lastVersion:  "",
			wantContains: []string{removedOneCount},
			wantAbsent:   []string{noteMarker},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tt.result.print(&buf, tt.lastVersion)
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

// TestBumpTerraformVersion pins bumpTerraformVersion's contract: it rewrites
// exactly the two version-carrying lines a scaffolded Makefile declares,
// refuses to guess when the file's shape is not what it expects (rather than
// silently leaving one line stale), and never assumes a binary-name suffix
// beyond the "_v<version>_" substring it is told to replace.
func TestBumpTerraformVersion(t *testing.T) {
	const (
		oldTFVersion = "2.37.1"
		newTFVersion = "2.38.0"
	)
	validMakefile := "export TERRAFORM_PROVIDER_SOURCE ?= hashicorp/kubernetes\n" +
		"export TERRAFORM_PROVIDER_VERSION ?= " + oldTFVersion + "\n" +
		"export TERRAFORM_NATIVE_PROVIDER_BINARY ?= terraform-provider-kubernetes_v" + oldTFVersion + "_x5\n"

	tests := []struct {
		name        string
		makefile    string
		newVersion  string
		wantErr     bool
		wantOld     string
		wantContain []string
		wantMissing []string
	}{
		{
			name:        "rewrites both lines and reports the old version",
			makefile:    validMakefile,
			newVersion:  newTFVersion,
			wantOld:     oldTFVersion,
			wantContain: []string{"export TERRAFORM_PROVIDER_VERSION ?= " + newTFVersion, "terraform-provider-kubernetes_v" + newTFVersion + "_x5"},
			wantMissing: []string{oldTFVersion},
		},
		{
			name:       "missing TERRAFORM_PROVIDER_VERSION line errors",
			makefile:   "export TERRAFORM_NATIVE_PROVIDER_BINARY ?= terraform-provider-kubernetes_v" + oldTFVersion + "_x5\n",
			newVersion: newTFVersion,
			wantErr:    true,
		},
		{
			name: "two TERRAFORM_PROVIDER_VERSION lines errors",
			makefile: "export TERRAFORM_PROVIDER_VERSION ?= " + oldTFVersion + "\n" +
				"export TERRAFORM_PROVIDER_VERSION ?= " + oldTFVersion + "\n" +
				"export TERRAFORM_NATIVE_PROVIDER_BINARY ?= terraform-provider-kubernetes_v" + oldTFVersion + "_x5\n",
			newVersion: newTFVersion,
			wantErr:    true,
		},
		{
			name: "binary line with a non-_x5 suffix keeps the suffix, replaces only the version marker",
			makefile: "export TERRAFORM_PROVIDER_VERSION ?= " + oldTFVersion + "\n" +
				"export TERRAFORM_NATIVE_PROVIDER_BINARY ?= terraform-provider-kubernetes_v" + oldTFVersion + "_x3\n",
			newVersion:  newTFVersion,
			wantOld:     oldTFVersion,
			wantContain: []string{"terraform-provider-kubernetes_v" + newTFVersion + "_x3"},
			wantMissing: []string{oldTFVersion},
		},
		{
			name:        "new version equal to old version is idempotent, not an error",
			makefile:    validMakefile,
			newVersion:  oldTFVersion,
			wantOld:     oldTFVersion,
			wantContain: []string{"export TERRAFORM_PROVIDER_VERSION ?= " + oldTFVersion, "terraform-provider-kubernetes_v" + oldTFVersion + "_x5"},
		},
		{
			name: "binary line missing the old version marker errors",
			makefile: "export TERRAFORM_PROVIDER_VERSION ?= " + oldTFVersion + "\n" +
				"export TERRAFORM_NATIVE_PROVIDER_BINARY ?= terraform-provider-kubernetes_v9.9.9_x5\n",
			newVersion: newTFVersion,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, oldVersion, err := bumpTerraformVersion([]byte(tt.makefile), tt.newVersion)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("bumpTerraformVersion() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("bumpTerraformVersion() unexpected error: %v", err)
			}
			if oldVersion != tt.wantOld {
				t.Errorf("oldVersion = %q, want %q", oldVersion, tt.wantOld)
			}
			for _, want := range tt.wantContain {
				if !strings.Contains(string(got), want) {
					t.Errorf("result = %q, want it to contain %q", got, want)
				}
			}
			for _, missing := range tt.wantMissing {
				if strings.Contains(string(got), missing) {
					t.Errorf("result = %q, want it to not contain %q", got, missing)
				}
			}
		})
	}
}

// TestUpdateFlagRejects pins the four independent ways
// --terraform-provider-version is refused. Conditions 2-4 are pure flag
// checks and must reject before prepare ever touches git or the filesystem;
// condition 1 (flavor) is checked separately from those since it needs
// PROJECT's flavor, which only exists after prepare runs.
func TestUpdateFlagRejects(t *testing.T) {
	t.Run("project flavor is not upjet", func(t *testing.T) {
		err := checkTerraformVersionFlavor(core.FlavorNative)
		if err == nil {
			t.Fatal("checkTerraformVersionFlavor(native) = nil, want an error")
		}
		if !strings.Contains(err.Error(), "native") {
			t.Errorf("error = %q, want it to name the flavor", err)
		}
	})

	t.Run("adopt and terraform-provider-version are mutually exclusive", func(t *testing.T) {
		cmd := NewUpdateCommand()
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.SetArgs([]string{"--adopt", "--terraform-provider-version=2.38.0"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("Execute() = nil, want an error when --adopt and --terraform-provider-version are both set")
		}
		// Must be cobra's mutual-exclusion error specifically, not some later
		// failure (e.g. requireCleanTree) that Execute() would also return
		// nil-checked-only tests can't tell apart from the real rejection.
		if !strings.Contains(err.Error(), "[adopt terraform-provider-version]") {
			t.Errorf("err = %q, want cobra's mutually-exclusive-flags error naming [adopt terraform-provider-version]", err)
		}
	})

	t.Run("an inherited environment variable would silently override the flag", func(t *testing.T) {
		t.Setenv("TERRAFORM_PROVIDER_VERSION", "2.37.1")
		if err := checkTerraformVersionFlag("2.38.0"); err == nil {
			t.Error("checkTerraformVersionFlag() = nil, want an error when TERRAFORM_PROVIDER_VERSION is set")
		}
		t.Setenv("TERRAFORM_PROVIDER_VERSION", "")
		t.Setenv("TERRAFORM_NATIVE_PROVIDER_BINARY", "terraform-provider-kubernetes_v2.37.1_x5")
		if err := checkTerraformVersionFlag("2.38.0"); err == nil {
			t.Error("checkTerraformVersionFlag() = nil, want an error when TERRAFORM_NATIVE_PROVIDER_BINARY is set")
		}
	})

	t.Run("version value is not valid semver", func(t *testing.T) {
		// Clear both env vars first: otherwise an inherited
		// TERRAFORM_PROVIDER_VERSION or TERRAFORM_NATIVE_PROVIDER_BINARY
		// (plausible in an upjet developer's shell) would make this pass
		// for the wrong reason — the env-var check, not the semver one.
		for _, name := range terraformVersionEnvVars {
			t.Setenv(name, "")
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}
		}
		err := checkTerraformVersionFlag("not-a-version")
		if err == nil {
			t.Fatal("checkTerraformVersionFlag(\"not-a-version\") = nil, want an error")
		}
		if !strings.Contains(err.Error(), "must be a semver value") {
			t.Errorf("err = %q, want it to reject on the semver check, not the env-var check", err)
		}
	})
}

func assertContains(t *testing.T, label string, list []string, want string) {
	t.Helper()
	if !slices.Contains(list, want) {
		t.Errorf("%s = %v, want to contain %q", label, list, want)
	}
}
