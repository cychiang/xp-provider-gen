package v2

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang"
)

const pluginName = "crossplane." + golang.DefaultNameQualifier

var (
	pluginVersion            = plugin.Version{Number: 2}
	supportedProjectVersions = []config.Version{{Number: 3}}
)

type Plugin struct{}

func (p Plugin) Name() string { return pluginName }

func (p Plugin) Version() plugin.Version { return pluginVersion }

func (p Plugin) SupportedProjectVersions() []config.Version { return supportedProjectVersions }

func (p Plugin) GetInitSubcommand() plugin.InitSubcommand { return &initSubcommand{} }

func (p Plugin) GetCreateAPISubcommand() plugin.CreateAPISubcommand { return &createAPISubcommand{} }

func (p Plugin) GetCreateWebhookSubcommand() plugin.CreateWebhookSubcommand { return nil }

func (p Plugin) GetEditSubcommand() plugin.EditSubcommand { return nil }

func (p Plugin) DeprecationWarning() string { return "" }

var _ plugin.Full = Plugin{}
