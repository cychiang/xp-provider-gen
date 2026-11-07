package provider

import (
	"errors"

	"github.com/alecthomas/kingpin/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
)

// Region is the --region flag's value, shared with the controllers.
var Region = new(string)

// Flags registers the provider's own flag, as the scaffold's comment shows.
func Flags(app *kingpin.Application) {
	Region = app.Flag("region", "Target region.").Envar("REGION").Default("us-east-1").String()
}

// Configure rejects a flag combination before startup.
func Configure(_ *controller.Options) error {
	if *Region == "" {
		return errors.New("--region must not be empty")
	}
	return nil
}
