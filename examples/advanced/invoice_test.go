package main

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil"
)

func TestInvoiceOutput(t *testing.T) {
	// ========================================
	// PART 1: ANALYZE THE INPUT TEMPLATE FILE
	// ========================================
	templatePath := "invoice.docx"
	templateFile, err := os.Open(templatePath)
	if err != nil {
		t.Fatalf("Failed to open template: %v", err)
	}
	defer templateFile.Close()

	// Get template file info
	templateInfo, err := templateFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat template file: %v", err)
	}

	// Create zip reader for template
	templateZipReader, err := zip.NewReader(templateFile, templateInfo.Size())
	if err != nil {
		t.Fatalf("Failed to create zip reader for template: %v", err)
	}

	// Read template document.xml
	var templateDocXML string
	for _, f := range templateZipReader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open template document.xml: %v", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("Failed to read template document.xml: %v", err)
			}
			templateDocXML = string(content)
			break
		}
	}

	if templateDocXML == "" {
		t.Fatal("document.xml not found in template file")
	}

	// Count template markers and structures in template
	templateMarkers := strings.Count(templateDocXML, "{{")
	templateTables := strings.Count(templateDocXML, "<w:tbl>")
	templateRows := strings.Count(templateDocXML, "<w:tr>")
	
	t.Logf("Template analysis:")
	t.Logf("- Template markers: %d", templateMarkers)
	t.Logf("- Tables: %d", templateTables)
	t.Logf("- Table rows: %d", templateRows)

	// Check for specific template features
	hasForLoop := strings.Contains(templateDocXML, "{{for") && strings.Contains(templateDocXML, "{{end}}")
	hasIfCondition := strings.Contains(templateDocXML, "{{if") && strings.Contains(templateDocXML, "{{end}}")
	
	t.Logf("- Has for loop: %v", hasForLoop)
	t.Logf("- Has if condition: %v", hasIfCondition)

	// ========================================
	// PART 2: RUN THE TEMPLATE RENDERING
	// ========================================
	outputPath := "output/invoice_output.docx"
	os.Remove(outputPath) // Clean previous output

	// Create engine
	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	// Run the invoice example
	loopsExample(engine)

	// Check if output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Expected output file %s was not created", outputPath)
	}

	// ========================================
	// PART 3: ANALYZE THE OUTPUT FILE
	// ========================================
	outputFile, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer outputFile.Close()

	// Get output file info
	outputInfo, err := outputFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}

	// Create zip reader for output
	outputZipReader, err := zip.NewReader(outputFile, outputInfo.Size())
	if err != nil {
		t.Fatalf("Failed to create zip reader for output: %v", err)
	}

	// Read output document.xml
	var outputDocXML string
	for _, f := range outputZipReader.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open output document.xml: %v", err)
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("Failed to read output document.xml: %v", err)
			}
			outputDocXML = string(content)
			break
		}
	}

	if outputDocXML == "" {
		t.Fatal("document.xml not found in output file")
	}

	// ========================================
	// PART 4: COMPARE TEMPLATE VS OUTPUT
	// ========================================
	
	// Verify no template markers remain
	if strings.Contains(outputDocXML, "{{") || strings.Contains(outputDocXML, "}}") {
		t.Error("Template markers found in output - template was not fully rendered")
	}

	// Verify expected invoice data was rendered
	expectedStrings := []string{
		"INV-2024-001",           // invoice number
		"Jane Smith",             // customer name
		"123 Main St, Anytown, USA", // customer address
		"jane@example.com",       // customer email
		"Widget A",               // first item
		"Gadget B",               // second item
		"Service C",              // third item
		"548.85",                 // subtotal
		"54.89",                  // tax
		"603.74",                 // total
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputDocXML, expected) {
			t.Errorf("Expected string '%s' not found in output", expected)
		}
	}

	// Test loop rendering - verify each item appears exactly once
	items := []struct {
		description string
		quantity    string
		price       string
		total       string
	}{
		{"Widget A", "10", "19.99", "199.9"}, // Note: might be formatted without trailing zero
		{"Gadget B", "5", "49.99", "249.95"},
		{"Service C", "1", "99", "99"}, // Note: might be formatted without .00
	}

	for _, item := range items {
		descCount := strings.Count(outputDocXML, item.description)
		if descCount != 1 {
			t.Errorf("Item '%s' appears %d times, expected 1", item.description, descCount)
		}
		
		// Verify associated values appear
		if !strings.Contains(outputDocXML, item.quantity) {
			t.Errorf("Quantity '%s' for item '%s' not found", item.quantity, item.description)
		}
		
		// For prices and totals, check both with and without decimal places
		priceFound := strings.Contains(outputDocXML, item.price) || 
			strings.Contains(outputDocXML, item.price+".00") ||
			strings.Contains(outputDocXML, item.price+"0") // for 19.990 -> 19.99
		if !priceFound {
			t.Errorf("Price '%s' for item '%s' not found", item.price, item.description)
		}
		
		totalFound := strings.Contains(outputDocXML, item.total) || 
			strings.Contains(outputDocXML, item.total+".00") ||
			strings.Contains(outputDocXML, item.total+"0") // for 199.90 -> 199.9
		if !totalFound {
			t.Errorf("Total '%s' for item '%s' not found", item.total, item.description)
		}
	}

	// Test conditional rendering - check if "PAID" appears when isPaid is true
	if strings.Contains(outputDocXML, "PAID") {
		t.Log("✓ Conditional 'PAID' text correctly rendered")
	} else {
		t.Log("Note: 'PAID' text not found - verify if conditional rendering is implemented")
	}

	// Note: This invoice template doesn't use actual Word tables (<w:tbl>)
	// It appears to use paragraph-based layout instead
	outputParagraphs := strings.Count(outputDocXML, "<w:p>")
	if outputParagraphs == 0 {
		t.Error("No paragraphs found in output")
	} else {
		t.Logf("Output contains %d paragraphs", outputParagraphs)
	}

	// Check formatting preservation
	if strings.Contains(outputDocXML, "<w:b/>") || strings.Contains(outputDocXML, "<w:b ") {
		t.Log("✓ Bold formatting preserved")
	}

	// Since the template uses paragraphs instead of tables, 
	// verify that the loop generated multiple item entries
	widgetCount := strings.Count(outputDocXML, "Widget A")
	gadgetCount := strings.Count(outputDocXML, "Gadget B") 
	serviceCount := strings.Count(outputDocXML, "Service C")
	
	totalItemsRendered := widgetCount + gadgetCount + serviceCount
	if totalItemsRendered == 3 {
		t.Logf("✓ All 3 items from the loop were rendered")
	} else {
		t.Errorf("Expected 3 items to be rendered, but found %d", totalItemsRendered)
	}

	// Additional formatting checks
	if strings.Contains(outputDocXML, "<w:jc") {
		t.Log("✓ Text alignment formatting preserved")
	}

	// File size sanity check
	if outputInfo.Size() == 0 {
		t.Error("Output file is empty")
	} else if outputInfo.Size() < 1000 {
		t.Errorf("Output file seems too small: %d bytes", outputInfo.Size())
	} else {
		t.Logf("✓ Output file size: %d bytes", outputInfo.Size())
	}
}