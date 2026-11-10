package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParsePrefix(t *testing.T) {
	for spec, want := range map[string]tcell.Key{
		"ctrl-a": tcell.KeyCtrlA, "C-b": tcell.KeyCtrlB,
		"ctrl-space": tcell.KeyCtrlSpace, `ctrl-\`: tcell.KeyCtrlBackslash,
	} {
		if got, err := parsePrefix(spec); err != nil || got != want {
			t.Errorf("%s: got %v, %v", spec, got, err)
		}
	}
	for _, bad := range []string{"", "alt-a", "ctrl-i", "ctrl-m", "ctrl-ab"} {
		if _, err := parsePrefix(bad); err == nil {
			t.Errorf("%q should fail", bad)
		}
	}
}
