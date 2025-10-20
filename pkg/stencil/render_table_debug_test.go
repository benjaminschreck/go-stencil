package stencil

import (
	"fmt"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

func TestRenderTableWithForLoop_Debug(t *testing.T) {
	// Create a table exactly like in report.docx
	table := &Table{
		Properties: &TableProperties{},
		Grid: &TableGrid{
			Columns: []GridColumn{
				{Width: 4531},
				{Width: 4531},
			},
		},
		Rows: []TableRow{
			// Row 1: {{for highlight in highlights}}
			{
				Cells: []TableCell{
					{
						Paragraphs: []Paragraph{
							{
								Runs: []Run{
									{Text: &Text{Content: "{{for highlight in highlights}}"}},
								},
							},
						},
					},
					{
						Paragraphs: []Paragraph{{}}, // Empty cell
					},
				},
			},
			// Row 2: {{highlight}} | Placeholder
			{
				Cells: []TableCell{
					{
						Paragraphs: []Paragraph{
							{
								Runs: []Run{
									{Text: &Text{Content: "{{highlight}}"}},
								},
							},
						},
					},
					{
						Paragraphs: []Paragraph{
							{
								Runs: []Run{
									{Text: &Text{Content: "Placeholder"}},
								},
							},
						},
					},
				},
			},
			// Row 3: {{end}}
			{
				Cells: []TableCell{
					{
						Paragraphs: []Paragraph{
							{
								Runs: []Run{
									{Text: &Text{Content: "{{end}}"}},
								},
							},
						},
					},
					{
						Paragraphs: []Paragraph{{}}, // Empty cell
					},
				},
			},
		},
	}

	// Test data
	data := TemplateData{
		"highlights": []string{
			"Revenue increased by 15%",
			"Customer satisfaction at all-time high",
			"New product launch successful",
		},
	}

	// Create a mock render context
	ctx := &renderContext{
		linkMarkers:      make(map[string]*LinkReplacementMarker),
		fragments:        make(map[string]*fragment),
		fragmentStack:    []string{},
		renderDepth:      0,
		ooxmlFragments:   make(map[string]interface{}),
	}

	// Test row detection
	fmt.Println("=== Testing row control structure detection ===")
	for i, row := range table.Rows {
		controlType, controlContent := render.DetectTableRowControlStructure(&row)
		fmt.Printf("Row %d: controlType=%q, controlContent=%q\n", i, controlType, controlContent)
	}

	// Render the table
	fmt.Println("\n=== Rendering table ===")
	rendered, err := RenderTableWithControlStructures(table, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render table: %v", err)
	}

	fmt.Printf("Rendered table has %d rows\n", len(rendered.Rows))
	
	// Print rendered rows
	for i, row := range rendered.Rows {
		fmt.Printf("\nRow %d (%d cells):\n", i, len(row.Cells))
		for j, cell := range row.Cells {
			text := ""
			if len(cell.Paragraphs) > 0 && len(cell.Paragraphs[0].Runs) > 0 && cell.Paragraphs[0].Runs[0].Text != nil {
				text = cell.Paragraphs[0].Runs[0].Text.Content
			}
			fmt.Printf("  Cell %d: %q\n", j, text)
		}
	}

	// Expected: 3 rows (one for each highlight)
	if len(rendered.Rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rendered.Rows))
	}
}