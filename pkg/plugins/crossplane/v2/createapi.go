package v2

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/automation"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/templates/engine"
	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/validation"
	"github.com/cychiang/xp-provider-gen/pkg/version"
)

// createAPIOptions are create api's flags.
type createAPIOptions struct {
	group, version, kind string

	// terraformResource is the Terraform resource an upjet kind is generated
	// from, e.g. kubernetes_secret. Unused by native providers.
	terraformResource string
}

// NewCreateAPICommand returns the `create api` command; the root adds it under `create`.
func NewCreateAPICommand() *cobra.Command {
	var o createAPIOptions
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Create a new Crossplane managed resource API",
		Long: `Create a new Crossplane managed resource API.

This command scaffolds a complete managed resource with:
- Custom Resource Definition (CRD) following Kubernetes standards
- Parameters and Observation structs for external resource lifecycle
- Controller implementation with crossplane-runtime v2 patterns
- External client interface for cloud API integration
- Automatic registration in controller manager`,
		Example: fmt.Sprintf(`  # Create a compute resource
  %[1]s create api --group=compute --version=v1alpha1 --kind=Instance

  # Create a storage resource with stable version
  %[1]s create api --group=storage --version=v1beta1 --kind=Bucket

  # Create a network resource
  %[1]s create api --group=network --version=v1alpha1 --kind=VPC`, version.CommandName),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCreateAPI(cmd.Context(), afero.NewOsFs(), o)
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&o.group, "group", "", "Resource Group (e.g., batch, apps)")
	fs.StringVar(&o.version, "version", "", "Resource Version (e.g., v1, v1beta1)")
	fs.StringVar(&o.kind, "kind", "", "Resource Kind (e.g., CronJob, Deployment)")
	fs.StringVar(&o.terraformResource, "terraform-resource", "",
		"Terraform resource this kind is generated from, e.g. kubernetes_secret (required on an upjet provider)")
	return cmd
}

// runCreateAPI adds one kind to the provider in dst, which must be the tree
// rooted at the working directory: git, make and go run there, not through
// dst. PROJECT is saved after the files are written and before the automation
// pipeline, as in runInit.
func runCreateAPI(ctx context.Context, dst afero.Fs, o createAPIOptions) error {
	st, err := loadProjectStore(dst)
	if err != nil {
		return err
	}
	cfg := st.Config()
	res, err := newResource(cfg, o)
	if err != nil {
		return err
	}
	meta, err := checkCreateAPI(ctx, cfg, res, o)
	if err != nil {
		return err
	}
	if err := writeCreateAPI(dst, cfg, meta, res, o); err != nil {
		return err
	}
	if err := st.Save(); err != nil {
		return fmt.Errorf("saving PROJECT: %w", err)
	}

	pipeline := automation.APICommitPipelineFor(meta.Flavor, core.NewPluginConfig(), res.Kind)
	fmt.Println("Running post-scaffolding automation...")
	if err := pipeline.Run(); err != nil {
		return validation.CreateAPIError("post-scaffolding automation", err)
	}

	fmt.Printf("Crossplane managed resource %s created successfully!\n", res.Kind)
	if meta.Flavor == core.FlavorUpjet {
		fmt.Printf("Configured %s in config/%s/config.go.\n", o.terraformResource, strings.ToLower(res.Kind))
	}
	fmt.Printf("Next steps:\n")
	if meta.Flavor == core.FlavorUpjet {
		fmt.Printf("  1. Run 'make generate' to generate its API types and controller\n")
		fmt.Printf("  2. Map any new credentials in internal/clients/clients.go\n")
		fmt.Printf("  3. Write %s: %s — 'create-test' and 'make e2e' both derive from it\n",
			examplePath(*res), upjetExampleHint(*res))
		fmt.Printf("  4. Run '%s create-test --kind=%s' once that example exists\n", version.CommandName, res.Kind)
		return nil
	}
	fmt.Printf("  1. Customize the %sParameters and %sObservation structs\n", res.Kind, res.Kind)
	fmt.Printf("  2. Implement the external client logic\n")
	fmt.Printf("  3. Update controller reconciliation logic\n")
	fmt.Printf("  4. Run 'make generate' to generate CRDs\n")
	fmt.Printf("  5. Check %s for usage examples\n", examplePath(*res))

	return nil
}

// newResource builds and validates the kind o names, as the project cfg
// declares it.
func newResource(cfg config.Config, o createAPIOptions) (*resource.Resource, error) {
	// A flag value that starts with "-" is the next flag, taken as the value
	// because the flag had none. No field accepts a "-" first, and checking
	// all three before any emptiness check keeps the error on the flag the
	// author left empty.
	for _, f := range []struct{ name, value string }{
		{"group", o.group}, {"version", o.version}, {"kind", o.kind},
	} {
		if strings.HasPrefix(f.value, "-") {
			return nil, fmt.Errorf("created invalid resource: %s flag present but empty", f.name)
		}
	}

	res := &resource.Resource{
		GVK: resource.GVK{
			Group:   strings.TrimSpace(o.group),
			Domain:  cfg.GetDomain(),
			Version: strings.TrimSpace(o.version),
			Kind:    strings.TrimSpace(o.kind),
		},
		Plural:     resource.RegularPlural(o.kind),
		Path:       core.APIImportPath(cfg.GetRepository(), strings.TrimSpace(o.group), strings.TrimSpace(o.version)),
		API:        &resource.API{CRDVersion: "v1", Namespaced: true},
		Controller: true,
		Webhooks:   &resource.Webhooks{},
	}
	if err := res.Validate(); err != nil {
		return nil, fmt.Errorf("created invalid resource: %w", err)
	}
	return res, nil
}

// checkCreateAPI refuses what create api must not go on from: an init that did
// not finish, an unusable PROJECT, another generator's version, an invalid
// resource, a missing or malformed --terraform-resource. It returns the
// project's plugin block.
func checkCreateAPI(
	ctx context.Context, cfg config.Config, res *resource.Resource, o createAPIOptions,
) (projectMeta, error) {
	if err := requireInitCommit(ctx, ""); err != nil {
		return projectMeta{}, validation.CreateAPIError("init check", err)
	}

	meta, err := validateProject(cfg)
	if err != nil {
		return projectMeta{}, validation.CreateAPIError("configuration check", err)
	}

	if err := checkSameGenerator(meta.Version, version.Get().Version); err != nil {
		return projectMeta{}, validation.CreateAPIError("version check", err)
	}

	if err := validation.ValidatorFor(meta.Flavor).ValidateResource(res); err != nil {
		return projectMeta{}, validation.CreateAPIError("resource validation", err)
	}

	if res.Domain == "" {
		return projectMeta{}, validation.CreateAPIError("configuration check",
			fmt.Errorf("resource domain is required - ensure project is properly initialized"))
	}

	if meta.Flavor == core.FlavorUpjet {
		if o.terraformResource == "" {
			return projectMeta{}, validation.CreateAPIError("missing flag",
				fmt.Errorf("--terraform-resource is required on an upjet provider "+
					"(e.g. --terraform-resource=%s_secret)", meta.Upjet.TerraformResourcePrefix))
		}
		if err := validation.ValidateTerraformResource(o.terraformResource, meta.Upjet.TerraformResourcePrefix); err != nil {
			return projectMeta{}, validation.CreateAPIError("terraform resource validation", err)
		}
	}

	return meta, nil
}

// requireInitCommit refuses a git repository that has no commit in dir ("" is
// the working directory). init makes the scaffold commit last, so a repository
// without one means init stopped midway; going on would let create api's commit
// step sweep the half-made tree into a commit titled for the new kind. A
// directory with no .git passes, as before.
func requireInitCommit(ctx context.Context, dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	git := core.NewGitCommandRunner(dir)
	if _, err := git.RunCommandWithOutput(ctx, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		return errors.New("this repository has no commit, so init did not finish; " +
			"fix what stopped it and run init again in an empty directory")
	}
	return nil
}

// writeCreateAPI records res in cfg and writes its files into dst.
func writeCreateAPI(
	dst afero.Fs, cfg config.Config, meta projectMeta, res *resource.Resource, o createAPIOptions,
) error {
	fmt.Printf("Creating Crossplane managed resource API %s/%s %s\n",
		res.Group, res.Version, res.Kind)

	if err := cfg.AddResource(*res); err != nil {
		return validation.CreateAPIError("recording resource in project config", err)
	}

	// Per-resource templates need the Terraform coordinates as well as the kind.
	// --terraform-resource presence and shape were already validated in checkCreateAPI.
	upjet := meta.Upjet
	if upjet != nil {
		settings := *upjet
		settings.TerraformResource = o.terraformResource
		upjet = &settings
	}

	resources, err := cfg.GetResources()
	if err != nil {
		return validation.CreateAPIError("reading project resources", err)
	}

	// Render the new kind's templates and regenerate the registration files
	// from the full resource list in memory, then write through the ownership
	// rule update uses: headered files are refreshed, files without the header
	// are the author's and kept. ScopeKind renders no init-layer file, so
	// seeding user-owned files stays limited to this kind's paths.
	mem := afero.NewMemMapFs()
	if err := engine.Render(machinery.Filesystem{FS: mem}, cfg, meta.Flavor, upjet, resources,
		engine.ScopeKind, res); err != nil {
		return validation.CreateAPIError("scaffolding", err)
	}
	result, err := core.Apply(mem, dst, true)
	if err != nil {
		return validation.CreateAPIError("writing scaffold", err)
	}
	printKept(os.Stdout, result.Skipped)
	for _, rel := range result.Skipped {
		if engine.IsRegistrationFile(meta.Flavor, rel) {
			fmt.Printf("  warning: %s has no generated header, so %s was not registered in it; add it yourself\n",
				rel, res.Kind)
		}
	}

	return nil
}
