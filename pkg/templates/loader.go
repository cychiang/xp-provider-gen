package templates

import (
	"embed"
	"fmt"
)

//go:embed files files/project/.gitignore.tmpl upjet upjet/project/.gitignore.tmpl generators
var TemplateFS embed.FS

// GeneratorBody returns the template body for a generator-emitted file.
// Generator bodies live under generators/ — deliberately outside files/, so
// the scaffold auto-discovery never renders them directly; the generators
// that own them supply their computed data instead. The FS is embedded at
// compile time, so a missing name is a build defect and panics.
func GeneratorBody(name string) string {
	body, err := TemplateFS.ReadFile("generators/" + name)
	if err != nil {
		panic(fmt.Errorf("reading embedded generator body %q: %w", name, err))
	}
	return string(body)
}
