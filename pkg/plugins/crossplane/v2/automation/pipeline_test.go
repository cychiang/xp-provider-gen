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
	p := NewInitPipeline(cfg, "provider-test")

	assertStepOrder(t, p, []string{
		"Initialize git repository",
		"Add build submodule from " + cfg.Git.BuildSubmoduleURL,
		"Run make submodules",
		"Download dependencies (go mod tidy)",
		"Run make generate",
		"Run make reviewable",
		stepNameInitialCommit,
	})
}

func TestNewAPICommitPipeline_CommitsLast(t *testing.T) {
	cfg := core.NewPluginConfig("crossplane")
	p := NewAPICommitPipeline(cfg, "Bucket")

	assertStepOrder(t, p, []string{
		"Run make generate",
		stepNameInitialCommit,
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
