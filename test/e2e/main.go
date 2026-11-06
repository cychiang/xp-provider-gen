// Command e2e runs the declared end-to-end stages for one flavor against a
// scaffold built by the binary under test. See docs/testing.md.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
)

func main() { os.Exit(run()) }

func run() int {
	flavor := flag.String("flavor", "native", "flavor to test")
	mode := flag.String("mode", string(ModeFull),
		"full runs every stage and fails without Docker; fast runs the stages that need none and reports PARTIAL")
	from := flag.String("from", fromHead, "binary that scaffolds: head, or the path of a released binary")
	list := flag.Bool("list", false, "print the declared stages and exit")
	bin := flag.String("bin", "bin/xp-provider-gen", "binary under test")
	overlay := flag.String("overlay", "test/e2e/testdata/native", "overlay tree, laid over the scaffold at the same paths")
	logs := flag.String("logs", "", "directory for per-stage logs (default: a new temporary directory)")
	flag.Parse()

	c := Case{Flavor: *flavor, From: *from}
	stages, err := stagesFor(c.Flavor)
	if err != nil {
		return fail(err)
	}
	if *list {
		for _, s := range stages {
			fmt.Printf("%-20s needs=%v release-only=%t\n", s.Name, s.Needs, s.When != nil)
		}
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	have := map[Capability]bool{CapDocker: dockerUp(ctx)}
	plan, err := NewPlan(c, stages, Mode(*mode), have)
	if err != nil {
		return fail(err)
	}

	env, err := newEnv(ctx, c, *bin, *overlay, *logs)
	if err != nil {
		return fail(err)
	}
	fmt.Printf("work %s\nlogs %s\ncluster %s\n", env.Work, env.Logs, env.Cluster)

	verdict := Execute(plan, env, os.Stdout)
	env.cleanup(plan.Mode == ModeFull, verdict != VerdictFailed)
	if verdict == VerdictFailed {
		return 1
	}
	return 0
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "e2e:", err)
	return 1
}

func stagesFor(flavor string) ([]Stage, error) {
	if flavor != "native" {
		return nil, fmt.Errorf("no stages declared for flavor %q", flavor)
	}
	return nativeStages(), nil
}

func dockerUp(ctx context.Context) bool {
	return exec.CommandContext(ctx, "docker", "info").Run() == nil
}

// newEnv makes every path private to this run, so parallel runs cannot
// collide and none can touch the caller's kubeconfig or clusters.
func newEnv(ctx context.Context, c Case, bin, overlay, logs string) (*Env, error) {
	repoRoot, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	work, err := os.MkdirTemp("", "xpg-e2e-")
	if err != nil {
		return nil, err
	}
	if logs == "" {
		if logs, err = os.MkdirTemp("", "xpg-e2e-logs-"); err != nil {
			return nil, err
		}
	}
	e := &Env{
		Ctx:        ctx,
		Repo:       repoRoot,
		Head:       absolute(repoRoot, bin),
		Gen:        absolute(repoRoot, bin),
		Overlay:    absolute(repoRoot, overlay),
		Work:       work,
		Dir:        filepath.Join(work, provider),
		Logs:       absolute(repoRoot, logs),
		Kubeconfig: filepath.Join(work, "kubeconfig"),
		Cluster:    "e2e-" + strings.TrimPrefix(filepath.Base(work), "xpg-e2e-"),
	}
	if c.Release() {
		e.Gen = absolute(repoRoot, c.From)
	}
	for _, p := range []string{e.Head, e.Gen} {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("finding generator binary (run make build): %w", err)
		}
	}
	if err := os.MkdirAll(e.Dir, 0o750); err != nil {
		return nil, err
	}
	return e, os.MkdirAll(e.Logs, 0o750)
}

func absolute(root, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

// cleanup deletes this run's cluster whatever the verdict, even after an
// interrupt. The work directory goes too unless a stage failed, when it stays
// for inspection.
func (e *Env) cleanup(usedCluster, remove bool) {
	e.Ctx, e.log = context.Background(), os.Stderr
	if usedCluster {
		_, _ = e.exec(e.Work, "", e.tool("kind"), "delete", "cluster", "--name", e.Cluster)
	}
	if remove {
		_ = os.RemoveAll(e.Work)
	}
}
