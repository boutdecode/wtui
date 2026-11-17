package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"window-tui/internal/i18n"
	"window-tui/internal/ui"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

// langFromArgs finds --lang/-lang (with = or a separate value) before flag
// parsing, so that the help text itself can be localised.
func langFromArgs(args []string) (string, bool) {
	for i, a := range args {
		if a == "--" {
			break
		}
		a = strings.TrimLeft(a, "-")
		if a == "lang" && i+1 < len(args) && strings.HasPrefix(args[i], "-") {
			return args[i+1], true
		}
		if strings.HasPrefix(a, "lang=") && strings.HasPrefix(args[i], "-") {
			return strings.TrimPrefix(a, "lang="), true
		}
	}
	return "", false
}

func main() {
	i18n.Set(i18n.Detect())
	if v, ok := langFromArgs(os.Args[1:]); ok {
		l, valid := i18n.Normalize(v)
		if !valid {
			fmt.Fprintln(os.Stderr, "wtui:", i18n.T("err.lang", v))
			os.Exit(2)
		}
		i18n.Set(l)
	}

	def := os.Getenv("WTUI_PREFIX")
	if def == "" {
		def = "ctrl-a"
	}
	prefix := flag.String("prefix", def, i18n.T("flag.prefix"))
	borders := flag.Bool("borders", os.Getenv("WTUI_BORDERS") == "1", i18n.T("flag.borders"))
	cwd := flag.String("cwd", os.Getenv("WTUI_CWD"), i18n.T("flag.cwd"))
	flag.StringVar(cwd, "working-directory", *cwd, i18n.T("flag.cwd_alias"))
	flag.String("lang", string(i18n.Current()), i18n.T("flag.lang"))
	showVersion := flag.Bool("version", false, i18n.T("flag.version"))
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\n", i18n.T("usage.line"))
		fmt.Fprintf(os.Stderr, "%s\n\n", i18n.T("usage.command"))
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("wtui", version)
		return
	}
	if err := ui.Run(ui.Options{Prefix: *prefix, Borders: *borders, Cwd: *cwd, Command: flag.Args()}); err != nil {
		fmt.Fprintln(os.Stderr, "wtui:", err)
		os.Exit(1)
	}
}
