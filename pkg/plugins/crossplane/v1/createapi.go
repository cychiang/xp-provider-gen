package v1

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	// Standard kubebuilder flags
	Group     string
	Version   string
	Kind      string
	Resource  string
	Namespaced bool

	// Crossplane-specific flags
	ExternalName string
	ProviderType string
	GenerateClient bool

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
	// Standard flags
	fs.StringVar(&p.Group, "group", "", "API group name for the resource")
	fs.StringVar(&p.Version, "version", "", "API version for the resource")
	fs.StringVar(&p.Kind, "kind", "", "Kind name for the resource")
	fs.StringVar(&p.Resource, "resource", "", "Resource name (plural of kind, auto-generated if not specified)")
	fs.BoolVar(&p.Namespaced, "namespaced", true, "whether the resource is namespaced")

	// Crossplane-specific flags
	fs.StringVar(&p.ExternalName, "external-name", "", "external resource name (defaults to lowercase kind)")
	fs.StringVar(&p.ProviderType, "provider-type", "custom", "provider type (aws, gcp, azure, custom)")
	fs.BoolVar(&p.GenerateClient, "generate-client", true, "generate external client interface")
}

// InjectConfig injects the project configuration
func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	// Validate required flags
	if p.Group == "" || p.Version == "" || p.Kind == "" {
		return fmt.Errorf("group, version, and kind are required")
	}

	// Set defaults
	if p.Resource == "" {
		p.Resource = strings.ToLower(p.Kind) + "s"
	}
	if p.ExternalName == "" {
		p.ExternalName = strings.ToLower(p.Kind)
	}

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

	// Store API metadata for scaffolding (note: we'll use the resource injection instead)
	// Config in kubebuilder v4 is more structured, so we store these as fields
	p.config = c

	return nil
}

// InjectResource injects the resource model
func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	// Update our flags from the resource if they were set via resource
	if res != nil {
		if p.Group == "" {
			p.Group = res.Group
		}
		if p.Version == "" {
			p.Version = res.Version
		}
		if p.Kind == "" {
			p.Kind = res.Kind
		}
		if p.Resource == "" {
			p.Resource = res.Plural
		}
		// Note: Kubebuilder v4 doesn't have Namespaced in resource model
		// We keep our own flag for Crossplane-specific logic
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
	// TODO: Implement Crossplane managed resource scaffolding
	// This should generate:
	// - apis/<group>/<version>/<kind>_types.go with Crossplane patterns
	// - apis/<group>/<version>/doc.go
	// - apis/<group>/<version>/groupversion_info.go  
	// - internal/controller/<group>/<kind>.go with crossplane-runtime patterns
	// - internal/clients/<group>/<kind>.go (if generateClient is true)
	// - examples/<group>/<kind>.yaml

	fmt.Printf("Creating Crossplane managed resource API %s/%s %s\n", p.Group, p.Version, p.Kind)
	fmt.Printf("External name: %s\n", p.ExternalName)
	fmt.Printf("Provider type: %s\n", p.ProviderType)
	fmt.Printf("Generate client: %t\n", p.GenerateClient)
	fmt.Println("TODO: Implement Crossplane managed resource scaffolding")

	return nil
}

// PostScaffold runs after scaffolding
func (p *createAPISubcommand) PostScaffold() error {
	// TODO: Add post-scaffolding logic
	fmt.Printf("Crossplane managed resource %s created successfully!\n", p.Kind)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Customize the %sParameters and %sObservation structs\n", p.Kind, p.Kind)
	fmt.Printf("  2. Implement the external client logic\n")
	fmt.Printf("  3. Update controller reconciliation logic\n")
	fmt.Printf("  4. Run 'make manifests' to generate CRDs\n")

	return nil
}