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

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Main{}

// Main scaffolds the main.go file for a Crossplane provider
type Main struct {
	machinery.TemplateMixin
	machinery.BoilerplateMixin
	machinery.RepositoryMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Main) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("cmd", "provider", "main.go")
	}

	f.TemplateBody = mainTemplate

	return nil
}

const mainTemplate = `{{ .Boilerplate }}

package main

import (
	"os"

	"github.com/alecthomas/kingpin/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"

	"{{ .Repo }}/apis"
	"{{ .Repo }}/internal/controller/config"
	"{{ .Repo }}/internal/version"
)

func main() {
	var (
		app          = kingpin.New("provider", "Crossplane provider.").DefaultEnvars()
		debug        = app.Flag("debug", "Run with debug logging.").Short('d').Bool()
		pollInterval = app.Flag("poll", "How often individual resources will be checked for drift from the desired state").Default("1m").Duration()
		maxReconcileRate = app.Flag("max-reconcile-rate", "The global maximum rate per second at which resources may checked for drift from the desired state.").Default("10").Int()
		leaderElection = app.Flag("leader-election", "Use leader election for the controller manager.").Short('l').Default("false").Bool()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	zl := zap.New(zap.UseDevMode(*debug))
	log := logging.NewLogrLogger(zl.WithName("provider"))
	ctrl.SetLogger(zl)

	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Info("Cannot get API server rest config", "error", err)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ratelimiter.LimitRESTConfig(cfg, *maxReconcileRate), ctrl.Options{
		LeaderElection:   *leaderElection,
		LeaderElectionID: "crossplane-leader-election-provider",
	})
	if err != nil {
		log.Info("Cannot create controller manager", "error", err)
		os.Exit(1)
	}

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Info("Cannot add APIs to scheme", "error", err)
		os.Exit(1)
	}

	o := controller.Options{
		Logger:                  log,
		MaxConcurrentReconciles: *maxReconcileRate,
		PollInterval:            *pollInterval,
		GlobalRateLimiter:       ratelimiter.NewGlobal(*maxReconcileRate),
		Features:                &feature.Flags{},
	}

	if err := config.Setup(mgr, o); err != nil {
		log.Info("Cannot setup ProviderConfig controller", "error", err)
		os.Exit(1)
	}

	// TODO: Setup your managed resource controllers here
	// Example:
	// if err := mytype.Setup(mgr, o); err != nil {
	//     log.Error(err, "Cannot setup MyType controller")
	//     os.Exit(1)
	// }

	log.Info("Starting manager", "version", version.Version)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Info("Cannot start controller manager", "error", err)
		os.Exit(1)
	}
}
`