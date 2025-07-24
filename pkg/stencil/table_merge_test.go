package stencil

import (
	"testing"
)

func TestMergeConsecutiveTables(t *testing.T) {
	tests := []struct {
		name     string
		elements []BodyElement
		expected int // expected number of elements after merge
	}{
		{
			name: "no tables",
			elements: []BodyElement{
				&Paragraph{},
				&Paragraph{},
			},
			expected: 2,
		},
		{
			name: "single table",
			elements: []BodyElement{
				createTestTable("Region", []string{"North"}),
			},
			expected: 1,
		},
		{
			name: "non-consecutive tables",
			elements: []BodyElement{
				createTestTable("Region", []string{"North"}),
				&Paragraph{},
				createTestTable("Product", []string{"Widget"}),
			},
			expected: 3,
		},
		{
			name: "header and data tables should merge",
			elements: []BodyElement{
				createTestTable("Region", []string{"Q1", "Q2", "Q3", "Q4", "Total"}), // header
				createTestTable("North", []string{"100", "200", "300", "400", "1000"}), // data row 1
				createTestTable("South", []string{"80", "90", "100", "110", "380"}),    // data row 2
				createTestTable("TOTAL", []string{"180", "290", "400", "510", "1380"}), // total row
			},
			expected: 1, // should merge into single table
		},
		{
			name: "header, data, and separate table",
			elements: []BodyElement{
				createTestTable("Region", []string{"Q1", "Q2", "Q3", "Q4", "Total"}), // header
				createTestTable("North", []string{"100", "200", "300", "400", "1000"}), // data
				createTestTable("South", []string{"80", "90", "100", "110", "380"}),    // data
				createTestTable("TOTAL", []string{"180", "290", "400", "510", "1380"}), // total
				&Paragraph{}, // separator
				createTestTable("Product", []string{"Sales"}), // new table (different structure)
			},
			expected: 3, // merged table + paragraph + new table
		},
		{
			name: "tables with different column counts",
			elements: []BodyElement{
				createTestTable("Name", []string{"Age", "City"}),          // 3 columns
				createTestTable("John", []string{"30", "New York"}),       // 3 columns
				createTestTable("Product", []string{"Price", "Stock", "Category"}), // 4 columns - different
			},
			expected: 2, // first two merge, third stays separate
		},
		{
			name: "multiple header tables",
			elements: []BodyElement{
				createTestTable("Region", []string{"Q1", "Q2"}),    // header 1
				createTestTable("Product", []string{"Price", "Stock"}), // header 2
			},
			expected: 1, // consecutive tables with same structure merge
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeConsecutiveTables(tt.elements)
			if len(result) != tt.expected {
				t.Errorf("MergeConsecutiveTables() returned %d elements, expected %d", len(result), tt.expected)
				
				// Debug output
				t.Log("Result elements:")
				for i, elem := range result {
					switch e := elem.(type) {
					case *Table:
						t.Logf("  [%d] Table with %d rows", i, len(e.Rows))
					case *Paragraph:
						t.Logf("  [%d] Paragraph", i)
					default:
						t.Logf("  [%d] %T", i, elem)
					}
				}
			}
			
			// For the merge test case, verify the merged table has all rows
			if tt.name == "header and data tables should merge" && len(result) == 1 {
				table, ok := result[0].(*Table)
				if !ok {
					t.Error("Expected first element to be a Table")
				} else if len(table.Rows) != 4 {
					t.Errorf("Merged table has %d rows, expected 4", len(table.Rows))
				}
			}
		})
	}
}

// createTestTable creates a simple test table with one row
func createTestTable(firstCell string, otherCells []string) *Table {
	cells := []TableCell{
		{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: firstCell}}}}}},
	}
	
	for _, cellText := range otherCells {
		cells = append(cells, TableCell{
			Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: cellText}}}}},
		})
	}
	
	// Create grid based on number of cells
	var grid *TableGrid
	if len(cells) > 0 {
		grid = &TableGrid{
			Columns: make([]GridColumn, len(cells)),
		}
		for i := range grid.Columns {
			grid.Columns[i] = GridColumn{Width: 2000} // arbitrary width
		}
	}
	
	return &Table{
		Grid: grid,
		Rows: []TableRow{
			{Cells: cells},
		},
	}
}

func TestTableMergingScenarios(t *testing.T) {
	t.Run("split table from for loop", func(t *testing.T) {
		// This simulates the output from table_demo.docx where a for loop
		// outside tables creates multiple single-row tables
		elements := []BodyElement{
			// First table (properly rendered with for loop inside)
			&Table{
				Grid: &TableGrid{Columns: []GridColumn{{Width: 2000}, {Width: 2000}}},
				Rows: []TableRow{
					{Cells: []TableCell{
						{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Region"}}}}}},
						{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "Sales"}}}}}},
					}},
					{Cells: []TableCell{
						{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "North"}}}}}},
						{Paragraphs: []Paragraph{{Runs: []Run{{Text: &Text{Content: "1000"}}}}}},
					}},
				},
			},
			// Second set of tables (split due to for loop outside table)
			createTestTable("Region", []string{"Q1", "Q2", "Q3", "Q4", "Total"}), // header
			createTestTable("North", []string{"100000", "120000", "110000", "150000", "480000"}),
			createTestTable("South", []string{"80000", "85000", "90000", "95000", "350000"}),
			createTestTable("East", []string{"95000", "100000", "105000", "110000", "410000"}),
			createTestTable("West", []string{"110000", "115000", "120000", "125000", "470000"}),
			createTestTable("TOTAL", []string{"", "", "", "", "1710000"}), // total row
		}
		
		result := MergeConsecutiveTables(elements)
		
		// Should have 2 tables: the first one unchanged, and the 6 split tables merged into 1
		if len(result) != 2 {
			t.Errorf("Expected 2 elements after merge, got %d", len(result))
		}
		
		// Check the second table has all the merged rows
		if len(result) > 1 {
			mergedTable, ok := result[1].(*Table)
			if !ok {
				t.Error("Expected second element to be a Table")
			} else if len(mergedTable.Rows) != 6 {
				t.Errorf("Expected merged table to have 6 rows (header + 4 data + total), got %d", len(mergedTable.Rows))
			}
		}
	})
}