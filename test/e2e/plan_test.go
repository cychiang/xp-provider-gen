package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func counting(name string, ran *[]string, err error, needs ...Capability) Stage {
	return Stage{Name: name, Needs: needs, Run: func(*Env) error {
		*ran = append(*ran, name)
		return err
	}}
}

func TestFullModeFailsBeforeAnyStageWhenACapabilityIsMissing(t *testing.T) {
	var ran []string
	stages := []Stage{
		counting("init", &ran, nil),
		counting("install", &ran, nil, CapDocker),
	}
	_, err := NewPlan(Case{From: fromHead}, stages, ModeFull, map[Capability]bool{})
	if err == nil {
		t.Fatal("full mode planned a run on a host without docker")
	}
	for _, want := range []string{"install", string(CapDocker)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
	if len(ran) != 0 {
		t.Errorf("stages ran before the failure: %v", ran)
	}
}

func TestFastModeSetsDockerStagesAsideAndReportsPartial(t *testing.T) {
	var ran []string
	stages := []Stage{
		counting("init", &ran, nil),
		counting("install", &ran, nil, CapDocker),
		counting("delete", &ran, nil, CapDocker),
	}
	for name, have := range map[string]map[Capability]bool{
		"without docker": {},
		"with docker":    {CapDocker: true},
	} {
		t.Run(name, func(t *testing.T) {
			ran = nil
			plan, err := NewPlan(Case{From: fromHead}, stages, ModeFast, have)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if v := Execute(plan, &Env{}, &out); v != VerdictPartial {
				t.Fatalf("verdict %s, want PARTIAL", v)
			}
			if strings.Join(ran, ",") != "init" {
				t.Errorf("ran %v, want only init", ran)
			}
			for _, want := range []string{"VERDICT PARTIAL", "NOT-RUN install", "NOT-RUN delete", "not run: install, delete"} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("output lacks %q:\n%s", want, out.String())
				}
			}
		})
	}
}

func TestFirstFailureStopsTheRun(t *testing.T) {
	var ran []string
	stages := []Stage{
		counting("one", &ran, nil),
		counting("two", &ran, errors.New("boom")),
		counting("three", &ran, nil),
	}
	plan, err := NewPlan(Case{From: fromHead}, stages, ModeFull, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if v := Execute(plan, &Env{}, &out); v != VerdictFailed {
		t.Fatalf("verdict %s, want FAILED", v)
	}
	if strings.Join(ran, ",") != "one,two" {
		t.Errorf("ran %v, want one,two", ran)
	}
	for _, want := range []string{"FAIL    two", "NOT-RUN three", "VERDICT FAILED at two"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
}

func TestFullModeWithEveryStagePassingIsComplete(t *testing.T) {
	var ran []string
	stages := []Stage{counting("one", &ran, nil), counting("two", &ran, nil, CapDocker)}
	plan, err := NewPlan(Case{From: fromHead}, stages, ModeFull, map[Capability]bool{CapDocker: true})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if v := Execute(plan, &Env{}, &out); v != VerdictComplete {
		t.Fatalf("verdict %s, want COMPLETE:\n%s", v, out.String())
	}
}

func TestReleaseOnlyStagesRunForReleaseCasesOnly(t *testing.T) {
	var ran []string
	release := counting("update-to-head", &ran, nil)
	release.When = Case.Release
	stages := []Stage{counting("init", &ran, nil), release}
	for from, want := range map[string]string{fromHead: "init", "/bin/v0.3.0": "init,update-to-head"} {
		ran = nil
		plan, err := NewPlan(Case{From: from}, stages, ModeFull, nil)
		if err != nil {
			t.Fatal(err)
		}
		Execute(plan, &Env{}, &bytes.Buffer{})
		if got := strings.Join(ran, ","); got != want {
			t.Errorf("from %s ran %s, want %s", from, got, want)
		}
	}
}

func TestNativeStagesAreDeclaredInPlanOrder(t *testing.T) {
	want := "init create-api generate user-logic update-to-head update-noop create-api-refresh adopt create-test " +
		"stock-e2e build-xpkg install-in-kind provider-config mr-ready-synced mr-update drift-corrected " +
		"mr-delete external-gone"
	var got []string
	for _, s := range nativeStages() {
		got = append(got, s.Name)
	}
	if strings.Join(got, " ") != want {
		t.Errorf("stages\n got %v\nwant %s", got, want)
	}
}

func TestNativeFastModeRunsExactlyTheDockerFreeStages(t *testing.T) {
	plan, err := NewPlan(Case{From: fromHead}, nativeStages(), ModeFast, map[Capability]bool{})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range plan.Run {
		got = append(got, s.Name)
	}
	want := "init create-api generate user-logic update-noop create-api-refresh adopt create-test"
	if strings.Join(got, " ") != want {
		t.Errorf("fast mode runs\n got %v\nwant %s", got, want)
	}
}

func TestUnknownModeIsRejectedBeforeAnythingRuns(t *testing.T) {
	var ran []string
	stages := []Stage{counting("init", &ran, nil)}
	_, err := NewPlan(Case{From: fromHead}, stages, Mode("ful"), map[Capability]bool{CapDocker: true})
	if !errors.Is(err, ErrUnknownMode) {
		t.Fatalf("error %v, want ErrUnknownMode", err)
	}
	if len(ran) != 0 {
		t.Errorf("stages ran: %v", ran)
	}
}

func TestAPlanWithNothingToRunIsRejected(t *testing.T) {
	var ran []string
	stages := []Stage{counting("install", &ran, nil, CapDocker)}
	if _, err := NewPlan(Case{From: fromHead}, stages, ModeFast, nil); err == nil {
		t.Fatal("fast mode accepted a plan with no stage to run")
	}
}
