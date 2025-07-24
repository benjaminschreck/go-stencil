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

// TestCustomFunctionsTemplate runs comprehensive tests for the custom_functions.docx template
func TestCustomFunctionsTemplate(t *testing.T) {
	// Run subtests for different aspects of the template
	t.Run("Custom Function Rendering", testCustomFunctionRendering)
	t.Run("Built-in Function Rendering", testBuiltInFunctionRendering)
	t.Run("Function Preservation", testFunctionPreservation)
}

// testCustomFunctionRendering verifies that custom functions work correctly
func testCustomFunctionRendering(t *testing.T) {
	// Remove existing output file if it exists
	outputPath := "output/custom_functions_output.docx"
	os.Remove(outputPath)

	// Create engine with custom functions
	engine := stencil.NewWithOptions(
		stencil.WithCache(100),
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	// Run the custom functions example
	customFunctionsExample(engine)

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

	// Test custom greeting function outputs
	// The template has {{greeting(user.firstName, user.title)}} which should output "Hello, Dr. Alice!"
	expectedGreetings := []string{
		"Hello, Dr. Alice!",  // greeting(user.firstName, user.title)
	}

	for _, expected := range expectedGreetings {
		if !strings.Contains(documentXML, expected) {
			t.Errorf("Expected greeting '%s' not found in document.xml", expected)
		}
	}

	// Test timestamp function - should contain current date/time
	// Check for year to verify timestamp was rendered
	currentYear := time.Now().Format("2006")
	if !strings.Contains(documentXML, currentYear) {
		t.Error("Timestamp function output not found - expected current year")
	}

	// Verify no template markers remain
	if strings.Contains(documentXML, "{{") || strings.Contains(documentXML, "}}") {
		t.Error("Template markers found in output - template was not fully rendered")
	}
}

// testBuiltInFunctionRendering verifies built-in functions work correctly
func testBuiltInFunctionRendering(t *testing.T) {
	// Open the output file
	outputPath := "output/custom_functions_output.docx"
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
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read document.xml: %v", err)
			}
			documentXML = string(content)
			break
		}
	}

	// Based on the template content, we test what's actually there:
	// - User data rendering (Dr. Alice Johnson)
	// - Numbers loop rendering (10 20 30 40 50)
	// - Price rendering ($123.456)
	// - Items loop rendering (Apple, Banana, Cherry, Date)
	
	// Test user data rendering
	if !strings.Contains(documentXML, "Dr. Alice Johnson") {
		t.Error("User data not rendered correctly - expected 'Dr. Alice Johnson'")
	}

	// Test numbers loop - all numbers should be present
	numbers := []string{"10", "20", "30", "40", "50"}
	for _, num := range numbers {
		if !strings.Contains(documentXML, num) {
			t.Errorf("Number '%s' not found in rendered output", num)
		}
	}

	// Test price rendering
	if !strings.Contains(documentXML, "$123.456") {
		t.Error("Price not rendered correctly - expected '$123.456'")
	}

	// Test items loop - all items should be present
	items := []string{"Apple", "Banana", "Cherry", "Date"}
	for _, item := range items {
		if !strings.Contains(documentXML, item) {
			t.Errorf("Item '%s' not found in rendered output", item)
		}
	}
	
	// Test that loops were properly expanded
	// The template has "- {{item}}" which should appear as "- Apple", "- Banana", etc.
	for _, item := range items {
		expectedBullet := "- " + item
		if !strings.Contains(documentXML, expectedBullet) {
			t.Errorf("Bulleted item '%s' not found in rendered output", expectedBullet)
		}
	}
}

// testFunctionPreservation analyzes template vs output structure
func testFunctionPreservation(t *testing.T) {
	// ========================================
	// PART 1: ANALYZE THE INPUT TEMPLATE FILE
	// ========================================
	
	// First, analyze the input template
	templatePath := "custom_functions.docx"
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

	// Count various elements in template
	var templateParagraphs int
	var templateRuns int
	var templateTables int
	var templateRows int
	var templateFunctionCalls int

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
			templateParagraphs = strings.Count(docXML, "<w:p>") + strings.Count(docXML, "<w:p ")
			templateRuns = strings.Count(docXML, "<w:r>") + strings.Count(docXML, "<w:r ")
			templateTables = strings.Count(docXML, "<w:tbl>") + strings.Count(docXML, "<w:tbl ")
			templateRows = strings.Count(docXML, "<w:tr>") + strings.Count(docXML, "<w:tr ")
			
			// Count function calls (looking for patterns like {{functionName(
			templateFunctionCalls = strings.Count(docXML, "{{") // Rough count of template expressions
			
			t.Logf("Template structure: %d paragraphs, %d runs, %d tables, %d rows, ~%d template expressions",
				templateParagraphs, templateRuns, templateTables, templateRows, templateFunctionCalls)
		}
	}

	// ========================================
	// PART 2: ANALYZE THE OUTPUT FILE
	// ========================================
	
	// Analyze the output
	outputPath := "output/custom_functions_output.docx"
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

	// Count elements in output
	var outputParagraphs int
	var outputRuns int
	var outputTables int
	var outputRows int
	var outputTemplateTags int

	// Check document.xml in output
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
			outputParagraphs = strings.Count(docXML, "<w:p>") + strings.Count(docXML, "<w:p ")
			outputRuns = strings.Count(docXML, "<w:r>") + strings.Count(docXML, "<w:r ")
			outputTables = strings.Count(docXML, "<w:tbl>") + strings.Count(docXML, "<w:tbl ")
			outputRows = strings.Count(docXML, "<w:tr>") + strings.Count(docXML, "<w:tr ")
			outputTemplateTags = strings.Count(docXML, "{{")
			
			t.Logf("Output structure: %d paragraphs, %d runs, %d tables, %d rows, %d remaining template tags",
				outputParagraphs, outputRuns, outputTables, outputRows, outputTemplateTags)
		}
	}

	// ========================================
	// PART 3: COMPARE TEMPLATE VS OUTPUT
	// ========================================
	
	t.Log("\n=== Document Structure Preservation Summary ===")
	t.Logf("Template paragraphs: %d, Output paragraphs: %d", templateParagraphs, outputParagraphs)
	t.Logf("Template tables: %d, Output tables: %d", templateTables, outputTables)
	t.Logf("Template rows: %d, Output rows: %d", templateRows, outputRows)
	
	// Verify tables are preserved
	if outputTables != templateTables {
		t.Errorf("Table count mismatch: template had %d, output has %d", 
			templateTables, outputTables)
	}

	// Verify no template markers remain
	if outputTemplateTags > 0 {
		t.Errorf("Found %d template markers in output - rendering incomplete", outputTemplateTags)
	}

	// Paragraphs might change slightly due to rendering, but shouldn't be drastically different
	paragraphDiff := outputParagraphs - templateParagraphs
	if paragraphDiff < -5 || paragraphDiff > 5 {
		t.Logf("WARNING: Large paragraph count difference: %d", paragraphDiff)
	}

	if outputTables == templateTables && outputTemplateTags == 0 {
		t.Log("✓ Document structure preserved and all functions rendered successfully!")
	}
}