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

// The --terraform-provider-version flag: validating it and rewriting the
// scaffolded upjet Makefile's version-carrying lines.
package v2

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/validation"
)

// terraformVersionEnvVars are the Makefile's `?=`-assigned variables that would
// make --terraform-provider-version silently no-op if inherited from the
// environment: `?=` only assigns when the variable is unset, so an existing
// value in the environment would win over the flag's Makefile edit without
// any warning.
var terraformVersionEnvVars = []string{"TERRAFORM_PROVIDER_VERSION", "TERRAFORM_NATIVE_PROVIDER_BINARY"}

// checkTerraformVersionFlag rejects a --terraform-provider-version value
// before prepare touches git or the filesystem: either it is not a
// syntactically safe semver value, or an inherited environment variable would
// silently override the Makefile edit this run is about to make.
func checkTerraformVersionFlag(v string) error {
	for _, name := range terraformVersionEnvVars {
		if _, ok := os.LookupEnv(name); ok {
			return fmt.Errorf(
				"rejecting --terraform-provider-version: %s is set in the environment and would "+
					"silently override the Makefile's '?=' assignment; unset it first", name)
		}
	}
	if err := validation.ValidateTerraformProviderVersion(v); err != nil {
		return fmt.Errorf("validating --terraform-provider-version: %w", err)
	}
	return nil
}

// checkTerraformVersionFlavor rejects --terraform-provider-version on a
// non-upjet project: the flag rewrites Makefile lines only an upjet
// project's Makefile.tmpl declares.
func checkTerraformVersionFlavor(flavor core.Flavor) error {
	if flavor != core.FlavorUpjet {
		return fmt.Errorf("rejecting --terraform-provider-version: PROJECT's flavor is %q, not %q",
			flavor, core.FlavorUpjet)
	}
	return nil
}

// terraformProviderVersionLineRe matches the single line a scaffolded upjet
// Makefile declares the wrapped Terraform provider's version on.
var terraformProviderVersionLineRe = regexp.MustCompile(`(?m)^export TERRAFORM_PROVIDER_VERSION \?= (\S+)$`)

// terraformNativeProviderBinaryLineRe matches the single line a scaffolded
// upjet Makefile declares the native provider binary name on, capturing its
// value so the `_v<version>_` substring is only ever replaced inside that
// line, never anywhere else in the file a version string happens to match.
var terraformNativeProviderBinaryLineRe = regexp.MustCompile(`(?m)^export TERRAFORM_NATIVE_PROVIDER_BINARY \?= (.*)$`)

// bumpTerraformVersion rewrites a scaffolded upjet Makefile's two
// version-carrying lines to newVersion: the single
// `export TERRAFORM_PROVIDER_VERSION ?=` line, and the `_v<old>_` substring
// of the TERRAFORM_NATIVE_PROVIDER_BINARY line — and only inside that line's
// value, so a version string that happens to appear elsewhere in the
// Makefile (a comment, a user target) is never touched. It returns the
// rewritten content and the version it replaced.
//
// The Makefile is user-owned, so this only ever changes the value already
// declared there — it never assumes a binary-name suffix (`_x5`) or provider
// name beyond that substring. It requires exactly one line of each kind, and
// requires the binary line to already carry that same old version: a
// Makefile that fails either check already disagrees with itself about the
// current version, and guessing which line is right would be worse than
// refusing.
func bumpTerraformVersion(makefile []byte, newVersion string) ([]byte, string, error) {
	versionMatches := terraformProviderVersionLineRe.FindAllSubmatch(makefile, -1)
	if len(versionMatches) != 1 {
		return nil, "", fmt.Errorf(
			"finding a single 'export TERRAFORM_PROVIDER_VERSION ?=' line in Makefile (found %d)", len(versionMatches))
	}
	oldVersion := string(versionMatches[0][1])

	binaryMatches := terraformNativeProviderBinaryLineRe.FindAllSubmatch(makefile, -1)
	if len(binaryMatches) != 1 {
		return nil, "", fmt.Errorf(
			"finding a single 'export TERRAFORM_NATIVE_PROVIDER_BINARY ?=' line in Makefile (found %d)", len(binaryMatches))
	}
	oldMarker := []byte("_v" + oldVersion + "_")
	if !bytes.Contains(binaryMatches[0][1], oldMarker) {
		return nil, "", fmt.Errorf(
			"finding %q in Makefile's TERRAFORM_NATIVE_PROVIDER_BINARY line", oldMarker)
	}
	newMarker := []byte("_v" + newVersion + "_")

	updated := terraformProviderVersionLineRe.ReplaceAllFunc(makefile, func([]byte) []byte {
		return []byte("export TERRAFORM_PROVIDER_VERSION ?= " + newVersion)
	})
	updated = terraformNativeProviderBinaryLineRe.ReplaceAllFunc(updated, func(line []byte) []byte {
		return bytes.Replace(line, oldMarker, newMarker, 1)
	})
	return updated, oldVersion, nil
}

// errMakefileWriteFailed marks an applyTerraformVersionBump failure that
// happened during the write itself, once afero.WriteFile's O_TRUNC may
// already have truncated the file on disk. Unlike the read/parse failures
// above it, "no changes were made" would be false here — the tree is left
// dirty, and only a plain 'git reset --hard' (revertAdvice's no-seeded-files
// case, correct here since reconcile has not run yet) restores it.
var errMakefileWriteFailed = errors.New("writing Makefile")

// applyTerraformVersionBump reads the project's Makefile from dst, rewrites it
// via bumpTerraformVersion, and writes the result back, returning the version
// it replaced. It must run after prepare, never before: rewriting the
// Makefile ahead of requireCleanTree would make the flag reject itself on
// every run, since the tree requireCleanTree then sees is no longer clean.
func applyTerraformVersionBump(dst afero.Fs, newVersion string) (string, error) {
	const makefilePath = "Makefile"
	content, err := afero.ReadFile(dst, makefilePath)
	if err != nil {
		return "", fmt.Errorf("reading Makefile: %w", err)
	}
	updated, oldVersion, err := bumpTerraformVersion(content, newVersion)
	if err != nil {
		return "", fmt.Errorf("bumping Terraform provider version in Makefile: %w", err)
	}
	if err := afero.WriteFile(dst, makefilePath, updated, core.FileMode(makefilePath)); err != nil {
		return "", fmt.Errorf("%w: %w", errMakefileWriteFailed, err)
	}
	return oldVersion, nil
}

// applyTerraformVersionFlagIfSet rewrites the Makefile's wrapped Terraform
// provider version when --terraform-provider-version was given; it is a
// no-op otherwise. runUpdate calls it after checkNotDowngrade and before any
// other write, so every error path here still reports "no changes were made"
// except the write failure itself.
func applyTerraformVersionFlagIfSet(flavor core.Flavor, terraformProviderVersion string) error {
	if terraformProviderVersion == "" {
		return nil
	}
	if err := checkTerraformVersionFlavor(flavor); err != nil {
		return fmt.Errorf("%w\n  no changes were made; nothing to revert", err)
	}
	oldVersion, err := applyTerraformVersionBump(afero.NewOsFs(), terraformProviderVersion)
	if err != nil {
		if errors.Is(err, errMakefileWriteFailed) {
			// The write itself failed: afero.WriteFile truncates before
			// writing, so the Makefile on disk may already differ from
			// what git has committed. "no changes were made" would be
			// false here; reconcile has not run yet, so a plain 'git
			// reset --hard' (revertAdvice's no-seeded-files case) is
			// enough to restore it.
			return fmt.Errorf("%w\n%s", err, revertAdvice(nil))
		}
		return fmt.Errorf("%w\n  no changes were made; nothing to revert", err)
	}
	fmt.Printf("Bumped Terraform provider version %s -> %s in Makefile.\n", oldVersion, terraformProviderVersion)
	return nil
}
