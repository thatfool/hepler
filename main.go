package main

import (
	"fmt"
	"os"
)

var version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "edit":
		os.Exit(runEdit(os.Args[2:]))
	case "init":
		os.Exit(runInit(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Println("hepler", version)
	case "help", "--help", "-h":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "hepler: unknown command %q\n\n", os.Args[1])
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w *os.File) {
	fmt.Fprint(w, `hepler - an LLM helper for your shell command line

Usage:
  hepler init <bash|zsh|fish>        Print the shell integration snippet for the given shell
  hepler version                     Print the version
 
For integration:
  hepler edit [--yes]                Read a command line on stdin, print the result on stdout

Configuration (environment):
  HEPLER_OPENAI_API_BASE   Base URL of an OpenAI-compatible endpoint
  HEPLER_OPENAI_API_KEY    API key for that endpoint (optional if not required)
  HEPLER_MODEL_NAME        Qualified name of the model to use

Shell setup, e.g. for zsh, add to ~/.zshrc:
  source <(hepler init zsh)
`)
}
