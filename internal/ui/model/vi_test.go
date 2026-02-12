package model

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

func TestViState(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()
		vs := viState{}
		require.False(t, vs.enabled)
		require.Equal(t, viInsert, vs.mode)
	})

	t.Run("enabled starts in normal mode after init", func(t *testing.T) {
		t.Parallel()
		vs := viState{enabled: true, mode: viNormal}
		require.True(t, vs.enabled)
		require.Equal(t, viNormal, vs.mode)
	})
}

func TestViModeIndicator(t *testing.T) {
	t.Parallel()

	t.Run("disabled returns empty", func(t *testing.T) {
		t.Parallel()
		ui := &UI{vi: viState{enabled: false}}
		require.Empty(t, ui.viModeIndicator())
	})

	t.Run("normal mode", func(t *testing.T) {
		t.Parallel()
		ui := &UI{vi: viState{enabled: true, mode: viNormal}}
		require.Equal(t, "NORMAL", ui.viModeIndicator())
	})

	t.Run("insert mode", func(t *testing.T) {
		t.Parallel()
		ui := &UI{vi: viState{enabled: true, mode: viInsert}}
		require.Equal(t, "INSERT", ui.viModeIndicator())
	})

	t.Run("pending command", func(t *testing.T) {
		t.Parallel()
		ui := &UI{vi: viState{enabled: true, mode: viNormal, pending: "d"}}
		require.Equal(t, "d", ui.viModeIndicator())
	})
}

func TestViHelpers(t *testing.T) {
	t.Parallel()

	t.Run("viEnabled", func(t *testing.T) {
		t.Parallel()
		ui := &UI{vi: viState{enabled: false}}
		require.False(t, ui.viEnabled())

		ui.vi.enabled = true
		require.True(t, ui.viEnabled())
	})

	t.Run("viIsNormal", func(t *testing.T) {
		t.Parallel()
		ui := &UI{vi: viState{enabled: true, mode: viInsert}}
		require.False(t, ui.viIsNormal())

		ui.vi.mode = viNormal
		require.True(t, ui.viIsNormal())

		ui.vi.enabled = false
		require.False(t, ui.viIsNormal())
	})
}

func TestViKeyMsg(t *testing.T) {
	t.Parallel()

	msg := viKeyMsg(tea.KeyLeft, 0)
	require.Equal(t, tea.KeyLeft, msg.Code)
	require.Equal(t, tea.KeyMod(0), msg.Mod)

	msg = viKeyMsg(tea.KeyRight, tea.ModAlt)
	require.Equal(t, tea.KeyRight, msg.Code)
	require.Equal(t, tea.ModAlt, msg.Mod)
}
