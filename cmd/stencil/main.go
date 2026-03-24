package main

import (
	"fmt"
	"io"
	"os"

	"github.com/benjaminschreck/go-stencil/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printBanner(stdout)
		printUsage(stderr)
		return 1
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "go-stencil version %s\n", version.Details())
		return 0
	case "render":
		fmt.Fprintln(stdout, "Render command not yet implemented")
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func printBanner(w io.Writer) {
	fmt.Fprintln(w, "go-stencil - Template engine for DOCX/PPTX files")
	fmt.Fprintf(w, "Version: %s\n", version.Details())
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: stencil <command> [arguments]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  render <template> <data>    Render a template with data")
	fmt.Fprintln(w, "  version                     Show version information")
}
