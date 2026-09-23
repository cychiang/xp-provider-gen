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

// Tests for tfversion.go: the Makefile rewrite itself, the write-failure
// error marking runUpdate depends on for its revert advice, and the four
// independent ways --terraform-provider-version is rejected.
package v2

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

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

// TestApplyTerraformVersionBump_WriteFailure pins that a failure writing the
// Makefile back is distinguishable (via errors.Is(err, errMakefileWriteFailed))
// from a failure reading or parsing it: only the write can leave the tree
// dirty (afero.WriteFile truncates before writing), so runUpdate must give it
// different revert advice than the "no changes were made" it gives the other
// two.
func TestApplyTerraformVersionBump_WriteFailure(t *testing.T) {
	makefile := "export TERRAFORM_PROVIDER_VERSION ?= 2.37.1\n" +
		"export TERRAFORM_NATIVE_PROVIDER_BINARY ?= terraform-provider-kubernetes_v2.37.1_x5\n"

	t.Run("write failure is marked with errMakefileWriteFailed", func(t *testing.T) {
		mem := afero.NewMemMapFs()
		if err := afero.WriteFile(mem, "Makefile", []byte(makefile), 0o644); err != nil {
			t.Fatal(err)
		}
		ro := afero.NewReadOnlyFs(mem)

		_, err := applyTerraformVersionBump(ro, "2.38.0")
		if err == nil {
			t.Fatal("applyTerraformVersionBump() on a read-only Fs = nil, want a write error")
		}
		if !errors.Is(err, errMakefileWriteFailed) {
			t.Errorf("err = %q, want errors.Is(err, errMakefileWriteFailed)", err)
		}
	})

	t.Run("read failure is not marked with errMakefileWriteFailed", func(t *testing.T) {
		mem := afero.NewMemMapFs() // no Makefile written at all

		_, err := applyTerraformVersionBump(mem, "2.38.0")
		if err == nil {
			t.Fatal("applyTerraformVersionBump() with no Makefile = nil, want a read error")
		}
		if errors.Is(err, errMakefileWriteFailed) {
			t.Errorf("err = %q, a read failure must not be marked as a write failure", err)
		}
	})

	t.Run("parse failure is not marked with errMakefileWriteFailed", func(t *testing.T) {
		mem := afero.NewMemMapFs()
		if err := afero.WriteFile(mem, "Makefile", []byte("no version lines here\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err := applyTerraformVersionBump(mem, "2.38.0")
		if err == nil {
			t.Fatal("applyTerraformVersionBump() on an unparsable Makefile = nil, want an error")
		}
		if errors.Is(err, errMakefileWriteFailed) {
			t.Errorf("err = %q, a parse failure must not be marked as a write failure", err)
		}
	})
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
		// failure (e.g. requireCleanTree) that Execute() would also return —
		// a bare nil check can't tell the two apart.
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
		if err := os.Unsetenv("TERRAFORM_PROVIDER_VERSION"); err != nil {
			t.Fatal(err)
		}
		t.Setenv("TERRAFORM_NATIVE_PROVIDER_BINARY", "terraform-provider-kubernetes_v2.37.1_x5")
		if err := checkTerraformVersionFlag("2.38.0"); err == nil {
			t.Error("checkTerraformVersionFlag() = nil, want an error when TERRAFORM_NATIVE_PROVIDER_BINARY is set")
		}
		if err := os.Unsetenv("TERRAFORM_NATIVE_PROVIDER_BINARY"); err != nil {
			t.Fatal(err)
		}

		// make's `?=` cares whether the variable is defined at all, not
		// whether it is non-empty: exporting it as "" still overrides the
		// Makefile's default, so the flag must reject that too (this pins
		// os.LookupEnv, not os.Getenv(name) != "").
		t.Setenv("TERRAFORM_PROVIDER_VERSION", "")
		if err := checkTerraformVersionFlag("2.38.0"); err == nil {
			t.Error("checkTerraformVersionFlag() = nil, want an error when TERRAFORM_PROVIDER_VERSION is exported empty")
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
