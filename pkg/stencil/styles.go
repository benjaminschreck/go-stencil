package stencil

import (
	"encoding/xml"
	"fmt"
)

// Styles represents the w:styles element in styles.xml
type Styles struct {
	XMLName      xml.Name        `xml:"styles"`
	Namespace    string          `xml:"xmlns:w,attr"`
	Styles       []DocumentStyle `xml:"style"`
	RawXML       []byte          `xml:",innerxml"` // Store the raw XML
}

// DocumentStyle represents a single w:style element (renamed to avoid conflict with table style)
type DocumentStyle struct {
	XMLName  xml.Name `xml:"style"`
	Type     string   `xml:"type,attr"`
	StyleID  string   `xml:"styleId,attr"`
	RawXML   []byte   `xml:",innerxml"` // Store the entire style definition as raw XML
}

// parseStyles parses a styles.xml file
func parseStyles(stylesXML []byte) (*Styles, error) {
	var styles Styles
	err := xml.Unmarshal(stylesXML, &styles)
	if err != nil {
		return nil, fmt.Errorf("failed to parse styles.xml: %w", err)
	}
	return &styles, nil
}

// mergeStyles merges fragment styles into the main styles
// It adds any styles from fragmentStyles that don't exist in mainStyles (by styleId)
func mergeStyles(mainStylesXML []byte, fragmentStylesXMLs ...[]byte) ([]byte, error) {
	// Parse main styles
	mainStyles, err := parseStyles(mainStylesXML)
	if err != nil {
		return nil, err
	}

	// Create a map of existing style IDs
	existingStyles := make(map[string]bool)
	for _, style := range mainStyles.Styles {
		existingStyles[style.StyleID] = true
	}

	// Collect new styles from fragments
	var newStyles []DocumentStyle
	for _, fragmentStylesXML := range fragmentStylesXMLs {
		fragmentStyles, err := parseStyles(fragmentStylesXML)
		if err != nil {
			// Skip if we can't parse
			continue
		}

		for _, style := range fragmentStyles.Styles {
			// Only add table styles that don't already exist
			if style.Type == "table" && !existingStyles[style.StyleID] {
				newStyles = append(newStyles, style)
				existingStyles[style.StyleID] = true
			}
		}
	}

	// If no new styles, return original
	if len(newStyles) == 0 {
		return mainStylesXML, nil
	}

	// Rebuild the styles.xml with new styles added
	return rebuildStylesXML(mainStylesXML, newStyles)
}

// rebuildStylesXML adds new styles to the existing styles.xml
func rebuildStylesXML(originalXML []byte, newStyles []DocumentStyle) ([]byte, error) {
	xmlStr := string(originalXML)

	// Find the closing </w:styles> tag
	closingTag := "</w:styles>"
	closingIndex := len(xmlStr) - len(closingTag) - 1

	// Find the last occurrence of </w:styles>
	for i := len(xmlStr) - 1; i >= 0; i-- {
		if i+len(closingTag) <= len(xmlStr) && xmlStr[i:i+len(closingTag)] == closingTag {
			closingIndex = i
			break
		}
	}

	// Build the new styles XML
	var newStylesXML string
	for _, style := range newStyles {
		// Reconstruct the style element
		styleXML := fmt.Sprintf(`<w:style w:type="%s" w:styleId="%s">%s</w:style>`,
			style.Type, style.StyleID, string(style.RawXML))
		newStylesXML += styleXML
	}

	// Insert new styles before the closing tag
	result := xmlStr[:closingIndex] + newStylesXML + xmlStr[closingIndex:]

	return []byte(result), nil
}

// extractTableStyleIDs extracts all table style IDs referenced in a document
func extractTableStyleIDs(doc *Document) []string {
	styleIDs := make(map[string]bool)

	if doc.Body == nil {
		return nil
	}

	for _, elem := range doc.Body.Elements {
		if table, ok := elem.(*Table); ok {
			if table.Properties != nil && table.Properties.Style != nil {
				if table.Properties.Style.Val != "" {
					styleIDs[table.Properties.Style.Val] = true
				}
			}
		}
	}

	// Convert map to slice
	var result []string
	for id := range styleIDs {
		result = append(result, id)
	}

	return result
}
