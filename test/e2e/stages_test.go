//go:build !windows

package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func suite(cases string) string {
	return `<testsuites name="chainsaw-report"><testsuite name="` + lifecycleTest + `">` + cases + `</testsuite></testsuites>`
}

func TestAssertChainsawRanNeedsTheLifecycleTestInTheReport(t *testing.T) {
	cases := []struct {
		name    string
		report  string
		wantErr bool
	}{
		{"passed", suite(`<testcase name="` + lifecycleTest + `"/>`), false},
		{"skipped", suite(`<testcase name="` + lifecycleTest + `"><skipped/></testcase>`), true},
		{"failure", suite(`<testcase name="` + lifecycleTest + `"><failure message="x"/></testcase>`), true},
		{"error", suite(`<testcase name="` + lifecycleTest + `"><error message="x"/></testcase>`), true},
		{"absent", suite(`<testcase name="other"/>`), true},
		{"report without the test", `<testsuites><testsuite name="uptest"/></testsuites>`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := filepath.Join(t.TempDir(), "junit.xml")
			if err := os.WriteFile(f, []byte(c.report), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := assertChainsawRan(f); (err != nil) != c.wantErr {
				t.Fatalf("assertChainsawRan = %v, wantErr %t", err, c.wantErr)
			}
		})
	}
}

func TestAssertChainsawRanFailsWithoutAReport(t *testing.T) {
	if err := assertChainsawRan(filepath.Join(t.TempDir(), "junit.xml")); err == nil {
		t.Fatal("a missing report passed")
	}
}

func TestAssertProviderPollsNeedsThePollArgOnTheProviderDeployment(t *testing.T) {
	deploy := func(label, args string) string {
		return `{"items":[{"spec":{"selector":{"matchLabels":{"pkg.crossplane.io/provider":"` + label + `"}},` +
			`"template":{"spec":{"containers":[{"args":[` + args + `]}]}}}}]}`
	}
	cases := []struct {
		name        string
		deployments string
		wantErr     bool
	}{
		{"short poll applied", deploy(provider, `"--debug","`+pollArg+`"`), false},
		{"fallback runtime config", deploy(provider, `"--debug"`), true},
		{"another provider's deployment", deploy("provider-other", `"`+pollArg+`"`), true},
		{"no deployments", `{"items":[]}`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := assertProviderPolls(c.deployments); (err != nil) != c.wantErr {
				t.Fatalf("assertProviderPolls = %v, wantErr %t", err, c.wantErr)
			}
		})
	}
}

func owned(files map[string]string) map[string][]byte {
	m := map[string][]byte{}
	for rel, s := range files {
		m[rel] = []byte(s)
	}
	return m
}

func TestCompareToolOwnedNeedsTheFreshScaffoldsFilesWithTheSameBytes(t *testing.T) {
	fresh := func() map[string]string {
		m := map[string]string{}
		for _, rel := range requiredOwned {
			m[rel] = "head " + rel
		}
		return m
	}
	tampered := fresh()
	tampered[requiredOwned[0]] = "stale"
	missing := fresh()
	delete(missing, requiredOwned[1])
	extra := fresh()
	extra["internal/old.go"] = "a file head no longer renders"
	noConnector := fresh()
	delete(noConnector, requiredOwned[0])

	cases := []struct {
		name      string
		got, want map[string]string
		wantErr   string
	}{
		{"identical", fresh(), fresh(), ""},
		{"a stale file", tampered, fresh(), "differs from a fresh scaffold: " + requiredOwned[0]},
		{"a file update never wrote", missing, fresh(), "missing after update: " + requiredOwned[1]},
		{"a file update left behind", extra, fresh(), "not in a fresh scaffold: internal/old.go"},
		{"an empty fresh set", fresh(), nil, errNoOwned.Error()},
		{"a fresh set without the connector", noConnector, noConnector, "no tool-owned " + requiredOwned[0]},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := compareToolOwned(owned(c.got), owned(c.want))
			if c.wantErr == "" && err != nil {
				t.Fatalf("compareToolOwned = %v, want nil", err)
			}
			if c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
				t.Fatalf("compareToolOwned = %v, want %q", err, c.wantErr)
			}
		})
	}
}

func TestMakeSkippingGenerateSkipsOnlyGenerateAndReviewable(t *testing.T) {
	e := &Env{Ctx: t.Context(), Work: t.TempDir(), log: io.Discard}
	env, err := e.makeSkippingGenerate()
	if err != nil {
		t.Fatal(err)
	}
	for target, wantRan := range map[string]bool{"generate": false, "reviewable": false, "no-such-target": true} {
		_, err := e.execEnv(env, e.Work, "", filepath.Join(e.Work, "shim", "make"), target)
		if ran := err != nil; ran != wantRan {
			t.Errorf("make %s reached the real make: %t, want %t (%v)", target, ran, wantRan, err)
		}
	}
}

func TestCompareUserOwnedIgnoresGeneratedOutputAndFlagsAuthorFiles(t *testing.T) {
	const crd = "package/crds/acme.example_providerconfigs.yaml"
	const deepcopy = "apis/v1alpha1/zz_generated.deepcopy.go"
	const author = "internal/provider/options.go"
	const mine, oldGen, newGen = "mine", "old", "new"
	filter := func(m map[string]string) map[string][]byte {
		out := map[string][]byte{}
		for rel, b := range m {
			out[rel] = []byte(b)
		}
		return keepUserOwned(out)
	}
	before := map[string]string{author: mine, crd: oldGen, deepcopy: oldGen}
	cases := []struct {
		name    string
		now     map[string]string
		wantErr string
	}{
		{"a controller-gen bump", map[string]string{author: mine, crd: newGen, deepcopy: newGen}, ""},
		{"a generated file update adds", map[string]string{author: mine, crd: oldGen, deepcopy: oldGen, "apis/v1alpha1/zz_generated.managed.go": newGen}, ""},
		{"an author file changes", map[string]string{author: "theirs", crd: oldGen, deepcopy: oldGen}, "changed by update: " + author},
		{"an author file goes", map[string]string{crd: oldGen, deepcopy: oldGen}, "gone after update: " + author},
		{"update adds an author file", map[string]string{author: mine, crd: oldGen, deepcopy: oldGen, "internal/provider/extra.go": "x"}, "added by update: internal/provider/extra.go"},
		{"a name that only resembles generated output", map[string]string{author: mine, "my_zz_generated.go": "x", "docs/package/crds/a.yaml": "x"}, "added by update: docs/package/crds/a.yaml\nadded by update: my_zz_generated.go"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := compareUserOwned(filter(before), filter(c.now))
			if c.wantErr == "" && err != nil {
				t.Fatalf("compareUserOwned = %v, want nil", err)
			}
			if c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
				t.Fatalf("compareUserOwned = %v, want %q", err, c.wantErr)
			}
		})
	}
}
