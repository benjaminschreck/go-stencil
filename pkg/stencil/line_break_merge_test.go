package stencil

import (
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

func TestRenderParagraphWithLineBreaksBetweenVariables(t *testing.T) {
	// This test reproduces the exact structure from comprehensive_features.docx paragraph 93
	// where {{isVIP}} and {{discount}} appear to be merged
	para := &Paragraph{
		Properties: &ParagraphProperties{},
		Runs: []Run{
			// {{age }}
			{Properties: &RunProperties{}, Text: &Text{Content: "{{"}},
			{Properties: &RunProperties{}, Text: &Text{Content: "age"}},
			{Properties: &RunProperties{}, Text: &Text{Content: " "}},
			{Properties: &RunProperties{}, Text: &Text{Content: "}}"}},
			{Properties: &RunProperties{}, Break: &Break{}},
			// {{hasID }}
			{Properties: &RunProperties{}, Text: &Text{Content: "{{"}},
			{Properties: &RunProperties{}, Text: &Text{Content: "hasID"}},
			{Properties: &RunProperties{}, Text: &Text{Content: " "}},
			{Properties: &RunProperties{}, Text: &Text{Content: "}}"}},
			{Properties: &RunProperties{}, Break: &Break{}},
			// {{isVIP}}
			{Properties: &RunProperties{}, Text: &Text{Content: "{{"}},
			{Properties: &RunProperties{}, Text: &Text{Content: "isVIP"}},
			{Properties: &RunProperties{}, Text: &Text{Content: "}}"}},
			// {{discount}} - note this is in a single run with a break before it
			{Properties: &RunProperties{}, Break: &Break{}, Text: &Text{Content: "{{discount}}"}},
		},
	}

	// First merge consecutive runs to handle split template variables
	render.MergeConsecutiveRuns(para)

	// Check merged result
	t.Logf("After merging, paragraph has %d runs", len(para.Runs))
	for i, run := range para.Runs {
		if run.Text != nil {
			t.Logf("Run %d: Text = %q", i, run.Text.Content)
		}
		if run.Break != nil {
			t.Logf("Run %d: Break", i)
		}
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

	// Log the rendered output
	t.Logf("\nRendered paragraph has %d runs:", len(rendered.Runs))
	for i, run := range rendered.Runs {
		if run.Text != nil {
			t.Logf("Run %d: Text = %q", i, run.Text.Content)
		}
		if run.Break != nil {
			t.Logf("Run %d: Break", i)
		}
	}

	// Check that we have the correct structure
	// We should have: 21, break, true, break, false, break, 15
	expectedValues := []string{"21", "true", "false", "15"}
	valueIndex := 0
	
	for _, run := range rendered.Runs {
		if run.Text != nil && valueIndex < len(expectedValues) {
			if run.Text.Content != expectedValues[valueIndex] {
				// Check if this run contains multiple values concatenated
				if valueIndex == 2 && run.Text.Content == "false15" {
					t.Errorf("Values 'false' and '15' were concatenated into a single run")
				} else {
					t.Errorf("Expected value %q, got %q", expectedValues[valueIndex], run.Text.Content)
				}
			}
			valueIndex++
		}
	}
}

func TestRunWithBothBreakAndText(t *testing.T) {
	// Test the specific case where a run has both a break and text
	// This seems to be the structure in the original template
	run := Run{
		Properties: &RunProperties{},
		Break:      &Break{},
		Text:       &Text{Content: "{{discount}}"},
	}

	data := TemplateData{
		"discount": 15,
	}

	// Render the run
	rendered, err := RenderRun(&run, data)
	if err != nil {
		t.Fatalf("Failed to render run: %v", err)
	}

	// Check the result
	t.Logf("Rendered run - Break: %v, Text: %v", rendered.Break != nil, rendered.Text != nil)
	if rendered.Text != nil {
		t.Logf("Text content: %q", rendered.Text.Content)
	}

	// A run with both break and text should maintain both after rendering
	if rendered.Break == nil {
		t.Error("Expected break to be preserved")
	}
	if rendered.Text == nil || rendered.Text.Content != "15" {
		t.Errorf("Expected text to be '15', got %v", rendered.Text)
	}
}