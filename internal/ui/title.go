package ui

import (
	"os"
	"strings"
	"unicode"

	"window-tui/internal/pane"
	"window-tui/internal/window"
)

const hostTitleSuffix = " - wtui"

// paneTitle is the label of a pane: its name, else the title its program
// set, else "sh".
func paneTitle(p *pane.Pane) string {
	if p.Name != "" {
		return p.Name
	}
	if t := p.Title(); t != "" {
		return t
	}
	return "sh"
}

// sanitize strips control characters (a pane title comes from an untrusted
// program and is written to the host terminal), truncates it, and never
// returns an empty string.
func sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	if rs := []rune(s); len(rs) > 200 {
		s = string(rs[:200])
	}
	if strings.TrimSpace(s) == "" {
		return "sh"
	}
	return s
}

func hostTitle(p *pane.Pane) string {
	return sanitize(paneTitle(p)) + hostTitleSuffix
}

// syncTitle updates the host terminal's window title when it changed.
func (a *app) syncTitle(w *window.Window) {
	t := hostTitle(w.Active)
	if t == a.lastTitle {
		return
	}
	a.lastTitle = t
	if a.setTitle != nil {
		a.setTitle(t)
		return
	}
	writeHost("\x1b]2;" + t + "\x1b\\")
}

func writeHost(s string) { os.Stdout.WriteString(s) }
