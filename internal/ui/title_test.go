package ui

import (
	"strings"
	"testing"
	"time"

	"window-tui/internal/pane"
	"window-tui/internal/window"
)

func newPane(t *testing.T, script string) *pane.Pane {
	t.Helper()
	p, err := pane.New([]string{"sh", "-c", script}, "", 20, 5, func() {}, func(*pane.Pane) {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func waitTitle(t *testing.T, p *pane.Pane, want string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if p.Title() == want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("title = %q, want %q", p.Title(), want)
}

func TestPaneAndHostTitle(t *testing.T) {
	p := newPane(t, `printf '\033]2;vim\007'; sleep 5`)
	waitTitle(t, p, "vim")
	if got := hostTitle(p); got != "vim - wtui" {
		t.Fatalf("hostTitle = %q", got)
	}
	p.Name = "work"
	if got := paneTitle(p); got != "work" {
		t.Fatalf("paneTitle = %q", got)
	}
	q := newPane(t, `sleep 5`)
	if got := hostTitle(q); got != "sh - wtui" {
		t.Fatalf("fallback = %q", got)
	}
}

func TestSanitize(t *testing.T) {
	if got := sanitize("a\x1b]0;x\x07b\nc\x7f\u009d"); got != "a]0;xbc" {
		t.Fatalf("got %q", got)
	}
	if got := sanitize(strings.Repeat("é", 500)); len([]rune(got)) != 200 {
		t.Fatalf("len = %d", len([]rune(got)))
	}
	for _, e := range []string{"", "  ", "\x1b\x07"} {
		if got := sanitize(e); got != "sh" {
			t.Fatalf("sanitize(%q) = %q", e, got)
		}
	}
}

func TestSyncTitle(t *testing.T) {
	p1 := newPane(t, `sleep 5`)
	p2 := newPane(t, `printf '\033]2;two\007'; sleep 5`)
	waitTitle(t, p2, "two")
	w := &window.Window{Active: p1}
	var got []string
	a := &app{setTitle: func(s string) { got = append(got, s) }}

	a.syncTitle(w)
	a.syncTitle(w)
	if len(got) != 1 || got[0] != "sh - wtui" {
		t.Fatalf("got %q", got)
	}
	w.Active = p2
	a.syncTitle(w)
	if len(got) != 2 || got[1] != "two - wtui" {
		t.Fatalf("got %q", got)
	}
	p2.Name = "renamed"
	a.syncTitle(w)
	if len(got) != 3 || got[2] != "renamed - wtui" {
		t.Fatalf("got %q", got)
	}
}
