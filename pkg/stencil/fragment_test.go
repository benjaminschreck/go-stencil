package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestFragmentInclude(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		fragments      map[string]string
		data           map[string]interface{}
		expected       string
		expectError    bool
		errorContains  string
	}{
		{
			name:     "simple fragment include with string literal",
			template: "Before {{include \"header\"}} After",
			fragments: map[string]string{
				"header": "Header Content",
			},
			data:     map[string]interface{}{},
			expected: "Before Header Content After",
		},
		{
			name:     "fragment include with variable",
			template: "Before {{include fragmentName}} After",
			fragments: map[string]string{
				"myFragment": "Fragment Content",
			},
			data: map[string]interface{}{
				"fragmentName": "myFragment",
			},
			expected: "Before Fragment Content After",
		},
		{
			name:     "fragment with template expressions",
			template: "{{include \"greeting\"}}",
			fragments: map[string]string{
				"greeting": "Hello {{name}}!",
			},
			data: map[string]interface{}{
				"name": "World",
			},
			expected: "Hello World!",
		},
		{
			name:     "nested fragments",
			template: "{{include \"outer\"}}",
			fragments: map[string]string{
				"outer": "Start {{include \"inner\"}} End",
				"inner": "Nested Content",
			},
			data:     map[string]interface{}{},
			expected: "Start Nested Content End",
		},
		{
			name:     "fragment with control structures",
			template: "{{include \"list\"}}",
			fragments: map[string]string{
				"list": "Items:{{for item in items}} {{item}}{{end}}",
			},
			data: map[string]interface{}{
				"items": []string{"A", "B", "C"},
			},
			expected: "Items: A B C",
		},
		{
			name:     "fragment not found",
			template: "{{include \"missing\"}}",
			fragments: map[string]string{},
			data:     map[string]interface{}{},
			expectError: true,
			errorContains: "fragment not found: missing",
		},
		{
			name:     "fragment name evaluates to non-string",
			template: "{{include 123}}",
			fragments: map[string]string{},
			data:     map[string]interface{}{},
			expectError: true,
			errorContains: "fragment name must be a string",
		},
		{
			name:     "complex nested fragments with data",
			template: "{{include \"page\"}}",
			fragments: map[string]string{
				"page": "{{include \"header\"}} {{content}} {{include \"footer\"}}",
				"header": "=== {{title}} ===",
				"footer": "--- Page {{pageNum}} ---",
			},
			data: map[string]interface{}{
				"title":   "My Document",
				"content": "Main content here",
				"pageNum": 1,
			},
			expected: "=== My Document === Main content here --- Page 1 ---",
		},
		{
			name:     "fragment with functions",
			template: "{{include \"formatted\"}}",
			fragments: map[string]string{
				"formatted": "Name: {{uppercase(name)}}, Count: {{str(count)}}",
			},
			data: map[string]interface{}{
				"name":  "john",
				"count": 42,
			},
			expected: "Name: JOHN, Count: 42",
		},
		{
			name:     "multiple includes of same fragment",
			template: "{{include \"item\"}} and {{include \"item\"}}",
			fragments: map[string]string{
				"item": "[{{value}}]",
			},
			data: map[string]interface{}{
				"value": "X",
			},
			expected: "[X] and [X]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create template
			tmpl, err := Parse("test.docx", tt.template)
			if err != nil {
				t.Fatalf("failed to parse template: %v", err)
			}

			// Add fragments
			for name, content := range tt.fragments {
				err := tmpl.AddFragment(name, content)
				if err != nil {
					t.Fatalf("failed to add fragment %s: %v", name, err)
				}
			}

			// Render
			result, err := tmpl.Render(tt.data)
			
			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("got %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

func TestFragmentInDOCX(t *testing.T) {
	// Create a simple DOCX with fragment includes
	docxContent := createTestDOCXWithFragments(t)
	
	// Create fragment DOCX files
	headerFragment := createFragmentDOCX(t, "Header: {{title}}")
	footerFragment := createFragmentDOCX(t, "Footer: Page {{page}}")
	
	// Create template
	tmpl, err := ParseBytes(docxContent)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	
	// Add fragments as DOCX
	err = tmpl.AddFragmentFromBytes("header", headerFragment)
	if err != nil {
		t.Fatalf("failed to add header fragment: %v", err)
	}
	
	err = tmpl.AddFragmentFromBytes("footer", footerFragment)
	if err != nil {
		t.Fatalf("failed to add footer fragment: %v", err)
	}
	
	// Render with data
	data := map[string]interface{}{
		"title":   "Test Document",
		"content": "Main content here",
		"page":    1,
	}
	
	rendered, err := tmpl.RenderToBytes(data)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}
	
	// Verify the rendered content
	content := extractTextFromDOCX(t, rendered)
	t.Logf("Rendered content: %q", content)
	
	// Also log the raw document XML for debugging
	r, _ := zip.NewReader(bytes.NewReader(rendered), int64(len(rendered)))
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			docXML, _ := io.ReadAll(rc)
			rc.Close()
			t.Logf("Document XML:\n%s", docXML)
			break
		}
	}
	
	if !strings.Contains(content, "Header: Test Document") {
		t.Errorf("rendered content does not contain expected header, got: %q", content)
	}
	if !strings.Contains(content, "Main content here") {
		t.Errorf("rendered content does not contain expected main content, got: %q", content)
	}
	if !strings.Contains(content, "Footer: Page 1") {
		t.Errorf("rendered content does not contain expected footer, got: %q", content)
	}
}

func TestFragmentStyleMerging(t *testing.T) {
	// Create main document with a style
	mainDoc := createDOCXWithStyle(t, "Main {{include \"fragment\"}}", "MainStyle", "color:blue")
	
	// Create fragment with its own style
	fragmentDoc := createDOCXWithStyle(t, "Fragment Text", "FragStyle", "color:red")
	
	// Parse and add fragment
	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	
	err = tmpl.AddFragmentFromBytes("fragment", fragmentDoc)
	if err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}
	
	// Render
	rendered, err := tmpl.RenderToBytes(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}
	
	// Verify both styles are present in the rendered document
	styles := extractStylesFromDOCX(t, rendered)
	t.Logf("Found styles: %v", styles)
	
	hasMainStyle := false
	hasFragStyle := false
	for _, style := range styles {
		if style == "MainStyle" {
			hasMainStyle = true
		}
		if style == "FragStyle" {
			hasFragStyle = true
		}
	}
	if !hasMainStyle {
		t.Error("MainStyle not found in rendered document")
	}
	if !hasFragStyle {
		t.Error("FragStyle not found in rendered document")
	}
}

func TestFragmentStyleMergingMultipleFragments(t *testing.T) {
	// Create main document with a style
	mainDoc := createDOCXWithStyle(t, "Main {{include \"frag1\"}} {{include \"frag2\"}}", "MainStyle", "<w:color w:val=\"0000FF\"/>")

	// Create first fragment with its own style
	frag1Doc := createDOCXWithStyle(t, "Fragment 1", "Frag1Style", "<w:color w:val=\"FF0000\"/>")

	// Create second fragment with its own style
	frag2Doc := createDOCXWithStyle(t, "Fragment 2", "Frag2Style", "<w:color w:val=\"00FF00\"/>")

	// Parse and add fragments
	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	err = tmpl.AddFragmentFromBytes("frag1", frag1Doc)
	if err != nil {
		t.Fatalf("failed to add fragment 1: %v", err)
	}

	err = tmpl.AddFragmentFromBytes("frag2", frag2Doc)
	if err != nil {
		t.Fatalf("failed to add fragment 2: %v", err)
	}

	// Render
	rendered, err := tmpl.RenderToBytes(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	// Verify all three styles are present
	styles := extractStylesFromDOCX(t, rendered)
	t.Logf("Found styles: %v", styles)

	expectedStyles := map[string]bool{
		"MainStyle":   false,
		"Frag1Style":  false,
		"Frag2Style":  false,
	}

	for _, style := range styles {
		if _, exists := expectedStyles[style]; exists {
			expectedStyles[style] = true
		}
	}

	for styleName, found := range expectedStyles {
		if !found {
			t.Errorf("%s not found in rendered document", styleName)
		}
	}
}

func TestFragmentStyleMergingNoDuplicates(t *testing.T) {
	// Create main document with a style
	mainDoc := createDOCXWithStyle(t, "Main {{include \"frag1\"}} {{include \"frag2\"}}", "SharedStyle", "<w:color w:val=\"0000FF\"/>")

	// Create both fragments with the SAME style ID (should not duplicate)
	frag1Doc := createDOCXWithStyle(t, "Fragment 1", "SharedStyle", "<w:color w:val=\"0000FF\"/>")
	frag2Doc := createDOCXWithStyle(t, "Fragment 2", "SharedStyle", "<w:color w:val=\"0000FF\"/>")

	// Parse and add fragments
	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	err = tmpl.AddFragmentFromBytes("frag1", frag1Doc)
	if err != nil {
		t.Fatalf("failed to add fragment 1: %v", err)
	}

	err = tmpl.AddFragmentFromBytes("frag2", frag2Doc)
	if err != nil {
		t.Fatalf("failed to add fragment 2: %v", err)
	}

	// Render
	rendered, err := tmpl.RenderToBytes(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	// Verify SharedStyle appears only once
	styles := extractStylesFromDOCX(t, rendered)
	t.Logf("Found styles: %v", styles)

	sharedStyleCount := 0
	for _, style := range styles {
		if style == "SharedStyle" {
			sharedStyleCount++
		}
	}

	if sharedStyleCount != 1 {
		t.Errorf("SharedStyle should appear exactly once, but appeared %d times", sharedStyleCount)
	}
}

func TestFragmentStyleMergingTableStyles(t *testing.T) {
	// Create main document with a table style
	mainDoc := createDOCXWithTableStyle(t, "Main {{include \"fragment\"}}", "MainTableStyle")

	// Create fragment with its own table style
	fragmentDoc := createDOCXWithTableStyle(t, "Fragment with table", "FragTableStyle")

	// Parse and add fragment
	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	err = tmpl.AddFragmentFromBytes("fragment", fragmentDoc)
	if err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	// Render
	rendered, err := tmpl.RenderToBytes(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	// Verify both table styles are present
	styles := extractStylesFromDOCX(t, rendered)
	t.Logf("Found styles: %v", styles)

	hasMainStyle := false
	hasFragStyle := false
	for _, style := range styles {
		if style == "MainTableStyle" {
			hasMainStyle = true
		}
		if style == "FragTableStyle" {
			hasFragStyle = true
		}
	}

	if !hasMainStyle {
		t.Error("MainTableStyle not found in rendered document")
	}
	if !hasFragStyle {
		t.Error("FragTableStyle not found in rendered document")
	}
}

func TestFragmentAdvancedFeatures(t *testing.T) {
	// Test that fragments support all template features:
	// - Control structures (if/for/unless)
	// - Tables
	// - Functions (including OOXML functions like pageBreak, html, xml)
	// - Variable substitution and expressions

	tests := []struct {
		name         string
		fragmentText string
		data         TemplateData
		expected     string
	}{
		{
			name:         "fragment with if statement",
			fragmentText: "{{if show}}Visible{{else}}Hidden{{end}}",
			data:         TemplateData{"show": true},
			expected:     "Visible",
		},
		{
			name:         "fragment with for loop",
			fragmentText: "Items:{{for item in items}} {{item}}{{end}}",
			data:         TemplateData{"items": []string{"A", "B", "C"}},
			expected:     "Items: A B C",
		},
		{
			name:         "fragment with unless",
			fragmentText: "{{unless hide}}Shown{{end}}",
			data:         TemplateData{"hide": false},
			expected:     "Shown",
		},
		{
			name:         "fragment with nested control structures",
			fragmentText: "{{for i in nums}}{{if i > 5}}Big: {{i}} {{end}}{{end}}",
			data:         TemplateData{"nums": []int{3, 6, 8}},
			expected:     "Big: 6 Big: 8 ",
		},
		{
			name:         "fragment with functions",
			fragmentText: "Name: {{uppercase(name)}}, Count: {{str(count)}}",
			data:         TemplateData{"name": "alice", "count": 42},
			expected:     "Name: ALICE, Count: 42",
		},
		{
			name:         "fragment with expressions",
			fragmentText: "Total: {{price * quantity}}",
			data:         TemplateData{"price": 10.5, "quantity": 3},
			expected:     "Total: 31.5",
		},
		{
			name:         "fragment with nested fragments",
			fragmentText: "Outer {{include \"inner\"}}",
			data:         TemplateData{},
			expected:     "Outer Inner Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create main template
			tmpl, err := Parse("test.docx", "{{include \"test\"}}")
			if err != nil {
				t.Fatalf("failed to parse template: %v", err)
			}

			// Add fragment
			err = tmpl.AddFragment("test", tt.fragmentText)
			if err != nil {
				t.Fatalf("failed to add fragment: %v", err)
			}

			// For nested fragment test, add the inner fragment
			if tt.name == "fragment with nested fragments" {
				err = tmpl.AddFragment("inner", "Inner Content")
				if err != nil {
					t.Fatalf("failed to add inner fragment: %v", err)
				}
			}

			// Render
			result, err := tmpl.Render(tt.data)
			if err != nil {
				t.Fatalf("failed to render: %v", err)
			}

			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFragmentWithTablesAndComplexStructures(t *testing.T) {
	// Create a DOCX fragment with a table and control structures
	fragmentContent := `
	<w:tbl>
		<w:tr>
			<w:tc><w:p><w:r><w:t>{{if header}}Header{{end}}</w:t></w:r></w:p></w:tc>
		</w:tr>
		{{for row in rows}}
		<w:tr>
			<w:tc><w:p><w:r><w:t>{{row}}</w:t></w:r></w:p></w:tc>
		</w:tr>
		{{end}}
	</w:tbl>
	`

	tmpl, err := Parse("test.docx", "{{include \"tableFragment\"}}")
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	err = tmpl.AddFragment("tableFragment", fragmentContent)
	if err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	data := TemplateData{
		"header": true,
		"rows":   []string{"Row 1", "Row 2", "Row 3"},
	}

	_, err = tmpl.Render(data)
	if err != nil {
		t.Fatalf("failed to render fragment with table: %v", err)
	}

	// If we got here without error, tables work in fragments
	t.Log("Fragment with table and control structures rendered successfully")
}

func TestFragmentStyleMergingPreservesTableBorders(t *testing.T) {
	// This test validates the original use case from commit 6f6a439:
	// "Fixes issue where table borders were lost when including fragments
	//  that referenced table styles not present in the main document."

	// Create main document with a basic paragraph style (but no table style)
	mainDoc := createDOCXWithStyle(t, "Main {{include \"tableFragment\"}}", "NormalPara", "<w:color w:val=\"000000\"/>")

	// Create fragment with a table that has custom borders (via table style)
	fragmentDoc := createDOCXWithTableStyle(t, "Fragment with styled table", "CustomTableStyle")

	// Parse and add fragment
	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	err = tmpl.AddFragmentFromBytes("tableFragment", fragmentDoc)
	if err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	// Render
	rendered, err := tmpl.RenderToBytes(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	// Verify the table style from the fragment is present in the rendered document
	styles := extractStylesFromDOCX(t, rendered)
	t.Logf("Found styles: %v", styles)

	hasCustomTableStyle := false
	for _, style := range styles {
		if style == "CustomTableStyle" {
			hasCustomTableStyle = true
			break
		}
	}

	if !hasCustomTableStyle {
		t.Error("CustomTableStyle from fragment not found - table borders would be lost")
	}
}

func TestFragmentCircularReference(t *testing.T) {
	// Create template
	tmpl, err := Parse("test.docx", "{{include \"a\"}}")
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	
	// Add circular fragments
	err = tmpl.AddFragment("a", "A: {{include \"b\"}}")
	if err != nil {
		t.Fatalf("failed to add fragment a: %v", err)
	}
	
	err = tmpl.AddFragment("b", "B: {{include \"a\"}}")
	if err != nil {
		t.Fatalf("failed to add fragment b: %v", err)
	}
	
	// Render should detect circular reference
	_, err = tmpl.Render(map[string]interface{}{})
	if err == nil {
		t.Fatal("expected circular reference error but got none")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error %q does not contain 'circular'", err.Error())
	}
}

// Helper functions

func createTestDOCXWithFragments(_ *testing.T) []byte {
	// Create a DOCX with multiple paragraphs, each containing an include or content
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
	
	// Add word/document.xml with three paragraphs
	doc, _ := w.Create("word/document.xml")
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>{{include "header"}}</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:t>{{content}}</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:t>{{include "footer"}}</w:t>
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

func createFragmentDOCX(t *testing.T, content string) []byte {
	return createSimpleDOCX(t, content)
}

func createDOCXWithStyle(_ *testing.T, content, styleName, styleProps string) []byte {
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
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`)
	
	// Add word/styles.xml with custom style
	styles, _ := w.Create("word/styles.xml")
	io.WriteString(styles, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="`+styleName+`">
    <w:name w:val="`+styleName+`"/>
    <w:rPr>`+styleProps+`</w:rPr>
  </w:style>
</w:styles>`)
	
	// Add word/document.xml
	doc, _ := w.Create("word/document.xml")
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>`+content+`</w:t>
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
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`)
	
	w.Close()
	return buf.Bytes()
}

func createDOCXWithTableStyle(_ *testing.T, content, styleName string) []byte {
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
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`)

	// Add word/styles.xml with custom table style
	styles, _ := w.Create("word/styles.xml")
	io.WriteString(styles, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="table" w:styleId="`+styleName+`">
    <w:name w:val="`+styleName+`"/>
    <w:tblPr>
      <w:tblBorders>
        <w:top w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      </w:tblBorders>
    </w:tblPr>
  </w:style>
</w:styles>`)

	// Add word/document.xml
	doc, _ := w.Create("word/document.xml")
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>`+content+`</w:t>
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
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`)

	w.Close()
	return buf.Bytes()
}

func extractStylesFromDOCX(t *testing.T, docxBytes []byte) []string {
	r, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}
	
	var styles []string
	for _, f := range r.File {
		if f.Name == "word/styles.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open styles.xml: %v", err)
			}
			defer rc.Close()
			
			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("failed to read styles.xml: %v", err)
			}
			
			// Extract style IDs using regex for more accurate extraction
			text := string(content)
			// Look for w:styleId="..." patterns
			re := regexp.MustCompile(`w:styleId="([^"]+)"`)
			matches := re.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) > 1 {
					styles = append(styles, match[1])
				}
			}
		}
	}
	
	return styles
}

// createSimpleDOCX creates a simple DOCX file with the given content for testing
func createSimpleDOCX(t *testing.T, content string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	
	// Add _rels/.rels
	rels, err := w.Create("_rels/.rels")
	if err != nil {
		t.Fatalf("failed to create _rels/.rels: %v", err)
	}
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)
	
	// Add word/_rels/document.xml.rels
	wordRels, err := w.Create("word/_rels/document.xml.rels")
	if err != nil {
		t.Fatalf("failed to create word/_rels/document.xml.rels: %v", err)
	}
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`)
	
	// Add word/document.xml
	doc, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatalf("failed to create word/document.xml: %v", err)
	}
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>`+content+`</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`)
	
	// Add [Content_Types].xml
	ct, err := w.Create("[Content_Types].xml")
	if err != nil {
		t.Fatalf("failed to create [Content_Types].xml: %v", err)
	}
	io.WriteString(ct, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`)
	
	w.Close()
	return buf.Bytes()
}

// extractTextFromDOCX extracts text content from a DOCX file
func extractTextFromDOCX(t *testing.T, docxBytes []byte) string {
	r, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("failed to read zip: %v", err)
	}
	
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open document.xml: %v", err)
			}
			defer rc.Close()
			
			content, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("failed to read document.xml: %v", err)
			}
			
			// Extract text from w:t elements
			return extractTextFromDocumentXML(string(content))
		}
	}
	
	return ""
}

// extractTextFromDocumentXML extracts plain text from document XML
func extractTextFromDocumentXML(xmlContent string) string {
	// Simple extraction of text between <w:t> tags
	var result strings.Builder
	inText := false
	tagStart := -1
	
	for i, ch := range xmlContent {
		if ch == '<' {
			tagStart = i
			if inText && tagStart > 0 {
				// We were in text, now we're not
				inText = false
			}
		} else if ch == '>' && tagStart >= 0 {
			tag := xmlContent[tagStart+1 : i]
			if tag == "w:t" || strings.HasPrefix(tag, "w:t ") {
				inText = true
			} else if tag == "/w:t" {
				inText = false
			}
			tagStart = -1
		} else if inText && tagStart < 0 {
			result.WriteRune(ch)
		}
	}
	
	return result.String()
}

func TestFragmentParsingPreservesNamespaces(t *testing.T) {
	// Load a real fragment DOCX
	fragmentPath := "../../examples/advanced/fragments/fragment1.docx"
	fragmentBytes, err := os.ReadFile(fragmentPath)
	if err != nil {
		t.Skip("Fragment file not available")
	}

	// Parse as DOCX
	reader := bytes.NewReader(fragmentBytes)
	docxReader, err := NewDocxReader(reader, int64(len(fragmentBytes)))
	if err != nil {
		t.Fatalf("Failed to read DOCX: %v", err)
	}

	// Get document XML
	docXML, err := docxReader.GetDocumentXML()
	if err != nil {
		t.Fatalf("Failed to get document XML: %v", err)
	}

	// Parse document
	doc, err := ParseDocument(bytes.NewReader([]byte(docXML)))
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Verify Attrs were preserved during parsing
	if len(doc.Attrs) == 0 {
		t.Fatal("No attributes preserved during parsing!")
	}

	// Log what we found
	t.Logf("✅ Fragment parsed with %d attributes:", len(doc.Attrs))
	for _, attr := range doc.Attrs {
		t.Logf("  - %s:%s = %s", attr.Name.Space, attr.Name.Local, attr.Value)
	}

	// Check for namespace attributes
	hasNamespaces := false
	for _, attr := range doc.Attrs {
		if attr.Name.Local == "xmlns" || strings.HasPrefix(attr.Name.Local, "xmlns:") || attr.Name.Space == "xmlns" {
			hasNamespaces = true
			break
		}
	}

	if !hasNamespaces {
		t.Error("No namespace attributes found in parsed document")
	}
}