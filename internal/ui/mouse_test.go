package ui

import (
	"testing"

	"github.com/charmbracelet/x/vt"
	"github.com/gdamore/tcell/v2"

	"window-tui/internal/layout"
)

func TestPaneAt(t *testing.T) {
	rects := map[layout.Item]layout.Rect{
		"left":  {X: 0, Y: 0, W: 10, H: 5},
		"right": {X: 11, Y: 0, W: 10, H: 5}, // column 10 is a separator
	}
	for _, c := range []struct {
		x, y int
		want layout.Item
	}{{0, 0, "left"}, {9, 4, "left"}, {11, 0, "right"}, {20, 4, "right"}, {10, 2, nil}, {5, 5, nil}, {30, 1, nil}} {
		if got := paneAt(rects, c.x, c.y); got != c.want {
			t.Errorf("(%d,%d) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}

func TestMouseTransition(t *testing.T) {
	for _, c := range []struct {
		prev, cur tcell.ButtonMask
		kind      mouseKind
		btn       tcell.ButtonMask
	}{
		{0, tcell.Button1, mousePress, tcell.Button1},
		{tcell.Button1, tcell.Button1, mouseDrag, tcell.Button1},
		{tcell.Button1, 0, mouseRelease, tcell.Button1},
		{0, 0, mouseHover, 0},
		{tcell.Button1, tcell.Button1 | tcell.Button2, mousePress, tcell.Button2},
	} {
		kind, btn := mouseTransition(c.prev, c.cur)
		if kind != c.kind || btn != c.btn {
			t.Errorf("%v->%v = (%v,%v), want (%v,%v)", c.prev, c.cur, kind, btn, c.kind, c.btn)
		}
	}
}

func TestToVTButton(t *testing.T) {
	if toVTButton(tcell.Button1) != vt.MouseLeft ||
		toVTButton(tcell.Button2) != vt.MouseRight ||
		toVTButton(tcell.Button3) != vt.MouseMiddle ||
		toVTButton(0) != vt.MouseNone {
		t.Fatal("wrong button mapping")
	}
}

func TestLocalPos(t *testing.T) {
	r := layout.Rect{X: 11, Y: 2, W: 10, H: 5}
	for _, c := range []struct{ x, y, wx, wy int }{
		{11, 2, 0, 0}, {15, 4, 4, 2}, {40, 30, 9, 4}, {0, 0, 0, 0},
	} {
		if x, y := localPos(r, c.x, c.y); x != c.wx || y != c.wy {
			t.Errorf("(%d,%d) = (%d,%d), want (%d,%d)", c.x, c.y, x, y, c.wx, c.wy)
		}
	}
}
