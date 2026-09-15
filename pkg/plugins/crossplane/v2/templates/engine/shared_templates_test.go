package engine

import (
	"bytes"
	"testing"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

// identicalAcrossFlavors are the templates both flavor roots carry verbatim.
// Each root stays self-contained (one directory shows everything a flavor
// gets), so these are deliberate copies; this list is what keeps them copies.
var identicalAcrossFlavors = []string{
	"LICENSE.tmpl",
	"hack/boilerplate.go.txt.tmpl",
	"package/crossplane.yaml.tmpl",
	"project/.gitignore.tmpl",
	"project/OWNERS.md.tmpl",
}

func TestSharedTemplatesStayIdentical(t *testing.T) {
	for _, rel := range identicalAcrossFlavors {
		native := readTemplate(t, core.FlavorNative.TemplateRoot()+"/"+rel)
		upjet := readTemplate(t, core.FlavorUpjet.TemplateRoot()+"/"+rel)
		if !bytes.Equal(native, upjet) {
			t.Errorf("%s differs between the %s and %s roots; these copies must stay identical — apply the change to both",
				rel, core.FlavorNative.TemplateRoot(), core.FlavorUpjet.TemplateRoot())
		}
	}
}

func readTemplate(t *testing.T, path string) []byte {
	t.Helper()
	body, err := templates.TemplateFS.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return body
}
