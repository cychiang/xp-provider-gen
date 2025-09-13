package templates

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

func MainGo(cfg config.Config) machinery.Template {
	return SimpleFile(cfg, "cmd/provider/main.go", mainGoTemplate)
}

const mainGoTemplate = `package main

import (
	"context"
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	
	"{{ .Repo }}/internal/controller"
)

func main() {
	// Provider setup code here
	controller.Setup(mgr, controller.Options{})
}`
