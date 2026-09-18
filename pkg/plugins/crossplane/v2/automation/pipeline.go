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

// scaffoldCommitMessage builds the initial-scaffold commit message shared by
// newInitPipeline and newUpjetInitPipeline, which differ only in how they
// describe what was scaffolded (e.g. "Crossplane provider project" vs.
// "upjet Crossplane provider project"). Their step lists, by contrast, are
// kept flat and duplicated on purpose (see
// TestInitPipelines_ShareLeadingStepsAndFinalStep in pipeline_test.go).
func scaffoldCommitMessage(description, providerName string) string {
	return fmt.Sprintf(`Initial commit

Scaffolded %s for %s

%s`, description, providerName, ScaffoldCommitTrailer)
}

// newUpjetInitPipeline is the init pipeline for an upjet provider. It stops
// short of building: a freshly scaffolded upjet project deliberately does not
// compile yet, because cmd/provider imports the API and controller packages
// that `make generate` produces from the Terraform schema. Running tidy or
// reviewable here would fail on work the user has not been able to do.
func newUpjetInitPipeline(config *core.PluginConfig, providerName string) *Pipeline {
	commitMessage := scaffoldCommitMessage("upjet Crossplane provider project", providerName)

	return &Pipeline{
		steps: []Step{
			NewGitInitStep(config),
			NewExecutableBitStep(""),
			NewGitSubmoduleStep(config),
			NewMakeStep("submodules"),
			NewGoModDownloadStep(),
			NewGitCommitStep(config, commitMessage),
		},
	}
}

func newInitPipeline(config *core.PluginConfig, providerName string) *Pipeline {
	commitMessage := scaffoldCommitMessage("Crossplane provider project", providerName)

	return &Pipeline{
		steps: []Step{
			NewGitInitStep(config),
			NewExecutableBitStep(""),
			NewGitSubmoduleStep(config),
			NewMakeStep("submodules"),
			NewGoModTidyStep(),
			NewMakeStep("generate"),
			NewMakeStep("reviewable"),
			NewGitCommitStep(config, commitMessage),
		},
	}
}

// InitPipelineFor returns init's post-scaffold pipeline for a project of the
// given flavor. It is the one place init chooses between the two, mirroring
// UpdateFinalizePipelineFor.
func InitPipelineFor(flavor core.Flavor, config *core.PluginConfig, providerName string) *Pipeline {
	if flavor == core.FlavorUpjet {
		return newUpjetInitPipeline(config, providerName)
	}
	return newInitPipeline(config, providerName)
}

// newUpjetAPICommitPipeline commits a newly configured upjet resource without
// running `make generate`. Generation there downloads Terraform, the provider
// schema and the provider's docs, then runs the upjet pipeline — minutes of
// network work the user should start deliberately, not as a side effect of
// adding a resource.
func newUpjetAPICommitPipeline(config *core.PluginConfig, resourceKind string) *Pipeline {
	commitMessage := fmt.Sprintf(`Configure %s managed resource

Added the upjet configuration for %s; run 'make generate' to generate its
API types and controller.`, resourceKind, resourceKind)

	return &Pipeline{
		steps: []Step{
			NewGitFoldCommitStep(config, commitMessage),
		},
	}
}

func newAPICommitPipeline(config *core.PluginConfig, resourceKind string) *Pipeline {
	commitMessage := fmt.Sprintf(`Add %s managed resource

Scaffolded CRD, controller, and client code for %s resource`, resourceKind, resourceKind)

	return &Pipeline{
		steps: []Step{
			NewMakeStep("generate"),
			NewGitFoldCommitStep(config, commitMessage),
		},
	}
}

// APICommitPipelineFor returns create api's post-scaffold commit pipeline for
// a project of the given flavor. It is the one place create api chooses
// between the two, mirroring UpdateFinalizePipelineFor.
func APICommitPipelineFor(flavor core.Flavor, config *core.PluginConfig, resourceKind string) *Pipeline {
	if flavor == core.FlavorUpjet {
		return newUpjetAPICommitPipeline(config, resourceKind)
	}
	return newAPICommitPipeline(config, resourceKind)
}

// newUpdateFinalizePipeline brings a native provider back to a reviewable
// state after `update` has refreshed its files and dependency versions. Every
// step streams, since together they take minutes.
func newUpdateFinalizePipeline() *Pipeline {
	return &Pipeline{
		steps: []Step{
			NewStreamingCommandStep("go", "mod", "tidy"),
			NewStreamingCommandStep("make", "generate"),
			NewStreamingCommandStep("make", "reviewable"),
		},
	}
}

// newUpjetUpdateFinalizePipeline is newUpdateFinalizePipeline for an upjet
// provider, and differs only in order: `make generate` runs before `go mod
// tidy`. cmd/provider imports the API and controller packages that generation
// produces from the Terraform schema, so tidy fails on a project where they do
// not exist yet, while the generator itself only needs modules go.mod already
// requires. Generating first finalizes a never-generated project too, and
// re-generating is not optional on an already-generated one: the generated
// code must track the upjet version `update` just applied. Generation needs
// network access (Terraform, the provider schema) and takes minutes on a cold
// cache. Verified on an e2e-generated provider: never-generated, generated,
// and after a framework dependency bump.
func newUpjetUpdateFinalizePipeline() *Pipeline {
	return &Pipeline{
		steps: []Step{
			NewStreamingCommandStep("make", "generate"),
			NewStreamingCommandStep("go", "mod", "tidy"),
			NewStreamingCommandStep("make", "reviewable"),
		},
	}
}

// UpdateFinalizePipelineFor returns update's finalize pipeline for a project of
// the given flavor. It is the one place update chooses between the two.
func UpdateFinalizePipelineFor(flavor core.Flavor) *Pipeline {
	if flavor == core.FlavorUpjet {
		return newUpjetUpdateFinalizePipeline()
	}
	return newUpdateFinalizePipeline()
}

func (p *Pipeline) Run() error {
	for i, step := range p.steps {
		fmt.Printf("  %d. %s...\n", i+1, step.Name())

		if err := step.Execute(); err != nil {
			return fmt.Errorf("%s failed: %w", step.Name(), err)
		}
	}

	return nil
}
