package templates

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func GitIgnore(cfg config.Config) machinery.Template {
	return StaticFile(cfg, ".gitignore", gitIgnoreTemplate)
}

const gitIgnoreTemplate = `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/
_output/

# Test binary, built with go test -c
*.test

# Output of the go coverage tool
*.out
coverage.*

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Build artifacts
build/_output/`
