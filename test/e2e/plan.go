//go:build !windows

package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
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

// Case is one cell of the matrix: a flavor and the binary that scaffolds it.
// From is "head" for the binary under test, or the tag of a published release.
type Case struct {
	Flavor string
	From   string
	// Toolchain is the GOTOOLCHAIN a released binary's init and create api run
	// under. Empty for head.
	Toolchain string
	// CreateAPIGuard is whether the release's create api refuses a project a
	// different clean release last updated.
	CreateAPIGuard bool
}

const fromHead = "head"

func (c Case) Release() bool { return c.From != fromHead }

// Release is what the harness knows about one published release.
type Release struct {
	// Toolchain is the GOTOOLCHAIN the release's init and create api run
	// under: its own go_version, so the scaffold is the one the release
	// produced whatever Go the host has.
	Toolchain string
	// NotApplicable maps a flavor to the reason this release cannot run it.
	NotApplicable map[string]string
	// CreateAPIGuard is whether create api refuses a project a different clean
	// release last updated. update and update --adopt refuse a newer project
	// in every release. v0.1.0 predates the create api check and accepts.
	CreateAPIGuard bool
}

// Releases lists every published release. A new release fails the preflight
// until it is added here.
var Releases = map[string]Release{
	"v0.1.0": {Toolchain: toolchain1268},
	"v0.2.0": {Toolchain: toolchain1268, CreateAPIGuard: true},
	"v0.3.0": {Toolchain: toolchain1268, CreateAPIGuard: true},
}

const toolchain1268 = "go1.26.8"

var (
	ErrUnknownRelease = errors.New("unknown release")
	ErrNotApplicable  = errors.New("not applicable")
	ErrNoReleases     = errors.New("release list is empty")
)

// NewCase resolves a --from value. A release pair marked NotApplicable returns
// ErrNotApplicable with the declared reason, so the caller can say why nothing ran.
func NewCase(flavor, from string) (Case, error) {
	c := Case{Flavor: flavor, From: from}
	if !c.Release() {
		return c, nil
	}
	r, ok := Releases[from]
	if !ok {
		return c, fmt.Errorf("%w %q: declared releases are %s", ErrUnknownRelease, from, strings.Join(releaseTags(), ", "))
	}
	if reason, na := r.NotApplicable[flavor]; na {
		return c, fmt.Errorf("%w: %s from %s: %s", ErrNotApplicable, flavor, from, reason)
	}
	c.Toolchain, c.CreateAPIGuard = r.Toolchain, r.CreateAPIGuard
	return c, nil
}

func releaseTags() []string {
	tags := slices.Collect(maps.Keys(Releases))
	slices.Sort(tags)
	return tags
}

// Preflight fails when the newest published release, first in listed, has no
// entry in Releases: a release the harness never ran must not pass unnoticed.
func Preflight(listed []string) error {
	if len(listed) == 0 {
		return ErrNoReleases
	}
	if _, ok := Releases[listed[0]]; !ok {
		return fmt.Errorf("newest release %s is missing from Releases in test/e2e/plan.go: add it with its toolchain",
			listed[0])
	}
	return nil
}

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
