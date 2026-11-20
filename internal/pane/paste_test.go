package pane

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

// pasteScreen runs script, waits for the program to enable (or not) bracketed
// paste, pastes text and returns the screen once the program exits.
func pasteScreen(t *testing.T, script, text string) []string {
	t.Helper()
	done := make(chan struct{})
	p, err := New([]string{"sh", "-c", script}, "", 80, 5, func() {}, func(*Pane) { close(done) })
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	time.Sleep(300 * time.Millisecond) // let the program print its mode switch
	p.Paste([]byte(text))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
	var out []string
	p.WithTerm(func(e *vt.Emulator) {
		for y := 0; y < e.Height(); y++ {
			var sb strings.Builder
			for x := 0; x < e.Width(); x++ {
				if c := e.CellAt(x, y); c != nil && c.Content != "" {
					sb.WriteString(c.Content)
				}
			}
			out = append(out, sb.String())
		}
	})
	return out
}

func TestPasteRaw(t *testing.T) {
	out := pasteScreen(t, `stty raw -echo; head -c 5 | od -An -c`, "hello")
	if !strings.Contains(strings.Join(out, " "), "h   e   l   l   o") {
		t.Fatalf("raw paste mangled: %q", out)
	}
}

func TestPasteBracketed(t *testing.T) {
	out := pasteScreen(t, `printf '\033[?2004h'; stty raw -echo; head -c 17 | od -An -c`, "hello")
	// ESC [ 2 0 0 ~ h e l l o ... wrapped in markers
	if !strings.Contains(strings.Join(out, " "), "033   [   2   0   0   ~   h   e   l   l   o") {
		t.Fatalf("bracketed markers missing: %q", out)
	}
}
