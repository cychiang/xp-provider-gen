//go:build !windows

package main

import (
	"os"
	"path/filepath"
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
