// Package window holds the session: tabs (windows), each with a split tree of panes.
package window

import (
	"window-tui/internal/layout"
	"window-tui/internal/pane"
)

type Window struct {
	Name   string // optional tab name, shown before the active pane title
	Root   *layout.Node
	Active *pane.Pane
}

// Panes lists the window's panes in tree order.
func (w *Window) Panes() []*pane.Pane {
	var out []*pane.Pane
	for _, it := range w.Root.Leaves() {
		out = append(out, it.(*pane.Pane))
	}
	return out
}

type Session struct {
	Windows []*Window
	Cur     int
}

func (s *Session) Current() *Window {
	if len(s.Windows) == 0 {
		return nil
	}
	return s.Windows[s.Cur]
}

func (s *Session) Add(p *pane.Pane) {
	s.Windows = append(s.Windows, &Window{Root: layout.NewLeaf(p), Active: p})
	s.Cur = len(s.Windows) - 1
}

func (s *Session) Select(i int) {
	if i >= 0 && i < len(s.Windows) {
		s.Cur = i
	}
}

func (s *Session) Next(delta int) {
	if n := len(s.Windows); n > 0 {
		s.Cur = ((s.Cur+delta)%n + n) % n
	}
}

// RemoveWindow closes all panes of window i.
func (s *Session) RemoveWindow(i int) {
	for _, p := range s.Windows[i].Panes() {
		p.Close()
	}
	s.Windows = append(s.Windows[:i], s.Windows[i+1:]...)
	if s.Cur >= len(s.Windows) {
		s.Cur = len(s.Windows) - 1
	}
	if s.Cur < 0 {
		s.Cur = 0
	}
}

// ClosePane removes p wherever it lives (closing its window if it was the last pane).
func (s *Session) ClosePane(p *pane.Pane) {
	for i, w := range s.Windows {
		if w.Root.Find(p) == nil {
			continue
		}
		if w.Active == p {
			ps := w.Panes()
			for j, q := range ps {
				if q == p {
					if j > 0 {
						w.Active = ps[j-1]
					} else if len(ps) > 1 {
						w.Active = ps[1]
					}
				}
			}
		}
		if w.Root.Remove(p) {
			p.Close()
			s.RemoveWindow(i)
		} else {
			p.Close()
		}
		return
	}
}
