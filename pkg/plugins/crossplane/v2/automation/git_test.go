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
