package pane

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

// start runs script in a pane and returns it once ready() is true.
func start(t *testing.T, cols, rows int, script string) *Pane {
	t.Helper()
	p, err := New([]string{"sh", "-c", script}, "", cols, rows, func() {}, func(*Pane) {})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func screen(p *Pane, off int) string {
	var sb strings.Builder
	p.WithTerm(func(e *vt.Emulator) {
		for y := 0; y < e.Height(); y++ {
			for x := 0; x < e.Width(); x++ {
				if c := ViewCell(e, off, x, y); c != nil && c.Content != "" {
					sb.WriteString(c.Content)
				} else {
					sb.WriteByte(' ')
				}
			}
			sb.WriteByte('\n')
		}
	})
	return sb.String()
}

func waitFor(t *testing.T, p *Pane, off int, want string) string {
	t.Helper()
	var s string
	for i := 0; i < 100; i++ {
		if s = screen(p, off); strings.Contains(s, want) {
			return s
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("%q not found in:\n%s", want, s)
	return s
}

func TestWheelScrollsHistory(t *testing.T) {
	p := start(t, 10, 5, `seq 1 100; sleep 5`)
	waitFor(t, p, 0, "100")
	if strings.Contains(screen(p, 0), "90") {
		t.Fatal("live view should only show the last lines")
	}
	p.Wheel(0, 0, true) // up one notch: 3 lines back
	if got := p.scrollNow(); got != WheelStep {
		t.Fatalf("scroll = %d", got)
	}
	view := screen(p, WheelStep)
	if !strings.HasPrefix(view, "9") || strings.Contains(view, "100") {
		t.Fatalf("scrolled view:\n%s", view)
	}
	p.Wheel(0, 0, false)
	if p.scrollNow() != 0 {
		t.Fatal("wheel down should return to the bottom")
	}
	p.Wheel(0, 0, true)
	p.Write([]byte("x")) // typing resets the view
	if p.scrollNow() != 0 {
		t.Fatal("typing should reset scroll")
	}
}

func TestWheelForwardedToMouseProgram(t *testing.T) {
	p := start(t, 30, 5, `printf '\033[?1000h\033[?1006h'; cat -v`)
	time.Sleep(300 * time.Millisecond) // let the program enable mouse tracking
	p.Wheel(2, 1, true)
	waitFor(t, p, 0, "^[[<64;3;2M")
}

func TestWheelSendsArrowsOnAltScreen(t *testing.T) {
	p := start(t, 30, 5, `printf '\033[?1049h'; cat -v`)
	time.Sleep(300 * time.Millisecond)
	p.Wheel(0, 0, false)
	// Wait for all arrows: the first one can show up before the others.
	s := waitFor(t, p, 0, strings.Repeat("^[[B", WheelStep))
	if strings.Count(s, "^[[B") != WheelStep {
		t.Fatalf("want %d arrows:\n%s", WheelStep, s)
	}
}

func (p *Pane) scrollNow() int {
	p.emuMu.Lock()
	defer p.emuMu.Unlock()
	return p.scroll
}
