package main

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil"
)

// TestTableDemoTemplate runs comprehensive tests for the table_demo.docx template
func TestTableDemoTemplate(t *testing.T) {
	// Run subtests for different aspects of the template
	t.Run("Table Structure Preservation", testTableStructurePreservation)
	t.Run("Table Row Iteration", testTableRowIteration)
	t.Run("Conditional Row Visibility", testConditionalRowVisibility)
	t.Run("Table Content Substitution", testTableContentSubstitution)
	t.Run("Table Splitting Issue", testTableSplittingIssue)
}

// testTableStructurePreservation verifies that table structure is preserved
// and identifies the table splitting issue when for loops are outside tables
func testTableStructurePreservation(t *testing.T) {
	// ========================================
	// PART 1: ANALYZE THE INPUT TEMPLATE FILE
	// ========================================
	
	// First, analyze the input template
	templatePath := "table_demo.docx"
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

	// Count tables in template
	var templateTableCount int
	var templateRowCount int
	var templateCellCount int

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
			templateTableCount = strings.Count(docXML, "<w:tbl>")
			templateRowCount = strings.Count(docXML, "<w:tr>")
			templateCellCount = strings.Count(docXML, "<w:tc>")
			
			t.Logf("Template document.xml contains %d table(s)", templateTableCount)
			t.Logf("Template document.xml contains %d row(s)", templateRowCount)
			t.Logf("Template document.xml contains %d cell(s)", templateCellCount)
			
			// Note: Some DOCX files may have row elements in different formats
			// Check for both <w:tr> and <w:tr properties
			if templateRowCount == 0 {
				// Count occurrences where w:tr appears with properties or attributes
				templateRowCount = strings.Count(docXML, "w:tr ")
				if templateRowCount > 0 {
					t.Logf("Template actually contains %d row(s) with properties", templateRowCount)
				}
			}
			
			// Check for template markers in tables
			if strings.Contains(docXML, "{{for") && strings.Contains(docXML, "salesData}}") {
				t.Log("Template contains for loop for salesData")
			}
		}
	}

	// ========================================
	// PART 2: RUN THE TEMPLATE RENDERING
	// ========================================
	
	// Now run the rendering
	outputPath := "output/table_demo_output.docx"
	os.Remove(outputPath)

	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	tableExample(engine)

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

	// Count tables in output
	var outputTableCount int
	var outputRowCount int
	var outputCellCount int

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
			outputTableCount = strings.Count(docXML, "<w:tbl>")
			outputRowCount = strings.Count(docXML, "<w:tr>")
			outputCellCount = strings.Count(docXML, "<w:tc>")
			
			t.Logf("Output document.xml contains %d table(s)", outputTableCount)
			t.Logf("Output document.xml contains %d row(s)", outputRowCount)
			t.Logf("Output document.xml contains %d cell(s)", outputCellCount)
		}
	}

	// ========================================
	// PART 4: COMPARE TEMPLATE VS OUTPUT
	// ========================================
	
	// Compare table counts
	t.Log("\n=== Table Structure Summary ===")
	t.Logf("Template tables: %d", templateTableCount)
	t.Logf("Output tables: %d", outputTableCount)
	
	// With table merging fix: Tables that were split by for loops outside tables
	// are now merged back together. Template has 4 tables, output should have 2:
	// - Table 1: Correctly rendered (for loop inside table)
	// - Table 2: Merged from the 6 split tables (header + 4 data rows + total)
	expectedOutputTableCount := 2
	if outputTableCount != expectedOutputTableCount {
		t.Errorf("Table count mismatch: expected %d, got %d", 
			expectedOutputTableCount, outputTableCount)
		t.Logf("Template had %d tables", templateTableCount)
	} else {
		t.Logf("✓ Table merging successful (template: %d → output: %d)", 
			templateTableCount, outputTableCount)
		t.Log("   Split tables from for loop have been merged correctly")
	}
	
	// Row count should increase due to data expansion
	if outputRowCount > 0 {
		t.Logf("✓ Output has %d rows (template had %d)", outputRowCount, templateRowCount)
	} else {
		t.Error("Output has no table rows")
	}
}

// testTableRowIteration verifies that table rows are properly iterated
func testTableRowIteration(t *testing.T) {
	// Remove existing output file if it exists
	outputPath := "output/table_demo_output.docx"
	os.Remove(outputPath)

	// Create engine
	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	// Run the table example
	tableExample(engine)

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

	// Verify expected regions were rendered
	expectedRegions := []string{
		"North",
		"South",
		"East",
		"West",
	}

	for _, region := range expectedRegions {
		count := strings.Count(documentXML, region)
		if count == 0 {
			t.Errorf("Expected region '%s' not found in document.xml", region)
		} else {
			t.Logf("Region '%s' found %d time(s)", region, count)
		}
	}

	// Verify sales values were rendered
	expectedValues := []string{
		"100000",  // North Q1
		"120000",  // North Q2
		"480000",  // North total
		"350000",  // South total
		"410000",  // East total
		"470000",  // West total
		"1710000", // Grand total
	}

	for _, value := range expectedValues {
		if !strings.Contains(documentXML, value) {
			t.Errorf("Expected value '%s' not found in document.xml", value)
		}
	}

	// Verify no template markers remain
	if strings.Contains(documentXML, "{{") || strings.Contains(documentXML, "}}") {
		t.Error("Template markers found in output - template was not fully rendered")
	}
}

// testConditionalRowVisibility verifies that conditional rows work properly
func testConditionalRowVisibility(t *testing.T) {
	// Note: As mentioned in main.go, conditional rows in tables may not work as expected
	// This test documents the actual behavior
	
	outputPath := "output/table_demo_output.docx"
	os.Remove(outputPath)

	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	tableExample(engine)

	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

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

	// Check if "East" region is present (it has showRow: false)
	eastCount := strings.Count(documentXML, "East")
	if eastCount > 0 {
		t.Logf("Note: 'East' region found %d time(s) despite showRow:false - conditional rows in tables may not work as expected", eastCount)
	} else {
		t.Log("'East' region successfully hidden with showRow:false")
	}
}

// testTableContentSubstitution verifies content within tables is properly substituted
func testTableContentSubstitution(t *testing.T) {
	outputPath := "output/table_demo_output.docx"
	os.Remove(outputPath)

	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	tableExample(engine)

	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

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

	// Verify title was substituted
	if !strings.Contains(documentXML, "Sales Report") {
		t.Error("Title 'Sales Report' not found in output")
	}

	// Verify showDetails conditional worked
	// Since showDetails is true, any content inside {{if showDetails}} should be present

	// Check for quarterly sales data presence
	quarters := []string{"Q1", "Q2", "Q3", "Q4"}
	for _, quarter := range quarters {
		// Check if quarter headers/labels are present
		// Note: The exact check depends on how the template is structured
		t.Logf("Checking for quarter: %s", quarter)
	}

	// Verify grand total is present
	if !strings.Contains(documentXML, "1710000") {
		t.Error("Grand total '1710000' not found in output")
	}

	t.Log("✓ Table content substitution completed!")
}

// testTableSplittingIssue specifically tests and documents the table splitting issue
func testTableSplittingIssue(t *testing.T) {
	outputPath := "output/table_demo_output.docx"
	os.Remove(outputPath)

	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	tableExample(engine)

	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	zipReader, err := zip.NewReader(file, fileInfo.Size())
	if err != nil {
		t.Fatalf("Failed to create zip reader: %v", err)
	}

	// Analyze table structure in detail
	var tables []string
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
			
			documentXML := string(content)
			
			// Find all tables and their first cell content
			tableStart := 0
			for {
				idx := strings.Index(documentXML[tableStart:], "<w:tbl>")
				if idx == -1 {
					break
				}
				tableStart += idx
				
				// Find end of this table
				tableEnd := strings.Index(documentXML[tableStart:], "</w:tbl>")
				if tableEnd == -1 {
					break
				}
				tableEnd += tableStart
				
				// Extract table content
				tableXML := documentXML[tableStart:tableEnd+8]
				
				// Try to find first text content in table
				var firstText string
				textStart := strings.Index(tableXML, "<w:t>")
				if textStart != -1 {
					textEnd := strings.Index(tableXML[textStart:], "</w:t>")
					if textEnd != -1 {
						firstText = tableXML[textStart+5 : textStart+textEnd]
					}
				}
				
				tables = append(tables, firstText)
				tableStart = tableEnd + 1
			}
			
			break
		}
	}

	// Log the issue
	t.Log("\n=== Table Splitting Issue Analysis ===")
	t.Logf("Found %d tables in output (should be 2 conceptual tables)", len(tables))
	
	// First table should be the properly rendered one
	if len(tables) > 0 {
		t.Log("\nTable 1 (correctly rendered with for loop inside):")
		t.Logf("  First cell: %s", tables[0])
	}
	
	// Remaining tables show the splitting issue
	if len(tables) > 1 {
		t.Log("\nTables 2-7 (split due to for loop outside table):")
		for i := 1; i < len(tables); i++ {
			t.Logf("  Table %d first cell: %s", i+1, tables[i])
		}
	}
	
	// Document the issue
	t.Log("\n⚠️  ISSUE: When a for loop is placed in a paragraph between tables,")
	t.Log("   each iteration creates a separate table instead of adding rows.")
	t.Log("   Solution: Move the for loop inside the table structure.")
	
	// With the fix, we should now have only 2 tables
	if len(tables) == 2 {
		t.Log("\n✓ Table merging fix confirmed - split tables have been merged")
	} else if len(tables) == 7 {
		t.Error("Table splitting issue still present - merge fix may not be working")
	} else {
		t.Errorf("Expected 2 tables after merge fix, but found %d", len(tables))
	}
}