package ui

import (
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
	"github.com/gdamore/tcell/v2"

	"window-tui/internal/layout"
)

const wheelMask = tcell.WheelUp | tcell.WheelDown | tcell.WheelLeft | tcell.WheelRight

type mouseKind int

const (
	mouseHover mouseKind = iota
	mousePress
	mouseDrag
	mouseRelease
)

// mouseTransition compares the button state before and after an event and
// returns what happened and to which button. Wheel bits must be stripped first.
func mouseTransition(prev, cur tcell.ButtonMask) (mouseKind, tcell.ButtonMask) {
	switch {
	case cur&^prev != 0:
		return mousePress, lowestButton(cur &^ prev)
	case prev&^cur != 0:
		return mouseRelease, lowestButton(prev &^ cur)
	case cur != 0:
		return mouseDrag, lowestButton(cur)
	}
	return mouseHover, 0
}

func lowestButton(m tcell.ButtonMask) tcell.ButtonMask { return m & -m }

// toVTButton converts a single tcell button (tcell's Button2 is the right one,
// Button3 the middle one).
func toVTButton(b tcell.ButtonMask) vt.MouseButton {
	switch b {
	case tcell.Button1:
		return vt.MouseLeft
	case tcell.Button2:
		return vt.MouseRight
	case tcell.Button3:
		return vt.MouseMiddle
	}
	return vt.MouseNone
}

func toVTMod(m tcell.ModMask) uv.KeyMod {
	var mod uv.KeyMod
	if m&tcell.ModShift != 0 {
		mod |= uv.ModShift
	}
	if m&tcell.ModAlt != 0 {
		mod |= uv.ModAlt
	}
	if m&tcell.ModCtrl != 0 {
		mod |= uv.ModCtrl
	}
	return mod
}

// localPos converts screen coordinates to coordinates relative to r, clamped
// inside it (a drag may leave the pane where it started).
func localPos(r layout.Rect, x, y int) (int, int) {
	x, y = x-r.X, y-r.Y
	x = max(0, min(x, r.W-1))
	y = max(0, min(y, r.H-1))
	return x, y
}

// newMouseEvent builds the event to forward for a transition.
func newMouseEvent(kind mouseKind, x, y int, btn vt.MouseButton, mod uv.KeyMod) vt.Mouse {
	switch kind {
	case mousePress:
		return vt.MouseClick{X: x, Y: y, Button: btn, Mod: mod}
	case mouseRelease:
		return vt.MouseRelease{X: x, Y: y, Button: btn, Mod: mod}
	}
	return vt.MouseMotion{X: x, Y: y, Button: btn, Mod: mod}
}
