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
	Name   string
	Stages func() []Stage
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
	// StockMsg is the subject of the commit stock-e2e runs on; empty means the
	// first commit.
	StockMsg string
	// ToolOwned must carry the generated header and UserOwned must not.
	ToolOwned, UserOwned []string
	// RequiredOwned must be in the tool-owned set of a fresh scaffold.
	RequiredOwned []string
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
