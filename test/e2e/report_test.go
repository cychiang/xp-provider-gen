//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestAFailedRunStillWritesItsReportWithEveryStagesStatus(t *testing.T) {
	var ran []string
	stages := []Stage{
		counting("one", &ran, nil),
		counting("two", &ran, errors.New("boom")),
		counting("three", &ran, nil),
		counting("cluster", &ran, nil, CapDocker),
	}
	plan, err := NewPlan(Case{From: newestTag}, stages, ModeFull, map[Capability]bool{CapDocker: true})
	if err != nil {
		t.Fatal(err)
	}
	env := &Env{Upgrade: &Upgrade{ToolOwnedCompared: 9, ToolOwnedDiffering: 2, UserOwnedChanged: 1, GeneratedChanged: 3}}
	verdict, results := Execute(plan, env, io.Discard)
	path := filepath.Join(t.TempDir(), "nested", "report.json")
	rep := Report{
		Schema: reportSchema, Flavor: nativeName, From: newestTag, Head: "v1.0.0",
		Verdict: verdict, Stages: results, Upgrade: env.Upgrade,
	}
	if err := rep.write(path); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path) // #nosec G304 -- a file this test wrote
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Schema  int    `json:"schema"`
		Flavor  string `json:"flavor"`
		From    string `json:"from"`
		Head    string `json:"head"`
		Verdict string `json:"verdict"`
		Stages  []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"stages"`
		Upgrade map[string]int `json:"upgrade"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != reportSchema || got.Flavor != nativeName || got.From != newestTag || got.Head != "v1.0.0" || got.Verdict != "FAILED" {
		t.Errorf("report header is %+v", got)
	}
	want := map[string]string{"one": "pass", "two": "fail", "three": "not-run", "cluster": "not-run"}
	if len(got.Stages) != len(want) {
		t.Fatalf("report lists %d stages, want %d: %s", len(got.Stages), len(want), b)
	}
	for _, s := range got.Stages {
		if want[s.Name] != s.Status {
			t.Errorf("stage %s is %q, want %q", s.Name, s.Status, want[s.Name])
		}
	}
	wantCounts := map[string]int{
		"tool_owned_compared": 9, "tool_owned_differing": 2, "user_owned_changed": 1, "generated_changed": 3,
	}
	for k, v := range wantCounts {
		if got.Upgrade[k] != v {
			t.Errorf("upgrade %s is %d, want %d", k, got.Upgrade[k], v)
		}
	}
}

func TestAHeadRunsReportHasNoUpgradeAndNoFrom(t *testing.T) {
	b, err := json.Marshal(Report{Schema: reportSchema, Flavor: upjetName, Verdict: VerdictComplete})
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{`"upgrade"`} {
		if bytes.Contains(b, []byte(absent)) {
			t.Errorf("report %s holds %s", b, absent)
		}
	}
	if !bytes.Contains(b, []byte(`"from":""`)) {
		t.Errorf("report %s does not give an empty from", b)
	}
}

func TestAssertGenerateChangesNothingCountsEveryFileGenerateRewrites(t *testing.T) {
	const makefile = "generate:\n\t@echo fresh > a.txt\n\t@echo fresh > b.txt\n"
	for _, c := range []struct {
		name        string
		files       map[string]string
		wantChanged int
	}{
		{"output is current", map[string]string{"a.txt": "fresh\n", "b.txt": "fresh\n"}, 0},
		{"one file is stale", map[string]string{"a.txt": "stale\n", "b.txt": "fresh\n"}, 1},
		{"two stale and one missing", map[string]string{"a.txt": "stale\n"}, 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			e := &Env{Ctx: t.Context(), Dir: dir, Work: dir, log: io.Discard}
			files := map[string]string{"Makefile": makefile}
			for k, v := range c.files {
				files[k] = v
			}
			for rel, body := range files {
				if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := e.git("init", "-q"); err != nil {
				t.Fatal(err)
			}
			changed, err := e.assertGenerateChangesNothing()
			if changed != c.wantChanged || (err != nil) != (c.wantChanged > 0) {
				t.Fatalf("changed %d, error %v, want %d changed", changed, err, c.wantChanged)
			}
		})
	}
}
