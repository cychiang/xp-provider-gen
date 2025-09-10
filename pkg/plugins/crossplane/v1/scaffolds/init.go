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

package scaffolds

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v1/scaffolds/internal/templates"
)


// InitScaffolder scaffolds the project structure for a Crossplane provider
type InitScaffolder struct {
	config      config.Config
	boilerplate string
}

// NewInitScaffolder returns a new InitScaffolder for Crossplane provider projects
func NewInitScaffolder(config config.Config) *InitScaffolder {
	return &InitScaffolder{
		config: config,
		boilerplate: `/*
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
*/`,
	}
}

// Scaffold scaffolds the Crossplane provider project structure
func (s *InitScaffolder) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project structure...\n")

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(s.boilerplate),
	)

	// Execute scaffolding templates for Crossplane provider
	if err := scaffold.Execute(
		&templates.GoMod{},
		&templates.Main{},
		&templates.Makefile{},
		&templates.Dockerfile{},
		&templates.ReadMe{},
		&templates.GitIgnore{},
		&templates.GitModules{},
	); err != nil {
		return fmt.Errorf("error scaffolding Crossplane provider project: %w", err)
	}

	fmt.Printf("Crossplane provider project scaffolded successfully!\n")
	return nil
}