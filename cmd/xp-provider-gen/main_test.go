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

package main

import "testing"

// TestFirstPositionalArg pins the arg-parsing refuseAlpha relies on to block
// "alpha" before cli.New(...) ever runs (cli.New itself corrupts PROJECT for
// any "alpha generate" invocation — see refuseAlpha's doc comment). Getting
// this wrong either fails to block a real "alpha ..." invocation, or
// misfires on an unrelated command that merely takes a flag before its first
// positional argument.
func TestFirstPositionalArg(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "no args", args: nil, want: ""},
		{name: "bare alpha", args: []string{alphaCommand}, want: alphaCommand},
		{name: "alpha generate", args: []string{alphaCommand, "generate"}, want: alphaCommand},
		{name: "alpha generate --help", args: []string{alphaCommand, "generate", "--help"}, want: alphaCommand},
		{name: "alpha update with flags", args: []string{alphaCommand, "update", "--from-version=v1", "--to-version=v2"}, want: alphaCommand},
		{name: "global flag before command", args: []string{"--plugins=go/v4", alphaCommand}, want: alphaCommand},
		{name: "global flag with separate value before command", args: []string{"--plugins", "go/v4", alphaCommand}, want: alphaCommand},
		{name: "unrelated command", args: []string{"create", "api", "--group=g"}, want: "create"},
		{name: "only a boolean flag, no positional", args: []string{"--help"}, want: ""},
		{name: "init with domain", args: []string{"init", "--domain=example.com"}, want: "init"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstPositionalArg(tt.args); got != tt.want {
				t.Errorf("firstPositionalArg(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
