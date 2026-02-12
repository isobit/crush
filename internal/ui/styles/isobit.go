package styles

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

// IsobitPalette returns a custom palette with darker backgrounds and blue accents.
func IsobitPalette() Palette {
	blue := lipgloss.Color("#2475f4")

	p := DefaultPalette()
	p.Secondary = blue
	p.BgBase = lipgloss.Color("#000")
	p.BgBaseLighter = lipgloss.Color("#111")
	p.BgSubtle = lipgloss.Color("#222")
	p.BgOverlay = lipgloss.Color("#333")
	p.BorderFocus = blue

	return p
}

// IsobitStyles returns the isobit theme styles.
func IsobitStyles() Styles {
	p := IsobitPalette()
	s := NewStyles(p)

	s.TextSelection = lipgloss.NewStyle().Foreground(charmtone.Salt).Background(p.Secondary)

	s.ResourceOfflineIcon = lipgloss.NewStyle().Foreground(charmtone.Squid).SetString("●")
	s.ResourceBusyIcon = s.ResourceOfflineIcon.Foreground(charmtone.Citron)
	s.ResourceErrorIcon = s.ResourceOfflineIcon.Foreground(charmtone.Coral)
	s.ResourceOnlineIcon = s.ResourceOfflineIcon.Foreground(charmtone.Guac)

	s.EditorPromptYoloIconFocused = lipgloss.NewStyle().MarginRight(1).Foreground(charmtone.Oyster).Background(charmtone.Citron).Bold(true).SetString(" ! ")
	s.EditorPromptYoloIconBlurred = s.EditorPromptYoloIconFocused.Foreground(charmtone.Pepper).Background(charmtone.Squid)
	s.EditorPromptYoloDotsFocused = lipgloss.NewStyle().MarginRight(1).Foreground(charmtone.Zest).SetString(":::")
	s.EditorPromptYoloDotsBlurred = s.EditorPromptYoloDotsFocused.Foreground(charmtone.Squid)

	return s
}
