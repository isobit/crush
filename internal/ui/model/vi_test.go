package model

import (
	"testing"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/crush/internal/ui/dialog"
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

func TestViNormalModeRoutesEditorKeys(t *testing.T) {
	t.Parallel()

	ui := newViTestUI("hello")
	ui.textarea.MoveToEnd()

	ui.handleKeyPressMsg(tea.KeyPressMsg{Code: 'h'})
	require.Equal(t, 4, ui.textarea.Column())

	ui.handleKeyPressMsg(tea.KeyPressMsg{Code: 'z'})
	require.Equal(t, "hello", ui.textarea.Value())
}

func TestViChangeMotions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  string
		column int
		keys   string
		want   string
	}{
		{name: "change inner word", value: "hello world", column: 1, keys: "ciw", want: " world"},
		{name: "change around word", value: "hello world", column: 1, keys: "caw", want: "world"},
		{name: "change word", value: "hello world", column: 1, keys: "cw", want: "h world"},
		{name: "change WORD", value: "hello-world next", column: 1, keys: "cW", want: "h next"},
		{name: "change to end", value: "hello world", column: 3, keys: "c$", want: "hel"},
		{name: "change to start", value: "hello world", column: 3, keys: "c0", want: "lo world"},
		{name: "change line", value: "hello\nworld", column: 2, keys: "cc", want: "\nworld"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ui := newViTestUI(tt.value)
			ui.textarea.SetCursorColumn(tt.column)
			for _, k := range tt.keys {
				ui.handleKeyPressMsg(tea.KeyPressMsg{Code: k})
			}

			require.Equal(t, tt.want, ui.textarea.Value())
			require.Equal(t, viInsert, ui.vi.mode)
		})
	}
}

func newViTestUI(value string) *UI {
	ta := textarea.New()
	ta.Focus()
	ta.SetValue(value)
	ta.MoveToBegin()
	return &UI{
		dialog:   dialog.NewOverlay(),
		focus:    uiFocusEditor,
		state:    uiLanding,
		textarea: ta,
		vi:       viState{enabled: true, mode: viNormal},
	}
}
