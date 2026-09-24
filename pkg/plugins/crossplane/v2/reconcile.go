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

// Reconciling the rendered template set onto disk for the update command: the
// ownership-gated copy (reconcile, applyFile), the tracked-files listing and
// orphan removal, and the result type update's caller reports through.
package v2

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// reconcile copies every rendered file from src onto dst through the ownership
// gate: tool-owned (headered) files are overwritten, new files seeded, and
// user-owned (headerless) files left untouched. seedUserOwned says whether a
// user-owned file missing on disk is seeded or only recorded as unseeded.
func reconcile(src, dst afero.Fs, seedUserOwned bool) (reconcileResult, error) {
	var result reconcileResult
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

// trackedFiles lists the files git tracks in the project, the only files a
// deletion may consider: a file git does not track cannot be restored with
// 'git reset --hard', and `git status --porcelain` — the clean-tree gate — does
// not see ignored files at all, so a disk walk could silently and irreversibly
// delete a headered file sitting in .work/, _output/ or vendor/.
func trackedFiles(ctx context.Context) ([]string, error) {
	out, err := core.NewCommandRunner("").RunWithOutput(ctx, "git", "ls-files", "-z")
	if err != nil {
		return nil, fmt.Errorf("listing git-tracked files: %w", err)
	}
	var files []string
	for _, f := range strings.Split(strings.TrimRight(out, "\x00"), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

// removeOrphans deletes the tracked files that carry the generated header but
// are absent from this render: the tool-owned files this generator no longer
// produces. It returns the paths it removed, in the order given.
func removeOrphans(tracked []string, src, dst afero.Fs) ([]string, error) {
	var removed []string
	for _, rel := range tracked {
		if err := checkContained(rel); err != nil {
			return removed, err
		}
		fi, err := dst.Stat(rel)
		switch {
		case os.IsNotExist(err):
			continue // tracked but not on disk — the user already removed it
		case err != nil:
			return removed, err
		case fi.IsDir():
			// A submodule gitlink: git ls-files reports it, but it is not a
			// regular file to read or remove.
			continue
		}
		existing, err := afero.ReadFile(dst, rel)
		if err != nil {
			return removed, err
		}
		if !core.IsToolOwned(existing) {
			continue // user-owned — never touched
		}
		stillRendered, err := afero.Exists(src, rel)
		if err != nil {
			return removed, err
		}
		if stillRendered {
			continue // this run still produces it
		}
		if err := dst.Remove(rel); err != nil {
			return removed, err
		}
		removed = append(removed, rel)
	}
	return removed, nil
}

// checkContained rejects a rendered path that would write outside the project
// directory. Rendered paths come from PROJECT (group/version/kind are
// substituted into template paths), so a hand-edited PROJECT must not be able
// to turn `update` into an arbitrary-file-write primitive.
//
// filepath.IsLocal is the whole check: it rejects absolute paths, any path that
// climbs out of the working directory, and the empty path — using the host's
// own separator rules. Hand-rolling it as a "../" prefix test missed
// backslash-separated escapes on Windows, which is a release target.
func checkContained(rel string) error {
	if !filepath.IsLocal(rel) {
		return fmt.Errorf("refusing to write outside the project: %q", rel)
	}
	return nil
}

// applyFile reconciles one rendered file onto dst per the ownership gate. It
// returns Unseeded when the file is missing on disk, user-owned, and
// seedUserOwned is false: nothing is written.
func applyFile(src, dst afero.Fs, srcPath, rel string, seedUserOwned bool) (core.WriteDecision, error) {
	if err := checkContained(rel); err != nil {
		return core.Skip, err
	}
	exists, err := afero.Exists(dst, rel)
	if err != nil {
		return core.Skip, err
	}
	var existing []byte
	if exists {
		if existing, err = afero.ReadFile(dst, rel); err != nil {
			return core.Skip, err
		}
	}

	decision := core.DecideWrite(exists, existing)
	if decision == core.Skip {
		return decision, nil
	}

	newContent, err := afero.ReadFile(src, srcPath)
	if err != nil {
		return decision, err
	}
	if !exists && !seedUserOwned && !core.IsToolOwned(newContent) {
		return core.Unseeded, nil
	}
	if err := dst.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		return decision, err
	}
	// Scripts are exec'd directly (uptest runs test/setup.sh), so the write
	// layer owns the executable bit — seeded .sh files must not land 0644.
	return decision, afero.WriteFile(dst, rel, newContent, core.FileMode(rel))
}

type reconcileResult struct {
	overwritten []string
	seeded      []string
	skipped     []string
	// unseeded are user-owned files missing on disk that were deliberately not
	// seeded (upjet: some need init-time Terraform settings PROJECT lacks, so
	// none are recreated).
	unseeded []string
	// removed are tracked, tool-owned files this render no longer produces,
	// deleted by removeOrphans.
	removed []string
}

func (r *reconcileResult) record(decision core.WriteDecision, rel string) {
	switch decision {
	case core.Skip:
		r.skipped = append(r.skipped, rel)
	case core.Seed:
		r.seeded = append(r.seeded, rel)
	case core.Overwrite:
		r.overwritten = append(r.overwritten, rel)
	case core.Unseeded:
		r.unseeded = append(r.unseeded, rel)
	}
}

// print writes the update summary to w. lastVersion is the generator version
// PROJECT was stamped with before this run (empty for a project predating
// provenance stamping); curVersion is the version running now. Whenever both
// are known and differ, a neutral "Updating from generator ... to ..." line
// leads the summary — stated as fact, not a warning: checkNotDowngrade
// already refused before print ever runs if cur were genuinely older, so an
// undecidable or forward difference reaching here is never something to
// second-guess.
func (r reconcileResult) print(w io.Writer, lastVersion, curVersion string) {
	if lastVersion != "" && lastVersion != curVersion {
		fmt.Fprintf(w, "Updating from generator %s to %s.\n", lastVersion, curVersion)
	}
	fmt.Fprintf(w, "Refreshed %d tool-owned file(s), added %d, removed %d, left %d user-owned file(s) untouched.\n",
		len(r.overwritten), len(r.seeded), len(r.removed), len(r.skipped))
	for _, rel := range r.removed {
		fmt.Fprintf(w, "  removed %s: carries the generated header but is no longer generated — "+
			"if this file is yours, remove the header\n", rel)
	}
	if len(r.unseeded) > 0 {
		fmt.Fprintf(w, "Not seeded (user-owned; update does not recreate these on an upjet provider): %s\n",
			strings.Join(r.unseeded, ", "))
	}
}
