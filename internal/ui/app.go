// Package ui runs the event loop, key handling and rendering.
package ui

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"
	"unicode/utf8"
	"window-tui/internal/i18n"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/gdamore/tcell/v2"

	"window-tui/internal/layout"
	"window-tui/internal/pane"
	"window-tui/internal/sysstat"
	"window-tui/internal/window"
)

type app struct {
	pasting    bool   // between the start and end of a bracketed paste
	pasteBuf   []byte // input accumulated during a paste
	screen     tcell.Screen
	sess       *window.Session
	redraw     chan struct{}
	exited     chan *pane.Pane
	prefix     bool
	prefixKey  tcell.Key
	prefixName string
	cwd        string   // directory of the first pane; empty = wtui's own
	command    []string // program started in every new pane; empty = $SHELL
	stats      *sysstat.Stats
	borders    bool
	buttons    tcell.ButtonMask // buttons held after the previous mouse event
	mousePane  *pane.Pane       // pane that received the current press
	sel        *pane.Selection  // text selection, in selPane view coordinates
	selPane    *pane.Pane
	selecting  bool // a left-drag selection is in progress
	clip       *clipboard
	notice     string // transient message shown in the status bar
	renaming   bool
	renameBuf  []rune
	quit       bool
	rects      map[layout.Item]layout.Rect
	seps       []layout.Rect
	lastTitle  string       // last host window title emitted
	setTitle   func(string) // overrides the host title writer (tests)
}

// Run starts the multiplexer; prefixSpec is e.g. "ctrl-a".
// Options configures Run.
type Options struct {
	Prefix  string   // prefix key, e.g. "ctrl-a"
	Borders bool     // draw separator lines between panes (otherwise panes touch)
	Command []string // program started in every pane; empty = $SHELL
	Cwd     string   // directory of the first pane; later panes inherit the active pane's
}

func Run(opt Options) error {
	prefixSpec, command := opt.Prefix, opt.Command
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
	// Save the host title now; restore it after tcell has finished.
	writeHost("\x1b[22;0t")
	defer writeHost("\x1b[23;0t")
	defer s.Fini()
	s.EnableMouse()
	s.EnablePaste()

	a := &app{
		screen:     s,
		prefixKey:  key,
		prefixName: prefixSpec,
		command:    command,
		cwd:        opt.Cwd,
		borders:    opt.Borders,
		sess:       &window.Session{},
		clip:       newClipboard(),
		redraw:     make(chan struct{}, 1),
		exited:     make(chan *pane.Pane, 16),
	}
	a.stats = sysstat.Start(statsLen, time.Second, func() {
		select {
		case a.redraw <- struct{}{}:
		default:
		}
	})
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
			// Process whatever is already queued before redrawing once.
		drain:
			for !a.quit {
				select {
				case ev, ok := <-events:
					if !ok {
						return nil
					}
					a.handleEvent(ev)
				default:
					break drain
				}
			}
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
	dir := a.cwd
	if w := a.sess.Current(); w != nil && w.Active != nil {
		if d := w.Active.Cwd(); d != "" {
			dir = d
		}
	}
	return pane.New(a.command, dir, 80, 24,
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
	case *tcell.EventPaste:
		a.handlePaste(ev)
	case *tcell.EventKey:
		if a.pasting {
			a.pasteBuf = append(a.pasteBuf, encodeKey(ev)...)
			return
		}
		a.handleKey(ev)
	case *tcell.EventMouse:
		a.handleMouse(ev)
	}
}

// handlePaste buffers the keys of a bracketed paste and sends them to the
// active pane in one write when the paste ends.
func (a *app) handlePaste(ev *tcell.EventPaste) {
	if ev.Start() {
		a.pasting, a.pasteBuf = true, a.pasteBuf[:0]
		return
	}
	a.pasting = false
	buf := a.pasteBuf
	a.pasteBuf = nil
	a.sel, a.notice = nil, ""
	if w := a.sess.Current(); w != nil && w.Active != nil && !a.renaming {
		w.Active.Paste(buf)
	}
}

// handleMouse routes mouse events: the wheel acts on the pane under the
// pointer; a left press also focuses it; presses, drags and releases are
// forwarded to programs that track the mouse (a drag stays with the pane where
// the press happened).
func (a *app) handleMouse(ev *tcell.EventMouse) {
	x, y := ev.Position()
	if b := ev.Buttons(); b&(tcell.WheelUp|tcell.WheelDown) != 0 {
		a.sel, a.notice = nil, ""
		if it := paneAt(a.rects, x, y); it != nil {
			r := a.rects[it]
			it.(*pane.Pane).Wheel(x-r.X, y-r.Y, b&tcell.WheelUp != 0)
		}
		return
	}
	cur := ev.Buttons() &^ wheelMask
	kind, btn := mouseTransition(a.buttons, cur)
	a.buttons = cur
	w := a.sess.Current()
	if w == nil {
		return
	}

	var target *pane.Pane
	switch kind {
	case mouseDrag, mouseRelease:
		target = a.mousePane // may be nil if the press was outside any pane
	default:
		if it := paneAt(a.rects, x, y); it != nil {
			target = it.(*pane.Pane)
		}
	}
	if kind == mousePress {
		a.mousePane = target
		a.sel, a.selecting, a.notice = nil, false, ""
		if btn == tcell.Button1 && target != nil {
			w.Active = target
			a.selecting = shouldSelect(target.MouseTracking(), ev.Modifiers())
		}
	}
	if target == nil {
		return
	}
	if r, ok := a.rects[target]; ok {
		lx, ly := localPos(r, x, y)
		if a.selecting && target == a.mousePane {
			a.updateSelection(kind, target, lx, ly)
			if kind == mouseRelease && cur == 0 {
				a.mousePane = nil
				a.selecting = false
			}
			return
		}
		if kind == mouseHover {
			btn = 0
		}
		target.SendMouse(newMouseEvent(kind, lx, ly, toVTButton(btn), toVTMod(ev.Modifiers())))
	}
	if kind == mouseRelease && cur == 0 {
		a.mousePane = nil
	}
}

func (a *app) updateSelection(kind mouseKind, p *pane.Pane, lx, ly int) {
	switch kind {
	case mousePress:
		a.sel = &pane.Selection{AX: lx, AY: ly, BX: lx, BY: ly}
		a.selPane = p
	case mouseDrag, mouseRelease:
		if a.sel == nil {
			return
		}
		a.sel.BX, a.sel.BY = lx, ly
		if kind == mouseRelease {
			sel := a.sel
			a.sel, a.selPane = nil, nil // highlight disappears once copied
			if sel.Empty() {
				return
			}
			var text string
			p.WithTerm(func(t *vt.Emulator) {
				text = sel.Text(t, p.ScrollOffset(), t.Width())
			})
			if text != "" {
				a.clip.Copy(text)
				a.notice = i18n.T("bar.copied", utf8.RuneCountInString(text))
			}
		}
	}
}

// shouldSelect reports whether a left press starts a text selection: always
// when the program does not track the mouse; otherwise only with Shift, Alt
// or Ctrl+Shift held (many terminals reserve Shift for their own selection).
func shouldSelect(tracking bool, mod tcell.ModMask) bool {
	if !tracking {
		return true
	}
	return mod&(tcell.ModShift|tcell.ModAlt) != 0
}

// paneAt returns the item whose rectangle contains (x, y), or nil.
func paneAt(rects map[layout.Item]layout.Rect, x, y int) layout.Item {
	for it, r := range rects {
		if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return it
		}
	}
	return nil
}

func (a *app) handleKey(ev *tcell.EventKey) {
	a.sel, a.notice = nil, ""
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
	shift := ev.Modifiers()&tcell.ModShift != 0
	switch ev.Key() {
	case a.prefixKey:
		w.Active.Write([]byte{byte(a.prefixKey)})
	case tcell.KeyLeft, tcell.KeyRight, tcell.KeyUp, tcell.KeyDown:
		if shift && !ctrl {
			a.move(w, ev.Key()) // prefix+Shift+arrow: swap pane with neighbour
		} else {
			a.arrow(w, ev.Key(), ctrl)
		}
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
	if n := layout.Neighbor(a.rects, w.Active, keyDir(k)); n != nil {
		w.Active = n.(*pane.Pane)
	}
}

// move swaps the active pane with its neighbour; focus stays on the active pane.
func (a *app) move(w *window.Window, k tcell.Key) {
	if n := layout.Neighbor(a.rects, w.Active, keyDir(k)); n != nil {
		w.Root.Swap(w.Active, n)
	}
}

func keyDir(k tcell.Key) layout.Dir {
	return map[tcell.Key]layout.Dir{
		tcell.KeyLeft: layout.Left, tcell.KeyRight: layout.Right,
		tcell.KeyUp: layout.Up, tcell.KeyDown: layout.Down,
	}[k]
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
	gap := 0
	if a.borders {
		gap = 1
	}
	w.Root.Compute(layout.Rect{X: 0, Y: 0, W: cols, H: rows - 2}, gap, a.rects, &a.seps)

	active := a.rects[w.Active]
	for pt, arms := range borderArms(a.seps) {
		heavy := onRing(active, pt.x, pt.y)
		st := styleBorder
		if heavy {
			st = styleBorderActive
		}
		s.SetContent(pt.x, pt.y, borderGlyph(arms, heavy), nil, st)
	}
	// Without separator lines, inactive panes are dimmed to show the focus.
	dimInactive := !a.borders && len(a.rects) > 1

	showCursor := false
	for it, r := range a.rects {
		p := it.(*pane.Pane)
		p.Resize(r.W, r.H)
		var sel *pane.Selection
		if p == a.selPane {
			sel = a.sel
		}
		cx, cy, vis := drawPane(s, p, r, dimInactive && p != w.Active, sel)
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
	a.syncTitle(w)
	s.Show()
}

func drawPane(s tcell.Screen, p *pane.Pane, r layout.Rect, dim bool, sel *pane.Selection) (cx, cy int, vis bool) {
	p.WithTerm(func(t *vt.Emulator) {
		off := p.ScrollOffset()
		tc, tr := t.Width(), t.Height()
		for y := 0; y < r.H && y < tr; y++ {
			for x := 0; x < r.W && x < tc; x++ {
				c := pane.ViewCell(t, off, x, y)
				selected := sel != nil && !sel.Empty() && sel.Contains(x, y)
				if c == nil {
					if selected {
						s.SetContent(r.X+x, r.Y+y, ' ', nil, styleSelection)
					}
					continue
				}
				if c.Width == 0 && c.Content == "" {
					continue // right half of a wide character
				}
				rs := []rune(c.Content)
				if len(rs) == 0 || (c.Width == 2 && x+1 >= r.W) {
					rs = []rune{' '}
				}
				st := cellStyle(c.Style.Fg, c.Style.Bg, c.Style.Attrs, c.Style.Underline != 0, dim)
				if selected {
					st = selectionStyle(st)
				}
				s.SetContent(r.X+x, r.Y+y, rs[0], rs[1:], st)
			}
		}
		pos := t.CursorPosition()
		cx, cy = r.X+pos.X, r.Y+pos.Y
		vis = off == 0 && !t.CursorHidden() && pos.X < r.W && pos.Y < r.H
		if off > 0 {
			badge := []rune(fmt.Sprintf(" ↑ %d ", off))
			bx := r.X + r.W - len(badge)
			for i, ch := range badge {
				if bx+i >= r.X {
					s.SetContent(bx+i, r.Y, ch, nil, styleScrollBadge)
				}
			}
		}
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

func cellStyle(fg, bg color.Color, attrs uint8, underline, dim bool) tcell.Style {
	st := tcell.StyleDefault.Foreground(toTcell(fg)).Background(toTcell(bg))
	return st.
		Bold(attrs&attrBold != 0).
		Dim(dim || attrs&attrFaint != 0).
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
		put(i18n.T("bar.prefix"), stylePrefix)
	}
	if a.notice != "" {
		put(" ", styleBar)
		put(a.notice, styleBar)
	}
}

func title(w *window.Window) string { return paneTitle(w.Active) }

func (a *app) drawHelp(y, cols int) {
	for x := 0; x < cols; x++ {
		a.screen.SetContent(x, y, ' ', nil, styleHelpBar)
	}
	limit := cols
	if cols >= minColsForStats {
		limit = cols - a.drawStats(y, cols)
	}
	x := 0
	put := func(str string, st tcell.Style) {
		for _, r := range str {
			if x < limit {
				a.screen.SetContent(x, y, r, nil, st)
			}
			x++
		}
	}
	put(" "+a.prefixName+" ", stylePrefix)
	items := [][2]string{
		{"n", i18n.T("help.new")}, {"c", i18n.T("help.close")}, {"r", i18n.T("help.rename")},
		{"v", i18n.T("help.split_v")}, {"h", i18n.T("help.split_h")},
		{"Ctrl+Shift+←↑→↓", i18n.T("help.focus")}, {"prefix+Shift+←↑→↓", i18n.T("help.move")},
		{"prefix+Ctrl+←↑→↓", i18n.T("help.resize")}, {"Tab/0-9", i18n.T("help.window")},
		{i18n.T("key.wheel"), i18n.T("help.scroll")}, {i18n.T("key.drag"), i18n.T("help.select")},
		{"d", i18n.T("help.quit")},
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
	put(i18n.T("bar.rename"), stylePrefix)
	put(" ", styleHelpBar)
	put(string(a.renameBuf), styleHelpKey)
	a.screen.ShowCursor(x, y)
	put(i18n.T("bar.rename_hint"), styleHelpSep)
}

const (
	statsLen        = 10 // samples shown in each sparkline
	minColsForStats = 90 // hide the CPU/RAM graphs on narrower screens
)

var sparkBlocks = []rune("▁▂▃▄▅▆▇█")

// sparkline renders values (0-100) as block characters, right-aligned in n cells.
func sparkline(vals []float64, n int) string {
	out := make([]rune, n)
	for i := range out {
		out[i] = ' '
	}
	off := n - len(vals)
	for i, v := range vals {
		eighths := int(v/100*8 + 0.5) // height in 1/8 of a cell; 0 stays blank
		if eighths > 8 {
			eighths = 8
		}
		if eighths > 0 && off+i >= 0 {
			out[off+i] = sparkBlocks[eighths-1]
		}
	}
	return string(out)
}

// drawStats draws "CPU ▂▃▅ 42%  RAM ▅▅▆ 63%" at the right of row y and
// returns the width it used (including a one-cell gap on its left).
func (a *app) drawStats(y, cols int) int {
	cpu, mem := a.stats.Snapshot()
	last := func(v []float64) float64 {
		if len(v) == 0 {
			return 0
		}
		return v[len(v)-1]
	}
	type seg struct {
		text string
		st   tcell.Style
	}
	segs := []seg{
		{" CPU ", styleHelpBar},
		{sparkline(cpu, statsLen), styleSparkCPU},
		{fmt.Sprintf(" %3.0f%%", last(cpu)), styleHelpKey},
		{"  RAM ", styleHelpBar},
		{sparkline(mem, statsLen), styleSparkMem},
		{fmt.Sprintf(" %3.0f%% ", last(mem)), styleHelpKey},
	}
	w := 0
	for _, sg := range segs {
		w += len([]rune(sg.text))
	}
	x := cols - w
	for _, sg := range segs {
		for _, r := range sg.text {
			a.screen.SetContent(x, y, r, nil, sg.st)
			x++
		}
	}
	return w + 1
}
