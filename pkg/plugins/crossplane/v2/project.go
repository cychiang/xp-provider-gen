package v2

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3" // registers config version 3 with the yaml store
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

const (
	// pluginName keys this generator's block under PROJECT's plugins:.
	pluginName = "crossplane.go.kubebuilder.io"
	// pluginKey is the one entry PROJECT's layout: lists, marking a project this tool scaffolded.
	pluginKey = pluginName + "/v2"
)

// newProjectStore returns the store for a project init is about to create. An
// existing PROJECT is refused: the one that loads is "already initialized", and
// any other failure to read it is reported as it is.
func newProjectStore(dst afero.Fs) (store.Store, error) {
	st := yaml.New(machinery.Filesystem{FS: dst})
	switch err := st.Load(); {
	case err == nil:
		if err := requireOwnLayout(st); err != nil {
			return nil, err
		}
		return nil, errors.New("already initialized: PROJECT exists; run init in an empty directory")
	case !errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("PROJECT exists but cannot be loaded: %w", err)
	}
	if err := st.New(cfgv3.Version); err != nil {
		return nil, fmt.Errorf("creating project configuration: %w", err)
	}
	if err := st.Config().SetPluginChain([]string{pluginKey}); err != nil {
		return nil, fmt.Errorf("setting PROJECT layout: %w", err)
	}
	return st, nil
}

// loadProjectStore loads the PROJECT of a project this tool scaffolded.
func loadProjectStore(dst afero.Fs) (store.Store, error) {
	st := yaml.New(machinery.Filesystem{FS: dst})
	if err := st.Load(); err != nil {
		return nil, fmt.Errorf("not a provider project (cannot load PROJECT): %w", err)
	}
	if err := requireOwnLayout(st); err != nil {
		return nil, err
	}
	return st, nil
}

// requireOwnLayout refuses a PROJECT whose layout is anything but exactly this
// generator's key. A project from another tool, or one with no layout, would
// otherwise read as a native provider and be rewritten as one.
func requireOwnLayout(st store.Store) error {
	if got := st.Config().GetPluginChain(); !slices.Equal(got, []string{pluginKey}) {
		return fmt.Errorf("PROJECT layout %v is not [%s]: not a project this tool scaffolded", got, pluginKey)
	}
	return nil
}
