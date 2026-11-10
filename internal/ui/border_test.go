package ui

import (
	"testing"

	"window-tui/internal/layout"
)

func TestBorderJunctions(t *testing.T) {
	// Vertical line at x=5 (rows 0-9), horizontal line at y=4 from x=6 to 11
	// ending against it: a T junction at (5,4).
	seps := []layout.Rect{{X: 5, Y: 0, W: 1, H: 10}, {X: 6, Y: 4, W: 6, H: 1}}
	arms := borderArms(seps)
	if g := borderGlyph(arms[point{5, 4}], false); g != '├' {
		t.Fatalf("junction = %q", g)
	}
	if g := borderGlyph(arms[point{5, 3}], false); g != '│' {
		t.Fatalf("plain vertical = %q", g)
	}
	if g := borderGlyph(arms[point{8, 4}], true); g != '━' {
		t.Fatalf("heavy horizontal = %q", g)
	}
}

func TestOnRingIncludesCorners(t *testing.T) {
	r := layout.Rect{X: 2, Y: 2, W: 3, H: 3}
	for _, p := range []point{{1, 1}, {5, 1}, {1, 5}, {5, 5}, {3, 1}, {1, 3}} {
		if !onRing(r, p.x, p.y) {
			t.Errorf("%v should be on ring", p)
		}
	}
	if onRing(r, 3, 3) || onRing(r, 7, 3) {
		t.Error("inside/outside cells must not be on ring")
	}
}

func TestSparkline(t *testing.T) {
	if got := sparkline([]float64{0, 50, 100}, 5); got != "   ▄█" {
		t.Fatalf("sparkline = %q", got)
	}
}
