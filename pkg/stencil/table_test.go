package stencil

import (
	"strings"
	"testing"
)

func TestTableDetection(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected bool
	}{
		{
			name: "simple table",
			xml: `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
				<w:body>
					<w:tbl>
						<w:tr>
							<w:tc>
								<w:p><w:r><w:t>Cell 1</w:t></w:r></w:p>
							</w:tc>
							<w:tc>
								<w:p><w:r><w:t>Cell 2</w:t></w:r></w:p>
							</w:tc>
						</w:tr>
					</w:tbl>
				</w:body>
			</w:document>`,
			expected: true,
		},
		{
			name: "no table",
			xml: `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
				<w:body>
					<w:p>
						<w:r><w:t>Just a paragraph</w:t></w:r>
					</w:p>
				</w:body>
			</w:document>`,
			expected: false,
		},
		{
			name: "nested table in cell",
			xml: `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
				<w:body>
					<w:tbl>
						<w:tr>
							<w:tc>
								<w:tbl>
									<w:tr>
										<w:tc>
											<w:p><w:r><w:t>Nested cell</w:t></w:r></w:p>
										</w:tc>
									</w:tr>
								</w:tbl>
							</w:tc>
						</w:tr>
					</w:tbl>
				</w:body>
			</w:document>`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseDocument(strings.NewReader(tt.xml))
			if err != nil {
				t.Fatalf("Failed to parse document: %v", err)
			}

			hasTable := detectTables(doc)
			if hasTable != tt.expected {
				t.Errorf("detectTables() = %v, want %v", hasTable, tt.expected)
			}
		})
	}
}

func TestTableContext(t *testing.T) {
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
		<w:body>
			<w:tbl>
				<w:tr>
					<w:tc>
						<w:p><w:r><w:t>{{name}}</w:t></w:r></w:p>
					</w:tc>
					<w:tc>
						<w:p><w:r><w:t>{{price}}</w:t></w:r></w:p>
					</w:tc>
				</w:tr>
			</w:tbl>
		</w:body>
	</w:document>`

	doc, err := ParseDocument(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tables := extractTables(doc)
	if len(tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	if len(table.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(table.Rows))
	}

	if len(table.Rows[0].Cells) != 2 {
		t.Errorf("Expected 2 cells, got %d", len(table.Rows[0].Cells))
	}

	// Test cell content
	cell1Text := table.Rows[0].Cells[0].GetText()
	if cell1Text != "{{name}}" {
		t.Errorf("Expected '{{name}}', got '%s'", cell1Text)
	}

	cell2Text := table.Rows[0].Cells[1].GetText()
	if cell2Text != "{{price}}" {
		t.Errorf("Expected '{{price}}', got '%s'", cell2Text)
	}
}

func TestCellIdentification(t *testing.T) {
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
		<w:body>
			<w:tbl>
				<w:tr>
					<w:tc>
						<w:tcPr>
							<w:gridSpan w:val="2"/>
						</w:tcPr>
						<w:p><w:r><w:t>Merged cell</w:t></w:r></w:p>
					</w:tc>
					<w:tc>
						<w:p><w:r><w:t>Normal cell</w:t></w:r></w:p>
					</w:tc>
				</w:tr>
				<w:tr>
					<w:tc>
						<w:p><w:r><w:t>Cell A2</w:t></w:r></w:p>
					</w:tc>
					<w:tc>
						<w:p><w:r><w:t>Cell B2</w:t></w:r></w:p>
					</w:tc>
					<w:tc>
						<w:p><w:r><w:t>Cell C2</w:t></w:r></w:p>
					</w:tc>
				</w:tr>
			</w:tbl>
		</w:body>
	</w:document>`

	doc, err := ParseDocument(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tables := extractTables(doc)
	if len(tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(tables))
	}

	table := tables[0]
	if len(table.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(table.Rows))
	}

	// Test first row - has merged cell
	row1 := table.Rows[0]
	if len(row1.Cells) != 2 {
		t.Errorf("Expected 2 cells in first row, got %d", len(row1.Cells))
	}

	// Test cell span detection
	cell1Span := getCellSpan(&row1.Cells[0])
	if cell1Span != 2 {
		t.Errorf("Expected cell span of 2, got %d", cell1Span)
	}

	cell2Span := getCellSpan(&row1.Cells[1])
	if cell2Span != 1 {
		t.Errorf("Expected cell span of 1, got %d", cell2Span)
	}

	// Test second row - no merged cells
	row2 := table.Rows[1]
	if len(row2.Cells) != 3 {
		t.Errorf("Expected 3 cells in second row, got %d", len(row2.Cells))
	}
}

func TestTableStructureParsing(t *testing.T) {
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
		<w:body>
			<w:tbl>
				<w:tblPr>
					<w:tblStyle w:val="TableGrid"/>
				</w:tblPr>
				<w:tblGrid>
					<w:gridCol w:w="2000"/>
					<w:gridCol w:w="3000"/>
					<w:gridCol w:w="2500"/>
				</w:tblGrid>
				<w:tr>
					<w:trPr>
						<w:trHeight w:val="400"/>
					</w:trPr>
					<w:tc>
						<w:tcPr>
							<w:tcW w:w="2000" w:type="dxa"/>
						</w:tcPr>
						<w:p><w:r><w:t>Header 1</w:t></w:r></w:p>
					</w:tc>
					<w:tc>
						<w:tcPr>
							<w:tcW w:w="3000" w:type="dxa"/>
						</w:tcPr>
						<w:p><w:r><w:t>Header 2</w:t></w:r></w:p>
					</w:tc>
					<w:tc>
						<w:tcPr>
							<w:tcW w:w="2500" w:type="dxa"/>
						</w:tcPr>
						<w:p><w:r><w:t>Header 3</w:t></w:r></w:p>
					</w:tc>
				</w:tr>
			</w:tbl>
		</w:body>
	</w:document>`

	doc, err := ParseDocument(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	tables := extractTables(doc)
	if len(tables) != 1 {
		t.Fatalf("Expected 1 table, got %d", len(tables))
	}

	table := tables[0]

	// Test table properties
	if table.Properties == nil {
		t.Error("Expected table properties to be set")
	} else if table.Properties.Style == nil {
		t.Error("Expected table style to be set")
	} else if table.Properties.Style.Val != "TableGrid" {
		t.Errorf("Expected table style 'TableGrid', got '%s'", table.Properties.Style.Val)
	}

	// Test table grid
	if table.Grid == nil {
		t.Error("Expected table grid to be set")
	} else if len(table.Grid.Columns) != 3 {
		t.Errorf("Expected 3 columns in grid, got %d", len(table.Grid.Columns))
	} else {
		expectedWidths := []int{2000, 3000, 2500}
		for i, col := range table.Grid.Columns {
			if col.Width != expectedWidths[i] {
				t.Errorf("Expected column %d width %d, got %d", i, expectedWidths[i], col.Width)
			}
		}
	}

	// Test row properties
	row := table.Rows[0]
	if row.Properties == nil {
		t.Error("Expected row properties to be set")
	} else if row.Properties.Height == nil {
		t.Error("Expected row height to be set")
	} else if row.Properties.Height.Val != 400 {
		t.Errorf("Expected row height 400, got %d", row.Properties.Height.Val)
	}

	// Test cell properties
	for i, cell := range row.Cells {
		if cell.Properties == nil {
			t.Errorf("Expected cell %d properties to be set", i)
		} else if cell.Properties.Width == nil {
			t.Errorf("Expected cell %d width to be set", i)
		}
	}
}