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
	"os/exec"
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

func TestAddSubmodule_PathThatIsNotARepoFailsWithoutTouchingParent(t *testing.T) {
	allowFileSubmodules(t)
	upstream, older, _ := buildUpstream(t)
	git := initTestRepo(t)
	if err := os.Mkdir("build", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("build", "keep"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	head := gitOut(t, ".", "rev-parse", "HEAD")

	err := git.AddSubmodule(context.Background(), upstream, "build", older)
	if err == nil || !strings.Contains(err.Error(), "not a git submodule checkout") {
		t.Fatalf("AddSubmodule error = %v, want one saying build is not a git submodule checkout", err)
	}
	if got := gitOut(t, ".", "rev-parse", "HEAD"); got != head {
		t.Errorf("parent HEAD moved to %s, want %s", got, head)
	}
}

func TestBuildDrift_PrintedCommandMovesAShallowCloneToThePin(t *testing.T) {
	upstream, older, newer := buildUpstream(t)
	git := initTestRepo(t)
	gitOut(t, ".", "clone", "--depth", "1", "file://"+upstream, "build")
	gitOut(t, ".", "add", "build")
	if got := gitlink(t); got != newer {
		t.Fatalf("fixture: gitlink = %s, want upstream HEAD %s", got, newer)
	}

	line, err := git.BuildDrift(context.Background(), "build", older)
	if err != nil || line == "" {
		t.Fatalf("BuildDrift = %q, %v; want a drift line", line, err)
	}
	_, command, ok := strings.Cut(line, "update does not move it: ")
	if !ok {
		t.Fatalf("drift line %q has no command", line)
	}
	if out, err := exec.CommandContext(t.Context(), "sh", "-c", command).CombinedOutput(); err != nil {
		t.Fatalf("running %q: %v\n%s", command, err, out)
	}
	if got := gitlink(t); got != older {
		t.Errorf("gitlink after the printed command = %s, want pin %s", got, older)
	}
}

// TestGeneratorCommitIdentity pins #217: the identity of the generator's
// commits comes from init's flags, then git's config, then the fallback, it
// applies whether or not the repository already existed, and it is never
// written into the repository's config.
func TestGeneratorCommitIdentity(t *testing.T) {
	const (
		flagName  = "Flag Name"
		flagEmail = "flag@example.com"
		repoName  = "Repo Name"
		repoEmail = "repo@example.com"
	)
	flags := core.GitConfig{Author: flagName, Email: flagEmail}
	flagIdentity := flagName + " <" + flagEmail + ">"
	repoIdentity := map[string]string{"user.name": repoName, "user.email": repoEmail}
	fallback := fallbackAuthor + " <" + fallbackEmail + ">"
	tests := []struct {
		name string
		// existing pre-creates the repository with identity in its config;
		// otherwise Init creates it.
		existing      bool
		identity      map[string]string
		flags         core.GitConfig
		wantAuthor    string
		wantCommitter string
	}{
		{
			name: "flags win over the repository's identity", existing: true, identity: repoIdentity, flags: flags,
			wantAuthor: flagIdentity, wantCommitter: flagIdentity,
		},
		{
			name: "git's identity is used without flags", existing: true, identity: repoIdentity,
			wantAuthor: repoName + " <" + repoEmail + ">", wantCommitter: repoName + " <" + repoEmail + ">",
		},
		{
			name: "a missing field falls back to git's value", existing: true, identity: repoIdentity,
			flags:      core.GitConfig{Author: flagName},
			wantAuthor: flagName + " <" + repoEmail + ">", wantCommitter: flagName + " <" + repoEmail + ">",
		},
		{
			name: "flags apply in an existing repository with no identity", existing: true, flags: flags,
			wantAuthor: flagIdentity, wantCommitter: flagIdentity,
		},
		{
			name: "the fallback applies per field", existing: true, flags: core.GitConfig{Author: flagName},
			wantAuthor: flagName + " <" + fallbackEmail + ">", wantCommitter: flagName + " <" + fallbackEmail + ">",
		},
		{
			name: "a repository Init creates gets no identity", flags: flags,
			wantAuthor: flagIdentity, wantCommitter: flagIdentity,
		},
		{
			name:       "no identity anywhere uses the fallback",
			wantAuthor: fallback, wantCommitter: fallback,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newIdentityTestDir(t, tt.existing, tt.identity)
			localBefore := localUserConfig(t)

			ops := NewGitOperations(&core.PluginConfig{Git: tt.flags})
			if err := ops.Init(context.Background()); err != nil {
				t.Fatalf("Init() = %v", err)
			}
			writeAndCommit(t, "a.txt", "generator commit", ops.CreateCommit)

			assertHeadIdentity(t, tt.wantAuthor, tt.wantCommitter)
			if got := localUserConfig(t); got != localBefore {
				t.Errorf("repository user config changed from %q to %q, want it untouched", localBefore, got)
			}
		})
	}
}

// TestFoldAmendKeepsInitsAuthor pins what create api's fold amend does to
// the identity: the author stays the one init committed as, and the
// committer is create api's own identity, since create api has no flags.
func TestFoldAmendKeepsInitsAuthor(t *testing.T) {
	newIdentityTestDir(t, false, nil)
	ctx := context.Background()
	initOps := NewGitOperations(&core.PluginConfig{Git: core.GitConfig{Author: "Flag Name", Email: "flag@example.com"}})
	if err := initOps.Init(ctx); err != nil {
		t.Fatalf("Init() = %v", err)
	}
	writeAndCommit(t, "a.txt", "Initial commit\n\n"+ScaffoldCommitTrailer+"\n", initOps.CreateCommit)

	createAPIOps := NewGitOperations(core.NewPluginConfig())
	writeAndCommit(t, "b.txt", "fold commit", createAPIOps.CommitOrAmendScaffold)

	if got := gitOut(t, "", "rev-list", "--count", "HEAD"); got != "1" {
		t.Errorf("commit count = %s, want 1 (the amend folds into the initial commit)", got)
	}
	assertHeadIdentity(t, "Flag Name <flag@example.com>", fallbackAuthor+" <"+fallbackEmail+">")
}

func assertHeadIdentity(t *testing.T, wantAuthor, wantCommitter string) {
	t.Helper()
	if got := gitOut(t, "", "log", "-1", "--format=%an <%ae>"); got != wantAuthor {
		t.Errorf("author = %q, want %q", got, wantAuthor)
	}
	if got := gitOut(t, "", "log", "-1", "--format=%cn <%ce>"); got != wantCommitter {
		t.Errorf("committer = %q, want %q", got, wantCommitter)
	}
}

// newIdentityTestDir chdirs the test into a new directory with the machine's
// git config hidden. When existing is set, the directory is already a
// repository whose own config sets identity.
func newIdentityTestDir(t *testing.T, existing bool, identity map[string]string) {
	t.Helper()
	isolateGitConfig(t)
	t.Chdir(t.TempDir())
	if !existing {
		return
	}
	gitOut(t, "", "init", "-q")
	for k, v := range identity {
		gitOut(t, "", "config", k, v)
	}
}

func writeAndCommit(t *testing.T, file, message string, commit func(context.Context, string) error) {
	t.Helper()
	if err := os.WriteFile(file, []byte(file+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := commit(context.Background(), message); err != nil {
		t.Fatalf("committing %s: %v", file, err)
	}
}

// isolateGitConfig hides the machine's global and system git config, so the
// test sees only the identity it configures itself.
func isolateGitConfig(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL", "EMAIL"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

// localUserConfig returns the user.* entries of the repository's own config.
func localUserConfig(t *testing.T) string {
	t.Helper()
	out, _ := core.NewGitCommandRunner("").RunCommandWithOutput(context.Background(),
		"config", "--local", "--get-regexp", `^user\.`)
	return out
}
