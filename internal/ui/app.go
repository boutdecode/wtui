// Package ui runs the event loop, key handling and rendering.
package ui

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/gdamore/tcell/v2"

	"window-tui/internal/layout"
	"window-tui/internal/pane"
	"window-tui/internal/window"
)

type app struct {
	screen     tcell.Screen
	sess       *window.Session
	redraw     chan struct{}
	exited     chan *pane.Pane
	prefix     bool
	prefixKey  tcell.Key
	prefixName string
	command    []string // program started in every new pane; empty = $SHELL
	renaming   bool
	renameBuf  []rune
	quit       bool
	rects      map[layout.Item]layout.Rect
	seps       []layout.Rect
}

// Run starts the multiplexer; prefixSpec is e.g. "ctrl-a".
func Run(prefixSpec string, command []string) error {
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
		command:    command,
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
	return pane.New(a.command, 80, 24,
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
	if a.renaming {
		a.handleRenameKey(ev, w)
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
		case r == 'r':
			a.renaming = true
			a.renameBuf = []rune(w.Active.Name)
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

	active := a.rects[w.Active]
	for _, sp := range a.seps {
		for y := sp.Y; y < sp.Y+sp.H; y++ {
			for x := sp.X; x < sp.X+sp.W; x++ {
				ch, st := '│', styleBorder
				if sp.H == 1 {
					ch = '─'
				}
				if adjacent(active, x, y) {
					st = styleBorderActive
					if sp.H == 1 {
						ch = '━'
					} else {
						ch = '┃'
					}
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
	if a.renaming {
		a.drawRename(rows-1, cols)
	} else {
		a.drawHelp(rows-1, cols)
	}
	s.Show()
}

func drawPane(s tcell.Screen, p *pane.Pane, r layout.Rect) (cx, cy int, vis bool) {
	p.WithTerm(func(t *vt.Emulator) {
		tc, tr := t.Width(), t.Height()
		for y := 0; y < r.H && y < tr; y++ {
			for x := 0; x < r.W && x < tc; x++ {
				c := t.CellAt(x, y)
				if c == nil {
					continue
				}
				if c.Width == 0 && c.Content == "" {
					continue // right half of a wide character
				}
				rs := []rune(c.Content)
				if len(rs) == 0 || (c.Width == 2 && x+1 >= r.W) {
					rs = []rune{' '}
				}
				s.SetContent(r.X+x, r.Y+y, rs[0], rs[1:], cellStyle(c.Style.Fg, c.Style.Bg, c.Style.Attrs, c.Style.Underline != 0))
			}
		}
		pos := t.CursorPosition()
		cx, cy = r.X+pos.X, r.Y+pos.Y
		vis = !t.CursorHidden() && pos.X < r.W && pos.Y < r.H
	})
	return cx, cy, vis
}

// Text attribute bits of ultraviolet's Style.Attrs.
const (
	attrBold          = 1 << 0
	attrFaint         = 1 << 1
	attrItalic        = 1 << 2
	attrBlink         = 1 << 3
	attrReverse       = 1 << 5
	attrStrikethrough = 1 << 7
)

func cellStyle(fg, bg color.Color, attrs uint8, underline bool) tcell.Style {
	st := tcell.StyleDefault.Foreground(toTcell(fg)).Background(toTcell(bg))
	return st.
		Bold(attrs&attrBold != 0).
		Dim(attrs&attrFaint != 0).
		Italic(attrs&attrItalic != 0).
		Blink(attrs&attrBlink != 0).
		Reverse(attrs&attrReverse != 0).
		StrikeThrough(attrs&attrStrikethrough != 0).
		Underline(underline)
}

func toTcell(c color.Color) tcell.Color {
	switch v := c.(type) {
	case nil:
		return tcell.ColorDefault
	case ansi.BasicColor:
		return tcell.PaletteColor(int(v))
	case ansi.IndexedColor:
		return tcell.PaletteColor(int(v))
	default:
		r, g, b, _ := c.RGBA()
		return tcell.NewRGBColor(int32(r>>8), int32(g>>8), int32(b>>8))
	}
}

func (a *app) drawStatus(y, cols int) {
	for x := 0; x < cols; x++ {
		a.screen.SetContent(x, y, ' ', nil, styleBar)
	}
	x := 0
	put := func(str string, st tcell.Style) {
		for _, r := range str {
			if x < cols {
				a.screen.SetContent(x, y, r, nil, st)
			}
			x++
		}
	}
	host, _ := os.Hostname()
	put(" "+host+" ", styleHost)
	put(" ", styleBar)
	for i, w := range a.sess.Windows {
		if i == a.sess.Cur {
			put(fmt.Sprintf(" %d %s ", i, title(w)), styleTabActive)
		} else {
			put(fmt.Sprintf(" %d %s ", i, title(w)), styleTab)
		}
	}
	if a.prefix {
		put(" ", styleBar)
		put(" PREFIX ", stylePrefix)
	}
}

func title(w *window.Window) string {
	if w.Active.Name != "" {
		return w.Active.Name
	}
	if t := w.Active.Title(); t != "" {
		return t
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
	for x := 0; x < cols; x++ {
		a.screen.SetContent(x, y, ' ', nil, styleHelpBar)
	}
	x := 0
	put := func(str string, st tcell.Style) {
		for _, r := range str {
			if x < cols {
				a.screen.SetContent(x, y, r, nil, st)
			}
			x++
		}
	}
	put(" "+a.prefixName+" ", stylePrefix)
	items := [][2]string{
		{"n", "new"}, {"c", "close"}, {"r", "rename"}, {"v", "split |"}, {"h", "split -"},
		{"Ctrl+Shift+←↑→↓", "focus"}, {"prefix+Ctrl+←↑→↓", "resize"},
		{"Tab/0-9", "window"}, {"o", "next pane"}, {"d", "quit"},
	}
	for i, it := range items {
		if i > 0 {
			put("  •", styleHelpSep)
		}
		put("  ", styleHelpBar)
		put(it[0], styleHelpKey)
		put(" "+it[1], styleHelpBar)
	}
}

func (a *app) handleRenameKey(ev *tcell.EventKey, w *window.Window) {
	switch ev.Key() {
	case tcell.KeyEnter:
		w.Active.Name = strings.TrimSpace(string(a.renameBuf))
		a.renaming = false
	case tcell.KeyEsc, tcell.KeyCtrlC:
		a.renaming = false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if n := len(a.renameBuf); n > 0 {
			a.renameBuf = a.renameBuf[:n-1]
		}
	case tcell.KeyCtrlU:
		a.renameBuf = nil
	case tcell.KeyRune:
		if len(a.renameBuf) < 40 {
			a.renameBuf = append(a.renameBuf, ev.Rune())
		}
	}
}

func (a *app) drawRename(y, cols int) {
	for x := 0; x < cols; x++ {
		a.screen.SetContent(x, y, ' ', nil, styleHelpBar)
	}
	x := 0
	put := func(str string, st tcell.Style) {
		for _, r := range str {
			if x < cols {
				a.screen.SetContent(x, y, r, nil, st)
			}
			x++
		}
	}
	put(" RENAME ", stylePrefix)
	put(" ", styleHelpBar)
	put(string(a.renameBuf), styleHelpKey)
	a.screen.ShowCursor(x, y)
	put("   Enter=ok  Esc=cancel  empty=auto title", styleHelpSep)
}
