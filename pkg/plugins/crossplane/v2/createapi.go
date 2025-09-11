package v2

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"

	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/templates/api"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/templates/controllers"
	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds/templates"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	// Crossplane-specific flags
	GenerateClient bool
	Force          bool

	// Injected dependencies
	config   config.Config
	resource *resource.Resource
	
	// Utility dependencies
	pluginConfig    *PluginConfig
	stringUtils     *StringUtils
	validationUtils *ValidationUtils
	metadataUtils   *MetadataUtils
}

// UpdateMetadata updates the plugin metadata
func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	// Initialize utilities if not already done
	p.ensureUtilities()
	
	subcmdMeta.Description = `Create a new Crossplane managed resource API.

This command scaffolds a new Crossplane managed resource with the necessary
controller implementation using crossplane-runtime patterns. The generated
API will include proper Crossplane annotations, status conditions, and
external client scaffolding.
`
	subcmdMeta.Examples = p.metadataUtils.GetCreateAPIExamples(cliMeta.CommandName)
}

// BindFlags binds the subcommand flags
func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	// Initialize utilities if not already done
	p.ensureUtilities()
	
	help := p.pluginConfig.GetFlagHelp()
	defaults := p.pluginConfig.Defaults
	
	// Crossplane-specific flags using centralized defaults and help text
	fs.BoolVar(&p.GenerateClient, "generate-client", defaults.GenerateClient, help.GenerateClient)
	fs.BoolVar(&p.Force, "force", defaults.Force, help.Force)
}

// InjectConfig injects the project configuration
func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	// Initialize utilities
	p.ensureUtilities()
	
	// No validation needed for simplified provider
	p.config = c
	return nil
}

// InjectResource injects the resource model
func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	// Ensure utilities are initialized
	p.ensureUtilities()

	// Set defaults based on the resource information
	if res != nil {
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

	// Ensure utilities are initialized
	p.ensureUtilities()
	
	// Extract provider name from repository
	var providerName string
	if p.config.GetRepository() != "" {
		parts := strings.Split(p.config.GetRepository(), "/")
		if len(parts) > 0 {
			providerName = parts[len(parts)-1]
		}
	}
	if providerName == "" {
		providerName = "provider-example"
	}
	
	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(p.config),
		machinery.WithBoilerplate(p.pluginConfig.GetBoilerplate()),
		machinery.WithResource(p.resource),
	)

	// Execute the scaffolding templates
	if err := scaffold.Execute(
		&api.CrossplaneGroup{},
		&api.CrossplaneTypes{
			Force: p.Force,
		},
		&controllers.CrossplaneController{
			Force: p.Force,
			RepositoryMixin: machinery.RepositoryMixin{Repo: p.config.GetRepository()},
			DomainMixin: machinery.DomainMixin{Domain: p.config.GetDomain()},
		},
		&templates.TemplateUpdater{
			Force: true, // Always update register.go to include new controller
			RepositoryMixin: machinery.RepositoryMixin{Repo: p.config.GetRepository()},
			ProviderName: providerName,
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

// ensureUtilities initializes utility dependencies if they haven't been created yet
func (p *createAPISubcommand) ensureUtilities() {
	if p.pluginConfig == nil {
		p.pluginConfig = NewPluginConfig()
		p.stringUtils = NewStringUtils()
		p.validationUtils = NewValidationUtils(p.pluginConfig)
		p.metadataUtils = NewMetadataUtils(p.pluginConfig)
	}
}