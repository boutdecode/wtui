package pane

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/vt"
)

// run starts a shell command in a pane, waits for it to exit and returns the
// emulated screen rows.
func run(t *testing.T, cols, rows int, script string) []string {
	t.Helper()
	done := make(chan struct{})
	p, err := New([]string{"sh", "-c", script}, "", cols, rows, func() {}, func(*Pane) { close(done) })
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
	var out []string
	p.WithTerm(func(e *vt.Emulator) {
		for y := 0; y < e.Height(); y++ {
			var sb strings.Builder
			for x := 0; x < e.Width(); x++ {
				c := e.CellAt(x, y)
				if c == nil || (c.Width == 0 && c.Content == "") {
					continue
				}
				if c.Content == "" {
					sb.WriteByte(' ')
				} else {
					sb.WriteString(c.Content)
				}
			}
			out = append(out, strings.TrimRight(sb.String(), " "))
		}
	})
	return out
}

func TestRepeatAndCombining(t *testing.T) {
	// REP (CSI b) repeats the previous character; u + U+0308 is one cell.
	got := run(t, 20, 5, `printf 'a\033[3bX u\314\210Z'`)
	if got[0] != "aaaaX u\u0308Z" { // the combining mark stays in the same cell
		t.Fatalf("row 0 = %q", got[0])
	}
}

func TestScrollRegion(t *testing.T) {
	// 6 rows; region rows 2-5; reverse index at region top scrolls down inside it.
	script := `printf '\033[2J\033[H1\r\n2\r\n3\r\n4\r\n5\r\n6\033[2;5r\033[2;1H\033M'`
	got := run(t, 10, 6, script)
	want := []string{"1", "", "2", "3", "4", "6"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rows = %q, want %q", got, want)
		}
	}
}

func TestInsertDeleteLines(t *testing.T) {
	// IL then DL inside the region must keep the rows outside intact.
	script := `printf '\033[2J\033[H1\r\n2\r\n3\r\n4\r\n5\033[2;4r\033[3;1H\033[L\033[2;1H\033[M'`
	got := run(t, 10, 5, script)
	if got[0] != "1" || got[4] != "5" {
		t.Fatalf("rows outside region changed: %q", got)
	}
}

func TestResizeKeepsContent(t *testing.T) {
	var gotCols int
	p, err := New([]string{"sh", "-c", "sleep 2"}, "", 80, 24, func() {}, func(*Pane) {})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	p.Resize(30, 7)
	p.WithTerm(func(e *vt.Emulator) { gotCols = e.Width() })
	if gotCols != 30 {
		t.Fatalf("width = %d", gotCols)
	}
}

func TestNewHonoursDir(t *testing.T) {
	dir := t.TempDir()
	p, err := New([]string{"sh", "-c", "sleep 2"}, dir, 80, 24, func() {}, func(*Pane) {})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	got, _ := filepath.EvalSymlinks(p.Cwd())
	want, _ := filepath.EvalSymlinks(dir)
	if got != want {
		t.Fatalf("Cwd() = %q, want %q", got, want)
	}
}
