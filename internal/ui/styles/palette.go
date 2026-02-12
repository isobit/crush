package styles

import (
	"image/color"

	"github.com/charmbracelet/x/exp/charmtone"
)

// Palette defines the color values used to build a Styles.
type Palette struct {
	Primary   color.Color
	Secondary color.Color
	Tertiary  color.Color

	BgBase        color.Color
	BgBaseLighter color.Color
	BgSubtle      color.Color
	BgOverlay     color.Color

	FgBase      color.Color
	FgMuted     color.Color
	FgHalfMuted color.Color
	FgSubtle    color.Color

	Border      color.Color
	BorderFocus color.Color

	Error   color.Color
	Warning color.Color
	Info    color.Color

	White      color.Color
	BlueLight  color.Color
	Blue       color.Color
	BlueDark   color.Color
	Yellow     color.Color
	GreenLight color.Color
	Green      color.Color
	GreenDark  color.Color
	Red        color.Color
	RedDark    color.Color
}

// DefaultPalette returns the default charmtone color palette.
func DefaultPalette() Palette {
	return Palette{
		Primary:   charmtone.Charple,
		Secondary: charmtone.Dolly,
		Tertiary:  charmtone.Bok,

		BgBase:        charmtone.Pepper,
		BgBaseLighter: charmtone.BBQ,
		BgSubtle:      charmtone.Charcoal,
		BgOverlay:     charmtone.Iron,

		FgBase:      charmtone.Ash,
		FgMuted:     charmtone.Squid,
		FgHalfMuted: charmtone.Smoke,
		FgSubtle:    charmtone.Oyster,

		Border:      charmtone.Charcoal,
		BorderFocus: charmtone.Charple,

		Error:   charmtone.Sriracha,
		Warning: charmtone.Zest,
		Info:    charmtone.Malibu,

		White:      charmtone.Butter,
		BlueLight:  charmtone.Sardine,
		Blue:       charmtone.Malibu,
		BlueDark:   charmtone.Damson,
		Yellow:     charmtone.Mustard,
		GreenLight: charmtone.Bok,
		Green:      charmtone.Julep,
		GreenDark:  charmtone.Guac,
		Red:        charmtone.Coral,
		RedDark:    charmtone.Sriracha,
	}
}
