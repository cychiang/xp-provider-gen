package v2

import (
	"context"
	"fmt"
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
	// The Terraform CLI version is deliberately not here: it is a tool
	// decision (pkg/versions.TerraformVersion, applied when templates render),
	// not a flag.
	upjet             bool
	tfProvider        string
	tfProviderVersion string
	tfProviderRepo    string
	tfDocsPath        string

	pluginConfig *core.PluginConfig
}

// flavor reports which flavor this project is being scaffolded as, the one
// place --upjet is turned into a core.Flavor.
func (p *initSubcommand) flavor() core.Flavor {
	if p.upjet {
		return core.FlavorUpjet
	}
	return core.FlavorNative
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
	fs.StringVar(&p.domain, "domain", "", "domain for API groups (required)")
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
		TerraformResourcePrefix:  name,
	}, nil
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	p.ensureConfig()

	// Resolve git configuration in priority order: CLI flags > System config > Project defaults
	p.resolveGitConfig()

	validator := validation.ValidatorFor(p.flavor())

	if err := validator.ValidateDomain(p.domain); err != nil {
		return validation.InitError("domain validation", err)
	}

	if err := p.config.SetDomain(p.domain); err != nil {
		return validation.InitError("configuration", err)
	}

	repo := p.repo
	if repo == "" {
		repo = p.pluginConfig.GenerateDefaultRepo()
		fmt.Printf("No --repo flag provided, using default: %s\n", repo)
	}

	if err := validator.ValidateRepository(repo); err != nil {
		return validation.InitError("repository validation", err)
	}
	if !validation.IsConventionalRepoName(repo) {
		parts := strings.Split(repo, "/")
		fmt.Printf("Warning: Repository name '%s' doesn't follow Crossplane convention 'provider-*'\n",
			parts[len(parts)-1])
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
	flavor := p.flavor()
	var upjet *core.UpjetSettings
	if flavor == core.FlavorUpjet {
		var err error
		if upjet, err = p.upjetSettings(); err != nil {
			return err
		}
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

	providerName := core.ExtractProviderName(p.config.GetRepository())
	pipeline := automation.InitPipelineFor(p.flavor(), p.pluginConfig, providerName)

	fmt.Println("Running post-init automation...")
	if err := pipeline.Run(); err != nil {
		return validation.InitError("post-init automation", err)
	}

	fmt.Println("Crossplane provider project initialized successfully!")
	fmt.Printf("Next steps:\n")
	if p.flavor() == core.FlavorUpjet {
		// The project does not compile until upjet has generated the API types
		// and controllers from the Terraform schema, so that comes first.
		fmt.Printf("  1. Use 'xp-provider-gen create api --terraform-resource=...' to add resources\n")
		fmt.Printf("  2. Run 'make generate' to fetch the Terraform schema and generate types and controllers\n")
		fmt.Printf("  3. Map your credentials in internal/clients/clients.go\n")
		fmt.Printf("  4. Run 'make build' to build the provider\n")
		return nil
	}
	fmt.Printf("  1. Use 'xp-provider-gen create api' to add managed resources\n")
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

// resolveGitConfig fills p.pluginConfig.Git from gitIdentity, querying the
// current directory's git configuration for values neither --git-name nor
// --git-email supplied.
func (p *initSubcommand) resolveGitConfig() {
	p.pluginConfig.Git.Author, p.pluginConfig.Git.Email = p.gitIdentity(systemGitConfig)
}

// gitIdentity resolves the git identity used for automation commits, in
// priority order: CLI flags, then system git config (via query), then the
// project defaults already in pluginConfig.
func (p *initSubcommand) gitIdentity(query func() (name, email string)) (string, string) {
	name := p.pluginConfig.Git.Author
	email := p.pluginConfig.Git.Email

	if p.gitName == "" || p.gitEmail == "" {
		sysName, sysEmail := query()
		if sysName != "" {
			name = sysName
		}
		if sysEmail != "" {
			email = sysEmail
		}
	}

	if p.gitName != "" {
		name = p.gitName
	}
	if p.gitEmail != "" {
		email = p.gitEmail
	}

	return name, email
}

// systemGitConfig reads user.name and user.email from git's own config
// resolution for the current directory, returning "" for a value that is
// unset or unreadable.
func systemGitConfig() (string, string) {
	var name, email string
	runner := core.NewGitCommandRunner("")
	if v, err := runner.GetUserName(context.Background()); err == nil {
		name = v
	}
	if v, err := runner.GetUserEmail(context.Background()); err == nil {
		email = v
	}
	return name, email
}
