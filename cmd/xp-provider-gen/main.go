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
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/cli"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"

	crossplanev2 "github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2"
	"github.com/cychiang/xp-provider-gen/pkg/version"
)

// commandName is this binary's name, as the built binary, the Go module, the
// docs and the skill all spell it — the plugin-style "crossplane-" prefixed
// name this file used to hardcode here does not exist anywhere else in the
// project. Extracted to one constant so the two user-facing strings naming
// it below cannot drift apart from each other the way
// go.mod/Dockerfile/dependencies.yaml did on the previous branch.
const commandName = "xp-provider-gen"

// alphaCommand is the name of Kubebuilder's "alpha" command tree, which this
// generator blocks (refuseAlpha) and hides (hideInertCommands) — see both for
// why.
const alphaCommand = "alpha"

// refuseAlpha exits before cli.New(...) is ever called if the invocation's
// first positional argument is "alpha". This must run before construction,
// not after: Kubebuilder v4.15.0's cli.New itself (pkg/cli/cli.go,
// getInfoFromConfigFile -> isAlphaGenerateCommand ->
// patchProjectFileInMemoryIfNeeded) rewrites PROJECT on disk for any
// invocation whose positional args are "alpha generate" (including "alpha
// generate --help") — it does a blind strings.ReplaceAll of
// "go.kubebuilder.io/v2" -> "go.kubebuilder.io/v4", and our plugin key
// "crossplane.go.kubebuilder.io/v2" contains that literal substring, so it
// gets corrupted into a key this generator never registers. That write
// happens before the cobra tree exists, so hiding "alpha" from --help
// (hideInertCommands, below) cannot prevent it — only never calling cli.New
// for an alpha invocation can. Blocking the whole "alpha" tree here, not just
// "alpha generate", is simpler than mirroring Kubebuilder's narrower check:
// every alpha subcommand is inert for this plugin anyway (they hardcode a
// lookup for the standard "go.kubebuilder.io/v4" plugin key, which this
// generator never registers).
func refuseAlpha(args []string) {
	if firstPositionalArg(args) != alphaCommand {
		return
	}
	fmt.Fprintln(os.Stderr, "Error: alpha commands are not supported by "+commandName)
	os.Exit(1)
}

// firstPositionalArg returns the first non-flag argument in args, skipping
// "--flag value" pairs the same way Kubebuilder's own isAlphaGenerateCommand
// does (a "--flag=value" or a boolean "--flag" consumes only itself; a
// "--flag value" pair consumes both tokens, recognized by the next token not
// itself looking like a flag).
func firstPositionalArg(args []string) string {
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				skipNext = true
			}
			continue
		}
		return arg
	}
	return ""
}

func main() {
	refuseAlpha(os.Args[1:])

	versionInfo := version.Get()

	cli, err := cli.New(
		cli.WithCommandName(commandName),
		cli.WithVersion(versionInfo.Short()),
		cli.WithDescription("Crossplane Provider Generator - A tool for scaffolding Crossplane providers "+
			"and managed resources following Crossplane v2 patterns"),
		cli.WithDefaultProjectVersion(cfgv3.Version),
		cli.WithPlugins(&crossplanev2.Plugin{}),
		cli.WithDefaultPlugins(cfgv3.Version, &crossplanev2.Plugin{}),
		cli.WithExtraCommands(crossplanev2.NewUpdateCommand(), crossplanev2.NewCreateTestCommand()),
		cli.WithCompletion(),
	)
	if err != nil {
		os.Exit(1)
	}
	hideInertCommands(cli.Command())
	if err := cli.Run(); err != nil {
		os.Exit(1)
	}
}

// hideInertCommands hides Kubebuilder-provided subcommands that our plugin
// does not implement. Kubebuilder's CLI always wires "edit", "create webhook"
// and "alpha" into the command tree regardless of what the plugin supports:
// our Plugin declares GetEditSubcommand/GetCreateWebhookSubcommand returning
// nil (this plugin offers no edit or webhook flow), which the CLI dereferences
// unconditionally and panics rather than reporting a clean error; "alpha
// generate"/"alpha update" hardcode a lookup for the standard
// "go.kubebuilder.io/v4" plugin key, which this project never registers, so
// they always fail with "no plugin could be resolved" regardless of project
// state. Hidden commands still run if invoked by name — Cobra has no runtime
// "disabled" concept — but they no longer clutter --help with functionality
// this generator does not offer. "alpha" is additionally blocked from running
// at all by refuseAlpha above (Hidden alone cannot stop its PROJECT-corrupting
// side effect); it stays in this list too so a bare `--help` doesn't list it.
func hideInertCommands(root *cobra.Command) {
	hide(root, "edit")
	hide(root, "create", "webhook")
	hide(root, alphaCommand)
}

// hide walks path (each element the Name() of one level) and marks the
// command it resolves to Hidden. It is a no-op if the path does not resolve,
// so a Kubebuilder upgrade that renames or removes one of these commands
// fails safe (the command simply reappears in --help) rather than panicking.
func hide(root *cobra.Command, path ...string) {
	cmd := root
	for _, name := range path {
		var next *cobra.Command
		for _, sub := range cmd.Commands() {
			if sub.Name() == name {
				next = sub
				break
			}
		}
		if next == nil {
			return
		}
		cmd = next
	}
	cmd.Hidden = true
}
