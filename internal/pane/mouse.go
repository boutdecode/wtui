package pane

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

// MouseTracking reports whether the program asked for mouse events.
func (p *Pane) MouseTracking() bool {
	p.emuMu.Lock()
	defer p.emuMu.Unlock()
	return len(p.mouse) > 0
}

// SendMouse forwards a click, release or motion event (coordinates relative to
// the pane) to the program when it tracks the mouse. It reports whether the
// event was delivered; false means the program does not want it.
func (p *Pane) SendMouse(m vt.Mouse) bool {
	p.emuMu.Lock()
	defer p.emuMu.Unlock()
	if len(p.mouse) == 0 {
		return false
	}
	if mot, ok := m.(vt.MouseMotion); ok {
		switch {
		case mot.Button != vt.MouseNone:
			// Dragging: reported by the button-event (1002) and any-event (1003) modes.
			if !p.mouse[ansi.ModeMouseButtonEvent] && !p.mouse[ansi.ModeMouseAnyEvent] {
				return false
			}
		default:
			// Plain hover: only the any-event (1003) mode wants it.
			if !p.mouse[ansi.ModeMouseAnyEvent] {
				return false
			}
		}
	}
	p.scroll = 0
	p.term.SendMouse(m)
	return true
}
