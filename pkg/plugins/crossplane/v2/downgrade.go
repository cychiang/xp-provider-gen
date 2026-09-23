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

// Downgrade detection: refusing 'update' and 'update --adopt' when the
// running generator is an older clean release than the one that last
// stamped PROJECT's provenance.
package v2

import (
	"fmt"
	"regexp"

	"golang.org/x/mod/semver"
)

// releaseVersionRe matches a clean release version. Only those are ordered:
// git-describe output such as v0.2.0-5-gabc1234 is valid semver but a
// *prerelease*, so semver.Compare would rank a maintainer's own development
// build below the release it follows — comparing it would refuse an update
// that only looks like a downgrade. This also means a last version stamped
// by a development build (e.g. v0.3.0-5-gabc1234) against a genuinely older
// cur (v0.2.0) is never caught: accepted deliberately, since it only happens
// when a maintainer updates their own project with a development build.
var releaseVersionRe = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// compareGenerators orders two generator versions: it returns
// semver.Compare(cur, last) and true when both are clean release versions
// (releaseVersionRe matches both), otherwise 0 and false. Every other shape —
// git-describe output, a "-dirty" suffix, "dev", a pseudo-version, a legacy
// bare hash, or empty — is undecidable and never compared.
func compareGenerators(last, cur string) (int, bool) {
	if !releaseVersionRe.MatchString(last) || !releaseVersionRe.MatchString(cur) {
		return 0, false
	}
	return semver.Compare(cur, last), true
}

// checkNotDowngrade refuses when both versions are clean release versions
// and cur is older than last. The returned error carries the whole
// user-facing message, including the deliberate-downgrade exit and the
// trailing "no changes were made; nothing to revert": runUpdate wraps
// prepare's own errors with that sentence, but runAdopt returns prepare's
// errors unwrapped, so the sentence cannot be left to either caller to add —
// it has to live in the error itself.
func checkNotDowngrade(last, cur string) error {
	cmp, ordered := compareGenerators(last, cur)
	if !ordered || cmp >= 0 {
		return nil
	}
	return fmt.Errorf(
		"refusing to update: this generator (%s) is older than the one that last updated this project (%s).\n"+
			"  Run the %s generator or newer. To downgrade deliberately, edit `version:` under the\n"+
			"  %s plugin in PROJECT, commit it, then run update again.\n"+
			"  no changes were made; nothing to revert",
		cur, last, last, pluginName)
}
