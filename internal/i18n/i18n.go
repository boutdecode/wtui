// Package i18n provides a tiny message catalogue (en, fr) with no dependencies.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

// Lang is a supported language code.
type Lang string

const (
	EN Lang = "en"
	FR Lang = "fr"
)

var current = EN

// Normalize maps values such as "fr", "fr_FR.UTF-8" or "FR-ca" to a supported
// language. ok is false when the value is empty or unsupported.
func Normalize(s string) (Lang, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "fr" || strings.HasPrefix(s, "fr_") || strings.HasPrefix(s, "fr-") || strings.HasPrefix(s, "fr."):
		return FR, true
	case s == "en" || strings.HasPrefix(s, "en_") || strings.HasPrefix(s, "en-") || strings.HasPrefix(s, "en."):
		return EN, true
	}
	return EN, false
}

// Detect picks a language from WTUI_LANG, LC_ALL, LC_MESSAGES then LANG,
// defaulting to English.
func Detect() Lang {
	for _, k := range []string{"WTUI_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if l, ok := Normalize(os.Getenv(k)); ok {
			return l
		}
	}
	return EN
}

// Set selects the active language.
func Set(l Lang) {
	if _, ok := catalog[l]; ok {
		current = l
	}
}

// Current returns the active language.
func Current() Lang { return current }

// T returns the translation of key, formatted with args when given.
// It falls back to English, then to the key itself.
func T(key string, args ...any) string {
	msg, ok := catalog[current][key]
	if !ok {
		if msg, ok = catalog[EN][key]; !ok {
			msg = key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}

var catalog = map[Lang]map[string]string{
	EN: {
		"help.new":            "new",
		"help.close":          "close",
		"help.rename":         "rename",
		"help.split_v":        "split |",
		"help.split_h":        "split -",
		"help.focus":          "focus",
		"help.move":           "move",
		"help.resize":         "resize",
		"help.window":         "window",
		"help.scroll":         "scroll",
		"help.select":         "select+copy",
		"help.quit":           "quit",
		"key.wheel":           "Wheel",
		"key.drag":            "Drag",
		"bar.prefix":          " PREFIX ",
		"bar.rename":          " RENAME ",
		"bar.rename_hint":     "   Enter=ok  Esc=cancel  empty=auto title",
		"err.prefix_conflict": "prefix %q conflicts with Tab/Enter/Backspace",
		"err.prefix_invalid":  "invalid prefix %q (try ctrl-a, ctrl-space, ctrl-\\)",
		"err.lang":            "unsupported language %q (use en or fr)",
		"flag.prefix":         "prefix key (ctrl-a, ctrl-b, ctrl-space, ctrl-\\)",
		"flag.borders":        "draw separator lines between panes (default: panes touch, inactive ones are dimmed)",
		"flag.cwd":            "directory of the first pane (default: current directory)",
		"flag.cwd_alias":      "alias for --cwd",
		"flag.lang":           "interface language: en or fr (default: auto-detected from the environment)",
		"flag.version":        "print version and exit",
		"usage.line":          "Usage: wtui [--prefix KEY] [--borders] [--cwd DIR] [--lang en|fr] [command [args...]]",
		"usage.command":       "command is started in every new pane (default: $SHELL).",
	},
	FR: {
		"help.new":            "nouveau",
		"help.close":          "fermer",
		"help.rename":         "renommer",
		"help.split_v":        "scinder |",
		"help.split_h":        "scinder -",
		"help.focus":          "focus",
		"help.move":           "déplacer",
		"help.resize":         "redimensionner",
		"help.window":         "fenêtre",
		"help.scroll":         "défiler",
		"help.select":         "sélection+copie",
		"help.quit":           "quitter",
		"key.wheel":           "Molette",
		"key.drag":            "Glisser",
		"bar.prefix":          " PRÉFIXE ",
		"bar.rename":          " RENOMMER ",
		"bar.rename_hint":     "   Entrée=ok  Échap=annuler  vide=titre auto",
		"err.prefix_conflict": "le préfixe %q est en conflit avec Tab/Entrée/Retour arrière",
		"err.prefix_invalid":  "préfixe %q invalide (essayez ctrl-a, ctrl-space, ctrl-\\)",
		"err.lang":            "langue %q non prise en charge (utilisez en ou fr)",
		"flag.prefix":         "touche préfixe (ctrl-a, ctrl-b, ctrl-space, ctrl-\\)",
		"flag.borders":        "dessine des séparateurs entre les panneaux (défaut : panneaux collés, les inactifs sont atténués)",
		"flag.cwd":            "répertoire du premier panneau (défaut : répertoire courant)",
		"flag.cwd_alias":      "alias de --cwd",
		"flag.lang":           "langue de l'interface : en ou fr (défaut : détectée via l'environnement)",
		"flag.version":        "affiche la version et quitte",
		"usage.line":          "Usage : wtui [--prefix TOUCHE] [--borders] [--cwd DIR] [--lang en|fr] [commande [args...]]",
		"usage.command":       "la commande est lancée dans chaque nouveau panneau (défaut : $SHELL).",
	},
}
