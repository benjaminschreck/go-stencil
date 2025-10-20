package stencil

import (
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

func TestComprehensiveFeaturesControlStructure(t *testing.T) {
	// This test reproduces the exact structure from comprehensive_features.docx paragraph 115
	// where "Combined conditions:" has a line break before {{if (age >= 18 & hasID) | isVIP}}
	para := &Paragraph{
		Properties: &ParagraphProperties{},
		Runs: []Run{
			{Properties: &RunProperties{}, Text: &Text{Content: "  "}},
			{Properties: &RunProperties{}, Text: &Text{Content: "Combined"}},
			{Properties: &RunProperties{}, Text: &Text{Content: " "}},
			{Properties: &RunProperties{}, Text: &Text{Content: "conditions"}},
			{Properties: &RunProperties{}, Text: &Text{Content: ": "}},
			{Properties: &RunProperties{}, Text: &Text{Content: " "}},
			{Properties: &RunProperties{}, Break: &Break{}},
			{Properties: &RunProperties{}, Text: &Text{Content: "{{"}},
			{Properties: &RunProperties{}, Text: &Text{Content: "if"}},
			{Properties: &RunProperties{}, Text: &Text{Content: " ("}},
			{Properties: &RunProperties{}, Text: &Text{Content: "age"}},
			{Properties: &RunProperties{}, Text: &Text{Content: " >= 18 & "}},
			{Properties: &RunProperties{}, Text: &Text{Content: "hasID"}},
			{Properties: &RunProperties{}, Text: &Text{Content: ") | "}},
			{Properties: &RunProperties{}, Text: &Text{Content: "isVIP"}},
			{Properties: &RunProperties{}, Text: &Text{Content: "}}"}},
		},
	}

	// Add the content within the if statement and the end tag
	// In separate paragraphs as they appear in the template
	para2 := &Paragraph{
		Properties: &ParagraphProperties{},
		Runs: []Run{
			{Properties: &RunProperties{}, Text: &Text{Content: "Welcome to the exclusive area!"}},
		},
	}

	para3 := &Paragraph{
		Properties: &ParagraphProperties{},
		Runs: []Run{
			{Properties: &RunProperties{}, Text: &Text{Content: "{{end}}"}},
		},
	}

	// First merge consecutive runs
	render.MergeConsecutiveRuns(para)
	render.MergeConsecutiveRuns(para2)
	render.MergeConsecutiveRuns(para3)

	// Test data
	data := TemplateData{
		"age":   21,
		"hasID": true,
		"isVIP": false,
	}

	// Render each paragraph
	rendered1, err := RenderParagraph(para, data)
	if err != nil {
		t.Fatalf("Failed to render paragraph 1: %v", err)
	}

	_, err = RenderParagraph(para2, data)
	if err != nil {
		t.Fatalf("Failed to render paragraph 2: %v", err)
	}

	_, err = RenderParagraph(para3, data)
	if err != nil {
		t.Fatalf("Failed to render paragraph 3: %v", err)
	}

	// Check the first paragraph structure
	t.Logf("First paragraph has %d runs:", len(rendered1.Runs))
	hasBreak := false
	for i, run := range rendered1.Runs {
		if run.Text != nil {
			t.Logf("  Run %d: Text = %q", i, run.Text.Content)
		}
		if run.Break != nil {
			t.Logf("  Run %d: Break", i)
			hasBreak = true
		}
	}

	if !hasBreak {
		t.Error("Expected line break to be preserved in first paragraph")
	}

	// Extract text from first paragraph
	var text1 strings.Builder
	for _, run := range rendered1.Runs {
		if run.Text != nil {
			text1.WriteString(run.Text.Content)
		}
		if run.Break != nil {
			text1.WriteString("\n")
		}
	}

	// Check that the line break is preserved
	if !strings.Contains(text1.String(), "\n") {
		t.Errorf("Line break not preserved. Got: %q", text1.String())
	}

	// Check specific structure
	expectedPrefix := "  Combined conditions:  \n"
	if !strings.HasPrefix(text1.String(), expectedPrefix) {
		t.Errorf("Expected text to start with %q, got %q", expectedPrefix, text1.String())
	}
}

func TestControlStructureAcrossParagraphs(t *testing.T) {
	// Test how control structures that span multiple paragraphs are handled
	// This is important for understanding the comprehensive_features.docx behavior
	
	// Create a body with control structure spanning paragraphs
	body := &Body{
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
	}

	data := TemplateData{
		"age":   21,
		"hasID": true,
		"isVIP": false,
	}

	// Render the body
	rendered, err := RenderBody(body, data)
	if err != nil {
		t.Fatalf("Failed to render body: %v", err)
	}

	// Check the structure
	t.Logf("Rendered body has %d elements", len(rendered.Elements))
	
	// We should have one paragraph with the combined content
	if len(rendered.Elements) != 1 {
		t.Errorf("Expected 1 paragraph after rendering control structure, got %d", len(rendered.Elements))
	}

	// Check the first paragraph
	if para, ok := rendered.Elements[0].(*Paragraph); ok {
		t.Logf("Paragraph has %d runs:", len(para.Runs))
		
		// Extract text with breaks
		var text strings.Builder
		for i, run := range para.Runs {
			if run.Text != nil {
				t.Logf("  Run %d: Text = %q", i, run.Text.Content)
				text.WriteString(run.Text.Content)
			}
			if run.Break != nil {
				t.Logf("  Run %d: Break", i)
				text.WriteString("\n")
			}
		}

		fullText := text.String()
		t.Logf("Full text: %q", fullText)

		// Check that line break is preserved
		if !strings.Contains(fullText, "Combined conditions:  \n") {
			t.Error("Line break after 'Combined conditions:' was not preserved")
		}
	}
}