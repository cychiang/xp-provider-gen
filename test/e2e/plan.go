//go:build !windows

package main

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

type Mode string

const (
	ModeFull Mode = "full"
	ModeFast Mode = "fast"
)

var ErrUnknownMode = errors.New("unknown mode")

// Capability is a host prerequisite a stage cannot run without.
type Capability string

const CapDocker Capability = "docker"

// Case is a flavor and the binary that scaffolds it.
// From is "head" for the binary under test, or the tag of a published release.
type Case struct {
	Flavor string
	From   string
	// Toolchain is the GOTOOLCHAIN a released binary's init and create api run
	// under. Empty for head and for any tag toolchains does not list.
	Toolchain string
}

const (
	fromHead   = "head"
	fromLatest = "latest"
)

func (c Case) Release() bool { return c.From != fromHead }

func (c Case) Head() bool { return !c.Release() }

// toolchains lists the releases whose init and create api need the Go they
// were built with, so the scaffold is the one the release produced whatever Go
// the host has. A tag not listed runs under the host's own.
var toolchains = map[string]string{"v0.3.0": "go1.26.8"}

// NewCase resolves a --from value that is already a tag or "head".
func NewCase(flavor, from string) Case {
	return Case{Flavor: flavor, From: from, Toolchain: toolchains[from]}
}

// Stage is one declared step. It returns an error or it passed: there is no
// skip result, so a required stage cannot pass without running.
type Stage struct {
	Name  string
	Needs []Capability
	// When limits the stage to some cases. Nil means every case.
	When func(Case) bool
	Run  func(*Env) error
	// Background starts the stage where it is declared and lets the stages
	// after it run beside it, until one with Join and before the verdict. Such
	// a stage may share nothing the stages beside it touch, so whatever it reads
	// is made before it starts. The one exception is a resource only the Join
	// stage uses beside it, and the Background stage must free that first.
	Background bool
	// Join makes the stage wait for every running Background stage first.
	Join bool
}

// Plan is what a mode will do with the stages that apply to a case.
type Plan struct {
	Mode   Mode
	Run    []Stage
	NotRun []Stage
}

// NewPlan rejects an unknown mode and a plan with nothing to run. It fails in
// full mode when a stage needs a capability the host lacks, so the run stops
// before the first stage. Fast mode sets those stages aside.
func NewPlan(c Case, stages []Stage, mode Mode, have map[Capability]bool) (Plan, error) {
	if mode != ModeFull && mode != ModeFast {
		return Plan{}, fmt.Errorf("%w %q: want %q or %q", ErrUnknownMode, mode, ModeFull, ModeFast)
	}
	p := Plan{Mode: mode}
	var missing []string
	for _, s := range stages {
		if s.When != nil && !s.When(c) {
			continue
		}
		lacking := lacks(s.Needs, have)
		switch {
		case mode == ModeFast && len(s.Needs) > 0:
			p.NotRun = append(p.NotRun, s)
		case len(lacking) > 0:
			missing = append(missing, fmt.Sprintf("%s needs %v", s.Name, lacking))
		default:
			p.Run = append(p.Run, s)
		}
	}
	if len(missing) > 0 {
		return Plan{}, fmt.Errorf("full mode cannot run, nothing was started: %s", strings.Join(missing, "; "))
	}
	if len(p.Run) == 0 {
		return Plan{}, fmt.Errorf("%s mode has no stage to run, so a pass would prove nothing", mode)
	}
	return p, nil
}

func lacks(needs []Capability, have map[Capability]bool) []Capability {
	var lacking []Capability
	for _, need := range needs {
		if !have[need] {
			lacking = append(lacking, need)
		}
	}
	return lacking
}

type Verdict string

const (
	VerdictComplete Verdict = "COMPLETE"
	VerdictPartial  Verdict = "PARTIAL"
	VerdictFailed   Verdict = "FAILED"
)

// outcome is how one stage ended; i is its place in Plan.Run.
type outcome struct {
	i       int
	err     error
	elapsed time.Duration
}

// execution is the state of one Execute: the results in declared order, the
// first stage that failed, and the background stages still running.
type execution struct {
	plan     Plan
	out      io.Writer
	results  []StageResult
	failed   string
	finished chan outcome
	running  int
}

func (x *execution) take(o outcome) {
	s := x.plan.Run[o.i]
	status, line := StatusPass, fmt.Sprintf("PASS    %-20s %6s", s.Name, o.elapsed.Round(time.Second))
	if o.err != nil {
		status, line = StatusFail, fmt.Sprintf("FAIL    %-20s %6s %v", s.Name, o.elapsed.Round(time.Second), o.err)
		if x.failed == "" {
			x.failed = s.Name
		}
	}
	fmt.Fprintln(x.out, line)
	x.results[o.i] = StageResult{Name: s.Name, Status: status, Seconds: tenths(o.elapsed)}
}

func (x *execution) join() {
	for ; x.running > 0; x.running-- {
		x.take(<-x.finished)
	}
}

// start runs stage i on its own copy of env, so its log is its own.
func (x *execution) start(env *Env, i int) {
	x.running++
	own := *env
	go func() {
		start := time.Now()
		err := own.runStage(x.plan.Run[i])
		x.finished <- outcome{i, err, time.Since(start)}
	}()
}

// Execute runs the plan in order and stops at the first failure. A background
// stage runs beside the stages after it, and its outcome is taken when it is
// joined. Every stage that did not run is named in the output and in the
// results, which are in declared order.
func Execute(p Plan, env *Env, out io.Writer) (Verdict, []StageResult) {
	x := &execution{plan: p, out: out, results: make([]StageResult, len(p.Run)), finished: make(chan outcome)}
	for i, s := range p.Run {
		if s.Join {
			x.join()
		}
		switch {
		case x.failed != "":
			fmt.Fprintf(out, "NOT-RUN %-20s after %s failed\n", s.Name, x.failed)
			x.results[i] = StageResult{Name: s.Name, Status: StatusNotRun}
		case s.Background:
			x.start(env, i)
		default:
			start := time.Now()
			err := env.runStage(s)
			x.take(outcome{i, err, time.Since(start)})
		}
	}
	x.join()
	var notRun []string
	var skipped []StageResult
	for _, s := range p.NotRun {
		fmt.Fprintf(out, "NOT-RUN %-20s needs %v\n", s.Name, s.Needs)
		notRun = append(notRun, s.Name)
		skipped = append(skipped, StageResult{Name: s.Name, Status: StatusNotRun})
	}
	results := slices.Concat(x.results, skipped)
	switch {
	case x.failed != "":
		fmt.Fprintf(out, "VERDICT FAILED at %s\n", x.failed)
		return VerdictFailed, results
	case len(notRun) > 0:
		fmt.Fprintf(out, "VERDICT PARTIAL, not run: %s\n", strings.Join(notRun, ", "))
		return VerdictPartial, results
	}
	fmt.Fprintf(out, "VERDICT COMPLETE, %d stages passed\n", len(p.Run))
	return VerdictComplete, results
}
