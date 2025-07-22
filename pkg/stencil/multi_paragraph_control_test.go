package stencil

import (
	"strings"
	"testing"
)

func TestMultiParagraphControlStructure(t *testing.T) {
	// Test control structures that span multiple paragraphs
	testCases := []struct {
		name           string
		body           *Body
		data           TemplateData
		expectedParas  int
		checkContent   func(t *testing.T, body *Body)
	}{
		{
			name: "if statement with content in separate paragraph",
			body: &Body{
				Elements: []BodyElement{
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "Combined conditions:  "}},
							{Break: &Break{}},
							{Text: &Text{Content: "{{if (age >= 18 & hasID) | isVIP}}"}},
						},
					},
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "Welcome to the exclusive area!"}},
						},
					},
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "{{end}}"}},
						},
					},
				},
			},
			data: TemplateData{
				"age":   21,
				"hasID": true,
				"isVIP": false,
			},
			expectedParas: 1, // Should combine into one paragraph but preserve internal structure
			checkContent: func(t *testing.T, body *Body) {
				if len(body.Elements) < 1 {
					t.Fatal("No paragraphs in output")
				}
				
				para, ok := body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatal("First element is not a paragraph")
				}
				
				// Extract text with line breaks
				var text strings.Builder
				for i, run := range para.Runs {
					if run.Text != nil {
						text.WriteString(run.Text.Content)
						t.Logf("Run %d: Text = %q", i, run.Text.Content)
					}
					if run.Break != nil {
						text.WriteString("\n")
						t.Logf("Run %d: Break", i)
					}
				}
				
				fullText := text.String()
				t.Logf("Full rendered text: %q", fullText)
				
				// Check that we have the expected structure:
				// "Combined conditions:  \nWelcome to the exclusive area!"
				if !strings.Contains(fullText, "Combined conditions:  \n") {
					t.Error("Line break after 'Combined conditions:' was not preserved")
				}
				
				if !strings.Contains(fullText, "Welcome to the exclusive area!") {
					t.Error("Content 'Welcome to the exclusive area!' was not rendered")
				}
			},
		},
		{
			name: "if statement with multiple paragraphs inside",
			body: &Body{
				Elements: []BodyElement{
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "Start:"}},
						},
					},
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "{{if showDetails}}"}},
						},
					},
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "First paragraph of details."}},
						},
					},
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "Second paragraph of details."}},
						},
					},
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "{{end}}"}},
						},
					},
					&Paragraph{
						Runs: []Run{
							{Text: &Text{Content: "End."}},
						},
					},
				},
			},
			data: TemplateData{
				"showDetails": true,
			},
			expectedParas: 4, // Start, First para, Second para, End
			checkContent: func(t *testing.T, body *Body) {
				if len(body.Elements) != 4 {
					t.Errorf("Expected 4 paragraphs, got %d", len(body.Elements))
					for i, elem := range body.Elements {
						if para, ok := elem.(*Paragraph); ok {
							t.Logf("Paragraph %d: %s", i, getParaText(para))
						}
					}
				}
				
				// Check content of each paragraph
				expectedTexts := []string{
					"Start:",
					"First paragraph of details.",
					"Second paragraph of details.",
					"End.",
				}
				
				for i, expected := range expectedTexts {
					if i >= len(body.Elements) {
						break
					}
					if para, ok := body.Elements[i].(*Paragraph); ok {
						text := getParaText(para)
						if text != expected {
							t.Errorf("Paragraph %d: expected %q, got %q", i, expected, text)
						}
					}
				}
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Render the body
			t.Logf("Rendering body with %d elements", len(tc.body.Elements))
			rendered, err := RenderBody(tc.body, tc.data)
			if err != nil {
				t.Fatalf("Failed to render body: %v", err)
			}
			t.Logf("Rendered body has %d elements", len(rendered.Elements))
			
			// Check the result
			tc.checkContent(t, rendered)
		})
	}
}

func TestSingleParagraphIfWithLineBreak(t *testing.T) {
	// Test the case where everything is in a single paragraph with line breaks
	para := &Paragraph{
		Properties: &ParagraphProperties{},
		Runs: []Run{
			{Text: &Text{Content: "Combined conditions:  "}},
			{Break: &Break{}},
			{Text: &Text{Content: "{{if (age >= 18 & hasID) | isVIP}}Welcome to the exclusive area!{{end}}"}},
		},
	}
	
	// Merge runs
	mergeConsecutiveRuns(para)
	
	data := TemplateData{
		"age":   21,
		"hasID": true,
		"isVIP": false,
	}
	
	// Render
	rendered, err := RenderParagraph(para, data)
	if err != nil {
		t.Fatalf("Failed to render paragraph: %v", err)
	}
	
	// Extract text
	var text strings.Builder
	for i, run := range rendered.Runs {
		if run.Text != nil {
			text.WriteString(run.Text.Content)
			t.Logf("Run %d: Text = %q", i, run.Text.Content)
		}
		if run.Break != nil {
			text.WriteString("\n")
			t.Logf("Run %d: Break", i)
		}
	}
	
	fullText := text.String()
	t.Logf("Full text: %q", fullText)
	
	// This should work correctly based on our previous fix
	expected := "Combined conditions:  \nWelcome to the exclusive area!"
	if fullText != expected {
		t.Errorf("Expected %q, got %q", expected, fullText)
	}
}

// Helper function to get paragraph text for testing
func getParaText(para *Paragraph) string {
	var text strings.Builder
	for _, run := range para.Runs {
		if run.Text != nil {
			text.WriteString(run.Text.Content)
		}
	}
	return text.String()
}