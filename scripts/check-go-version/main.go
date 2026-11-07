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

// Command check-go-version checks that pkg/versions/dependencies.yaml's
// go_version, the Go language version generated providers declare, is new
// enough for every dependency generated providers pin, fetching each
// dependency's own go.mod from proxy.golang.org rather than guessing. It says
// nothing about the Go this repo is itself built with: the generator and the
// providers it scaffolds move independently.
//
// Run via `make check-go-version`, or `go run ./scripts/check-go-version`
// from the repo root. Requires network access.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cychiang/xp-provider-gen/pkg/versions"
)

func main() {
	if err := checkDependencyFloors(versions.GoVersion); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	fmt.Printf("OK: go_version %s is new enough for every pinned dependency.\n", versions.GoVersion)
}

// checkDependencyFloors fails if goVersion is older than the `go` directive
// of any dependency pkg/versions/dependencies.yaml pins, fetched live from
// proxy.golang.org (never guessed).
func checkDependencyFloors(goVersion string) error {
	deps := versions.UpjetGoModDependencies()
	client := &http.Client{Timeout: 15 * time.Second}
	for _, d := range deps {
		depDirective, err := fetchGoDirective(client, d.Module, d.Version)
		if err != nil {
			return fmt.Errorf("fetching %s@%s from proxy.golang.org: %w", d.Module, d.Version, err)
		}
		if depDirective == "" {
			continue // no go directive in that module's go.mod (pre-modules-era) — nothing to require
		}
		if err := requireFloor(goVersion, d.Module, d.Version, depDirective); err != nil {
			return err
		}
	}
	return nil
}

// requireFloor fails if goVersion is older than depDirective, the `go`
// directive of module@version.
func requireFloor(goVersion, module, version, depDirective string) error {
	cmp, err := compareGoVersions(goVersion, depDirective)
	if err != nil {
		return err
	}
	if cmp < 0 {
		return fmt.Errorf(
			"%s@%s requires go %s, but pkg/versions/dependencies.yaml's go_version is only %s — "+
				"bump go_version to at least %s",
			module, version, depDirective, goVersion, depDirective)
	}
	return nil
}

// goDirectiveRe matches a go.mod `go` directive line: "go 1.26.8" or the
// older two-component form "go 1.17".
var goDirectiveRe = regexp.MustCompile(`(?m)^go\s+(\d+\.\d+(?:\.\d+)?)\s*$`)

// extractGoDirective returns the `go` directive from go.mod content.
func extractGoDirective(content string) (string, error) {
	m := goDirectiveRe.FindStringSubmatch(content)
	if m == nil {
		return "", fmt.Errorf("no `go` directive found")
	}
	return m[1], nil
}

// compareGoVersions compares two dotted numeric Go versions (2 or 3
// components; a missing component is treated as 0), returning -1, 0, or 1
// the way strings.Compare does. Purely numeric, not lexicographic:
// "1.9.0" < "1.10.0".
func compareGoVersions(a, b string) (int, error) {
	pa, err := parseGoVersion(a)
	if err != nil {
		return 0, err
	}
	pb, err := parseGoVersion(b)
	if err != nil {
		return 0, err
	}
	for i := range pa {
		switch {
		case pa[i] < pb[i]:
			return -1, nil
		case pa[i] > pb[i]:
			return 1, nil
		}
	}
	return 0, nil
}

func parseGoVersion(v string) ([3]int, error) {
	var out [3]int
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return out, fmt.Errorf("not a valid go version: %q", v)
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, fmt.Errorf("not a valid go version: %q: %w", v, err)
		}
		out[i] = n
	}
	return out, nil
}

// escapeModulePath applies the Go module proxy's case-encoding
// (https://go.dev/ref/mod#module-proxy): each uppercase letter becomes '!'
// followed by its lowercase form, since module paths are case-sensitive but
// most filesystems and some proxies are not.
func escapeModulePath(path string) string {
	var b strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// fetchGoDirective downloads module@version's go.mod from proxy.golang.org
// and extracts its `go` directive. Returns "" (not an error) if that go.mod
// has none, which is legitimate for pre-modules-era packages (e.g.
// github.com/pkg/errors).
func fetchGoDirective(client *http.Client, module, version string) (string, error) {
	url := fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.mod", escapeModulePath(module), version)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("proxy returned %s for %s", resp.Status, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// No `go` directive is a legitimate outcome (pre-modules-era go.mod, e.g.
	// github.com/pkg/errors) — extractGoDirective's error here just means
	// "nothing to require", not a fetch failure, so it is deliberately
	// swallowed rather than propagated.
	directive, _ := extractGoDirective(string(body))
	return directive, nil
}
