//go:build !windows

package main

import (
	"fmt"
	"os"
	"strings"
)

// The files a scaffold must contain. These lists are the one place that says so:
// the stages that scaffold check them right after init and create api.
const (
	makefile       = "Makefile"
	agentsDoc      = "AGENTS.md"
	ownershipDoc   = "docs/ownership.md"
	providerMain   = "cmd/provider/main.go"
	connectorGo    = "internal/provider/connector.go"
	makeFragment   = "hack/xp-provider-gen.mk"
	configProvider = "config/provider.go"
	configZZ       = "config/zz_resources.go"
	probeConfig    = "config/secret/config.go"
)

var (
	nativeBaseLayout = []string{
		makefile, makeFragment, "go.mod", ".gitignore", "apis", providerMain,
		"internal/controller", connectorGo, "internal/provider/client.go",
		"internal/provider/options.go", "cluster/local/integration_tests.sh", "test/setup.sh", "test/README.md",
		ownershipDoc, agentsDoc,
	}
	upjetBaseLayout = []string{
		makefile, makeFragment, configProvider, configZZ,
		"internal/clients/clients.go", "internal/clients/resolve.go", "cmd/generator/main.go", "apis/generate.go",
	}
)

// nativeKindLayout is what create api adds for one native kind. The example is
// the single source of a valid manifest for the kind: uptest's lifecycle input,
// create-test's input and the docs.
func nativeKindLayout(kind string) []string {
	lower := strings.ToLower(kind)
	return []string{
		"apis/" + group + "/" + apiVersion + "/" + lower + "_types.go",
		"internal/controller/" + lower + "/external.go",
		"internal/controller/" + lower + "/wiring.go",
		"examples/" + group + "/" + lower + ".yaml",
		"test/behavior/" + lower + "-pause/chainsaw-test.yaml",
	}
}

// assertLayout names every file of want that the scaffold lacks.
func (e *Env) assertLayout(want []string) error {
	var missing []string
	for _, rel := range want {
		if _, err := os.Stat(e.path(rel)); err != nil {
			missing = append(missing, rel)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("the scaffold lacks:\n%s", strings.Join(missing, "\n"))
	}
	return nil
}
