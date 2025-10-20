package render

// This file contains body rendering functions extracted from render_docx.go
// These functions handle control structures (if/for/unless) at the document body level

import (
	"fmt"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/xml"
)

// ElseBranch represents an else/elsif branch in an if statement
type ElseBranch struct {
	Index      int    // Index of the branch paragraph
	BranchType string // "else", "elsif", "elif", "elseif"
	Condition  string // Condition for elsif branches
}

// FindMatchingEndInElements finds the matching {{end}} for a control structure in elements
func FindMatchingEndInElements(elements []xml.BodyElement, startIdx int) (int, error) {
	depth := 1
	for i := startIdx + 1; i < len(elements); i++ {
		if para, ok := elements[i].(*xml.Paragraph); ok {
			controlType, _ := DetectControlStructure(para)
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
	}
	return -1, fmt.Errorf("no matching end found")
}

// FindIfStructureInElements finds the structure of an if statement including elsif/else branches
func FindIfStructureInElements(elements []xml.BodyElement, startIdx int) (endIdx int, branches []ElseBranch, err error) {
	depth := 1
	branches = []ElseBranch{}

	for i := startIdx + 1; i < len(elements); i++ {
		if para, ok := elements[i].(*xml.Paragraph); ok {
			controlType, condition := DetectControlStructure(para)

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
	}

	return -1, nil, fmt.Errorf("no matching end found")
}
