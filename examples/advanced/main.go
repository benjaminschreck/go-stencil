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

	// Example 6: HTML formatting showcase
	fmt.Println("\n=== Example 6: HTML Formatting ===")
	htmlExample(engine)

	// Example 7: Comprehensive features showcase
	fmt.Println("\n=== Example 7: Comprehensive Features ===")
	comprehensiveFeaturesExample(engine)

	// Example 8: Production example with German legal document
	fmt.Println("\n=== Example 8: Production Example (German Legal Document) ===")
	productionExample(engine)

	// Example 9: Nested fragments showcase
	fmt.Println("\n=== Example 9: Nested Fragments Showcase ===")
	nestedFragmentsExample(engine)

	// Example 10: Template validation
	fmt.Println("\n=== Example 10: Template Validation ===")
	validationExample(engine)
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
				"people": []map[string]interface{}{
					{
						"name": "John Doe",
						"age":  30,
						"city": "New York",
					},

					{
						"name": "Jane Doe",
						"age":  25,
						"city": "Los Angeles",
					},
					{
						"name": "Jim Doe",
						"age":  35,
						"city": "Chicago",
					},
				}},
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
		"companies":  []map[string]interface{}{},
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

func htmlExample(engine *stencil.Engine) {
	tmpl, err := engine.PrepareFile("html_showcase.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Create data with various HTML examples
	data := stencil.TemplateData{
		// Dynamic HTML content
		"htmlContent": `<b>Dynamic content</b> with <i>various</i> <u>formatting</u> options and <sup>special</sup> characters`,

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
		"greeting":     "<b>Hello</b>",
		"customerName": "John Doe",
		"message":      "<i>we have an important update for you</i>",

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

	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	saveOutput(output, "output/html_showcase_output.docx")
}

func comprehensiveFeaturesExample(engine *stencil.Engine) {
	tmpl, err := engine.PrepareFile("comprehensive_features.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Add fragments for inclusion
	err = tmpl.AddFragment("header", "=== COMPREHENSIVE FEATURE TEST ===\nGenerated on: {{timestamp()}}")
	if err != nil {
		log.Fatalf("Failed to add header fragment: %v", err)
	}

	err = tmpl.AddFragment("footer", "--- End of comprehensive feature test ---\nAll features demonstrated successfully!")
	if err != nil {
		log.Fatalf("Failed to add footer fragment: %v", err)
	}

	err = tmpl.AddFragment("kopfzeile", "Kopfzeile: {{timestamp()}}")
	if err != nil {
		log.Fatalf("Failed to add header fragment: %v", err)
	}

	err = tmpl.AddFragment("fusszeile", "Fusszeile: {{timestamp()}}")
	if err != nil {
		log.Fatalf("Failed to add footer fragment: %v", err)
	}

	// Create comprehensive test data
	data := stencil.TemplateData{
		// User data for bracket notation and advanced access
		"user": map[string]interface{}{
			"firstName": "Alice",
			"lastName":  "Johnson",
			"email":     "alice@example.com",
		},

		// Array data for indexing
		"items": []map[string]interface{}{
			{"name": "Widget Pro", "price": 99.99},
			{"name": "Gadget Plus", "price": 149.99},
			{"name": "Tool Elite", "price": 199.99},
		},

		// Score for conditional examples
		"score": 85,

		// Weekend flag
		"isWeekend": false,

		// Days for indexed loop
		"weekDays": []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},

		// Permission flags for logical operators
		"isAdmin":       false,
		"isOwner":       true,
		"isLoggedIn":    true,
		"hasEditRights": true,
		"isPublic":      false,

		// Type conversion examples
		"stringNumber": "42",
		"stringPrice":  "19.99",

		// Date for formatting
		"eventDate": time.Date(2024, 12, 25, 15, 30, 0, 0, time.UTC),

		// Optional field (empty)
		"optionalField": nil,

		// Fruits for contains check
		"fruits": []string{"apple", "banana", "orange"},

		// Titles for coalesce
		"userTitle":    "",
		"defaultTitle": "Guest",

		// Status for switch
		"status": "pending",

		// String manipulation
		"description": "This is the old version of the text",
		"name":        "Go-Stencil Template Engine",
		"features":    []string{"Fast", "Flexible", "Powerful"},

		// Products for table operations
		"products": []map[string]interface{}{
			{"name": "Product A", "price": "$10", "stock": 5},
			{"name": "Product B", "price": "$20", "stock": 0}, // This row should be hidden
			{"name": "Product C", "price": "$30", "stock": 15},
			{"name": "Product D", "price": "$40", "stock": 0}, // This row should be hidden
		},

		// Column hiding flags
		"hideQ1": false,
		"hideQ2": false,
		"hideQ3": true, // Hide Q3
		"hideQ4": false,

		// replaceLink functionality
		"newWebsiteUrl": "https://github.com/benjaminschreck/go-stencil",

		// Complex expression data
		"basePrice": 100.0,
		"quantity":  3,
		"discount":  15.0,
		"taxRate":   8.5,

		// Additional permission flags
		"age":   21,
		"hasID": true,
		"isVIP": false,
	}

	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	saveOutput(output, "output/comprehensive_features_output.docx")
}

func productionExample(engine *stencil.Engine) {
	// This example uses a real-world production template
	tmpl, err := engine.PrepareFile("production_legal.docx")
	if err != nil {
		log.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	//read production_header.docx from fragments folder
	headerBytes, err := os.ReadFile("fragments/production_header.docx")
	if err != nil {
		log.Fatalf("Failed to read production header: %v", err)
	}
	err = tmpl.AddFragmentFromBytes("production_header", headerBytes)
	if err != nil {
		log.Fatalf("Failed to add production header: %v", err)
	}

	// The template contains a {{customText}} variable that can be replaced
	// with any custom text or clause
	data := stencil.TemplateData{
		"customText": `Sample text demonstrating template variable substitution.

This shows how the template engine handles multi-line text replacement.

Additional paragraphs are preserved with proper formatting.`,
	}

	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	saveOutput(output, "output/production_legal_output.docx")
}

func nestedFragmentsExample(engine *stencil.Engine) {
	// This example demonstrates nested fragments with complex data structures

	// Try to load the DOCX template, if it doesn't exist, inform the user
	tmpl, err := engine.PrepareFile("comprehensive_features_with_fragments.docx")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer tmpl.Close()

	fragmentFiles := []string{"fragment1.docx", "fragment2.docx", "fragment3.docx"}
	for _, fragFile := range fragmentFiles {
		fragPath := filepath.Join("fragments", fragFile)
		fragBytes, err := os.ReadFile(fragPath)
		if err != nil {
			fmt.Printf("Warning: Could not read %s : %v\n", fragFile, err)
			continue
		}

		fragName := strings.TrimSuffix(fragFile, ".docx")
		err = tmpl.AddFragmentFromBytes(fragName, fragBytes)
		if err != nil {
			fmt.Printf("Warning: Could not add fragment %s: %v\n", fragName, err)
			continue
		}
		fmt.Printf("Added nested fragment: %s\n", fragName)
	}

	// Create comprehensive data that will be used by all nested fragments
	data := stencil.TemplateData{
		// Fragment 1 (Company Header) data
		"showSales":          true,
		"showSupport":        true,
		"documentId":         "DOC-2025-001",
		"version":            "1.0",
		"classification":     "confidential",
		"isConfidential":     true,
		"includeSubFragment": true, // This will cause fragment1 to include fragment2

		// Fragment 2 (Product Catalog) data
		"quarter": "Q1",
		"year":    2025,
		"products": []map[string]interface{}{
			{
				"code":        "PROD-001",
				"name":        "Pro Widget",
				"category":    "electronics",
				"price":       299.99,
				"stock":       50,
				"description": "High-performance widget with advanced features",
				"features": []string{
					"<b>Dual-core processor</b>",
					"<i>Energy efficient</i>",
					"<u>2-year warranty</u>",
				},
				"hasDiscount":     true,
				"discountPercent": 15,
				"salePrice":       254.99,
			},
			{
				"code":         "PROD-002",
				"name":         "Elite Gadget",
				"category":     "accessories",
				"price":        149.99,
				"stock":        0,
				"description":  "Premium gadget for professionals",
				"features":     []string{"<b>Compact design</b>", "Water resistant"},
				"hasDiscount":  false,
				"expectedDate": "2025-02-15",
			},
			{
				"code":        "PROD-003",
				"name":        "Ultimate Tool",
				"category":    "tools",
				"price":       499.99,
				"stock":       25,
				"description": "Professional-grade tool for experts",
				"features":    []string{"<b>Lifetime warranty</b>", "<i>Premium materials</i>", "Ergonomic design"},
				"hasDiscount": false,
			},
		},
		"electronicsCategories": []map[string]interface{}{
			{"type": "Widgets", "count": 15},
			{"type": "Gadgets", "count": 8},
			{"type": "Tools", "count": 12},
		},
		"pricingTiers": []map[string]interface{}{
			{"name": "Basic", "minQty": 1, "maxQty": 10, "unitPrice": 100.00, "savingsPercent": 0.0},
			{"name": "Business", "minQty": 11, "maxQty": 50, "unitPrice": 90.00, "savingsPercent": 10.0},
			{"name": "Enterprise", "minQty": 51, "maxQty": 999, "unitPrice": 75.00, "savingsPercent": 25.0},
		},
		"includeRecommendations": true,
		"recommendations": []map[string]interface{}{
			{"name": "Accessory Pack", "reason": "Complements your selection", "price": 49.99, "inStock": true},
			{"name": "Extended Warranty", "reason": "Protect your investment", "price": 99.99, "inStock": true, "shipDays": 0},
			{"name": "Premium Case", "reason": "Safe storage", "price": 29.99, "inStock": false, "shipDays": 3},
		},
		"basePrice":     100.0,
		"quantity":      5,
		"taxRate":       8.5,
		"includeFooter": true, // This will cause fragment2 to include fragment3

		// Fragment 3 (Legal Footer) data
		"requiresFullTerms": true,
		"companyName":       "ACME Corporation",
		"companyWebsite":    "https://acme-corp.example.com",
		"acceptsLiability":  false,
		"includeGDPR":       true,
		"authorizedRecipients": []map[string]interface{}{
			{"name": "John Doe", "role": "CFO", "accessLevel": "full"},
			{"name": "Jane Smith", "role": "Legal Counsel", "accessLevel": "full"},
			{"name": "Mike Johnson", "role": "Auditor", "accessLevel": "read-only"},
		},
		"intellectualPropertyTypes": []string{
			"Copyrights",
			"Trademarks",
			"Patents",
			"Trade Secrets",
			"Design Rights",
		},
		"dataController": map[string]interface{}{
			"name":  "ACME Data Protection Officer",
			"email": "dpo@acme-corp.example.com",
		},
		"dataPurpose":     "Business document generation and processing",
		"legalBasis":      "Legitimate business interest",
		"retentionPeriod": "7 years from document creation",
		"dataSubjectRights": []map[string]interface{}{
			{"name": "Right to Access", "description": "Request a copy of your personal data", "contactRequired": true},
			{"name": "Right to Rectification", "description": "Request correction of inaccurate data", "contactRequired": true},
			{"name": "Right to Erasure", "description": "Request deletion of your data", "contactRequired": true},
		},
		"multiJurisdiction": true,
		"jurisdictions": []map[string]interface{}{
			{
				"country":              "Germany",
				"governingLaw":         "German Civil Code (BGB)",
				"competentCourts":      "Courts of Berlin",
				"regulations":          []string{"GDPR", "BDSG", "TMG"},
				"hasSpecialProvisions": true,
				"specialProvisions":    []string{"German data protection requirements apply", "14-day cooling-off period for consumers"},
			},
			{
				"country":              "United States",
				"governingLaw":         "Laws of the State of Delaware",
				"competentCourts":      "Delaware Court of Chancery",
				"regulations":          []string{"CCPA", "CPRA"},
				"hasSpecialProvisions": false,
			},
		},
		"createdDate":  time.Now().AddDate(0, -1, -15),
		"lastModified": time.Now(),
		"author": map[string]interface{}{
			"name":       "Alice Johnson",
			"department": "Legal & Compliance",
		},
		"reviewer": map[string]interface{}{
			"name": "Bob Smith",
		},
		"status":             "approved",
		"showCertifications": true,
		"certifications": []map[string]interface{}{
			{"name": "ISO 27001", "standard": "Information Security", "issueDate": time.Now().AddDate(-2, 0, 0), "expiryDate": time.Now().AddDate(1, 0, 0), "isExpiringSoon": false},
			{"name": "SOC 2 Type II", "standard": "Service Organization Control", "issueDate": time.Now().AddDate(-1, 0, 0), "expiryDate": time.Now().AddDate(0, 11, 0), "isExpiringSoon": true},
		},
		"legalAddress": map[string]interface{}{
			"street":  "123 Business Blvd, Suite 456",
			"city":    "Berlin",
			"state":   "Berlin",
			"zip":     "10115",
			"country": "Germany",
		},
		"legalContact": map[string]interface{}{
			"email":          "legal@acme-corp.example.com",
			"phone":          "+49 30 12345678",
			"fax":            "+49 30 12345679",
			"officeHours":    "Monday-Friday, 9:00-17:00 CET",
			"emergencyPhone": "+49 30 12345680",
		},
		"hasTrademarks":       true,
		"trademarks":          []string{"ACME", "ACME Pro", "ACME Enterprise"},
		"includeHash":         true,
		"documentHash":        "a1b2c3d4e5f6g7h8i9j0",
		"showRevisionHistory": true,
		"revisionHistory": []map[string]interface{}{
			{"version": "0.1", "date": time.Now().AddDate(0, -2, 0), "author": "Alice J.", "summary": "Initial draft"},
			{"version": "0.5", "date": time.Now().AddDate(0, -1, -15), "author": "Alice J.", "summary": "Added legal sections"},
			{"version": "1.0", "date": time.Now(), "author": "Alice J.", "summary": "Final approval"},
		},
		"templateEngineVersion": "1.0.0",

		// Main document data
		"user": map[string]interface{}{
			"firstName": "Alice",
			"lastName":  "Johnson",
			"email":     "alice@example.com",
		},
		"items": []map[string]interface{}{
			{"name": "Widget Pro", "price": 99.99},
			{"name": "Gadget Plus", "price": 149.99},
			{"name": "Tool Elite", "price": 199.99},
		},
		"score":         85,
		"isWeekend":     false,
		"weekDays":      []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
		"isAdmin":       false,
		"isOwner":       true,
		"isLoggedIn":    true,
		"stringNumber":  "42",
		"stringPrice":   "19.99",
		"eventDate":     time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC),
		"optionalField": nil,
		"fruits":        []string{"apple", "banana", "orange"},
		"userTitle":     "",
		"defaultTitle":  "Guest",
		"description":   "This is the old version of the text",
		"name":          "Go-Stencil Template Engine",
		"features":      []string{"Fast", "Flexible", "Powerful"},
		"discount":      15.0,
		"age":           21,
		"hasID":         true,
		"isVIP":         false,
	}

	output, err := tmpl.Render(data)
	if err != nil {
		log.Fatalf("Failed to render nested fragments template: %v", err)
	}

	saveOutput(output, "output/nested_fragments_output.docx")

	fmt.Println("\nNested Fragments Demo:")
	fmt.Println("- fragment1 (Company Header) includes fragment2")
	fmt.Println("- fragment2 (Product Catalog) includes fragment3")
	fmt.Println("- fragment3 (Legal Footer) is the deepest level")
}

func validationExample(engine *stencil.Engine) {
	fmt.Println("Template Validation API allows checking templates for errors without rendering.")
	fmt.Println()

	// Example 1: Validate a template file
	fmt.Println("--- Validating invoice.docx ---")
	tmpl, err := engine.PrepareFile("invoice.docx")
	if err != nil {
		fmt.Printf("Could not prepare template: %v\n", err)
	} else {
		defer tmpl.Close()

		// Basic validation
		result := tmpl.Validate()

		fmt.Printf("Valid: %v\n", result.Valid)
		fmt.Printf("Variables found: %v\n", result.Variables)
		fmt.Printf("Functions called: %v\n", result.Functions)
		fmt.Printf("Control structures: %d\n", len(result.ControlStructs))

		if len(result.Errors) > 0 {
			fmt.Println("Errors:")
			for _, e := range result.Errors {
				fmt.Printf("  - %s\n", e.Error())
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, w := range result.Warnings {
				fmt.Printf("  - %s\n", w.Error())
			}
		}
	}
	fmt.Println()

	// Example 2: Validate with fragment checking
	fmt.Println("--- Validating report.docx with fragment checking ---")
	reportTmpl, err := engine.PrepareFile("report.docx")
	if err != nil {
		fmt.Printf("Could not prepare template: %v\n", err)
	} else {
		defer reportTmpl.Close()

		// First validate without adding fragments - should show missing fragment errors
		result := reportTmpl.Validate()
		fmt.Printf("Before adding fragments - Valid: %v\n", result.Valid)
		if len(result.FragmentRefs) > 0 {
			fmt.Printf("Fragments referenced: %v\n", result.FragmentRefs)
		}
		if len(result.Errors) > 0 {
			fmt.Println("Missing fragment errors:")
			for _, e := range result.Errors {
				if e.Type == stencil.ValidationErrorMissingFragment {
					fmt.Printf("  - %s\n", e.Error())
				}
			}
		}

		// Add required fragments
		reportTmpl.AddFragment("disclaimer", "This is confidential.")
		reportTmpl.AddFragment("copyright", "© 2024 ACME Corp")
		reportTmpl.AddFragment("header", "ACME Report Header")

		// Validate again - should pass now
		result = reportTmpl.Validate()
		fmt.Printf("After adding fragments - Valid: %v\n", result.Valid)
	}
	fmt.Println()

	// Example 3: Validate expressions directly
	fmt.Println("--- Direct expression validation ---")
	expressions := []string{
		"name",
		"customer.address.city",
		"price * quantity",
		"uppercase(name)",
		"unclosed(",        // Invalid - unclosed parenthesis
		"'unclosed string", // Invalid - unclosed string
	}

	for _, expr := range expressions {
		err := stencil.ValidateExpression(expr)
		if err != nil {
			fmt.Printf("  %-25s -> INVALID: %v\n", expr, err)
		} else {
			fmt.Printf("  %-25s -> valid\n", expr)
		}
	}
	fmt.Println()

	// Example 4: Validate for loop syntax
	fmt.Println("--- For loop syntax validation ---")
	forLoops := []string{
		"item in items",
		"i, item in items",
		"x in range(1, 10)",
		"invalid syntax",    // Invalid - missing 'in'
		"x y z in items",    // Invalid - too many variables
	}

	for _, loop := range forLoops {
		err := stencil.ValidateForSyntax(loop)
		if err != nil {
			fmt.Printf("  %-25s -> INVALID: %v\n", loop, err)
		} else {
			fmt.Printf("  %-25s -> valid\n", loop)
		}
	}
	fmt.Println()

	// Example 5: Validation with custom options
	fmt.Println("--- Validation with custom options ---")
	if tmpl != nil {
		// Disable function checking - useful when using custom functions
		opts := stencil.ValidationOptions{
			CheckFunctions: false, // Don't check if functions exist
			CheckFragments: true,
			StrictMode:     false,
		}

		result := tmpl.ValidateWithOptions(opts)
		fmt.Printf("With function checking disabled - Valid: %v\n", result.Valid)

		// Enable strict mode - warnings become errors
		opts.StrictMode = true
		result = tmpl.ValidateWithOptions(opts)
		fmt.Printf("With strict mode enabled - Valid: %v\n", result.Valid)
	}
	fmt.Println()

	// Example 6: Extract template tokens for analysis
	fmt.Println("--- Token extraction ---")
	sampleText := "Hello {{name}}, your order {{order.id}} totals {{format('%.2f', total)}}."
	tokens := stencil.ExtractTemplateTokens(sampleText)
	fmt.Printf("Sample: %s\n", sampleText)
	fmt.Printf("Tokens found: %v\n", tokens)
	fmt.Println()

	// Example 7: Using ValidationResult methods
	fmt.Println("--- ValidationResult methods ---")
	if tmpl != nil {
		result := tmpl.Validate()

		// Check using helper methods
		fmt.Printf("HasErrors(): %v\n", result.HasErrors())
		fmt.Printf("HasWarnings(): %v\n", result.HasWarnings())

		// Get combined error (useful for error handling)
		if err := result.Error(); err != nil {
			fmt.Printf("Error(): %v\n", err)
		} else {
			fmt.Println("Error(): nil (template is valid)")
		}

		// Get human-readable summary
		fmt.Println("\nFull validation summary:")
		fmt.Println(result.String())
	}
}
