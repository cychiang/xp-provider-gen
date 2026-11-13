package v2

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// TestProjectBytes pins the PROJECT file init followed by create api writes,
// for both flavors, byte for byte against files taken from the last release
// built on the Kubebuilder CLI. The version line reads dev because that is
// what version.Get reports under go test.
func TestProjectBytes(t *testing.T) {
	tests := []struct {
		flavor core.Flavor
		init   initOptions
		api    createAPIOptions
	}{
		{
			core.FlavorNative,
			initOptions{domain: testDomain, repo: testProviderRepo, tfDocsPath: core.DefaultTerraformDocsPath},
			createAPIOptions{group: createAPIGroup, version: updateTestVersion, kind: kindBucket},
		},
		{
			core.FlavorUpjet,
			initOptions{
				domain: testDomain, repo: testProviderRepo, upjet: true,
				tfProvider: "hashicorp/kubernetes", tfProviderVersion: "2.38.0",
				tfDocsPath: core.DefaultTerraformDocsPath,
			},
			createAPIOptions{group: "core", version: "v1alpha1", kind: testKindSecret, terraformResource: "kubernetes_secret"},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.flavor), func(t *testing.T) {
			want, err := os.ReadFile("testdata/project/" + string(tt.flavor) + ".golden")
			if err != nil {
				t.Fatal(err)
			}
			t.Chdir(t.TempDir())
			dst := afero.NewMemMapFs()

			st, err := newProjectStore(dst)
			if err != nil {
				t.Fatalf("newProjectStore: %v", err)
			}
			captureStdout(t, func() {
				if _, err := writeInit(dst, st.Config(), tt.init); err != nil {
					t.Fatalf("writeInit: %v", err)
				}
			})
			saveProject(t, st)

			st, err = loadProjectStore(dst)
			if err != nil {
				t.Fatalf("loadProjectStore: %v", err)
			}
			res, err := newResource(st.Config(), tt.api)
			if err != nil {
				t.Fatalf("newResource: %v", err)
			}
			meta, err := checkCreateAPI(t.Context(), st.Config(), res, tt.api)
			if err != nil {
				t.Fatalf("checkCreateAPI: %v", err)
			}
			captureStdout(t, func() {
				if err := writeCreateAPI(dst, st.Config(), meta, res, tt.api); err != nil {
					t.Fatalf("writeCreateAPI: %v", err)
				}
			})
			saveProject(t, st)

			if got := readFile(t, dst, "PROJECT"); got != string(want) {
				t.Errorf("PROJECT differs from testdata/project/%s.golden:\n%s", tt.flavor, got)
			}
		})
	}
}

func saveProject(t *testing.T, st store.Store) {
	t.Helper()
	if err := st.Save(); err != nil {
		t.Fatalf("saving PROJECT: %v", err)
	}
}

// writeProject writes a PROJECT file with the given body into a new filesystem.
func writeProject(t *testing.T, body string) afero.Fs {
	t.Helper()
	dst := afero.NewMemMapFs()
	writeFile(t, dst, "PROJECT", body)
	return dst
}

const (
	projectHead = "domain: example.com\nrepo: github.com/example/provider-test\nversion: \"3\"\n"
	nativeBlock = "plugins:\n  crossplane.go.kubebuilder.io:\n    flavor: native\n"
)

// TestRequireOwnLayout pins that PROJECT is used only when its layout is
// exactly this generator's key: nothing else is a project this tool scaffolded,
// whether the layout is missing, partial, foreign or longer. Each load goes
// through loadProjectStore, the gate update, create api and create-test share.
func TestRequireOwnLayout(t *testing.T) {
	tests := []struct {
		name    string
		layout  string
		wantErr string
	}{
		{"own key", "layout:\n- crossplane.go.kubebuilder.io/v2\n", ""},
		{"no layout", "", "layout []"},
		{"partial key", "layout:\n- crossplane.go.kubebuilder.io\n", "layout [crossplane.go.kubebuilder.io]"},
		{"shorter key", "layout:\n- crossplane\n", "layout [crossplane]"},
		{"foreign key", "layout:\n- go.kubebuilder.io/v4\n", "layout [go.kubebuilder.io/v4]"},
		{"own key plus another", "layout:\n- crossplane.go.kubebuilder.io/v2\n- helm.kubebuilder.io/v1-alpha\n", "layout [crossplane.go.kubebuilder.io/v2 helm.kubebuilder.io/v1-alpha]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadProjectStore(writeProject(t, projectHead+tt.layout+nativeBlock))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("loadProjectStore = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("loadProjectStore = %v, want an error containing %q", err, tt.wantErr)
			}
		})
	}
}

// TestNewProjectStore pins what init does with a PROJECT already on disk: an
// intact one of this tool is "already initialized", and anything else says
// what is wrong with it rather than claiming that.
func TestNewProjectStore(t *testing.T) {
	const ownLayout = "layout:\n- crossplane.go.kubebuilder.io/v2\n"
	tests := []struct {
		name    string
		project *string
		wantErr string
	}{
		{"no PROJECT", nil, ""},
		{"own PROJECT", ptr(projectHead + ownLayout + nativeBlock), "already initialized"},
		{"corrupt yaml", ptr("domain: [unclosed\n"), "cannot be loaded"},
		{"foreign layout", ptr(projectHead + "layout:\n- go.kubebuilder.io/v4\n"), "layout [go.kubebuilder.io/v4]"},
		{"no layout", ptr(projectHead), "layout []"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := afero.NewMemMapFs()
			if tt.project != nil {
				writeFile(t, dst, "PROJECT", *tt.project)
			}
			st, err := newProjectStore(dst)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("newProjectStore = %v, want nil", err)
				}
				if got := st.Config().GetPluginChain(); len(got) != 1 || got[0] != pluginKey {
					t.Errorf("layout = %v, want [%s]", got, pluginKey)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("newProjectStore = %v, want an error containing %q", err, tt.wantErr)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
