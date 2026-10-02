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
// tracked-files listing and orphan removal, and the result type update's caller
// reports through, plus the small print helpers init and create api share. The
// ownership-gated copy itself is core.Apply.
package v2

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

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
		if err := core.CheckContained(rel); err != nil {
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

// updateResult is what update reports: the core.Apply outcome plus the orphans
// removeOrphans deleted.
type updateResult struct {
	core.ApplyResult
	// removed are tracked, tool-owned files this render no longer produces,
	// deleted by removeOrphans.
	removed []string
}

// print writes the update summary to w. lastVersion is the generator version
// PROJECT was stamped with before this run (empty for a project predating
// provenance stamping); curVersion is the version running now. Whenever both
// are known and differ, a neutral "Updating from generator ... to ..." line
// leads the summary — stated as fact, not a warning: checkNotDowngrade
// already refused before print ever runs if cur were genuinely older, so an
// undecidable or forward difference reaching here is never something to
// second-guess.
func (r updateResult) print(w io.Writer, lastVersion, curVersion string, verbose bool) {
	if lastVersion != "" && lastVersion != curVersion {
		fmt.Fprintf(w, "Updating from generator %s to %s.\n", lastVersion, curVersion)
	}
	fmt.Fprintf(w, "Refreshed %d tool-owned file(s), added %d, removed %d, left %d user-owned file(s) untouched.\n",
		len(r.Overwritten), len(r.Seeded), len(r.removed), len(r.Skipped))
	fmt.Fprintf(w, "Unchanged %d tool-owned file(s) (identical content, not rewritten).\n", len(r.Unchanged))
	for _, rel := range r.removed {
		fmt.Fprintf(w, "  removed %s: carries the generated header but is no longer generated — "+
			"if this file is yours, remove the header\n", rel)
	}
	if len(r.Unseeded) > 0 {
		fmt.Fprintf(w, "Not seeded (user-owned; update does not recreate these on an upjet provider): %s\n",
			strings.Join(r.Unseeded, ", "))
	}
	if verbose {
		r.printVerbose(w)
	}
}

// printVerbose lists, per category, the files the summary counts. removed and
// Not seeded are already listed file by file above, so they are not repeated.
func (r updateResult) printVerbose(w io.Writer) {
	for _, c := range []struct {
		heading string
		paths   []string
	}{
		{"Refreshed (tool-owned, content changed):", r.Overwritten},
		{"Unchanged (tool-owned, identical content, not rewritten):", r.Unchanged},
		{"Added (missing, seeded):", r.Seeded},
		{"Kept (no generated header, user-owned):", r.Skipped},
	} {
		if len(c.paths) > 0 {
			fmt.Fprintln(w, c.heading)
			printPaths(w, c.paths)
		}
	}
}

// printPaths writes one indented path per line.
func printPaths(w io.Writer, paths []string) {
	for _, p := range paths {
		fmt.Fprintf(w, "  %s\n", p)
	}
}

// printKept lists the files init and create api left alone because they carry
// no generated header. It prints nothing when there are none.
func printKept(w io.Writer, kept []string) {
	if len(kept) == 0 {
		return
	}
	fmt.Fprintf(w, "Kept %d existing file(s) without the generated header:\n", len(kept))
	printPaths(w, kept)
}
