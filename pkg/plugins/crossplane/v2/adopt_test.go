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

// Tests for adopt.go: inserting the generated header, including the
// shebang-script case, and adoptHeaders' walk over the rendered tree.
package v2

import (
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

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
