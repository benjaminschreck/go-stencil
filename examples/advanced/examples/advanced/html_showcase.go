package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/benjaminschreck/go-stencil/pkg/stencil"
)

func main() {
	// Create engine
	engine := stencil.New()
	
	// Prepare template
	tmpl, err := engine.PrepareFile("html_showcase.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()
	
	// Create data with various HTML examples
	data := stencil.TemplateData{
		// Dynamic HTML content
		"htmlContent": `<b>Dynamic content</b> with <i>various</i> <u>formatting</u> options and <sup>special</sup> characters like &amp; and &lt;brackets&gt;`,
		
		// Formatted items for loop
		"formattedItems": []map[string]interface{}{
			{"formatted": "<b>First item</b> - Important"},
			{"formatted": "<i>Second item</i> - Emphasis added"},
			{"formatted": "<u>Third item</u> - Underlined for attention"},
			{"formatted": "<s>Fourth item</s> - Deprecated"},
			{"formatted": "<b><i>Fifth item</i></b> - Bold and italic"},
		},
		
		// Conditional flag
		"showImportant": true,
		
		// Variables for HTML with variables example
		"greeting": "<b>Hello</b>",
		"customerName": "John Doe",
		"message": "<i>we have an important update for you</i>",
		
		// HTML table data
		"htmlTable": []map[string]interface{}{
			{
				"col1": "<b>Product</b>",
				"col2": "<i>Description</i>",
				"col3": "<u>Price</u>",
			},
			{
				"col1": "<strong>Widget A</strong>",
				"col2": "High-quality widget with <sup>premium</sup> features",
				"col3": "$19<sup>99</sup>",
			},
			{
				"col1": "<strong>Gadget B</strong>",
				"col2": "Standard gadget with H<sub>2</sub>O resistance",
				"col3": "$29<sup>99</sup>",
			},
			{
				"col1": "<strong>Tool C</strong>",
				"col2": "<em>Professional</em> tool with <b>lifetime</b> warranty",
				"col3": "<s>$49<sup>99</sup></s> $39<sup>99</sup>",
			},
		},
	}
	
	// Render the template
	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}
	
	// Save output
	outputPath := filepath.Join("output", "html_showcase_output.docx")
	if err := saveOutput(output, outputPath); err != nil {
		log.Fatalf("Failed to save output: %v", err)
	}
	
	fmt.Printf("HTML showcase rendered successfully to: %s\n", outputPath)
	
	// Also generate a report showing what HTML was rendered
	generateHTMLReport()
}

func saveOutput(reader io.Reader, filename string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Create output file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Copy rendered content to file
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	
	return nil
}

func generateHTMLReport() {
	report := `
HTML Showcase Report
Generated: ` + time.Now().Format("2006-01-02 15:04:05") + `

Supported HTML Tags:
- <b> / <strong>: Bold text
- <i> / <em>: Italic text
- <u>: Underlined text
- <s> / <strike>: Strikethrough text
- <sup>: Superscript
- <sub>: Subscript
- <br>: Line break
- <span>: Generic inline container (no formatting)

Features Demonstrated:
1. Basic formatting with individual tags
2. Combined/nested formatting
3. Deep nesting of multiple tags
4. Scientific notation (superscript/subscript)
5. Line breaks within HTML content
6. Semantic HTML (strong/em)
7. Dynamic HTML content from variables
8. HTML in loops
9. Conditional HTML rendering
10. HTML mixed with template variables
11. HTML in table cells

Note: All HTML content is properly escaped and converted to valid OOXML.
`
	
	fmt.Println(report)
}