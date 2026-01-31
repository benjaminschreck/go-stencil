package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

// createTestDocxForValidation creates a minimal DOCX with the given content for validation testing
func createTestDocxForValidation(content string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add _rels/.rels
	rels, _ := w.Create("_rels/.rels")
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	// Add word/_rels/document.xml.rels
	wordRels, _ := w.Create("word/_rels/document.xml.rels")
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`)

	// Add word/document.xml with the test content
	doc, _ := w.Create("word/document.xml")
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>`+escapeXML(content)+`</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`)

	// Add [Content_Types].xml
	ct, _ := w.Create("[Content_Types].xml")
	io.WriteString(ct, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`)

	w.Close()
	return buf.Bytes()
}

// createMultiParagraphDocx creates a DOCX with multiple paragraphs
func createMultiParagraphDocx(paragraphs []string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Add _rels/.rels
	rels, _ := w.Create("_rels/.rels")
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	// Add word/_rels/document.xml.rels
	wordRels, _ := w.Create("word/_rels/document.xml.rels")
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`)

	// Build paragraphs XML
	var parasXML strings.Builder
	for _, p := range paragraphs {
		parasXML.WriteString(`<w:p><w:r><w:t>`)
		parasXML.WriteString(escapeXML(p))
		parasXML.WriteString(`</w:t></w:r></w:p>`)
	}

	// Add word/document.xml
	doc, _ := w.Create("word/document.xml")
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>`+parasXML.String()+`</w:body>
</w:document>`)

	// Add [Content_Types].xml
	ct, _ := w.Create("[Content_Types].xml")
	io.WriteString(ct, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`)

	w.Close()
	return buf.Bytes()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func TestValidate_ValidTemplate(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "simple variable",
			content: "Hello {{name}}!",
		},
		{
			name:    "nested variable",
			content: "Address: {{customer.address.city}}",
		},
		{
			name:    "expression",
			content: "Total: {{price * quantity}}",
		},
		{
			name:    "function call",
			content: "Name: {{uppercase(name)}}",
		},
		{
			name:    "inline if",
			content: "{{if active}}Active{{end}}",
		},
		{
			name:    "inline for",
			content: "{{for item in items}}{{item}}{{end}}",
		},
		{
			name:    "complex expression",
			content: "{{if (price > 100) & (quantity > 5)}}Discount{{end}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docxBytes := createTestDocxForValidation(tt.content)
			tmpl, err := Prepare(bytes.NewReader(docxBytes))
			if err != nil {
				t.Fatalf("failed to prepare template: %v", err)
			}
			defer tmpl.Close()

			result := tmpl.Validate()

			if !result.Valid {
				t.Errorf("expected valid template, got errors: %v", result.Errors)
			}
		})
	}
}

func TestValidate_InvalidExpressions(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		errorType   ValidationErrorType
		errorSubstr string
	}{
		{
			name:        "unknown function",
			content:     "{{unknownFunc(x)}}",
			errorType:   ValidationErrorUnknownFunction,
			errorSubstr: "unknownFunc",
		},
		{
			name:        "invalid for loop",
			content:     "{{for x}}content{{end}}",
			errorType:   ValidationErrorInvalidFor,
			errorSubstr: "in",
		},
		{
			name:        "unclosed string in expression",
			content:     "{{name + 'unclosed}}",
			errorType:   ValidationErrorExpression,
			errorSubstr: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docxBytes := createTestDocxForValidation(tt.content)
			tmpl, err := Prepare(bytes.NewReader(docxBytes))
			if err != nil {
				t.Fatalf("failed to prepare template: %v", err)
			}
			defer tmpl.Close()

			result := tmpl.Validate()

			if result.Valid {
				t.Errorf("expected invalid template")
			}

			found := false
			for _, e := range result.Errors {
				if e.Type == tt.errorType && strings.Contains(e.Message, tt.errorSubstr) {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected error type %s containing %q, got: %v",
					tt.errorType, tt.errorSubstr, result.Errors)
			}
		})
	}
}

func TestValidate_UnbalancedControlStructures(t *testing.T) {
	tests := []struct {
		name       string
		paragraphs []string
		errorCount int
		errorType  ValidationErrorType
	}{
		{
			name: "missing end for if",
			paragraphs: []string{
				"{{if condition}}",
				"Some content",
				// Missing {{end}}
			},
			errorCount: 1,
			errorType:  ValidationErrorUnbalanced,
		},
		{
			name: "missing end for for loop",
			paragraphs: []string{
				"{{for item in items}}",
				"{{item}}",
				// Missing {{end}}
			},
			errorCount: 1,
			errorType:  ValidationErrorUnbalanced,
		},
		{
			name: "extra end",
			paragraphs: []string{
				"Some content",
				"{{end}}",
			},
			errorCount: 1,
			errorType:  ValidationErrorUnbalanced,
		},
		{
			name: "else without if",
			paragraphs: []string{
				"{{else}}",
				"Content",
				"{{end}}",
			},
			errorCount: 2, // else without if AND end without opener
			errorType:  ValidationErrorUnbalanced,
		},
		{
			name: "properly balanced",
			paragraphs: []string{
				"{{if condition}}",
				"Content",
				"{{end}}",
			},
			errorCount: 0,
		},
		{
			name: "nested balanced",
			paragraphs: []string{
				"{{for item in items}}",
				"{{if item.active}}",
				"{{item.name}}",
				"{{end}}",
				"{{end}}",
			},
			errorCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docxBytes := createMultiParagraphDocx(tt.paragraphs)
			tmpl, err := Prepare(bytes.NewReader(docxBytes))
			if err != nil {
				t.Fatalf("failed to prepare template: %v", err)
			}
			defer tmpl.Close()

			result := tmpl.Validate()

			if len(result.Errors) != tt.errorCount {
				t.Errorf("expected %d errors, got %d: %v",
					tt.errorCount, len(result.Errors), result.Errors)
			}

			if tt.errorCount > 0 && len(result.Errors) > 0 {
				if result.Errors[0].Type != tt.errorType {
					t.Errorf("expected error type %s, got %s",
						tt.errorType, result.Errors[0].Type)
				}
			}
		})
	}
}

func TestValidate_VariableExtraction(t *testing.T) {
	content := "Hello {{name}}, your order {{order.id}} totals {{price * quantity}}."
	docxBytes := createTestDocxForValidation(content)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	result := tmpl.Validate()

	if !result.Valid {
		t.Errorf("expected valid template, got errors: %v", result.Errors)
	}

	expectedVars := map[string]bool{
		"name":     true,
		"order":    true,
		"price":    true,
		"quantity": true,
	}

	for _, v := range result.Variables {
		if !expectedVars[v] {
			// It's okay to have extra variables from nested access
		}
		delete(expectedVars, v)
	}

	// All expected variables should be found
	for v := range expectedVars {
		t.Errorf("expected variable %q not found", v)
	}
}

func TestValidate_FunctionExtraction(t *testing.T) {
	content := "{{uppercase(name)}} - {{format('%.2f', price)}} - {{sum(totals)}}"
	docxBytes := createTestDocxForValidation(content)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	result := tmpl.Validate()

	if !result.Valid {
		t.Errorf("expected valid template, got errors: %v", result.Errors)
	}

	expectedFuncs := map[string]bool{
		"uppercase": true,
		"format":    true,
		"sum":       true,
	}

	for _, f := range result.Functions {
		delete(expectedFuncs, f)
	}

	for f := range expectedFuncs {
		t.Errorf("expected function %q not found", f)
	}
}

func TestValidate_FragmentValidation(t *testing.T) {
	content := `{{include "header"}}`
	docxBytes := createTestDocxForValidation(content)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Without adding the fragment, validation should fail
	result := tmpl.Validate()

	if result.Valid {
		t.Errorf("expected invalid template (missing fragment)")
	}

	foundMissingFragment := false
	for _, e := range result.Errors {
		if e.Type == ValidationErrorMissingFragment {
			foundMissingFragment = true
			break
		}
	}

	if !foundMissingFragment {
		t.Errorf("expected missing fragment error, got: %v", result.Errors)
	}

	// After adding the fragment, validation should pass
	tmpl.AddFragment("header", "Header Content")
	result = tmpl.Validate()

	if !result.Valid {
		t.Errorf("expected valid template after adding fragment, got: %v", result.Errors)
	}
}

func TestValidate_WithOptions(t *testing.T) {
	content := "{{unknownFunc(x)}}"
	docxBytes := createTestDocxForValidation(content)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// With function checking disabled, should be valid
	opts := ValidationOptions{
		CheckFunctions: false,
		CheckFragments: false,
		StrictMode:     false,
	}

	result := tmpl.ValidateWithOptions(opts)

	if !result.Valid {
		t.Errorf("expected valid template with function checking disabled, got: %v", result.Errors)
	}
}

func TestValidate_StrictMode(t *testing.T) {
	// Create content that produces warnings (extra closing delimiter)
	content := "Hello {{name}} }}"
	docxBytes := createTestDocxForValidation(content)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Normal mode: warnings don't affect validity
	result := tmpl.Validate()
	hasWarnings := len(result.Warnings) > 0

	// Strict mode: warnings become errors
	opts := ValidationOptions{
		CheckFunctions: true,
		CheckFragments: true,
		StrictMode:     true,
	}
	strictResult := tmpl.ValidateWithOptions(opts)

	if hasWarnings && strictResult.Valid {
		t.Errorf("expected strict mode to treat warnings as errors")
	}
}

func TestValidate_ControlStructureInfo(t *testing.T) {
	paragraphs := []string{
		"{{for item in items}}",
		"{{if item.active}}",
		"Active: {{item.name}}",
		"{{end}}",
		"{{end}}",
	}

	docxBytes := createMultiParagraphDocx(paragraphs)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	result := tmpl.Validate()

	if !result.Valid {
		t.Errorf("expected valid template, got: %v", result.Errors)
	}

	if len(result.ControlStructs) < 2 {
		t.Errorf("expected at least 2 control structures, got %d", len(result.ControlStructs))
	}

	foundFor := false
	foundIf := false
	for _, cs := range result.ControlStructs {
		if cs.Type == "for" {
			foundFor = true
		}
		if cs.Type == "if" {
			foundIf = true
		}
	}

	if !foundFor {
		t.Error("expected to find 'for' control structure")
	}
	if !foundIf {
		t.Error("expected to find 'if' control structure")
	}
}

func TestValidationResult_String(t *testing.T) {
	content := "Hello {{name}}!"
	docxBytes := createTestDocxForValidation(content)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	result := tmpl.Validate()
	str := result.String()

	if !strings.Contains(str, "valid") {
		t.Errorf("expected 'valid' in string output, got: %s", str)
	}

	if !strings.Contains(str, "name") {
		t.Errorf("expected variable 'name' in string output, got: %s", str)
	}
}

func TestValidationResult_Error(t *testing.T) {
	content := "{{unknownFunc()}}"
	docxBytes := createTestDocxForValidation(content)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	result := tmpl.Validate()

	if result.Valid {
		t.Skip("template is valid, cannot test Error()")
	}

	err = result.Error()
	if err == nil {
		t.Error("expected Error() to return error for invalid template")
	}

	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected 'validation failed' in error, got: %v", err)
	}
}

func TestValidateExpression(t *testing.T) {
	tests := []struct {
		expr    string
		wantErr bool
	}{
		{"name", false},
		{"price * quantity", false},
		{"uppercase(name)", false},
		{"unclosed(", true},          // Unclosed parenthesis
		{"'unclosed string", true},   // Unclosed string
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			err := ValidateExpression(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExpression(%q) error = %v, wantErr %v",
					tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestValidateForSyntax(t *testing.T) {
	tests := []struct {
		expr    string
		wantErr bool
	}{
		{"item in items", false},
		{"i, item in items", false},
		{"item items", true}, // Missing 'in'
		{"", true},           // Empty
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			err := ValidateForSyntax(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateForSyntax(%q) error = %v, wantErr %v",
					tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestExtractTemplateTokens(t *testing.T) {
	text := "Hello {{name}}, your order is {{order.id}}. Total: {{price * qty}}"

	tokens := ExtractTemplateTokens(text)

	expected := []string{"name", "order.id", "price * qty"}

	if len(tokens) != len(expected) {
		t.Errorf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}

	for i, tok := range tokens {
		if i < len(expected) && tok != expected[i] {
			t.Errorf("token %d: expected %q, got %q", i, expected[i], tok)
		}
	}
}

func TestValidate_NilTemplate(t *testing.T) {
	var tmpl *PreparedTemplate
	result := tmpl.Validate()

	if result.Valid {
		t.Error("expected invalid result for nil template")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error for nil template")
	}
}

func TestValidate_EmptyCondition(t *testing.T) {
	content := "{{if }}content{{end}}"
	docxBytes := createTestDocxForValidation(content)
	tmpl, err := Prepare(bytes.NewReader(docxBytes))
	if err != nil {
		t.Fatalf("failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	result := tmpl.Validate()

	foundEmptyCondition := false
	for _, e := range result.Errors {
		if e.Type == ValidationErrorInvalidCondition && strings.Contains(e.Message, "empty") {
			foundEmptyCondition = true
			break
		}
	}

	if !foundEmptyCondition {
		t.Errorf("expected empty condition error, got: %v", result.Errors)
	}
}
