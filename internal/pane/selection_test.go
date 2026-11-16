package pane

import (
	"testing"

	"github.com/charmbracelet/x/vt"
)

func TestSelectionContains(t *testing.T) {
	s := Selection{AX: 5, AY: 1, BX: 2, BY: 3} // backwards drag
	for _, c := range []struct {
		x, y int
		want bool
	}{{2, 1, false}, {5, 1, true}, {0, 2, true}, {79, 2, true}, {2, 3, true}, {3, 3, false}, {9, 0, false}, {0, 4, false}} {
		if got := s.Contains(c.x, c.y); got != c.want {
			t.Errorf("(%d,%d) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}

func TestSelectionText(t *testing.T) {
	e := vt.NewEmulator(20, 4)
	e.Write([]byte("hello world\r\nsecond line  \r\n日本語"))
	for _, c := range []struct {
		s    Selection
		want string
	}{
		{Selection{AX: 6, AY: 0, BX: 10, BY: 0}, "world"},
		{Selection{AX: 6, AY: 0, BX: 5, BY: 1}, "world\nsecond"},
		{Selection{AX: 5, AY: 1, BX: 6, BY: 0}, "world\nsecond"}, // reversed
		{Selection{AX: 0, AY: 2, BX: 5, BY: 2}, "日本語"},
	} {
		if got := c.s.Text(e, 0, 20); got != c.want {
			t.Errorf("%+v = %q, want %q", c.s, got, c.want)
		}
	}
}
