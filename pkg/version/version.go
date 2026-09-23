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

package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// CommandName is this binary's name, as the built binary, the Go module, the
// docs and the skill all spell it. One constant so every user-facing string
// naming it cannot drift apart from the others.
const CommandName = "xp-provider-gen"

// Build information. Populated at build-time via -ldflags.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
	Compiler  = runtime.Compiler
	Platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info contains version and build information.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// Get returns version and build information.
func Get() Info {
	return Info{
		Version:   resolveVersion(Version, buildInfoVersion()),
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Compiler:  Compiler,
		Platform:  Platform,
	}
}

// resolveVersion prefers the ldflags-injected version; a plain
// `go install …@vX.Y.Z` sets none, so Version stays "dev", and this falls
// back to the module version recorded in the build info. "(devel)" (a local
// checkout, not a version-pinned install) and "" (no build info available)
// both keep "dev".
func resolveVersion(ldflags, buildInfo string) string {
	if ldflags != "dev" {
		return ldflags
	}
	if buildInfo == "" || buildInfo == "(devel)" {
		return "dev"
	}
	return buildInfo
}

// buildInfoVersion returns the main module's version as recorded by the Go
// toolchain (e.g. what `go install pkg@vX.Y.Z` embeds), or "" if build info
// is unavailable.
func buildInfoVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return bi.Main.Version
}

// Short returns a short version string, always "v"-prefixed. Version may
// already carry the "v" (an ldflags-injected release tag or a build-info
// module version both do); prepending unconditionally would produce "vv0.1.0".
func (i Info) Short() string {
	if strings.HasPrefix(i.Version, "v") {
		return i.Version
	}
	return fmt.Sprintf("v%s", i.Version)
}
