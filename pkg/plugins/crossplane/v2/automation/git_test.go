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
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// initTestRepo creates a real git repository in a temp dir, configures a
// commit identity, and makes one initial commit so HEAD exists. It chdirs
// the test into that repo, since GitOperations always runs against the
// process cwd (core.NewGitCommandRunner("")).
func initTestRepo(t *testing.T) *GitOperations {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)

	run := func(args ...string) {
		t.Helper()
		g := core.NewGitCommandRunner("")
		if err := g.RunCommand(context.Background(), args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	run("init")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed\n"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "seed commit")

	return &GitOperations{
		config: &core.PluginConfig{},
		runner: core.NewGitCommandRunner(""),
	}
}

func commitCount(t *testing.T) int {
	t.Helper()
	g := core.NewGitCommandRunner("")
	out, err := g.RunCommandWithOutput(context.Background(), "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list --count: %v", err)
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		t.Fatalf("parse rev-list count %q: %v", out, err)
	}
	return n
}

func TestCreateCommit_NoChanges_SkipsWithoutError(t *testing.T) {
	git := initTestRepo(t)
	before := commitCount(t)

	if err := git.CreateCommit(context.Background(), "no-op commit"); err != nil {
		t.Fatalf("CreateCommit() with no changes = %v, want nil", err)
	}

	if after := commitCount(t); after != before {
		t.Errorf("commit count = %d, want unchanged %d", after, before)
	}
}

func TestCreateCommit_WithChanges_Commits(t *testing.T) {
	git := initTestRepo(t)
	before := commitCount(t)

	if err := os.WriteFile("new.txt", []byte("data\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := git.CreateCommit(context.Background(), "add new.txt"); err != nil {
		t.Fatalf("CreateCommit() with changes = %v", err)
	}

	if after := commitCount(t); after != before+1 {
		t.Errorf("commit count = %d, want %d", after, before+1)
	}
}

func TestCommitOrAmendScaffold_NoChanges_SkipsWithoutError(t *testing.T) {
	git := initTestRepo(t)
	before := commitCount(t)
	g := core.NewGitCommandRunner("")
	headBefore, err := g.RunCommandWithOutput(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}

	if err := git.CommitOrAmendScaffold(context.Background(), "fold commit"); err != nil {
		t.Fatalf("CommitOrAmendScaffold() with no changes = %v, want nil", err)
	}

	if after := commitCount(t); after != before {
		t.Errorf("commit count = %d, want unchanged %d", after, before)
	}
	headAfter, err := g.RunCommandWithOutput(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	if headAfter != headBefore {
		t.Errorf("HEAD changed from %s to %s, want unchanged (no amend on no-op)", headBefore, headAfter)
	}
}

func TestCommitOrAmendScaffold_WithChanges_AmendsScaffoldHead(t *testing.T) {
	git := initTestRepo(t)
	g := core.NewGitCommandRunner("")

	// Make HEAD look like the tool's own scaffold commit.
	if err := os.WriteFile("scaffold.txt", []byte("scaffold\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := g.RunCommand(context.Background(), "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	scaffoldMsg := "Initial commit\n\n" + ScaffoldCommitTrailer + "\n"
	if err := g.RunCommandWithStdin(context.Background(), scaffoldMsg, "commit", "-F", "-"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	before := commitCount(t)

	if err := os.WriteFile("more.txt", []byte("more\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := git.CommitOrAmendScaffold(context.Background(), "fold commit"); err != nil {
		t.Fatalf("CommitOrAmendScaffold() with changes = %v", err)
	}

	if after := commitCount(t); after != before {
		t.Errorf("commit count = %d, want unchanged %d (amend, not a new commit)", after, before)
	}
	msg, err := g.RunCommandWithOutput(context.Background(), "log", "-1", "--pretty=%B")
	if err != nil {
		t.Fatalf("log -1: %v", err)
	}
	if !strings.Contains(msg, ScaffoldCommitTrailer) {
		t.Errorf("amended commit message = %q, want it to still carry the scaffold trailer", msg)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := core.NewGitCommandRunner(dir).RunCommandWithOutput(context.Background(), args...)
	if err != nil {
		t.Fatalf("git %v in %s: %v", args, dir, err)
	}
	return out
}

// buildUpstream makes a repository with two commits and returns its path and
// both SHAs, the older one first. The newer is the upstream's HEAD.
func buildUpstream(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	gitOut(t, dir, "init")
	gitOut(t, dir, "config", "user.name", "Test User")
	gitOut(t, dir, "config", "user.email", "test@example.com")
	for _, name := range []string{"one", "two"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		gitOut(t, dir, "add", ".")
		gitOut(t, dir, "commit", "-m", name)
	}
	return dir, gitOut(t, dir, "rev-parse", "HEAD~1"), gitOut(t, dir, "rev-parse", "HEAD")
}

func allowFileSubmodules(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "protocol.file.allow")
	t.Setenv("GIT_CONFIG_VALUE_0", "always")
}

func gitlink(t *testing.T) string {
	t.Helper()
	fields := strings.Fields(gitOut(t, ".", "ls-files", "--stage", "--", "build"))
	if len(fields) < 2 || fields[0] != gitlinkMode {
		t.Fatalf("build is not a gitlink in the index: %v", fields)
	}
	return fields[1]
}

func TestAddSubmodule_PinsRefAndStaysThroughSubmoduleUpdate(t *testing.T) {
	allowFileSubmodules(t)
	upstream, older, _ := buildUpstream(t)
	git := initTestRepo(t)
	ctx := context.Background()

	if err := git.AddSubmodule(ctx, upstream, "build", older); err != nil {
		t.Fatalf("AddSubmodule: %v", err)
	}
	if got := gitlink(t); got != older {
		t.Errorf("gitlink = %s, want build_ref %s", got, older)
	}
	if err := git.runner.RunCommand(ctx, "submodule", "update", "--init", "--recursive"); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, "build", "rev-parse", "HEAD"); got != older {
		t.Errorf("build checked out at %s after submodule update, want %s", got, older)
	}
}

func TestAddSubmodule_RerunOverInterruptedScaffoldEndsAtRef(t *testing.T) {
	allowFileSubmodules(t)
	upstream, older, newer := buildUpstream(t)
	git := initTestRepo(t)
	ctx := context.Background()

	if err := git.runner.AddSubmodule(ctx, upstream, "build"); err != nil {
		t.Fatal(err)
	}
	if got := gitlink(t); got != newer {
		t.Fatalf("fixture: interrupted scaffold gitlink = %s, want upstream HEAD %s", got, newer)
	}
	if err := git.AddSubmodule(ctx, upstream, "build", older); err != nil {
		t.Fatalf("AddSubmodule rerun: %v", err)
	}
	if got := gitlink(t); got != older {
		t.Errorf("gitlink = %s, want build_ref %s", got, older)
	}
	if got := gitOut(t, "build", "rev-parse", "HEAD"); got != older {
		t.Errorf("build checked out at %s, want %s", got, older)
	}
}

func TestAddSubmodule_MissingRefFailsNamingIt(t *testing.T) {
	allowFileSubmodules(t)
	upstream, _, _ := buildUpstream(t)
	git := initTestRepo(t)
	missing := strings.Repeat("ab", 20)

	err := git.AddSubmodule(context.Background(), upstream, "build", missing)
	if err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("AddSubmodule error = %v, want one naming %s", err, missing)
	}
}

func TestBuildDrift(t *testing.T) {
	allowFileSubmodules(t)
	upstream, older, newer := buildUpstream(t)
	git := initTestRepo(t)
	ctx := context.Background()

	if got, err := git.BuildDrift(ctx, "build", older); err != nil || got != "" {
		t.Errorf("no submodule: BuildDrift = %q, %v; want empty", got, err)
	}
	if err := git.runner.AddSubmodule(ctx, upstream, "build"); err != nil {
		t.Fatal(err)
	}
	if got, err := git.BuildDrift(ctx, "build", newer); err != nil || got != "" {
		t.Errorf("gitlink matches: BuildDrift = %q, %v; want empty", got, err)
	}
	got, err := git.BuildDrift(ctx, "build", older)
	if err != nil || !strings.Contains(got, newer) || !strings.Contains(got, older) {
		t.Errorf("gitlink differs: BuildDrift = %q, %v; want a line naming %s and %s", got, err, newer, older)
	}
}

func TestBuildDrift_GitlinkWithNoLocalObject(t *testing.T) {
	git := initTestRepo(t)
	unknown := strings.Repeat("cd", 20)
	gitOut(t, ".", "update-index", "--add", "--cacheinfo", gitlinkMode+","+unknown+",build")

	got, err := git.BuildDrift(context.Background(), "build", strings.Repeat("ab", 20))
	if err != nil || !strings.Contains(got, unknown) {
		t.Errorf("BuildDrift = %q, %v; want a drift line naming the unfetched %s", got, err, unknown)
	}
}
