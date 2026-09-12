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

package engine

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/templates"
)

// ChainsawTestGenerator renders a chainsaw behavior-test skeleton for one kind
// into test/behavior/<name>/chainsaw-test.yaml. The output is user-owned (no
// generated header) and is never overwritten: scaffolding an existing test
// name is an error.
//
// APIVersion, Namespace and Spec are read from the project's own
// examples/<group>/<kind>.yaml by the caller (loadExampleManifest in package
// v2) rather than derived here from Resource — that is what lets one
// template body serve both flavors with no flavor branch: whichever flavor
// produced the example, these three strings are already in its shape.
type ChainsawTestGenerator struct {
	machinery.TemplateMixin

	TestName   string
	Resource   resource.Resource
	APIVersion string
	Namespace  string
	// Spec is the example's spec, re-marshaled to YAML and indented to
	// splice directly under the skeleton's `spec:` key.
	Spec string
}

var _ machinery.Template = &ChainsawTestGenerator{}

// NewChainsawTestGenerator builds the chainsaw skeleton generator.
func NewChainsawTestGenerator(
	testName string, res resource.Resource, apiVersion, namespace, specYAML string,
) *ChainsawTestGenerator {
	return &ChainsawTestGenerator{
		TestName:   testName,
		Resource:   res,
		APIVersion: apiVersion,
		Namespace:  namespace,
		Spec:       specYAML,
	}
}

func (f *ChainsawTestGenerator) SetTemplateDefaults() error {
	f.Path = filepath.Join("test", "behavior", f.TestName, "chainsaw-test.yaml")
	f.IfExistsAction = machinery.Error
	f.TemplateBody = templates.GeneratorBody("chainsaw_test.yaml.tmpl")
	return nil
}
