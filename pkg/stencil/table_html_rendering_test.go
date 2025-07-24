package stencil

import (
	"strings"
	"testing"
)

func TestHTMLFunctionInTableCells(t *testing.T) {
	// Create a table with HTML function calls
	table := &Table{
		Rows: []TableRow{
			{
				Cells: []TableCell{
					{
						Paragraphs: []Paragraph{
							{
								Runs: []Run{
									{Text: &Text{Content: "{{html(row.col1)}}"}},
								},
							},
						},
					},
					{
						Paragraphs: []Paragraph{
							{
								Runs: []Run{
									{Text: &Text{Content: "{{html(row.col2)}}"}},
								},
							},
						},
					},
				},
			},
		},
	}

	// Test data
	data := TemplateData{
		"row": map[string]interface{}{
			"col1": "<b>Bold text</b>",
			"col2": "<i>Italic text</i>",
		},
	}

	// Create render context
	ctx := &renderContext{
		linkMarkers:       make(map[string]*LinkReplacementMarker),
		fragments:         make(map[string]*fragment),
		ooxmlFragments:    make(map[string]interface{}),
	}

	// Render the table
	rendered, err := RenderTableWithControlStructures(table, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render table: %v", err)
	}

	// Check the first cell
	if len(rendered.Rows) == 0 || len(rendered.Rows[0].Cells) < 2 {
		t.Fatal("Expected at least one row with two cells")
	}

	// Get text from first cell
	cell1Text := rendered.Rows[0].Cells[0].GetText()
	cell2Text := rendered.Rows[0].Cells[1].GetText()

	t.Logf("Cell 1 text: %q", cell1Text)
	t.Logf("Cell 2 text: %q", cell2Text)

	// The template expressions should be processed
	if strings.Contains(cell1Text, "html(") {
		t.Errorf("Cell 1 still contains unprocessed template expression: %s", cell1Text)
	}
	if strings.Contains(cell2Text, "html(") {
		t.Errorf("Cell 2 still contains unprocessed template expression: %s", cell2Text)
	}

	// Check that the HTML was properly rendered to text
	// The HTML should be converted to plain text (just the content without tags)
	if cell1Text != "Bold text" {
		t.Errorf("Cell 1 text should be 'Bold text', got: %s", cell1Text)
	}
	if cell2Text != "Italic text" {
		t.Errorf("Cell 2 text should be 'Italic text', got: %s", cell2Text)
	}
}

func TestHTMLFunctionInTableCellsWithSplitRuns(t *testing.T) {
	// Create a table with HTML function calls split across runs (like Word does)
	table := &Table{
		Rows: []TableRow{
			{
				Cells: []TableCell{
					{
						Paragraphs: []Paragraph{
							{
								Runs: []Run{
									{Text: &Text{Content: "{{"}},
									{Text: &Text{Content: "html("}},
									{Text: &Text{Content: "row.col1"}},
									{Text: &Text{Content: ")}}"}},
								},
							},
						},
					},
				},
			},
		},
	}

	// Test data
	data := TemplateData{
		"row": map[string]interface{}{
			"col1": "<b>Bold text</b>",
		},
	}

	// Create render context
	ctx := &renderContext{
		linkMarkers:       make(map[string]*LinkReplacementMarker),
		fragments:         make(map[string]*fragment),
		ooxmlFragments:    make(map[string]interface{}),
	}

	// Render the table
	rendered, err := RenderTableWithControlStructures(table, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render table: %v", err)
	}

	// Check the cell
	if len(rendered.Rows) == 0 || len(rendered.Rows[0].Cells) == 0 {
		t.Fatal("Expected at least one row with one cell")
	}

	// Get text from cell
	cellText := rendered.Rows[0].Cells[0].GetText()
	t.Logf("Cell text: %q", cellText)

	// The template expression should be processed
	if strings.Contains(cellText, "html(") {
		t.Errorf("Cell still contains unprocessed template expression: %s", cellText)
	}

	// Check that the HTML was properly rendered to text
	// The HTML should be converted to plain text (just the content without tags)
	if cellText != "Bold text" {
		t.Errorf("Cell text should be 'Bold text', got: %s", cellText)
	}
}