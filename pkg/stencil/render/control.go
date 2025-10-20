package render

import (
	"strings"

	. "github.com/benjaminschreck/go-stencil/pkg/stencil/xml"
)

// DetectControlStructure checks if a paragraph contains a control structure
func DetectControlStructure(para *Paragraph) (string, string) {
	// Get the text content of the paragraph
	text := GetParagraphText(para)
	text = strings.TrimSpace(text)

	// Check for control structures
	if strings.Contains(text, "{{for ") && strings.Contains(text, "{{end}}") {
		// Handle inline for loop (e.g., "{{for item in items}} content {{end}}")
		return "inline-for", text
	}

	// Check for inline if statements
	if strings.Contains(text, "{{if ") && strings.Contains(text, "{{end}}") {
		// This paragraph contains a complete if statement
		// Let RenderParagraphWithContext handle it
		return "", ""
	}

	// Check for inline unless statements
	if strings.Contains(text, "{{unless ") && strings.Contains(text, "{{end}}") {
		// This paragraph contains a complete unless statement
		// Let RenderParagraphWithContext handle it
		return "", ""
	}

	// Check if text starts with a control structure (even if other content follows)
	if strings.HasPrefix(text, "{{for ") {
		// Extract just the for part
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[5:endIdx] // Remove {{for
			return "for", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{if ") {
		// Extract just the if part
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[4:endIdx] // Remove {{if
			return "if", strings.TrimSpace(content)
		}
	}

	// Check if text contains a for structure (not necessarily at the start)
	if idx := strings.Index(text, "{{for "); idx >= 0 && !strings.Contains(text, "{{end}}") {
		// Extract the for content
		startIdx := idx + 6 // Skip "{{for "
		endIdx := strings.Index(text[startIdx:], "}}")
		if endIdx > 0 {
			content := text[startIdx : startIdx+endIdx]
			return "for", strings.TrimSpace(content)
		}
	}

	// Check if text contains an if structure (not necessarily at the start)
	if idx := strings.Index(text, "{{if "); idx >= 0 && !strings.Contains(text, "{{end}}") {
		// Extract the if condition
		startIdx := idx + 5 // Skip "{{if "
		endIdx := strings.Index(text[startIdx:], "}}")
		if endIdx > 0 {
			content := text[startIdx : startIdx+endIdx]
			return "if", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{unless ") {
		// Extract just the unless part
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[8:endIdx] // Remove {{unless
			return "unless", strings.TrimSpace(content)
		}
	}

	// Check if text contains an unless structure (not necessarily at the start)
	if idx := strings.Index(text, "{{unless "); idx >= 0 && !strings.Contains(text, "{{end}}") {
		// Extract the unless condition
		startIdx := idx + 9 // Skip "{{unless "
		endIdx := strings.Index(text[startIdx:], "}}")
		if endIdx > 0 {
			content := text[startIdx : startIdx+endIdx]
			return "unless", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{end}}") {
		return "end", ""
	}

	if strings.HasPrefix(text, "{{else}}") {
		return "else", ""
	}

	// Check for elsif/elseif/elif variants
	if strings.HasPrefix(text, "{{elsif ") {
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[8:endIdx] // Remove {{elsif
			return "elsif", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{elseif ") {
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[9:endIdx] // Remove {{elseif
			return "elseif", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{elif ") {
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[7:endIdx] // Remove {{elif
			return "elif", strings.TrimSpace(content)
		}
	}

	// Check for include directive
	if strings.HasPrefix(text, "{{include ") {
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[10:endIdx] // Remove {{include
			return "include", strings.TrimSpace(content)
		}
	}

	return "", ""
}

// GetParagraphText extracts all text from a paragraph
func GetParagraphText(para *Paragraph) string {
	var result strings.Builder
	for _, run := range para.Runs {
		if run.Text != nil {
			result.WriteString(run.Text.Content)
		}
	}
	return result.String()
}

// FindMatchingEnd finds the position of the matching {{end}} for a control structure
// starting from the given position, counting depth
func FindMatchingEnd(text string, startPos int) int {
	depth := 1
	searchPos := startPos

	for depth > 0 {
		// Find next control structure marker
		nextFor := strings.Index(text[searchPos:], "{{for ")
		nextIf := strings.Index(text[searchPos:], "{{if ")
		nextUnless := strings.Index(text[searchPos:], "{{unless ")
		nextEnd := strings.Index(text[searchPos:], "{{end}}")

		if nextEnd < 0 {
			return -1 // No matching end found
		}

		// Adjust positions to absolute
		if nextFor >= 0 {
			nextFor += searchPos
		}
		if nextIf >= 0 {
			nextIf += searchPos
		}
		if nextUnless >= 0 {
			nextUnless += searchPos
		}
		nextEnd += searchPos

		// Find the earliest marker
		earliest := nextEnd
		if nextFor >= 0 && nextFor < earliest {
			earliest = nextFor
		}
		if nextIf >= 0 && nextIf < earliest {
			earliest = nextIf
		}
		if nextUnless >= 0 && nextUnless < earliest {
			earliest = nextUnless
		}

		if earliest == nextEnd {
			depth--
			if depth == 0 {
				return nextEnd
			}
			searchPos = nextEnd + 7 // Move past {{end}}
		} else {
			depth++
			// Move past the opening marker
			endOfMarker := strings.Index(text[earliest:], "}}") + earliest + 2
			searchPos = endOfMarker
		}
	}

	return -1
}
