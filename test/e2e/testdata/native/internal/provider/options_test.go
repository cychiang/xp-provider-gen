package provider

import (
	"testing"

	"github.com/alecthomas/kingpin/v2"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
)

func TestFlagsParseRegion(t *testing.T) {
	app := kingpin.New("test", "")
	Flags(app)
	if _, err := app.Parse([]string{"--region=eu-west-1"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if *Region != "eu-west-1" {
		t.Fatalf("region: want eu-west-1, got %q", *Region)
	}
}

func TestConfigureRejectsAnEmptyRegion(t *testing.T) {
	*Region = ""
	if err := Configure(&controller.Options{}); err == nil {
		t.Fatal("Configure with an empty region: want an error, got nil")
	}
	*Region = "us-east-1"
	if err := Configure(&controller.Options{}); err != nil {
		t.Fatalf("Configure with a region: %v", err)
	}
}
