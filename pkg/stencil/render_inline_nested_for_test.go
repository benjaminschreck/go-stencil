package stencil

import (
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

// TestRenderInlineForLoopNested tests nested for loops (for inside for)
func TestRenderInlineForLoopNested(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for group in groups}}[{{for item in group.items}}{{item}}{{end}}]{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"groups": []interface{}{
			map[string]interface{}{
				"items": []interface{}{"a", "b"},
			},
			map[string]interface{}{
				"items": []interface{}{"c", "d"},
			},
		},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := render.DetectControlStructure(para)

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

	expected := "[ab][cd]"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestRenderInlineForLoopTripleNested tests 3-level nested for loops
// This mirrors the user's Klagerubrum template structure
func TestRenderInlineForLoopTripleNested(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for partei in parteien}}{{partei.name}}{{for v1 in partei.vertreter}}, vertreten durch {{v1.name}}{{for v2 in v1.vertreter}}, wiederum vertreten durch {{v2.name}}{{end}}{{end}}; {{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"parteien": []interface{}{
			map[string]interface{}{
				"name": "Alice",
				"vertreter": []interface{}{
					map[string]interface{}{
						"name": "Bob",
						"vertreter": []interface{}{
							map[string]interface{}{"name": "Charlie"},
						},
					},
				},
			},
			map[string]interface{}{
				"name": "Dave",
				"vertreter": []interface{}{
					map[string]interface{}{
						"name":      "Eve",
						"vertreter": []interface{}{},
					},
				},
			},
		},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := render.DetectControlStructure(para)

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

	expected := "Alice, vertreten durch Bob, wiederum vertreten durch Charlie; Dave, vertreten durch Eve; "
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestRenderInlineForLoopNestedWithIf tests for loop nested inside if inside for
func TestRenderInlineForLoopNestedForInsideIf(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for partei in parteien}}{{partei.name}}{{if partei.hasVertreter}}{{for v in partei.vertreter}} ({{v.name}}){{end}}{{end}}; {{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"parteien": []interface{}{
			map[string]interface{}{
				"name":          "Alice",
				"hasVertreter":  true,
				"vertreter": []interface{}{
					map[string]interface{}{"name": "Bob"},
					map[string]interface{}{"name": "Charlie"},
				},
			},
			map[string]interface{}{
				"name":          "Dave",
				"hasVertreter":  false,
				"vertreter": []interface{}{},
			},
		},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := render.DetectControlStructure(para)

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

	expected := "Alice (Bob) (Charlie); Dave; "
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestRenderInlineForLoopNestedWithIndex tests nested for with index variables
func TestRenderInlineForLoopNestedWithIndex(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for i, group in groups}}{{if i > 0}} | {{end}}{{for j, item in group.items}}{{if j > 0}},{{end}}{{item}}{{end}}{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"groups": []interface{}{
			map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
			map[string]interface{}{
				"items": []interface{}{"x", "y"},
			},
		},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := render.DetectControlStructure(para)

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

	expected := "a,b,c | x,y"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}

// TestRenderInlineForLoopNestedEmptyInner tests nested for with empty inner collection
func TestRenderInlineForLoopNestedEmptyInner(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Text: &Text{
					Content: "{{for group in groups}}[{{for item in group.items}}{{item}}{{end}}]{{end}}",
					Space:   "preserve",
				},
			},
		},
	}

	data := TemplateData{
		"groups": []interface{}{
			map[string]interface{}{
				"items": []interface{}{"a", "b"},
			},
			map[string]interface{}{
				"items": []interface{}{},
			},
			map[string]interface{}{
				"items": []interface{}{"z"},
			},
		},
	}

	render.MergeConsecutiveRuns(para)
	controlType, controlContent := render.DetectControlStructure(para)

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

	expected := "[ab][][z]"
	actual := resultText.String()

	if actual != expected {
		t.Errorf("Expected %q, got %q", expected, actual)
	}
}
