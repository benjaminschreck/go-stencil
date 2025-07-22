package stencil

import (
	"fmt"
	"strconv"
	"strings"
)

// TableColumnMarker represents a marker for column operations
type TableColumnMarker struct {
	Action         string // "hide"
	ColumnIndex    int
	ResizeStrategy string // "redistribute", "proportional", "fixed", or empty for default
}

// String returns the string representation of the marker for rendering
func (m TableColumnMarker) String() string {
	if m.ResizeStrategy != "" {
		return fmt.Sprintf("{{TABLE_COLUMN_MARKER:%s:%d:%s}}", m.Action, m.ColumnIndex, m.ResizeStrategy)
	}
	return fmt.Sprintf("{{TABLE_COLUMN_MARKER:%s:%d}}", m.Action, m.ColumnIndex)
}

// hideColumn hides a table column at the specified index
// When called without arguments, it hides the column containing the function call
func hideColumn(args ...interface{}) (interface{}, error) {
	marker := TableColumnMarker{Action: "hide", ColumnIndex: -1} // -1 means "current column"

	if len(args) > 2 {
		return nil, fmt.Errorf("hideColumn: too many arguments (expected 0-2, got %d)", len(args))
	}

	// Parse column index if provided
	if len(args) >= 1 {
		switch v := args[0].(type) {
		case int:
			if v < 0 {
				return nil, fmt.Errorf("hideColumn: column index must be non-negative, got %d", v)
			}
			marker.ColumnIndex = v
		case int64:
			if v < 0 {
				return nil, fmt.Errorf("hideColumn: column index must be non-negative, got %d", v)
			}
			marker.ColumnIndex = int(v)
		case float64:
			if v < 0 || v != float64(int(v)) {
				return nil, fmt.Errorf("hideColumn: column index must be a non-negative integer, got %v", v)
			}
			marker.ColumnIndex = int(v)
		default:
			return nil, fmt.Errorf("hideColumn: column index must be an integer, got %T", v)
		}
	}

	// Parse resize strategy if provided
	if len(args) == 2 {
		strategy, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("hideColumn: resize strategy must be a string, got %T", args[1])
		}

		// Validate resize strategy
		switch strategy {
		case "redistribute", "proportional", "fixed":
			marker.ResizeStrategy = strategy
		default:
			return nil, fmt.Errorf("hideColumn: invalid resize strategy '%s' (must be 'redistribute', 'proportional', or 'fixed')", strategy)
		}
	}

	return &marker, nil
}

// registerTableColumnFunctions registers table column-related functions
func registerTableColumnFunctions(registry *DefaultFunctionRegistry) {
	// hideColumn() function - marks a table column for removal
	hideColumnFn := NewSimpleFunction("hideColumn", 0, 2, hideColumn)
	registry.RegisterFunction(hideColumnFn)
}

// ProcessTableColumnMarkers processes column markers in the document
func ProcessTableColumnMarkers(doc *Document) error {
	logger := GetLogger()
	logger.Debug("ProcessTableColumnMarkers called")
	if doc == nil || doc.Body == nil {
		logger.Debug("Document or body is nil")
		return nil
	}
	
	// Process each element in the document
	var newElements []BodyElement
	for i, elem := range doc.Body.Elements {
		if table, ok := elem.(Table); ok {
			logger.Debug("Found table at element %d", i)
			processedTable, err := processTableColumnMarkersInTable(&table)
			if err != nil {
				return err
			}
			if processedTable != nil {
				newElements = append(newElements, *processedTable)
			}
		} else if tablePtr, ok := elem.(*Table); ok {
			logger.Debug("Found table pointer at element %d", i)
			processedTable, err := processTableColumnMarkersInTable(tablePtr)
			if err != nil {
				return err
			}
			if processedTable != nil {
				newElements = append(newElements, *processedTable)
			}
		} else {
			logger.Debug("Element %d is type %T", i, elem)
			newElements = append(newElements, elem)
		}
	}
	
	doc.Body.Elements = newElements
	logger.Debug("ProcessTableColumnMarkers completed, processed %d elements", len(newElements))
	return nil
}

// processTableColumnMarkersInTable processes column markers in a single table
func processTableColumnMarkersInTable(table *Table) (*Table, error) {
	logger := GetLogger()
	if table == nil {
		return nil, nil
	}
	
	// Find column markers and determine which columns to hide
	columnsToHide := make(map[int]string) // column index -> resize strategy
	
	logger.Debug("Scanning table with %d rows for column markers", len(table.Rows))
	for rowIdx, row := range table.Rows {
		cellIndex := 0
		for colIdx, cell := range row.Cells {
			// Check if this cell contains a hide column marker
			if marker, strategy := getCellColumnMarker(&cell); marker != nil {
				logger.Debug("Found marker in row %d, cell %d", rowIdx, colIdx)
				// If ColumnIndex is -1, use the current cell's column index
				columnIdx := marker.ColumnIndex
				if columnIdx == -1 {
					columnIdx = cellIndex
				}
				logger.Debug("Found hideColumn marker in cell %d (cellIndex=%d), will hide column %d", colIdx, cellIndex, columnIdx)
				columnsToHide[columnIdx] = strategy
			}
			
			// Account for grid span
			gridSpan := getCellGridSpan(&cell)
			cellIndex += gridSpan
		}
	}
	
	if len(columnsToHide) == 0 {
		return table, nil
	}
	
	logger.Debug("Columns to hide: %v", columnsToHide)
	
	// Create a new table with updated structure
	newTable := &Table{
		Properties: table.Properties,
		Grid:       processTableGrid(table.Grid, columnsToHide),
		Rows:       []TableRow{},
	}
	
	// Process each row to remove hidden columns
	for rowIdx, row := range table.Rows {
		logger.Debug("Processing row %d with %d cells", rowIdx, len(row.Cells))
		processedRow := processTableRow(&row, columnsToHide)
		if processedRow != nil {
			logger.Debug("Processed row %d has %d cells", rowIdx, len(processedRow.Cells))
			newTable.Rows = append(newTable.Rows, *processedRow)
		}
	}
	
	return newTable, nil
}

// getCellColumnMarker checks if a cell contains a column marker and returns it
func getCellColumnMarker(cell *TableCell) (*TableColumnMarker, string) {
	logger := GetLogger()
	for pIdx, para := range cell.Paragraphs {
		for rIdx, run := range para.Runs {
			if run.Text != nil {
				logger.Debug("Checking cell text in para %d, run %d: %q", pIdx, rIdx, run.Text.Content)
				if marker, strategy := parseColumnMarkerFromText(run.Text.Content); marker != nil {
					logger.Debug("Found column marker!")
					return marker, strategy
				}
			}
		}
	}
	return nil, ""
}

// parseColumnMarkerFromText parses a column marker from text
func parseColumnMarkerFromText(text string) (*TableColumnMarker, string) {
	if !strings.Contains(text, "TABLE_COLUMN_MARKER:hide:") {
		return nil, ""
	}
	
	// Parse marker format: {{TABLE_COLUMN_MARKER:hide:index}} or {{TABLE_COLUMN_MARKER:hide:index:strategy}}
	markerStart := strings.Index(text, "{{TABLE_COLUMN_MARKER:hide:")
	if markerStart == -1 {
		return nil, ""
	}
	
	markerEnd := strings.Index(text[markerStart:], "}}") + markerStart + 2
	if markerEnd <= markerStart {
		return nil, ""
	}
	
	marker := text[markerStart:markerEnd]
	parts := strings.Split(marker[2:len(marker)-2], ":")
	
	if len(parts) < 3 {
		return nil, ""
	}
	
	colIdx, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, ""
	}
	
	strategy := ""
	if len(parts) >= 4 {
		strategy = parts[3]
	}
	
	return &TableColumnMarker{
		Action:      "hide",
		ColumnIndex: colIdx,
	}, strategy
}

// processTableGrid processes the table grid to remove hidden columns
func processTableGrid(grid *TableGrid, columnsToHide map[int]string) *TableGrid {
	if grid == nil || len(columnsToHide) == 0 {
		return grid
	}
	
	// Extract column widths
	var oldWidths []int
	for _, col := range grid.Columns {
		oldWidths = append(oldWidths, col.Width)
	}
	
	// Calculate new widths based on resize strategy
	newWidths := calculateNewGridColumns(oldWidths, columnsToHide)
	
	// Create new grid
	newGrid := &TableGrid{
		Columns: []GridColumn{},
	}
	
	for _, width := range newWidths {
		newGrid.Columns = append(newGrid.Columns, GridColumn{
			Width: width,
		})
	}
	
	return newGrid
}

// calculateNewGridColumns calculates new column widths after hiding columns
func calculateNewGridColumns(oldWidths []int, columnsToHide map[int]string) []int {
	if len(columnsToHide) == 0 {
		return oldWidths
	}
	
	// Determine resize strategy (use first non-empty strategy found)
	strategy := "fixed" // default
	for _, s := range columnsToHide {
		if s != "" {
			strategy = s
			break
		}
	}
	
	// Calculate total width being removed
	totalRemoved := 0
	remainingCols := 0
	for i, width := range oldWidths {
		if _, hide := columnsToHide[i]; hide {
			totalRemoved += width
		} else {
			remainingCols++
		}
	}
	
	// Build new widths
	var newWidths []int
	
	switch strategy {
	case "redistribute":
		// Distribute removed width equally among remaining columns
		if remainingCols > 0 {
			extraPerCol := totalRemoved / remainingCols
			for i, width := range oldWidths {
				if _, hide := columnsToHide[i]; !hide {
					newWidths = append(newWidths, width+extraPerCol)
				}
			}
		}
		
	case "proportional":
		// Distribute removed width proportionally
		if remainingCols > 0 {
			totalRemaining := 0
			for i, width := range oldWidths {
				if _, hide := columnsToHide[i]; !hide {
					totalRemaining += width
				}
			}
			
			for i, width := range oldWidths {
				if _, hide := columnsToHide[i]; !hide {
					ratio := float64(width) / float64(totalRemaining)
					newWidth := width + int(float64(totalRemoved)*ratio)
					newWidths = append(newWidths, newWidth)
				}
			}
		}
		
	default: // "fixed"
		// Keep remaining column widths unchanged
		for i, width := range oldWidths {
			if _, hide := columnsToHide[i]; !hide {
				newWidths = append(newWidths, width)
			}
		}
	}
	
	return newWidths
}

// processTableRow processes a table row to remove hidden columns
func processTableRow(row *TableRow, columnsToHide map[int]string) *TableRow {
	logger := GetLogger()
	if row == nil || len(columnsToHide) == 0 {
		return row
	}
	
	newRow := &TableRow{
		Properties: row.Properties,
		Cells:      []TableCell{},
	}
	
	cellIndex := 0
	for cellNum, cell := range row.Cells {
		// Get grid span
		gridSpan := getCellGridSpan(&cell)
		
		// Check if any of the spanned columns should be hidden
		hideCell := false
		newGridSpan := gridSpan
		
		for col := cellIndex; col < cellIndex+gridSpan; col++ {
			if _, hide := columnsToHide[col]; hide {
				logger.Debug("Cell %d (cellIndex=%d, span=%d): column %d should be hidden", cellNum, cellIndex, gridSpan, col)
				hideCell = true
				newGridSpan--
			}
		}
		
		if !hideCell {
			// Keep the cell, but update grid span if necessary
			logger.Debug("Cell %d: keeping (not hidden)", cellNum)
			cellCopy := cell
			if newGridSpan != gridSpan && newGridSpan > 0 {
				cellCopy = updateCellGridSpan(&cell, newGridSpan)
			}
			
			// Clean any column markers from the cell
			cellCopy = cleanColumnMarkersFromCell(&cellCopy)
			
			newRow.Cells = append(newRow.Cells, cellCopy)
		} else if newGridSpan > 0 {
			// Partially hide merged cell
			logger.Debug("Cell %d: partially hiding (newSpan=%d)", cellNum, newGridSpan)
			cellCopy := updateCellGridSpan(&cell, newGridSpan)
			cellCopy = cleanColumnMarkersFromCell(&cellCopy)
			newRow.Cells = append(newRow.Cells, cellCopy)
		} else {
			logger.Debug("Cell %d: hiding completely", cellNum)
		}
		
		cellIndex += gridSpan
	}
	
	return newRow
}

// getCellGridSpan returns the grid span of a cell
func getCellGridSpan(cell *TableCell) int {
	if cell.Properties != nil && cell.Properties.GridSpan != nil {
		return cell.Properties.GridSpan.Val
	}
	return 1
}

// updateCellGridSpan updates the grid span of a cell
func updateCellGridSpan(cell *TableCell, newSpan int) TableCell {
	cellCopy := *cell
	
	if cellCopy.Properties == nil {
		cellCopy.Properties = &TableCellProperties{}
	}
	
	if newSpan > 1 {
		cellCopy.Properties.GridSpan = &GridSpan{Val: newSpan}
	} else {
		// Remove grid span if it's 1
		cellCopy.Properties.GridSpan = nil
	}
	
	return cellCopy
}

// cleanColumnMarkersFromCell removes column marker placeholders from a cell
func cleanColumnMarkersFromCell(cell *TableCell) TableCell {
	cellCopy := *cell
	
	// Clean markers from paragraphs
	for i, para := range cellCopy.Paragraphs {
		for j, run := range para.Runs {
			if run.Text != nil {
				cleanedText := removeColumnMarkerFromText(run.Text.Content)
				if cleanedText != run.Text.Content {
					cellCopy.Paragraphs[i].Runs[j].Text.Content = cleanedText
				}
			}
		}
	}
	
	return cellCopy
}

// removeColumnMarkerFromText removes column marker placeholders from text
func removeColumnMarkerFromText(text string) string {
	for strings.Contains(text, "{{TABLE_COLUMN_MARKER:") {
		start := strings.Index(text, "{{TABLE_COLUMN_MARKER:")
		end := strings.Index(text[start:], "}}") + start + 2
		if end > start {
			text = text[:start] + text[end:]
		} else {
			break
		}
	}
	return text
}


