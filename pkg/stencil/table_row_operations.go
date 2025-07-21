package stencil

import (
	"strings"
)

// TableRowMarker represents a marker for table row operations
type TableRowMarker struct {
	Action string // "hide" for hideRow()
}

// registerTableRowFunctions registers table row operation functions
func registerTableRowFunctions(registry *DefaultFunctionRegistry) {
	// hideRow() function - marks a table row for removal
	hideRowFn := NewSimpleFunction("hideRow", 0, 0, func(args ...interface{}) (interface{}, error) {
		// Create a table row marker with action "hide"
		return &TableRowMarker{Action: "hide"}, nil
	})
	registry.RegisterFunction(hideRowFn)
}

// ProcessTableRowMarkers processes table row markers in a document and removes marked rows
func ProcessTableRowMarkers(doc *Document) error {
	if doc == nil || doc.Body == nil {
		return nil
	}
	
	// Process each table in the document
	var newTables []Table
	for _, table := range doc.Body.Tables {
		processedTable, err := processTableRowMarkersInTable(&table)
		if err != nil {
			return err
		}
		if processedTable != nil {
			newTables = append(newTables, *processedTable)
		}
	}
	
	doc.Body.Tables = newTables
	return nil
}

// processTableRowMarkersInTable processes row markers in a single table
func processTableRowMarkersInTable(table *Table) (*Table, error) {
	if table == nil {
		return nil, nil
	}
	
	// Create a new table with the same properties
	newTable := &Table{
		Properties: table.Properties,
		Grid:       table.Grid,
		Rows:       []TableRow{},
	}
	
	// Check if we need to handle border preservation
	// Note: In the current XML structure, borders are not directly on TableProperties
	// This is a simplified implementation - actual border handling would need to
	// check the table style or cell properties
	hasTopBorder := false
	hasBottomBorder := false
	
	// Process each row
	firstRowKept := false
	lastKeptRowIndex := -1
	
	for i, row := range table.Rows {
		// Check if this row contains a hide marker
		if containsHideRowMarker(&row) {
			// If this is the first row and it has a top border, we need to preserve it
			if i == 0 && hasTopBorder && len(table.Rows) > 1 {
				// The border will be moved to the next non-hidden row
				// This will be handled when we process the next row
			}
			
			// Skip this row (it's hidden)
			continue
		}
		
		// This row is not hidden, so we keep it
		rowCopy := row
		
		// If this is the first row we're keeping and the original first row was hidden,
		// we might need to add the top border
		if !firstRowKept && i > 0 && hasTopBorder {
			// Add top border to this row if the table had a top border
			// and the original first row was hidden
			// Note: This is a simplified implementation - the actual border
			// handling might need to be more sophisticated
		}
		
		firstRowKept = true
		lastKeptRowIndex = len(newTable.Rows)
		newTable.Rows = append(newTable.Rows, rowCopy)
	}
	
	// If the last row was hidden and the table had a bottom border,
	// we need to add it to the last kept row
	if hasBottomBorder && lastKeptRowIndex >= 0 && lastKeptRowIndex < len(table.Rows)-1 {
		// The last kept row should have the bottom border
		// Note: This is a simplified implementation
	}
	
	return newTable, nil
}

// containsHideRowMarker checks if a table row contains a hide row marker
func containsHideRowMarker(row *TableRow) bool {
	for _, cell := range row.Cells {
		if cellContainsHideRowMarker(&cell) {
			return true
		}
	}
	return false
}

// cellContainsHideRowMarker checks if a table cell contains a hide row marker
func cellContainsHideRowMarker(cell *TableCell) bool {
	for _, para := range cell.Paragraphs {
		for _, run := range para.Runs {
			if run.Text != nil && containsTableRowMarkerPlaceholder(run.Text.Content) {
				return true
			}
		}
	}
	return false
}

// containsTableRowMarkerPlaceholder checks if text contains a table row marker placeholder
func containsTableRowMarkerPlaceholder(text string) bool {
	// Look for the specific placeholder pattern that would be inserted
	// when a TableRowMarker is rendered
	// This will be coordinated with the rendering logic
	return strings.Contains(text, "TABLE_ROW_MARKER:hide")
}