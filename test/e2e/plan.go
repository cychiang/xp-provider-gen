package main

import (
	"errors"
	"fmt"
	"io"
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

// Case is one cell of the matrix: a flavor and the binary that scaffolds it.
// From is "head" for the binary under test, or the path of a released binary.
type Case struct {
	Flavor string
	From   string
}

const fromHead = "head"

func (c Case) Release() bool { return c.From != fromHead }

// Stage is one declared step. It returns an error or it passed: there is no
// skip result, so a required stage cannot pass without running.
type Stage struct {
	Name  string
	Needs []Capability
	// When limits the stage to some cases. Nil means every case.
	When func(Case) bool
	Run  func(*Env) error
}

// Plan is what a mode will do with the stages that apply to a case.
type Plan struct {
	Mode   Mode
	Run    []Stage
	NotRun []Stage
}

// NewPlan rejects an unknown mode and a plan with nothing to run. It fails in full mode when a stage needs a capability the host lacks,
// so the run stops before the first stage. Fast mode sets those stages aside.
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
		var lacking []Capability
		for _, need := range s.Needs {
			if !have[need] {
				lacking = append(lacking, need)
			}
		}
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

type Verdict string

const (
	VerdictComplete Verdict = "COMPLETE"
	VerdictPartial  Verdict = "PARTIAL"
	VerdictFailed   Verdict = "FAILED"
)

// Execute runs the plan in order and stops at the first failure. Every stage
// that did not run is named in the output.
func Execute(p Plan, env *Env, out io.Writer) Verdict {
	failed := ""
	for _, s := range p.Run {
		if failed != "" {
			fmt.Fprintf(out, "NOT-RUN %-20s after %s failed\n", s.Name, failed)
			continue
		}
		start := time.Now()
		err := env.runStage(s)
		elapsed := time.Since(start).Round(time.Second)
		if err != nil {
			failed = s.Name
			fmt.Fprintf(out, "FAIL    %-20s %6s %v\n", s.Name, elapsed, err)
			continue
		}
		fmt.Fprintf(out, "PASS    %-20s %6s\n", s.Name, elapsed)
	}
	var notRun []string
	for _, s := range p.NotRun {
		fmt.Fprintf(out, "NOT-RUN %-20s needs %v\n", s.Name, s.Needs)
		notRun = append(notRun, s.Name)
	}
	switch {
	case failed != "":
		fmt.Fprintf(out, "VERDICT FAILED at %s\n", failed)
		return VerdictFailed
	case len(notRun) > 0:
		fmt.Fprintf(out, "VERDICT PARTIAL, not run: %s\n", strings.Join(notRun, ", "))
		return VerdictPartial
	}
	fmt.Fprintf(out, "VERDICT COMPLETE, %d stages passed\n", len(p.Run))
	return VerdictComplete
}
