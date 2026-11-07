//go:build !windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	tfProvider        = "hashicorp/kubernetes"
	tfFrom            = "2.37.1"
	tfTo              = "2.38.0"
	upjetGroup        = "core"
	probeKind         = "Secret"
	configMapKind     = "ConfigMap"
	probeResource     = "kubernetes_secret"
	configMapResource = "kubernetes_config_map"
	crossplaneNS      = "crossplane-system"
	providerConfigYML = "examples/providerconfig/providerconfig.yaml"
	clientsGo         = "internal/clients/clients.go"
	configLine        = "ps.Configuration = creds"
	startupWatchdog   = 15 * time.Second
	prechecked        = "SafeStart precheck failed"
	startedMark       = "Starting Controller"
	fakeKubeconfig    = `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:1
    insecure-skip-tls-verify: true
  name: fake
contexts:
- context:
    cluster: fake
    user: fake
  name: fake
current-context: fake
users:
- name: fake
  user:
    token: fake-token
`
)

var (
	tfCLIVersion   = regexp.MustCompile(`(?m)^export TERRAFORM_VERSION\s*\?*=\s*(\S+)$`)
	upjetResources = map[string]string{probeKind: probeResource, configMapKind: configMapResource}
	upjetSeams     = []string{
		clientsGo, "internal/clients/user.go", "internal/clients/user_test.go",
		probeConfig, "config/configmap/config.go", "examples/core/configmap.yaml",
	}
	// upjetOnly names the stages upjet adds to the ones both flavors declare.
	upjetOnly = []string{"tf-bump", "provider-starts"}
)

var upjetFlavor = &Flavor{
	Name:   upjetName,
	Stages: upjetStages,
	Init: slices.Concat(commonInit,
		[]string{"--upjet", "--terraform-provider=" + tfProvider, "--terraform-provider-version=" + tfFrom}),
	CreateAPI: upjetCreateAPI,
	Kinds:     []string{probeKind, configMapKind},
	ProbeKind: probeKind,
	TestKind:  configMapKind,
	ToolProbe: configZZ,
	UserProbe: probeConfig,
	AdoptFile: configProvider,
	StockMsg:  userLogicMsg,
	ToolOwned: []string{
		configProvider, configZZ, makeFragment, ownershipDoc, providerMain, "internal/clients/resolve.go",
	},
	UserOwned: []string{
		makefile, agentsDoc, probeConfig, "config/configmap/config.go", clientsGo,
		"apis/namespaced/v1beta1/types.go", providerConfigYML,
	},
	RequiredOwned: []string{configProvider, configZZ, makeFragment},
	Seams:         upjetSeams,
	AssertUserLogic: func(e *Env) error {
		_, err := e.exec(e.Dir, "", "go", "test", "-count=1", "./internal/clients/")
		return err
	},
	Live: Live{
		Managed:      "configmap.core.example.m.com/example",
		ManagedNS:    crossplaneNS,
		Example:      "examples/core/configmap.yaml",
		ExternalName: "example",
		ExternalKey:  "hello",
		Created:      "world",
		Updated:      "updated-value",
		SpecPatch: func(value string) string {
			return fmt.Sprintf(`{"spec":{"forProvider":{"data":{"hello":%q}}}}`, value)
		},
		ProviderConfig: func(e *Env) error {
			_, err := e.kubectl("", "apply", "-f", providerConfigYML)
			return err
		},
		Installed: assertControllersStarted,
	},
}

func upjetCreateAPI(kind string) []string {
	resource, ok := upjetResources[kind]
	if !ok {
		resource = probeResource
	}
	return []string{
		"create", "api", "--group=" + upjetGroup, "--version=" + apiVersion, "--kind=" + kind,
		"--terraform-resource=" + resource,
	}
}

func upjetStages() []Stage {
	docker := []Capability{CapDocker}
	return []Stage{
		{Name: cmdInit, Run: stageUpjetInit},
		{Name: "create-api", Run: stageUpjetCreateAPI},
		{Name: "generate", Run: stageUpjetGenerate},
		{Name: "user-logic", Run: stageUpjetUserLogic},
		{Name: "update-to-head", When: Case.Release, Run: stageUpdateToHead},
		{Name: "refuse-old-generator", When: Case.Release, Run: stageRefuseOldGenerator},
		{Name: "update-noop", Run: stageUpdateNoop},
		{Name: "orphan-removal", Run: stageOrphanRemoval},
		{Name: "create-api-refresh", Run: stageCreateAPIRefresh},
		{Name: "adopt", Run: stageAdopt},
		{Name: cmdCreateTest, Run: stageCreateTest},
		{Name: "tf-bump", Run: stageTerraformBump},
		{Name: "provider-starts", Run: stageProviderStarts},
		{Name: "stock-e2e", Needs: docker, When: Case.Head, Run: stageStockE2E},
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

func stageUpjetInit(e *Env) error {
	if err := e.scaffold(upjetBaseLayout); err != nil {
		return err
	}
	makefile, err := e.read(makefile)
	if err != nil {
		return err
	}
	if err := contains(makefile, tfProvider, "the Makefile"); err != nil {
		return err
	}
	return e.assertCleanTree("init")
}

func stageUpjetCreateAPI(e *Env) error {
	for _, kind := range e.F.Kinds {
		if _, err := e.gen(e.F.CreateAPI(kind)...); err != nil {
			return err
		}
		lower := strings.ToLower(kind)
		if err := e.assertLayout([]string{"config/" + lower + "/config.go"}); err != nil {
			return err
		}
		resources, err := e.read(e.F.ToolProbe)
		if err != nil {
			return err
		}
		for _, want := range []string{lower + ".Configure", lower + ".TerraformResource"} {
			if err := contains(resources, want, e.F.ToolProbe); err != nil {
				return err
			}
		}
	}
	return nil
}

// stageUpjetGenerate runs upjet's real pipeline: it downloads Terraform, reads
// the provider schema and scrapes its docs, then writes the types, controllers
// and CRDs. Unlike native's, its output is not part of the scaffold, so the
// stage commits it for the stages that update the tree. The first generate
// leaves go.mod listing as indirect a module the generated code imports, which
// the next generate corrects, so the stage tidies first: otherwise a
// same-generator update would change go.mod.
func stageUpjetGenerate(e *Env) error {
	subjects, err := e.git("log", "--format=%s")
	if err != nil {
		return err
	}
	if subjects != "Initial commit" {
		return fmt.Errorf("want one Initial commit before generate, git log has:\n%s", subjects)
	}
	if err := e.make("submodules", targetGenerate); err != nil {
		return err
	}
	if err := e.assertUpjetGenerated(); err != nil {
		return err
	}
	if _, err := e.exec(e.Dir, "", "go", "mod", "tidy"); err != nil {
		return err
	}
	if _, err := e.exec(e.Dir, "", "go", "build", "./..."); err != nil {
		return err
	}
	if err := e.assertOwnership(); err != nil {
		return err
	}
	return e.commit("Generate provider")
}

func (e *Env) assertUpjetGenerated() error {
	want := []string{"apis/cluster/zz_register.go", "internal/controller/cluster/zz_setup.go"}
	for _, kind := range e.F.Kinds {
		lower := strings.ToLower(kind)
		want = append(want,
			fmt.Sprintf("apis/cluster/%s/%s/zz_%s_types.go", upjetGroup, apiVersion, lower),
			fmt.Sprintf("internal/controller/cluster/%s/%s/zz_controller.go", upjetGroup, lower))
		crds, err := filepath.Glob(e.path(fmt.Sprintf("package/crds/*%ss.yaml", lower)))
		if err != nil || len(crds) == 0 {
			return fmt.Errorf("generate wrote no CRD for %s: %w", kind, err)
		}
	}
	if err := e.assertLayout(want); err != nil {
		return err
	}
	types, err := e.read(want[2])
	if err != nil {
		return err
	}
	return contains(types, "kubernetes", want[2]+" (the types should reflect the provider schema)")
}

func stageUpjetUserLogic(e *Env) error {
	if err := e.copyOverlay(); err != nil {
		return err
	}
	if err := e.wireUserDefaults(); err != nil {
		return err
	}
	if err := e.F.AssertUserLogic(e); err != nil {
		return err
	}
	if _, err := e.exec(e.Dir, "", "go", "build", "./..."); err != nil {
		return err
	}
	return e.commit(userLogicMsg)
}

// wireUserDefaults makes the scaffold's own clients.go call the author's
// function, so the deployed provider runs the author's code.
func (e *Env) wireUserDefaults() error {
	src, err := e.read(clientsGo)
	if err != nil {
		return err
	}
	if strings.Count(src, configLine) != 1 {
		return fmt.Errorf("%s has no single %q to wire the user defaults into", clientsGo, configLine)
	}
	return e.write(clientsGo, []byte(strings.Replace(src, configLine, "ps.Configuration = withUserDefaults(creds)", 1)))
}

// stageTerraformBump moves the wrapped Terraform provider from tfFrom to tfTo
// with update. It makes two tool-owned files stale and deletes one user-owned
// file first, so the stage fails unless update refreshes the first two and
// does not re-seed the third. The one user-owned file update may rewrite is the
// Makefile, and only by replacing the version.
func stageTerraformBump(e *Env) error {
	before, err := e.userOwned()
	if err != nil {
		return err
	}
	makefile, err := e.read(makefile)
	if err != nil {
		return err
	}
	const mark = "# tf-bump: stale tool-owned content"
	for _, rel := range []string{configProvider, makeFragment} {
		if err := e.appendLine(rel, mark); err != nil {
			return err
		}
	}
	if err := os.Remove(e.path(providerConfigYML)); err != nil {
		return err
	}
	if err := e.commit("make tool-owned files stale and delete the ProviderConfig example"); err != nil {
		return err
	}
	out, err := e.head("update", "--terraform-provider-version="+tfTo)
	if err != nil {
		return err
	}
	if err := contains(out, "removed 0", "update --terraform-provider-version"); err != nil {
		return err
	}
	if err := e.assertBumped(makefile); err != nil {
		return err
	}
	if err := e.assertCachesVersioned(); err != nil {
		return err
	}
	if err := e.assertRefreshedNotReseeded(mark, out); err != nil {
		return err
	}
	delete(before, providerConfigYML)
	if err := e.assertUserOwnedUnchanged(before, makefile); err != nil {
		return err
	}
	if err := e.assertToolOwnedMatchesFreshScaffold(); err != nil {
		return err
	}
	if _, err := e.git("checkout", "HEAD~1", "--", providerConfigYML); err != nil {
		return err
	}
	return e.commit("update provider")
}

// assertBumped requires the Makefile to be the old one with the version
// replaced and nothing else, and to pin the new provider binary.
func (e *Env) assertBumped(old string) error {
	now, err := e.read(makefile)
	if err != nil {
		return err
	}
	if want := strings.ReplaceAll(old, tfFrom, tfTo); now != want {
		return fmt.Errorf("update did more to the Makefile than replace %s with %s", tfFrom, tfTo)
	}
	if strings.Contains(now, tfFrom) || strings.Count(now, tfTo) != 2 {
		return fmt.Errorf("the Makefile pins %s %d times and %s still appears: %t, want 2 and false",
			tfTo, strings.Count(now, tfTo), tfFrom, strings.Contains(now, tfFrom))
	}
	return contains(now, "TERRAFORM_NATIVE_PROVIDER_BINARY ?= terraform-provider-kubernetes_v"+tfTo+"_x5", "the Makefile")
}

// assertCachesVersioned requires both versions' docs checkouts and the new
// Terraform workdir, so a bump can neither reuse the old docs nor wipe them.
func (e *Env) assertCachesVersioned() error {
	docs := ".work/" + tfProvider
	for _, rel := range []string{docs + "@" + tfFrom, docs + "@" + tfTo, ".work/terraform@" + tfTo} {
		if _, err := os.Stat(e.path(rel)); err != nil {
			return fmt.Errorf("the versioned cache is missing: %w", err)
		}
	}
	if _, err := os.Stat(e.path(docs)); err == nil {
		return fmt.Errorf("%s, the unversioned docs cache, is still in use", docs)
	}
	tag, err := e.git("-C", docs+"@"+tfTo, "describe", "--tags")
	if err != nil {
		return err
	}
	if tag != "v"+tfTo {
		return fmt.Errorf("the new docs checkout is at %q, want v%s", tag, tfTo)
	}
	return nil
}

func (e *Env) assertRefreshedNotReseeded(mark, out string) error {
	for _, rel := range []string{configProvider, makeFragment} {
		s, err := e.read(rel)
		if err != nil {
			return err
		}
		if strings.Contains(s, mark) {
			return fmt.Errorf("update did not refresh the tool-owned %s", rel)
		}
	}
	if _, err := os.Stat(e.path(providerConfigYML)); err == nil {
		return fmt.Errorf("update re-seeded the user-owned %s", providerConfigYML)
	}
	return contains(out, "Not seeded", "update's report of the deleted "+providerConfigYML)
}

// stageProviderStarts proves the generated provider runs, not only builds: it
// builds through the scaffold's own make and prints its flags, then registers
// its scheme against an API server that is not there and must reach its
// startup precheck without a panic. The stages that deploy it show the
// controllers start against a real API server.
func stageProviderStarts(e *Env) error {
	if err := e.make("go.build"); err != nil {
		return err
	}
	built, err := filepath.Glob(e.path("_output/bin/*/provider"))
	if err != nil || len(built) == 0 {
		return fmt.Errorf("make go.build produced no provider binary: %w", err)
	}
	if _, err := e.exec(e.Dir, "", built[0], "--help"); err != nil {
		return err
	}
	mk, err := e.read(makeFragment)
	if err != nil {
		return err
	}
	m := tfCLIVersion.FindStringSubmatch(mk)
	if m == nil {
		return fmt.Errorf("hack/xp-provider-gen.mk pins no TERRAFORM_VERSION")
	}
	kubeconfig := filepath.Join(e.Work, "unreachable.kubeconfig")
	if err := os.WriteFile(kubeconfig, []byte(fakeKubeconfig), 0o600); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(e.Ctx, startupWatchdog)
	defer cancel()
	timed := *e
	timed.Ctx = ctx
	out, _ := timed.execEnv([]string{"KUBECONFIG=" + kubeconfig}, e.Dir, "", built[0],
		"--terraform-version="+m[1], "--terraform-provider-source="+tfProvider,
		"--terraform-provider-version="+tfTo, "--debug")
	if strings.Contains(out, "panic:") || strings.Contains(out, "runtime error") {
		return fmt.Errorf("the provider panicked registering its scheme:\n%s", out)
	}
	return contains(out, prechecked, "the provider's startup against an unreachable API server")
}

// assertControllersStarted waits for the deployed provider to log that it
// started a controller for every kind it generated and for ProviderConfig. A
// provider can be Healthy while a generated controller never registers, and
// only the kind the cluster stages reconcile would show it.
func assertControllersStarted(e *Env) error {
	return e.eventually("controllers not yet started", func() (string, error) {
		logs, err := e.kubectl("", "-n", crossplaneNS, "logs", "-l", providerLabel+"="+provider,
			"--all-containers", "--tail=-1")
		if err != nil {
			return "", err
		}
		return strings.Join(unstartedControllers(logs, e.F.Kinds), ","), nil
	}, "")
}

// unstartedControllers names the kinds, and ProviderConfig, that no
// "Starting Controller" line of the provider's log mentions.
func unstartedControllers(logs string, kinds []string) []string {
	var missing []string
	for _, name := range append(slices.Clone(kinds), "ProviderConfig") {
		started := slices.ContainsFunc(strings.Split(logs, "\n"), func(line string) bool {
			return strings.Contains(line, startedMark) && strings.Contains(strings.ToLower(line), strings.ToLower(name))
		})
		if !started {
			missing = append(missing, name)
		}
	}
	return missing
}
