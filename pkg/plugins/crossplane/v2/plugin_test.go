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
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

func TestPlugin_Interface(t *testing.T) {
	// Ensure Plugin implements kubebuilder plugin.Full interface
	var _ plugin.Full = &Plugin{}
}

func TestPlugin_Name(t *testing.T) {
	p := Plugin{}
	name := p.Name()

	if name == "" {
		t.Error("Plugin name should not be empty")
	}

	// Should follow kubebuilder naming convention
	expected := "crossplane.go.kubebuilder.io"
	if name != expected {
		t.Errorf("Plugin name = %s, want %s", name, expected)
	}
}

func TestPlugin_Version(t *testing.T) {
	p := Plugin{}
	version := p.Version()

	if version.Number != 2 {
		t.Errorf("Plugin version = %d, want 2", version.Number)
	}
}

func TestPlugin_SupportedProjectVersions(t *testing.T) {
	p := Plugin{}
	versions := p.SupportedProjectVersions()

	if len(versions) == 0 {
		t.Error("Plugin should support at least one project version")
	}

	// Should support kubebuilder v3 projects
	hasV3 := false
	for _, v := range versions {
		if v.Number == 3 {
			hasV3 = true
			break
		}
	}

	if !hasV3 {
		t.Error("Plugin should support kubebuilder project version 3")
	}
}

func TestPlugin_Subcommands(t *testing.T) {
	p := Plugin{}

	// Should provide init subcommand
	initCmd := p.GetInitSubcommand()
	if initCmd == nil {
		t.Error("Plugin should provide init subcommand")
	}

	// Should provide create api subcommand
	createAPICmd := p.GetCreateAPISubcommand()
	if createAPICmd == nil {
		t.Error("Plugin should provide create api subcommand")
	}

	// Should not provide webhook subcommand (Crossplane doesn't use them typically)
	webhookCmd := p.GetCreateWebhookSubcommand()
	if webhookCmd != nil {
		t.Error("Plugin should not provide webhook subcommand")
	}

	// Should not provide edit subcommand
	editCmd := p.GetEditSubcommand()
	if editCmd != nil {
		t.Error("Plugin should not provide edit subcommand")
	}
}

func TestPlugin_DeprecationWarning(t *testing.T) {
	p := Plugin{}
	warning := p.DeprecationWarning()

	// Should not have deprecation warning for current version
	if warning != "" {
		t.Errorf("Plugin should not have deprecation warning, got: %s", warning)
	}
}

func TestInitSubcommand_Interface(t *testing.T) {
	// Ensure initSubcommand implements kubebuilder plugin interface
	var _ plugin.InitSubcommand = &initSubcommand{}
}

func TestCreateAPISubcommand_Interface(t *testing.T) {
	// Ensure createAPISubcommand implements kubebuilder plugin interface
	var _ plugin.CreateAPISubcommand = &createAPISubcommand{}
}

func TestPluginConfig_Defaults(t *testing.T) {
	cfg := NewPluginConfig()

	if cfg.Name == "" {
		t.Error("Plugin config name should not be empty")
	}

	if cfg.Version == "" {
		t.Error("Plugin config version should not be empty")
	}

	if cfg.Git.BuildSubmoduleURL == "" {
		t.Error("Build submodule URL should not be empty")
	}

	// Should have Crossplane build submodule URL
	expected := "https://github.com/crossplane/build"
	if cfg.Git.BuildSubmoduleURL != expected {
		t.Errorf("Build submodule URL = %s, want %s", cfg.Git.BuildSubmoduleURL, expected)
	}
}

func TestPluginConfig_GenerateDefaultRepo(t *testing.T) {
	cfg := NewPluginConfig()

	repo := cfg.GenerateDefaultRepo()
	if repo == "" {
		t.Error("Generated default repo should not be empty")
	}

	// Should contain provider prefix by default
	if !contains(repo, "provider-") {
		t.Error("Generated repo should contain 'provider-' prefix")
	}

	// Should contain crossplane-contrib prefix by default
	if !contains(repo, "github.com/crossplane-contrib") {
		t.Error("Generated repo should use crossplane-contrib by default")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOfSubstring(s, substr) >= 0))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
