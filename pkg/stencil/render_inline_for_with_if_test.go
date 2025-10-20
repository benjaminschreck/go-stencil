package stencil

import (
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

// TestRenderInlineForLoopWithIf tests the specific bug reported by the user:
// {{for i, sale in salesData}}{{if i > 0}}, {{end}}{{sale.region}}{{end}}
func TestRenderInlineForLoopWithIf(t *testing.T) {
	// Create a paragraph with an inline for loop containing an if statement
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for i, sale in salesData}}{{if i > 0}}, {{end}}{{sale.region}}{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	// Test data
	data := TemplateData{
		"salesData": []interface{}{
			map[string]interface{}{"region": "North"},
			map[string]interface{}{"region": "South"},
			map[string]interface{}{"region": "East"},
		},
	}

	// Merge consecutive runs (simulates what happens in real rendering)
	render.MergeConsecutiveRuns(para)

	// Check if this is an inline for loop
	controlType, controlContent := detectControlStructure(para)

	if controlType != "inline-for" {
		t.Fatalf("Expected controlType='inline-for', got %q", controlType)
	}

	// Render the inline for loop
	ctx := &renderContext{}
	renderedParas, err := renderInlineForLoop(para, controlContent, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render inline for loop: %v", err)
	}

	if len(renderedParas) != 1 {
		t.Fatalf("Expected 1 rendered paragraph, got %d", len(renderedParas))
	}

	// Extract the text from the rendered paragraph
	var resultText strings.Builder
	for _, run := range renderedParas[0].Runs {
		if run.Text != nil {
			resultText.WriteString(run.Text.Content)
		}
	}

	expected := "North, South, East"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestRenderInlineForLoopWithUnless tests unless in inline for loops
func TestRenderInlineForLoopWithUnless(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for i, item in items}}{{unless i == 0}} | {{end}}{{item}}{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"items": []interface{}{"A", "B", "C"},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := detectControlStructure(para)

	if controlType != "inline-for" {
		t.Fatalf("Expected controlType='inline-for', got %q", controlType)
	}

	ctx := &renderContext{}
	renderedParas, err := renderInlineForLoop(para, controlContent, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render inline for loop: %v", err)
	}

	var resultText strings.Builder
	for _, run := range renderedParas[0].Runs {
		if run.Text != nil {
			resultText.WriteString(run.Text.Content)
		}
	}

	expected := "A | B | C"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestRenderInlineForLoopWithComplexIf tests more complex if conditions
func TestRenderInlineForLoopWithComplexIf(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for i, sale in salesData}}{{if i > 0 & sale.active}}, {{end}}{{sale.name}}{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"salesData": []interface{}{
			map[string]interface{}{"name": "First", "active": true},
			map[string]interface{}{"name": "Second", "active": true},
			map[string]interface{}{"name": "Third", "active": false},
			map[string]interface{}{"name": "Fourth", "active": true},
		},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := detectControlStructure(para)

	if controlType != "inline-for" {
		t.Fatalf("Expected controlType='inline-for', got %q", controlType)
	}

	ctx := &renderContext{}
	renderedParas, err := renderInlineForLoop(para, controlContent, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render inline for loop: %v", err)
	}

	var resultText strings.Builder
	for _, run := range renderedParas[0].Runs {
		if run.Text != nil {
			resultText.WriteString(run.Text.Content)
		}
	}

	expected := "First, SecondThird, Fourth"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestRenderInlineForLoopWithSuffix tests that suffix is also processed
func TestRenderInlineForLoopWithSuffix(t *testing.T) {
	// Test the exact pattern from the user's issue
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for i, sale in salesData}}{{if i > 0}}, {{end}}{{sale.region}}{{end}}{{if length(salesData) == 0}}{{hideRow()}}{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	// Test with non-empty data (hideRow should not be called)
	data := TemplateData{
		"salesData": []interface{}{
			map[string]interface{}{"region": "North"},
			map[string]interface{}{"region": "South"},
			map[string]interface{}{"region": "East"},
			map[string]interface{}{"region": "West"},
		},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := detectControlStructure(para)

	if controlType != "inline-for" {
		t.Fatalf("Expected controlType='inline-for', got %q", controlType)
	}

	ctx := &renderContext{}
	renderedParas, err := renderInlineForLoop(para, controlContent, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render inline for loop: %v", err)
	}

	var resultText strings.Builder
	for _, run := range renderedParas[0].Runs {
		if run.Text != nil {
			resultText.WriteString(run.Text.Content)
		}
	}

	// The suffix {{if length(salesData) == 0}}{{hideRow()}}{{end}} should be processed
	// Since salesData is not empty, the if condition is false, so hideRow() is not called
	expected := "North, South, East, West"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestRenderInlineForLoopWithEmptyCollection tests hideRow with empty collection
func TestRenderInlineForLoopWithEmptyCollection(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for i, sale in salesData}}{{if i > 0}}, {{end}}{{sale.region}}{{end}}{{if length(salesData) == 0}}EMPTY{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	// Test with empty data
	data := TemplateData{
		"salesData": []interface{}{},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := detectControlStructure(para)

	if controlType != "inline-for" {
		t.Fatalf("Expected controlType='inline-for', got %q", controlType)
	}

	ctx := &renderContext{}
	renderedParas, err := renderInlineForLoop(para, controlContent, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render inline for loop: %v", err)
	}

	var resultText strings.Builder
	for _, run := range renderedParas[0].Runs {
		if run.Text != nil {
			resultText.WriteString(run.Text.Content)
		}
	}

	// Since salesData is empty, the for loop produces nothing, but the if in suffix should show "EMPTY"
	expected := "EMPTY"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}
