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
	"fmt"
	"go/version"

	"github.com/spf13/afero"
	"golang.org/x/mod/modfile"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// raiseGoDirective returns go.mod content whose `go` directive is at least
// goVersion, and the directive it found ("" when there was none). It only
// raises: a directive already at or above goVersion is the author's choice,
// so the content comes back unchanged with raised false. A missing directive
// means Go 1.16 to the go command, so it is raised too.
func raiseGoDirective(content []byte, goVersion string) ([]byte, string, bool, error) {
	f, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return nil, "", false, fmt.Errorf("parsing go.mod: %w", err)
	}
	from := ""
	if f.Go != nil {
		from = f.Go.Version
	}
	if from != "" && version.Compare("go"+from, "go"+goVersion) >= 0 {
		return content, from, false, nil
	}
	if err := f.AddGoStmt(goVersion); err != nil {
		return nil, "", false, fmt.Errorf("setting go.mod's go directive to %s: %w", goVersion, err)
	}
	updated, err := f.Format()
	if err != nil {
		return nil, "", false, fmt.Errorf("formatting go.mod: %w", err)
	}
	return updated, from, true, nil
}

// applyGoDirective raises the go directive of the go.mod in dst to goVersion
// (the manifest's go_version), so an existing provider gets the same Go floor
// a fresh init writes. go.mod is the author's file: nothing else in it changes,
// and a directive the author already set higher is kept.
func applyGoDirective(dst afero.Fs, goVersion string) error {
	const goModPath = "go.mod"
	content, err := afero.ReadFile(dst, goModPath)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}
	updated, from, raised, err := raiseGoDirective(content, goVersion)
	if err != nil {
		return err
	}
	if !raised {
		return nil
	}
	if err := afero.WriteFile(dst, goModPath, updated, core.FileMode(goModPath)); err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}
	if from == "" {
		from = "none"
	}
	fmt.Printf("Raised go.mod's go directive %s -> %s.\n", from, goVersion)
	return nil
}
