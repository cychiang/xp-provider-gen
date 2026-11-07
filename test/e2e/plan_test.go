//go:build !windows

package main

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
)

const newestTag = "v0.3.0"

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
			if v, _ := Execute(plan, &Env{}, &out); v != VerdictPartial {
				t.Fatalf("verdict %s, want PARTIAL", v)
			}
			if strings.Join(ran, ",") != cmdInit {
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
	if v, _ := Execute(plan, &Env{}, &out); v != VerdictFailed {
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
	if v, _ := Execute(plan, &Env{}, &out); v != VerdictComplete {
		t.Fatalf("verdict %s, want COMPLETE:\n%s", v, out.String())
	}
}

func TestReleaseOnlyStagesRunForReleaseCasesOnly(t *testing.T) {
	var ran []string
	release := counting("update-to-head", &ran, nil)
	release.When = Case.Release
	stages := []Stage{counting("init", &ran, nil), release}
	for from, want := range map[string]string{fromHead: "init", newestTag: "init,update-to-head"} {
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

func TestStockE2ERunsForHeadOnly(t *testing.T) {
	for _, flavor := range []string{nativeName, upjetName} {
		f, err := flavorFor(flavor)
		if err != nil {
			t.Fatal(err)
		}
		for from, want := range map[string]bool{fromHead: true, newestTag: false} {
			plan, err := NewPlan(Case{From: from}, f.Stages(), ModeFull, map[Capability]bool{CapDocker: true})
			if err != nil {
				t.Fatal(err)
			}
			if ran := slices.Contains(stageNames(plan.Run), "stock-e2e"); ran != want {
				t.Errorf("%s from %s: stock-e2e ran %t, want %t", flavor, from, ran, want)
			}
		}
	}
}

func stageNames(stages []Stage) []string {
	var names []string
	for _, s := range stages {
		names = append(names, s.Name)
	}
	return names
}

func TestNativeStagesAreDeclaredInPlanOrder(t *testing.T) {
	want := "init create-api generate user-logic update-to-head refuse-old-generator update-noop orphan-removal create-api-refresh adopt create-test " +
		"stock-e2e build-xpkg install-in-kind provider-config mr-ready-synced mr-update drift-corrected " +
		"mr-delete external-gone"
	if got := strings.Join(stageNames(nativeStages()), " "); got != want {
		t.Errorf("stages\n got %v\nwant %s", got, want)
	}
}

func TestUpjetDeclaresTheNativeStagesAndOnlyItsOwnAdditions(t *testing.T) {
	shared := slices.DeleteFunc(stageNames(upjetStages()), func(name string) bool { return slices.Contains(upjetOnly, name) })
	if want := stageNames(nativeStages()); !slices.Equal(shared, want) {
		t.Errorf("upjet's stages less %v\n got %v\nwant native's %v", upjetOnly, shared, want)
	}
	for _, name := range upjetOnly {
		if !slices.Contains(stageNames(upjetStages()), name) {
			t.Errorf("upjetOnly names %s, which upjet does not declare", name)
		}
	}
}

func TestFastModeRunsExactlyTheDockerFreeStages(t *testing.T) {
	for flavor, want := range map[string]string{
		nativeName: "init create-api generate user-logic update-noop orphan-removal create-api-refresh adopt create-test",
		upjetName: "init create-api generate user-logic update-noop orphan-removal create-api-refresh adopt create-test " +
			"tf-bump provider-starts",
	} {
		t.Run(flavor, func(t *testing.T) {
			f, err := flavorFor(flavor)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := NewPlan(Case{From: fromHead}, f.Stages(), ModeFast, map[Capability]bool{})
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Join(stageNames(plan.Run), " "); got != want {
				t.Errorf("fast mode runs\n got %v\nwant %s", got, want)
			}
		})
	}
}

func TestEveryFlavorsStagesNeedDockerExactlyWhereTheClusterIsUsed(t *testing.T) {
	for _, flavor := range []string{nativeName, upjetName} {
		f, err := flavorFor(flavor)
		if err != nil {
			t.Fatal(err)
		}
		var docker []string
		for _, s := range f.Stages() {
			if len(s.Needs) > 0 {
				docker = append(docker, s.Name)
			}
		}
		want := "stock-e2e build-xpkg install-in-kind provider-config mr-ready-synced mr-update drift-corrected mr-delete external-gone"
		if got := strings.Join(docker, " "); got != want {
			t.Errorf("%s stages needing docker\n got %v\nwant %s", flavor, got, want)
		}
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

func TestNewCaseGivesAToolchainOnlyToTheReleasesListed(t *testing.T) {
	for _, c := range []struct{ from, want string }{
		{fromHead, ""},
		{newestTag, "go1.26.8"},
		{"v9.9.9", ""},
	} {
		if got := NewCase(nativeName, c.from).Toolchain; got != c.want {
			t.Errorf("%s runs under toolchain %q, want %q", c.from, got, c.want)
		}
	}
}
