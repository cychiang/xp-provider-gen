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

import "testing"

func TestIsToolOwned(t *testing.T) {
	toolFile := "/*\nCopyright\n*/\n\n" + GeneratedHeader + "\n\npackage foo\n"
	userFile := "/*\nCopyright\n*/\n\npackage foo\n\nfunc Observe() {}\n"

	if !IsToolOwned([]byte(toolFile)) {
		t.Error("file with the generated header should be tool-owned")
	}
	if IsToolOwned([]byte(userFile)) {
		t.Error("file without the header should be user-owned")
	}
}

func TestDecideWrite(t *testing.T) {
	header := []byte(GeneratedHeader + "\npackage foo\n")
	user := []byte("package foo\nfunc Observe() {}\n")

	tests := []struct {
		name     string
		exists   bool
		existing []byte
		want     WriteDecision
	}{
		{name: "absent target is seeded", exists: false, existing: nil, want: Seed},
		{name: "tool-owned target is overwritten", exists: true, existing: header, want: Overwrite},
		{name: "user-owned target is skipped", exists: true, existing: user, want: Skip},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecideWrite(tt.exists, tt.existing); got != tt.want {
				t.Errorf("DecideWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}
