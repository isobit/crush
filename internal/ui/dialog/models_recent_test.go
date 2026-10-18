package dialog

import (
	"testing"

	"charm.land/catwalk/pkg/catwalk"
	"github.com/charmbracelet/crush/internal/config"
	"github.com/charmbracelet/crush/internal/csync"
	"github.com/charmbracelet/crush/internal/ui/common"
	"github.com/charmbracelet/crush/internal/ui/styles"
	"github.com/charmbracelet/crush/internal/workspace"
	"github.com/stretchr/testify/require"
)

// staticConfigWorkspace serves a fixed config to the model switcher.
type staticConfigWorkspace struct {
	workspace.Workspace
	cfg *config.Config
}

func (w staticConfigWorkspace) Config() *config.Config { return w.cfg }

func newRecentsTestModels(t *testing.T, recents []config.SelectedModel, current config.SelectedModel) *Models {
	t.Helper()
	cfg := &config.Config{
		Providers: csync.NewMap[string, config.ProviderConfig](),
		Options: &config.Options{
			DisableProviderAutoUpdate: true,
			DisableDefaultProviders:   true,
		},
	}
	var models []catwalk.Model
	for _, id := range []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10"} {
		models = append(models, catwalk.Model{ID: id, Name: id})
	}
	cfg.Providers.Set("p1", config.ProviderConfig{ID: "p1", Name: "P1", Type: "openai", APIKey: "k", Models: models})
	cfg.Models = map[config.SelectedModelType]config.SelectedModel{config.SelectedModelTypeLarge: current}
	if recents != nil {
		cfg.RecentModels = map[config.SelectedModelType][]config.SelectedModel{config.SelectedModelTypeLarge: recents}
	}

	sty := styles.IsobitStyles()
	m, err := NewModels(&common.Common{Workspace: staticConfigWorkspace{cfg: cfg}, Styles: &sty}, false)
	require.NoError(t, err)
	return m
}

func TestModels_ScrollsToActiveModelWithoutRecents(t *testing.T) {
	t.Parallel()

	current := config.SelectedModel{Provider: "p1", Model: "a9"}
	m := newRecentsTestModels(t, nil, current)

	item, ok := m.list.SelectedItem().(*ModelItem)
	require.True(t, ok)
	require.Equal(t, "p1:a9", item.ID())
	require.True(t, m.scrollToSelected)
}
