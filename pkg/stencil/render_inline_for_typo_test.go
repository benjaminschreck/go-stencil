package stencil

import (
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

// TestRenderInlineForLoopWithTypo tests what happens when there's a typo in the variable name
func TestRenderInlineForLoopWithTypo(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					// Note the typo: "comany" instead of "company"
					Content: "{{for i, company in companies}}{{if i > 0}}, {{end}}{{comany.name}}{{end}}{{if length(companies) == 0}}{{hideRow()}}{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"companies": []interface{}{
			map[string]interface{}{"name": "CompanyA"},
			map[string]interface{}{"name": "CompanyB"},
			map[string]interface{}{"name": "CompanyC"},
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

	actual := resultText.String()

	// With the typo, comany.name will evaluate to empty string for each iteration
	// So we expect: ", , " (comma-space for each item except the first)
	t.Logf("Result with typo: %q", actual)

	// The expected result depends on how we handle missing variables
	// Currently, missing variables return empty string
}

// TestRenderInlineForLoopCorrect tests the corrected version
func TestRenderInlineForLoopCorrect(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for i, company in companies}}{{if i > 0}}, {{end}}{{company.name}}{{end}}{{if length(companies) == 0}}{{hideRow()}}{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"companies": []interface{}{
			map[string]interface{}{"name": "CompanyA"},
			map[string]interface{}{"name": "CompanyB"},
			map[string]interface{}{"name": "CompanyC"},
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

	expected := "CompanyA, CompanyB, CompanyC"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestHideRowFunction tests the hideRow() function behavior
func TestHideRowFunctionInSuffix(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for i, company in companies}}{{if i > 0}}, {{end}}{{company.name}}{{end}}{{if length(companies) == 0}}{{hideRow()}}{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	// Empty companies
	data := TemplateData{
		"companies": []interface{}{},
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

	actual := resultText.String()
	t.Logf("Result with empty companies: %q", actual)

	// hideRow() should return a special marker that gets handled at table level
	// The exact format depends on the hideRow implementation
}
