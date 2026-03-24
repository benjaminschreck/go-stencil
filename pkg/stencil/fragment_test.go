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
		name          string
		template      string
		fragments     map[string]string
		data          map[string]interface{}
		expected      string
		expectError   bool
		errorContains string
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
			name:     "fragment include with German typographic quotes",
			template: "Before {{include \u201Eheader\"}} After", // „header" with ASCII closing quote
			fragments: map[string]string{
				"header": "Header Content",
			},
			data:     map[string]interface{}{},
			expected: "Before Header Content After",
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
			name:          "fragment not found",
			template:      "{{include \"missing\"}}",
			fragments:     map[string]string{},
			data:          map[string]interface{}{},
			expectError:   true,
			errorContains: "fragment not found: missing",
		},
		{
			name:          "fragment name evaluates to non-string",
			template:      "{{include 123}}",
			fragments:     map[string]string{},
			data:          map[string]interface{}{},
			expectError:   true,
			errorContains: "fragment name must be a string",
		},
		{
			name:     "complex nested fragments with data",
			template: "{{include \"page\"}}",
			fragments: map[string]string{
				"page":   "{{include \"header\"}} {{content}} {{include \"footer\"}}",
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

func TestDOCXIncludeSameFragmentMultipleParagraphs(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "frag"}}`,
		`{{include "frag"}}`,
	})
	fragmentDoc := createFragmentDOCX(t, "Hallo")

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{})
	if err != nil {
		t.Fatalf("unexpected error rendering docx: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	if count := strings.Count(content, "Hallo"); count != 2 {
		t.Fatalf("expected fragment text twice, got %d occurrences in %q", count, content)
	}
}

func TestDOCXFragmentInsideInlineIf(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{if show}}{{include "frag"}}{{end}}`,
	})
	fragmentDoc := createFragmentDOCX(t, "Fragment {{name}}")

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{
		"show": true,
		"name": "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error rendering docx fragment in inline if: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	if !strings.Contains(content, "Fragment Alice") {
		t.Fatalf("expected rendered fragment text in output, got %q", content)
	}

	renderedHidden, err := tmpl.RenderToBytes(TemplateData{
		"show": false,
		"name": "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error rendering hidden docx fragment: %v", err)
	}

	hiddenContent := extractTextFromDOCX(t, renderedHidden)
	if strings.Contains(hiddenContent, "Fragment Alice") {
		t.Fatalf("expected fragment text to be omitted when condition is false, got %q", hiddenContent)
	}
}

func TestDOCXFragmentInlineIfWithBraceSplitEndAcrossRuns(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "frag"}}`,
	})
	fragmentDoc := createDOCXWithBodyXML(t, `
    <w:p>
      <w:r>
        <w:t>{{if show}}</w:t>
      </w:r>
      <w:r>
        <w:t>Visible</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:b/></w:rPr>
        <w:t>{</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:i/></w:rPr>
        <w:t>{end}}</w:t>
      </w:r>
    </w:p>`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{
		"show": true,
	})
	if err != nil {
		t.Fatalf("unexpected error rendering fragment with brace-split {{end}}: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	if !strings.Contains(content, "Visible") {
		t.Fatalf("expected rendered fragment text in output, got %q", content)
	}

	renderedHidden, err := tmpl.RenderToBytes(TemplateData{
		"show": false,
	})
	if err != nil {
		t.Fatalf("unexpected error rendering hidden fragment with brace-split {{end}}: %v", err)
	}

	hiddenContent := extractTextFromDOCX(t, renderedHidden)
	if strings.Contains(hiddenContent, "Visible") {
		t.Fatalf("expected fragment text to be omitted when condition is false, got %q", hiddenContent)
	}
}

func TestDOCXNestedIncludedFragmentWithManyNestedInlineStatementsAndBraceSplitEnd(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "outer"}}`,
	})
	outerDoc := createDOCXWithParagraphs(t, []string{
		`{{if showOuter}}{{unless suppressOuter}}{{end}}{{for _ in emptyOuter}}{{end}}`,
		`Outer prefix`,
		`{{include "middle"}}`,
		`{{else}}`,
		`Outer hidden`,
		`{{end}}`,
	})
	middleDoc := createDOCXWithParagraphs(t, []string{
		`{{unless hideMiddle}}{{if showMiddleHeader}}{{end}}{{for _ in emptyMiddle}}{{end}}`,
		`Middle prefix`,
		`{{include "inner"}}`,
		`{{else}}`,
		`Middle hidden`,
		`{{end}}`,
	})
	innerDoc := createDOCXWithBodyXML(t, `
    <w:p>
      <w:r>
        <w:t>{{if showInner}}{{if isVIP}}VIP{{else}}STD{{end}}{{unless muted}} / {{end}}{{if hasRef}}Ref: {{reference}}{{end}}{{if showNote}}{{if urgent}} / URGENT{{else}} / note{{end}}{{end}}</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:b/></w:rPr>
        <w:t>{</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:i/></w:rPr>
        <w:t>{end}}</w:t>
      </w:r>
    </w:p>`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("middle", middleDoc); err != nil {
		t.Fatalf("failed to add middle fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", innerDoc); err != nil {
		t.Fatalf("failed to add inner fragment: %v", err)
	}

	renderedVisible, err := tmpl.RenderToBytes(TemplateData{
		"showOuter":        true,
		"suppressOuter":    false,
		"emptyOuter":       []interface{}{},
		"hideMiddle":       false,
		"showMiddleHeader": true,
		"emptyMiddle":      []interface{}{},
		"showInner":        true,
		"isVIP":            true,
		"muted":            false,
		"hasRef":           true,
		"reference":        "4711",
		"showNote":         true,
		"urgent":           true,
	})
	if err != nil {
		t.Fatalf("unexpected error rendering visible nested fragment chain: %v", err)
	}

	visibleContent := extractTextFromDOCX(t, renderedVisible)
	if !strings.Contains(visibleContent, "Outer prefix") {
		t.Fatalf("expected outer content in visible output, got %q", visibleContent)
	}
	if !strings.Contains(visibleContent, "Middle prefix") {
		t.Fatalf("expected middle content in visible output, got %q", visibleContent)
	}
	if !strings.Contains(visibleContent, "VIP / Ref: 4711 / URGENT") {
		t.Fatalf("expected inner nested inline content in visible output, got %q", visibleContent)
	}
	if strings.Contains(visibleContent, "Outer hidden") || strings.Contains(visibleContent, "Middle hidden") {
		t.Fatalf("did not expect hidden branches in visible output, got %q", visibleContent)
	}

	renderedMiddleHidden, err := tmpl.RenderToBytes(TemplateData{
		"showOuter":        true,
		"suppressOuter":    false,
		"emptyOuter":       []interface{}{},
		"hideMiddle":       true,
		"showMiddleHeader": true,
		"emptyMiddle":      []interface{}{},
		"showInner":        true,
		"isVIP":            true,
		"muted":            false,
		"hasRef":           true,
		"reference":        "4711",
		"showNote":         true,
		"urgent":           true,
	})
	if err != nil {
		t.Fatalf("unexpected error rendering middle-hidden nested fragment chain: %v", err)
	}

	middleHiddenContent := extractTextFromDOCX(t, renderedMiddleHidden)
	if !strings.Contains(middleHiddenContent, "Outer prefix") || !strings.Contains(middleHiddenContent, "Middle hidden") {
		t.Fatalf("expected outer content and middle hidden branch, got %q", middleHiddenContent)
	}
	if strings.Contains(middleHiddenContent, "VIP / Ref: 4711 / URGENT") {
		t.Fatalf("did not expect inner content when middle branch is hidden, got %q", middleHiddenContent)
	}

	renderedOuterHidden, err := tmpl.RenderToBytes(TemplateData{
		"showOuter":        false,
		"suppressOuter":    false,
		"emptyOuter":       []interface{}{},
		"hideMiddle":       false,
		"showMiddleHeader": true,
		"emptyMiddle":      []interface{}{},
		"showInner":        true,
		"isVIP":            false,
		"muted":            true,
		"hasRef":           true,
		"reference":        "4711",
		"showNote":         true,
		"urgent":           false,
	})
	if err != nil {
		t.Fatalf("unexpected error rendering outer-hidden nested fragment chain: %v", err)
	}

	outerHiddenContent := extractTextFromDOCX(t, renderedOuterHidden)
	if !strings.Contains(outerHiddenContent, "Outer hidden") {
		t.Fatalf("expected outer hidden branch, got %q", outerHiddenContent)
	}
	if strings.Contains(outerHiddenContent, "Middle prefix") || strings.Contains(outerHiddenContent, "VIP") || strings.Contains(outerHiddenContent, "STD") {
		t.Fatalf("did not expect nested content when outer branch is hidden, got %q", outerHiddenContent)
	}
}

func TestDOCXNestedIncludedFragmentWithNestedIfAndForLoopAndBraceSplitEnd(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "outer"}}`,
	})
	outerDoc := createDOCXWithParagraphs(t, []string{
		`{{if showOuter}}{{include "middle"}}{{else}}Outer hidden{{end}}`,
	})
	middleDoc := createDOCXWithParagraphs(t, []string{
		`{{unless hideMiddle}}{{include "inner"}}{{else}}Middle hidden{{end}}`,
	})
	innerDoc := createDOCXWithBodyXML(t, `
    <w:p>
      <w:r>
        <w:t>{{if showInner}}{{unless muted}}{{end}}{{for _ in emptyInner}}</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:b/></w:rPr>
        <w:t>{</w:t>
      </w:r>
      <w:r>
        <w:rPr><w:i/></w:rPr>
        <w:t>{end}}</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:t>Lead: {{for i, item in items}}{{if i &gt; 0}}; {{end}}{{if item.active}}{{item.label}}{{else}}skip{{end}}{{end}}{{if showTail}}{{if urgent}} / HOT{{else}} / tail{{end}}{{end}}</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:t>{{else}}</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:t>Inner hidden</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:t>{{end}}</w:t>
      </w:r>
    </w:p>`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("middle", middleDoc); err != nil {
		t.Fatalf("failed to add middle fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", innerDoc); err != nil {
		t.Fatalf("failed to add inner fragment: %v", err)
	}

	renderedVisible, err := tmpl.RenderToBytes(TemplateData{
		"showOuter":  true,
		"hideMiddle": false,
		"showInner":  true,
		"muted":      true,
		"emptyInner": []interface{}{},
		"showTail":   true,
		"urgent":     false,
		"items": []interface{}{
			map[string]interface{}{"label": "A", "active": true},
			map[string]interface{}{"label": "B", "active": false},
			map[string]interface{}{"label": "C", "active": true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering visible nested if/for fragment chain: %v", err)
	}

	visibleContent := extractTextFromDOCX(t, renderedVisible)
	if !strings.Contains(visibleContent, "Lead: A; skip; C / tail") {
		t.Fatalf("expected nested if/for content in visible output, got %q", visibleContent)
	}
	if strings.Contains(visibleContent, "Outer hidden") || strings.Contains(visibleContent, "Middle hidden") {
		t.Fatalf("did not expect hidden branches in visible output, got %q", visibleContent)
	}

	renderedMiddleHidden, err := tmpl.RenderToBytes(TemplateData{
		"showOuter":  true,
		"hideMiddle": true,
		"showInner":  true,
		"muted":      true,
		"emptyInner": []interface{}{},
		"showTail":   true,
		"urgent":     true,
		"items": []interface{}{
			map[string]interface{}{"label": "A", "active": true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering middle hidden nested if/for fragment chain: %v", err)
	}

	middleHiddenContent := extractTextFromDOCX(t, renderedMiddleHidden)
	if !strings.Contains(middleHiddenContent, "Middle hidden") {
		t.Fatalf("expected middle hidden branch, got %q", middleHiddenContent)
	}
	if strings.Contains(middleHiddenContent, "Lead:") {
		t.Fatalf("did not expect inner loop content when middle branch is hidden, got %q", middleHiddenContent)
	}

	renderedOuterHidden, err := tmpl.RenderToBytes(TemplateData{
		"showOuter":  false,
		"hideMiddle": false,
		"showInner":  true,
		"muted":      true,
		"emptyInner": []interface{}{},
		"showTail":   true,
		"urgent":     true,
		"items": []interface{}{
			map[string]interface{}{"label": "A", "active": true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering outer hidden nested if/for fragment chain: %v", err)
	}

	outerHiddenContent := extractTextFromDOCX(t, renderedOuterHidden)
	if !strings.Contains(outerHiddenContent, "Outer hidden") {
		t.Fatalf("expected outer hidden branch, got %q", outerHiddenContent)
	}
	if strings.Contains(outerHiddenContent, "Lead:") || strings.Contains(outerHiddenContent, "Middle hidden") {
		t.Fatalf("did not expect nested content when outer branch is hidden, got %q", outerHiddenContent)
	}
}

func TestDOCXFragmentPreservesLiteralTextAroundIncludeInSameParagraph(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`Before {{include "frag"}} After`,
	})
	fragmentDoc := createFragmentDOCX(t, "Fragment")

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{})
	if err != nil {
		t.Fatalf("unexpected error rendering docx fragment with surrounding text: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	if !strings.Contains(content, "Before Fragment After") {
		t.Fatalf("expected surrounding text to be preserved, got %q", content)
	}
}

func TestDOCXFragmentPreservesLiteralTextAroundMultipleIncludesInSameParagraph(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`A {{include "left"}} B {{include "right"}} C`,
	})
	leftDoc := createFragmentDOCX(t, "L")
	rightDoc := createFragmentDOCX(t, "R")

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("left", leftDoc); err != nil {
		t.Fatalf("failed to add left fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("right", rightDoc); err != nil {
		t.Fatalf("failed to add right fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{})
	if err != nil {
		t.Fatalf("unexpected error rendering multiple docx fragments with surrounding text: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	if !strings.Contains(content, "A L B R C") {
		t.Fatalf("expected surrounding text for multiple includes to be preserved, got %q", content)
	}
}

func TestDOCXFragmentPreservesLiteralTextAroundIncludeInsideIncludedFragment(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "outer"}}`,
	})
	outerDoc := createDOCXWithParagraphs(t, []string{
		`Before {{include "inner"}} After`,
	})
	innerDoc := createFragmentDOCX(t, "Nested")

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", innerDoc); err != nil {
		t.Fatalf("failed to add inner fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{})
	if err != nil {
		t.Fatalf("unexpected error rendering nested included fragment with surrounding text: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	if !strings.Contains(content, "Before Nested After") {
		t.Fatalf("expected surrounding text inside included fragment to be preserved, got %q", content)
	}
}

func TestDOCXFragmentElseIfElseEndAndNestedControlsInSameParagraph(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "frag"}}`,
	})
	fragmentDoc := createDOCXWithParagraphs(t, []string{
		`{{if isOwner}}{{for aktivpartei in aktivseite}}{{if aktivpartei == ownerName}}Eigentümer {{aktivpartei}}{{end}}{{end}}`,
		`{{elseif fahrerMandant}}{{for aktivpartei in aktivseite}}{{if aktivpartei == fahrerName}}{{if isVIP}}VIP {{else}}Fahrer {{end}}{{aktivpartei}}{{end}}{{end}}{{else}}der Fahrer{{end}}{{if leasingTyp == "Finanzierung"}} finanziert{{elseif leasingTyp == "Leasing"}} geleast{{else}} im Eigentum{{end}}, {{for i, passivpartei in passivseite}}{{if i > 0}} / {{end}}{{passivpartei}}{{end}}`,
	})

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	renderedDriver, err := tmpl.RenderToBytes(TemplateData{
		"isOwner":       false,
		"fahrerMandant": true,
		"isVIP":         true,
		"ownerName":     "Alice",
		"fahrerName":    "Bob",
		"aktivseite":    []string{"Alice", "Bob"},
		"leasingTyp":    "Finanzierung",
		"passivseite":   []string{"Beklagte 1", "Beklagte 2"},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering fragment with elseif/else/end in same paragraph: %v", err)
	}

	driverContent := extractTextFromDOCX(t, renderedDriver)
	if !strings.Contains(driverContent, "VIP Bob finanziert, Beklagte 1 / Beklagte 2") {
		t.Fatalf("expected driver branch with nested controls, got %q", driverContent)
	}

	renderedElse, err := tmpl.RenderToBytes(TemplateData{
		"isOwner":       false,
		"fahrerMandant": false,
		"isVIP":         false,
		"ownerName":     "Alice",
		"fahrerName":    "Bob",
		"aktivseite":    []string{"Alice", "Bob"},
		"leasingTyp":    "Leasing",
		"passivseite":   []string{"Beklagte 1", "Beklagte 2"},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering else branch fragment with same-paragraph controls: %v", err)
	}

	elseContent := extractTextFromDOCX(t, renderedElse)
	if !strings.Contains(elseContent, "der Fahrer geleast, Beklagte 1 / Beklagte 2") {
		t.Fatalf("expected else branch with trailing controls, got %q", elseContent)
	}
}

func TestDOCXFragmentAllControlsNestedInSameParagraph(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "frag"}}`,
	})
	fragmentDoc := createDOCXWithParagraphs(t, []string{
		`{{for i, item in items}}{{if i > 0}} | {{end}}{{if item.kind == "vip"}}VIP {{item.name}}{{elseif item.kind == "std"}}{{unless item.skip}}STD {{item.name}}{{else}}SKIP {{item.name}}{{end}}{{else}}OTHER {{item.name}}{{end}}{{end}}`,
	})

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{
		"items": []interface{}{
			map[string]interface{}{"name": "Alice", "kind": "vip", "skip": false},
			map[string]interface{}{"name": "Bob", "kind": "std", "skip": false},
			map[string]interface{}{"name": "Cara", "kind": "std", "skip": true},
			map[string]interface{}{"name": "Dora", "kind": "other", "skip": false},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering fragment with all nested same-paragraph controls: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	if !strings.Contains(content, "VIP Alice | STD Bob | SKIP Cara | OTHER Dora") {
		t.Fatalf("expected fully nested same-paragraph control output, got %q", content)
	}
}

func TestDOCXFragmentForNestedInIfSameParagraph(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "frag"}}`,
	})
	fragmentDoc := createDOCXWithParagraphs(t, []string{
		`{{if show}}{{for i, item in items}}{{if i > 0}} | {{end}}{{if i == 0}}FIRST {{item.name}}{{elseif item.active}}ON {{item.name}}{{else}}OFF {{item.name}}{{end}}{{end}}{{else}}Hidden{{end}}`,
	})

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	renderedShown, err := tmpl.RenderToBytes(TemplateData{
		"show": true,
		"items": []interface{}{
			map[string]interface{}{"name": "A", "active": true},
			map[string]interface{}{"name": "B", "active": false},
			map[string]interface{}{"name": "C", "active": true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering shown nested same-paragraph fragment: %v", err)
	}

	shownContent := extractTextFromDOCX(t, renderedShown)
	if !strings.Contains(shownContent, "FIRST A | OFF B | ON C") {
		t.Fatalf("expected shown nested same-paragraph fragment output, got %q", shownContent)
	}

	renderedHidden, err := tmpl.RenderToBytes(TemplateData{
		"show":  false,
		"items": []interface{}{},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering hidden nested same-paragraph fragment: %v", err)
	}

	hiddenContent := extractTextFromDOCX(t, renderedHidden)
	if !strings.Contains(hiddenContent, "Hidden") {
		t.Fatalf("expected hidden nested same-paragraph fragment output, got %q", hiddenContent)
	}
}

func TestDOCXIncludedFragmentWithBlockIfOpeningParagraphContainsNestedInlineControls(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "outer"}}`,
	})
	outerDoc := createDOCXWithParagraphs(t, []string{
		`{{if show}}{{unless suppress}}{{end}}{{for _ in empty}}{{end}}`,
		`Visible`,
		`{{include "inner"}}`,
		`{{else}}`,
		`Hidden`,
		`{{end}}`,
	})
	innerDoc := createFragmentDOCX(t, `Inner {{name}}`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", innerDoc); err != nil {
		t.Fatalf("failed to add inner fragment: %v", err)
	}

	renderedVisible, err := tmpl.RenderToBytes(TemplateData{
		"show":     true,
		"suppress": false,
		"empty":    []interface{}{},
		"name":     "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error rendering visible included fragment: %v", err)
	}

	visibleContent := extractTextFromDOCX(t, renderedVisible)
	if !strings.Contains(visibleContent, "Visible") || !strings.Contains(visibleContent, "Inner Alice") {
		t.Fatalf("expected visible branch with nested include, got %q", visibleContent)
	}
	if strings.Contains(visibleContent, "Hidden") {
		t.Fatalf("did not expect hidden branch in visible output, got %q", visibleContent)
	}

	renderedHidden, err := tmpl.RenderToBytes(TemplateData{
		"show":     false,
		"suppress": false,
		"empty":    []interface{}{},
		"name":     "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error rendering hidden included fragment: %v", err)
	}

	hiddenContent := extractTextFromDOCX(t, renderedHidden)
	if !strings.Contains(hiddenContent, "Hidden") {
		t.Fatalf("expected hidden branch in output, got %q", hiddenContent)
	}
	if strings.Contains(hiddenContent, "Visible Inner Alice") {
		t.Fatalf("did not expect visible branch in hidden output, got %q", hiddenContent)
	}
}

func TestDOCXIncludedFragmentWithBlockUnlessOpeningParagraphContainsNestedInlineControls(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "outer"}}`,
	})
	outerDoc := createDOCXWithParagraphs(t, []string{
		`{{unless hide}}{{if showHeader}}{{end}}{{for _ in empty}}{{end}}`,
		`Shown`,
		`{{else}}`,
		`{{include "fallback"}}`,
		`{{end}}`,
	})
	fallbackDoc := createFragmentDOCX(t, `Fallback {{name}}`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("fallback", fallbackDoc); err != nil {
		t.Fatalf("failed to add fallback fragment: %v", err)
	}

	renderedShown, err := tmpl.RenderToBytes(TemplateData{
		"hide":       false,
		"showHeader": true,
		"empty":      []interface{}{},
		"name":       "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error rendering shown unless branch: %v", err)
	}

	shownContent := extractTextFromDOCX(t, renderedShown)
	if !strings.Contains(shownContent, "Shown") {
		t.Fatalf("expected shown branch in output, got %q", shownContent)
	}
	if strings.Contains(shownContent, "Fallback Alice") {
		t.Fatalf("did not expect fallback branch in shown output, got %q", shownContent)
	}

	renderedFallback, err := tmpl.RenderToBytes(TemplateData{
		"hide":       true,
		"showHeader": true,
		"empty":      []interface{}{},
		"name":       "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error rendering fallback unless branch: %v", err)
	}

	fallbackContent := extractTextFromDOCX(t, renderedFallback)
	if !strings.Contains(fallbackContent, "Fallback Alice") {
		t.Fatalf("expected fallback branch with nested include, got %q", fallbackContent)
	}
	if strings.Contains(fallbackContent, "Shown") {
		t.Fatalf("did not expect shown branch in fallback output, got %q", fallbackContent)
	}
}

func TestDOCXIncludedFragmentChainWithBlockForOpeningParagraphContainsNestedInlineControls(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "outer"}}`,
	})
	outerDoc := createDOCXWithParagraphs(t, []string{
		`{{for item in items}}{{if showHeader}}{{end}}{{unless skipBody}}{{end}}{{for _ in empty}}{{end}}`,
		`{{include "row"}}`,
		`{{end}}`,
	})
	rowDoc := createDOCXWithParagraphs(t, []string{
		`{{if item.active}}Item: {{item.name}}{{else}}Inactive: {{item.name}}{{end}}`,
	})

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("row", rowDoc); err != nil {
		t.Fatalf("failed to add row fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{
		"items": []interface{}{
			map[string]interface{}{"name": "A", "active": true},
			map[string]interface{}{"name": "B", "active": false},
			map[string]interface{}{"name": "C", "active": true},
		},
		"showHeader": false,
		"skipBody":   false,
		"empty":      []interface{}{},
	})
	if err != nil {
		t.Fatalf("unexpected error rendering included fragment chain: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	for _, expected := range []string{"Item: A", "Inactive: B", "Item: C"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected %q in rendered output, got %q", expected, content)
		}
	}
}

func TestNestedDOCXFragmentsDepth3PreserveRunStyling(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "outer"}}`,
	})
	outerDoc := createDOCXWithBodyXML(t, `
    <w:p>
      <w:r>
        <w:rPr><w:b/><w:bCs/></w:rPr>
        <w:t>Outer Bold</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r><w:t>{{include "middle"}}</w:t></w:r>
    </w:p>`)
	middleDoc := createDOCXWithBodyXML(t, `
    <w:p>
      <w:r>
        <w:rPr><w:i/><w:iCs/></w:rPr>
        <w:t>Middle Italic</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r><w:t>{{include "inner"}}</w:t></w:r>
    </w:p>`)
	innerDoc := createDOCXWithBodyXML(t, `
    <w:p>
      <w:r>
        <w:rPr><w:b/><w:bCs/></w:rPr>
        <w:t>Inner {{value}}</w:t>
      </w:r>
    </w:p>`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("middle", middleDoc); err != nil {
		t.Fatalf("failed to add middle fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", innerDoc); err != nil {
		t.Fatalf("failed to add inner fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{"value": "Leaf"})
	if err != nil {
		t.Fatalf("failed to render nested styled fragments: %v", err)
	}

	content := extractTextFromDOCX(t, rendered)
	for _, expected := range []string{"Outer Bold", "Middle Italic", "Inner Leaf"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected %q in rendered content, got %q", expected, content)
		}
	}

	docXML := extractDocumentXMLFromDOCX(t, rendered)
	if !strings.Contains(docXML, "<w:b/>") {
		t.Fatalf("expected bold run formatting in output XML, got: %s", docXML)
	}
	if !strings.Contains(docXML, "<w:i/>") {
		t.Fatalf("expected italic run formatting in output XML, got: %s", docXML)
	}
}

func TestNestedDOCXFragmentStyleMergingDepth3(t *testing.T) {
	mainDoc := createDOCXWithStyle(t, `Main {{include "outer"}}`, "MainStyle", `<w:color w:val="111111"/>`)
	outerDoc := createDOCXWithStyle(t, `Outer {{include "middle"}}`, "OuterStyle", `<w:color w:val="0000FF"/>`)
	middleDoc := createDOCXWithStyle(t, `Middle {{include "inner"}}`, "MiddleStyle", `<w:i/>`)
	innerDoc := createDOCXWithStyle(t, `Inner`, "InnerStyle", `<w:b/>`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("middle", middleDoc); err != nil {
		t.Fatalf("failed to add middle fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", innerDoc); err != nil {
		t.Fatalf("failed to add inner fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{})
	if err != nil {
		t.Fatalf("failed to render nested styled fragments: %v", err)
	}

	styles := extractStylesFromDOCX(t, rendered)
	expectedStyles := map[string]bool{
		"MainStyle":   false,
		"OuterStyle":  false,
		"MiddleStyle": false,
		"InnerStyle":  false,
	}
	for _, style := range styles {
		if _, ok := expectedStyles[style]; ok {
			expectedStyles[style] = true
		}
	}
	for styleName, found := range expectedStyles {
		if !found {
			t.Fatalf("expected %s in merged styles, found styles=%v", styleName, styles)
		}
	}
}

func TestNestedTableFragmentsDepth3AtBodyLevel(t *testing.T) {
	mainDoc := createDOCXWithParagraphs(t, []string{
		`{{include "outer"}}`,
	})
	outerDoc := createDOCXWithBodyXML(t, `
    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Outer Table</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>
    <w:p><w:r><w:t>{{include "middle"}}</w:t></w:r></w:p>`)
	middleDoc := createDOCXWithBodyXML(t, `
    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Middle Table</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>
    <w:p><w:r><w:t>{{include "inner"}}</w:t></w:r></w:p>`)
	innerDoc := createDOCXWithBodyXML(t, `
    <w:tbl>
      <w:tr>
        <w:tc><w:p><w:r><w:t>Inner Table {{value}}</w:t></w:r></w:p></w:tc>
      </w:tr>
    </w:tbl>`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("middle", middleDoc); err != nil {
		t.Fatalf("failed to add middle fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", innerDoc); err != nil {
		t.Fatalf("failed to add inner fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{"value": "Leaf"})
	if err != nil {
		t.Fatalf("failed to render nested table fragments: %v", err)
	}

	docXML := extractDocumentXMLFromDOCX(t, rendered)
	if count := strings.Count(docXML, "<w:tbl>"); count != 1 {
		t.Fatalf("expected nested fragment tables to be merged into 1 table, got %d in %s", count, docXML)
	}
	if rowCount := strings.Count(docXML, "<w:tr>"); rowCount != 3 {
		t.Fatalf("expected merged table to contain 3 rows, got %d in %s", rowCount, docXML)
	}
	for _, expected := range []string{"Outer Table", "Middle Table", "Inner Table Leaf"} {
		if !strings.Contains(docXML, expected) {
			t.Fatalf("expected %q in rendered document XML, got %s", expected, docXML)
		}
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
		"MainStyle":  false,
		"Frag1Style": false,
		"Frag2Style": false,
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

func createDOCXWithParagraphs(_ *testing.T, paragraphs []string) []byte {
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

	// Add word/document.xml with paragraph list
	doc, _ := w.Create("word/document.xml")
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>`)
	for _, p := range paragraphs {
		io.WriteString(doc, `
    <w:p>
      <w:r>
        <w:t>`+p+`</w:t>
      </w:r>
    </w:p>`)
	}
	io.WriteString(doc, `
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

func createDOCXWithBodyXML(t *testing.T, bodyXML string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	rels, err := w.Create("_rels/.rels")
	if err != nil {
		t.Fatalf("failed to create _rels/.rels: %v", err)
	}
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	wordRels, err := w.Create("word/_rels/document.xml.rels")
	if err != nil {
		t.Fatalf("failed to create word/_rels/document.xml.rels: %v", err)
	}
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`)

	doc, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatalf("failed to create word/document.xml: %v", err)
	}
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>`+bodyXML+`
  </w:body>
</w:document>`)

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

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

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

func extractDocumentXMLFromDOCX(t *testing.T, docxBytes []byte) string {
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

			return string(content)
		}
	}

	t.Fatal("word/document.xml not found in docx")
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
