package main

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benjaminschreck/go-stencil/pkg/stencil"
)

// TestBasicTemplate runs comprehensive tests for the basic.docx template
func TestBasicTemplate(t *testing.T) {
	// Run subtests for different aspects of the template
	t.Run("Content Rendering", testBasicContentRendering)
	t.Run("Hyperlink Preservation", testBasicHyperlinkPreservation)
}

// testBasicContentRendering verifies that template variables are properly rendered
func testBasicContentRendering(t *testing.T) {
	// Remove existing output file if it exists
	outputPath := "output/basic_output.docx"
	os.Remove(outputPath)

	// Create engine
	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	// Run the basic example
	basicExample(engine)

	// Check if output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Expected output file %s was not created", outputPath)
	}

	// Open and verify the output file contains expected content
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer file.Close()

	// Read file info
	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// Create zip reader
	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

	// Find and read document.xml
	var documentXML string
	for _, f := range zipReader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open document.xml: %v", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("Failed to read document.xml: %v", err)
			}
			documentXML = string(content)
			break
		}
	}

	if documentXML == "" {
		t.Fatal("document.xml not found in output file")
	}

	// Verify expected content was rendered
	expectedStrings := []string{
		"John Doe",                // name variable
		"Acme Corp",               // company variable
		"Software Engineer",       // position variable
		time.Now().Format("2006"), // year from date
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(documentXML, expected) {
			t.Errorf("Expected string '%s' not found in document.xml", expected)
		}
	}

	// Verify no template markers remain
	if strings.Contains(documentXML, "{{") || strings.Contains(documentXML, "}}") {
		t.Error("Template markers found in output - template was not fully rendered")
	}
}

// testBasicHyperlinkPreservation verifies that hyperlinks and their styling are preserved
func testBasicHyperlinkPreservation(t *testing.T) {
	// ========================================
	// PART 1: ANALYZE THE INPUT TEMPLATE FILE
	// ========================================
	
	// First, analyze the input template
	templatePath := "basic.docx"
	templateFile, err := os.Open(templatePath)
	if err != nil {
		t.Fatalf("Failed to open template file: %v", err)
	}
	defer templateFile.Close()

	templateInfo, err := templateFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat template file: %v", err)
	}

	// Read template as ZIP
	templateZip, err := zip.NewReader(templateFile, templateInfo.Size())
	if err != nil {
		t.Fatalf("Failed to read template as ZIP: %v", err)
	}

	// Count hyperlinks in template
	var templateHyperlinkCount int
	var templateHyperlinkRels int
	var templateURLs []string

	// === CHECKING TEMPLATE FILE ===
	// Check document.xml in template
	for _, f := range templateZip.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open template document.xml: %v", err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read template document.xml: %v", err)
			}
			
			docXML := string(content)
			templateHyperlinkCount = strings.Count(docXML, "<w:hyperlink")
			t.Logf("Template document.xml contains %d hyperlink(s)", templateHyperlinkCount)
		}
		
		if f.Name == "word/_rels/document.xml.rels" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open template document.xml.rels: %v", err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read template document.xml.rels: %v", err)
			}
			
			relsXML := string(content)
			
			// Extract URLs from relationships and count external hyperlinks
			// Split by relationship elements instead of lines since XML may be on one line
			parts := strings.Split(relsXML, "<Relationship")
			templateHyperlinkRels = 0
			for i := 1; i < len(parts); i++ {
				relElement := "<Relationship" + parts[i]
				if strings.Contains(relElement, "Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink\"") && strings.Contains(relElement, "TargetMode=\"External\"") {
					templateHyperlinkRels++
					if idx := strings.Index(relElement, "Target=\""); idx >= 0 {
						start := idx + 8
						end := strings.Index(relElement[start:], "\"")
						if end > 0 {
							url := relElement[start:start+end]
							templateURLs = append(templateURLs, url)
							t.Logf("Template contains hyperlink to: %s", url)
						}
					}
				}
			}
			t.Logf("Template document.xml.rels contains %d hyperlink relationship(s)", templateHyperlinkRels)
		}
	}

	// ========================================
	// PART 2: RUN THE TEMPLATE RENDERING
	// ========================================
	
	// Now run the rendering
	outputPath := "output/basic_output.docx"
	os.Remove(outputPath)

	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	basicExample(engine)

	// ========================================
	// PART 3: ANALYZE THE OUTPUT FILE
	// ========================================
	
	// Analyze the output
	outputFile, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer outputFile.Close()

	outputInfo, err := outputFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	outputZip, err := zip.NewReader(outputFile, outputInfo.Size())
	if err != nil {
		t.Fatalf("Failed to read output as ZIP: %v", err)
	}

	// Count hyperlinks in output
	var outputHyperlinkCount int
	var outputHyperlinkRels int
	var outputURLs []string
	var outputHasHyperlinkStyle bool

	// === CHECKING OUTPUT FILE ===
	for _, f := range outputZip.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open output document.xml: %v", err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read output document.xml: %v", err)
			}
			
			docXML := string(content)
			outputHyperlinkCount = strings.Count(docXML, "<w:hyperlink")
			t.Logf("Output document.xml contains %d hyperlink(s)", outputHyperlinkCount)
			
			// Check if hyperlink styling is preserved
			// Note: The style element might have a space before the closing tag
			if strings.Contains(docXML, "<w:rStyle w:val=\"Hyperlink\"") {
				outputHasHyperlinkStyle = true
				t.Log("Output hyperlinks have style preserved")
			} else {
				t.Log("WARNING: Output hyperlinks are missing style")
			}
		}
		
		if f.Name == "word/_rels/document.xml.rels" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open output document.xml.rels: %v", err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read output document.xml.rels: %v", err)
			}
			
			relsXML := string(content)
			
			// Extract URLs and count external hyperlinks
			// Split by relationship elements instead of lines since XML may be on one line
			parts := strings.Split(relsXML, "<Relationship")
			outputHyperlinkRels = 0
			for i := 1; i < len(parts); i++ {
				relElement := "<Relationship" + parts[i]
				if strings.Contains(relElement, "Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink\"") && strings.Contains(relElement, "TargetMode=\"External\"") {
					outputHyperlinkRels++
					if idx := strings.Index(relElement, "Target=\""); idx >= 0 {
						start := idx + 8
						end := strings.Index(relElement[start:], "\"")
						if end > 0 {
							url := relElement[start:start+end]
							outputURLs = append(outputURLs, url)
							t.Logf("Output contains hyperlink to: %s", url)
						}
					}
				}
			}
			t.Logf("Output document.xml.rels contains %d hyperlink relationship(s)", outputHyperlinkRels)
		}
	}

	// ========================================
	// PART 4: COMPARE TEMPLATE VS OUTPUT
	// ========================================
	
	// Compare counts
	t.Log("\n=== Hyperlink Preservation Summary ===")
	t.Logf("Template hyperlinks in document.xml: %d", templateHyperlinkCount)
	t.Logf("Output hyperlinks in document.xml: %d", outputHyperlinkCount)
	t.Logf("Template hyperlink relationships: %d", templateHyperlinkRels)
	t.Logf("Output hyperlink relationships: %d", outputHyperlinkRels)
	t.Logf("Output has hyperlink styling: %v", outputHasHyperlinkStyle)

	// Verify preservation
	if outputHyperlinkCount != templateHyperlinkCount {
		t.Errorf("Hyperlink count mismatch: template had %d, output has %d", 
			templateHyperlinkCount, outputHyperlinkCount)
	}

	if outputHyperlinkRels != templateHyperlinkRels {
		t.Errorf("Hyperlink relationship count mismatch: template had %d, output has %d", 
			templateHyperlinkRels, outputHyperlinkRels)
	}

	// Verify URLs are preserved
	if len(outputURLs) != len(templateURLs) {
		t.Errorf("URL count mismatch: template had %d URLs, output has %d", 
			len(templateURLs), len(outputURLs))
	} else {
		for i, url := range templateURLs {
			if i < len(outputURLs) && outputURLs[i] != url {
				t.Errorf("URL mismatch at position %d: template had '%s', output has '%s'", 
					i, url, outputURLs[i])
			}
		}
	}

	// Verify hyperlink styling is preserved
	if !outputHasHyperlinkStyle && templateHyperlinkCount > 0 {
		t.Error("Hyperlink styling was not preserved in output")
	}
	
	if outputHyperlinkCount == templateHyperlinkCount && 
	   outputHyperlinkRels == templateHyperlinkRels && 
	   len(outputURLs) == len(templateURLs) &&
	   outputHasHyperlinkStyle {
		t.Log("✓ All hyperlinks and styling preserved successfully!")
	}
}