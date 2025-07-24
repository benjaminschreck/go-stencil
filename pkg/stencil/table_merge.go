package stencil

// MergeConsecutiveTables merges consecutive tables that were split by template loops
// This implementation is content-agnostic and merges based purely on position and structure
func MergeConsecutiveTables(elements []BodyElement) []BodyElement {
	if len(elements) < 2 {
		return elements
	}

	var result []BodyElement
	var currentTableGroup []*Table

	for i := 0; i < len(elements); i++ {
		table, isTable := elements[i].(*Table)
		
		if !isTable {
			// Not a table - finalize any current group and add this element
			if len(currentTableGroup) > 0 {
				merged := mergeTables(currentTableGroup)
				result = append(result, merged)
				currentTableGroup = nil
			}
			result = append(result, elements[i])
			continue
		}

		// This is a table
		if len(currentTableGroup) == 0 {
			// Start a new group
			currentTableGroup = []*Table{table}
		} else {
			// Check if this table is compatible with the group
			if areTablesCompatible(currentTableGroup[0], table) {
				// Add to current group
				currentTableGroup = append(currentTableGroup, table)
			} else {
				// Not compatible - finalize current group and start new one
				merged := mergeTables(currentTableGroup)
				result = append(result, merged)
				currentTableGroup = []*Table{table}
			}
		}
	}

	// Don't forget the last group
	if len(currentTableGroup) > 0 {
		merged := mergeTables(currentTableGroup)
		result = append(result, merged)
	}

	return result
}

// areTablesCompatible checks if two tables can be merged
// Uses simple structural compatibility - same number of columns
func areTablesCompatible(t1, t2 *Table) bool {
	if t1 == nil || t2 == nil || len(t1.Rows) == 0 || len(t2.Rows) == 0 {
		return false
	}

	// Get column count from first row of each table
	cols1 := len(t1.Rows[0].Cells)
	cols2 := len(t2.Rows[0].Cells)

	// Special case: single-cell tables (like section headers) are compatible with multi-column tables
	if cols1 == 1 || cols2 == 1 {
		return true
	}

	// Otherwise, must have same column count
	return cols1 == cols2
}

// mergeTables combines multiple tables into one
func mergeTables(tables []*Table) *Table {
	if len(tables) == 0 {
		return nil
	}
	if len(tables) == 1 {
		return tables[0]
	}

	// Start with a copy of the first table's structure
	merged := &Table{
		Properties: tables[0].Properties,
		Grid:       tables[0].Grid,
		Rows:       []TableRow{},
	}

	// Add all rows from all tables
	for _, table := range tables {
		merged.Rows = append(merged.Rows, table.Rows...)
	}

	return merged
}