//go:build !windows

package main

import (
	"bytes"
	"errors"
	"fmt"
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
	errNoOwned    = errors.New("no tool-owned file found")
	requiredOwned = []string{"internal/provider/connector.go", widgetWiring, "hack/xp-provider-gen.mk"}
	userTestPkgs  = []string{"./internal/provider/", "./" + widgetDir + "/"}
	providerCRD   = "package/crds/" + domain + "_providerconfigs.yaml"
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
func (e *Env) assertToolOwnedMatchesFreshScaffold() error {
	fresh := filepath.Join(e.Work, "fresh")
	if err := os.MkdirAll(fresh, 0o750); err != nil {
		return err
	}
	pathEnv, err := e.makeSkippingGenerate()
	if err != nil {
		return err
	}
	if _, err := e.execEnv(pathEnv, fresh, "", e.Head, initArgs()...); err != nil {
		return err
	}
	for _, kind := range []string{widgetKind, gadgetKind} {
		if _, err := e.execEnv(pathEnv, fresh, "", e.Head, createAPIArgs(kind)...); err != nil {
			return err
		}
	}
	got, err := e.toolOwned(e.Dir)
	if err != nil {
		return err
	}
	want, err := e.toolOwned(fresh)
	if err != nil {
		return err
	}
	return compareToolOwned(got, want)
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

// toolOwned reads the files of a scaffold, tracked or not but not ignored,
// that carry the generator's header in their first bytes.
func (e *Env) toolOwned(dir string) (map[string][]byte, error) {
	listed, err := e.exec(dir, "", "git", "ls-files", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	owned := map[string][]byte{}
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
		if bytes.Contains(b[:min(len(b), headerWindow)], []byte(generatedHead)) {
			owned[rel] = b
		}
	}
	return owned, nil
}

// compareToolOwned fails unless the updated tree has the fresh scaffold's
// tool-owned files, no more and no fewer, with the same bytes. It also fails
// on a set missing the files whose refresh matters most, so an empty or
// header-less walk cannot pass.
func compareToolOwned(got, want map[string][]byte) error {
	if len(want) == 0 {
		return errNoOwned
	}
	for _, rel := range requiredOwned {
		if _, ok := want[rel]; !ok {
			return fmt.Errorf("fresh scaffold has no tool-owned %s", rel)
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
		return fmt.Errorf("tool-owned files after update do not match a fresh scaffold:\n%s", strings.Join(problems, "\n"))
	}
	return nil
}

// stageRefuseOldGenerator runs the release binary against the updated tree.
// A released binary orders only clean release versions, so the tree is
// stamped with one by hand and the stamp is restored afterwards. The
// refusals are the release's own; one that predates a check is not asked.
func stageRefuseOldGenerator(e *Env) error {
	want, err := e.stamp()
	if err != nil {
		return err
	}
	if err := e.withStamp(newerStamp, e.refuseOnNewerProject); err != nil {
		return err
	}
	if e.CreateAPIGuard {
		err := e.withStamp(olderStamp, func() error { return e.refusedCreateAPI(refuseNewer, "an older project") })
		if err != nil {
			return err
		}
	}
	return e.assertStamp(want)
}

func (e *Env) refuseOnNewerProject() error {
	for _, args := range [][]string{{"update"}, {"update", "--adopt"}} {
		out, err := e.exec(e.Dir, "", e.Gen, args...)
		if err := refused(out, err, refuseOlder, "release "+strings.Join(args, " ")+" on a newer project"); err != nil {
			return err
		}
		if err := e.assertCleanTree("a refused " + strings.Join(args, " ")); err != nil {
			return err
		}
	}
	if !e.CreateAPIGuard {
		return nil
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
	out, err := e.gen(createAPIArgs(volumeKind)...)
	if err := refused(out, err, reason, "release create api on "+on); err != nil {
		return err
	}
	return e.assertCleanTree("a refused create api on " + on)
}
