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
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// TestRefuseUnsupportedFlavor pins A1: update used to render the native
// template set unconditionally, overwriting an upjet project's plumbing with
// native files and leaving it unable to build. It must now refuse an upjet
// PROJECT outright and leave a native one unaffected.
func TestRefuseUnsupportedFlavor(t *testing.T) {
	newCfg := func(t *testing.T) config.Config {
		t.Helper()
		cfg, err := config.New(cfgv3.Version)
		if err != nil {
			t.Fatalf("config.New: %v", err)
		}
		return cfg
	}

	t.Run("native project is unaffected", func(t *testing.T) {
		cfg := newCfg(t)
		if err := cfg.EncodePluginConfig(pluginName, projectMeta{Flavor: core.FlavorNative}); err != nil {
			t.Fatalf("EncodePluginConfig: %v", err)
		}
		if err := refuseUnsupportedFlavor(cfg); err != nil {
			t.Errorf("refuseUnsupportedFlavor() on a native project = %v, want nil", err)
		}
	})

	t.Run("no plugin block at all defaults to native and is unaffected", func(t *testing.T) {
		cfg := newCfg(t)
		if err := refuseUnsupportedFlavor(cfg); err != nil {
			t.Errorf("refuseUnsupportedFlavor() on an unstamped project = %v, want nil", err)
		}
	})

	t.Run("upjet project is refused", func(t *testing.T) {
		cfg := newCfg(t)
		meta := projectMeta{Flavor: core.FlavorUpjet, Upjet: &core.UpjetSettings{TerraformProvider: "hashicorp/kubernetes"}}
		if err := cfg.EncodePluginConfig(pluginName, meta); err != nil {
			t.Fatalf("EncodePluginConfig: %v", err)
		}
		err := refuseUnsupportedFlavor(cfg)
		if err == nil || !strings.Contains(err.Error(), "upjet") {
			t.Fatalf("refuseUnsupportedFlavor() on an upjet project = %v, want an error naming upjet", err)
		}
	})
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

	result, err := reconcile(src, dst)
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

// TestInsertGeneratedHeader_Shebang pins A9: a naive insertion before "\npackage "
// (or at byte 0 when there is none) used to land the header ABOVE a script's
// shebang line, breaking it ("//: No such file or directory" on line 1). The
// header must sit after the shebang instead.
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

	// User-owned render (no header) + on-disk user file — must NOT be adopted.
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

	if _, err := reconcile(src, dst); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	got, err := afero.ReadFile(dst, "apis/newgroup/v1/groupversion_info.go")
	if err != nil || string(got) != content {
		t.Errorf("nested seed = %q (err %v), want the rendered content", got, err)
	}
}

// TestRevertAdvice pins A10: 'git reset --hard' alone does not undo a failed
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

func assertContains(t *testing.T, label string, list []string, want string) {
	t.Helper()
	if !slices.Contains(list, want) {
		t.Errorf("%s = %v, want to contain %q", label, list, want)
	}
}
