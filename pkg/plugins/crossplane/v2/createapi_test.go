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
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

const (
	createAPIMarker   = "STALE-MARKER"
	createAPIGroup    = "storage"
	createAPIUpjetTF  = "kubernetes_secret"
	createAPITFName   = "kubernetes"
	createAPIResFile  = "config/zz_resources.go"
	createAPIUpjetSrc = "hashicorp/" + createAPITFName
)

// createAPIFlavor is one flavor's files the create api tests look at, for the
// kind createAPIRun scaffolds.
type createAPIFlavor struct {
	flavor       core.Flavor
	group, kind  string
	toolOwned    string // a per-kind or generator file carrying the generated header
	userOwned    string // a seed the author owns
	registration string // a generator-owned registration file
}

var (
	createAPINative = createAPIFlavor{
		flavor: core.FlavorNative, group: createAPIGroup, kind: "Bucket",
		toolOwned:    "internal/controller/bucket/wiring.go",
		userOwned:    "internal/controller/bucket/external.go",
		registration: "apis/register.go",
	}
	createAPIUpjet = createAPIFlavor{
		flavor: core.FlavorUpjet, group: updateTestGroup, kind: "Secret",
		toolOwned:    createAPIResFile,
		userOwned:    "config/secret/config.go",
		registration: createAPIResFile,
	}
)

// createAPIRun runs create api for fl's kind against dst, as the CLI does up to
// PostScaffold, and returns what it printed. A fresh config each call stands
// for a fresh process reading PROJECT.
func createAPIRun(t *testing.T, fl createAPIFlavor, dst afero.Fs, args ...string) string {
	t.Helper()
	cfg, err := config.New(cfgv3.Version)
	if err != nil {
		t.Fatalf("config.New: %v", err)
	}
	if err := cfg.SetDomain(testDomain); err != nil {
		t.Fatalf("SetDomain: %v", err)
	}
	if err := cfg.SetRepository(testProviderRepo); err != nil {
		t.Fatalf("SetRepository: %v", err)
	}
	if err := saveProjectMeta(cfg, func(m *projectMeta) {
		m.Flavor = fl.flavor
		if fl.flavor == core.FlavorUpjet {
			m.Upjet = &core.UpjetSettings{
				TerraformProvider:        createAPIUpjetSrc,
				TerraformProviderName:    createAPITFName,
				TerraformProviderVersion: "2.38.0",
				TerraformProviderRepo:    core.DefaultProviderRepo(createAPIUpjetSrc),
				TerraformDocsPath:        core.DefaultTerraformDocsPath,
				TerraformResourcePrefix:  createAPITFName,
			}
		}
	}); err != nil {
		t.Fatalf("saveProjectMeta: %v", err)
	}

	p := &createAPISubcommand{}
	fs := pflag.NewFlagSet("create api", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	p.BindFlags(fs)
	if fl.flavor == core.FlavorUpjet {
		args = append(args, "--terraform-resource="+createAPIUpjetTF)
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parsing flags: %v", err)
	}
	if err := p.InjectConfig(cfg); err != nil {
		t.Fatalf("InjectConfig: %v", err)
	}
	res := &resource.Resource{GVK: resource.GVK{Group: fl.group, Version: "v1alpha1", Kind: fl.kind}}
	if err := p.InjectResource(res); err != nil {
		t.Fatalf("InjectResource: %v", err)
	}
	machineryFS := machinery.Filesystem{FS: dst}
	if err := p.PreScaffold(machineryFS); err != nil {
		t.Fatalf("PreScaffold: %v", err)
	}
	return captureStdout(t, func() {
		if err := p.Scaffold(machineryFS); err != nil {
			t.Fatalf("Scaffold: %v", err)
		}
	})
}

func readFile(t *testing.T, fs afero.Fs, path string) string {
	t.Helper()
	b, err := afero.ReadFile(fs, path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

func writeFile(t *testing.T, fs afero.Fs, path, body string) {
	t.Helper()
	if err := afero.WriteFile(fs, path, []byte(body), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// scaffoldedKind returns a filesystem create api has already run on once.
func scaffoldedKind(t *testing.T, fl createAPIFlavor) afero.Fs {
	t.Helper()
	dst := afero.NewMemMapFs()
	createAPIRun(t, fl, dst)
	for _, path := range []string{fl.toolOwned, fl.userOwned} {
		if ok, _ := afero.Exists(dst, path); !ok {
			t.Fatalf("first create api did not write %s", path)
		}
	}
	return dst
}

// TestCreateAPIRefreshesStaleToolOwned pins that re-running create api for an
// existing kind refreshes its tool-owned files without any flag and leaves the
// author's files alone.
func TestCreateAPIRefreshesStaleToolOwned(t *testing.T) {
	for _, fl := range []createAPIFlavor{createAPINative, createAPIUpjet} {
		t.Run(string(fl.flavor), func(t *testing.T) {
			dst := scaffoldedKind(t, fl)
			writeFile(t, dst, fl.toolOwned, core.GeneratedHeader+"\n// "+createAPIMarker+"\n")
			userBody := readFile(t, dst, fl.userOwned) + "\n// " + createAPIMarker + "\n"
			writeFile(t, dst, fl.userOwned, userBody)

			createAPIRun(t, fl, dst)

			if got := readFile(t, dst, fl.toolOwned); strings.Contains(got, createAPIMarker) {
				t.Errorf("%s still holds the stale marker; create api did not refresh it", fl.toolOwned)
			}
			if got := readFile(t, dst, fl.userOwned); got != userBody {
				t.Errorf("%s changed; user-owned files must be kept", fl.userOwned)
			}
		})
	}
}

// TestCreateAPIKeepsHeaderlessRegistration pins that a registration file whose
// header the author removed is kept, listed, and called out, since the new
// kind is not wired into it. The Kept list also holds the kind's other
// user-owned files, which already exist on a re-run, so only its heading and
// this path are pinned.
func TestCreateAPIKeepsHeaderlessRegistration(t *testing.T) {
	for _, fl := range []createAPIFlavor{createAPINative, createAPIUpjet} {
		t.Run(string(fl.flavor), func(t *testing.T) {
			dst := scaffoldedKind(t, fl)
			own := "package x // " + createAPIMarker + "\n"
			writeFile(t, dst, fl.registration, own)

			out := createAPIRun(t, fl, dst)

			if got := readFile(t, dst, fl.registration); got != own {
				t.Errorf("%s was overwritten; it has no generated header", fl.registration)
			}
			for _, want := range []string{
				"existing file(s) without the generated header:\n",
				"\n  " + fl.registration + "\n",
				"  warning: " + fl.registration + " has no generated header, so " + fl.kind +
					" was not registered in it; add it yourself\n",
			} {
				if !strings.Contains(out, want) {
					t.Errorf("stdout lacks %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestCreateAPIForceIsDeprecatedNoOp pins that --force is accepted, marked
// deprecated, and changes nothing: the same stale tree ends up the same with
// and without it.
func TestCreateAPIForceIsDeprecatedNoOp(t *testing.T) {
	fs := pflag.NewFlagSet("create api", pflag.ContinueOnError)
	(&createAPISubcommand{}).BindFlags(fs)
	if f := fs.Lookup("force"); f == nil || f.Deprecated == "" {
		t.Fatalf("--force is not marked deprecated: %+v", f)
	}

	run := func(args ...string) []string {
		dst := scaffoldedKind(t, createAPINative)
		writeFile(t, dst, createAPINative.toolOwned, core.GeneratedHeader+"\n// "+createAPIMarker+"\n")
		writeFile(t, dst, createAPINative.userOwned, "package x // "+createAPIMarker+"\n")
		out := createAPIRun(t, createAPINative, dst, args...)
		return []string{
			readFile(t, dst, createAPINative.toolOwned), readFile(t, dst, createAPINative.userOwned), out,
		}
	}
	if got, forced := run(), run("--force"); !slices.Equal(got, forced) {
		t.Errorf("--force changed the outcome:\nwithout: %q\nwith:    %q", got, forced)
	}
}

// TestCreateAPIForceKeepsAuthorOwnedFile pins that a tool-owned file whose
// header the author removed stays theirs even with --force, which used to
// overwrite it.
func TestCreateAPIForceKeepsAuthorOwnedFile(t *testing.T) {
	for _, args := range [][]string{nil, {"--force"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			dst := scaffoldedKind(t, createAPINative)
			own := "package bucket // " + createAPIMarker + "\n"
			writeFile(t, dst, createAPINative.toolOwned, own)

			out := createAPIRun(t, createAPINative, dst, args...)

			if got := readFile(t, dst, createAPINative.toolOwned); got != own {
				t.Errorf("%s was overwritten", createAPINative.toolOwned)
			}
			if !strings.Contains(out, "\n  "+createAPINative.toolOwned+"\n") {
				t.Errorf("stdout does not list %s as kept:\n%s", createAPINative.toolOwned, out)
			}
		})
	}
}
