//go:build !windows

// Command e2e runs the declared end-to-end stages for one flavor against a
// scaffold built by the binary under test. See docs/testing.md.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func main() { os.Exit(run()) }

func run() int {
	flavor := flag.String("flavor", "native", "flavor to test")
	mode := flag.String("mode", string(ModeFull),
		"full runs every stage and fails without Docker; fast runs the stages that need none and reports PARTIAL")
	from := flag.String("from", fromHead, "binary that scaffolds: head, or the tag of a published release such as v0.3.0")
	list := flag.Bool("list", false, "print the declared stages and exit; with --from a release, only those that apply")
	bin := flag.String("bin", "bin/xp-provider-gen", "binary under test")
	overlay := flag.String("overlay", "",
		"overlay tree, laid over the scaffold at the same paths (default test/e2e/testdata/<flavor>)")
	logs := flag.String("logs", "", "directory for per-stage logs (default: a new temporary directory)")
	flag.Parse()

	c, err := NewCase(*flavor, *from)
	if errors.Is(err, ErrNotApplicable) {
		fmt.Printf("VERDICT NOT-APPLICABLE, nothing ran: %v\n", err)
		return 0
	}
	if err != nil {
		return fail(err)
	}
	f, err := flavorFor(c.Flavor)
	if err != nil {
		return fail(err)
	}
	stages := f.Stages()
	if *list {
		printStages(c, stages)
		return 0
	}
	if *overlay == "" {
		*overlay = filepath.Join("test/e2e/testdata", f.Name)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if c.Release() {
		listed, err := newestReleases(ctx)
		if err == nil {
			err = Preflight(listed)
		}
		if err != nil {
			return fail(err)
		}
	}
	have := map[Capability]bool{CapDocker: dockerUp(ctx)}
	plan, err := NewPlan(c, stages, Mode(*mode), have)
	if err != nil {
		return fail(err)
	}

	env, err := newEnv(ctx, c, f, *bin, *overlay, *logs)
	if err != nil {
		return fail(err)
	}
	fmt.Printf("work %s\nlogs %s\ncluster %s\ngenerator %s\n", env.Work, env.Logs, env.Cluster, env.Gen)
	if c.Toolchain != "" {
		fmt.Printf("GOTOOLCHAIN=%s on the released binary's init and create api only\n", c.Toolchain)
	}

	verdict := Execute(plan, env, os.Stdout)
	env.cleanup(plan.Mode == ModeFull, verdict != VerdictFailed)
	if verdict == VerdictFailed {
		return 1
	}
	return 0
}

func printStages(c Case, stages []Stage) {
	for _, s := range stages {
		if c.Release() && s.When != nil && !s.When(c) {
			continue
		}
		fmt.Printf("%-20s needs=%v release-only=%t\n", s.Name, s.Needs, s.When != nil && !s.When(Case{From: fromHead}))
	}
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "e2e:", err)
	return 1
}

func dockerUp(ctx context.Context) bool {
	return exec.CommandContext(ctx, "docker", "info").Run() == nil
}

// newEnv makes every path private to this run, so parallel runs cannot
// collide and none can touch the caller's kubeconfig or clusters.
func newEnv(ctx context.Context, c Case, f *Flavor, bin, overlay, logs string) (*Env, error) {
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
		Ctx:            ctx,
		Repo:           repoRoot,
		Head:           absolute(repoRoot, bin),
		Gen:            absolute(repoRoot, bin),
		Overlay:        absolute(repoRoot, overlay),
		Work:           work,
		Dir:            filepath.Join(work, provider),
		Logs:           absolute(repoRoot, logs),
		Kubeconfig:     filepath.Join(work, "kubeconfig"),
		Cluster:        "e2e-" + strings.TrimPrefix(filepath.Base(work), "xpg-e2e-"),
		F:              f,
		Toolchain:      c.Toolchain,
		CreateAPIGuard: c.CreateAPIGuard,
	}
	if err := e.prepare(ctx, c); err != nil {
		e.cleanup(false, true)
		return nil, err
	}
	return e, nil
}

// prepare fetches the release and makes the directories. A failure here
// returns before run's own cleanup, so newEnv removes the work directory.
func (e *Env) prepare(ctx context.Context, c Case) error {
	if c.Release() {
		gen, err := fetchRelease(ctx, c.From, e.Work)
		if err != nil {
			return err
		}
		e.Gen = gen
	}
	if _, err := os.Stat(e.Head); err != nil {
		return fmt.Errorf("finding generator binary (run make build): %w", err)
	}
	if err := os.MkdirAll(e.Dir, 0o750); err != nil {
		return err
	}
	return os.MkdirAll(e.Logs, 0o750)
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
