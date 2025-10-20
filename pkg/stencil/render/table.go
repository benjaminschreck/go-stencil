package render

// This file contains table rendering helper functions extracted from render_docx.go
// These functions handle control structure detection and matching in table rows

import (
	"fmt"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/xml"
)

// DetectTableRowControlStructure detects control structures in the first cell of a table row
func DetectTableRowControlStructure(row *xml.TableRow) (string, string) {
	if len(row.Cells) == 0 || len(row.Cells[0].Paragraphs) == 0 {
		return "", ""
	}

	// Check first paragraph of first cell
	return DetectControlStructure(&row.Cells[0].Paragraphs[0])
}

// FindMatchingTableEnd finds the matching {{end}} for a control structure in table rows
func FindMatchingTableEnd(rows []xml.TableRow, startIdx int) (int, error) {
	depth := 1
	for i := startIdx + 1; i < len(rows); i++ {
		controlType, _ := DetectTableRowControlStructure(&rows[i])

		switch controlType {
		case "for", "if", "unless":
			depth++
		case "end":
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("no matching end found")
}

// FindMatchingTableIfEnd finds the matching else/elsif/end for a table if/unless
func FindMatchingTableIfEnd(rows []xml.TableRow, startIdx int) (endIdx int, branches []ElseBranch, error error) {
	depth := 1
	branches = []ElseBranch{}

	for i := startIdx + 1; i < len(rows); i++ {
		controlType, condition := DetectTableRowControlStructure(&rows[i])

		if depth == 1 {
			switch controlType {
			case "elsif", "elseif", "elif":
				branches = append(branches, ElseBranch{
					Index:      i,
					BranchType: "elsif",
					Condition:  condition,
				})
			case "else":
				branches = append(branches, ElseBranch{
					Index:      i,
					BranchType: "else",
					Condition:  "",
				})
			}
		}

		switch controlType {
		case "for", "if", "unless":
			depth++
		case "end":
			depth--
			if depth == 0 {
				return i, branches, nil
			}
		}
	}
	return -1, nil, fmt.Errorf("no matching end found")
}

// FindMatchingTableEndInSlice finds the matching {{end}} for a control structure in a slice of table rows
func FindMatchingTableEndInSlice(rows []xml.TableRow, startIdx int) (int, error) {
	depth := 1
	for i := startIdx + 1; i < len(rows); i++ {
		controlType, _ := DetectTableRowControlStructure(&rows[i])

		switch controlType {
		case "for", "if", "unless":
			depth++
		case "end":
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("no matching end found")
}

// FindMatchingTableIfEndInSlice finds the matching else/elsif/end for a table if/unless in a slice
func FindMatchingTableIfEndInSlice(rows []xml.TableRow, startIdx int) (endIdx int, branches []ElseBranch, error error) {
	depth := 1
	branches = []ElseBranch{}

	for i := startIdx + 1; i < len(rows); i++ {
		controlType, condition := DetectTableRowControlStructure(&rows[i])

		if depth == 1 {
			switch controlType {
			case "elsif", "elseif", "elif":
				branches = append(branches, ElseBranch{
					Index:      i,
					BranchType: "elsif",
					Condition:  condition,
				})
			case "else":
				branches = append(branches, ElseBranch{
					Index:      i,
					BranchType: "else",
					Condition:  "",
				})
			}
		}

		switch controlType {
		case "for", "if", "unless":
			depth++
		case "end":
			depth--
			if depth == 0 {
				return i, branches, nil
			}
		}
	}
	return -1, nil, fmt.Errorf("no matching end found")
}
