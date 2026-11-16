package pane

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

// WheelStep is the number of lines (or arrow key presses) per wheel notch.
const WheelStep = 3

func isMouseMode(m ansi.Mode) bool {
	switch m {
	case ansi.ModeMouseX10, ansi.ModeMouseNormal, ansi.ModeMouseHighlight,
		ansi.ModeMouseButtonEvent, ansi.ModeMouseAnyEvent:
		return true
	}
	return false
}

// Wheel handles one wheel notch at (x, y), relative to the pane:
//   - a program tracking the mouse receives the wheel event;
//   - on the alternate screen (less, vim...) it receives arrow keys;
//   - on the normal screen the view scrolls through the history.
func (p *Pane) Wheel(x, y int, up bool) {
	p.emuMu.Lock()
	defer p.emuMu.Unlock()
	switch {
	case len(p.mouse) > 0:
		p.scroll = 0
		btn := vt.MouseWheelDown
		if up {
			btn = vt.MouseWheelUp
		}
		p.term.SendMouse(vt.MouseWheel{X: x, Y: y, Button: btn})
	case p.term.IsAltScreen():
		p.scroll = 0
		code := uv.KeyDown
		if up {
			code = uv.KeyUp
		}
		for i := 0; i < WheelStep; i++ {
			p.term.SendKey(vt.KeyPressEvent{Code: code})
		}
	default:
		if up {
			p.scroll += WheelStep
		} else {
			p.scroll -= WheelStep
		}
		if max := p.term.ScrollbackLen(); p.scroll > max {
			p.scroll = max
		}
		if p.scroll < 0 {
			p.scroll = 0
		}
	}
}

// ScrollOffset returns how many lines the view is scrolled back. It must be
// called inside WithTerm (or equivalent locking) when used with ViewCell.
func (p *Pane) ScrollOffset() int {
	if p.term.IsAltScreen() {
		return 0
	}
	if max := p.term.ScrollbackLen(); p.scroll > max {
		return max
	}
	return p.scroll
}

// ViewCell returns the cell shown at (x, y) when the view is scrolled back by
// off lines: history lines first, then the live screen.
func ViewCell(t *vt.Emulator, off, x, y int) *uv.Cell {
	if off <= 0 {
		return t.CellAt(x, y)
	}
	sb := t.ScrollbackLen()
	v := sb - off + y
	if v < sb {
		return t.ScrollbackCellAt(x, v)
	}
	return t.CellAt(x, v-sb)
}
