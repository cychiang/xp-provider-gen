package v1

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"

	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v1/scaffolds/templates/api"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v1/scaffolds/templates/controllers"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	// Crossplane-specific flags
	ExternalName   string
	ProviderType   string
	GenerateClient bool
	Force          bool

	// Injected dependencies
	config   config.Config
	resource *resource.Resource
}

// UpdateMetadata updates the plugin metadata
func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Create a new Crossplane managed resource API.

This command scaffolds a new Crossplane managed resource with the necessary
controller implementation using crossplane-runtime patterns. The generated
API will include proper Crossplane annotations, status conditions, and
external client scaffolding.
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Create a basic managed resource
  %[1]s create api --plugins=%s --group=compute --version=v1alpha1 --kind=Instance

  # Create with custom external name and provider type
  %[1]s create api --plugins=%s --group=storage --version=v1beta1 --kind=Bucket \
    --external-name=s3-bucket --provider-type=aws --generate-client

  # Create a cluster-scoped resource
  %[1]s create api --plugins=%s --group=network --version=v1alpha1 --kind=VPC \
    --namespaced=false
`,
		cliMeta.CommandName, pluginName, pluginName, pluginName)
}

// BindFlags binds the subcommand flags
func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	// Crossplane-specific flags only - core flags are handled by kubebuilder CLI
	fs.StringVar(&p.ExternalName, "external-name", "", "external resource name (defaults to lowercase kind)")
	fs.StringVar(&p.ProviderType, "provider-type", "custom", "provider type (aws, gcp, azure, custom)")
	fs.BoolVar(&p.GenerateClient, "generate-client", true, "generate external client interface")
	fs.BoolVar(&p.Force, "force", false, "overwrite existing files if they exist")
}

// InjectConfig injects the project configuration
func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	// Validate provider type
	validProviderTypes := []string{"aws", "gcp", "azure", "custom"}
	isValid := false
	for _, valid := range validProviderTypes {
		if p.ProviderType == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid provider type: %s (must be one of: %v)", p.ProviderType, validProviderTypes)
	}

	p.config = c
	return nil
}

// InjectResource injects the resource model
func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	// Set defaults based on the resource information
	if res != nil {
		if p.ExternalName == "" {
			p.ExternalName = strings.ToLower(res.Kind)
		}
		
		// Set the resource path using the repository from config
		// This is essential for proper import paths in generated code
		res.Path = resource.APIPackagePath(p.config.GetRepository(), res.Group, res.Version, p.config.IsMultiGroup())
		
		// Mark this as having an API
		res.API = &resource.API{
			CRDVersion: "v1",
			Namespaced: true, // Crossplane managed resources are typically namespaced
		}
		
		// Mark that we have a controller
		res.Controller = true
	}

	return nil
}

// PreScaffold runs before scaffolding
func (p *createAPISubcommand) PreScaffold(machinery.Filesystem) error {
	// TODO: Add validation logic, check if project is initialized, etc.
	return nil
}

// Scaffold scaffolds the managed resource API
func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Creating Crossplane managed resource API %s/%s %s\n", p.resource.Group, p.resource.Version, p.resource.Kind)

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(p.config),
		machinery.WithBoilerplate(`/*
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
*/`),
		machinery.WithResource(p.resource),
	)

	// Execute the scaffolding templates
	if err := scaffold.Execute(
		&api.CrossplaneGroup{
			ProviderDomain: p.ProviderType,
		},
		&api.CrossplaneTypes{
			ProviderType: p.ProviderType,
			ExternalName: p.ExternalName,
			Force:        p.Force,
		},
		&controllers.CrossplaneController{
			ExternalName: p.ExternalName,
			Force:        p.Force,
		},
	); err != nil {
		return fmt.Errorf("error scaffolding Crossplane managed resource: %w", err)
	}

	fmt.Printf("Successfully scaffolded Crossplane managed resource %s\n", p.resource.Kind)
	return nil
}

// PostScaffold runs after scaffolding
func (p *createAPISubcommand) PostScaffold() error {
	// TODO: Add post-scaffolding logic
	fmt.Printf("Crossplane managed resource %s created successfully!\n", p.resource.Kind)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Customize the %sParameters and %sObservation structs\n", p.resource.Kind, p.resource.Kind)
	fmt.Printf("  2. Implement the external client logic\n")
	fmt.Printf("  3. Update controller reconciliation logic\n")
	fmt.Printf("  4. Run 'make manifests' to generate CRDs\n")

	return nil
}