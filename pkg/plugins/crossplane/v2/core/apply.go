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
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// ApplyResult lists, per outcome, the files Apply visited.
type ApplyResult struct {
	Overwritten []string
	Seeded      []string
	Skipped     []string
	// Unseeded are user-owned files missing on disk that were deliberately not
	// seeded (upjet: some need init-time Terraform settings PROJECT lacks, so
	// none are recreated).
	Unseeded []string
}

// Apply copies every rendered file from src onto dst through the ownership
// gate: tool-owned (headered) files are overwritten, new files seeded, and
// user-owned (headerless) files left untouched. seedUserOwned says whether a
// user-owned file missing on disk is seeded or only recorded as unseeded.
func Apply(src, dst afero.Fs, seedUserOwned bool) (ApplyResult, error) {
	var result ApplyResult
	err := afero.Walk(src, ".", func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		rel := strings.TrimPrefix(filepath.ToSlash(path), "/")
		decision, err := applyFile(src, dst, path, rel, seedUserOwned)
		if err != nil {
			return err
		}
		result.record(decision, rel)
		return nil
	})
	return result, err
}

func (r *ApplyResult) record(decision WriteDecision, rel string) {
	switch decision {
	case Skip:
		r.Skipped = append(r.Skipped, rel)
	case Seed:
		r.Seeded = append(r.Seeded, rel)
	case Overwrite:
		r.Overwritten = append(r.Overwritten, rel)
	case Unseeded:
		r.Unseeded = append(r.Unseeded, rel)
	}
}

// CheckContained rejects a rendered path that would write outside the project
// directory. Rendered paths come from PROJECT (group/version/kind are
// substituted into template paths), so a hand-edited PROJECT must not be able
// to turn `update` into an arbitrary-file-write primitive.
//
// filepath.IsLocal is the whole check: it rejects absolute paths, any path that
// climbs out of the working directory, and the empty path — using the host's
// own separator rules. Hand-rolling it as a "../" prefix test missed
// backslash-separated escapes on Windows, which is a release target.
func CheckContained(rel string) error {
	if !filepath.IsLocal(rel) {
		return fmt.Errorf("refusing to write outside the project: %q", rel)
	}
	return nil
}

// applyFile reconciles one rendered file onto dst per the ownership gate. It
// returns Unseeded when the file is missing on disk, user-owned, and
// seedUserOwned is false: nothing is written.
func applyFile(src, dst afero.Fs, srcPath, rel string, seedUserOwned bool) (WriteDecision, error) {
	if err := CheckContained(rel); err != nil {
		return Skip, err
	}
	exists, err := afero.Exists(dst, rel)
	if err != nil {
		return Skip, err
	}
	var existing []byte
	if exists {
		if existing, err = afero.ReadFile(dst, rel); err != nil {
			return Skip, err
		}
	}

	decision := DecideWrite(exists, existing)
	if decision == Skip {
		return decision, nil
	}

	newContent, err := afero.ReadFile(src, srcPath)
	if err != nil {
		return decision, err
	}
	if !exists && !seedUserOwned && !IsToolOwned(newContent) {
		return Unseeded, nil
	}
	if err := dst.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		return decision, err
	}
	// Scripts are exec'd directly (uptest runs test/setup.sh), so the write
	// layer owns the executable bit — seeded .sh files must not land 0644.
	return decision, afero.WriteFile(dst, rel, newContent, FileMode(rel))
}
