package widget

import (
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"

	"github.com/xpg-e2e/provider-acme/internal/provider"
)

func TestReconcilerOptionsCarryTheRegionFlag(t *testing.T) {
	o := controller.Options{Logger: logging.NewNopLogger()}
	*provider.Region = ""
	if _, err := ReconcilerOptions(nil, o); err == nil {
		t.Fatal("ReconcilerOptions with an empty region: want an error, got nil")
	}
	*provider.Region = "eu-west-1"
	opts, err := ReconcilerOptions(nil, o)
	if err != nil {
		t.Fatalf("ReconcilerOptions: %v", err)
	}
	if len(opts) != 1 {
		t.Fatalf("ReconcilerOptions: want 1 option, got %d", len(opts))
	}
}
