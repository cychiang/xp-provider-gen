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

// Tests for update.go: the shared test config/meta helpers (also used by
// reconcile_test.go, adopt_test.go and tfversion_test.go), project loading
// and validation, template rendering, revert advice, and the flag-wiring
// test that has to go through cmd.Execute().
package v2

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
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
	fillUpdateTestConfig(t, cfg, meta, kinds...)
	return cfg
}

func fillUpdateTestConfig(t *testing.T, cfg config.Config, meta projectMeta, kinds ...string) {
	t.Helper()
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
			wantPaths: []string{nativeConnectorPath, makeFragmentPath, apisRegisterPath},
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
	apisRegisterPath        = "apis/register.go"
)

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

	seeded := []string{apisRegisterPath, "internal/provider/"}
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

// TestUpdateCommand_RejectsTerraformVersionFlagBeforeRunning pins that RunE
// actually calls checkTerraformVersionFlag, not just that the function
// rejects in isolation (TestUpdateFlagRejects calls it directly, so it can't
// tell whether RunE is still wired to it). Going through cmd.Execute() with a
// real flag value proves the wiring; the env var makes this reject before
// runUpdate ever touches git or the filesystem, so it's safe to run as a unit
// test with no fixture project.
func TestUpdateCommand_RejectsTerraformVersionFlagBeforeRunning(t *testing.T) {
	t.Setenv("TERRAFORM_PROVIDER_VERSION", "2.37.1")

	cmd := NewUpdateCommand()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--terraform-provider-version=2.38.0"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() = nil, want an error when TERRAFORM_PROVIDER_VERSION is set in the environment")
	}
	if !strings.Contains(err.Error(), "TERRAFORM_PROVIDER_VERSION is set in the environment") {
		t.Errorf("err = %q, want it to name the env-var rejection (RunE must call checkTerraformVersionFlag)", err)
	}
}

// TestUpdateCommand_AdoptReturnsPrepareErrorBare pins that the --adopt branch
// of RunE returns runAdopt's error unwrapped, not every error appended with
// "changes are uncommitted; run 'git reset --hard' to revert" (revertAdvice's
// no-seeded-files case). Before this, RunE always appended that advice, so
// even a prepare failure — nothing written yet, since prepare fails before
// any write — told the user to revert a change that was never made. This
// runs somewhere that isn't a git repository at all, so prepare fails at
// requireCleanTree's very first step: a real regression test without needing
// a full scaffolded project on disk.
func TestUpdateCommand_AdoptReturnsPrepareErrorBare(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	cmd := NewUpdateCommand()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--adopt"})

	got := cmd.Execute()
	if got == nil {
		t.Fatal("Execute() = nil, want an error outside a git repository")
	}
	if strings.Contains(got.Error(), "git reset --hard") {
		t.Errorf("err = %q, want it to NOT advise 'git reset --hard' for a failure before any write", got)
	}
}

func assertContains(t *testing.T, label string, list []string, want string) {
	t.Helper()
	if !slices.Contains(list, want) {
		t.Errorf("%s = %v, want to contain %q", label, list, want)
	}
}

const (
	staleGoDirective = "1.20"
	orphanPath       = "internal/retired.go"
	ignoredOrphan    = "_output/retired.go"
	shimLogName      = "calls.log"
)

type updateFixture struct {
	dir string
	fs  afero.Fs
	log string
}

func newUpdateFixture(t *testing.T) *updateFixture {
	t.Helper()
	dir := t.TempDir()
	fx := &updateFixture{dir: dir, fs: afero.NewBasePathFs(afero.NewOsFs(), dir)}

	meta := projectMeta{Flavor: core.FlavorNative}
	st := yaml.New(machinery.Filesystem{FS: fx.fs})
	if err := st.New(cfgv3.Version); err != nil {
		t.Fatalf("store.New: %v", err)
	}
	fillUpdateTestConfig(t, st.Config(), meta, "Widget")
	if err := st.Save(); err != nil {
		t.Fatalf("saving PROJECT: %v", err)
	}
	mem := afero.NewMemMapFs()
	if err := renderToMemFS(st.Config(), meta, machinery.Filesystem{FS: mem}); err != nil {
		t.Fatalf("renderToMemFS: %v", err)
	}
	if _, err := core.Apply(mem, fx.fs, true); err != nil {
		t.Fatalf("seeding project: %v", err)
	}
	fx.write(t, "go.mod", "module github.com/example/provider-test\n\ngo "+staleGoDirective+"\n")
	fx.write(t, ".gitignore", "_output/\n")

	shims := t.TempDir()
	fx.log = filepath.Join(shims, shimLogName)
	for _, tool := range []string{"make", "go"} {
		script := "#!/bin/sh\necho \"" + tool + " $*\" >> '" + fx.log + "'\n"
		if err := os.WriteFile(filepath.Join(shims, tool), []byte(script), 0o700); err != nil { //nolint:gosec // an executable shim
			t.Fatalf("writing %s shim: %v", tool, err)
		}
	}
	t.Setenv("PATH", shims+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(dir)

	fx.git(t, "init", "-q")
	fx.commit(t)
	return fx
}

func (fx *updateFixture) write(t *testing.T, rel, content string) {
	t.Helper()
	if err := fx.fs.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		t.Fatalf("creating directory for %s: %v", rel, err)
	}
	if err := afero.WriteFile(fx.fs, rel, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", rel, err)
	}
}

func (fx *updateFixture) read(t *testing.T, rel string) string {
	t.Helper()
	b, err := afero.ReadFile(fx.fs, rel)
	if err != nil {
		t.Fatalf("reading %s: %v", rel, err)
	}
	return string(b)
}

func (fx *updateFixture) git(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = fx.dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=xp-provider-gen-test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=xp-provider-gen-test", "GIT_COMMITTER_EMAIL=test@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func (fx *updateFixture) commit(t *testing.T) {
	t.Helper()
	fx.git(t, "add", "-A")
	fx.git(t, "commit", "-q", "-m", "fixture")
}

func (fx *updateFixture) shimCalls(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(fx.log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("reading shim log: %v", err)
	}
	return string(b)
}

func (fx *updateFixture) projectVersion(t *testing.T) string {
	t.Helper()
	st := yaml.New(machinery.Filesystem{FS: fx.fs})
	if err := st.Load(); err != nil {
		t.Fatalf("loading PROJECT: %v", err)
	}
	meta, err := loadProjectMeta(st.Config())
	if err != nil {
		t.Fatalf("loadProjectMeta: %v", err)
	}
	return meta.Version
}

func (fx *updateFixture) goDirective(t *testing.T) string {
	t.Helper()
	for _, line := range strings.Split(fx.read(t, "go.mod"), "\n") {
		if v, ok := strings.CutPrefix(line, "go "); ok {
			return v
		}
	}
	t.Fatal("go.mod has no go directive")
	return ""
}

func TestRunUpdate_RaisesGoDirective(t *testing.T) {
	fx := newUpdateFixture(t)

	if err := runUpdate(t.Context(), fx.fs, "", false); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if got := fx.goDirective(t); got != versions.GoVersion {
		t.Errorf("go directive = %q, want %q", got, versions.GoVersion)
	}
}

func TestRunUpdate_RemovesOrphans(t *testing.T) {
	fx := newUpdateFixture(t)
	fx.write(t, orphanPath, core.GeneratedHeader+"\npackage internal\n")
	fx.write(t, ignoredOrphan, core.GeneratedHeader+"\npackage internal\n")
	fx.commit(t)

	if err := runUpdate(t.Context(), fx.fs, "", false); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if ok, _ := afero.Exists(fx.fs, orphanPath); ok {
		t.Errorf("%s still exists, want a tracked file this generator no longer produces removed", orphanPath)
	}
	if ok, _ := afero.Exists(fx.fs, ignoredOrphan); !ok {
		t.Errorf("%s was removed, want a gitignored file left alone", ignoredOrphan)
	}
}

func TestRunUpdate_RefusesDirtyTree(t *testing.T) {
	fx := newUpdateFixture(t)
	fx.write(t, "external.go", "package provider // edited\n")

	err := runUpdate(t.Context(), fx.fs, "", false)
	if err == nil || !strings.Contains(err.Error(), "working tree is not clean") {
		t.Fatalf("runUpdate error = %v, want it to name the dirty tree", err)
	}
	if got := fx.goDirective(t); got != staleGoDirective {
		t.Errorf("go directive = %q after a refused update, want it unchanged at %q", got, staleGoDirective)
	}
}

func TestRunUpdate_SecondRunChangesNothingButProvenance(t *testing.T) {
	fx := newUpdateFixture(t)
	if err := runUpdate(t.Context(), fx.fs, "", false); err != nil {
		t.Fatalf("first runUpdate: %v", err)
	}
	fx.commit(t)

	if err := runUpdate(t.Context(), fx.fs, "", false); err != nil {
		t.Fatalf("second runUpdate: %v", err)
	}
	if status := fx.git(t, "status", "--porcelain"); strings.TrimSpace(status) != "" {
		t.Errorf("git status after a second update = %q, want it empty", status)
	}
}

func TestRunUpdate_RunsFinalizeThroughShims(t *testing.T) {
	fx := newUpdateFixture(t)
	if err := runUpdate(t.Context(), fx.fs, "", false); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	calls := strings.Split(strings.TrimSpace(fx.shimCalls(t)), "\n")
	goGet := slices.IndexFunc(calls, func(c string) bool { return strings.HasPrefix(c, "go get ") })
	tidy := slices.Index(calls, "go mod tidy")
	if goGet < 0 || tidy < 0 || goGet > tidy {
		t.Errorf("shim calls = %q, want a go get before go mod tidy", calls)
	}
	for _, want := range []string{"make generate", "make reviewable"} {
		if !slices.Contains(calls, want) {
			t.Errorf("shim calls = %q, want %q", calls, want)
		}
	}
	if got := fx.projectVersion(t); got != version.Get().Version {
		t.Errorf("PROJECT version = %q, want %q stamped", got, version.Get().Version)
	}
}

func TestRunAdopt_RestoresStrippedHeader(t *testing.T) {
	fx := newUpdateFixture(t)
	stripped := strings.Replace(fx.read(t, apisRegisterPath), core.GeneratedHeader+"\n", "", 1)
	fx.write(t, apisRegisterPath, stripped)
	fx.write(t, "external.go", "package provider // user code\n")
	fx.commit(t)

	if err := runAdopt(t.Context(), fx.fs, false); err != nil {
		t.Fatalf("runAdopt: %v", err)
	}
	if !strings.Contains(fx.read(t, apisRegisterPath), core.GeneratedHeader) {
		t.Errorf("%s has no generated header after adopt", apisRegisterPath)
	}
	if got := fx.read(t, "external.go"); got != "package provider // user code\n" {
		t.Errorf("external.go = %q, want user-owned file untouched", got)
	}
	if got := fx.projectVersion(t); got != version.Get().Version {
		t.Errorf("PROJECT version = %q, want %q stamped", got, version.Get().Version)
	}
}

func TestRunCreateTest_ScaffoldsOnceThenRefuses(t *testing.T) {
	fx := newUpdateFixture(t)
	const testPath = "test/behavior/drift/chainsaw-test.yaml"
	var out strings.Builder

	if err := runCreateTest(fx.fs, "drift", "Widget", strings.NewReader(""), &out); err != nil {
		t.Fatalf("runCreateTest: %v", err)
	}
	first := fx.read(t, testPath)
	fx.write(t, testPath, first+"# edited\n")

	if err := runCreateTest(fx.fs, "drift", "Widget", strings.NewReader(""), &out); err == nil {
		t.Fatal("second runCreateTest = nil, want a refusal")
	}
	if got := fx.read(t, testPath); got != first+"# edited\n" {
		t.Errorf("%s changed by the refused rerun", testPath)
	}
}
