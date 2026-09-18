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
	"errors"
	"testing"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// Display names of the make steps several pipelines share.
const (
	stepNameMakeGenerate   = "Run make generate"
	stepNameMakeReviewable = "Run make reviewable"
)

// fakeStep records whether it ran and optionally fails.
type fakeStep struct {
	name string
	err  error
	ran  *bool
}

func (s fakeStep) Name() string   { return s.name }
func (s fakeStep) Execute() error { *s.ran = true; return s.err }

func stepNames(p *Pipeline) []string {
	names := make([]string, 0, len(p.steps))
	for _, s := range p.steps {
		names = append(names, s.Name())
	}
	return names
}

func assertStepOrder(t *testing.T, p *Pipeline, want []string) {
	t.Helper()
	got := stepNames(p)
	if len(got) != len(want) {
		t.Fatalf("step count = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("step %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNewInitPipeline_CommitsLast(t *testing.T) {
	cfg := core.NewPluginConfig("crossplane")
	p := newInitPipeline(cfg, "provider-test")

	assertStepOrder(t, p, []string{
		"Initialize git repository",
		"Mark scaffolded scripts executable",
		"Add build submodule from " + cfg.Git.BuildSubmoduleURL,
		"Run make submodules",
		"Download dependencies (go mod tidy)",
		stepNameMakeGenerate,
		stepNameMakeReviewable,
		stepNameInitialCommit,
	})
}

func TestUpdateFinalizePipelines(t *testing.T) {
	const stepNameGoModTidy = "Run go mod tidy"
	tests := []struct {
		name     string
		pipeline *Pipeline
		want     []string
	}{
		{"native", newUpdateFinalizePipeline(), []string{stepNameGoModTidy, stepNameMakeGenerate, stepNameMakeReviewable}},
		// Upjet generates first: tidy fails until the generated packages exist.
		{"upjet", newUpjetUpdateFinalizePipeline(), []string{stepNameMakeGenerate, stepNameGoModTidy, stepNameMakeReviewable}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStepOrder(t, tt.pipeline, tt.want)
		})
	}
}

// TestUpdateFinalizePipelineFor pins that update finalizes each flavor with its
// own pipeline: swapping them passes every step-order test above and fails
// only in e2e.
func TestUpdateFinalizePipelineFor(t *testing.T) {
	tests := []struct {
		flavor core.Flavor
		want   *Pipeline
	}{
		{core.FlavorNative, newUpdateFinalizePipeline()},
		{core.FlavorUpjet, newUpjetUpdateFinalizePipeline()},
	}
	for _, tt := range tests {
		t.Run(string(tt.flavor), func(t *testing.T) {
			assertStepOrder(t, UpdateFinalizePipelineFor(tt.flavor), stepNames(tt.want))
		})
	}
}

// TestInitPipelineFor pins that init scaffolds each flavor with its own
// pipeline: swapping them passes every step-order test above and fails only
// in e2e.
func TestInitPipelineFor(t *testing.T) {
	cfg := core.NewPluginConfig("crossplane")
	tests := []struct {
		flavor core.Flavor
		want   *Pipeline
	}{
		{core.FlavorNative, newInitPipeline(cfg, "provider-test")},
		{core.FlavorUpjet, newUpjetInitPipeline(cfg, "provider-test")},
	}
	for _, tt := range tests {
		t.Run(string(tt.flavor), func(t *testing.T) {
			assertStepOrder(t, InitPipelineFor(tt.flavor, cfg, "provider-test"), stepNames(tt.want))
		})
	}
}

// TestAPICommitPipelineFor pins that create api commits each flavor's
// resource with its own pipeline: swapping them passes every step-order test
// above and fails only in e2e.
func TestAPICommitPipelineFor(t *testing.T) {
	cfg := core.NewPluginConfig("crossplane")
	tests := []struct {
		flavor core.Flavor
		want   *Pipeline
	}{
		{core.FlavorNative, newAPICommitPipeline(cfg, "Bucket")},
		{core.FlavorUpjet, newUpjetAPICommitPipeline(cfg, "Bucket")},
	}
	for _, tt := range tests {
		t.Run(string(tt.flavor), func(t *testing.T) {
			assertStepOrder(t, APICommitPipelineFor(tt.flavor, cfg, "Bucket"), stepNames(tt.want))
		})
	}
}

// TestInitPipelines_ShareLeadingStepsAndFinalStep pins the invariant
// pipeline.go's own comments describe but never enforce: newInitPipeline and
// newUpjetInitPipeline share their first four steps (git init, executable
// bit, git submodule, make submodules) and their last (the commit), diverging
// only in the middle (native tidies/generates/reviews; upjet just downloads,
// since a fresh upjet project doesn't compile until `make generate` runs).
//
// The two pipelines are compared directly against EACH OTHER, not against a
// hardcoded list of expected literal step names — the point is to catch one
// prefix drifting away from the other (someone adds a step to one and
// forgets the other), not to re-assert today's exact wording. Step.Name()
// already exists on the interface (Pipeline.Run itself prints it, and
// TestNewInitPipeline_CommitsLast above already keys assertions off it), so
// no new accessor was needed: it is already the stable, meaningful
// identifier this codebase uses for "which step is this".
func TestInitPipelines_ShareLeadingStepsAndFinalStep(t *testing.T) {
	cfg := core.NewPluginConfig("crossplane")
	native := stepNames(newInitPipeline(cfg, "provider-test"))
	upjet := stepNames(newUpjetInitPipeline(cfg, "provider-test"))

	tests := []struct {
		desc     string
		position int // index into each pipeline's step list; -1 means "last"
	}{
		{"1st step (git init)", 0},
		{"2nd step (executable bit)", 1},
		{"3rd step (git submodule)", 2},
		{"4th step (make submodules)", 3},
		{"final step (commit)", -1},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ni, ok := resolveIndex(tt.position, len(native))
			if !ok {
				t.Fatalf("native init pipeline has no step at position %d (has %d steps: %v)", tt.position, len(native), native)
			}
			ui, ok := resolveIndex(tt.position, len(upjet))
			if !ok {
				t.Fatalf("upjet init pipeline has no step at position %d (has %d steps: %v)", tt.position, len(upjet), upjet)
			}
			if native[ni] != upjet[ui] {
				t.Errorf("%s diverges between init pipelines: native has %q, upjet has %q", tt.desc, native[ni], upjet[ui])
			}
		})
	}
}

// resolveIndex turns a possibly-negative logical index (-1 = last) into an
// absolute index into a slice of the given length, reporting false if it is
// out of range.
func resolveIndex(i, length int) (int, bool) {
	if i < 0 {
		i += length
	}
	if i < 0 || i >= length {
		return 0, false
	}
	return i, true
}

func TestNewAPICommitPipeline_CommitsLast(t *testing.T) {
	cfg := core.NewPluginConfig("crossplane")
	p := newAPICommitPipeline(cfg, "Bucket")

	assertStepOrder(t, p, []string{
		stepNameMakeGenerate,
		"Commit changes (fold into initial scaffold if applicable)",
	})
}

func TestPipeline_Run_AbortsOnFirstFailure(t *testing.T) {
	firstRan, secondRan := false, false
	wantErr := errors.New("boom")
	p := &Pipeline{steps: []Step{
		fakeStep{name: "first", err: wantErr, ran: &firstRan},
		fakeStep{name: "second", ran: &secondRan},
	}}

	err := p.Run()
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want wrapped %v", err, wantErr)
	}
	if !firstRan {
		t.Error("first step should have run")
	}
	if secondRan {
		t.Error("second step must not run after a failure")
	}
}

// TestPipeline_Run_StreamingStepFailureMessage pins the full message a user
// sees when a streaming step fails: the step name once, then the exit status,
// with no repeated command and no trailing space when there are no arguments.
func TestPipeline_Run_StreamingStepFailureMessage(t *testing.T) {
	tests := []struct {
		step *StreamingCommandStep
		want string
	}{
		{step: NewStreamingCommandStep("go", "bogus-subcommand"), want: "Run go bogus-subcommand failed: exit status 2"},
		{step: NewStreamingCommandStep("go"), want: "Run go failed: exit status 2"},
	}
	for _, tt := range tests {
		t.Run(tt.step.Name(), func(t *testing.T) {
			err := (&Pipeline{steps: []Step{tt.step}}).Run()
			if err == nil || err.Error() != tt.want {
				t.Fatalf("Run() error = %q, want %q", err, tt.want)
			}
		})
	}
}
