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
	"slices"
	"strings"

	"github.com/spf13/cobra"

	crossplanev2 "github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2"
	"github.com/cychiang/xp-provider-gen/pkg/version"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

// newRootCommand builds the command tree. The root keeps cobra's default Args,
// whose only check rejects an unknown subcommand of the root itself.
func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use: version.CommandName,
		Long: "Crossplane Provider Generator - A tool for scaffolding Crossplane providers " +
			"and managed resources following Crossplane v2 patterns",
		SilenceUsage: true,
	}
	root.AddCommand(
		crossplanev2.NewInitCommand(),
		newCreateCommand(),
		crossplanev2.NewCreateTestCommand(),
		crossplanev2.NewUpdateCommand(),
		&cobra.Command{
			Use:   "version",
			Short: "Print the " + version.CommandName + " version",
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, _ []string) {
				fmt.Fprintln(cmd.OutOrStdout(), version.Get().Short())
			},
		},
	)
	root.SetHelpCommand(newHelpCommand())
	return root
}

// newCreateCommand returns `create`, whose only subcommand is `api`. Cobra
// checks the arguments of a non-root command only after it has handled --help,
// so `create webhook --help` would print this help and succeed. Flag parsing
// is off so that every argument reaches RunE, which rejects anything but help.
func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "create",
		Short:              "Scaffold a Crossplane API",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				switch {
				case slices.Contains([]string{"-h", "--help"}, arg):
				case strings.HasPrefix(arg, "-"):
					return fmt.Errorf("unknown flag: %s", arg)
				default:
					return unknownCommand(cmd, arg)
				}
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(crossplanev2.NewCreateAPICommand())
	return cmd
}

// newHelpCommand replaces cobra's `help`, which prints the help of the nearest
// command it finds: `help create webhook` would print `create`'s help and succeed.
func newHelpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, rest, err := cmd.Root().Find(args)
			if err != nil {
				return err
			}
			if len(rest) > 0 {
				return unknownCommand(target, rest[0])
			}
			return target.Help()
		},
	}
}

func unknownCommand(parent *cobra.Command, name string) error {
	return fmt.Errorf("unknown command %q for %q", name, parent.CommandPath())
}
