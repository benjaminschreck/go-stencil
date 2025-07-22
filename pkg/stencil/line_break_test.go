package stencil

import (
	"strings"
	"testing"
)

func TestRenderParagraphWithLineBreaks(t *testing.T) {
	// Create a paragraph with template variables separated by line breaks
	// This mimics the structure found in comprehensive_features.docx
	para := &Paragraph{
		Properties: &ParagraphProperties{},
		Runs: []Run{
			// {{age }}
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "{{age }}"},
			},
			// Line break
			{
				Properties: &RunProperties{},
				Break:      &Break{},
			},
			// {{hasID }}
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "{{hasID }}"},
			},
			// Line break
			{
				Properties: &RunProperties{},
				Break:      &Break{},
			},
			// {{isVIP}}
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "{{isVIP}}"},
			},
			// Line break
			{
				Properties: &RunProperties{},
				Break:      &Break{},
			},
			// {{discount}}
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "{{discount}}"},
			},
		},
	}

	// Test data
	data := TemplateData{
		"age":      21,
		"hasID":    true,
		"isVIP":    false,
		"discount": 15,
	}

	// Render the paragraph
	rendered, err := RenderParagraph(para, data)
	if err != nil {
		t.Fatalf("Failed to render paragraph: %v", err)
	}

	// Check that we have the expected number of runs
	// We should have: text, break, text, break, text, break, text = 7 runs
	if len(rendered.Runs) != 7 {
		t.Errorf("Expected 7 runs, got %d", len(rendered.Runs))
		for i, run := range rendered.Runs {
			if run.Text != nil {
				t.Logf("Run %d: Text = %q", i, run.Text.Content)
			} else if run.Break != nil {
				t.Logf("Run %d: Break", i)
			}
		}
	}

	// Check the content of each run
	expectedTexts := []string{"21", "true", "false", "15"}
	textIndex := 0
	
	for i, run := range rendered.Runs {
		if i%2 == 0 { // Even indices should be text
			if run.Text == nil {
				t.Errorf("Run %d: expected text, got %+v", i, run)
			} else if textIndex < len(expectedTexts) && run.Text.Content != expectedTexts[textIndex] {
				t.Errorf("Run %d: expected %q, got %q", i, expectedTexts[textIndex], run.Text.Content)
			}
			textIndex++
		} else { // Odd indices should be breaks
			if run.Break == nil {
				t.Errorf("Run %d: expected break, got %+v", i, run)
			}
		}
	}

	// Also test that the text output contains all values
	var allText strings.Builder
	for _, run := range rendered.Runs {
		if run.Text != nil {
			allText.WriteString(run.Text.Content)
		}
	}
	
	combined := allText.String()
	for _, expected := range expectedTexts {
		if !strings.Contains(combined, expected) {
			t.Errorf("Output missing expected value %q", expected)
		}
	}
}

func TestRenderParagraphWithMergedRuns(t *testing.T) {
	// Test the case where template variables might be split across runs
	// This tests the mergeConsecutiveRuns functionality
	para := &Paragraph{
		Properties: &ParagraphProperties{},
		Runs: []Run{
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "{{"},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "age"},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: " "},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "}}"},
			},
			{
				Properties: &RunProperties{},
				Break:      &Break{},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "{{"},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "hasID"},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: " "},
			},
			{
				Properties: &RunProperties{},
				Text: &Text{Content: "}}"},
			},
		},
	}

	// First merge consecutive runs
	mergeConsecutiveRuns(para)

	// Test data
	data := TemplateData{
		"age":   21,
		"hasID": true,
	}

	// Render the paragraph
	rendered, err := RenderParagraph(para, data)
	if err != nil {
		t.Fatalf("Failed to render paragraph: %v", err)
	}

	// We should have: text("21"), break, text("true")
	if len(rendered.Runs) < 3 {
		t.Errorf("Expected at least 3 runs, got %d", len(rendered.Runs))
		for i, run := range rendered.Runs {
			if run.Text != nil {
				t.Logf("Run %d: Text = %q", i, run.Text.Content)
			} else if run.Break != nil {
				t.Logf("Run %d: Break", i)
			}
		}
	}

	// Check content
	hasBreak := false
	for _, run := range rendered.Runs {
		if run.Break != nil {
			hasBreak = true
		}
	}
	
	if !hasBreak {
		t.Error("Expected at least one line break in output")
	}
}