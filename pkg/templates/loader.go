package templates

import (
	"embed"
)

//go:embed files files/project/.gitignore.tmpl
var TemplateFS embed.FS
