package stencil

import (
	"strings"
	"testing"
)

func TestRenderTableCellWithSplitRuns(t *testing.T) {
	tests := []struct {
		name     string
		cell     *TableCell
		data     TemplateData
		expected string
	}{
		{
			name: "html function split across multiple runs",
			cell: &TableCell{
				Paragraphs: []Paragraph{
					{
						Runs: []Run{
							{Text: &Text{Content: "{{"}},
							{Text: &Text{Content: "html"}},
							{Text: &Text{Content: "("}},
							{Text: &Text{Content: "content"}},
							{Text: &Text{Content: ")}}"}},
						},
					},
				},
			},
			data: TemplateData{
				"content": "<b>Bold text</b>",
			},
			expected: "Bold text", // The HTML tags are stripped in GetText()
		},
		{
			name: "normal template variable split across runs",
			cell: &TableCell{
				Paragraphs: []Paragraph{
					{
						Runs: []Run{
							{Text: &Text{Content: "{{"}},
							{Text: &Text{Content: "name"}},
							{Text: &Text{Content: "}}"}},
						},
					},
				},
			},
			data: TemplateData{
				"name": "John Doe",
			},
			expected: "John Doe",
		},
		{
			name: "complex expression split across runs",
			cell: &TableCell{
				Paragraphs: []Paragraph{
					{
						Runs: []Run{
							{Text: &Text{Content: "{{"}},
							{Text: &Text{Content: "item"}},
							{Text: &Text{Content: "."}},
							{Text: &Text{Content: "price"}},
							{Text: &Text{Content: " * "}},
							{Text: &Text{Content: "1.2"}},
							{Text: &Text{Content: "}}"}},
						},
					},
				},
			},
			data: TemplateData{
				"item": map[string]interface{}{
					"price": 10.0,
				},
			},
			expected: "12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a render context
			ctx := &renderContext{
				linkMarkers:      make(map[string]*LinkReplacementMarker),
				fragments:        make(map[string]*fragment),
				ooxmlFragments:   make(map[string]interface{}),
			}

			// Render the cell
			rendered, err := RenderTableCell(tt.cell, tt.data, ctx)
			if err != nil {
				t.Fatalf("RenderTableCell returned error: %v", err)
			}

			// Get the text from the rendered cell
			got := rendered.GetText()
			got = strings.TrimSpace(got)

			// Check if the result matches expected
			if got != tt.expected {
				t.Errorf("RenderTableCell() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMergeConsecutiveRunsInTableCell(t *testing.T) {
	// Test that mergeConsecutiveRuns is called within RenderTableCell
	// This tests that split runs like "{{", "for", " ", "item", ... are merged correctly
	cell := &TableCell{
		Paragraphs: []Paragraph{
			{
				Runs: []Run{
					{Text: &Text{Content: "{{"}},
					{Text: &Text{Content: "for"}},
					{Text: &Text{Content: " "}},
					{Text: &Text{Content: "item"}},
					{Text: &Text{Content: " "}},
					{Text: &Text{Content: "in"}},
					{Text: &Text{Content: " "}},
					{Text: &Text{Content: "items"}},
					{Text: &Text{Content: "}}"}},
					{Text: &Text{Content: "{{item}}"}},
					{Text: &Text{Content: "{{end}}"}},
				},
			},
		},
	}

	// Create a copy to check the original isn't modified
	originalRunCount := len(cell.Paragraphs[0].Runs)

	ctx := &renderContext{
		linkMarkers:      make(map[string]*LinkReplacementMarker),
		fragments:        make(map[string]*fragment),
		ooxmlFragments:   make(map[string]interface{}),
	}

	// This should handle the split runs correctly even though it's a control structure
	_, err := RenderTableCell(cell, TemplateData{}, ctx)
	if err != nil {
		t.Fatalf("RenderTableCell returned error: %v", err)
	}

	// Verify that the original cell wasn't modified
	if len(cell.Paragraphs[0].Runs) != originalRunCount {
		t.Errorf("RenderTableCell modified the original cell runs")
	}
}