// Simple example demonstrating basic go-stencil usage
package main

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/benjaminschreck/go-stencil/pkg/stencil"
)

func main() {
	// Prepare a template from a file
	tmpl, err := stencil.PrepareFile("template.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Create template data
	data := stencil.TemplateData{
		"name":     "John Doe",
		"date":     time.Now().Format("January 2, 2006"),
		"company":  "Acme Corporation",
		"position": "Software Engineer",
		"items": []map[string]interface{}{
			{"name": "Task 1", "status": "Complete"},
			{"name": "Task 2", "status": "In Progress"},
			{"name": "Task 3", "status": "Pending"},
		},
	}

	// Render the template
	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	// Save the output
	outputFile, err := os.Create("output.docx")
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, output)
	if err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	log.Println("Template rendered successfully to output.docx")
}
