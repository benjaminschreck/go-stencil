package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("go-stencil - Template engine for DOCX/PPTX files")
	fmt.Println("Version: 0.1.0")
	
	if len(os.Args) < 2 {
		fmt.Println("\nUsage: stencil <command> [arguments]")
		fmt.Println("\nCommands:")
		fmt.Println("  render <template> <data>    Render a template with data")
		fmt.Println("  version                     Show version information")
		os.Exit(1)
	}
	
	command := os.Args[1]
	
	switch command {
	case "version":
		fmt.Println("go-stencil version 0.1.0")
	case "render":
		fmt.Println("Render command not yet implemented")
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}