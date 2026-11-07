//go:build !windows

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
	"syscall"
)

func main() { os.Exit(run()) }

// options are the flags of one run.
type options struct {
	flavor, mode, from, bin, overlay, logs, report string
	list                                           bool
}

func run() int {
	var o options
	flag.StringVar(&o.flavor, "flavor", "native", "flavor to test")
	flag.StringVar(&o.mode, "mode", string(ModeFull),
		"full runs every stage and fails without Docker; fast runs the stages that need none and reports PARTIAL")
	flag.StringVar(&o.from, "from", fromHead,
		"binary that scaffolds: head, latest (the newest published release), or a release tag such as v0.3.0")
	flag.BoolVar(&o.list, "list", false,
		"print the declared stages and exit; with --from a release, only those that apply")
	flag.StringVar(&o.bin, "bin", "bin/xp-provider-gen", "binary under test")
	flag.StringVar(&o.overlay, "overlay", "",
		"overlay tree, laid over the scaffold at the same paths (default test/e2e/testdata/<flavor>)")
	flag.StringVar(&o.logs, "logs", "", "directory for per-stage logs (default: a new temporary directory)")
	flag.StringVar(&o.report, "report", "", "file to write the JSON report to, also when the run fails")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rep := Report{Schema: reportSchema, Flavor: o.flavor, Verdict: VerdictFailed}
	code := runCase(ctx, o, &rep)
	if o.report != "" {
		if err := rep.write(o.report); err != nil {
			return fail(err)
		}
	}
	return code
}

// runCase fills rep as far as the run gets, so a run that fails still has a report.
func runCase(ctx context.Context, o options, rep *Report) int {
	tag, err := resolveFrom(ctx, o.from)
	if err != nil {
		return fail(err)
	}
	c := NewCase(o.flavor, tag)
	if c.Release() {
		rep.From = c.From
	}
	f, err := flavorFor(c.Flavor)
	if err != nil {
		return fail(err)
	}
	stages := f.Stages()
	if o.list {
		printStages(c, stages)
		return 0
	}
	if o.overlay == "" {
		o.overlay = filepath.Join("test/e2e/testdata", f.Name)
	}

	have := map[Capability]bool{CapDocker: dockerUp(ctx)}
	plan, err := NewPlan(c, stages, Mode(o.mode), have)
	if err != nil {
		return fail(err)
	}

	env, err := newEnv(ctx, c, f, o.bin, o.overlay, o.logs)
	if err != nil {
		return fail(err)
	}
	fmt.Printf("work %s\nlogs %s\ncluster %s\ngenerator %s\n", env.Work, env.Logs, env.Cluster, env.Gen)
	if c.Toolchain != "" {
		fmt.Printf("GOTOOLCHAIN=%s on the released binary's init and create api only\n", c.Toolchain)
	}
	rep.Head = versionOf(ctx, env.Head)

	verdict, results := Execute(plan, env, os.Stdout)
	rep.Verdict, rep.Stages, rep.Upgrade = verdict, results, env.Upgrade
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

// versionOf is the version the binary reports, empty when it cannot say.
func versionOf(ctx context.Context, bin string) string {
	out, err := exec.CommandContext(ctx, bin, "version").Output() // #nosec G204 -- the binary under test, no shell
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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
		F:          f,
		Toolchain:  c.Toolchain,
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
