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

package templates

import (
	"path/filepath"
)

// GoModTemplateProduct implements go.mod template
type GoModTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *GoModTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = "go.mod"
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("root/go.mod.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = goModTemplate
	}
	return nil
}

const goModTemplate = `module {{ .Repo }}

go 1.24.0

toolchain go1.24.5

tool sigs.k8s.io/controller-tools/cmd/controller-gen

tool github.com/crossplane/crossplane-tools/cmd/angryjet

require (
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/crossplane/crossplane-runtime/v2 v2.0.0
	github.com/google/go-cmp v0.7.0
	github.com/pkg/errors v0.9.1
	google.golang.org/grpc v1.74.2
	k8s.io/apimachinery v0.33.3
	k8s.io/client-go v0.33.3
	sigs.k8s.io/controller-runtime v0.21.0
)`

// MakefileTemplateProduct implements Makefile template
type MakefileTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *MakefileTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = "Makefile"
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("root/Makefile.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = makefileTemplate
	}
	return nil
}

const makefileTemplate = `# ====================================================================================
# Setup Project
PROJECT_NAME := {{ .ProviderName }}
PROJECT_REPO := {{ .Repo }}

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
GOLANGCILINT_VERSION = 2.1.2
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Images

IMAGES = {{ .ProviderName }}
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup XPKG

XPKG_REG_ORGS ?= xpkg.upbound.io/crossplane
# NOTE(hasheddan): skip promoting on xpkg.upbound.io as channel tags are
# inferred.
XPKG_REG_ORGS_NO_PROMOTE ?= xpkg.upbound.io/crossplane
XPKGS = {{ .ProviderName }}
-include build/makelib/xpkg.mk

# NOTE(hasheddan): we force image building to happen prior to xpkg build so that
# we ensure image is present in daemon.
xpkg.build.{{ .ProviderName }}: do.build.images

# ====================================================================================
# Targets

# NOTE: We rely on the default reviewable target from build/makelib/common.mk

# Update dependencies and setup build environment
submodules: $(SUBMODULES)
	@echo "Setting up build environment..."
	@go mod tidy
	@echo "Build environment setup complete."

# NOTE: We rely on the default generate target from build/makelib/golang.mk

# We want submodules to be set up the first time ` + "`make`" + ` is run.
# We manage the build/ folder and its Makefiles as a submodule.
# The first time ` + "`make`" + ` is run, the includes of build/*.mk files will
# all fail, and this target will be run. The next time, the default as defined
# by the includes will be run instead.
fallback: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/provider --debug

dev: $(KIND) $(KUBECTL)
	@$(INFO) Creating kind cluster
	@$(KIND) create cluster --name=$(PROJECT_NAME)-dev
	@$(KUBECTL) cluster-info --context kind-$(PROJECT_NAME)-dev
	@$(INFO) Installing Provider CRDs
	@$(KUBECTL) apply -R -f package/crds
	@$(INFO) Starting Provider controllers
	@$(GO) run cmd/provider/main.go --debug

dev-clean: $(KIND) $(KUBECTL)
	@$(INFO) Deleting kind cluster
	@$(KIND) delete cluster --name=$(PROJECT_NAME)-dev

.PHONY: submodules fallback run dev dev-clean

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run crossplane locally, out-of-cluster. Useful for development.
    dev                   Create kind cluster and run provider with CRDs.
    dev-clean             Clean up development kind cluster.

endef
# The reason CROSSPLANE_MAKE_HELP is used instead of CROSSPLANE_HELP is because the crossplane
# binary will try to use CROSSPLANE_HELP if it is set, and this is for something different.
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special`

// READMETemplateProduct implements README.md template
type READMETemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *READMETemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = "README.md"
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("root/README.md.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = readmeTemplate
	}
	return nil
}

const readmeTemplate = `# {{ .ProviderName }}

A Crossplane provider for managing cloud resources.

## Getting Started

### Install Provider

Apply the provider to your Crossplane cluster:

` + "```" + `bash
kubectl apply -f package/provider.yaml
` + "```" + `

### Configure Provider

Create a ProviderConfig:

` + "```" + `yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: {{ .ProviderName }}
spec:
  package: {{ .Repo }}
` + "```" + `

## Development

### Build

` + "```" + `bash
make build
` + "```" + `

### Test

` + "```" + `bash
make test
` + "```" + `

### Generate

` + "```" + `bash
make generate
` + "```" + `

## Contributing

We welcome contributions! Please see our [contributing guide](CONTRIBUTING.md) for more information.`

// GitIgnoreTemplateProduct implements .gitignore template
type GitIgnoreTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *GitIgnoreTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = ".gitignore"
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("root/.gitignore.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = gitIgnoreTemplate
	}
	return nil
}

const gitIgnoreTemplate = `/.cache
/.work
/_output
cover.out
/vendor
/.vendor-new
.vscode
.idea
.DS_Store`

// MainGoTemplateProduct implements cmd/provider/main.go template
type MainGoTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *MainGoTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("cmd", "provider", "main.go")
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("cmd/provider/main.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = mainGoTemplate
	}
	return nil
}

const mainGoTemplate = `{{ .Boilerplate }}

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	changelogsv1alpha1 "github.com/crossplane/crossplane-runtime/v2/apis/changelogs/proto/v1alpha1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"

	"{{ .Repo }}/apis"
	providercontroller "{{ .Repo }}/internal/controller"
	"{{ .Repo }}/internal/version"
)

func main() {
	var (
		app            = kingpin.New(filepath.Base(os.Args[0]), "{{ .ProviderName }} support for Crossplane.").DefaultEnvars()
		debug          = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		leaderElection = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").Envar("LEADER_ELECTION").Bool()

		syncInterval            = app.Flag("sync", "How often all resources will be double-checked for drift from the desired state.").Short('s').Default("1h").Duration()
		pollInterval            = app.Flag("poll", "How often individual resources will be checked for drift from the desired state").Default("1m").Duration()
		pollStateMetricInterval = app.Flag("poll-state-metric", "State metric recording interval").Default("5s").Duration()

		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()

		enableManagementPolicies = app.Flag("enable-management-policies", "Enable support for Management Policies.").Default("true").Envar("ENABLE_MANAGEMENT_POLICIES").Bool()
		enableChangeLogs         = app.Flag("enable-changelogs", "Enable support for capturing change logs during reconciliation.").Default("false").Envar("ENABLE_CHANGE_LOGS").Bool()
		changelogsSocketPath     = app.Flag("changelogs-socket-path", "Path for changelogs socket (if enabled)").Default("/var/run/changelogs/changelogs.sock").Envar("CHANGELOGS_SOCKET_PATH").String()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("{{ .ProviderName }}"))
	if *debug {
		// The controller-runtime is *very* verbose even at info level, so we only
		// provide it a real logger when we're running in debug mode.
		ctrl.SetLogger(zl)
	} else {
		// Setting the controller-runtime logger to a no-op logger by default. This
		// is not really needed, but otherwise we get a warning from the
		// controller-runtime.
		ctrl.SetLogger(zap.New(zap.WriteTo(io.Discard)))
	}

	cfg, err := ctrl.GetConfig()
	kingpin.FatalIfError(err, "Cannot get API server rest config")

	mgr, err := ctrl.NewManager(ratelimiter.LimitRESTConfig(cfg, *maxReconcileRate), ctrl.Options{
		// SyncPeriod in ctrl.Options has been removed since controller-runtime v0.16.0
		// The recommended way is to move it to cache.Options instead
		Cache: cache.Options{
			SyncPeriod: syncInterval,
		},

		// controller-runtime uses both ConfigMaps and Leases for leader
		// election by default. Leases expire after 15 seconds, with a
		// 10 seconds renewal deadline. We've observed leader loss due to
		// renewal deadlines being exceeded when under high load - i.e.
		// hundreds of reconciles per second and ~200rps to the API
		// server. Switching to Leases only and longer leases appears to
		// alleviate this.
		LeaderElection:             *leaderElection,
		LeaderElectionID:           "crossplane-leader-election-{{ .ProviderName }}",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		LeaseDuration:              func() *time.Duration { d := 60 * time.Second; return &d }(),
		RenewDeadline:              func() *time.Duration { d := 50 * time.Second; return &d }(),
	})
	kingpin.FatalIfError(err, "Cannot create controller manager")
	kingpin.FatalIfError(apis.AddToScheme(mgr.GetScheme()), "Cannot add {{ .ProviderName }} APIs to scheme")

	metricRecorder := managed.NewMRMetricRecorder()
	stateMetrics := statemetrics.NewMRStateMetrics()

	metrics.Registry.MustRegister(metricRecorder)
	metrics.Registry.MustRegister(stateMetrics)

	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
		Features:                &feature.Flags{},
		MetricOptions: &controller.MetricOptions{
			PollStateMetricInterval: *pollStateMetricInterval,
			MRMetrics:               metricRecorder,
			MRStateMetrics:          stateMetrics,
		},
	}

	if *enableManagementPolicies {
		o.Features.Enable(feature.EnableBetaManagementPolicies)
		log.Info("Beta feature enabled", "flag", feature.EnableBetaManagementPolicies)
	}

	if *enableChangeLogs {
		o.Features.Enable(feature.EnableAlphaChangeLogs)
		log.Info("Alpha feature enabled", "flag", feature.EnableAlphaChangeLogs)

		conn, err := grpc.NewClient("unix://"+*changelogsSocketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
		kingpin.FatalIfError(err, "failed to create change logs client connection at %s", *changelogsSocketPath)

		clo := controller.ChangeLogOptions{
			ChangeLogger: managed.NewGRPCChangeLogger(
				changelogsv1alpha1.NewChangeLogServiceClient(conn),
				managed.WithProviderVersion(fmt.Sprintf("{{ .ProviderName }}:%s", version.Version))),
		}
		o.ChangeLogOptions = &clo
	}

	kingpin.FatalIfError(providercontroller.Setup(mgr, o), "Cannot setup {{ .ProviderName }} controllers")
	kingpin.FatalIfError(mgr.Start(ctrl.SetupSignalHandler()), "Cannot start controller manager")
}`

// APIsTemplateProduct implements apis/apis.go
type APIsTemplateProduct struct {
	*BaseTemplateProduct
	loader *TemplateLoader
}

func (t *APIsTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("apis", "register.go")
	}

	// Try to load from file first, fallback to embedded constant
	if t.loader == nil {
		t.loader = NewTemplateLoader()
	}

	if templateContent, err := t.loader.LoadTemplate("apis/register.go.tmpl"); err == nil {
		t.TemplateBody = templateContent
	} else {
		// Fallback to embedded template for backward compatibility
		t.TemplateBody = apisTemplate
	}
	return nil
}

const apisTemplate = `{{ .Boilerplate }}

// Package apis contains Kubernetes API for the {{ .ProviderName }} provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	v1alpha1 "{{ .Repo }}/apis/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1alpha1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
`

// GenerateGoTemplateProduct implements apis/generate.go
type GenerateGoTemplateProduct struct {
	*BaseTemplateProduct
}

func (t *GenerateGoTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("apis", "generate.go")
	}
	t.TemplateBody = generateGoTemplate
	return nil
}

const generateGoTemplate = `{{ .Boilerplate }}

// NOTE: See the below link for details on what is happening here.
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

// Remove existing CRDs
//go:generate rm -rf ../package/crds

// Generate deepcopy methodsets and CRD manifests
//go:generate go run -tags generate sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../hack/boilerplate.go.txt paths=./... crd:crdVersions=v1 output:artifacts:config=../package/crds

// Generate crossplane-runtime methodsets (resource.Claim, etc)
//go:generate go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet generate-methodsets --header-file=../hack/boilerplate.go.txt ./...

package apis
`

// BoilerplateTemplateProduct implements hack/boilerplate.go.txt
type BoilerplateTemplateProduct struct {
	*BaseTemplateProduct
}

func (t *BoilerplateTemplateProduct) SetTemplateDefaults() error {
	if t.Path == "" {
		t.Path = filepath.Join("hack", "boilerplate.go.txt")
	}
	t.TemplateBody = boilerplateTemplate
	return nil
}

const boilerplateTemplate = `// SPDX-FileCopyrightText: 2025 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0
`