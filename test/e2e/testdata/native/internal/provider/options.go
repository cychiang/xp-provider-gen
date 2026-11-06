package provider

import (
	"github.com/alecthomas/kingpin/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
)

// Flags registers a provider flag, as the scaffold's comment shows, so a
// user-edited options.go has content the harness can see update leave alone.
func Flags(app *kingpin.Application) {
	app.Flag("e2e-marker", "Set by the e2e harness.").String()
}

func Configure(_ *controller.Options) error {
	return nil
}
