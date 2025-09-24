package v2

import (
	"fmt"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/automation"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/scaffold"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/validation"
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

This command scaffolds a complete Crossplane provider project with:
- ProviderConfig APIs for authentication
- Package metadata for Crossplane registry
- Build system integration via git submodules
- Controller scaffolding following Crossplane v2 patterns
- Go module and project structure`

	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a basic provider
  %s init --domain=example.com --repo=github.com/example/provider-aws

  # Initialize with custom organization
  %s init --domain=acme.com --repo=github.com/acme/provider-acme

  # Initialize in current directory (auto-detects name)
  %s init --domain=example.com

  # Initialize with owner for copyright
  %s init --domain=example.com --repo=github.com/example/provider-gcp --owner="Acme Corp"`,
		cliMeta.CommandName, cliMeta.CommandName, cliMeta.CommandName, cliMeta.CommandName)
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

	validator := validation.NewValidator()

	if p.domain != "" {
		if err := validator.ValidateDomain(p.domain); err != nil {
			return validation.InitError("domain validation", err)
		}

		if err := p.config.SetDomain(p.domain); err != nil {
			return validation.InitError("configuration", err)
		}
	}

	repo := p.repo
	if repo == "" {
		repo = p.pluginConfig.GenerateDefaultRepo()
		fmt.Printf("No --repo flag provided, using default: %s\n", repo)
	}

	if err := validator.ValidateRepository(repo); err != nil {
		return validation.InitError("repository validation", err)
	}

	if err := p.config.SetRepository(repo); err != nil {
		return validation.InitError("configuration", err)
	}

	return nil
}

func (p *initSubcommand) PreScaffold(machinery.Filesystem) error {
	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Scaffolding Crossplane provider project...\n")

	scaffolder := scaffold.NewInitScaffolder(p.config)
	return scaffolder.Scaffold(fs)
}

func (p *initSubcommand) PostScaffold() error {
	p.ensureConfig()

	// Save PROJECT file
	projectFile := core.NewProjectFile(p.config)
	if err := projectFile.Save(); err != nil {
		return validation.InitError("PROJECT file creation", err)
	}

	// Run automation pipeline
	providerName := core.ExtractProviderName(p.config.GetRepository())
	pipeline := automation.NewInitPipeline(p.pluginConfig, providerName)

	fmt.Println("Running post-init automation...")
	if err := pipeline.Run(); err != nil {
		fmt.Printf("Warning: Some automation steps failed: %v\n", err)
	}

	fmt.Println("Crossplane provider project initialized successfully!")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Use 'crossplane-provider-gen create api' to add managed resources\n")
	fmt.Printf("  2. Implement external client logic for your provider\n")
	fmt.Printf("  3. Run 'make build' to build the provider\n")
	fmt.Printf("  4. Run 'make run' to test the provider locally\n")

	return nil
}

func (p *initSubcommand) ensureConfig() {
	if p.pluginConfig == nil {
		p.pluginConfig = NewPluginConfig()
	}
}
