// Advanced example demonstrating the full capabilities of go-stencil
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benjaminschreck/go-stencil/pkg/stencil"
)

// CustomFunction demonstrates how to create a custom function
type GreetingFunction struct{}

func (f GreetingFunction) Name() string {
	return "greeting"
}

func (f GreetingFunction) Call(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return "Hello, World!", nil
	}

	name, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("greeting expects a string argument")
	}

	if len(args) >= 2 {
		title, ok := args[1].(string)
		if ok {
			return fmt.Sprintf("Hello, %s %s!", title, name), nil
		}
	}

	return fmt.Sprintf("Hello, %s!", name), nil
}

func (f GreetingFunction) MinArgs() int {
	return 0 // Can be called with no arguments
}

func (f GreetingFunction) MaxArgs() int {
	return 2 // Accepts up to 2 arguments: name and optional title
}

// CustomFunctionProvider provides a suite of custom functions
type CustomFunctionProvider struct{}

func (p CustomFunctionProvider) ProvideFunctions() map[string]stencil.Function {
	return map[string]stencil.Function{
		"greeting":  GreetingFunction{},
		"timestamp": TimestampFunction{},
	}
}

// TimestampFunction returns the current timestamp
type TimestampFunction struct{}

func (f TimestampFunction) Name() string {
	return "timestamp"
}

func (f TimestampFunction) Call(args ...interface{}) (interface{}, error) {
	format := "2006-01-02 15:04:05"
	if len(args) > 0 {
		if f, ok := args[0].(string); ok {
			format = f
		}
	}
	return time.Now().Format(format), nil
}

func (f TimestampFunction) MinArgs() int {
	return 0 // Can be called with no arguments
}

func (f TimestampFunction) MaxArgs() int {
	return 1 // Accepts up to 1 argument: optional format string
}

func main() {
	// Create a new engine with custom configuration
	engine := stencil.NewWithOptions(
		stencil.WithCache(100), // Enable caching with max 100 templates
		stencil.WithFunction("greeting", GreetingFunction{}),
		stencil.WithFunctionProvider(CustomFunctionProvider{}),
	)
	defer engine.Close()

	// Example 1: Basic template rendering
	fmt.Println("=== Example 1: Basic Template ===")
	basicExample(engine)

	// Example 2: Template with loops and conditionals
	fmt.Println("\n=== Example 2: Loops and Conditionals ===")
	loopsExample(engine)

	// Example 3: Using custom functions
	fmt.Println("\n=== Example 3: Custom Functions ===")
	customFunctionsExample(engine)

	// Example 4: Using fragments
	fmt.Println("\n=== Example 4: Fragments ===")
	fragmentsExample(engine)

	// Example 5: Table operations
	fmt.Println("\n=== Example 5: Table Operations ===")
	tableExample(engine)
}

func basicExample(engine *stencil.Engine) {
	// Prepare template
	tmpl, err := engine.PrepareFile("basic.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Create data
	data := stencil.TemplateData{
		"name":     "John Doe",
		"company":  "Acme Corp",
		"date":     time.Now(),
		"position": "Software Engineer",
	}

	// Render
	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	// Save output
	saveOutput(output, "output/basic_output.docx")
}

func loopsExample(engine *stencil.Engine) {
	tmpl, err := engine.PrepareFile("invoice.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Create invoice data
	data := stencil.TemplateData{
		"invoiceNumber": "INV-2024-001",
		"date":          time.Now(),
		"customer": map[string]interface{}{
			"name":    "Jane Smith",
			"address": "123 Main St, Anytown, USA",
			"email":   "jane@example.com",
		},
		"items": []map[string]interface{}{
			{
				"description": "Widget A",
				"quantity":    10,
				"price":       19.99,
				"total":       199.90,
			},
			{
				"description": "Gadget B",
				"quantity":    5,
				"price":       49.99,
				"total":       249.95,
			},
			{
				"description": "Service C",
				"quantity":    1,
				"price":       99.00,
				"total":       99.00,
			},
		},
		"subtotal": 548.85,
		"tax":      54.89,
		"total":    603.74,
		"isPaid":   true,
	}

	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	saveOutput(output, "output/invoice_output.docx")
}

func customFunctionsExample(engine *stencil.Engine) {
	tmpl, err := engine.PrepareFile("custom_functions.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	data := stencil.TemplateData{
		"user": map[string]interface{}{
			"firstName": "Alice",
			"lastName":  "Johnson",
			"title":     "Dr.",
		},
		"numbers": []int{10, 20, 30, 40, 50},
		"price":   123.456,
		"items": []string{
			"Apple",
			"Banana",
			"Cherry",
			"Date",
		},
	}

	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	saveOutput(output, "output/custom_functions_output.docx")
}

func fragmentsExample(engine *stencil.Engine) {
	tmpl, err := engine.PrepareFile("report.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Add text fragments
	err = tmpl.AddFragment("disclaimer", "This report is confidential and proprietary. Do not distribute without authorization.")
	if err != nil {
		log.Fatalf("Failed to add disclaimer fragment: %v", err)
	}

	err = tmpl.AddFragment("copyright", fmt.Sprintf("© %d Acme Corporation. All rights reserved.", time.Now().Year()))
	if err != nil {
		log.Fatalf("Failed to add copyright fragment: %v", err)
	}

	// Load all DOCX fragments from the fragments folder
	fragmentsDir := "fragments"
	entries, err := os.ReadDir(fragmentsDir)
	if err != nil {
		fmt.Printf("Warning: Could not read fragments directory: %v\n", err)
	} else {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".docx") {
				// Get fragment name (filename without extension)
				fragmentName := strings.TrimSuffix(entry.Name(), ".docx")
				
				// Read the fragment file
				fragmentPath := filepath.Join(fragmentsDir, entry.Name())
				fragmentBytes, err := os.ReadFile(fragmentPath)
				if err != nil {
					fmt.Printf("Warning: Could not read fragment %s: %v\n", entry.Name(), err)
					continue
				}
				
				// Add the fragment
				err = tmpl.AddFragmentFromBytes(fragmentName, fragmentBytes)
				if err != nil {
					fmt.Printf("Warning: Could not add fragment %s: %v\n", fragmentName, err)
					continue
				}
				
				fmt.Printf("Added fragment: %s\n", fragmentName)
			}
		}
	}
	
	// If no fragments were loaded and header.docx doesn't exist, add a default text fragment
	if len(entries) == 0 || err != nil {
		// Check if we need to add a default header fragment
		headerPath := filepath.Join(fragmentsDir, "header.docx")
		if _, err := os.Stat(headerPath); os.IsNotExist(err) {
			err = tmpl.AddFragment("header", "ACME CORPORATION\n================\nConfidential Report")
			if err != nil {
				log.Fatalf("Failed to add header fragment: %v", err)
			}
			fmt.Println("Added default text header fragment")
		}
	}

	data := stencil.TemplateData{
		"title":   "Quarterly Report",
		"quarter": "Q4",
		"year":    2024,
		"author":  "Analytics Team",
		"highlights": []string{
			"Revenue increased by 25%",
			"Customer satisfaction at all-time high",
			"New product launch successful",
		},
		"personal_contact": map[string]interface{}{
			"name":    "John Doe",
			"address": "123 Main St, Anytown, USA",
			"email":   "john.doe@example.com",
			"phone":   "123-456-7890",
		},
	}

	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	saveOutput(output, "output/report_output.docx")
}

func tableExample(engine *stencil.Engine) {
	tmpl, err := engine.PrepareFile("table_demo.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	data := stencil.TemplateData{
		"title":       "Sales Report",
		"showDetails": true,
		"salesData": []map[string]interface{}{
			{
				"region":  "North",
				"q1Sales": 100000,
				"q2Sales": 120000,
				"q3Sales": 110000,
				"q4Sales": 150000,
				"total":   480000,
				"showRow": true,
			},
			{
				"region":  "South",
				"q1Sales": 80000,
				"q2Sales": 85000,
				"q3Sales": 90000,
				"q4Sales": 95000,
				"total":   350000,
				"showRow": true,
			},
			{
				"region":  "East",
				"q1Sales": 95000,
				"q2Sales": 100000,
				"q3Sales": 105000,
				"q4Sales": 110000,
				"total":   410000,
				"showRow": false, // This row will be hidden (Note: conditional rows in tables may not work as expected)
			},
			{
				"region":  "West",
				"q1Sales": 110000,
				"q2Sales": 115000,
				"q3Sales": 120000,
				"q4Sales": 125000,
				"total":   470000,
				"showRow": true,
			},
		},
		"grandTotal": 1710000,
	}

	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	saveOutput(output, "output/table_demo_output.docx")
}

func saveOutput(reader io.Reader, filename string) {
	// Ensure output directory exists
	if err := os.MkdirAll("output", 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Create output file
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	// Copy rendered content to file
	_, err = io.Copy(file, reader)
	if err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	fmt.Printf("Output saved to: %s\n", filename)
}
