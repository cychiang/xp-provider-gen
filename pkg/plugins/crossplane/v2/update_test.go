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
	"testing"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

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

func assertContains(t *testing.T, label string, list []string, want string) {
	t.Helper()
	if !slices.Contains(list, want) {
		t.Errorf("%s = %v, want to contain %q", label, list, want)
	}
}
