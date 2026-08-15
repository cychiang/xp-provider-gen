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
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/templates/engine"
)

// testNameRe constrains test names to safe directory / DNS-ish names.
var testNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// NewCreateTestCommand scaffolds a chainsaw behavior test for one of the
// project's kinds: test/behavior/<name>/chainsaw-test.yaml. Missing flags are
// prompted for interactively when stdin is a terminal; in non-interactive use
// (CI, scripts) both flags are required.
func NewCreateTestCommand() *cobra.Command {
	var name, kind string

	cmd := &cobra.Command{
		Use:   "create-test",
		Short: "Scaffold a chainsaw behavior test for a managed resource kind",
		Long: `Scaffold a chainsaw behavior test skeleton under test/behavior/<name>/.

The skeleton applies one managed resource of the chosen kind and asserts it
becomes Ready — a starting point to pin real controller behavior (error paths,
drift, pause). Run it with 'make test-behavior' against a live cluster, or as
part of 'make e2e'.

Missing --name or --kind are prompted for when running interactively.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCreateTest(name, kind, os.Stdin, os.Stdout)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "test name (directory under test/behavior/)")
	cmd.Flags().StringVar(&kind, "kind", "", "managed resource kind the test exercises")
	return cmd
}

func runCreateTest(name, kind string, in io.Reader, out io.Writer) error {
	st, err := loadProjectStore()
	if err != nil {
		return err
	}
	cfg := st.Config()

	kinds, err := managedKinds(cfg)
	if err != nil {
		return err
	}

	prompts := bufio.NewReader(in)
	interactive := stdinIsTerminal(in)
	res, err := resolveKind(kinds, kind, interactive, prompts, out)
	if err != nil {
		return err
	}
	name, err = resolveName(name, interactive, prompts, out)
	if err != nil {
		return err
	}

	scaffold := machinery.NewScaffold(machinery.Filesystem{FS: afero.NewOsFs()}, machinery.WithConfig(cfg))
	gen := engine.NewChainsawTestGenerator(name, res)
	if err := scaffold.Execute(gen); err != nil {
		return fmt.Errorf("scaffolding chainsaw test (does it already exist?): %w", err)
	}

	fmt.Fprintf(out, "Created test/behavior/%s/chainsaw-test.yaml for kind %s.\n", name, res.Kind)
	fmt.Fprintln(out, "Fill in the TODO assertions, then run it with 'make test-behavior' (or 'make e2e').")
	return nil
}

// managedKinds returns the project's managed resource kinds.
func managedKinds(cfg config.Config) ([]resource.Resource, error) {
	all, err := cfg.GetResources()
	if err != nil {
		return nil, fmt.Errorf("reading project resources: %w", err)
	}
	kinds := engine.ManagedResources(all)
	if len(kinds) == 0 {
		return nil, fmt.Errorf("no managed resource kinds in this project; run 'create api' first")
	}
	return kinds, nil
}

// resolveKind picks the kind under test: by flag (case-insensitive), by the
// only kind there is, or by interactive choice.
func resolveKind(
	kinds []resource.Resource, flag string, interactive bool, in *bufio.Reader, out io.Writer,
) (resource.Resource, error) {
	if flag != "" {
		for _, r := range kinds {
			if strings.EqualFold(r.Kind, flag) {
				return r, nil
			}
		}
		return resource.Resource{}, fmt.Errorf("kind %q is not in this project (have: %s)", flag, kindNames(kinds))
	}
	if len(kinds) == 1 {
		return kinds[0], nil
	}
	if !interactive {
		return resource.Resource{}, fmt.Errorf("--kind is required (project has multiple kinds: %s)", kindNames(kinds))
	}

	fmt.Fprintln(out, "Which kind should the test exercise?")
	for i, r := range kinds {
		fmt.Fprintf(out, "  %d) %s (%s/%s)\n", i+1, r.Kind, r.Group, r.Version)
	}
	fmt.Fprint(out, "Enter a number: ")
	line, err := readLine(in)
	if err != nil {
		return resource.Resource{}, err
	}
	i, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || i < 1 || i > len(kinds) {
		return resource.Resource{}, fmt.Errorf("invalid choice %q: enter 1-%d", strings.TrimSpace(line), len(kinds))
	}
	return kinds[i-1], nil
}

// resolveName returns the test name from the flag or an interactive prompt.
func resolveName(flag string, interactive bool, in *bufio.Reader, out io.Writer) (string, error) {
	name := flag
	if name == "" {
		if !interactive {
			return "", fmt.Errorf("--name is required")
		}
		fmt.Fprint(out, "Test name (e.g. drift-check): ")
		line, err := readLine(in)
		if err != nil {
			return "", err
		}
		name = strings.TrimSpace(line)
	}
	if !testNameRe.MatchString(name) {
		return "", fmt.Errorf("invalid test name %q: use lowercase letters, digits and '-'", name)
	}
	return name, nil
}

func kindNames(kinds []resource.Resource) string {
	names := make([]string, 0, len(kinds))
	for _, r := range kinds {
		names = append(names, r.Kind)
	}
	return strings.Join(names, ", ")
}

func readLine(in *bufio.Reader) (string, error) {
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("reading input (pass --name and --kind to run non-interactively): %w", err)
	}
	return line, nil
}

// stdinIsTerminal reports whether in is an interactive terminal. Only a real
// *os.File character device counts; anything else (pipes, CI) is
// non-interactive so missing flags fail fast instead of hanging on a prompt.
func stdinIsTerminal(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
