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

package automation

import (
	"fmt"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

type Pipeline struct {
	steps []Step
}

func NewInitPipeline(config *core.PluginConfig, providerName string) *Pipeline {
	commitMessage := fmt.Sprintf(`Initial commit

Scaffolded Crossplane provider project for %s`, providerName)

	return &Pipeline{
		steps: []Step{
			NewGitInitStep(config),
			NewGitCommitStep(config, commitMessage),
			NewGitSubmoduleStep(config),
		},
	}
}

func (p *Pipeline) Run() error {
	for i, step := range p.steps {
		fmt.Printf("  %d. %s...\n", i+1, step.Name())

		if err := step.Execute(); err != nil {
			if step.IsRequired() {
				return fmt.Errorf("%s failed (required): %w", step.Name(), err)
			}
			fmt.Printf("    Warning: %s: %v (continuing...)\n", step.Name(), err)
		}
	}

	return nil
}
