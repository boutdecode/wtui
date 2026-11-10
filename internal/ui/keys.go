package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
)

// encodeKey converts a tcell key event into the bytes a terminal would send.
func encodeKey(ev *tcell.EventKey) []byte {
	mod := ev.Modifiers()
	alt := mod&tcell.ModAlt != 0
	key := ev.Key()

	if key == tcell.KeyRune {
		b := []byte(string(ev.Rune()))
		if alt {
			b = append([]byte{0x1b}, b...)
		}
		return b
	}

	// CSI parameter for modifiers: 1 + shift(1) + alt(2) + ctrl(4)
	m := 1
	if mod&tcell.ModShift != 0 {
		m += 1
	}
	if alt {
		m += 2
	}
	if mod&tcell.ModCtrl != 0 {
		m += 4
	}
	csi := func(final byte) []byte {
		if m == 1 {
			return []byte{0x1b, '[', final}
		}
		return []byte(fmt.Sprintf("\x1b[1;%d%c", m, final))
	}
	tilde := func(n int) []byte {
		if m == 1 {
			return []byte(fmt.Sprintf("\x1b[%d~", n))
		}
		return []byte(fmt.Sprintf("\x1b[%d;%d~", n, m))
	}

	switch key {
	case tcell.KeyUp:
		return csi('A')
	case tcell.KeyDown:
		return csi('B')
	case tcell.KeyRight:
		return csi('C')
	case tcell.KeyLeft:
		return csi('D')
	case tcell.KeyHome:
		return csi('H')
	case tcell.KeyEnd:
		return csi('F')
	case tcell.KeyBacktab:
		return []byte("\x1b[Z")
	case tcell.KeyInsert:
		return tilde(2)
	case tcell.KeyDelete:
		return tilde(3)
	case tcell.KeyPgUp:
		return tilde(5)
	case tcell.KeyPgDn:
		return tilde(6)
	case tcell.KeyBackspace2:
		return []byte{0x7f}
	case tcell.KeyF1:
		return []byte("\x1bOP")
	case tcell.KeyF2:
		return []byte("\x1bOQ")
	case tcell.KeyF3:
		return []byte("\x1bOR")
	case tcell.KeyF4:
		return []byte("\x1bOS")
	case tcell.KeyF5:
		return tilde(15)
	case tcell.KeyF6:
		return tilde(17)
	case tcell.KeyF7:
		return tilde(18)
	case tcell.KeyF8:
		return tilde(19)
	case tcell.KeyF9:
		return tilde(20)
	case tcell.KeyF10:
		return tilde(21)
	case tcell.KeyF11:
		return tilde(23)
	case tcell.KeyF12:
		return tilde(24)
	}
	if key < 32 { // Ctrl-A..Z, Tab, Enter, Esc, Ctrl-Space...
		b := []byte{byte(key)}
		if alt {
			b = append([]byte{0x1b}, b...)
		}
		return b
	}
	return nil
}
