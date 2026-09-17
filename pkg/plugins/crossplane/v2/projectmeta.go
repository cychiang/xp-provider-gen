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

package v2

import (
	"errors"
	"fmt"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"

	"github.com/cychiang/xp-provider-gen/pkg/plugins/crossplane/v2/core"
)

// projectMeta is this plugin's block in PROJECT. It records what the project is
// (flavor, and for upjet the Terraform coordinates) and the generator version
// that last touched it, so later commands never ask the user again.
type projectMeta struct {
	Version string              `json:"version,omitempty"`
	Flavor  core.Flavor         `json:"flavor,omitempty"`
	Upjet   *core.UpjetSettings `json:"upjet,omitempty"`
}

// loadProjectMeta reads this plugin's block. A project scaffolded before the
// block existed has an empty flavor, which reads as native, which is what it
// is. Any other unrecognized flavor is reported rather than silently
// defaulted, since it means the file is corrupt, hand-edited, or was written
// by a newer xp-provider-gen this build does not know how to handle. A block
// that declares flavor: upjet but carries no upjet: settings is not a
// missing block, it is a corrupt one — callers dereference meta.Upjet, so
// this is reported too.
func loadProjectMeta(cfg config.Config) (projectMeta, error) {
	var meta projectMeta
	if err := cfg.DecodePluginConfig(pluginName, &meta); err != nil &&
		!errors.Is(err, config.PluginKeyNotFoundError{Key: pluginName}) {
		return projectMeta{}, fmt.Errorf("decoding %s project block: %w", pluginName, err)
	}
	switch {
	case meta.Flavor == "":
		meta.Flavor = core.FlavorNative
	case !meta.Flavor.Valid():
		return projectMeta{}, fmt.Errorf(
			"PROJECT declares unknown flavor %q (known: %s); "+
				"the file is corrupt, hand-edited, or written by a newer xp-provider-gen",
			meta.Flavor, knownFlavors())
	}
	if meta.Flavor == core.FlavorUpjet && meta.Upjet == nil {
		return projectMeta{}, fmt.Errorf(
			"PROJECT declares flavor %q but has no upjet: settings block; "+
				"the file is corrupt or was hand-edited", meta.Flavor)
	}
	return meta, nil
}

// saveProjectMeta writes the block back, preserving every field the caller did
// not set — stamping a new generator version must never drop the flavor.
func saveProjectMeta(cfg config.Config, mutate func(*projectMeta)) error {
	meta, err := loadProjectMeta(cfg)
	if err != nil {
		return err
	}
	mutate(&meta)
	return cfg.EncodePluginConfig(pluginName, meta)
}

// knownFlavors lists core.Flavors for an error message, so the message cannot
// fall behind when a flavor is added.
func knownFlavors() string {
	names := make([]string, 0, len(core.Flavors))
	for _, f := range core.Flavors {
		names = append(names, string(f))
	}
	return strings.Join(names, ", ")
}
