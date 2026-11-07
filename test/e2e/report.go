//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// reportSchema is bumped when a field is renamed or removed.
const reportSchema = 1

type Status string

const (
	StatusPass   Status = "pass"
	StatusFail   Status = "fail"
	StatusNotRun Status = "not-run"
)

// StageResult is what one declared stage did. A stage that did not run has no
// seconds.
type StageResult struct {
	Name    string  `json:"name"`
	Status  Status  `json:"status"`
	Seconds float64 `json:"seconds"`
}

// Upgrade is what the update-to-head stage measured, each count taken before
// the stage fails on it. A count is null when its check errored before it
// could count, which is not the same as a measured zero. A run that never
// reached the stage has no Upgrade at all.
type Upgrade struct {
	// ToolOwnedCompared is the tool-owned files of a fresh head scaffold, and
	// ToolOwnedDiffering how many of those the updated tree lacks or holds with
	// other bytes, plus the tool-owned files it holds that a fresh one does not.
	ToolOwnedCompared  *int `json:"tool_owned_compared"`
	ToolOwnedDiffering *int `json:"tool_owned_differing"`
	// UserOwnedChanged is the author's files update removed, changed or added.
	UserOwnedChanged *int `json:"user_owned_changed"`
	// GeneratedChanged is the files the provider's own generate changed after
	// the update: output that was stale.
	GeneratedChanged *int `json:"generated_changed"`
}

// counted is a measured count.
func counted(n int) *int { return &n }

func countOrNA(n *int) string {
	if n == nil {
		return "n/a"
	}
	return strconv.Itoa(*n)
}

// Summary is the one log line of an upgrade run's counts.
func (u Upgrade) Summary() string {
	return fmt.Sprintf("UPGRADE tool_owned %s compared %s differing, user_owned %s changed, generated %s changed",
		countOrNA(u.ToolOwnedCompared), countOrNA(u.ToolOwnedDiffering),
		countOrNA(u.UserOwnedChanged), countOrNA(u.GeneratedChanged))
}

// Report is the machine-readable result of one run.
type Report struct {
	Schema  int           `json:"schema"`
	Flavor  string        `json:"flavor"`
	From    string        `json:"from"` // the release tag, empty for head
	Head    string        `json:"head"` // the binary under test's version
	Verdict Verdict       `json:"verdict"`
	Stages  []StageResult `json:"stages"`
	Upgrade *Upgrade      `json:"upgrade,omitempty"`
}

func (r Report) write(path string) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("writing the report: %w", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("writing the report: %w", err)
	}
	return nil
}

func tenths(d time.Duration) float64 { return math.Round(d.Seconds()*10) / 10 }
