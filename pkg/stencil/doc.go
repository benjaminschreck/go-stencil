// Package stencil provides a powerful template engine for Microsoft Word documents (DOCX).
//
// Go-stencil enables dynamic document generation by processing templates with placeholders,
// control structures, and built-in functions. It's designed for generating reports, invoices,
// contracts, and other documents that require programmatic content injection.
//
// # Quick Start
//
// The simplest way to use go-stencil is through the package-level functions:
//
//	tmpl, err := stencil.PrepareFile("template.docx")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tmpl.Close()
//
//	data := stencil.TemplateData{
//	    "name": "John Doe",
//	    "date": time.Now(),
//	}
//
//	output, err := tmpl.Render(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save to file
//	os.WriteFile("output.docx", output.Bytes(), 0644)
//
// # Template Syntax
//
// All template expressions use double curly braces {{}}:
//
// Variables and Expressions:
//
//	{{name}}                    - Simple variable
//	{{customer.address}}        - Nested field access
//	{{price * 1.2}}            - Mathematical expression
//	{{(basePrice + tax) * qty}} - Complex expression
//
// Control Structures:
//
//	{{if condition}}...{{end}}           - Conditional
//	{{if x > 5}}...{{else}}...{{end}}   - If-else
//	{{if x}}...{{elsif y}}...{{end}}    - If-elsif chain
//	{{unless condition}}...{{end}}       - Negated conditional
//	{{for item in items}}...{{end}}      - Loop
//	{{for i, item in items}}...{{end}}   - Indexed loop
//
// Functions:
//
//	{{uppercase(name)}}                  - String transformation
//	{{format("%.2f", price)}}           - Number formatting
//	{{date("2006-01-02", timestamp)}}   - Date formatting
//	{{sum(numbers)}}                     - Aggregate function
//
// Document Operations:
//
//	{{pageBreak()}}                      - Insert page break
//	{{html("<b>Bold text</b>")}}        - Insert HTML
//	{{include "Fragment Name"}}          - Include fragment
//
// # Architecture
//
// The package is organized into several sub-packages:
//
//   - xml: XML structure definitions for DOCX files (Document, Paragraph, Run, Table, etc.)
//   - render: Pure helper functions for template rendering (control structure detection, run merging)
//
// The main package provides:
//   - Template preparation and rendering (PrepareFile, Render)
//   - Data context management (TemplateData)
//   - Function registry (built-in and custom functions)
//   - Configuration and caching
//   - Error handling
//
// # Advanced Usage
//
// Custom Functions:
//
//	engine := stencil.New()
//	engine.RegisterFunction("greet", func(name string) string {
//	    return "Hello, " + name + "!"
//	})
//
// Configuration:
//
//	config := &stencil.Config{
//	    CacheMaxSize:    100,
//	    CacheExpiration: 24 * time.Hour,
//	}
//	engine := stencil.NewWithConfig(config)
//
// Fragment Handling:
//
//	fragments := map[string]io.Reader{
//	    "Header": headerFile,
//	    "Footer": footerFile,
//	}
//	output, err := tmpl.RenderWithFragments(data, fragments)
//
// # Performance
//
// Templates are compiled during preparation and can be reused multiple times with different data.
// Enable caching to avoid re-parsing templates:
//
//	stencil.SetGlobalConfig(&stencil.Config{
//	    CacheMaxSize:    100,
//	    CacheExpiration: 1 * time.Hour,
//	})
//
// Benchmarks show rendering performance of 10,000+ documents per second for typical templates.
//
// # Error Handling
//
// The package defines several error types for specific failure cases:
//
//   - TemplateError: Template syntax errors
//   - RenderError: Errors during rendering
//   - ParseError: Expression parsing errors
//
// Check error types using errors.As():
//
//	if errors.As(err, &stencil.TemplateError{}) {
//	    // Handle template syntax error
//	}
//
// # Thread Safety
//
// PreparedTemplate is safe for concurrent use. Multiple goroutines can call Render()
// on the same template simultaneously. The Engine and its cache are also thread-safe.
//
// # DOCX File Structure
//
// DOCX files are ZIP archives containing XML files. The main document content is in
// word/document.xml. Go-stencil parses this XML, processes templates, and generates
// a new DOCX file with the rendered content.
//
// Key XML structures:
//   - Document: Top-level container
//   - Body: Document body containing elements
//   - Paragraph: Text paragraph with formatting
//   - Run: Sequence of text with consistent formatting
//   - Table: Table with rows and cells
//
// # Limitations
//
// Some DOCX features are not yet fully supported:
//   - Complex table merging (partial support)
//   - Custom XML parts
//   - Embedded objects (charts, diagrams)
//   - Track changes and comments
//
// # See Also
//
// For more examples and detailed documentation:
//   - README.md: Comprehensive guide and feature list
//   - examples/: Example templates and usage patterns
//   - CLAUDE.md: Development philosophy and roadmap
package stencil
