package stencil

import (
	"testing"
)

// Helper function to create a Body with Tables
func createBodyWithTables(tables []Table) *Body {
	body := &Body{
		Elements: make([]BodyElement, len(tables)),
	}
	for i := range tables {
		body.Elements[i] = &tables[i]
	}
	return body
}

func TestHideRowFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		wantErr bool
	}{
		{
			name:    "hideRow() with no arguments",
			args:    []interface{}{},
			wantErr: false,
		},
		{
			name:    "hideRow() with unexpected arguments",
			args:    []interface{}{"arg1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := GetDefaultFunctionRegistry()
			fn, exists := registry.GetFunction("hideRow")
			if !exists {
				t.Fatalf("hideRow function not found in registry")
			}

			result, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("hideRow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check that result is a TableRowMarker
				marker, ok := result.(*TableRowMarker)
				if !ok {
					t.Errorf("Expected *TableRowMarker, got %T", result)
					return
				}

				// Check that it's a hide marker
				if marker.Action != "hide" {
					t.Errorf("Expected action 'hide', got %s", marker.Action)
				}
			}
		})
	}
}

// TestTableRowHidingInTemplate tests will be implemented once the full template rendering pipeline is in place
// For now, we focus on the hideRow function itself

// TestTableRowBorderPreservation will be implemented once the full template rendering pipeline is in place

func TestProcessTableRowMarkers(t *testing.T) {
	tests := []struct {
		name         string
		doc          *Document
		wantRowCount int
	}{
		{
			name: "remove row with hide marker",
			doc: &Document{
				Body: createBodyWithTables([]Table{
					{
						Rows: []TableRow{
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "Header"}},
												},
											},
										},
									},
								},
							},
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "{{TABLE_ROW_MARKER:hide}}"}},
												},
											},
										},
									},
								},
							},
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "Footer"}},
												},
											},
										},
									},
								},
							},
						},
					},
				}),
			},
			wantRowCount: 2,
		},
		{
			name: "remove multiple rows with hide markers",
			doc: &Document{
				Body: createBodyWithTables([]Table{
					{
						Rows: []TableRow{
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "Row 1"}},
												},
											},
										},
									},
								},
							},
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "{{TABLE_ROW_MARKER:hide}}"}},
												},
											},
										},
									},
								},
							},
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "Row 3"}},
												},
											},
										},
									},
								},
							},
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "{{TABLE_ROW_MARKER:hide}}"}},
												},
											},
										},
									},
								},
							},
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "Row 5"}},
												},
											},
										},
									},
								},
							},
						},
					},
				}),
			},
			wantRowCount: 3,
		},
		{
			name: "no rows removed when no markers",
			doc: &Document{
				Body: createBodyWithTables([]Table{
					{
						Rows: []TableRow{
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "Row 1"}},
												},
											},
										},
									},
								},
							},
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "Row 2"}},
												},
											},
										},
									},
								},
							},
							{
								Cells: []TableCell{
									{
										Paragraphs: []Paragraph{
											{
												Runs: []Run{
													{Text: &Text{Content: "Row 3"}},
												},
											},
										},
									},
								},
							},
						},
					},
				}),
			},
			wantRowCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Process table row markers
			err := ProcessTableRowMarkers(tt.doc)
			if err != nil {
				t.Fatalf("ProcessTableRowMarkers() error = %v", err)
			}

			// Check row count
			if tt.doc.Body != nil && len(tt.doc.Body.Elements) > 0 {
				// Find the first table in the elements
				for _, elem := range tt.doc.Body.Elements {
					if table, ok := elem.(*Table); ok {
						actualRowCount := len(table.Rows)
						if actualRowCount != tt.wantRowCount {
							t.Errorf("got %d rows, want %d rows", actualRowCount, tt.wantRowCount)
						}
						break
					}
				}
			}
		})
	}
}
