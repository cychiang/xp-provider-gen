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
// block existed reads as the native flavor, which is what it is.
func loadProjectMeta(cfg config.Config) projectMeta {
	var meta projectMeta
	// A missing or unreadable block is not an error: it just means defaults.
	_ = cfg.DecodePluginConfig(pluginName, &meta)
	if !meta.Flavor.Valid() {
		meta.Flavor = core.FlavorNative
	}
	return meta
}

// saveProjectMeta writes the block back, preserving every field the caller did
// not set — stamping a new generator version must never drop the flavor.
func saveProjectMeta(cfg config.Config, mutate func(*projectMeta)) error {
	meta := loadProjectMeta(cfg)
	mutate(&meta)
	return cfg.EncodePluginConfig(pluginName, meta)
}
