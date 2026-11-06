package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	domain        = "example.com"
	repo          = "github.com/xpg-e2e/provider-acme"
	provider      = "provider-acme"
	group         = "sample"
	apiVersion    = "v1alpha1"
	widgetKind    = "Widget"
	gadgetKind    = "Gadget"
	headerMark    = "DO NOT EDIT"
	externalNS    = "default"
	externalCM    = "widget-example"
	managed       = "widget.sample.example.com/example"
	lifecycleTest = "lifecycle"
	widgetDir     = "internal/controller/widget"
	pollEvery     = 3 * time.Second
	pollTimeout   = 3 * time.Minute
)

var (
	widgetExternal = filepath.Join(widgetDir, "external.go")
	widgetWiring   = filepath.Join(widgetDir, "wiring.go")
	seamFiles      = []string{widgetExternal, "internal/provider/client.go", "internal/provider/options.go"}
	versionStamp   = regexp.MustCompile(`(?m)^    version: (.+)$`)
	unchanged      = regexp.MustCompile(`Unchanged [1-9][0-9]* tool-owned file\(s\)`)
)

// Env is everything a stage may touch. Every subprocess gets the private
// Kubeconfig and Cluster, so no stage can reach the user's own cluster.
type Env struct {
	Ctx        context.Context
	Repo       string // the generator checkout
	Head       string // the binary under test
	Gen        string // the binary that scaffolds: Head, or a released one
	Overlay    string // directory holding the user-logic overlay files
	Work       string
	Dir        string // the scaffolded provider
	Logs       string
	Kubeconfig string
	Cluster    string
	log        io.Writer
}

func (e *Env) runStage(s Stage) error {
	e.log = io.Discard
	if e.Logs != "" {
		f, err := os.Create(filepath.Join(e.Logs, s.Name+".log"))
		if err != nil {
			return fmt.Errorf("creating stage log: %w", err)
		}
		defer f.Close()
		e.log = f
	}
	return s.Run(e)
}

func (e *Env) exec(dir, stdin, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(e.Ctx, name, args...) // #nosec G204 -- the harness runs its own fixed commands, no shell
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "KUBECONFIG="+e.Kubeconfig, "KIND_CLUSTER_NAME="+e.Cluster)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var buf bytes.Buffer
	w := io.MultiWriter(&buf, e.log)
	cmd.Stdout, cmd.Stderr = w, w
	fmt.Fprintf(e.log, "$ %s %s\n", name, strings.Join(args, " "))
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		return out, fmt.Errorf("running %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

func (e *Env) head(args ...string) (string, error) { return e.exec(e.Dir, "", e.Head, args...) }

func (e *Env) git(args ...string) (string, error) { return e.exec(e.Dir, "", "git", args...) }

func (e *Env) makeIn(dir string, targets ...string) (string, error) {
	return e.exec(dir, "", "make", targets...)
}

func (e *Env) make(targets ...string) error {
	_, err := e.makeIn(e.Dir, targets...)
	return err
}

func (e *Env) kubectl(stdin string, args ...string) (string, error) {
	return e.exec(e.Dir, stdin, e.tool("kubectl"), args...)
}

// tool finds a binary the provider's own build downloaded, else the PATH one.
func (e *Env) tool(name string) string {
	m, _ := filepath.Glob(filepath.Join(e.Dir, ".cache/tools/*", name+"-*"))
	if len(m) == 0 {
		return name
	}
	return m[0]
}

func (e *Env) path(rel string) string { return filepath.Join(e.Dir, rel) }

func (e *Env) read(rel string) (string, error) {
	b, err := os.ReadFile(e.path(rel))
	return string(b), err
}

// copyOverlay lays the overlay tree over the scaffold; its paths mirror the
// scaffold's own.
func (e *Env) copyOverlay() error {
	overlay := os.DirFS(e.Overlay)
	return fs.WalkDir(overlay, ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(overlay, rel)
		if err != nil {
			return err
		}
		return e.write(rel, b)
	})
}

// write replaces a scaffold file. Every path comes from this package's own
// tables, never from input.
func (e *Env) write(rel string, data []byte) error {
	return os.WriteFile(e.path(rel), data, 0o600) // #nosec G703 -- fixed paths inside the private work directory
}

func (e *Env) commit(msg string) error {
	if out, err := e.git("status", "--porcelain"); err != nil || out == "" {
		return err
	}
	if _, err := e.git("add", "-A"); err != nil {
		return err
	}
	_, err := e.git("commit", "-qm", msg)
	return err
}

func (e *Env) appendLine(rel, line string) error {
	f, err := os.OpenFile(e.path(rel), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "\n%s\n", line)
	return err
}

func (e *Env) stamp() (string, error) {
	project, err := e.read("PROJECT")
	if err != nil {
		return "", err
	}
	m := versionStamp.FindStringSubmatch(project)
	if m == nil {
		return "", fmt.Errorf("PROJECT has no generator version stamp")
	}
	return m[1], nil
}

func (e *Env) assertCleanTree(after string) error {
	out, err := e.git("status", "--porcelain")
	if err != nil {
		return err
	}
	if out != "" {
		return fmt.Errorf("working tree is dirty after %s:\n%s", after, out)
	}
	return nil
}

func (e *Env) readSeams() (map[string]string, error) {
	seams := map[string]string{}
	for _, rel := range seamFiles {
		s, err := e.read(rel)
		if err != nil {
			return nil, err
		}
		seams[rel] = s
	}
	return seams, nil
}

func (e *Env) assertSeamsUnchanged(before map[string]string, after string) error {
	now, err := e.readSeams()
	if err != nil {
		return err
	}
	for rel, s := range before {
		if now[rel] != s {
			return fmt.Errorf("%s changed the user-owned %s", after, rel)
		}
	}
	return nil
}

// assertOwnership checks the ownership contract on the scaffold: tool-owned
// files carry the generated header, user-owned files never do.
func (e *Env) assertOwnership() error {
	toolOwned := []string{
		"apis/register.go", "internal/controller/register.go", "cmd/provider/main.go",
		"internal/controller/config/config.go", "internal/provider/connector.go",
		widgetWiring, "hack/xp-provider-gen.mk", "docs/ownership.md",
	}
	userOwned := append([]string{
		"apis/" + group + "/" + apiVersion + "/widget_types.go", "apis/v1alpha1/types.go", "Makefile", "AGENTS.md",
	}, seamFiles...)
	for want, files := range map[bool][]string{true: toolOwned, false: userOwned} {
		for _, rel := range files {
			s, err := e.read(rel)
			if err != nil {
				return err
			}
			if strings.Contains(s, headerMark) != want {
				return fmt.Errorf("%s: generated header present is %t, want %t", rel, !want, want)
			}
		}
	}
	return nil
}

func (e *Env) eventually(what string, read func() (string, error), want string) error {
	deadline := time.Now().Add(pollTimeout)
	last := ""
	var lastErr error
	for time.Now().Before(deadline) {
		got, err := read()
		if err == nil && got == want {
			return nil
		}
		last, lastErr = got, err
		select {
		case <-time.After(pollEvery):
		case <-e.Ctx.Done():
			return e.Ctx.Err()
		}
	}
	err := fmt.Errorf("%s: want %q, last saw %q", what, want, last)
	if lastErr != nil {
		err = fmt.Errorf("%w; last error: %w", err, lastErr)
	}
	return err
}

func (e *Env) externalValue() (string, error) {
	return e.kubectl("", "-n", externalNS, "get", "configmap", externalCM, "-o", "jsonpath={.data.value}")
}

func (e *Env) patchManaged(value string) error {
	_, err := e.kubectl("", "-n", externalNS, "patch", managed, "--type=merge",
		"-p", fmt.Sprintf(`{"spec":{"forProvider":{"configurableField":%q}}}`, value))
	return err
}

func contains(out, want, what string) error {
	if !strings.Contains(strings.ToLower(out), strings.ToLower(want)) {
		return fmt.Errorf("%s: output lacks %q:\n%s", what, want, out)
	}
	return nil
}

// refused asserts a command failed and said why.
func refused(out string, err error, want, what string) error {
	if err == nil {
		return fmt.Errorf("%s was accepted", what)
	}
	return contains(out, want, what)
}

func (e *Env) assertStamp(want string) error {
	got, err := e.stamp()
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("PROJECT stamp is %q, want %q", got, want)
	}
	return nil
}

func nativeStages() []Stage {
	docker := []Capability{CapDocker}
	return []Stage{
		{Name: "init", Run: stageInit},
		{Name: "create-api", Run: stageCreateAPI},
		{Name: "generate", Run: stageGenerate},
		{Name: "user-logic", Run: stageUserLogic},
		{Name: "update-to-head", When: Case.Release, Run: stageUpdateToHead},
		{Name: "update-noop", Run: stageUpdateNoop},
		{Name: "create-api-refresh", Run: stageCreateAPIRefresh},
		{Name: "adopt", Run: stageAdopt},
		{Name: "create-test", Run: stageCreateTest},
		{Name: "stock-e2e", Needs: docker, Run: stageStockE2E},
		{Name: "build-xpkg", Needs: docker, Run: func(e *Env) error { return e.make("build") }},
		{Name: "install-in-kind", Needs: docker, Run: stageInstallInKind},
		{Name: "provider-config", Needs: docker, Run: stageProviderConfig},
		{Name: "mr-ready-synced", Needs: docker, Run: stageReadySynced},
		{Name: "mr-update", Needs: docker, Run: stageMRUpdate},
		{Name: "drift-corrected", Needs: docker, Run: stageDriftCorrected},
		{Name: "mr-delete", Needs: docker, Run: stageMRDelete},
		{Name: "external-gone", Needs: docker, Run: stageExternalGone},
	}
}

func stageInit(e *Env) error {
	if _, err := e.exec(e.Dir, "", e.Gen, "init", "--domain="+domain, "--repo="+repo,
		"--git-name=e2e", "--git-email=e2e@example.com"); err != nil {
		return err
	}
	if _, err := e.stamp(); err != nil {
		return fmt.Errorf("reading the stamp init wrote: %w", err)
	}
	if _, err := e.exec(e.Repo, "", filepath.Join(e.Repo, "scripts/assert-layout.sh"), e.Dir); err != nil {
		return err
	}
	return e.assertCleanTree("init")
}

func stageCreateAPI(e *Env) error {
	for _, kind := range []string{widgetKind, gadgetKind} {
		if _, err := e.exec(e.Dir, "", e.Gen, "create", "api",
			"--group="+group, "--version="+apiVersion, "--kind="+kind); err != nil {
			return err
		}
		if _, err := e.exec(e.Repo, "", filepath.Join(e.Repo, "scripts/assert-layout.sh"),
			e.Dir, group, apiVersion, kind); err != nil {
			return err
		}
	}
	return nil
}

func stageGenerate(e *Env) error {
	if err := e.make("submodules", "generate", "reviewable"); err != nil {
		return err
	}
	for _, kind := range []string{"widget", "gadget"} {
		for _, rel := range []string{
			fmt.Sprintf("package/crds/%s.%s_%ss.yaml", group, domain, kind),
			fmt.Sprintf("examples/%s/%s.yaml", group, kind),
		} {
			if _, err := os.Stat(e.path(rel)); err != nil {
				return fmt.Errorf("generate left no %s: %w", rel, err)
			}
		}
	}
	if err := e.assertOwnership(); err != nil {
		return err
	}
	if err := e.assertCleanTree("generate"); err != nil {
		return err
	}
	subjects, err := e.git("log", "--format=%s")
	if err != nil {
		return err
	}
	if subjects != "Initial commit" {
		return fmt.Errorf("want one Initial commit, git log has:\n%s", subjects)
	}
	return nil
}

func stageUserLogic(e *Env) error {
	if err := e.copyOverlay(); err != nil {
		return err
	}
	if _, err := e.exec(e.Dir, "", "go", "mod", "tidy"); err != nil {
		return err
	}
	if err := e.make("lint"); err != nil {
		return err
	}
	return e.commit("user logic")
}

func stageUpdateToHead(e *Env) error {
	seams, err := e.readSeams()
	if err != nil {
		return err
	}
	was, err := e.stamp()
	if err != nil {
		return err
	}
	if _, err := e.head("update"); err != nil {
		return err
	}
	if err := e.assertSeamsUnchanged(seams, "update"); err != nil {
		return err
	}
	now, err := e.stamp()
	if err != nil {
		return err
	}
	if now == was {
		return fmt.Errorf("update left the stamp at %q", was)
	}
	return e.commit("update to head")
}

func stageUpdateNoop(e *Env) error {
	seams, err := e.readSeams()
	if err != nil {
		return err
	}
	out, err := e.head("update")
	if err != nil {
		return err
	}
	for _, want := range []string{"removed 0", "Refreshed 0 tool-owned file(s)"} {
		if err := contains(out, want, "same-generator update"); err != nil {
			return err
		}
	}
	if !unchanged.MatchString(out) {
		return fmt.Errorf("same-generator update reported no unchanged files:\n%s", out)
	}
	if err := e.assertSeamsUnchanged(seams, "update"); err != nil {
		return err
	}
	if err := e.assertOwnership(); err != nil {
		return err
	}
	if err := e.assertCleanTree("a same-generator update"); err != nil {
		return err
	}

	if err := e.appendLine(widgetExternal, "// dirty"); err != nil {
		return err
	}
	out, err = e.head("update")
	if err := refused(out, err, "working tree", "update on a dirty tree"); err != nil {
		return err
	}
	if _, err := e.git("checkout", "--", widgetExternal); err != nil {
		return err
	}
	out, err = e.head("update", "--terraform-provider-version=1.0.0")
	if err := refused(out, err, `flavor is "native"`, "update --terraform-provider-version"); err != nil {
		return err
	}
	return e.assertCleanTree("the refused attempts")
}

func stageCreateAPIRefresh(e *Env) error {
	createWidget := []string{"create", "api", "--group=" + group, "--version=" + apiVersion, "--kind=" + widgetKind}
	if err := e.refreshKeepsUserEdits(createWidget); err != nil {
		return err
	}
	before, err := e.git("rev-parse", "HEAD")
	if err != nil {
		return err
	}
	out, err := e.head(createWidget...)
	if err != nil {
		return err
	}
	if err := contains(out, "No changes to commit", "second create api"); err != nil {
		return err
	}
	if after, err := e.git("rev-parse", "HEAD"); err != nil || after != before {
		return fmt.Errorf("second create api moved HEAD from %s to %s", before, after)
	}
	out, err = e.head(append(createWidget, "--force")...)
	if err := refused(out, err, "unknown flag: --force", "create api --force"); err != nil {
		return err
	}
	return e.assertCleanTree("the refused --force attempt")
}

// refreshKeepsUserEdits marks a tool-owned and a user-owned file, re-runs
// create api, and asserts only the tool-owned mark is gone.
func (e *Env) refreshKeepsUserEdits(createWidget []string) error {
	const toolMark, userMark = "// REFRESH-MARKER: stale tool-owned content", "// REFRESH-MARKER: user customization"
	if err := e.appendLine(widgetWiring, toolMark); err != nil {
		return err
	}
	if err := e.appendLine(widgetExternal, userMark); err != nil {
		return err
	}
	if err := e.commit("simulate drift"); err != nil {
		return err
	}
	if _, err := e.head(createWidget...); err != nil {
		return err
	}
	wiring, err := e.read(widgetWiring)
	if err != nil {
		return err
	}
	external, err := e.read(widgetExternal)
	if err != nil {
		return err
	}
	if strings.Contains(wiring, toolMark) {
		return fmt.Errorf("stale tool-owned %s survived create api", widgetWiring)
	}
	if !strings.Contains(external, userMark) {
		return fmt.Errorf("user-owned %s was overwritten by create api", widgetExternal)
	}
	return nil
}

// stageAdopt plants a sentinel stamp and strips a header, then asserts adopt
// restores both. The sentinel is always planted, so the stamp check cannot
// pass on a PROJECT that adopt never touched.
func stageAdopt(e *Env) error {
	const sentinel = "pre-contract"
	want, err := e.stamp()
	if err != nil {
		return err
	}
	project, err := e.read("PROJECT")
	if err != nil {
		return err
	}
	wiring, err := e.read(widgetWiring)
	if err != nil {
		return err
	}
	stripped, err := stripHeader(wiring)
	if err != nil {
		return fmt.Errorf("%s: %w", widgetWiring, err)
	}
	for rel, s := range map[string]string{
		"PROJECT":    versionStamp.ReplaceAllString(project, "    version: "+sentinel),
		widgetWiring: stripped,
	} {
		if err := e.write(rel, []byte(s)); err != nil {
			return err
		}
	}
	if err := e.assertStamp(sentinel); err != nil {
		return err
	}
	if err := e.commit("simulate a provider from before the ownership contract"); err != nil {
		return err
	}
	if _, err := e.head("update", "--adopt"); err != nil {
		return err
	}
	if wiring, err = e.read(widgetWiring); err != nil {
		return err
	}
	if !strings.Contains(wiring, headerMark) {
		return fmt.Errorf("header still missing on %s after adopt", widgetWiring)
	}
	if err := e.assertStamp(want); err != nil {
		return err
	}
	return e.commit("adopt")
}

func stripHeader(s string) (string, error) {
	var kept []string
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		if !strings.Contains(line, headerMark) {
			kept = append(kept, line)
		}
	}
	if len(kept) == len(lines) {
		return "", fmt.Errorf("no header to strip")
	}
	return strings.Join(kept, "\n"), nil
}

// stageStockE2E runs the author's entry point, make e2e, on the scaffold as
// the generator wrote it: a clone of its first commit, before user logic. The
// head's create-test adds a lifecycle test first, so the one run covers uptest,
// the pause tests and the test create-test wrote, all on the unmodified stub.
func stageStockE2E(e *Env) error {
	stock := filepath.Join(e.Work, "stock")
	root, err := e.git("rev-list", "--max-parents=0", "HEAD")
	if err != nil {
		return err
	}
	if _, err := e.exec(e.Work, "", "git", "clone", "-q", e.Dir, stock); err != nil {
		return err
	}
	if _, err := e.exec(stock, "", "git", "checkout", "-q", root); err != nil {
		return err
	}
	defer func() { _, _ = e.makeIn(stock, "e2e-clean") }()
	if _, err := e.makeIn(stock, "submodules"); err != nil {
		return err
	}
	if _, err := e.exec(stock, "", e.Head, "create-test", "--kind="+widgetKind, "--name="+lifecycleTest); err != nil {
		return err
	}
	_, err = e.makeIn(stock, "e2e")
	return err
}

// pollRuntimeConfig is the DeploymentRuntimeConfig the harness deploys the
// provider with: the build's default plus a short --poll, so drift-corrected
// waits seconds instead of the shipped one-minute poll interval. The build
// swaps in the image, so none is set here.
const pollRuntimeConfig = `apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: runtimeconfig-` + provider + `
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args: ["--debug", "--poll=10s"]
`

func stageInstallInKind(e *Env) error {
	drc := filepath.Join(e.Work, "runtimeconfig.yaml")
	if err := os.WriteFile(drc, []byte(pollRuntimeConfig), 0o600); err != nil {
		return err
	}
	deploy := "local.xpkg.deploy.provider." + provider
	if err := e.make("DRC_FILE="+drc, "controlplane.down", "controlplane.up", deploy); err != nil {
		return err
	}
	_, err := e.kubectl("", "wait", "provider.pkg", provider, "--for=condition=Healthy", "--timeout=300s")
	return err
}

func stageProviderConfig(e *Env) error {
	creds, err := e.exec(e.Dir, "", e.tool("kind"), "get", "kubeconfig", "--internal", "--name", e.Cluster)
	if err != nil {
		return err
	}
	if _, err := e.kubectl("", "apply", "-f", "examples/provider/config.yaml"); err != nil {
		return err
	}
	// The example's Secret holds placeholder credentials; replace them with
	// the kubeconfig the overlay client dials the external system with.
	f := filepath.Join(e.Work, "internal.kubeconfig")
	if err := os.WriteFile(f, []byte(creds), 0o600); err != nil {
		return err
	}
	secret, err := e.kubectl("", "-n", externalNS, "create", "secret", "generic", "example-provider-secret",
		"--from-file=credentials="+f, "--dry-run=client", "-o", "yaml")
	if err != nil {
		return err
	}
	_, err = e.kubectl(secret, "apply", "-f", "-")
	return err
}

func stageReadySynced(e *Env) error {
	if _, err := e.kubectl("", "apply", "-f", "examples/"+group+"/widget.yaml"); err != nil {
		return err
	}
	for _, cond := range []string{"Ready", "Synced"} {
		if _, err := e.kubectl("", "-n", externalNS, "wait", managed, "--for=condition="+cond, "--timeout=180s"); err != nil {
			return err
		}
	}
	return e.eventually("external value after create", e.externalValue, "test")
}

func stageMRUpdate(e *Env) error {
	if err := e.patchManaged("two"); err != nil {
		return err
	}
	return e.eventually("external value follows the spec", e.externalValue, "two")
}

func stageDriftCorrected(e *Env) error {
	if _, err := e.kubectl("", "-n", externalNS, "patch", "configmap", externalCM,
		"--type=merge", "-p", `{"data":{"value":"drifted"}}`); err != nil {
		return err
	}
	return e.eventually("hand edit restored", e.externalValue, "two")
}

func stageMRDelete(e *Env) error {
	_, err := e.kubectl("", "-n", externalNS, "delete", managed, "--timeout=180s")
	return err
}

func stageExternalGone(e *Env) error {
	out, err := e.kubectl("", "-n", externalNS, "get", "configmap", externalCM, "--ignore-not-found", "-o", "name")
	if err != nil {
		return err
	}
	if out != "" {
		return fmt.Errorf("external object outlived its managed resource: %s", out)
	}
	return nil
}

func stageCreateTest(e *Env) error {
	create := []string{"create-test", "--kind=" + widgetKind, "--name=" + lifecycleTest}
	if _, err := e.head(create...); err != nil {
		return err
	}
	if _, err := os.Stat(e.path("test/behavior/" + lifecycleTest + "/chainsaw-test.yaml")); err != nil {
		return fmt.Errorf("finding the test create-test wrote: %w", err)
	}
	out, err := e.head(create...)
	return refused(out, err, "already exist", "create-test over an existing test")
}
