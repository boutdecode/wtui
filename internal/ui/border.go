package ui

import "window-tui/internal/layout"

// Arms of a border cell.
const (
	armUp uint8 = 1 << iota
	armDown
	armLeft
	armRight
)

type point struct{ x, y int }

// borderArms maps every separator cell to the directions it connects to, so
// that crossings and corners get proper junction glyphs (┌ ┬ ┼ ...).
func borderArms(seps []layout.Rect) map[point]uint8 {
	arms := map[point]uint8{}
	horiz := map[point]bool{}
	vert := map[point]bool{}
	for _, sp := range seps {
		for y := sp.Y; y < sp.Y+sp.H; y++ {
			for x := sp.X; x < sp.X+sp.W; x++ {
				p := point{x, y}
				if sp.H == 1 {
					horiz[p] = true
					arms[p] |= armLeft | armRight
				} else {
					vert[p] = true
					arms[p] |= armUp | armDown
				}
			}
		}
	}
	// A line ending against a perpendicular one adds an arm to it.
	for p := range horiz {
		if vert[point{p.x - 1, p.y}] {
			arms[point{p.x - 1, p.y}] |= armRight
		}
		if vert[point{p.x + 1, p.y}] {
			arms[point{p.x + 1, p.y}] |= armLeft
		}
	}
	for p := range vert {
		if horiz[point{p.x, p.y - 1}] {
			arms[point{p.x, p.y - 1}] |= armDown
		}
		if horiz[point{p.x, p.y + 1}] {
			arms[point{p.x, p.y + 1}] |= armUp
		}
	}
	return arms
}

var (
	lightGlyphs = map[uint8]rune{
		armUp: '│', armDown: '│', armUp | armDown: '│',
		armLeft: '─', armRight: '─', armLeft | armRight: '─',
		armDown | armRight: '┌', armDown | armLeft: '┐',
		armUp | armRight: '└', armUp | armLeft: '┘',
		armUp | armDown | armRight: '├', armUp | armDown | armLeft: '┤',
		armLeft | armRight | armDown: '┬', armLeft | armRight | armUp: '┴',
		armUp | armDown | armLeft | armRight: '┼',
	}
	heavyGlyphs = map[uint8]rune{
		armUp: '┃', armDown: '┃', armUp | armDown: '┃',
		armLeft: '━', armRight: '━', armLeft | armRight: '━',
		armDown | armRight: '┏', armDown | armLeft: '┓',
		armUp | armRight: '┗', armUp | armLeft: '┛',
		armUp | armDown | armRight: '┣', armUp | armDown | armLeft: '┫',
		armLeft | armRight | armDown: '┳', armLeft | armRight | armUp: '┻',
		armUp | armDown | armLeft | armRight: '╋',
	}
)

func borderGlyph(arms uint8, heavy bool) rune {
	if heavy {
		return heavyGlyphs[arms]
	}
	return lightGlyphs[arms]
}

// onRing reports whether (x, y) is on the one-cell ring around r, corners
// included: these are the border cells that belong to the pane.
func onRing(r layout.Rect, x, y int) bool {
	if x < r.X-1 || x > r.X+r.W || y < r.Y-1 || y > r.Y+r.H {
		return false
	}
	return x == r.X-1 || x == r.X+r.W || y == r.Y-1 || y == r.Y+r.H
}
