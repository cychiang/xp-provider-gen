package v2

import (
	"fmt"
	"os"
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
	GenerateClient bool
	Force          bool

	config       config.Config
	resource     *resource.Resource
	pluginConfig *PluginConfig
}

func (p *createAPISubcommand) UpdateMetadata(cliMeta plugin.CLIMetadata, subcmdMeta *plugin.SubcommandMetadata) {
	p.ensureConfig()
	
	subcmdMeta.Description = `Create a new Crossplane managed resource API.

This command scaffolds a new Crossplane managed resource with the necessary
controller implementation using crossplane-runtime patterns.`
	
	subcmdMeta.Examples = fmt.Sprintf(`  # Create a basic managed resource
  %s create api --group=compute --version=v1alpha1 --kind=Instance

  # Create a storage resource
  %s create api --group=storage --version=v1beta1 --kind=Bucket`, 
		cliMeta.CommandName, cliMeta.CommandName)
}

func (p *createAPISubcommand) BindFlags(fs *pflag.FlagSet) {
	p.ensureConfig()
	
	defaults := p.pluginConfig.Defaults
	fs.BoolVar(&p.GenerateClient, "generate-client", defaults.GenerateClient, "generate external client interface")
	fs.BoolVar(&p.Force, "force", defaults.Force, "overwrite existing files if they exist")
}

func (p *createAPISubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *createAPISubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res

	if res != nil {
		res.Path = fmt.Sprintf("%s/apis/%s/%s", p.config.GetRepository(), res.Group, res.Version)
		res.Domain = p.config.GetDomain()
		res.API = &resource.API{
			CRDVersion: "v1",
			Namespaced: true,
		}
		res.Controller = true
	}

	return nil
}

func (p *createAPISubcommand) PreScaffold(machinery.Filesystem) error {
	return nil
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	fmt.Printf("Creating Crossplane managed resource API %s/%s %s\n", p.resource.Group, p.resource.Version, p.resource.Kind)

	p.ensureConfig()
	
	providerName := p.extractProviderName()
	
	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(p.config),
		machinery.WithBoilerplate(p.pluginConfig.GetBoilerplate()),
		machinery.WithResource(p.resource),
	)

	if err := scaffold.Execute(
		&api.CrossplaneGroup{},
		&api.CrossplaneTypes{Force: p.Force},
		&controllers.CrossplaneController{
			Force: p.Force,
			RepositoryMixin: machinery.RepositoryMixin{Repo: p.config.GetRepository()},
			DomainMixin: machinery.DomainMixin{Domain: p.config.GetDomain()},
		},
		&templates.TemplateUpdater{
			Force: true,
			RepositoryMixin: machinery.RepositoryMixin{Repo: p.config.GetRepository()},
			ProviderName: providerName,
		},
	); err != nil {
		return fmt.Errorf("error scaffolding Crossplane managed resource: %w", err)
	}

	fmt.Printf("Successfully scaffolded Crossplane managed resource %s\n", p.resource.Kind)
	return nil
}

func (p *createAPISubcommand) PostScaffold() error {
	if err := p.config.AddResource(*p.resource); err != nil {
		return fmt.Errorf("error adding resource to project config: %w", err)
	}
	
	if err := p.saveProjectFile(); err != nil {
		return fmt.Errorf("error saving PROJECT file: %w", err)
	}

	fmt.Printf("Crossplane managed resource %s created successfully!\n", p.resource.Kind)
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Customize the %sParameters and %sObservation structs\n", p.resource.Kind, p.resource.Kind)
	fmt.Printf("  2. Implement the external client logic\n")
	fmt.Printf("  3. Update controller reconciliation logic\n")
	fmt.Printf("  4. Run 'make manifests' to generate CRDs\n")

	return nil
}

func (p *createAPISubcommand) saveProjectFile() error {
	configBytes, err := p.config.MarshalYAML()
	if err != nil {
		return fmt.Errorf("error marshaling config to YAML: %w", err)
	}
	
	if err := os.WriteFile("PROJECT", configBytes, 0644); err != nil {
		return fmt.Errorf("error writing PROJECT file: %w", err)
	}
	
	return nil
}

func (p *createAPISubcommand) ensureConfig() {
	if p.pluginConfig == nil {
		p.pluginConfig = NewPluginConfig()
	}
}

func (p *createAPISubcommand) extractProviderName() string {
	if p.config.GetRepository() != "" {
		parts := strings.Split(p.config.GetRepository(), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return "provider-example"
}