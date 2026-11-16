package pane

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

func TestSendMouseIgnoredWithoutTracking(t *testing.T) {
	p := start(t, 30, 5, `cat -v`)
	time.Sleep(200 * time.Millisecond)
	if p.MouseTracking() {
		t.Fatal("no tracking expected")
	}
	if p.SendMouse(vt.MouseClick{X: 1, Y: 1, Button: vt.MouseLeft}) {
		t.Fatal("click should not be delivered")
	}
}

func TestSendMouseClickAndRelease(t *testing.T) {
	p := start(t, 30, 5, `printf '\033[?1000h\033[?1006h'; cat -v`)
	time.Sleep(300 * time.Millisecond)
	if !p.SendMouse(vt.MouseClick{X: 2, Y: 1, Button: vt.MouseLeft}) {
		t.Fatal("click should be delivered")
	}
	waitFor(t, p, 0, "^[[<0;3;2M")
	p.SendMouse(vt.MouseRelease{X: 2, Y: 1, Button: vt.MouseLeft})
	waitFor(t, p, 0, "^[[<0;3;2m")
}

func TestSendMouseMotionFiltering(t *testing.T) {
	// 1000: clicks only.
	p := start(t, 30, 5, `printf '\033[?1000h\033[?1006h'; cat -v`)
	time.Sleep(300 * time.Millisecond)
	if p.SendMouse(vt.MouseMotion{X: 1, Y: 1, Button: vt.MouseLeft}) {
		t.Fatal("drag must not be sent in mode 1000")
	}
	// 1002: drags but not hover.
	p = start(t, 30, 5, `printf '\033[?1002h\033[?1006h'; cat -v`)
	time.Sleep(300 * time.Millisecond)
	if p.SendMouse(vt.MouseMotion{X: 1, Y: 1, Button: vt.MouseNone}) {
		t.Fatal("hover must not be sent in mode 1002")
	}
	if !p.SendMouse(vt.MouseMotion{X: 1, Y: 1, Button: vt.MouseLeft}) {
		t.Fatal("drag should be sent in mode 1002")
	}
	s := waitFor(t, p, 0, "^[[<32;2;2M")
	if strings.Contains(s, "^[[<35") {
		t.Fatalf("unexpected hover event:\n%s", s)
	}
	// 1003: everything.
	p = start(t, 30, 5, `printf '\033[?1003h\033[?1006h'; cat -v`)
	time.Sleep(300 * time.Millisecond)
	if !p.SendMouse(vt.MouseMotion{X: 1, Y: 1, Button: vt.MouseNone}) {
		t.Fatal("hover should be sent in mode 1003")
	}
	waitFor(t, p, 0, "^[[<35;2;2M")
}
