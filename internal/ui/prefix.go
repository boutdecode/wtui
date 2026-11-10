package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// parsePrefix turns "ctrl-a", "ctrl-space", "ctrl-\" ... into a tcell key.
func parsePrefix(spec string) (tcell.Key, error) {
	s := strings.ToLower(strings.TrimSpace(spec))
	s = strings.TrimPrefix(strings.TrimPrefix(s, "c-"), "ctrl-")
	switch s {
	case "space", "@":
		return tcell.KeyCtrlSpace, nil
	case `\`:
		return tcell.KeyCtrlBackslash, nil
	case "]":
		return tcell.KeyCtrlRightSq, nil
	case "^":
		return tcell.KeyCtrlCarat, nil
	case "_":
		return tcell.KeyCtrlUnderscore, nil
	}
	if len(s) == 1 && s[0] >= 'a' && s[0] <= 'z' {
		k := tcell.Key(s[0]-'a') + tcell.KeyCtrlA
		switch k {
		case tcell.KeyTab, tcell.KeyEnter, tcell.KeyBackspace, tcell.KeyCtrlJ:
			return 0, fmt.Errorf("prefix %q conflicts with Tab/Enter/Backspace", spec)
		}
		return k, nil
	}
	return 0, fmt.Errorf("invalid prefix %q (try ctrl-a, ctrl-space, ctrl-\\)", spec)
}
