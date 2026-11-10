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
	flag.Parse()
	if err := ui.Run(*prefix); err != nil {
		fmt.Fprintln(os.Stderr, "wtui:", err)
		os.Exit(1)
	}
}
