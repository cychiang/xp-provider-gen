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
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
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
		flavor: core.FlavorNative, group: createAPIGroup, kind: kindBucket,
		toolOwned:    "internal/controller/bucket/wiring.go",
		userOwned:    "internal/controller/bucket/external.go",
		registration: "apis/register.go",
	}
	createAPIUpjet = createAPIFlavor{
		flavor: core.FlavorUpjet, group: updateTestGroup, kind: testKindSecret,
		toolOwned:    createAPIResFile,
		userOwned:    "config/secret/config.go",
		registration: createAPIResFile,
	}
)

// createAPIRun runs create api for fl's kind against dst, as runCreateAPI does
// up to the PROJECT save and the automation pipeline, and returns what it printed.
func createAPIRun(t *testing.T, fl createAPIFlavor, dst afero.Fs) string {
	t.Helper()
	cfg, res, o := createAPIPrepared(t, fl)
	meta, err := checkCreateAPI(t.Context(), cfg, res, o)
	if err != nil {
		t.Fatalf("checkCreateAPI: %v", err)
	}
	return captureStdout(t, func() {
		if err := writeCreateAPI(dst, cfg, meta, res, o); err != nil {
			t.Fatalf("writeCreateAPI: %v", err)
		}
	})
}

// createAPIPrepared returns the config, resource and options of create api for
// fl's kind, ready for checkCreateAPI. A fresh config each call stands for a
// fresh process reading PROJECT.
func createAPIPrepared(t *testing.T, fl createAPIFlavor) (config.Config, *resource.Resource, createAPIOptions) {
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
	meta := projectMeta{Flavor: fl.flavor}
	o := createAPIOptions{group: fl.group, version: updateTestVersion, kind: fl.kind}
	if fl.flavor == core.FlavorUpjet {
		meta.Upjet = &core.UpjetSettings{
			TerraformProvider:        createAPIUpjetSrc,
			TerraformProviderName:    createAPITFName,
			TerraformProviderVersion: "2.38.0",
			TerraformProviderRepo:    core.DefaultProviderRepo(createAPIUpjetSrc),
			TerraformDocsPath:        core.DefaultTerraformDocsPath,
			TerraformResourcePrefix:  createAPITFName,
		}
		o.terraformResource = createAPIUpjetTF
	}
	if err := cfg.EncodePluginConfig(pluginName, meta); err != nil {
		t.Fatalf("EncodePluginConfig: %v", err)
	}
	res, err := newResource(cfg, o)
	if err != nil {
		t.Fatalf("newResource: %v", err)
	}
	return cfg, res, o
}

// createAPICheck runs checkCreateAPI on fl's kind and returns its error.
func createAPICheck(t *testing.T, fl createAPIFlavor) error {
	t.Helper()
	cfg, res, o := createAPIPrepared(t, fl)
	_, err := checkCreateAPI(t.Context(), cfg, res, o)
	return err
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

// TestCreateAPIRefusesRepoWithoutCommit pins that create api stops before it
// writes anything when the repository has no commit, which is how a failed init
// leaves it, and that the refusal says init did not finish. A repository with a
// commit and a directory that is no repository both go on.
func TestCreateAPIRefusesRepoWithoutCommit(t *testing.T) {
	t.Run("no commit", func(t *testing.T) {
		dir := t.TempDir()
		gitInDir(t, dir, "init", "--quiet")
		t.Chdir(dir)

		err := createAPICheck(t, createAPINative)
		if err == nil || !strings.Contains(err.Error(), "init did not finish") {
			t.Fatalf("checkCreateAPI = %v, want a refusal saying init did not finish", err)
		}
	})
	t.Run("with a commit", func(t *testing.T) {
		dir := t.TempDir()
		gitInDir(t, dir, "init", "--quiet")
		gitInDir(t, dir, "-c", "user.name=t", "-c", "user.email=t@example.com",
			"commit", "--quiet", "--allow-empty", "-m", "scaffold")
		t.Chdir(dir)

		if err := createAPICheck(t, createAPINative); err != nil {
			t.Errorf("checkCreateAPI = %v, want nil", err)
		}
	})
	t.Run("not a repository", func(t *testing.T) {
		t.Chdir(t.TempDir())

		if err := createAPICheck(t, createAPINative); err != nil {
			t.Errorf("checkCreateAPI = %v, want nil", err)
		}
	})
}

func gitInDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
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

// TestCreateAPIHasNoForceFlag pins that --force is gone: create api always
// refreshes tool-owned files, so the flag is rejected like any unknown one.
func TestCreateAPIHasNoForceFlag(t *testing.T) {
	cmd := NewCreateAPICommand()
	if f := cmd.Flags().Lookup("force"); f != nil {
		t.Fatalf("--force is still defined: %+v", f)
	}
	if err := cmd.ParseFlags([]string{"--force"}); err == nil || !strings.Contains(err.Error(), "unknown flag: --force") {
		t.Errorf("ParseFlags(--force) error = %v, want unknown flag: --force", err)
	}
}

// TestNewResource pins how create api turns its flags into a resource: the
// value that starts with "-" is blamed on its own flag, however many are bad;
// GVK values are trimmed, an empty one is rejected, and the resource carries
// the project's domain and an API of the shape init projects record.
func TestNewResource(t *testing.T) {
	cfg, _, _ := createAPIPrepared(t, createAPINative)
	padded := createAPIOptions{group: " " + createAPIGroup + " ", version: " " + updateTestVersion + " ", kind: kindBucket}
	tests := []struct {
		name    string
		o       createAPIOptions
		wantErr string
	}{
		{"kind swallowed the next flag", createAPIOptions{group: "x", kind: "--version=v1"}, "kind flag present but empty"},
		{"group swallowed the next flag", createAPIOptions{group: "--kind=K", version: "v1", kind: kindBucket}, "group flag present but empty"},
		{"version missing", createAPIOptions{group: createAPIGroup, kind: kindBucket}, "version"},
		{"kind is not PascalCase", createAPIOptions{group: createAPIGroup, version: "v1", kind: "bucket"}, "invalid Kind"},
		{"valid, group and version padded with spaces", padded, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := newResource(cfg, tt.o)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("newResource = %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("newResource: %v", err)
			}
			if res.Group != createAPIGroup || res.Version != updateTestVersion || res.Kind != kindBucket || res.Domain != testDomain {
				t.Errorf("resource = %+v, want %s/%s %s in %s", res.GVK, createAPIGroup, updateTestVersion, kindBucket, testDomain)
			}
			if res.Plural != "buckets" || !res.Controller || res.API == nil || !res.API.Namespaced || res.API.CRDVersion != "v1" {
				t.Errorf("resource = %+v, want plural buckets, a controller and a namespaced v1 API", res)
			}
		})
	}
}

// TestCreateAPIKeepsAuthorOwnedFile pins that a tool-owned file whose header
// the author removed stays theirs; --force used to overwrite it in v0.1.0.
func TestCreateAPIKeepsAuthorOwnedFile(t *testing.T) {
	dst := scaffoldedKind(t, createAPINative)
	own := "package bucket // " + createAPIMarker + "\n"
	writeFile(t, dst, createAPINative.toolOwned, own)

	out := createAPIRun(t, createAPINative, dst)

	if got := readFile(t, dst, createAPINative.toolOwned); got != own {
		t.Errorf("%s was overwritten", createAPINative.toolOwned)
	}
	if !strings.Contains(out, "\n  "+createAPINative.toolOwned+"\n") {
		t.Errorf("stdout does not list %s as kept:\n%s", createAPINative.toolOwned, out)
	}
}
