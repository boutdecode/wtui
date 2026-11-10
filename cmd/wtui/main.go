package main

import (
	"flag"
	"fmt"
	"os"

	"window-tui/internal/ui"
)

func main() {
	def := os.Getenv("WTUI_PREFIX")
	if def == "" {
		def = "ctrl-a"
	}
	prefix := flag.String("prefix", def, "prefix key (ctrl-a, ctrl-b, ctrl-space, ctrl-\\)")
	borders := flag.Bool("borders", os.Getenv("WTUI_BORDERS") == "1", "draw separator lines between panes (default: panes touch, inactive ones are dimmed)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: wtui [--prefix KEY] [--borders] [command [args...]]\n\n")
		fmt.Fprintf(os.Stderr, "command is started in every new pane (default: $SHELL).\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if err := ui.Run(ui.Options{Prefix: *prefix, Borders: *borders, Command: flag.Args()}); err != nil {
		fmt.Fprintln(os.Stderr, "wtui:", err)
		os.Exit(1)
	}
}
