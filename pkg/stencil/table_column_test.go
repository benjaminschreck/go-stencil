package stencil

import (
	"strings"
	"testing"
)

func TestHideColumnFunction(t *testing.T) {
	tests := []struct {
		name        string
		args        []interface{}
		expected    *TableColumnMarker
		expectError bool
	}{
		{
			name:     "Hide column with no arguments defaults to column 0",
			args:     []interface{}{},
			expected: &TableColumnMarker{Action: "hide", ColumnIndex: 0},
		},
		{
			name:     "Hide specific column by index",
			args:     []interface{}{1},
			expected: &TableColumnMarker{Action: "hide", ColumnIndex: 1},
		},
		{
			name:     "Hide column with int64 index",
			args:     []interface{}{int64(2)},
			expected: &TableColumnMarker{Action: "hide", ColumnIndex: 2},
		},
		{
			name:     "Hide column with resize strategy",
			args:     []interface{}{1, "redistribute"},
			expected: &TableColumnMarker{Action: "hide", ColumnIndex: 1, ResizeStrategy: "redistribute"},
		},
		{
			name:     "Hide column with proportional resize",
			args:     []interface{}{0, "proportional"},
			expected: &TableColumnMarker{Action: "hide", ColumnIndex: 0, ResizeStrategy: "proportional"},
		},
		{
			name:     "Hide column with fixed resize",
			args:     []interface{}{2, "fixed"},
			expected: &TableColumnMarker{Action: "hide", ColumnIndex: 2, ResizeStrategy: "fixed"},
		},
		{
			name:        "Invalid column index (negative)",
			args:        []interface{}{-1},
			expectError: true,
		},
		{
			name:        "Invalid resize strategy",
			args:        []interface{}{0, "invalid"},
			expectError: true,
		},
		{
			name:        "Too many arguments",
			args:        []interface{}{0, "fixed", "extra"},
			expectError: true,
		},
		{
			name:        "Non-integer column index",
			args:        []interface{}{"abc"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := GetDefaultFunctionRegistry()
			fn, exists := registry.GetFunction("hideColumn")
			if !exists {
				t.Fatalf("hideColumn function not found in registry")
			}
			
			result, err := fn.Call(tt.args...)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none, result: %v", result)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else {
					marker, ok := result.(*TableColumnMarker)
					if !ok {
						t.Errorf("Expected *TableColumnMarker, got %T", result)
					} else if marker.Action != tt.expected.Action || 
						   marker.ColumnIndex != tt.expected.ColumnIndex ||
						   marker.ResizeStrategy != tt.expected.ResizeStrategy {
						t.Errorf("Expected %+v, got %+v", tt.expected, marker)
					}
				}
			}
		})
	}
}

func TestProcessTableColumnMarkers(t *testing.T) {
	tests := []struct {
		name            string
		table           Table
		expectedColumns int
		expectedCells   []int // Number of cells in each row
	}{
		{
			name: "Simple table with hidden column",
			table: Table{
				Grid: &TableGrid{
					Columns: []GridColumn{
						{Width: 2000},
						{Width: 3000},
						{Width: 2500},
					},
				},
				Rows: []TableRow{
					{
						Cells: []TableCell{
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Header 1"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "{{TABLE_COLUMN_MARKER:hide:1}}"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Header 3"}}}}}},
						},
					},
					{
						Cells: []TableCell{
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Data 1"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Data 2"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Data 3"}}}}}},
						},
					},
				},
			},
			expectedColumns: 2,
			expectedCells:   []int{2, 2},
		},
		{
			name: "Table with redistribute resize strategy",
			table: Table{
				Grid: &TableGrid{
					Columns: []GridColumn{
						{Width: 2000},
						{Width: 3000},
						{Width: 2000},
					},
				},
				Rows: []TableRow{
					{
						Cells: []TableCell{
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "{{TABLE_COLUMN_MARKER:hide:1:redistribute}}"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Col 2"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Col 3"}}}}}},
						},
					},
				},
			},
			expectedColumns: 2,
			expectedCells:   []int{2},
		},
		{
			name: "Table with proportional resize strategy",
			table: Table{
				Grid: &TableGrid{
					Columns: []GridColumn{
						{Width: 2000},
						{Width: 3000},
						{Width: 4000},
					},
				},
				Rows: []TableRow{
					{
						Cells: []TableCell{
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Col 1"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "{{TABLE_COLUMN_MARKER:hide:1:proportional}}"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Col 3"}}}}}},
						},
					},
				},
			},
			expectedColumns: 2,
			expectedCells:   []int{2},
		},
		{
			name: "Table with fixed resize strategy",
			table: Table{
				Grid: &TableGrid{
					Columns: []GridColumn{
						{Width: 2000},
						{Width: 3000},
						{Width: 4000},
					},
				},
				Rows: []TableRow{
					{
						Cells: []TableCell{
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "{{TABLE_COLUMN_MARKER:hide:0:fixed}}"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Col 2"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Col 3"}}}}}},
						},
					},
				},
			},
			expectedColumns: 2,
			expectedCells:   []int{2},
		},
		{
			name: "Table with merged cells (gridSpan)",
			table: Table{
				Grid: &TableGrid{
					Columns: []GridColumn{
						{Width: 2000},
						{Width: 2000},
						{Width: 2000},
					},
				},
				Rows: []TableRow{
					{
						Cells: []TableCell{
							{
								Properties: &TableCellProperties{GridSpan: &GridSpan{Val: 2}},
								Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Merged 1-2"}}}}},
							},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "{{TABLE_COLUMN_MARKER:hide:2}}"}}}}}},
						},
					},
					{
						Cells: []TableCell{
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Cell 1"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Cell 2"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Cell 3"}}}}}},
						},
					},
				},
			},
			expectedColumns: 2,
			expectedCells:   []int{1, 2},
		},
		{
			name: "Hide column affecting merged cell",
			table: Table{
				Grid: &TableGrid{
					Columns: []GridColumn{
						{Width: 2000},
						{Width: 2000},
						{Width: 2000},
					},
				},
				Rows: []TableRow{
					{
						Cells: []TableCell{
							{
								Properties: &TableCellProperties{GridSpan: &GridSpan{Val: 2}},
								Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "{{TABLE_COLUMN_MARKER:hide:1}}"}}}}},
							},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Cell 3"}}}}}},
						},
					},
				},
			},
			expectedColumns: 2,
			expectedCells:   []int{2},
		},
		{
			name: "Multiple columns hidden",
			table: Table{
				Grid: &TableGrid{
					Columns: []GridColumn{
						{Width: 1000},
						{Width: 2000},
						{Width: 3000},
						{Width: 4000},
					},
				},
				Rows: []TableRow{
					{
						Cells: []TableCell{
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "{{TABLE_COLUMN_MARKER:hide:0}}"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Col 2"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "{{TABLE_COLUMN_MARKER:hide:2}}"}}}}}},
							{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Col 4"}}}}}},
						},
					},
				},
			},
			expectedColumns: 2,
			expectedCells:   []int{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Process the table
			result, err := processTableColumnMarkersInTable(&tt.table)
			if err != nil {
				t.Fatalf("processTableColumnMarkersInTable() error = %v", err)
			}
			
			// Check grid columns
			if result.Grid == nil {
				t.Fatal("Result table has no grid")
			}
			if len(result.Grid.Columns) != tt.expectedColumns {
				t.Errorf("Expected %d columns, got %d", tt.expectedColumns, len(result.Grid.Columns))
			}
			
			// Check rows and cells
			if len(result.Rows) != len(tt.expectedCells) {
				t.Errorf("Expected %d rows, got %d", len(tt.expectedCells), len(result.Rows))
			}
			
			for i, row := range result.Rows {
				if i < len(tt.expectedCells) {
					if len(row.Cells) != tt.expectedCells[i] {
						t.Errorf("Row %d: expected %d cells, got %d", i, tt.expectedCells[i], len(row.Cells))
					}
				}
			}
			
			// Check that column markers are removed
			for _, row := range result.Rows {
				for _, cell := range row.Cells {
					for _, para := range cell.Paragraphs {
						for _, run := range para.Runs {
							if run.Text != nil && strings.Contains(run.Text.Content, "TABLE_COLUMN_MARKER") {
								t.Errorf("Column marker not removed from cell content: %s", run.Text.Content)
							}
						}
					}
				}
			}
		})
	}
}


func TestResizeStrategies(t *testing.T) {
	tests := []struct {
		name            string
		oldWidths       []int
		columnsToHide   map[int]string
		expectedWidths  []int
	}{
		{
			name:          "Redistribute strategy - equal distribution",
			oldWidths:     []int{2000, 3000, 2000},
			columnsToHide: map[int]string{1: "redistribute"},
			expectedWidths: []int{3500, 3500}, // 3000 distributed equally to remaining 2 columns
		},
		{
			name:          "Proportional strategy - proportional distribution",
			oldWidths:     []int{2000, 3000, 4000},
			columnsToHide: map[int]string{1: "proportional"},
			expectedWidths: []int{3000, 6000}, // 3000 distributed proportionally (1/3 to first, 2/3 to second)
		},
		{
			name:          "Fixed strategy - no redistribution",
			oldWidths:     []int{2000, 3000, 4000},
			columnsToHide: map[int]string{0: "fixed"},
			expectedWidths: []int{3000, 4000}, // Original widths maintained
		},
		{
			name:          "Multiple columns hidden with redistribute",
			oldWidths:     []int{1000, 2000, 3000, 4000},
			columnsToHide: map[int]string{0: "redistribute", 2: ""},
			expectedWidths: []int{4000, 6000}, // 4000 total distributed equally (2000 each)
		},
		{
			name:          "All but one column hidden",
			oldWidths:     []int{1000, 2000, 3000},
			columnsToHide: map[int]string{0: "redistribute", 2: ""},
			expectedWidths: []int{6000}, // All width goes to remaining column
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateNewGridColumns(tt.oldWidths, tt.columnsToHide)
			if len(result) != len(tt.expectedWidths) {
				t.Errorf("Expected %d columns, got %d", len(tt.expectedWidths), len(result))
				return
			}
			
			for i, width := range result {
				if width != tt.expectedWidths[i] {
					t.Errorf("Column %d: expected width %d, got %d", i, tt.expectedWidths[i], width)
				}
			}
		})
	}
}

