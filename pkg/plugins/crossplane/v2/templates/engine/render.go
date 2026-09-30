/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package engine

import (
	"errors"
	"fmt"
	"slices"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// Scope says how much of a project Render produces.
type Scope int

const (
	// ScopeInit renders what `init` writes: the init templates, the generators
	// with no resources yet, and the go.mod seed.
	ScopeInit Scope = iota
	// ScopeKind renders what `create api` writes for one kind: that kind's
	// templates plus the generators over every resource.
	ScopeKind
	// ScopeProject renders what `update` compares against: the init templates,
	// the generators over every resource, and every kind's templates.
	ScopeProject
)

// Render writes one scope of a project of the given flavor into dst, through
// the single assembly of templates and generators every command shares.
//
// upjet carries the Terraform coordinates the templates render with (nil for
// native). resources is the project's full resource list; ScopeInit ignores it,
// since init has no kinds yet. target is the kind ScopeKind renders, chosen by
// the caller rather than by position in resources; the other scopes ignore it.
// opts reach the per-kind templates only (for example WithForce); ScopeProject
// always forces them, as update decides per file what to write.
func Render(dst machinery.Filesystem, cfg config.Config, flavor core.Flavor, upjet *core.UpjetSettings,
	resources []resource.Resource, scope Scope, target *resource.Resource, opts ...Option,
) error {
	if scope == ScopeKind && target == nil {
		return errors.New("rendering kind scope: target resource is nil")
	}

	// init has no kinds yet. Seeding the registration files through the same
	// generators `create api` uses, with no resources, makes init and create
	// produce byte-identical register files for the base case.
	generatorResources := resources
	if scope == ScopeInit {
		generatorResources = nil
	}
	generators := CoreGeneratorsFor(flavor, cfg, generatorResources)
	factory := NewFactoryForFlavor(cfg, flavor)

	switch scope {
	case ScopeInit:
		return renderInit(dst, cfg, factory, flavor, upjet, generators)
	case ScopeKind:
		return renderKind(dst, cfg, factory, upjet, generators, target, opts)
	case ScopeProject:
		return renderProjectScope(dst, cfg, factory, upjet, generators, resources)
	default:
		return fmt.Errorf("rendering: unknown scope %d", scope)
	}
}

func newScaffold(dst machinery.Filesystem, cfg config.Config, extra ...machinery.ScaffoldOption) *machinery.Scaffold {
	opts := append([]machinery.ScaffoldOption{
		machinery.WithConfig(cfg),
		machinery.WithBoilerplate(DefaultBoilerplate()),
	}, extra...)
	return machinery.NewScaffold(dst, opts...)
}

// initBuilders returns the init templates followed by the generators.
func initBuilders(factory *CrossplaneTemplateFactory, upjet *core.UpjetSettings,
	generators []machinery.Builder,
) ([]machinery.Builder, error) {
	templates, err := factory.GetInitTemplates(WithUpjet(upjet))
	if err != nil {
		return nil, fmt.Errorf("getting init templates: %w", err)
	}
	return append(AsBuilders(templates), generators...), nil
}

// kindTemplates returns the per-kind templates of one resource.
func kindTemplates(factory *CrossplaneTemplateFactory, upjet *core.UpjetSettings, res *resource.Resource,
	opts []Option,
) ([]machinery.Builder, error) {
	templates, err := factory.GetAPITemplates(slices.Concat(opts, []Option{WithResource(res), WithUpjet(upjet)})...)
	if err != nil {
		return nil, fmt.Errorf("getting api templates for %s: %w", res.Kind, err)
	}
	return AsBuilders(templates), nil
}

func renderInit(dst machinery.Filesystem, cfg config.Config, factory *CrossplaneTemplateFactory,
	flavor core.Flavor, upjet *core.UpjetSettings, generators []machinery.Builder,
) error {
	builders, err := initBuilders(factory, upjet, generators)
	if err != nil {
		return err
	}
	builders = append(builders, NewGoModGenerator(cfg.GetRepository(), DependenciesFor(flavor)))
	if err := newScaffold(dst, cfg).Execute(builders...); err != nil {
		return fmt.Errorf("rendering init scope: %w", err)
	}
	return nil
}

func renderKind(dst machinery.Filesystem, cfg config.Config, factory *CrossplaneTemplateFactory,
	upjet *core.UpjetSettings, generators []machinery.Builder, target *resource.Resource, opts []Option,
) error {
	builders, err := kindTemplates(factory, upjet, target, opts)
	if err != nil {
		return err
	}
	builders = append(builders, generators...)
	if err := newScaffold(dst, cfg, machinery.WithResource(target)).Execute(builders...); err != nil {
		return fmt.Errorf("rendering kind scope for %s: %w", target.Kind, err)
	}
	return nil
}

// renderProjectScope has no go.mod seed: go.mod is the provider author's, which
// update never rewrites.
func renderProjectScope(dst machinery.Filesystem, cfg config.Config, factory *CrossplaneTemplateFactory,
	upjet *core.UpjetSettings, generators []machinery.Builder, resources []resource.Resource,
) error {
	builders, err := initBuilders(factory, upjet, generators)
	if err != nil {
		return err
	}
	if err := newScaffold(dst, cfg).Execute(builders...); err != nil {
		return fmt.Errorf("rendering project scope: %w", err)
	}

	for i := range resources {
		res := &resources[i]
		builders, err := kindTemplates(factory, upjet, res, []Option{WithForce(true)})
		if err != nil {
			return err
		}
		if err := newScaffold(dst, cfg, machinery.WithResource(res)).Execute(builders...); err != nil {
			return fmt.Errorf("rendering project scope for %s: %w", res.Kind, err)
		}
	}
	return nil
}
