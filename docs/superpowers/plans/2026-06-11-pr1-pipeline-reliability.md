# PR 1: Pipeline Reliability — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Make `init` and `create api` leave a clean, fully-generated, committed working tree by committing *after* all file-producing steps, making every step required, and asserting a clean tree in the e2e test.

**Architecture:** Reorder the step lists in `automation/pipeline.go` so the commit is last; flip every step's `required` flag to true so a failure aborts instead of warning; add a `git status --porcelain` clean-tree assertion to `scripts/e2e-test.sh` after init and after each `create api`.

**Tech Stack:** Go, `go test`, bash e2e (`scripts/e2e-test.sh`).

---

### Task 1: Pipeline ordering + required-ness unit test

**Files:**
- Test: Create `pkg/plugins/crossplane/v2/automation/pipeline_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
	"testing"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

func stepNames(p *Pipeline) []string {
	names := make([]string, 0, len(p.steps))
	for _, s := range p.steps {
		names = append(names, s.Name())
	}
	return names
}

func allRequired(p *Pipeline) bool {
	for _, s := range p.steps {
		if !s.IsRequired() {
			return false
		}
	}
	return true
}

func TestNewInitPipeline_CommitsLastAndAllRequired(t *testing.T) {
	cfg := core.NewPluginConfig()
	p := NewInitPipeline(cfg, "provider-test")

	got := stepNames(p)
	want := []string{
		"Initialize git repository",
		"Add build submodule from " + cfg.Git.BuildSubmoduleURL,
		"Run make submodules",
		"Download dependencies (go mod tidy)",
		"Run make generate",
		"Run make reviewable",
		"Create initial commit",
	}
	if len(got) != len(want) {
		t.Fatalf("step count = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("step %d = %q, want %q", i, got[i], want[i])
		}
	}
	if !allRequired(p) {
		t.Error("all init pipeline steps must be required")
	}
}

func TestNewAPICommitPipeline_CommitsLastAndAllRequired(t *testing.T) {
	cfg := core.NewPluginConfig()
	p := NewAPICommitPipeline(cfg, "Bucket")

	got := stepNames(p)
	want := []string{
		"Run make generate",
		"Create initial commit",
	}
	if len(got) != len(want) {
		t.Fatalf("step count = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("step %d = %q, want %q", i, got[i], want[i])
		}
	}
	if !allRequired(p) {
		t.Error("all api pipeline steps must be required")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/plugins/crossplane/v2/automation/ -run Pipeline -v`
Expected: FAIL — init order has commit at index 1 (not last); steps not all required.

> Note: `NewPluginConfig()` is the existing constructor in `core/config.go`. Confirm its name during implementation; if different, use the actual one.

---

### Task 2: Reorder pipelines so commit is last

**Files:**
- Modify: `pkg/plugins/crossplane/v2/automation/pipeline.go`

- [ ] **Step 1: Reorder both pipelines**

Replace the `steps` slices:

```go
func NewInitPipeline(config *core.PluginConfig, providerName string) *Pipeline {
	commitMessage := fmt.Sprintf(`Initial commit

Scaffolded Crossplane provider project for %s`, providerName)

	return &Pipeline{
		steps: []Step{
			NewGitInitStep(config),
			NewGitSubmoduleStep(config),
			NewMakeStep("submodules", true),
			NewGoModTidyStep(true),
			NewMakeStep("generate", true),
			NewMakeStep("reviewable", true),
			NewGitCommitStep(config, commitMessage),
		},
	}
}

func NewAPICommitPipeline(config *core.PluginConfig, resourceKind string) *Pipeline {
	commitMessage := fmt.Sprintf(`Add %s managed resource

Scaffolded CRD, controller, and client code for %s resource`, resourceKind, resourceKind)

	return &Pipeline{
		steps: []Step{
			NewMakeStep("generate", true),
			NewGitCommitStep(config, commitMessage),
		},
	}
}
```

---

### Task 3: Make git steps required

**Files:**
- Modify: `pkg/plugins/crossplane/v2/automation/steps.go`

- [ ] **Step 1: Flip the three git step constructors to required**

In `NewGitInitStep`, `NewGitCommitStep`, and `NewGitSubmoduleStep`, change `required: false` to `required: true`. (The `MakeStep`/`GoModTidyStep` requiredness is already passed by the pipeline in Task 2.)

- [ ] **Step 2: Run the unit test to verify it passes**

Run: `go test ./pkg/plugins/crossplane/v2/automation/ -run Pipeline -v`
Expected: PASS (both tests).

- [ ] **Step 3: Run full unit tests + lint**

Run: `make test && golangci-lint run --config .golangci.yml`
Expected: PASS, 0 lint issues.

- [ ] **Step 4: Commit**

```bash
git add pkg/plugins/crossplane/v2/automation/
git commit -m "fix(pipeline): commit after generation and make all steps required

Previously the init/create-api pipelines committed before running
submodules/tidy/generate/reviewable, leaving the working tree dirty
and the 'initial commit' incomplete. Reorder so the commit is last and
captures the fully generated, formatted tree; make every step required
so a failure aborts loudly instead of reporting false success."
```

---

### Task 4: Assert a clean tree in the e2e test

**Files:**
- Modify: `scripts/e2e-test.sh`

- [ ] **Step 1: Add a clean-tree assertion helper**

After the `verify_files_exist` function, add:

```bash
assert_clean_tree() {
    local context=$1
    log_info "Asserting clean git tree after $context..."
    local dirty
    dirty="$(git status --porcelain)"
    if [[ -n "$dirty" ]]; then
        log_error "Working tree is dirty after $context:"
        echo "$dirty"
        return 1
    fi
    log_success "Working tree is clean after $context"
}
```

- [ ] **Step 2: Call it after init and after the second API creation**

After the init reviewable step (end of Step 3 block in the script) add:
```bash
    assert_clean_tree "init"
```
After the post-API `reviewable` (end of Step 6 block) add:
```bash
    assert_clean_tree "create api"
```

- [ ] **Step 3: Run the e2e test**

Run: `make build && ./scripts/e2e-test.sh`
Expected: PASS, including both "Working tree is clean" assertions. (If dirty, inspect what generation leaves untracked and confirm the generated `.gitignore.tmpl` covers it — fix the template, not the assertion.)

- [ ] **Step 4: Commit**

```bash
git add scripts/e2e-test.sh
git commit -m "test(e2e): assert clean git tree after init and create api

Regression guard for the dirty-tree bug: the generated provider must
have no uncommitted/untracked files after init or after adding an API."
```

---

## Self-review

- **Spec coverage:** Implements spec §3.6 (required steps, clean-tree assertion) and the "generate-then-commit" half of the §2 git decision for `init`/`create api`. (`update`'s no-commit behavior is PR 5.)
- **Placeholder scan:** none — all steps show exact code/commands.
- **Type consistency:** uses existing `NewPluginConfig`, `NewInitPipeline`, `NewAPICommitPipeline`, `New*Step` constructors; only the `required` literals and step order change.
- **Risk:** the clean-tree assertion depends on the generated `.gitignore` covering build artifacts; Task 4 Step 3 handles the failure mode explicitly.
