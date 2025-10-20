package stencil

import (
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

// TestLineBreakBeforeControlStructure is removed because RenderText doesn't process control structures
// The actual test is TestLineBreakInParagraphWithControlStructure which tests the real scenario

func TestLineBreakInParagraphWithControlStructure(t *testing.T) {
	// Test paragraph rendering with line breaks before control structures
	para := &Paragraph{
		Properties: &ParagraphProperties{},
		Runs: []Run{
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "Combined conditions:  "},
			},
			{
				Properties: &RunProperties{},
				Break:      &Break{},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "{{if (age >= 18 & hasID) | isVIP}}"},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "Welcome to the exclusive area!"},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "{{end}}"},
			},
		},
	}

	// Merge runs
	render.MergeConsecutiveRuns(para)

	data := TemplateData{
		"age":   21,
		"hasID": true,
		"isVIP": false,
	}

	// Render the paragraph
	rendered, err := RenderParagraph(para, data)
	if err != nil {
		t.Fatalf("Failed to render paragraph: %v", err)
	}

	// Log the merged paragraph structure before rendering
	t.Logf("Merged paragraph has %d runs:", len(para.Runs))
	fullText := ""
	for i, run := range para.Runs {
		if run.Text != nil {
			t.Logf("  Run %d: Text = %q", i, run.Text.Content)
			fullText += run.Text.Content
		}
		if run.Break != nil {
			t.Logf("  Run %d: Break", i)
			fullText += "\n"
		}
	}
	t.Logf("Full text for rendering: %q", fullText)
	
	// Check that we have the expected structure
	t.Logf("\nRendered paragraph has %d runs:", len(rendered.Runs))
	for i, run := range rendered.Runs {
		if run.Text != nil {
			t.Logf("Run %d: Text = %q", i, run.Text.Content)
		}
		if run.Break != nil {
			t.Logf("Run %d: Break", i)
		}
	}

	// Extract all text
	var allText strings.Builder
	for _, run := range rendered.Runs {
		if run.Text != nil {
			allText.WriteString(run.Text.Content)
		}
		if run.Break != nil {
			allText.WriteString("\n")
		}
	}

	result := allText.String()
	expected := "Combined conditions:  \nWelcome to the exclusive area!"
	
	if result != expected {
		t.Errorf("Expected text with line break preserved:\nExpected: %q\nGot:      %q", expected, result)
	}
}

func TestControlStructureTextRendering(t *testing.T) {
	// Test the actual rendering of control structures in text
	testCases := []struct {
		name     string
		input    string
		data     TemplateData
		expected string
	}{
		{
			name:     "simple if with preceding newline",
			input:    "Line 1\n{{if true}}Line 2{{end}}",
			data:     TemplateData{},
			expected: "Line 1\nLine 2",
		},
		{
			name: "if with condition and preceding newline",
			input: "Combined conditions:\n{{if isVIP}}Welcome!{{end}}",
			data: TemplateData{"isVIP": true},
			expected: "Combined conditions:\nWelcome!",
		},
		{
			name: "whitespace before newline preserved",
			input: "Combined conditions:  \n{{if isVIP}}Welcome!{{end}}",
			data: TemplateData{"isVIP": true},
			expected: "Combined conditions:  \nWelcome!",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse control structures
			structures, err := ParseControlStructures(tc.input)
			if err != nil {
				t.Fatalf("Failed to parse control structures: %v", err)
			}

			// Render the structures
			result, err := renderControlBody(structures, tc.data)
			if err != nil {
				t.Fatalf("Failed to render control body: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Incorrect rendering:\nExpected: %q\nGot:      %q", tc.expected, result)
				// Log the parsed structures for debugging
				t.Logf("Parsed %d structures", len(structures))
				for i, s := range structures {
					switch v := s.(type) {
					case *TextNode:
						t.Logf("Structure %d: TextNode, Content=%q", i, v.Content)
					case *IfNode:
						t.Logf("Structure %d: IfNode, Condition=%q", i, v.Condition)
					case *ForNode:
						t.Logf("Structure %d: ForNode", i)
					}
				}
			}
		})
	}
}