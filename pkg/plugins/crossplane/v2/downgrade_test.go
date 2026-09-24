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
	"strings"
	"testing"
)

// Versions TestCheckNotDowngrade's table reuses across rows.
const (
	testGenV010 = "v0.1.0"
	testGenV020 = "v0.2.0"
)

// TestCheckNotDowngrade pins the comparison rule: refuse only when both
// versions are clean release versions (releaseVersionRe) and cur is
// genuinely older by semver.Compare — every other shape (git-describe
// output, "-dirty", "dev", a pseudo-version, a legacy bare hash, or empty)
// is undecidable and must never be refused.
func TestCheckNotDowngrade(t *testing.T) {
	tests := []struct {
		name       string
		last       string
		cur        string
		wantRefuse bool
	}{
		{name: "clean upgrade passes", last: testGenV010, cur: testGenV020},
		{name: "clean downgrade is refused", last: testGenV020, cur: testGenV010, wantRefuse: true},
		{name: "same version passes", last: testGenV020, cur: testGenV020},
		{
			name: "numeric, not lexicographic, comparison (v0.10.0 > v0.9.0)",
			last: "v0.10.0", cur: "v0.9.0", wantRefuse: true,
		},
		{name: "cur is a git-describe build ahead of last passes", last: testGenV020, cur: "v0.2.0-5-gabc1234"},
		{name: "cur is a dirty build passes", last: testGenV020, cur: "v0.2.0-dirty"},
		{name: "cur is dev passes", last: testGenV020, cur: "dev"},
		{
			name: "cur is a pseudo-version from a plain go build passes",
			last: testGenV020, cur: "v0.0.0-20260923200331-c7b96e644331",
		},
		{name: "last is a legacy bare hash passes", last: "7849359", cur: testGenV020},
		{name: "last is empty (never updated before) passes", last: "", cur: testGenV010},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkNotDowngrade(tt.last, tt.cur)
			if !tt.wantRefuse {
				if err != nil {
					t.Fatalf("checkNotDowngrade(%q, %q) = %v, want nil", tt.last, tt.cur, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("checkNotDowngrade(%q, %q) = nil, want a refusal", tt.last, tt.cur)
			}
			for _, want := range []string{
				"is older than the one that last updated this project",
				tt.last,
				tt.cur,
				"nothing to revert",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("checkNotDowngrade(%q, %q) error = %q, want it to contain %q", tt.last, tt.cur, err, want)
				}
			}
		})
	}
}
