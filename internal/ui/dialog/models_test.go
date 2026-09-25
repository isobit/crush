package dialog

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

func TestModels_ToggleSetAsDefault(t *testing.T) {
	t.Parallel()

	m := &Models{}
	m.keyMap.Default = key.NewBinding(key.WithKeys("ctrl+d"))

	m.HandleMsg(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	require.True(t, m.setAsDefault)

	m.HandleMsg(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	require.False(t, m.setAsDefault)

	m.isOnboarding = true
	m.HandleMsg(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	require.False(t, m.setAsDefault)
}
