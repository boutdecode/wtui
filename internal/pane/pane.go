// Package pane runs a shell in a PTY and feeds its output to a VT emulator.
package pane

import (
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// Pane is one shell process with its emulated screen.
type Pane struct {
	Term vt10x.Terminal

	ptmx *os.File
	cmd  *exec.Cmd

	mu         sync.Mutex
	cols, rows int
	closed     bool
}

// New starts the user's shell. onOutput is called (from another goroutine)
// whenever the screen changed; onExit when the shell terminated.
func New(cols, rows int, onOutput func(), onExit func(*Pane)) (*Pane, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	p := &Pane{
		Term: vt10x.New(vt10x.WithSize(cols, rows), vt10x.WithWriter(ptmx)),
		ptmx: ptmx,
		cmd:  cmd,
		cols: cols,
		rows: rows,
	}
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				p.Term.Write(buf[:n])
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

// Write sends input bytes to the shell.
func (p *Pane) Write(b []byte) {
	if len(b) > 0 {
		p.ptmx.Write(b)
	}
}

// Resize updates the PTY and emulator size if it changed.
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
	pty.Setsize(p.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	p.Term.Resize(cols, rows)
}

// Close kills the shell and releases the PTY. Safe to call twice.
func (p *Pane) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()
	p.ptmx.Close()
	if p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	go p.cmd.Wait()
}
