package v1

import (
	"fmt"

	"github.com/spf13/pflag"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"

	"github.com/crossplane/xp-kubebuilder-plugin/pkg/plugins/crossplane/v1/scaffolds"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	// Standard kubebuilder fields
	config config.Config
	
	// Standard init flags
	domain string
	repo   string
	owner  string
}

// UpdateMetadata updates the plugin metadata
func (p *initSubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	subcmdMeta.Description = `Initialize a new Crossplane provider project.

This command initializes a new Crossplane provider project with the necessary
scaffolding to develop and build a Kubernetes controller to manage external
resources following Crossplane patterns.
`
	subcmdMeta.Examples = fmt.Sprintf(`  # Initialize a basic Crossplane provider project
  %[1]s init --domain=example.com --repo=github.com/example/provider-example
`,
		cliMeta.CommandName)
}

// BindFlags binds the subcommand flags
func (p *initSubcommand) BindFlags(fs *pflag.FlagSet) {
	// Standard kubebuilder init flags
	fs.StringVar(&p.domain, "domain", "", "domain for API groups")
	fs.StringVar(&p.repo, "repo", "", "name to use for go module (e.g., github.com/user/repo)")
	fs.StringVar(&p.owner, "owner", "", "owner to add to the copyright")
}

// InjectConfig injects the project configuration  
func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	
	// Set domain if provided
	if p.domain != "" {
		if err := p.config.SetDomain(p.domain); err != nil {
			return fmt.Errorf("error setting domain: %w", err)
		}
	}
	
	// Set repository if provided
	if p.repo != "" {
		if err := p.config.SetRepository(p.repo); err != nil {
			return fmt.Errorf("error setting repository: %w", err)
		}
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
	fmt.Println("Crossplane provider project initialized successfully!")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Run 'go mod tidy' to download dependencies\n") 
	fmt.Printf("  2. Use 'kubebuilder create api' to add managed resources\n")
	fmt.Printf("  3. Implement external client logic for your provider\n")
	fmt.Printf("  4. Run 'make build' to build the provider\n")

	return nil
}