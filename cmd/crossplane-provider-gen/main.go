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

package main

import (
	"os"

	"sigs.k8s.io/kubebuilder/v4/pkg/cli"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	
	crossplanev1 "github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v1"
)

func main() {
	// Create a simplified CLI that only exposes init and create api
	cli, err := cli.New(
		cli.WithCommandName("crossplane-provider-gen"),
		cli.WithVersion("v1.0.0"),
		cli.WithDescription("Crossplane Provider Generator - A tool for scaffolding Crossplane providers and managed resources"),
		cli.WithDefaultProjectVersion(cfgv3.Version),
		cli.WithPlugins(&crossplanev1.Plugin{}),
		cli.WithDefaultPlugins(cfgv3.Version, &crossplanev1.Plugin{}),
		cli.WithCompletion(),
	)
	if err != nil {
		os.Exit(1)
	}
	if err := cli.Run(); err != nil {
		os.Exit(1)
	}
}