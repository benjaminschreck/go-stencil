package stencil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegrationRealWorldTemplates tests rendering of real-world templates
func TestIntegrationRealWorldTemplates(t *testing.T) {
	testCases := []struct {
		name         string
		templateFile string
		data         TemplateData
		validate     func(t *testing.T, output []byte)
	}{
		{
			name:         "invoice template",
			templateFile: "../../examples/advanced/invoice.docx",
			data: TemplateData{
				"invoiceNumber": "INV-2024-001",
				"invoiceDate":   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
				"dueDate":       time.Date(2024, 2, 14, 0, 0, 0, 0, time.UTC),
				"company": map[string]interface{}{
					"name":    "Acme Corporation",
					"address": "123 Business St",
					"city":    "New York",
					"state":   "NY",
					"zip":     "10001",
				},
				"customer": map[string]interface{}{
					"name":    "John Doe",
					"address": "456 Main St",
					"city":    "Los Angeles",
					"state":   "CA",
					"zip":     "90001",
				},
				"items": []map[string]interface{}{
					{"description": "Consulting Services", "quantity": 10, "unitPrice": 150.00},
					{"description": "Software License", "quantity": 1, "unitPrice": 2500.00},
					{"description": "Support Package", "quantity": 12, "unitPrice": 100.00},
				},
				"taxRate": 0.08,
				"notes":   "Thank you for your business!",
			},
			validate: func(t *testing.T, output []byte) {
				// Check that output is a valid DOCX
				if len(output) < 100 {
					t.Error("Output too small to be a valid DOCX")
				}
				// Check for DOCX signature
				if !bytes.HasPrefix(output, []byte("PK")) {
					t.Error("Output doesn't have DOCX (ZIP) signature")
				}
			},
		},
		{
			name:         "report template with tables",
			templateFile: "../../examples/advanced/report.docx",
			data: TemplateData{
				"title": "Sales Report",
				"quarter": "Q4",
				"year": "2024",
				"author": "Analytics Team",
				"highlights": []string{
					"Record breaking revenue in Q4",
					"25% growth in Asia Pacific region",
					"New product launch exceeded targets",
					"Customer satisfaction at all-time high",
				},
				"reportTitle": "Quarterly Sales Report",
				"reportDate":  time.Now(),
				"summary":     "This quarter showed significant growth across all regions.",
				"regions": []map[string]interface{}{
					{"name": "North America", "revenue": 1500000, "growth": 0.15},
					{"name": "Europe", "revenue": 1200000, "growth": 0.12},
					{"name": "Asia Pacific", "revenue": 900000, "growth": 0.25},
					{"name": "Latin America", "revenue": 400000, "growth": 0.08},
				},
				"topProducts": []map[string]interface{}{
					{"name": "Product A", "units": 5000, "revenue": 500000},
					{"name": "Product B", "units": 3500, "revenue": 420000},
					{"name": "Product C", "units": 2000, "revenue": 300000},
				},
				"conclusion": "Overall performance exceeded expectations.",
			},
			validate: func(t *testing.T, output []byte) {
				if len(output) < 100 {
					t.Error("Output too small to be a valid DOCX")
				}
			},
		},
		{
			name:         "basic template with conditionals",
			templateFile: "../../examples/advanced/basic.docx",
			data: TemplateData{
				"customerName": "Jane Smith",
				"isPremium":    true,
				"accountType":  "Business",
				"balance":      15000.50,
				"transactions": []map[string]interface{}{
					{"date": "2024-01-01", "description": "Deposit", "amount": 5000.00},
					{"date": "2024-01-05", "description": "Payment", "amount": -1200.00},
					{"date": "2024-01-10", "description": "Transfer", "amount": -500.00},
				},
			},
			validate: func(t *testing.T, output []byte) {
				if len(output) < 100 {
					t.Error("Output too small to be a valid DOCX")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip if template file doesn't exist
			if _, err := os.Stat(tc.templateFile); os.IsNotExist(err) {
				t.Skipf("Template file %s not found", tc.templateFile)
				return
			}

			// Prepare template
			tmpl, err := PrepareFile(tc.templateFile)
			if err != nil {
				t.Fatalf("Failed to prepare template: %v", err)
			}
			defer tmpl.Close()

			// Add fragments if the template uses them
			if strings.Contains(tc.templateFile, "report.docx") {
				// Add fragments that report.docx expects
				tmpl.AddFragment("header", "Quarterly Report Header")
				tmpl.AddFragment("disclaimer", "This report is for informational purposes only.")
				tmpl.AddFragment("copyright", "Copyright © 2024 - All rights reserved.")
				tmpl.AddFragment("footer", "Report Footer - Confidential")
			}

			// Render
			output, err := tmpl.Render(tc.data)
			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			// Read output
			var buf bytes.Buffer
			_, err = io.Copy(&buf, output)
			if err != nil {
				t.Fatalf("Failed to read output: %v", err)
			}

			// Validate
			tc.validate(t, buf.Bytes())
		})
	}
}

// TestIntegrationComplexScenarios tests complex template scenarios
func TestIntegrationComplexScenarios(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		data     TemplateData
		expected string
	}{
		{
			name: "nested loops with conditionals",
			template: `Departments:
{{for dept in departments}}
Department: {{dept.name}}
{{if dept.employees}}Employees:
{{for emp in dept.employees}}  - {{emp.name}} ({{emp.role}}){{if emp.isManager}} - Manager{{end}}
{{end}}{{else}}  No employees in this department.
{{end}}
{{end}}`,
			data: TemplateData{
				"departments": []map[string]interface{}{
					{
						"name": "Engineering",
						"employees": []map[string]interface{}{
							{"name": "Alice", "role": "Senior Developer", "isManager": false},
							{"name": "Bob", "role": "Team Lead", "isManager": true},
							{"name": "Charlie", "role": "Junior Developer", "isManager": false},
						},
					},
					{
						"name":      "Marketing",
						"employees": nil,
					},
					{
						"name": "Sales",
						"employees": []map[string]interface{}{
							{"name": "David", "role": "Sales Manager", "isManager": true},
							{"name": "Eve", "role": "Sales Rep", "isManager": false},
						},
					},
				},
			},
			expected: `Departments:

Department: Engineering
Employees:
  - Alice (Senior Developer)
  - Bob (Team Lead) - Manager
  - Charlie (Junior Developer)


Department: Marketing
  No employees in this department.


Department: Sales
Employees:
  - David (Sales Manager) - Manager
  - Eve (Sales Rep)
`,
		},
		{
			name: "complex expressions and functions",
			template: `
Order Summary:
Items: {{length(items)}}
Subtotal: ${{format("%.2f", sum(map("price", items)))}}
Tax ({{percent(taxRate)}}): ${{format("%.2f", sum(map("price", items)) * taxRate)}}
Total: ${{format("%.2f", sum(map("price", items)) * (1 + taxRate))}}

{{if sum(map("price", items)) > 100}}
Eligible for free shipping!
{{else}}
Shipping: ${{format("%.2f", shippingCost)}}
Final Total: ${{format("%.2f", sum(map("price", items)) * (1 + taxRate) + shippingCost)}}
{{end}}`,
			data: TemplateData{
				"items": []map[string]interface{}{
					{"name": "Widget", "price": 25.99},
					{"name": "Gadget", "price": 45.50},
					{"name": "Doohickey", "price": 15.00},
				},
				"taxRate":      0.0875,
				"shippingCost": 9.99,
			},
			expected: `Order Summary:
Items: 3
Subtotal: $86.49
Tax (8.75%): $7.57
Total: $94.06


Shipping: $9.99
Final Total: $104.05
`,
		},
		{
			name: "multiple control structures with functions",
			template: `{{for i, category in categories}}{{if i > 0}}
{{end}}Category: {{uppercase(category.name)}}
{{for product in category.products}}{{if product.inStock}}  - {{product.name}}: {{currency(product.price)}}{{else}}  - {{product.name}}: OUT OF STOCK{{end}}
{{end}}{{end}}`,
			data: TemplateData{
				"categories": []map[string]interface{}{
					{
						"name": "Electronics",
						"products": []map[string]interface{}{
							{"name": "Laptop", "price": 999.99, "inStock": true},
							{"name": "Monitor", "price": 299.99, "inStock": false},
						},
					},
					{
						"name": "Books",
						"products": []map[string]interface{}{
							{"name": "Go Programming", "price": 49.99, "inStock": true},
							{"name": "Clean Code", "price": 39.99, "inStock": true},
						},
					},
				},
			},
			expected: `Category: ELECTRONICS
  - Laptop: $999.99
  - Monitor: OUT OF STOCK

Category: BOOKS
  - Go Programming: $49.99
  - Clean Code: $39.99`,
		},
		{
			name: "fragment inclusion with conditionals",
			template: `{{if showHeader}}{{include "header"}}
{{end}}Main content here.
{{if showFooter}}{{include "footer"}}{{end}}`,
			data: TemplateData{
				"showHeader": true,
				"showFooter": true,
			},
			expected: `Company Header
Main content here.
Copyright 2024`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test template
			tmpl, err := Parse("test.docx", tc.template)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}
			defer tmpl.Close()

			// Add fragments if needed
			if strings.Contains(tc.template, "include") {
				tmpl.AddFragment("header", "Company Header")
				tmpl.AddFragment("footer", "Copyright 2024")
			}

			// Render
			result, err := tmpl.Render(tc.data)
			if err != nil {
				t.Fatalf("Failed to render template: %v", err)
			}

			// Normalize whitespace for comparison
			got := strings.TrimSpace(result)
			want := strings.TrimSpace(tc.expected)

			if got != want {
				t.Errorf("Rendered output mismatch\nGot:\n%s\nWant:\n%s", got, want)
			}
		})
	}
}

// TestIntegrationLargeDocuments tests handling of large documents
func TestIntegrationLargeDocuments(t *testing.T) {
	// Create a large dataset
	largeData := TemplateData{
		"title": "Large Document Test",
		"items": make([]map[string]interface{}, 1000),
	}

	// Generate 1000 items
	for i := 0; i < 1000; i++ {
		largeData["items"].([]map[string]interface{})[i] = map[string]interface{}{
			"id":          i + 1,
			"name":        fmt.Sprintf("Item %d", i+1),
			"description": fmt.Sprintf("Description for item %d with some additional text to make it longer", i+1),
			"price":       float64(i+1) * 10.99,
			"quantity":    (i % 10) + 1,
		}
	}

	template := `{{title}}

Items Report:
{{for item in items}}
ID: {{item.id}}
Name: {{item.name}}
Description: {{item.description}}
Price: ${{format("%.2f", item.price)}}
Quantity: {{item.quantity}}
Total: ${{format("%.2f", item.price * item.quantity)}}
---
{{end}}

Total items: {{length(items)}}
Total value: ${{format("%.2f", sum(map("price", items)))}}
`

	// Create template
	tmpl, err := Parse("large.docx", template)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	defer tmpl.Close()

	// Measure rendering time
	start := time.Now()
	result, err := tmpl.Render(largeData)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to render large template: %v", err)
	}

	// Verify result contains expected content
	if !strings.Contains(result, "Total items: 1000") {
		t.Error("Result doesn't contain expected total items count")
	}

	// Check performance (should complete in reasonable time)
	if duration > 5*time.Second {
		t.Errorf("Rendering took too long: %v", duration)
	}

	t.Logf("Large document rendered in %v", duration)
}

// TestIntegrationEdgeCases tests various edge cases
func TestIntegrationEdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		data     TemplateData
		wantErr  bool
		expected string
	}{
		{
			name:     "empty data",
			template: `Name: {{name}}, Age: {{age}}`,
			data:     TemplateData{},
			expected: `Name: , Age: `,
		},
		{
			name:     "nil values",
			template: `{{if user}}User: {{user.name}}{{else}}No user{{end}}`,
			data:     TemplateData{"user": nil},
			expected: `No user`,
		},
		{
			name:     "empty arrays",
			template: `Items: {{length(items)}}{{for item in items}} - {{item}}{{end}}`,
			data:     TemplateData{"items": []interface{}{}},
			expected: `Items: 0`,
		},
		{
			name:     "deeply nested access",
			template: `Value: {{a.b.c.d.e.f}}`,
			data: TemplateData{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": map[string]interface{}{
							"d": map[string]interface{}{
								"e": map[string]interface{}{
									"f": "found!",
								},
							},
						},
					},
				},
			},
			expected: `Value: found!`,
		},
		{
			name:     "missing nested values",
			template: `Value: {{coalesce(a.b.c.d, "default")}}`,
			data:     TemplateData{"a": map[string]interface{}{"b": nil}},
			expected: `Value: default`,
		},
		{
			name:     "special characters in strings",
			template: `{{text}}`,
			data:     TemplateData{"text": "Special chars: < > & \" ' \n\t"},
			expected: `Special chars: < > & " ' 
	`,
		},
		{
			name:     "unicode support",
			template: `Hello {{name}} 👋 Length: {{length(name)}}`,
			data:     TemplateData{"name": "世界"},
			expected: `Hello 世界 👋 Length: 2`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl, err := Parse("test.docx", tc.template)
			if err != nil {
				t.Fatalf("Failed to parse template: %v", err)
			}
			defer tmpl.Close()

			result, err := tmpl.Render(tc.data)
			if tc.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Result mismatch\nGot:  %q\nWant: %q", result, tc.expected)
			}
		})
	}
}

// TestIntegrationConcurrentRendering tests thread safety
func TestIntegrationConcurrentRendering(t *testing.T) {
	template := `User: {{user}}, ID: {{id}}, Time: {{time}}`
	
	// Create template
	tmpl, err := Parse("concurrent.docx", template)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
	defer tmpl.Close()

	// Run concurrent renders
	const numGoroutines = 100
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			data := TemplateData{
				"user": fmt.Sprintf("User%d", id),
				"id":   id,
				"time": time.Now().Format("15:04:05.000"),
			}

			result, err := tmpl.Render(data)
			if err != nil {
				errors <- err
			} else {
				expected := fmt.Sprintf("User: User%d, ID: %d, Time:", id, id)
				if !strings.HasPrefix(result, expected) {
					errors <- fmt.Errorf("unexpected result for goroutine %d: %s", id, result)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent rendering error: %v", err)
	}
}

// TestIntegrationMemoryLeaks tests for memory leaks
func TestIntegrationMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	template := `{{for item in items}}Item: {{item.name}} - {{item.value}}
{{end}}`

	// Run multiple iterations
	for i := 0; i < 100; i++ {
		tmpl, err := Parse("memory.docx", template)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		data := TemplateData{
			"items": []map[string]interface{}{
				{"name": "Item1", "value": "Value1"},
				{"name": "Item2", "value": "Value2"},
			},
		}

		_, err = tmpl.Render(data)
		if err != nil {
			t.Fatalf("Failed to render: %v", err)
		}

		// Important: Close to release resources
		tmpl.Close()
	}

	// If we get here without running out of memory, test passes
}

// TestIntegrationWithTestDataFiles tests with actual DOCX files from testdata
func TestIntegrationWithTestDataFiles(t *testing.T) {
	testDataDir := "../../testdata/docx"
	
	testCases := []struct {
		filename string
		data     TemplateData
	}{
		{
			filename: "test-control-conditionals.docx",
			data: TemplateData{
				"showSection1": true,
				"showSection2": false,
				"userType":     "premium",
				"score":        85,
			},
		},
		{
			filename: "test-control-loop.docx",
			data: TemplateData{
				"items": []map[string]interface{}{
					{"name": "First", "value": 100},
					{"name": "Second", "value": 200},
					{"name": "Third", "value": 300},
				},
			},
		},
		{
			filename: "test-embedded-html.docx",
			data: TemplateData{
				"htmlContent": "<b>Bold</b> and <i>italic</i> text",
				"title":       "HTML Test",
			},
		},
		{
			filename: "test-table-columns.docx",
			data: TemplateData{
				"showColumn1": true,
				"showColumn2": false,
				"showColumn3": true,
				"rows": []map[string]interface{}{
					{"col1": "A1", "col2": "B1", "col3": "C1"},
					{"col1": "A2", "col2": "B2", "col3": "C2"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			filePath := filepath.Join(testDataDir, tc.filename)
			
			// Skip if file doesn't exist
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Skipf("Test file %s not found", tc.filename)
				return
			}

			// Prepare template
			tmpl, err := PrepareFile(filePath)
			if err != nil {
				t.Fatalf("Failed to prepare template from %s: %v", tc.filename, err)
			}
			defer tmpl.Close()

			// Render
			output, err := tmpl.Render(tc.data)
			if err != nil {
				t.Fatalf("Failed to render %s: %v", tc.filename, err)
			}

			// Verify output is valid
			var buf bytes.Buffer
			n, err := io.Copy(&buf, output)
			if err != nil {
				t.Fatalf("Failed to read output: %v", err)
			}

			if n < 100 {
				t.Errorf("Output too small (%d bytes) for %s", n, tc.filename)
			}

			// Check for DOCX signature
			if !bytes.HasPrefix(buf.Bytes(), []byte("PK")) {
				t.Errorf("Output from %s doesn't have DOCX signature", tc.filename)
			}
		})
	}
}