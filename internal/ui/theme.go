package ui

import "github.com/gdamore/tcell/v2"

// Palette taken from charm.land (Charm's brand colors).
var (
	colPepper   = tcell.NewHexColor(0x201F26) // dark background
	colCharcoal = tcell.NewHexColor(0x3A3943)
	colOyster   = tcell.NewHexColor(0x858392)
	colSmoke    = tcell.NewHexColor(0xBFBCC8)
	colCream    = tcell.NewHexColor(0xFFFDF5)
	colCharple  = tcell.NewHexColor(0x6B50FF) // purple
	colFuchsia  = tcell.NewHexColor(0xFF7BF5) // pink
	colJulep    = tcell.NewHexColor(0x00FFB2) // green
)

var (
	styleBar       = tcell.StyleDefault.Background(colCharcoal).Foreground(colSmoke)
	styleHost      = tcell.StyleDefault.Background(colCharple).Foreground(colCream).Bold(true)
	styleTabActive = tcell.StyleDefault.Background(colFuchsia).Foreground(colPepper).Bold(true)
	styleTab       = styleBar
	stylePrefix    = tcell.StyleDefault.Background(colJulep).Foreground(colPepper).Bold(true)

	styleHelpBar = tcell.StyleDefault.Background(colPepper).Foreground(colOyster)
	styleHelpKey = tcell.StyleDefault.Background(colPepper).Foreground(colFuchsia).Bold(true)
	styleHelpSep = tcell.StyleDefault.Background(colPepper).Foreground(colCharcoal)

	styleSparkCPU = tcell.StyleDefault.Background(colCharcoal).Foreground(colFuchsia)
	styleSparkMem = tcell.StyleDefault.Background(colCharcoal).Foreground(colJulep)

	styleBorder       = tcell.StyleDefault.Foreground(colOyster)
	styleBorderActive = tcell.StyleDefault.Foreground(colCharple).Bold(true)
)
