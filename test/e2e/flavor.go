//go:build !windows

package main

import "fmt"

// Live is what the cluster stages know about the managed resource a flavor
// deploys and the external object behind it.
type Live struct {
	Managed      string // the managed resource, as kubectl names it
	ManagedNS    string
	Example      string // the manifest that creates it
	ExternalName string // a ConfigMap in externalNS that stands for the external system
	ExternalKey  string // the data key of that ConfigMap the managed resource sets
	Created      string
	Updated      string
	// SpecPatch is a merge patch that makes the managed resource ask for value.
	SpecPatch func(value string) string
	// ProviderConfig readies the cluster for the deployed provider.
	ProviderConfig func(*Env) error
	// Installed is checked once the deployed provider is Healthy; nil checks nothing more.
	Installed func(*Env) error
}

// Flavor is everything the shared stages read that differs between flavors.
type Flavor struct {
	Name string
	// Scaffold, CreateAPIStage, Generate and UserLogic are the stages of the
	// shared list whose work differs between flavors; the rest read the fields
	// below. Extra are the stages one flavor adds after create-test.
	Scaffold, CreateAPIStage, Generate, UserLogic func(*Env) error
	Extra                                         []Stage
	// Init and CreateAPI are the arguments the scaffold is made with.
	Init      []string
	CreateAPI func(kind string) []string
	Kinds     []string
	// ProbeKind is the kind create api is re-run for and refused on; TestKind is
	// the one create-test and the cluster stages use.
	ProbeKind, TestKind string
	// ToolProbe is a tool-owned file create api rewrites, UserProbe a user-owned
	// one it leaves alone, and AdoptFile a tool-owned one whose header adopt
	// restores.
	ToolProbe, UserProbe, AdoptFile string
	// StockMsg is the subject of the commit stock-e2e runs on.
	StockMsg string
	// ToolOwned must carry the generated header and UserOwned must not. Every
	// ToolOwned file must also be in the tool-owned set of a fresh scaffold.
	ToolOwned, UserOwned []string
	// Seams are user-owned files a same-generator update must leave alone.
	Seams []string
	// RefusesTerraformVersion is whether update --terraform-provider-version is
	// rejected for this flavor.
	RefusesTerraformVersion bool
	// AssertUserLogic is what must still hold of the author's code after the
	// overlay is applied and after an update.
	AssertUserLogic func(*Env) error
	Live            Live
}

func flavorFor(name string) (*Flavor, error) {
	switch name {
	case nativeName:
		return nativeFlavor, nil
	case upjetName:
		return upjetFlavor, nil
	}
	return nil, fmt.Errorf("no stages declared for flavor %q", name)
}

// Stages is the one declared list both flavors run, in plan order.
func (f *Flavor) Stages() []Stage {
	docker := []Capability{CapDocker}
	stages := []Stage{
		{Name: cmdInit, Run: f.Scaffold},
		{Name: "create-api", Run: f.CreateAPIStage},
		{Name: "generate", Run: f.Generate},
		{Name: "user-logic", Run: f.UserLogic},
		// stock-e2e works on a clone made before it starts, so the update stages
		// that run beside it share nothing with it but the cluster name:
		// install-in-kind joins first, and stock-e2e's e2e-clean deletes that
		// cluster, so install-in-kind starts on none.
		{Name: "stock-clone", Needs: docker, When: Case.Head, Run: stageStockClone},
		{Name: "stock-e2e", Needs: docker, When: Case.Head, Background: true, Run: stageStockE2E},
		{Name: "update-to-head", When: Case.Release, Run: stageUpdateToHead},
		{Name: "refuse-old-generator", When: Case.Release, Run: stageRefuseOldGenerator},
		{Name: "update-noop", Run: stageUpdateNoop},
		{Name: "orphan-removal", Run: stageOrphanRemoval},
		{Name: "create-api-refresh", Run: stageCreateAPIRefresh},
		{Name: "adopt", Run: stageAdopt},
		{Name: cmdCreateTest, Run: stageCreateTest},
	}
	stages = append(stages, f.Extra...)
	return append(stages,
		Stage{Name: "build-xpkg", Needs: docker, Run: func(e *Env) error { return e.make("build") }},
		Stage{Name: "install-in-kind", Needs: docker, Join: true, Run: stageInstallInKind},
		Stage{Name: "provider-config", Needs: docker, Run: func(e *Env) error { return e.F.Live.ProviderConfig(e) }},
		Stage{Name: "mr-ready-synced", Needs: docker, Run: stageReadySynced},
		Stage{Name: "mr-update", Needs: docker, Run: stageMRUpdate},
		Stage{Name: "drift-corrected", Needs: docker, Run: stageDriftCorrected},
		Stage{Name: "mr-delete", Needs: docker, Run: stageMRDelete},
		Stage{Name: "external-gone", Needs: docker, Run: stageExternalGone},
	)
}
