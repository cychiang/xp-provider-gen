//go:build !windows

package main

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const (
	credentialsField = "Credentials ProviderCredentials `json:\"credentials\"`\n}"
	endpointField    = "Credentials ProviderCredentials `json:\"credentials\"`\n\n" +
		"\t// Endpoint is the e2e overlay's own ProviderConfig field.\n" +
		"\t// +optional\n\tEndpoint string `json:\"endpoint,omitempty\"`\n}"
	endpointValue = "https://acme.example"
	headerWindow  = 1024
	// newerStamp and olderStamp are clean release versions, the only shape a
	// released binary orders, set by hand: no published release is newer than
	// the first or older than the second.
	newerStamp        = "v9.9.9"
	olderStamp        = "v0.0.1"
	refuseOlder       = "is older than the one that last updated this project"
	refuseNewer       = "update' first"
	refuseCreateOlder = "is older than"
	volumeKind        = "Volume"
)

var (
	errNoOwned   = errors.New("no tool-owned file found")
	userTestPkgs = []string{"./internal/provider/", "./" + widgetDir + "/"}
	providerCRD  = "package/crds/" + domain + "_providerconfigs.yaml"
)

// addProviderConfigField gives ProviderConfigSpec a field of the author's
// own, in the user-owned types.go. Every release scaffolds the same struct.
func (e *Env) addProviderConfigField() error {
	types, err := e.read(providerTypes)
	if err != nil {
		return err
	}
	if strings.Count(types, credentialsField) != 1 {
		return fmt.Errorf("%s has no ProviderConfigSpec to extend", providerTypes)
	}
	return e.write(providerTypes, []byte(strings.Replace(types, credentialsField, endpointField, 1)))
}

// assertCRDHasEndpoint fails when the generated ProviderConfig CRD lacks the
// field, so generate must have picked up the author's type.
func (e *Env) assertCRDHasEndpoint() error {
	crd, err := e.read(providerCRD)
	if err != nil {
		return err
	}
	if !strings.Contains(crd, "endpoint:") {
		return fmt.Errorf("%s lacks the user-added endpoint field", providerCRD)
	}
	return nil
}

// assertEndpointSurvivesInCluster sets the user-added field on the deployed
// ProviderConfig and reads it back: a CRD without the field prunes it.
func (e *Env) assertEndpointSurvivesInCluster() error {
	const resource = "providerconfigs." + domain
	if _, err := e.kubectl("", "-n", externalNS, "patch", resource, "example", "--type=merge",
		"-p", fmt.Sprintf(`{"spec":{"endpoint":%q}}`, endpointValue)); err != nil {
		return err
	}
	got, err := e.kubectl("", "-n", externalNS, "get", resource, "example", "-o", "jsonpath={.spec.endpoint}")
	if err != nil {
		return err
	}
	if got != endpointValue {
		return fmt.Errorf("deployed ProviderConfig kept endpoint %q, want %q: the CRD lacks the field", got, endpointValue)
	}
	return nil
}

// assertUserLogicWorks runs the author's own unit tests and checks the
// provider binary still offers the author's --region flag.
func (e *Env) assertUserLogicWorks() error {
	if _, err := e.exec(e.Dir, "", "go", append([]string{"test", "-count=1"}, userTestPkgs...)...); err != nil {
		return fmt.Errorf("user unit tests: %w", err)
	}
	bin := filepath.Join(e.Work, "provider-bin")
	if _, err := e.exec(e.Dir, "", "go", "build", "-o", bin, "./cmd/provider"); err != nil {
		return err
	}
	help, err := e.exec(e.Dir, "", bin, "--help")
	if err != nil {
		return err
	}
	return contains(help, "--region", "provider --help")
}

// assertToolOwnedMatchesFreshScaffold scaffolds the same project with the
// head binary and requires every tool-owned file of the updated tree to be
// byte-identical to the fresh one's, and the two sets to be the same. Only the
// scaffold's own files are compared, so the fresh run skips the generate and
// reviewable that init and create api would otherwise run for minutes.
func (e *Env) assertToolOwnedMatchesFreshScaffold() (int, int, error) {
	fresh, err := os.MkdirTemp(e.Work, "fresh-")
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = os.RemoveAll(fresh) }()
	pathEnv, err := e.makeSkippingGenerate()
	if err != nil {
		return 0, 0, err
	}
	if _, err := e.execEnv(pathEnv, fresh, "", e.Head, e.F.Init...); err != nil {
		return 0, 0, err
	}
	for _, kind := range e.F.Kinds {
		if _, err := e.execEnv(pathEnv, fresh, "", e.Head, e.F.CreateAPI(kind)...); err != nil {
			return 0, 0, err
		}
	}
	got, err := e.toolOwned(e.Dir)
	if err != nil {
		return 0, 0, err
	}
	want, err := e.toolOwned(fresh)
	if err != nil {
		return 0, 0, err
	}
	return compareToolOwned(got, want, e.F.RequiredOwned)
}

const shimScript = `#!/bin/sh
for t in "$@"; do case "$t" in generate|reviewable) exit 0;; esac; done
exec '%s' "$@"
`

// makeSkippingGenerate writes a make that exits 0 for generate and reviewable
// and runs the real one otherwise, and returns the PATH entry that puts it
// first.
func (e *Env) makeSkippingGenerate() ([]string, error) {
	realMake, err := exec.LookPath("make")
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(e.Work, "shim")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	script := fmt.Sprintf(shimScript, realMake)
	// #nosec G306 -- an executable script in the private work directory
	if err := os.WriteFile(filepath.Join(dir, "make"), []byte(script), 0o700); err != nil {
		return nil, err
	}
	return []string{"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")}, nil
}

// files reads every file of a scaffold, tracked or not but not ignored.
func (e *Env) files(dir string) (map[string][]byte, error) {
	listed, err := e.exec(dir, "", "git", "ls-files", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	all := map[string][]byte{}
	for _, rel := range strings.Split(listed, "\n") {
		path := filepath.Join(dir, rel)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) || (err == nil && info.IsDir()) {
			continue
		}
		if err != nil {
			return nil, err
		}
		b, err := os.ReadFile(path) // #nosec G304 -- a path git listed inside the private work directory
		if err != nil {
			return nil, err
		}
		all[rel] = b
	}
	return all, nil
}

func hasGeneratedHeader(b []byte) bool {
	return bytes.Contains(b[:min(len(b), headerWindow)], []byte(generatedHead))
}

// toolOwned reads the files of a scaffold that carry the generator's header
// in their first bytes.
func (e *Env) toolOwned(dir string) (map[string][]byte, error) {
	all, err := e.files(dir)
	if err != nil {
		return nil, err
	}
	maps.DeleteFunc(all, func(_ string, b []byte) bool { return !hasGeneratedHeader(b) })
	return all, nil
}

// compareToolOwned fails unless the updated tree has the fresh scaffold's
// tool-owned files, no more and no fewer, with the same bytes. It also fails
// on a set missing the files whose refresh matters most, so an empty or
// header-less walk cannot pass. It returns how many files it compared and how
// many differ, whether or not it fails.
func compareToolOwned(got, want map[string][]byte, required []string) (int, int, error) {
	if len(want) == 0 {
		return 0, 0, errNoOwned
	}
	for _, rel := range required {
		if _, ok := want[rel]; !ok {
			return len(want), 0, fmt.Errorf("fresh scaffold has no tool-owned %s", rel)
		}
	}
	var problems []string
	for rel, w := range want {
		g, ok := got[rel]
		switch {
		case !ok:
			problems = append(problems, "missing after update: "+rel)
		case !bytes.Equal(g, w):
			problems = append(problems, "differs from a fresh scaffold: "+rel)
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			problems = append(problems, "not in a fresh scaffold: "+rel)
		}
	}
	if len(problems) > 0 {
		slices.Sort(problems)
		return len(want), len(problems), fmt.Errorf("tool-owned files after update do not match a fresh scaffold:\n%s",
			strings.Join(problems, "\n"))
	}
	return len(want), 0, nil
}

// stageRefuseOldGenerator runs the release binary against the updated tree.
// A released binary orders only clean release versions, so the tree is
// stamped with one by hand and the stamp is restored afterwards. The
// refusals are the release's own.
func stageRefuseOldGenerator(e *Env) error {
	want, err := e.stamp()
	if err != nil {
		return err
	}
	if err := e.withStamp(newerStamp, e.refuseOnNewerProject); err != nil {
		return err
	}
	err = e.withStamp(olderStamp, func() error { return e.refusedCreateAPI(refuseNewer, "an older project") })
	if err != nil {
		return err
	}
	return e.assertStamp(want)
}

func (e *Env) refuseOnNewerProject() error {
	for _, args := range [][]string{{"update"}, {"update", "--adopt"}} {
		out, err := e.gen(args...)
		if err := refused(out, err, refuseOlder, "release "+strings.Join(args, " ")+" on a newer project"); err != nil {
			return err
		}
		if err := e.assertCleanTree("a refused " + strings.Join(args, " ")); err != nil {
			return err
		}
	}
	return e.refusedCreateAPI(refuseCreateOlder, "a newer project")
}

// withStamp commits a hand-set PROJECT stamp, runs check, then drops the commit.
func (e *Env) withStamp(version string, check func() error) error {
	project, err := e.read("PROJECT")
	if err != nil {
		return err
	}
	if err := e.write("PROJECT", []byte(versionStamp.ReplaceAllString(project, "    version: "+version))); err != nil {
		return err
	}
	if err := e.commit("stamp " + version); err != nil {
		return err
	}
	if err := check(); err != nil {
		return err
	}
	_, err = e.git("reset", "-q", "--hard", "HEAD~1")
	return err
}

func (e *Env) refusedCreateAPI(reason, on string) error {
	out, err := e.gen(e.F.CreateAPI(volumeKind)...)
	if err := refused(out, err, reason, "release create api on "+on); err != nil {
		return err
	}
	return e.assertCleanTree("a refused create api on " + on)
}

// updateRewrites names the files without the generator's header that update
// is allowed to change, each with why. Every other such file is the author's
// and must come out of update byte-identical, so a new user-owned file is
// covered without being listed.
var updateRewrites = []struct{ match, why string }{
	{"PROJECT", "update stamps the generator's version"},
	{"go.mod", "update moves the go directive to the one head pins"},
	{"go.sum", "follows go.mod"},
}

func rewrittenByUpdate(rel string) (string, bool) {
	for _, r := range updateRewrites {
		if ok, _ := filepath.Match(r.match, rel); ok {
			return r.why, true
		}
	}
	return "", false
}

// generatedOutputs names make generate's output, which carries no
// xp-provider-gen header and which a controller-gen, angryjet or upjet bump
// legitimately rewrites. No template renders into any of them, so no author
// file can match.
type generatedOutput struct {
	match func(rel string) bool
	why   string
}

var generatedOutputs = []generatedOutput{
	{
		func(rel string) bool { return strings.HasPrefix(filepath.Base(rel), "zz_") },
		"controller-gen, angryjet and upjet write their code to zz_* files",
	},
	{func(rel string) bool { return strings.HasPrefix(rel, "package/crds/") }, "the CRDs generated from the types"},
	{
		func(rel string) bool { return strings.HasPrefix(rel, "examples-generated/") },
		"upjet's examples from the Terraform docs",
	},
	{
		func(rel string) bool { return rel == "config/schema.json" || rel == "config/provider-metadata.yaml" },
		"upjet's scrape of the Terraform schema and docs, embedded in config",
	},
}

func isGeneratedOutput(rel string) bool {
	return slices.ContainsFunc(generatedOutputs, func(g generatedOutput) bool { return g.match(rel) })
}

// userOwned reads the files of a scaffold that carry no generator header and
// that update neither rewrites nor generates.
func (e *Env) userOwned() (map[string][]byte, error) {
	all, err := e.files(e.Dir)
	if err != nil {
		return nil, err
	}
	return keepUserOwned(all), nil
}

func keepUserOwned(all map[string][]byte) map[string][]byte {
	maps.DeleteFunc(all, func(rel string, b []byte) bool {
		_, rewritten := rewrittenByUpdate(rel)
		return hasGeneratedHeader(b) || rewritten || isGeneratedOutput(rel)
	})
	return all
}

// assertUserOwnedUnchanged compares the user-owned files now with before, leaving
// out the paths in except, which the caller checks itself. It returns how many
// files changed, whether or not it fails.
func (e *Env) assertUserOwnedUnchanged(before map[string][]byte, except ...string) (int, error) {
	now, err := e.userOwned()
	if err != nil {
		return 0, err
	}
	for _, rel := range except {
		delete(before, rel)
		delete(now, rel)
	}
	return compareUserOwned(before, now)
}

// compareUserOwned fails on a user-owned file that update removed, changed or
// added, and returns how many.
func compareUserOwned(before, now map[string][]byte) (int, error) {
	problems := differences(before, now, "update")
	if len(problems) > 0 {
		return len(problems), fmt.Errorf("update touched user-owned files:\n%s", strings.Join(problems, "\n"))
	}
	return 0, nil
}

// differences names, sorted, each file of before that now lacks or holds with
// other bytes and each file of now that before lacks.
func differences(before, now map[string][]byte, by string) []string {
	var problems []string
	for rel, b := range before {
		got, ok := now[rel]
		switch {
		case !ok:
			problems = append(problems, "gone after "+by+": "+rel)
		case !bytes.Equal(got, b):
			problems = append(problems, "changed by "+by+": "+rel)
		}
	}
	for rel := range now {
		if _, ok := before[rel]; !ok {
			problems = append(problems, "added by "+by+": "+rel)
		}
	}
	slices.Sort(problems)
	return problems
}

// assertGenerateChangesNothing runs the provider's own generate and fails on
// every file it changes, adds or removes. The byte comparison with a fresh
// scaffold leaves generated output out, and stale output that still compiles
// would otherwise pass. It returns how many files changed.
func (e *Env) assertGenerateChangesNothing() (int, error) {
	before, err := e.files(e.Dir)
	if err != nil {
		return 0, err
	}
	if err := e.make(targetGenerate); err != nil {
		return 0, err
	}
	after, err := e.files(e.Dir)
	if err != nil {
		return 0, err
	}
	problems := differences(before, after, targetGenerate)
	if len(problems) > 0 {
		return len(problems), fmt.Errorf("generate after update changed the tree, so its output was stale:\n%s",
			strings.Join(problems, "\n"))
	}
	return 0, nil
}
