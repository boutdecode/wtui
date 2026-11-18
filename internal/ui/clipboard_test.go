package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

type clipRun struct {
	name  string
	args  []string
	input string
}

func testClipboard(env map[string]string, installed map[string]bool, runErr error) (*clipboard, *[]clipRun, *[]string) {
	var runs []clipRun
	var osc []string
	return &clipboard{
		getenv: func(k string) string { return env[k] },
		lookPath: func(n string) (string, error) {
			if installed[n] {
				return "/usr/bin/" + n, nil
			}
			return "", errors.New("not found")
		},
		run: func(name string, args []string, input string) error {
			runs = append(runs, clipRun{name, args, input})
			return runErr
		},
		osc52: func(s string) { osc = append(osc, s) },
	}, &runs, &osc
}

func TestClipboardWayland(t *testing.T) {
	c, runs, osc := testClipboard(map[string]string{"WAYLAND_DISPLAY": "wayland-0"}, map[string]bool{"wl-copy": true}, nil)
	c.Copy("hello")
	if len(*runs) != 1 || (*runs)[0].name != "wl-copy" || (*runs)[0].input != "hello" || len(*osc) != 0 {
		t.Fatalf("runs=%v osc=%v", *runs, *osc)
	}
}

func TestClipboardX11OnlyClipboardSelection(t *testing.T) {
	c, runs, osc := testClipboard(map[string]string{"DISPLAY": ":0"}, map[string]bool{"xclip": true}, nil)
	c.Copy("hello")
	if len(*runs) != 1 || len(*osc) != 0 {
		t.Fatalf("want a single command, got runs=%v osc=%v", *runs, *osc)
	}
	r := (*runs)[0]
	if r.name != "xclip" || len(r.args) != 2 || r.args[1] != "clipboard" {
		t.Fatalf("unexpected command %v", r)
	}
	for _, a := range r.args {
		if strings.Contains(a, "primary") {
			t.Fatalf("PRIMARY must not be used: %v", r.args)
		}
	}
}

func TestClipboardXselFallback(t *testing.T) {
	c, runs, _ := testClipboard(map[string]string{"DISPLAY": ":0"}, map[string]bool{"xsel": true}, nil)
	c.Copy("x")
	if len(*runs) != 1 || (*runs)[0].name != "xsel" || (*runs)[0].args[0] != "--clipboard" {
		t.Fatalf("runs=%v", *runs)
	}
}

func TestClipboardOSC52Fallback(t *testing.T) {
	c, runs, osc := testClipboard(map[string]string{}, nil, nil)
	c.Copy("hello")
	if len(*runs) != 0 || len(*osc) != 1 || !strings.Contains((*osc)[0], "\x1b]52;") || strings.HasPrefix((*osc)[0], "\x1bP") {
		t.Fatalf("runs=%v osc=%q", *runs, *osc)
	}
}

func TestClipboardOSC52Tmux(t *testing.T) {
	c, _, osc := testClipboard(map[string]string{"TMUX": "/tmp/tmux"}, nil, nil)
	c.Copy("hello")
	if len(*osc) != 1 || !strings.HasPrefix((*osc)[0], "\x1bPtmux;") {
		t.Fatalf("osc=%q", *osc)
	}
}

func TestClipboardCommandFailureFallsBack(t *testing.T) {
	c, runs, osc := testClipboard(map[string]string{"DISPLAY": ":0"}, map[string]bool{"xclip": true}, errors.New("boom"))
	c.Copy("hello")
	if len(*runs) != 1 || len(*osc) != 1 {
		t.Fatalf("runs=%v osc=%v", *runs, *osc)
	}
}

func TestShouldSelect(t *testing.T) {
	cases := []struct {
		tracking bool
		mod      tcell.ModMask
		want     bool
	}{
		{false, 0, true},
		{true, 0, false},
		{true, tcell.ModShift, true},
		{true, tcell.ModAlt, true},
		{true, tcell.ModCtrl, false},
		{true, tcell.ModCtrl | tcell.ModShift, true},
	}
	for _, c := range cases {
		if got := shouldSelect(c.tracking, c.mod); got != c.want {
			t.Errorf("shouldSelect(%v,%v)=%v want %v", c.tracking, c.mod, got, c.want)
		}
	}
}
