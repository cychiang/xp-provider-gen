package v1

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

const pluginName = "crossplane." + golang.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 1}
	supportedProjectVersions = []config.Version{{Number: 4}}
)

// Plugin implements the kubebuilder plugin interface for Crossplane providers
type Plugin struct {
	// We can optionally compose with golang/v4 plugin later
	// golangv4.Plugin
}

// Name returns the name of the plugin
func (p Plugin) Name() string {
	return pluginName
}

// Version returns the plugin version
func (p Plugin) Version() plugin.Version {
	return pluginVersion
}

// SupportedProjectVersions returns the project versions supported by this plugin
func (p Plugin) SupportedProjectVersions() []config.Version {
	return supportedProjectVersions
}

// GetInitSubcommand returns the init subcommand for this plugin
func (p Plugin) GetInitSubcommand() plugin.InitSubcommand {
	return &initSubcommand{}
}

// GetCreateAPISubcommand returns the create api subcommand for this plugin
func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand {
	return &createAPISubcommand{}
}

// GetCreateWebhookSubcommand returns the create webhook subcommand for this plugin
func (p Plugin) GetCreateWebhookSubcommand() plugin.CreateWebhookSubcommand {
	return &createWebhookSubcommand{}
}

// GetEditSubcommand returns the edit subcommand for this plugin
func (p Plugin) GetEditSubcommand() plugin.EditSubcommand {
	return &editSubcommand{}
}

// DeprecationWarning returns any deprecation warning for this plugin
func (p Plugin) DeprecationWarning() string {
	return ""
}

var _ plugin.Full = Plugin{}