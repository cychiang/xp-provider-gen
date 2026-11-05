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

import (
	"strings"
	"testing"
)

// TestInertCommand pins which invocations refuseInert stops before
// cli.New(...) runs: "alpha" (cli.New itself corrupts PROJECT for "alpha
// generate", see refuseInert), "edit" and "create webhook" (the plugin has no
// such subcommand, so Kubebuilder panics), each with and without --help. Real
// commands, including "create api", must pass, as must a flag value that
// merely spells a refused name.
func TestInertCommand(t *testing.T) {
	const createWebhook = createCommand + " " + webhookCommand
	tests := []struct {
		name string
		args string
		want string
	}{
		{name: "no args", args: "", want: ""},
		{name: "only a boolean flag, no positional", args: "--help", want: ""},
		{name: "bare alpha", args: "alpha", want: alphaCommand},
		{name: "alpha generate --help", args: "alpha generate --help", want: alphaCommand},
		{name: "global flag before alpha", args: "--plugins=go/v4 alpha", want: alphaCommand},
		{name: "global flag with separate value before alpha", args: "--plugins go/v4 alpha", want: alphaCommand},
		{name: "edit", args: "edit", want: editCommand},
		{name: "edit --help", args: "edit --help", want: editCommand},
		{name: "edit with flag", args: "edit --multigroup", want: editCommand},
		{name: "create webhook", args: "create webhook", want: createWebhook},
		{name: "create webhook --help", args: "create webhook --help", want: createWebhook},
		{name: "create webhook with flags", args: "create webhook --group=g --kind K", want: createWebhook},
		{name: "create api", args: "create api --group=g", want: ""},
		{name: "create api --help", args: "create api --help", want: ""},
		{name: "create alone", args: "create", want: ""},
		{name: "init with domain", args: "init --domain=example.com", want: ""},
		{name: "flag value spells edit", args: "init --domain edit", want: ""},
		{name: "flag value spells webhook", args: "create api --kind webhook", want: ""},
		{name: "create-test", args: "create-test --kind=K", want: ""},
		{name: "update", args: "update", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inertCommand(strings.Fields(tt.args)); got != tt.want {
				t.Errorf("inertCommand(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
