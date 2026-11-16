package pane

import (
	"strings"

	"github.com/charmbracelet/x/vt"
)

// Selection is a text selection in pane view coordinates (anchor A, end B).
type Selection struct{ AX, AY, BX, BY int }

// Empty reports whether the selection covers nothing.
func (s Selection) Empty() bool { return s.AX == s.BX && s.AY == s.BY }

// ordered returns the start and end in reading order.
func (s Selection) ordered() (x0, y0, x1, y1 int) {
	if s.AY > s.BY || (s.AY == s.BY && s.AX > s.BX) {
		return s.BX, s.BY, s.AX, s.AY
	}
	return s.AX, s.AY, s.BX, s.BY
}

// Contains reports whether the cell (x, y) is selected (inclusive range,
// reading order like a terminal selection).
func (s Selection) Contains(x, y int) bool {
	x0, y0, x1, y1 := s.ordered()
	if y < y0 || y > y1 {
		return false
	}
	if y == y0 && x < x0 {
		return false
	}
	if y == y1 && x > x1 {
		return false
	}
	return true
}

// Text extracts the selected text from the view scrolled back by off lines,
// width cells wide. Trailing blanks of each line are dropped.
func (s Selection) Text(t *vt.Emulator, off, width int) string {
	_, y0, _, y1 := s.ordered()
	var lines []string
	for y := y0; y <= y1; y++ {
		var b strings.Builder
		for x := 0; x < width; x++ {
			if !s.Contains(x, y) {
				continue
			}
			c := ViewCell(t, off, x, y)
			if c == nil || (c.Width == 0 && c.Content == "") {
				if c == nil {
					b.WriteByte(' ')
				}
				continue
			}
			if c.Content == "" {
				b.WriteByte(' ')
			} else {
				b.WriteString(c.Content)
			}
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return strings.Join(lines, "\n")
}
