package v2

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/automation"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/templates/engine"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/validation"
	"github.com/cychiang/xp-provider-gen/pkg/version"
)

// initOptions are init's flags.
type initOptions struct {
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
}

// flavor reports which flavor this project is being scaffolded as, the one
// place --upjet is turned into a core.Flavor.
func (o initOptions) flavor() core.Flavor {
	if o.upjet {
		return core.FlavorUpjet
	}
	return core.FlavorNative
}

// NewInitCommand returns the `init` command.
func NewInitCommand() *cobra.Command {
	var o initOptions
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Crossplane provider project",
		Long: `Initialize a new Crossplane provider project.

This command scaffolds a complete Crossplane provider project with:
- ProviderConfig APIs for authentication
- Package metadata for Crossplane registry
- Build system integration via git submodules
- Controller scaffolding following Crossplane v2 patterns
- Go module and project structure`,
		Example: fmt.Sprintf(`  # Initialize a basic provider
  %[1]s init --domain=example.com --repo=github.com/example/provider-aws

  # Initialize with custom organization
  %[1]s init --domain=acme.com --repo=github.com/acme/provider-acme

  # Initialize in current directory (auto-detects name)
  %[1]s init --domain=example.com

  # Initialize with specific git user configuration
  %[1]s init --domain=example.com --repo=github.com/example/provider-aws \
    --git-name="Crossplane Provider Generator" --git-email="noreply@crossplane.io"`, version.CommandName),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd.Context(), afero.NewOsFs(), o)
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&o.domain, "domain", "", "domain for API groups (required)")
	fs.StringVar(&o.repo, "repo", "", "name to use for go module (e.g., github.com/user/repo)")
	fs.StringVar(&o.gitName, "git-name", "", "git user name for commits (uses system config if not provided)")
	fs.StringVar(&o.gitEmail, "git-email", "", "git user email for commits (uses system config if not provided)")

	fs.BoolVar(&o.upjet, "upjet", false,
		"scaffold an upjet provider: types and controllers are generated from a Terraform provider schema")
	fs.StringVar(&o.tfProvider, "terraform-provider", "",
		"Terraform provider to wrap, e.g. hashicorp/kubernetes (required with --upjet)")
	fs.StringVar(&o.tfProviderVersion, "terraform-provider-version", "",
		"version of the Terraform provider to wrap (required with --upjet)")
	fs.StringVar(&o.tfProviderRepo, "terraform-provider-repo", "",
		"git repository holding the Terraform provider's docs (defaults to its hashicorp GitHub repo)")
	fs.StringVar(&o.tfDocsPath, "terraform-provider-docs-path", core.DefaultTerraformDocsPath,
		"path to resource docs inside that repository")
	return cmd
}

// upjetSettings validates the Terraform coordinates and fills in defaults.
func (o initOptions) upjetSettings() (*core.UpjetSettings, error) {
	if o.tfProvider == "" || o.tfProviderVersion == "" {
		return nil, fmt.Errorf("--terraform-provider and --terraform-provider-version are required with --upjet")
	}
	if strings.Count(o.tfProvider, "/") != 1 {
		return nil, fmt.Errorf("--terraform-provider must be <org>/<name>, e.g. hashicorp/kubernetes (got %q)", o.tfProvider)
	}
	if err := validation.ValidateTerraformProviderVersion(o.tfProviderVersion); err != nil {
		return nil, fmt.Errorf("validating --terraform-provider-version: %w", err)
	}
	name := core.ProviderNameFromSource(o.tfProvider)
	repo := o.tfProviderRepo
	if repo == "" {
		repo = core.DefaultProviderRepo(o.tfProvider)
	}
	return &core.UpjetSettings{
		TerraformProvider:        o.tfProvider,
		TerraformProviderName:    name,
		TerraformProviderVersion: o.tfProviderVersion,
		TerraformProviderRepo:    repo,
		TerraformDocsPath:        o.tfDocsPath,
		TerraformResourcePrefix:  name,
	}, nil
}

// makeBuildStep is init's next step for building the provider, the same for
// both flavors.
const makeBuildStep = "Run 'make build' to build the provider binary and image (needs Docker)"

// runInit scaffolds a new provider into dst, which must be the tree rooted at
// the working directory: git, make and go run there, not through dst. PROJECT
// is saved after the files are written and before the automation pipeline, so
// a failed render leaves no PROJECT and a failed pipeline leaves one for the
// commit step to stage.
func runInit(ctx context.Context, dst afero.Fs, o initOptions) error {
	st, err := newProjectStore(dst)
	if err != nil {
		return err
	}
	pluginConfig, err := writeInit(ctx, dst, st.Config(), o)
	if err != nil {
		return err
	}
	if err := st.Save(); err != nil {
		return fmt.Errorf("saving PROJECT: %w", err)
	}

	providerName := core.ExtractProviderName(st.Config().GetRepository())
	pipeline := automation.InitPipelineFor(o.flavor(), pluginConfig, providerName)

	fmt.Println("Running post-init automation...")
	if err := pipeline.Run(); err != nil {
		return validation.InitError("post-init automation", err)
	}

	fmt.Println("Crossplane provider project initialized successfully!")
	fmt.Printf("Next steps:\n")
	if o.flavor() == core.FlavorUpjet {
		// The project does not compile until upjet has generated the API types
		// and controllers from the Terraform schema, so that comes first.
		fmt.Printf("  1. Use '%s create api --terraform-resource=...' to add resources\n", version.CommandName)
		fmt.Printf("  2. Run 'make generate' to fetch the Terraform schema and generate types and controllers\n")
		fmt.Printf("  3. Map your credentials in internal/clients/clients.go\n")
		fmt.Printf("  4. %s\n", makeBuildStep)
		return nil
	}
	fmt.Printf("  1. Use '%s create api' to add managed resources\n", version.CommandName)
	fmt.Printf("  2. Implement external client logic for your provider\n")
	fmt.Printf("  3. %s\n", makeBuildStep)
	fmt.Printf("  4. Run 'make run' to run the provider against your current kubeconfig context\n")

	return nil
}

// writeInit validates o, records the project in cfg and writes the scaffold
// into dst. It returns the plugin configuration the automation pipeline uses.
func writeInit(ctx context.Context, dst afero.Fs, cfg config.Config, o initOptions) (*core.PluginConfig, error) {
	pluginConfig := core.NewPluginConfig()
	// Resolve git configuration in priority order: CLI flags > System config > Project defaults
	pluginConfig.Git.Author, pluginConfig.Git.Email = o.gitIdentity(pluginConfig,
		func() (string, string) { return systemGitConfig(ctx) })

	flavor := o.flavor()
	validator := validation.ValidatorFor(flavor)

	if err := validator.ValidateDomain(o.domain); err != nil {
		return nil, validation.InitError("domain validation", err)
	}

	if err := cfg.SetDomain(o.domain); err != nil {
		return nil, validation.InitError("configuration", err)
	}

	repo := o.repo
	if repo == "" {
		repo = pluginConfig.GenerateDefaultRepo()
		fmt.Printf("No --repo flag provided, using default: %s\n", repo)
	}

	if err := validator.ValidateRepository(repo); err != nil {
		return nil, validation.InitError("repository validation", err)
	}
	if !validation.IsConventionalRepoName(repo) {
		parts := strings.Split(repo, "/")
		fmt.Printf("Warning: Repository name '%s' doesn't follow Crossplane convention 'provider-*'\n",
			parts[len(parts)-1])
	}

	if err := cfg.SetRepository(repo); err != nil {
		return nil, validation.InitError("configuration", err)
	}

	var upjet *core.UpjetSettings
	if flavor == core.FlavorUpjet {
		var err error
		if upjet, err = o.upjetSettings(); err != nil {
			return nil, err
		}
	}

	// Record what this project is, so create api and update never ask again.
	// Stamping the version here — not only on the first update — means a
	// fresh project is protected by checkNotDowngrade from the moment it
	// exists, not just after its first update.
	if err := cfg.EncodePluginConfig(pluginName, projectMeta{
		Version: version.Get().Version,
		Flavor:  flavor,
		Upjet:   upjet,
	}); err != nil {
		return nil, fmt.Errorf("recording project flavor: %w", err)
	}

	fmt.Printf("Scaffolding %s Crossplane provider project...\n", flavor)

	// Render in memory, then write through the ownership rule update uses: a
	// failed render leaves the directory untouched, headered files are
	// refreshed, and files without the header are the user's and kept.
	mem := afero.NewMemMapFs()
	if err := engine.Render(machinery.Filesystem{FS: mem}, cfg, flavor, upjet, nil, engine.ScopeInit, nil); err != nil {
		return nil, err
	}
	result, err := core.Apply(mem, dst, true)
	if err != nil {
		return nil, fmt.Errorf("writing scaffold: %w", err)
	}
	printKept(os.Stdout, result.Skipped)
	return pluginConfig, nil
}

// gitIdentity resolves the git identity used for automation commits, in
// priority order: CLI flags, then system git config (via query), then the
// project defaults already in defaults.
func (o initOptions) gitIdentity(defaults *core.PluginConfig, query func() (name, email string)) (string, string) {
	name := defaults.Git.Author
	email := defaults.Git.Email

	if o.gitName == "" || o.gitEmail == "" {
		sysName, sysEmail := query()
		if sysName != "" {
			name = sysName
		}
		if sysEmail != "" {
			email = sysEmail
		}
	}

	if o.gitName != "" {
		name = o.gitName
	}
	if o.gitEmail != "" {
		email = o.gitEmail
	}

	return name, email
}

// systemGitConfig reads user.name and user.email from git's own config
// resolution for the current directory, returning "" for a value that is
// unset or unreadable.
func systemGitConfig(ctx context.Context) (string, string) {
	var name, email string
	runner := core.NewGitCommandRunner("")
	if v, err := runner.GetUserName(ctx); err == nil {
		name = v
	}
	if v, err := runner.GetUserEmail(ctx); err == nil {
		email = v
	}
	return name, email
}
