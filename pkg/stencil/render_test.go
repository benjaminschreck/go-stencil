package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)


func TestRenderDocument(t *testing.T) {
	tests := []struct {
		name     string
		document *Document
		data     TemplateData
		wantText string
		wantErr  bool
	}{
		{
			name: "render simple variable",
			document: &Document{
				Body: createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{Text: &Text{Content: "Hello {{name}}!"}},
						},
					},
				}),
			},
			data: TemplateData{
				"name": "World",
			},
			wantText: "Hello World!",
			wantErr:  false,
		},
		{
			name: "render multiple variables",
			document: &Document{
				Body: createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{Text: &Text{Content: "{{greeting}} {{name}}, you have {{count}} messages"}},
						},
					},
				}),
			},
			data: TemplateData{
				"greeting": "Hello",
				"name":     "John",
				"count":    5,
			},
			wantText: "Hello John, you have 5 messages",
			wantErr:  false,
		},
		{
			name: "render with missing variable",
			document: &Document{
				Body: createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{Text: &Text{Content: "Hello {{name}}!"}},
						},
					},
				}),
			},
			data:     TemplateData{},
			wantText: "Hello !",
			wantErr:  false,
		},
		{
			name: "render multiple runs",
			document: &Document{
				Body: createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{Text: &Text{Content: "Hello "}},
							{Text: &Text{Content: "{{name}}"}},
							{Text: &Text{Content: "!"}},
						},
					},
				}),
			},
			data: TemplateData{
				"name": "World",
			},
			wantText: "Hello World!",
			wantErr:  false,
		},
		{
			name: "render with no templates",
			document: &Document{
				Body: createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{Text: &Text{Content: "Plain text without templates"}},
						},
					},
				}),
			},
			data:     TemplateData{},
			wantText: "Plain text without templates",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderDocument(tt.document, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderDocument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Extract text from the rendered document
				got := extractText(t, result)
				if got != tt.wantText {
					t.Errorf("RenderDocument() text = %v, want %v", got, tt.wantText)
				}
			}
		})
	}
}

func TestRenderRun(t *testing.T) {
	tests := []struct {
		name    string
		run     *Run
		data    TemplateData
		want    *Run
		wantErr bool
	}{
		{
			name: "render run with variable",
			run: &Run{
				Text: &Text{Content: "Hello {{name}}!"},
			},
			data: TemplateData{
				"name": "World",
			},
			want: &Run{
				Text: &Text{Content: "Hello World!"},
			},
			wantErr: false,
		},
		{
			name: "render run without template",
			run: &Run{
				Text: &Text{Content: "Plain text"},
			},
			data: TemplateData{},
			want: &Run{
				Text: &Text{Content: "Plain text"},
			},
			wantErr: false,
		},
		{
			name: "render run with nil text",
			run:  &Run{},
			data: TemplateData{},
			want: &Run{},
			wantErr: false,
		},
		{
			name: "render run with space preserve",
			run: &Run{
				Text: &Text{
					Content: "  {{name}}  ",
					Space:   "preserve",
				},
			},
			data: TemplateData{
				"name": "Test",
			},
			want: &Run{
				Text: &Text{
					Content: "  Test  ",
					Space:   "preserve",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderRun(tt.run, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if got.GetText() != tt.want.GetText() {
					t.Errorf("RenderRun() text = %v, want %v", got.GetText(), tt.want.GetText())
				}
				if got.Text != nil && tt.want.Text != nil {
					if got.Text.Space != tt.want.Text.Space {
						t.Errorf("RenderRun() space = %v, want %v", got.Text.Space, tt.want.Text.Space)
					}
				}
			}
		})
	}
}

// Helper function to extract text from a rendered document
func extractText(t *testing.T, doc *Document) string {
	var texts []string
	if doc.Body != nil {
		for _, elem := range doc.Body.Elements {
			if para, ok := elem.(*Paragraph); ok {
				texts = append(texts, para.GetText())
			}
		}
	}
	return strings.Join(texts, "\n")
}

func TestRenderExpressions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		data        TemplateData
		want        string
		description string
	}{
		{
			name:  "simple arithmetic expression",
			input: "Result: {{2 + 3}}",
			data:  TemplateData{},
			want:  "Result: 5",
			description: "Basic addition expression in template",
		},
		{
			name:  "variable arithmetic",
			input: "Total: {{price * quantity}}",
			data: TemplateData{
				"price":    10.5,
				"quantity": 3,
			},
			want: "Total: 31.5",
			description: "Arithmetic with variables",
		},
		{
			name:  "complex expression with parentheses",
			input: "Tax: {{(basePrice + fee) * taxRate}}",
			data: TemplateData{
				"basePrice": 100.0,
				"fee":       5.0,
				"taxRate":   0.08,
			},
			want: "Tax: 8.4",
			description: "Complex expression with precedence",
		},
		{
			name:  "string concatenation",
			input: "Greeting: {{\"Hello \" + name + \"!\"}}",
			data: TemplateData{
				"name": "World",
			},
			want: "Greeting: Hello World!",
			description: "String concatenation in expression",
		},
		{
			name:  "comparison result",
			input: "Is adult: {{age >= 18}}",
			data: TemplateData{
				"age": 25,
			},
			want: "Is adult: true",
			description: "Boolean comparison result",
		},
		{
			name:  "nested field access in expression",
			input: "City: {{customer.address.city}}",
			data: TemplateData{
				"customer": map[string]interface{}{
					"address": map[string]interface{}{
						"city": "New York",
					},
				},
			},
			want: "City: New York",
			description: "Nested field access in expressions",
		},
		{
			name:  "array access with arithmetic",
			input: "Price sum: {{items[0].price + items[1].price}}",
			data: TemplateData{
				"items": []interface{}{
					map[string]interface{}{"price": 10.0},
					map[string]interface{}{"price": 15.0},
				},
			},
			want: "Price sum: 25",
			description: "Array access combined with arithmetic",
		},
		{
			name:  "logical expression",
			input: "Valid: {{age >= 18 & hasLicense}}",
			data: TemplateData{
				"age":        25,
				"hasLicense": true,
			},
			want: "Valid: true",
			description: "Logical AND operation",
		},
		{
			name:  "fallback to simple variable",
			input: "Simple: {{name}}",
			data: TemplateData{
				"name": "John",
			},
			want: "Simple: John",
			description: "Fallback to simple variable evaluation",
		},
		{
			name:  "mixed expressions and variables",
			input: "{{name}}: {{basePrice + tax}} ({{taxRate * 100}}% tax)",
			data: TemplateData{
				"name":      "Product",
				"basePrice": 100.0,
				"tax":       8.0,
				"taxRate":   0.08,
			},
			want: "Product: 108 (8% tax)",
			description: "Multiple expressions in one template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple document with one paragraph and one run
			doc := &Document{
				Body: createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
								{
									Text: &Text{
										Content: tt.input,
									},
								},
						},
					},
				}),
			}

			result, err := RenderDocument(doc, tt.data)
			if err != nil {
				t.Fatalf("RenderDocument() error = %v", err)
			}

			// Extract the first paragraph from Elements
			if len(result.Body.Elements) == 0 {
				t.Fatal("No elements in result body")
			}
			para, ok := result.Body.Elements[0].(*Paragraph)
			if !ok {
				t.Fatal("First element is not a Paragraph")
			}
			if len(para.Runs) == 0 || para.Runs[0].Text == nil {
				t.Fatal("No runs or text in paragraph")
			}
			got := para.Runs[0].Text.Content
			if got != tt.want {
				t.Errorf("RenderDocument() = %q, want %q\nDescription: %s", got, tt.want, tt.description)
			}
		})
	}
}

// Integration test for full render pipeline
func TestRenderIntegration(t *testing.T) {
	// Create a test DOCX with variables
	buf := createTestDocx(t, "Hello {{name}}, your balance is {{balance}}!")
	reader := bytes.NewReader(buf.Bytes())
	
	// Prepare the template
	pt, err := Prepare(reader)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	defer pt.Close()
	
	// Render with data
	data := TemplateData{
		"name":    "John Doe",
		"balance": 1234.56,
	}
	
	output, err := pt.Render(data)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	
	// Read the output as a DOCX
	outputBuf := new(bytes.Buffer)
	outputBuf.ReadFrom(output)
	
	zipReader, err := zip.NewReader(bytes.NewReader(outputBuf.Bytes()), int64(outputBuf.Len()))
	if err != nil {
		t.Fatalf("Failed to read output as zip: %v", err)
	}
	
	// Find and read document.xml
	var docXML string
	for _, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			rc, err := file.Open()
			if err != nil {
				t.Fatalf("Failed to open document.xml: %v", err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("Failed to read document.xml: %v", err)
			}
			docXML = string(content)
			break
		}
	}
	
	// Check that the variables were replaced
	if strings.Contains(docXML, "{{name}}") {
		t.Error("Template variable {{name}} was not replaced")
	}
	if strings.Contains(docXML, "{{balance}}") {
		t.Error("Template variable {{balance}} was not replaced")
	}
	if !strings.Contains(docXML, "John Doe") {
		t.Error("Expected 'John Doe' in output")
	}
	if !strings.Contains(docXML, "1234.56") {
		t.Error("Expected '1234.56' in output")
	}
}

func TestPageBreakRendering(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		data        TemplateData
		expectBreak bool
		wantText    string
	}{
		{
			name:        "pageBreak function call",
			input:       "Before page break{{pageBreak()}}After page break",
			data:        TemplateData{},
			expectBreak: true,
			wantText:    "Before page breakAfter page break",
		},
		{
			name:        "multiple pageBreaks",
			input:       "Page 1{{pageBreak()}}Page 2{{pageBreak()}}Page 3",
			data:        TemplateData{},
			expectBreak: true,
			wantText:    "Page 1Page 2Page 3",
		},
		{
			name:        "pageBreak with variables",
			input:       "Hello {{name}}{{pageBreak()}}Goodbye {{name}}",
			data:        TemplateData{"name": "World"},
			expectBreak: true,
			wantText:    "Hello WorldGoodbye World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a document with the test content
			doc := &Document{
				Body: createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
								{
									Text: &Text{
										Content: tt.input,
									},
								},
						},
					},
				}),
			}

			result, err := RenderDocument(doc, tt.data)
			if err != nil {
				t.Fatalf("RenderDocument() error = %v", err)
			}

			// Check the rendered text content
			if len(result.Body.Elements) == 0 {
				t.Fatal("No elements in result body")
			}
			para, ok := result.Body.Elements[0].(*Paragraph)
			if !ok {
				t.Fatal("First element is not a Paragraph")
			}
			if len(para.Runs) == 0 {
				t.Fatal("No runs in paragraph")
			}
			
			// Collect all text from all runs
			var allText strings.Builder
			for _, run := range para.Runs {
				if run.Text != nil {
					allText.WriteString(run.Text.Content)
				}
			}
			gotText := allText.String()
			if gotText != tt.wantText {
				t.Errorf("RenderDocument() text = %q, want %q", gotText, tt.wantText)
			}

			// Check if a page break was created
			hasBreak := false
			for _, run := range para.Runs {
				if run.Break != nil && run.Break.Type == "page" {
					hasBreak = true
					break
				}
			}

			if hasBreak != tt.expectBreak {
				t.Errorf("RenderDocument() page break found = %v, want %v", hasBreak, tt.expectBreak)
			}
		})
	}
}

func TestPageBreakOOXMLFragmentHandling(t *testing.T) {
	// Test the OOXML fragment expansion logic
	text := &Text{
		Content: "Before{{OOXML_FRAGMENT:fragment_0}}After",
	}
	
	run := &Run{
		Text: text,
	}
	
	// Create a render context with a page break fragment
	ctx := &renderContext{
		ooxmlFragments: map[string]interface{}{
			"fragment_0": &Break{Type: "page"},
		},
	}
	
	results, err := expandOOXMLFragments(run, TemplateData{}, ctx)
	if err != nil {
		t.Fatalf("expandOOXMLFragments() error = %v", err)
	}
	
	// Should have 3 runs: "Before", page break, "After"
	if len(results) != 3 {
		t.Errorf("expandOOXMLFragments() returned %d runs, want 3", len(results))
		return
	}
	
	// Check first run contains "Before"
	if results[0].Text == nil || results[0].Text.Content != "Before" {
		t.Error("First run should contain 'Before'")
	}
	
	// Check second run is a page break
	if results[1].Break == nil || results[1].Break.Type != "page" {
		t.Error("Second run should be a page break")
	}
	
	// Check third run contains "After"
	if results[2].Text == nil || results[2].Text.Content != "After" {
		t.Error("Third run should contain 'After'")
	}
}