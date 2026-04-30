package styles

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

// IsobitStyles returns the isobit theme styles, built on top of the quickStyle
// system with darker backgrounds and blue accents.
func IsobitStyles() Styles {
	blue := lipgloss.Color("#2475f4")

	s := quickStyle(quickStyleOpts{
		primary:   charmtone.Charple,
		secondary: blue,
		accent:    charmtone.Bok,
		keyword:   charmtone.Blush,

		fgBase:       charmtone.Ash,
		fgMoreSubtle: charmtone.Squid,
		fgSubtle:     charmtone.Smoke,
		fgMostSubtle: charmtone.Oyster,

		onPrimary: charmtone.Butter,

		bgBase:         lipgloss.Color("#000"),
		bgLeastVisible: lipgloss.Color("#111"),
		bgLessVisible:  lipgloss.Color("#222"),
		bgMostVisible:  lipgloss.Color("#333"),

		separator: charmtone.Charcoal,

		destructive:       charmtone.Coral,
		error:             charmtone.Sriracha,
		warningSubtle:     charmtone.Zest,
		warning:           charmtone.Mustard,
		busy:              charmtone.Citron,
		info:              charmtone.Malibu,
		infoMoreSubtle:    charmtone.Sardine,
		infoMostSubtle:    charmtone.Damson,
		success:           charmtone.Julep,
		successMoreSubtle: charmtone.Bok,
		successMostSubtle: charmtone.Guac,
	})

	s.TextSelection = lipgloss.NewStyle().Foreground(charmtone.Salt).Background(blue)
	s.TextInput.Cursor.Shape = tea.CursorBar
	s.Editor.Textarea.Cursor.Shape = tea.CursorBar

	return s
}
