package stencil

// detectTables checks if a document contains any tables
func detectTables(doc *Document) bool {
	if doc == nil || doc.Body == nil {
		return false
	}
	
	tables := extractTables(doc)
	return len(tables) > 0 || hasNestedTables(tables)
}

// hasNestedTables recursively checks for tables nested within table cells
func hasNestedTables(tables []Table) bool {
	for _, table := range tables {
		for _, row := range table.Rows {
			for _, cell := range row.Cells {
				// Check if any paragraph in the cell contains nested tables
				// In OOXML, nested tables would be additional Table elements
				// within the cell's content, but our current XML structure
				// doesn't capture this. For now, we return false for nested checks.
				// This is a placeholder for future enhancement.
				_ = cell
			}
		}
	}
	return false
}

// extractTables returns all tables found in the document
func extractTables(doc *Document) []Table {
	if doc == nil || doc.Body == nil {
		return []Table{}
	}
	
	var tables []Table
	for _, elem := range doc.Body.Elements {
		if table, ok := elem.(Table); ok {
			tables = append(tables, table)
		}
	}
	return tables
}

// getCellSpan returns the column span of a table cell
func getCellSpan(cell *TableCell) int {
	if cell == nil || cell.Properties == nil || cell.Properties.GridSpan == nil {
		return 1
	}
	
	return cell.Properties.GridSpan.Val
}

// TableContext represents the context information for a table during processing
type TableContext struct {
	TableIndex int         // Index of the table in the document
	Table      *Table      // Pointer to the table
	RowIndex   int         // Current row being processed
	CellIndex  int         // Current cell being processed
	ColumnSpan int         // Column span of current cell
}

// NewTableContext creates a new table context
func NewTableContext(tableIndex int, table *Table) *TableContext {
	return &TableContext{
		TableIndex: tableIndex,
		Table:      table,
		RowIndex:   -1,
		CellIndex:  -1,
		ColumnSpan: 1,
	}
}

// IsInTable returns true if we're currently processing within a table
func (tc *TableContext) IsInTable() bool {
	return tc.Table != nil
}

// IsInCell returns true if we're currently processing within a table cell
func (tc *TableContext) IsInCell() bool {
	return tc.IsInTable() && tc.RowIndex >= 0 && tc.CellIndex >= 0
}

// CurrentCell returns the current cell being processed, or nil if not in a cell
func (tc *TableContext) CurrentCell() *TableCell {
	if !tc.IsInCell() {
		return nil
	}
	
	if tc.RowIndex >= len(tc.Table.Rows) {
		return nil
	}
	
	row := &tc.Table.Rows[tc.RowIndex]
	if tc.CellIndex >= len(row.Cells) {
		return nil
	}
	
	return &row.Cells[tc.CellIndex]
}

// EnterRow sets the context to process a specific row
func (tc *TableContext) EnterRow(rowIndex int) {
	tc.RowIndex = rowIndex
	tc.CellIndex = -1
	tc.ColumnSpan = 1
}

// EnterCell sets the context to process a specific cell
func (tc *TableContext) EnterCell(cellIndex int) {
	tc.CellIndex = cellIndex
	if cell := tc.CurrentCell(); cell != nil {
		tc.ColumnSpan = getCellSpan(cell)
	} else {
		tc.ColumnSpan = 1
	}
}

// ExitCell exits the current cell context
func (tc *TableContext) ExitCell() {
	tc.CellIndex = -1
	tc.ColumnSpan = 1
}

// ExitRow exits the current row context
func (tc *TableContext) ExitRow() {
	tc.RowIndex = -1
	tc.CellIndex = -1
	tc.ColumnSpan = 1
}

// TableInfo provides information about a table's structure
type TableInfo struct {
	Index       int    // Index of the table in the document
	RowCount    int    // Number of rows
	ColumnCount int    // Number of columns (maximum across all rows)
	HasGrid     bool   // Whether the table has explicit column definitions
	GridWidths  []int  // Column widths from table grid
}

// GetTableInfo analyzes a table and returns structural information
func GetTableInfo(table *Table) *TableInfo {
	info := &TableInfo{
		RowCount:    len(table.Rows),
		ColumnCount: 0,
		HasGrid:     false,
		GridWidths:  []int{},
	}
	
	// Check for table grid information
	if table.Grid != nil && len(table.Grid.Columns) > 0 {
		info.HasGrid = true
		for _, col := range table.Grid.Columns {
			info.GridWidths = append(info.GridWidths, col.Width)
		}
		info.ColumnCount = len(table.Grid.Columns)
	}
	
	// Calculate column count from actual rows if no grid is available
	if info.ColumnCount == 0 {
		for _, row := range table.Rows {
			columnCount := 0
			for _, cell := range row.Cells {
				columnCount += getCellSpan(&cell)
			}
			if columnCount > info.ColumnCount {
				info.ColumnCount = columnCount
			}
		}
	}
	
	return info
}

// FindTablesWithTemplates finds all tables that contain template expressions
func FindTablesWithTemplates(doc *Document) []*Table {
	var tablesWithTemplates []*Table
	
	for _, elem := range doc.Body.Elements {
		if table, ok := elem.(Table); ok {
			if tableHasTemplates(&table) {
				tablesWithTemplates = append(tablesWithTemplates, &table)
			}
		}
	}
	
	return tablesWithTemplates
}

// tableHasTemplates checks if a table contains any template expressions
func tableHasTemplates(table *Table) bool {
	for _, row := range table.Rows {
		for _, cell := range row.Cells {
			if cellHasTemplates(&cell) {
				return true
			}
		}
	}
	return false
}

// cellHasTemplates checks if a cell contains any template expressions
func cellHasTemplates(cell *TableCell) bool {
	for _, para := range cell.Paragraphs {
		if paragraphHasTemplates(&para) {
			return true
		}
	}
	return false
}

// paragraphHasTemplates checks if a paragraph contains any template expressions
func paragraphHasTemplates(para *Paragraph) bool {
	text := para.GetText()
	return containsTemplateExpression(text)
}

// containsTemplateExpression checks if text contains template expressions
func containsTemplateExpression(text string) bool {
	// Simple check for {{ }} template expressions
	// This could be enhanced to use the actual tokenizer
	for i := 0; i < len(text)-1; i++ {
		if text[i] == '{' && text[i+1] == '{' {
			return true
		}
	}
	return false
}