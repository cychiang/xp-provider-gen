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

// The update --adopt command: stamping the generated header onto an existing
// provider's tool-owned files so plain `update` can manage them afterward.
package v2

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/version"
)

func runAdopt(ctx context.Context) error {
	st, mem, _, err := prepare(ctx)
	if err != nil {
		return err
	}

	adopted, err := adoptHeaders(mem, afero.NewOsFs())
	if err != nil {
		return fmt.Errorf("adopting tool-owned files: %w", err)
	}
	if err := stampProvenance(st); err != nil {
		return fmt.Errorf("stamping provenance: %w", err)
	}

	fmt.Printf("Adopted %d tool-owned file(s) and stamped generator version %s in PROJECT.\n",
		len(adopted), version.Get().Version)
	fmt.Printf("Review with 'git diff', commit, then run '%s update' to refresh them.\n", version.CommandName)
	return nil
}

// adoptHeaders adds the generated header to on-disk files that the templates own
// (identified by the header in their rendered output) but that predate the
// ownership contract. User-owned files are left untouched.
func adoptHeaders(src, dst afero.Fs) ([]string, error) {
	var adopted []string
	err := afero.Walk(src, ".", func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		rel, ok, err := adoptFile(src, dst, path)
		if err != nil {
			return err
		}
		if ok {
			adopted = append(adopted, rel)
		}
		return nil
	})
	return adopted, err
}

// adoptFile adds the header to one on-disk file if the matching rendered file
// is tool-owned and the on-disk file exists without it. Returns whether it
// adopted. This must run for every tool-owned file, Go or not: a headerless
// pre-contract copy that adopt skipped would be classified user-owned by
// core.IsToolOwned and never refreshed by any later plain `update` either.
func adoptFile(src, dst afero.Fs, srcPath string) (string, bool, error) {
	rendered, err := afero.ReadFile(src, srcPath)
	if err != nil {
		return "", false, err
	}
	if !core.IsToolOwned(rendered) {
		return "", false, nil // user-owned template — never adopt
	}
	rel := strings.TrimPrefix(filepath.ToSlash(srcPath), "/")
	if err := checkContained(rel); err != nil {
		return "", false, err
	}
	exists, err := afero.Exists(dst, rel)
	if err != nil {
		return "", false, err
	}
	if !exists {
		return "", false, nil // absent on disk — a later `update` will seed it
	}
	existing, err := afero.ReadFile(dst, rel)
	if err != nil {
		return "", false, err
	}
	if core.IsToolOwned(existing) {
		return "", false, nil // already carries the header
	}
	if err := afero.WriteFile(dst, rel, insertGeneratedHeader(existing), core.FileMode(rel)); err != nil {
		return "", false, err
	}
	return rel, true, nil
}

// insertGeneratedHeader places the header just before the package clause, where
// IsToolOwned looks. It is a no-op if the header is already present. Content
// starting with a shebang gets the header after that first line instead —
// prepending it at byte 0 would break the interpreter line.
func insertGeneratedHeader(content []byte) []byte {
	if core.IsToolOwned(content) {
		return content
	}
	s := string(content)
	if strings.HasPrefix(s, "#!") {
		if nl := strings.IndexByte(s, '\n'); nl >= 0 {
			return []byte(s[:nl+1] + "# " + core.GeneratedHeader + "\n" + s[nl+1:])
		}
	}
	header := core.GeneratedHeader + "\n\n"
	if i := strings.Index(s, "\npackage "); i >= 0 {
		return []byte(s[:i+1] + header + s[i+1:])
	}
	return []byte(header + s)
}
