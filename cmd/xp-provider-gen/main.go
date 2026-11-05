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

// Names of the Kubebuilder commands this generator blocks (refuseInert) and
// hides (hideInertCommands) — see both for why.
const (
	alphaCommand   = "alpha"
	editCommand    = "edit"
	createCommand  = "create"
	webhookCommand = "webhook"
)

// refuseInert exits before cli.New(...) is ever called if the invocation is
// one of the Kubebuilder commands this generator cannot run. This must run
// before construction, not after, for "alpha": Kubebuilder v4.15.0's cli.New
// itself (pkg/cli/cli.go, getInfoFromConfigFile -> isAlphaGenerateCommand ->
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
//
// "edit" and "create webhook" are refused for a different reason: the plugin
// has no such subcommand, and Kubebuilder dereferences the nil one inside a
// project (cmd_helpers.go), crashing with a goroutine trace. Their --help
// is refused too, since it would list this plugin as supporting them.
func refuseInert(args []string) {
	if cmd := inertCommand(args); cmd != "" {
		fmt.Fprintf(os.Stderr, "Error: %s is not supported by %s\n", cmd, version.CommandName)
		os.Exit(1)
	}
}

// inertCommand returns the name of the unsupported command args invoke, or ""
// if args invoke a supported one.
func inertCommand(args []string) string {
	pos := positionalArgs(args)
	switch {
	case len(pos) == 0:
		return ""
	case pos[0] == alphaCommand || pos[0] == editCommand:
		return pos[0]
	case pos[0] == createCommand && len(pos) > 1 && pos[1] == webhookCommand:
		return createCommand + " " + webhookCommand
	}
	return ""
}

// positionalArgs returns the non-flag arguments in args, skipping
// "--flag value" pairs the same way Kubebuilder's own isAlphaGenerateCommand
// does (a "--flag=value" or a boolean "--flag" consumes only itself; a
// "--flag value" pair consumes both tokens, recognized by the next token not
// itself looking like a flag).
func positionalArgs(args []string) []string {
	var pos []string
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
		pos = append(pos, arg)
	}
	return pos
}

func main() {
	refuseInert(os.Args[1:])

	versionInfo := version.Get()

	c, err := cli.New(
		cli.WithCommandName(version.CommandName),
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
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	hideInertCommands(c.Command())
	if err := c.Run(); err != nil {
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
// this generator does not offer. All three are additionally blocked from
// running at all by refuseInert above; they stay in this list too so a bare
// `--help` doesn't list them.
func hideInertCommands(root *cobra.Command) {
	hide(root, editCommand)
	hide(root, createCommand, webhookCommand)
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
