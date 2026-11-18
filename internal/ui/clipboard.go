package ui

import (
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// clipboard copies text to the system clipboard (CLIPBOARD only, never
// PRIMARY). The dependencies are injectable for tests.
type clipboard struct {
	getenv   func(string) string
	lookPath func(string) (string, error)
	run      func(name string, args []string, input string) error
	osc52    func(seq string)
}

func newClipboard() *clipboard {
	return &clipboard{
		getenv:   os.Getenv,
		lookPath: exec.LookPath,
		run: func(name string, args []string, input string) error {
			cmd := exec.Command(name, args...)
			cmd.Stdin = strings.NewReader(input)
			return cmd.Run()
		},
		osc52: func(seq string) { os.Stdout.WriteString(seq) },
	}
}

type clipCmd struct {
	name string
	args []string
}

// commands lists the external tools to try, in order of preference.
func (c *clipboard) commands() []clipCmd {
	var cmds []clipCmd
	if c.getenv("WAYLAND_DISPLAY") != "" {
		cmds = append(cmds, clipCmd{"wl-copy", nil})
	}
	if c.getenv("DISPLAY") != "" {
		cmds = append(cmds,
			clipCmd{"xclip", []string{"-selection", "clipboard"}},
			clipCmd{"xsel", []string{"--clipboard", "--input"}},
		)
	}
	return cmds
}

// Copy puts text in the system clipboard. It uses the first available
// external tool and falls back to OSC 52 (wrapped in a DCS passthrough under
// tmux), which depends on the terminal supporting it.
func (c *clipboard) Copy(text string) {
	for _, cmd := range c.commands() {
		if _, err := c.lookPath(cmd.name); err != nil {
			continue
		}
		if c.run(cmd.name, cmd.args, text) == nil {
			return
		}
	}
	seq := ansi.SetSystemClipboard(text)
	if c.getenv("TMUX") != "" {
		seq = ansi.TmuxPassthrough(seq)
	}
	c.osc52(seq)
}
