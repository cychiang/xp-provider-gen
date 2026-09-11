package v2

import (
	"context"
	"fmt"
	"os"
	"strings"

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

	domain   string
	repo     string
	gitName  string
	gitEmail string

	// upjet selects the upjet flavor and carries its Terraform coordinates.
	upjet             bool
	tfProvider        string
	tfVersion         string
	tfProviderVersion string
	tfProviderRepo    string
	tfDocsPath        string

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

  # Initialize with specific git user configuration
  %s init --domain=example.com --repo=github.com/example/provider-aws \
    --git-name="Crossplane Provider Generator" --git-email="noreply@crossplane.io"`,
		cliMeta.CommandName, cliMeta.CommandName, cliMeta.CommandName, cliMeta.CommandName)
}

func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	p.ensureConfig()

	fs.StringVar(&p.domain, "domain", p.pluginConfig.Defaults.Domain, "domain for API groups")
	fs.StringVar(&p.repo, "repo", "", "name to use for go module (e.g., github.com/user/repo)")
	fs.StringVar(&p.gitName, "git-name", "", "git user name for commits (uses system config if not provided)")
	fs.StringVar(&p.gitEmail, "git-email", "", "git user email for commits (uses system config if not provided)")

	fs.BoolVar(&p.upjet, "upjet", false,
		"scaffold an upjet provider: types and controllers are generated from a Terraform provider schema")
	fs.StringVar(&p.tfProvider, "terraform-provider", "",
		"Terraform provider to wrap, e.g. hashicorp/kubernetes (required with --upjet)")
	fs.StringVar(&p.tfProviderVersion, "terraform-provider-version", "",
		"version of the Terraform provider to wrap (required with --upjet)")
	fs.StringVar(&p.tfProviderRepo, "terraform-provider-repo", "",
		"git repository holding the Terraform provider's docs (defaults to its hashicorp GitHub repo)")
	fs.StringVar(&p.tfDocsPath, "terraform-provider-docs-path", core.DefaultTerraformDocsPath,
		"path to resource docs inside that repository")
	fs.StringVar(&p.tfVersion, "terraform-version", core.DefaultTerraformVersion,
		"Terraform CLI version used to read the provider schema")
}

// upjetSettings validates the Terraform coordinates and fills in defaults.
func (p *initSubcommand) upjetSettings() (*core.UpjetSettings, error) {
	if p.tfProvider == "" || p.tfProviderVersion == "" {
		return nil, fmt.Errorf("--terraform-provider and --terraform-provider-version are required with --upjet")
	}
	if strings.Count(p.tfProvider, "/") != 1 {
		return nil, fmt.Errorf("--terraform-provider must be <org>/<name>, e.g. hashicorp/kubernetes (got %q)", p.tfProvider)
	}
	name := core.ProviderNameFromSource(p.tfProvider)
	repo := p.tfProviderRepo
	if repo == "" {
		repo = core.DefaultProviderRepo(p.tfProvider)
	}
	return &core.UpjetSettings{
		TerraformProvider:        p.tfProvider,
		TerraformProviderName:    name,
		TerraformProviderVersion: p.tfProviderVersion,
		TerraformProviderRepo:    repo,
		TerraformDocsPath:        p.tfDocsPath,
		TerraformVersion:         p.tfVersion,
		TerraformResourcePrefix:  name,
		NamespacedDomain:         core.NamespacedDomain(p.domain),
	}, nil
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	p.ensureConfig()

	// Resolve git configuration in priority order: CLI flags > System config > Project defaults
	p.resolveGitConfig()

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
	flavor := core.FlavorNative
	var upjet *core.UpjetSettings
	if p.upjet {
		var err error
		if upjet, err = p.upjetSettings(); err != nil {
			return err
		}
		flavor = core.FlavorUpjet
	}

	// Record what this project is, so create api and update never ask again.
	if err := saveProjectMeta(p.config, func(m *projectMeta) {
		m.Flavor = flavor
		m.Upjet = upjet
	}); err != nil {
		return fmt.Errorf("recording project flavor: %w", err)
	}

	fmt.Printf("Scaffolding %s Crossplane provider project...\n", flavor)
	return scaffold.NewInitScaffolder(p.config, flavor, upjet).Scaffold(fs)
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
	if p.upjet {
		pipeline = automation.NewUpjetInitPipeline(p.pluginConfig, providerName)
	}

	fmt.Println("Running post-init automation...")
	if err := pipeline.Run(); err != nil {
		return validation.InitError("post-init automation", err)
	}

	fmt.Println("Crossplane provider project initialized successfully!")
	fmt.Printf("Next steps:\n")
	if p.upjet {
		// The project does not compile until upjet has generated the API types
		// and controllers from the Terraform schema, so that comes first.
		fmt.Printf("  1. Use 'crossplane-provider-gen create api --terraform-resource=...' to add resources\n")
		fmt.Printf("  2. Run 'make generate' to fetch the Terraform schema and generate types and controllers\n")
		fmt.Printf("  3. Map your credentials in internal/clients/clients.go\n")
		fmt.Printf("  4. Run 'make build' to build the provider\n")
		return nil
	}
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

func (p *initSubcommand) resolveGitConfig() {
	// Priority: CLI flags > System config > Project defaults

	// Start with project defaults
	finalName := p.pluginConfig.Git.Author
	finalEmail := p.pluginConfig.Git.Email

	// Override with system config if CLI flags not provided
	p.resolveSystemConfig(&finalName, &finalEmail)

	// CLI flags override everything
	if p.gitName != "" {
		finalName = p.gitName
	}
	if p.gitEmail != "" {
		finalEmail = p.gitEmail
	}

	// Update the plugin config with resolved values
	p.pluginConfig.Git.Author = finalName
	p.pluginConfig.Git.Email = finalEmail
}

func (p *initSubcommand) resolveSystemConfig(name, email *string) {
	if p.gitName != "" && p.gitEmail != "" {
		return // Both CLI flags provided, skip system config
	}

	wd, _ := os.Getwd()
	runner := core.NewGitCommandRunner(wd)

	if p.gitName == "" {
		if systemName, err := runner.GetUserName(context.Background()); err == nil {
			if trimmed := strings.TrimSpace(systemName); trimmed != "" {
				*name = trimmed
			}
		}
	}

	if p.gitEmail == "" {
		if systemEmail, err := runner.GetUserEmail(context.Background()); err == nil {
			if trimmed := strings.TrimSpace(systemEmail); trimmed != "" {
				*email = trimmed
			}
		}
	}
}
