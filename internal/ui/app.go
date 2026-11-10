// Package ui runs the event loop, key handling and rendering.
package ui

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/hinshun/vt10x"

	"window-tui/internal/layout"
	"window-tui/internal/pane"
	"window-tui/internal/window"
)

// vt10x attribute bits (unexported upstream).
const (
	attrReverse   = 1 << 0
	attrUnderline = 1 << 1
	attrBold      = 1 << 2
	attrItalic    = 1 << 4
	attrBlink     = 1 << 5
)

type app struct {
	screen     tcell.Screen
	sess       *window.Session
	redraw     chan struct{}
	exited     chan *pane.Pane
	prefix     bool
	prefixKey  tcell.Key
	prefixName string
	quit       bool
	rects      map[layout.Item]layout.Rect
	seps       []layout.Rect
}

// Run starts the multiplexer; prefixSpec is e.g. "ctrl-a".
func Run(prefixSpec string) error {
	key, err := parsePrefix(prefixSpec)
	if err != nil {
		return err
	}
	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}
	defer s.Fini()

	a := &app{
		screen:     s,
		prefixKey:  key,
		prefixName: prefixSpec,
		sess:       &window.Session{},
		redraw:     make(chan struct{}, 1),
		exited:     make(chan *pane.Pane, 16),
	}
	if err := a.newWindow(); err != nil {
		return err
	}

	events := make(chan tcell.Event, 32)
	go func() {
		for {
			ev := s.PollEvent()
			if ev == nil {
				close(events)
				return
			}
			events <- ev
		}
	}()

	a.draw()
	for !a.quit {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			a.handleEvent(ev)
		case <-a.redraw:
		case p := <-a.exited:
			a.sess.ClosePane(p)
		}
		if len(a.sess.Windows) == 0 {
			break
		}
		a.draw()
	}
	for _, w := range a.sess.Windows {
		for _, p := range w.Panes() {
			p.Close()
		}
	}
	return nil
}

func (a *app) newPane() (*pane.Pane, error) {
	return pane.New(80, 24,
		func() {
			select {
			case a.redraw <- struct{}{}:
			default:
			}
		},
		func(p *pane.Pane) { a.exited <- p },
	)
}

func (a *app) newWindow() error {
	p, err := a.newPane()
	if err != nil {
		return err
	}
	a.sess.Add(p)
	return nil
}

func (a *app) split(vert bool) {
	w := a.sess.Current()
	p, err := a.newPane()
	if err != nil {
		return
	}
	w.Root.Split(w.Active, p, vert)
	w.Active = p
}

func (a *app) handleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		a.screen.Sync()
	case *tcell.EventKey:
		a.handleKey(ev)
	}
}

func (a *app) handleKey(ev *tcell.EventKey) {
	w := a.sess.Current()
	if w == nil {
		return
	}
	if !a.prefix {
		if ev.Key() == a.prefixKey {
			a.prefix = true
			return
		}
		if ev.Modifiers() == tcell.ModCtrl|tcell.ModShift {
			switch ev.Key() {
			case tcell.KeyLeft, tcell.KeyRight, tcell.KeyUp, tcell.KeyDown:
				a.arrow(w, ev.Key(), false) // Ctrl+Shift+arrow: focus, no prefix
				return
			}
		}
		w.Active.Write(encodeKey(ev))
		return
	}
	a.prefix = false
	ctrl := ev.Modifiers()&tcell.ModCtrl != 0
	switch ev.Key() {
	case a.prefixKey:
		w.Active.Write([]byte{byte(a.prefixKey)})
	case tcell.KeyLeft, tcell.KeyRight, tcell.KeyUp, tcell.KeyDown:
		a.arrow(w, ev.Key(), ctrl)
	case tcell.KeyTab:
		a.sess.Next(1)
	case tcell.KeyBacktab:
		a.sess.Next(-1)
	case tcell.KeyRune:
		switch r := ev.Rune(); {
		case r == 'n':
			a.newWindow()
		case r == 'c':
			a.sess.ClosePane(w.Active)
		case r == 'v':
			a.split(true)
		case r == 'h':
			a.split(false)
		case r >= '0' && r <= '9':
			a.sess.Select(int(r - '0'))
		case r == 'o':
			ps := w.Panes()
			for i, p := range ps {
				if p == w.Active {
					w.Active = ps[(i+1)%len(ps)]
					break
				}
			}
		case r == 'd' || r == 'q':
			a.quit = true
		}
	}
}

func (a *app) arrow(w *window.Window, k tcell.Key, resize bool) {
	if resize {
		const step = 0.05
		switch k {
		case tcell.KeyLeft:
			w.Root.Adjust(w.Active, true, -step)
		case tcell.KeyRight:
			w.Root.Adjust(w.Active, true, step)
		case tcell.KeyUp:
			w.Root.Adjust(w.Active, false, -step)
		case tcell.KeyDown:
			w.Root.Adjust(w.Active, false, step)
		}
		return
	}
	d := map[tcell.Key]layout.Dir{
		tcell.KeyLeft: layout.Left, tcell.KeyRight: layout.Right,
		tcell.KeyUp: layout.Up, tcell.KeyDown: layout.Down,
	}[k]
	if n := layout.Neighbor(a.rects, w.Active, d); n != nil {
		w.Active = n.(*pane.Pane)
	}
}

func (a *app) draw() {
	s := a.screen
	s.Clear()
	cols, rows := s.Size()
	w := a.sess.Current()
	if w == nil || rows < 3 {
		s.Show()
		return
	}
	a.rects = map[layout.Item]layout.Rect{}
	a.seps = a.seps[:0]
	w.Root.Compute(layout.Rect{X: 0, Y: 0, W: cols, H: rows - 2}, a.rects, &a.seps)

	sepStyle := tcell.StyleDefault.Foreground(tcell.ColorGray)
	activeStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true)
	active := a.rects[w.Active]
	for _, sp := range a.seps {
		for y := sp.Y; y < sp.Y+sp.H; y++ {
			for x := sp.X; x < sp.X+sp.W; x++ {
				ch := '│'
				if sp.H == 1 {
					ch = '─'
				}
				st := sepStyle
				if adjacent(active, x, y) {
					st = activeStyle
				}
				s.SetContent(x, y, ch, nil, st)
			}
		}
	}

	showCursor := false
	for it, r := range a.rects {
		p := it.(*pane.Pane)
		p.Resize(r.W, r.H)
		cx, cy, vis := drawPane(s, p, r)
		if p == w.Active && vis {
			s.ShowCursor(cx, cy)
			showCursor = true
		}
	}
	if !showCursor {
		s.HideCursor()
	}
	a.drawStatus(rows-2, cols)
	a.drawHelp(rows-1, cols)
	s.Show()
}

func drawPane(s tcell.Screen, p *pane.Pane, r layout.Rect) (cx, cy int, vis bool) {
	t := p.Term
	t.Lock()
	defer t.Unlock()
	tc, tr := t.Size()
	for y := 0; y < r.H && y < tr; y++ {
		for x := 0; x < r.W && x < tc; x++ {
			g := t.Cell(x, y)
			ch := g.Char
			if ch == 0 {
				ch = ' '
			}
			s.SetContent(r.X+x, r.Y+y, ch, nil, glyphStyle(g))
		}
	}
	c := t.Cursor()
	return r.X + c.X, r.Y + c.Y, t.CursorVisible() && c.X < r.W && c.Y < r.H
}

func glyphStyle(g vt10x.Glyph) tcell.Style {
	st := tcell.StyleDefault.Foreground(color(g.FG)).Background(color(g.BG))
	m := g.Mode
	if m&attrBold != 0 {
		st = st.Bold(true)
	}
	if m&attrUnderline != 0 {
		st = st.Underline(true)
	}
	if m&attrItalic != 0 {
		st = st.Italic(true)
	}
	if m&attrBlink != 0 {
		st = st.Blink(true)
	}
	if m&attrReverse != 0 {
		st = st.Reverse(true)
	}
	return st
}

func color(c vt10x.Color) tcell.Color {
	switch {
	case c < 256:
		return tcell.PaletteColor(int(c))
	case c >= 1<<24:
		return tcell.ColorDefault // DefaultFG/BG/Cursor
	default:
		return tcell.NewHexColor(int32(c))
	}
}

func (a *app) drawStatus(y, cols int) {
	bar := tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack)
	for x := 0; x < cols; x++ {
		a.screen.SetContent(x, y, ' ', nil, bar)
	}
	x := 1
	put := func(str string, st tcell.Style) {
		for _, r := range str {
			a.screen.SetContent(x, y, r, nil, st)
			x++
		}
	}
	host, _ := os.Hostname()
	put("["+host+"] ", bar.Bold(true))
	for i, w := range a.sess.Windows {
		flag := " "
		st := bar
		if i == a.sess.Cur {
			flag = "*"
			st = bar.Reverse(true)
		}
		put(fmt.Sprintf("%d:%s%s", i, title(w), flag), st)
		put(" ", bar)
	}
	if a.prefix {
		put(" [prefix]", bar.Bold(true))
	}
}

func title(w *window.Window) string {
	t := w.Active.Term
	t.Lock()
	name := t.Title()
	t.Unlock()
	if name != "" {
		return name
	}
	return "sh"
}

// adjacent reports whether cell (x, y) lies on the border directly around r.
func adjacent(r layout.Rect, x, y int) bool {
	inX := x >= r.X && x < r.X+r.W
	inY := y >= r.Y && y < r.Y+r.H
	return (inX && (y == r.Y-1 || y == r.Y+r.H)) || (inY && (x == r.X-1 || x == r.X+r.W))
}

func (a *app) drawHelp(y, cols int) {
	st := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorSilver)
	for x := 0; x < cols; x++ {
		a.screen.SetContent(x, y, ' ', nil, st)
	}
	text := fmt.Sprintf(" %s, puis :  n=new  c=close  v=split|  h=split-  Ctrl+Shift+arrows=focus  (prefix)Ctrl+arrows=resize  Tab/Shift+Tab/0-9=window  o=next pane  d=quit", a.prefixName)
	x := 0
	for _, r := range text {
		if x >= cols {
			break
		}
		a.screen.SetContent(x, y, r, nil, st)
		x++
	}
}
