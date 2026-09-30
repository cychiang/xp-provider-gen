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

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/version"
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

// TestInitStampsGeneratorVersion pins that init stamps the running
// generator's version into PROJECT's plugin block on the way in, the same
// field update and adopt later read as checkNotDowngrade's "last" — without
// this, every project would be unprotected against a downgrade until its
// first update. Scaffold only renders into the given machinery.Filesystem
// (here an in-memory one); the post-init automation pipeline that touches
// real disk and runs external commands lives in PostScaffold, which this
// test never calls.
func TestInitStampsGeneratorVersion(t *testing.T) {
	cfg, err := config.New(cfgv3.Version)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}
	p := &initSubcommand{domain: testDomain, repo: testProviderRepo}
	if err := p.InjectConfig(cfg); err != nil {
		t.Fatalf("InjectConfig: %v", err)
	}
	if err := p.Scaffold(machinery.Filesystem{FS: afero.NewMemMapFs()}); err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	meta, err := loadProjectMeta(p.config)
	if err != nil {
		t.Fatalf("loadProjectMeta: %v", err)
	}
	if want := version.Get().Version; meta.Version != want {
		t.Errorf("PROJECT's stamped version = %q, want %q", meta.Version, want)
	}
}

// TestInitNonEmptyDir pins how init treats files already on disk when there is
// no PROJECT (kubebuilder refuses one that has it): the same ownership rule as
// update. A headered file is refreshed from the render, whichever kind of
// output it is; a headerless one is the user's, kept and listed.
func TestInitNonEmptyDir(t *testing.T) {
	const marker = "STALE-MARKER"
	headered := core.GeneratedHeader + "\n// " + marker + "\n"
	const (
		generatorOutput = "apis/register.go"
		templateOutput  = "hack/xp-provider-gen.mk"
	)

	tests := []struct {
		name, path, existing string
		wantKept             bool
	}{
		{"headered generator output is refreshed", generatorOutput, headered, false},
		{"headerless generator output is kept and listed", generatorOutput, "package apis // " + marker + "\n", true},
		{"headered template output is refreshed", templateOutput, headered, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.New(cfgv3.Version)
			if err != nil {
				t.Fatalf("config.New: %v", err)
			}
			p := &initSubcommand{domain: testDomain, repo: testProviderRepo}
			if err := p.InjectConfig(cfg); err != nil {
				t.Fatalf("InjectConfig: %v", err)
			}
			dst := afero.NewMemMapFs()
			if err := afero.WriteFile(dst, tt.path, []byte(tt.existing), 0o644); err != nil {
				t.Fatalf("seeding %s: %v", tt.path, err)
			}

			out := captureStdout(t, func() {
				if err := p.Scaffold(machinery.Filesystem{FS: dst}); err != nil {
					t.Fatalf("Scaffold: %v", err)
				}
			})

			got, err := afero.ReadFile(dst, tt.path)
			if err != nil {
				t.Fatalf("reading %s: %v", tt.path, err)
			}
			if kept := strings.Contains(string(got), marker); kept != tt.wantKept {
				t.Errorf("%s keeps its old content = %v, want %v", tt.path, kept, tt.wantKept)
			}
			if listed := strings.Contains(out, "  "+tt.path); listed != tt.wantKept {
				t.Errorf("%s listed under Kept = %v, want %v:\n%s", tt.path, listed, tt.wantKept, out)
			}
			const keptHeader = "Kept 1 existing file(s) without the generated header:"
			if hasHeader := strings.Contains(out, keptHeader); hasHeader != tt.wantKept {
				t.Errorf("stdout has %q = %v, want %v:\n%s", keptHeader, hasHeader, tt.wantKept, out)
			}
			if !tt.wantKept && strings.Contains(out, "Kept ") {
				t.Errorf("stdout mentions Kept although nothing was kept:\n%s", out)
			}
		})
	}
}

// TestInitEmptyDir pins what a fresh init writes through core.Apply: a
// headerless (user-owned) file is seeded rather than withheld, and a script
// lands 0755 at write time, now that no step chmods it afterwards.
func TestInitEmptyDir(t *testing.T) {
	const setupScript = "test/setup.sh" // headerless and executable in both flavors
	tests := []struct {
		name string
		p    *initSubcommand
	}{
		{string(core.FlavorNative), &initSubcommand{domain: testDomain, repo: testProviderRepo}},
		{string(core.FlavorUpjet), &initSubcommand{
			domain: testDomain, repo: testProviderRepo, upjet: true,
			tfProvider: "hashicorp/kubernetes", tfProviderVersion: "2.30.0",
			tfDocsPath: core.DefaultTerraformDocsPath,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.New(cfgv3.Version)
			if err != nil {
				t.Fatalf("config.New: %v", err)
			}
			if err := tt.p.InjectConfig(cfg); err != nil {
				t.Fatalf("InjectConfig: %v", err)
			}
			dst := afero.NewMemMapFs()
			out := captureStdout(t, func() {
				if err := tt.p.Scaffold(machinery.Filesystem{FS: dst}); err != nil {
					t.Fatalf("Scaffold: %v", err)
				}
			})

			body, err := afero.ReadFile(dst, setupScript)
			if err != nil {
				t.Fatalf("reading %s: %v", setupScript, err)
			}
			if core.IsToolOwned(body) {
				t.Errorf("%s carries the generated header, want a user-owned file", setupScript)
			}
			info, err := dst.Stat(setupScript)
			if err != nil {
				t.Fatalf("Stat: %v", err)
			}
			if got := info.Mode().Perm(); got != core.ScriptMode {
				t.Errorf("%s mode = %#o, want %#o", setupScript, got, core.ScriptMode)
			}
			if strings.Contains(out, "Kept ") {
				t.Errorf("fresh init printed a Kept list:\n%s", out)
			}
		})
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
