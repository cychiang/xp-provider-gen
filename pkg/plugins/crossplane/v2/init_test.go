/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
)

// captureStdout runs fn with os.Stdout redirected, returning what it wrote.
// Package v2 and validation print straight to os.Stdout rather than an
// injected writer, so this is the only way to assert on that output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return buf.String()
}

// TestInitSubcommand_InjectConfig_RequiresDomain pins that init rejects an
// empty --domain: it would produce a provider whose ProviderConfig group is
// "" and whose CRDs get deleted by apis/generate.go's "no empty group" step.
func TestInitSubcommand_InjectConfig_RequiresDomain(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		repo    string
		wantErr string
	}{
		{
			name:    "empty domain is rejected",
			domain:  "",
			repo:    testProviderRepo,
			wantErr: "domain",
		},
		{
			name:   "valid domain is accepted",
			domain: testDomain,
			repo:   testProviderRepo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.New(cfgv3.Version)
			if err != nil {
				t.Fatalf("config.New: %v", err)
			}

			p := &initSubcommand{domain: tt.domain, repo: tt.repo}
			err = p.InjectConfig(cfg)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("InjectConfig() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("InjectConfig() unexpected error: %v", err)
			}
			if got := cfg.GetDomain(); got != tt.domain {
				t.Errorf("GetDomain() = %q, want %q", got, tt.domain)
			}
		})
	}
}

// TestInitSubcommand_InjectConfig_WarnsOnUnconventionalRepoName pins that
// init prints the naming-convention warning from its own success path
// (the validator itself must never print, see
// TestValidator_ValidateRepository_NeverPrints) — exactly once.
func TestInitSubcommand_InjectConfig_WarnsOnUnconventionalRepoName(t *testing.T) {
	cfg, err := config.New(cfgv3.Version)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}
	p := &initSubcommand{domain: testDomain, repo: "github.com/example/not-conventional"}

	out := captureStdout(t, func() {
		if err := p.InjectConfig(cfg); err != nil {
			t.Fatalf("InjectConfig() unexpected error: %v", err)
		}
	})

	if got := strings.Count(out, "doesn't follow Crossplane convention"); got != 1 {
		t.Errorf("naming warning printed %d time(s), want exactly 1:\n%s", got, out)
	}
}

// Fixture values for TestInitSubcommand_GitIdentity.
const (
	testFlagGitName    = "Flag Name"
	testFlagGitEmail   = "flag@example.com"
	testSystemGitName  = "System Name"
	testSystemGitEmail = "system@example.com"
)

// TestInitSubcommand_GitIdentity pins gitIdentity's three-tier priority: CLI
// flags, then system git config (via query), then the project defaults
// already in pluginConfig.
func TestInitSubcommand_GitIdentity(t *testing.T) {
	tests := []struct {
		name      string
		gitName   string
		gitEmail  string
		sysName   string
		sysEmail  string
		wantName  string
		wantEmail string
	}{
		{
			name:      "both flags given wins over system config",
			gitName:   testFlagGitName,
			gitEmail:  testFlagGitEmail,
			sysName:   testSystemGitName,
			sysEmail:  testSystemGitEmail,
			wantName:  testFlagGitName,
			wantEmail: testFlagGitEmail,
		},
		{
			name:      "only name flag given, email falls back to system config",
			gitName:   testFlagGitName,
			sysName:   testSystemGitName,
			sysEmail:  testSystemGitEmail,
			wantName:  testFlagGitName,
			wantEmail: testSystemGitEmail,
		},
		{
			name:      "no flags given, system config has values",
			sysName:   testSystemGitName,
			sysEmail:  testSystemGitEmail,
			wantName:  testSystemGitName,
			wantEmail: testSystemGitEmail,
		},
		{
			name:      "no flags given, system config empty falls back to project defaults",
			wantName:  "Crossplane Provider Generator",
			wantEmail: "noreply@crossplane.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &initSubcommand{
				gitName:      tt.gitName,
				gitEmail:     tt.gitEmail,
				pluginConfig: NewPluginConfig(),
			}
			query := func() (string, string) { return tt.sysName, tt.sysEmail }

			gotName, gotEmail := p.gitIdentity(query)

			if gotName != tt.wantName {
				t.Errorf("gitIdentity() name = %q, want %q", gotName, tt.wantName)
			}
			if gotEmail != tt.wantEmail {
				t.Errorf("gitIdentity() email = %q, want %q", gotEmail, tt.wantEmail)
			}
		})
	}
}
