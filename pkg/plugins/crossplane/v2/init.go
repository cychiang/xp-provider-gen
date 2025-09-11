package v2

import (
	"fmt"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"

	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v2/scaffolds"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	// Standard kubebuilder fields
	config config.Config
	
	// Standard init flags
	domain string
	repo   string
	owner  string
	
	// Utility dependencies
	pluginConfig *PluginConfig
	gitUtils     *GitUtils
	stringUtils  *StringUtils
	metadataUtils *MetadataUtils
}

// UpdateMetadata updates the plugin metadata
func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	// Initialize utilities if not already done
	p.ensureUtilities()
	
	subcmdMeta.Description = `Initialize a new Crossplane provider project.

This command initializes a new Crossplane provider project with the necessary
scaffolding to develop and build a Kubernetes controller to manage external
resources following Crossplane patterns.
`
	subcmdMeta.Examples = p.metadataUtils.GetInitExamples(cliMeta.CommandName)
}

// BindFlags binds the subcommand flags
func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	// Initialize utilities if not already done
	p.ensureUtilities()
	
	help := p.pluginConfig.GetFlagHelp()
	
	// Standard kubebuilder init flags with centralized help text and defaults
	fs.StringVar(&p.domain, "domain", p.pluginConfig.Defaults.Domain, help.Domain)
	fs.StringVar(&p.repo, "repo", "", help.Repo)
	fs.StringVar(&p.owner, "owner", p.pluginConfig.Defaults.Owner, help.Owner)
}

// InjectConfig injects the project configuration  
func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	
	// Initialize utilities
	p.ensureUtilities()
	
	// Set domain if provided
	if p.domain != "" {
		if err := p.config.SetDomain(p.domain); err != nil {
			return fmt.Errorf("error setting domain: %w", err)
		}
	}
	
	// Set repository - use provided value or generate default
	repo := p.repo
	if repo == "" {
		// Generate default repository name using centralized logic
		repo = p.pluginConfig.GenerateDefaultRepo()
		fmt.Printf("No --repo flag provided, using default: %s\n", repo)
	}
	
	if err := p.config.SetRepository(repo); err != nil {
		return fmt.Errorf("error setting repository: %w", err)
	}
	
	return nil
}

// PreScaffold runs before scaffolding
func (p *initSubcommand) PreScaffold(machinery.Filesystem) error {
	// TODO: Add any pre-scaffolding logic
	return nil
}

// Scaffold scaffolds the initial project structure
func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project...\n")
	
	scaffolder := scaffolds.NewInitScaffolder(p.config)
	return scaffolder.Scaffold(fs)
}

// PostScaffold runs after scaffolding
func (p *initSubcommand) PostScaffold() error {
	// Ensure utilities are initialized
	p.ensureUtilities()
	
	// Initialize git repository if not already initialized
	if err := p.gitUtils.InitRepo(); err != nil {
		fmt.Printf("Warning: Could not initialize git repository: %v\n", err)
	} else {
		// Add initial files and create first commit
		if err := p.gitUtils.CreateInitialCommit(); err != nil {
			fmt.Printf("Warning: Could not create initial commit: %v\n", err)
		}
		
		// Add the build submodule as required by provider-template pattern
		if err := p.gitUtils.AddBuildSubmodule(); err != nil {
			fmt.Printf("Warning: Could not add build submodule: %v\n", err)
			fmt.Printf("You can manually add it later with: git submodule add %s build\n", 
				p.pluginConfig.Git.BuildSubmoduleURL)
		}
	}

	fmt.Println("Crossplane provider project initialized successfully!")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Run 'make submodules' to initialize the build system\n") 
	fmt.Printf("  2. Run 'go mod tidy' to download dependencies\n")
	fmt.Printf("  3. Run 'make generate' to generate required code\n")
	fmt.Printf("  4. Run 'make reviewable' to ensure code quality\n")
	fmt.Printf("  5. Use 'crossplane-provider-gen create api' to add managed resources\n")
	fmt.Printf("  6. Implement external client logic for your provider\n")
	fmt.Printf("  7. Run 'make build' to build the provider\n")

	return nil
}

// ensureUtilities initializes utility dependencies if they haven't been created yet
func (p *initSubcommand) ensureUtilities() {
	if p.pluginConfig == nil {
		p.pluginConfig = NewPluginConfig()
		p.gitUtils = NewGitUtils(p.pluginConfig)
		p.stringUtils = NewStringUtils()
		p.metadataUtils = NewMetadataUtils(p.pluginConfig)
	}
}