// Package pane runs a program in a PTY and feeds its output to a VT emulator.
package pane

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

// Pane is one process with its emulated screen.
type Pane struct {
	// Name is a user-chosen label; empty means use the terminal title.
	// Only accessed from the UI goroutine.
	Name string

	ptmx *os.File
	cmd  *exec.Cmd

	// emuMu guards term and title: the emulator is not thread-safe.
	emuMu sync.Mutex
	term  *vt.Emulator
	title string
	// mouse holds the mouse-tracking modes the program enabled.
	mouse map[ansi.Mode]bool
	// scroll is how many lines the view is scrolled back into the history.
	scroll int

	mu         sync.Mutex
	cols, rows int
	closed     bool
	rec        *os.File // raw PTY output, when WTUI_RECORD is set
}

// New starts command (argv) in dir (empty = wtui's own directory); an empty
// command starts the user's shell.
// onOutput is called (from another goroutine) whenever the screen changed;
// onExit when the process terminated.
func New(command []string, dir string, cols, rows int, onOutput func(), onExit func(*Pane)) (*Pane, error) {
	if len(command) == 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		command = []string{shell}
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	p := &Pane{
		term:  vt.NewEmulator(cols, rows),
		mouse: map[ansi.Mode]bool{},
		ptmx:  ptmx,
		cmd:   cmd,
		cols:  cols,
		rows:  rows,
	}
	p.term.SetCallbacks(vt.Callbacks{
		// Called from term.Write, so emuMu is already held.
		Title: func(t string) { p.title = t },
		EnableMode: func(m ansi.Mode) {
			if isMouseMode(m) {
				p.mouse[m] = true
			}
		},
		DisableMode: func(m ansi.Mode) { delete(p.mouse, m) },
	})
	if path := os.Getenv("WTUI_RECORD"); path != "" {
		p.rec, _ = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	}

	// Answers from the emulator (device attributes, cursor position...)
	// must be sent back to the program, or the emulator blocks on its pipe.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := p.term.Read(buf)
			if n > 0 {
				ptmx.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if p.rec != nil {
					p.rec.Write(buf[:n])
				}
				p.emuMu.Lock()
				p.term.Write(buf[:n])
				p.emuMu.Unlock()
				onOutput()
			}
			if err != nil {
				break
			}
		}
		onExit(p)
	}()
	return p, nil
}

// Write sends input bytes to the program.
func (p *Pane) Write(b []byte) {
	if len(b) > 0 {
		p.emuMu.Lock()
		p.scroll = 0 // typing brings the view back to the bottom
		p.emuMu.Unlock()
		p.ptmx.Write(b)
	}
}

// WithTerm runs f with exclusive access to the emulator.
func (p *Pane) WithTerm(f func(t *vt.Emulator)) {
	p.emuMu.Lock()
	defer p.emuMu.Unlock()
	f(p.term)
}

// Title returns the title set by the program (empty if none).
func (p *Pane) Title() string {
	p.emuMu.Lock()
	defer p.emuMu.Unlock()
	return p.title
}

// Resize updates the emulator and PTY size if it changed. The emulator is
// resized first so that output produced in answer to SIGWINCH is parsed at
// the new size.
func (p *Pane) Resize(cols, rows int) {
	if cols < 1 || rows < 1 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if cols == p.cols && rows == p.rows {
		return
	}
	p.cols, p.rows = cols, rows
	p.emuMu.Lock()
	p.term.Resize(cols, rows)
	p.emuMu.Unlock()
	pty.Setsize(p.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Close kills the process and releases the PTY. Safe to call twice.
// Cwd returns the current working directory of the pane's process, or "" if
// it cannot be determined.
func (p *Pane) Cwd() string {
	if p.cmd == nil || p.cmd.Process == nil {
		return ""
	}
	d, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", p.cmd.Process.Pid))
	if err != nil {
		return ""
	}
	return d
}

func (p *Pane) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	p.ptmx.Close()
	if p.rec != nil {
		p.rec.Close()
	}
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	go p.cmd.Wait()
	p.term.Close() // unblocks the response goroutine
}
