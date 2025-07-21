package stencil

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Common benchmark data structures
var (
	// Simple data for basic benchmarks
	benchmarkSimpleData = map[string]interface{}{
		"name":    "John Doe",
		"company": "ACME Corp",
		"date":    "2024-01-15",
		"amount":  1234.56,
	}

	// Complex nested data for comprehensive benchmarks
	benchmarkComplexData = map[string]interface{}{
		"company": map[string]interface{}{
			"name":    "Tech Innovations Inc.",
			"address": "123 Silicon Valley Blvd",
			"city":    "San Francisco",
			"state":   "CA",
			"zip":     "94105",
		},
		"invoice": map[string]interface{}{
			"number": "INV-2024-001",
			"date":   "2024-01-15",
			"due":    "2024-02-14",
		},
		"items": []map[string]interface{}{
			{
				"description": "Software License - Pro",
				"quantity":    5,
				"price":       299.99,
				"total":       1499.95,
			},
			{
				"description": "Support Package - Gold",
				"quantity":    5,
				"price":       99.99,
				"total":       499.95,
			},
			{
				"description": "Training Session",
				"quantity":    2,
				"price":       500.00,
				"total":       1000.00,
			},
			{
				"description": "Custom Development",
				"quantity":    40,
				"price":       150.00,
				"total":       6000.00,
			},
			{
				"description": "Deployment Assistance",
				"quantity":    8,
				"price":       200.00,
				"total":       1600.00,
			},
		},
		"subtotal": 10599.90,
		"tax":      847.99,
		"total":    11447.89,
		"notes":    "Payment due within 30 days. Late payments subject to 1.5% monthly interest.",
	}

	// Large dataset for stress testing
	benchmarkLargeData = generateLargeDataset()

	// Template content for benchmarks
	simpleTemplate = "Hello {{name}}, welcome to {{company}}!"
	
	complexTemplate = `{{company.name}}
Invoice #{{invoice.number}} - {{date(invoice.date, "MMMM d, yyyy")}}

{{for item in items}}
{{item.description}} - Qty: {{item.quantity}} @ ${{format("%.2f", item.price)}} = ${{format("%.2f", item.total)}}
{{end}}

Total: ${{format("%.2f", total)}}

{{if notes}}
{{notes}}
{{end}}`

	// Expression for stress testing
	complexExpression = "((price * quantity) * (1 + taxRate)) / exchangeRate"
)

// createBenchDocx creates a test DOCX for benchmarking
func createBenchDocx(b *testing.B, content string) *bytes.Buffer {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	
	// Add document.xml with the content
	f, err := w.Create("word/document.xml")
	if err != nil {
		b.Fatal(err)
	}
	
	// Wrap content in minimal document structure
	docXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>%s</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`, content)
	
	if _, err := f.Write([]byte(docXML)); err != nil {
		b.Fatal(err)
	}
	
	// Add required relationships
	rels, err := w.Create("word/_rels/document.xml.rels")
	if err != nil {
		b.Fatal(err)
	}
	
	relsXML := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
	
	if _, err := rels.Write([]byte(relsXML)); err != nil {
		b.Fatal(err)
	}
	
	// Add content types
	ct, err := w.Create("[Content_Types].xml")
	if err != nil {
		b.Fatal(err)
	}
	
	ctXML := `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
	
	if _, err := ct.Write([]byte(ctXML)); err != nil {
		b.Fatal(err)
	}
	
	if err := w.Close(); err != nil {
		b.Fatal(err)
	}
	
	return buf
}

// generateLargeDataset creates a large dataset for stress testing
func generateLargeDataset() map[string]interface{} {
	items := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		items[i] = map[string]interface{}{
			"id":          i + 1,
			"name":        "Product " + string(rune('A'+i%26)),
			"description": "This is a detailed description for product number " + string(rune('A'+i%26)),
			"price":       float64(10+i%90) + 0.99,
			"quantity":    1 + i%10,
			"inStock":     i%3 != 0,
			"category":    []string{"Electronics", "Books", "Clothing", "Food", "Toys"}[i%5],
		}
	}

	return map[string]interface{}{
		"items":      items,
		"totalItems": len(items),
		"date":       "2024-01-15",
		"customer": map[string]interface{}{
			"name":    "Big Customer Corp",
			"id":      "CUST-12345",
			"address": "456 Enterprise Way",
		},
	}
}

// Benchmark template preparation with simple template
func BenchmarkPrepareTemplate_Simple(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := createBenchDocx(b, simpleTemplate)
		_, err := Prepare(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark template preparation with complex template
func BenchmarkPrepareTemplate_Complex(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := createBenchDocx(b, complexTemplate)
		_, err := Prepare(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark simple variable substitution
func BenchmarkRender_SimpleSubstitution(b *testing.B) {
	reader := createBenchDocx(b, simpleTemplate)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(benchmarkSimpleData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark complex template rendering
func BenchmarkRender_Complex(b *testing.B) {
	reader := createBenchDocx(b, complexTemplate)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(benchmarkComplexData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark rendering with large dataset
func BenchmarkRender_LargeDataset(b *testing.B) {
	// Create template with loop over 100 items
	largeTemplate := `Customer: {{customer.name}}
Date: {{date}}

Items:
{{for item in items}}
{{item.name}} - {{item.category}} - ${{format("%.2f", item.price)}} x {{item.quantity}}
{{end}}

Total items: {{totalItems}}`

	reader := createBenchDocx(b, largeTemplate)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(benchmarkLargeData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark nested template rendering to test expression evaluation performance
func BenchmarkRender_NestedData(b *testing.B) {
	templateText := `Customer: {{customer.name}}
Address: {{customer.address.street}}, {{customer.address.city}} {{customer.address.zip}}`
	
	reader := createBenchDocx(b, templateText)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}
	
	data := map[string]interface{}{
		"customer": map[string]interface{}{
			"name": "John Doe",
			"address": map[string]interface{}{
				"street": "123 Main St",
				"city":   "New York",
				"zip":    "10001",
			},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark mathematical expression rendering
func BenchmarkRender_MathExpressions(b *testing.B) {
	templateText := `Subtotal: ${{price * quantity}}
Tax: ${{(price * quantity) * taxRate}}
Total: ${{(price * quantity) * (1 + taxRate)}}`
	
	reader := createBenchDocx(b, templateText)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}
	
	data := map[string]interface{}{
		"price":    99.99,
		"quantity": 5,
		"taxRate":  0.08,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark function call rendering
func BenchmarkRender_FunctionCalls(b *testing.B) {
	templateText := `Price: ${{format("%.2f", price)}}
Total: ${{format("%.2f", price * 1.08)}}
Date: {{date(orderDate, "MMMM d, yyyy")}}`
	
	reader := createBenchDocx(b, templateText)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}
	
	data := map[string]interface{}{
		"price":     99.99,
		"orderDate": "2024-01-15",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark conditional expression evaluation
func BenchmarkConditional_Simple(b *testing.B) {
	templateText := `{{if active}}Active{{else}}Inactive{{end}}`
	reader := createBenchDocx(b, templateText)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}

	data := map[string]interface{}{
		"active": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark loop performance
func BenchmarkLoop_Small(b *testing.B) {
	templateText := `{{for i in items}}Item {{i}} {{end}}`
	reader := createBenchDocx(b, templateText)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}

	data := map[string]interface{}{
		"items": []int{1, 2, 3, 4, 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark string function performance
func BenchmarkFunction_StringOperations(b *testing.B) {
	templateText := `{{uppercase(lowercase(join(items, ", ")))}}`
	reader := createBenchDocx(b, templateText)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}

	data := map[string]interface{}{
		"items": []string{"apple", "banana", "cherry"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tmpl.Render(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark caching performance
func BenchmarkCache_Hit(b *testing.B) {
	// Prepare a template and cache it
	reader := createBenchDocx(b, simpleTemplate)
	cache := NewTemplateCache()
	
	// Pre-populate cache
	_, err := cache.Prepare(reader, "benchmark-key")
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpl, exists := cache.Get("benchmark-key")
		if !exists || tmpl == nil {
			b.Fatal("Cache miss when hit expected")
		}
	}
}

// Benchmark cache miss and prepare
func BenchmarkCache_Miss(b *testing.B) {
	cache := NewTemplateCache()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create unique key to force cache miss
		key := strings.Repeat("x", i%100)
		reader := createBenchDocx(b, simpleTemplate)
		_, err := cache.Prepare(reader, key)
		if err != nil {
			b.Fatal(err)
		}
		// Clear to avoid filling up cache
		cache.Clear()
	}
}