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

package scaffold

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/templates/engine"
	"github.com/cychiang/xp-provider-gen/pkg/versions"
)

type InitScaffolder struct {
	config config.Config
	flavor core.Flavor
	upjet  *core.UpjetSettings
}

// NewInitScaffolder builds the scaffolder for one project flavor. upjet is nil
// for native providers.
func NewInitScaffolder(config config.Config, flavor core.Flavor, upjet *core.UpjetSettings) *InitScaffolder {
	return &InitScaffolder{config: config, flavor: flavor, upjet: upjet}
}

func (s *InitScaffolder) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project structure...\n")

	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(engine.DefaultBoilerplate()),
	)

	factory := engine.NewFactoryForFlavor(s.config, s.flavor)

	initTemplates, err := factory.GetInitTemplates(engine.WithUpjet(s.upjet))
	if err != nil {
		return fmt.Errorf("failed to get init templates: %w", err)
	}

	allTemplates := engine.AsBuilders(initTemplates)

	// Seed the registration files through the same deterministic generators used
	// by `create api` (with no managed resources yet), so init and create produce
	// byte-identical register.go for the base case — one source of truth.
	deps, err := s.dependencies()
	if err != nil {
		return fmt.Errorf("failed to load dependency manifest: %w", err)
	}
	if s.flavor == core.FlavorUpjet {
		allTemplates = append(allTemplates, engine.UpjetCoreGenerators(s.config, nil)...)
	} else {
		allTemplates = append(allTemplates, engine.CoreGenerators(s.config, nil)...)
	}
	allTemplates = append(allTemplates, engine.NewGoModGenerator(s.config.GetRepository(), deps))

	if err := scaffold.Execute(allTemplates...); err != nil {
		return fmt.Errorf("error scaffolding Crossplane provider project: %w", err)
	}

	fmt.Printf("Crossplane provider project scaffolded successfully!\n")

	return nil
}

// dependencies returns the go.mod dependency set for this project's flavor.
func (s *InitScaffolder) dependencies() ([]versions.Dependency, error) {
	if s.flavor == core.FlavorUpjet {
		return versions.UpjetGoModDependencies()
	}
	return versions.GoModDependencies()
}
