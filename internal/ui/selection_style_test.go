package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestSelectionStyle(t *testing.T) {
	base := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlue).
		Bold(true).Underline(true).Reverse(true).Dim(true)
	fg, bg, attrs := selectionStyle(base).Decompose()
	if fg != colCream || bg != colCharple {
		t.Errorf("colors = %v/%v, want %v/%v", fg, bg, colCream, colCharple)
	}
	if attrs&tcell.AttrBold == 0 || attrs&tcell.AttrUnderline == 0 {
		t.Errorf("bold/underline lost: %v", attrs)
	}
	if attrs&tcell.AttrReverse != 0 || attrs&tcell.AttrDim != 0 {
		t.Errorf("reverse/dim kept: %v", attrs)
	}
}
