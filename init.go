package main

import (
	_ "embed"
	"fmt"
	"os"
)

//go:embed shell/hepler.zsh
var zshIntegration string

//go:embed shell/hepler.bash
var bashIntegration string

//go:embed shell/hepler.fish
var fishIntegration string

// runInit prints the shell integration snippet for the requested shell, so it
// can be sourced directly, e.g. `source <(hepler init zsh)`.
func runInit(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: hepler init <bash|zsh|fish>")
		return 2
	}
	switch args[0] {
	case "zsh":
		fmt.Print(zshIntegration)
	case "bash":
		fmt.Print(bashIntegration)
	case "fish":
		fmt.Print(fishIntegration)
	default:
		fmt.Fprintf(os.Stderr, "hepler: unsupported shell %q (want bash, zsh, or fish)\n", args[0])
		return 2
	}
	return 0
}
