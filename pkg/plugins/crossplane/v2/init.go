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
	config config.Config
	
	domain string
	repo   string
	owner  string
	
	pluginConfig *PluginConfig
}

func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Initialize a new Crossplane provider project.

This command initializes a new Crossplane provider project with the necessary
scaffolding to develop and build a Kubernetes controller to manage external
resources following Crossplane patterns.`
	
	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a basic Crossplane provider project
  %s init --domain=example.com --repo=github.com/example/provider-example`, cliMeta.CommandName)
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	p.ensureConfig()
	
	fs.StringVar(&p.domain, "domain", p.pluginConfig.Defaults.Domain, "domain for API groups")
	fs.StringVar(&p.repo, "repo", "", "name to use for go module (e.g., github.com/user/repo)")
	fs.StringVar(&p.owner, "owner", p.pluginConfig.Defaults.Owner, "owner to add to the copyright")
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	p.ensureConfig()
	
	if p.domain != "" {
		if err := p.config.SetDomain(p.domain); err != nil {
			return fmt.Errorf("error setting domain: %w", err)
		}
	}
	
	repo := p.repo
	if repo == "" {
		repo = p.pluginConfig.GenerateDefaultRepo()
		fmt.Printf("No --repo flag provided, using default: %s\n", repo)
	}
	
	if err := p.config.SetRepository(repo); err != nil {
		return fmt.Errorf("error setting repository: %w", err)
	}
	
	return nil
}

func (p *initSubcommand) PreScaffold(machinery.Filesystem) error {
	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project...\n")
	
	scaffolder := scaffolds.NewInitScaffolder(p.config)
	return scaffolder.Scaffold(fs)
}

func (p *initSubcommand) PostScaffold() error {
	p.ensureConfig()
	gitUtils := NewGitUtils(p.pluginConfig)
	
	if err := gitUtils.InitRepo(); err != nil {
		fmt.Printf("Warning: Could not initialize git repository: %v\n", err)
	} else {
		if err := gitUtils.CreateInitialCommit(); err != nil {
			fmt.Printf("Warning: Could not create initial commit: %v\n", err)
		}
		
		if err := gitUtils.AddBuildSubmodule(); err != nil {
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

func (p *initSubcommand) ensureConfig() {
	if p.pluginConfig == nil {
		p.pluginConfig = NewPluginConfig()
	}
}