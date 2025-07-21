package stencil

import (
	"fmt"
	"regexp"
	"strings"
)

// processLinkReplacements processes link replacement markers in the document
// and returns updated XML and relationships
func processLinkReplacements(xmlContent []byte, linkMarkers map[string]*LinkReplacementMarker, relationships []Relationship) ([]byte, []Relationship, error) {
	if len(linkMarkers) == 0 {
		return xmlContent, relationships, nil
	}

	// Make a copy of relationships to avoid modifying the original
	updatedRels := make([]Relationship, len(relationships))
	copy(updatedRels, relationships)

	// Convert XML content to string for processing
	content := string(xmlContent)

	// Find all link replacement markers and their preceding hyperlinks
	markerPattern := regexp.MustCompile(`{{LINK_REPLACEMENT:(link_\d+)}}`)
	hyperlinkPattern := regexp.MustCompile(`<w:hyperlink[^>]+r:id="([^"]+)"[^>]*>`)

	matches := markerPattern.FindAllStringSubmatchIndex(content, -1)
	
	// Process from end to beginning to maintain string positions
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		markerStart := match[0]
		markerEnd := match[1]
		markerKey := content[match[2]:match[3]]

		marker, ok := linkMarkers[markerKey]
		if !ok {
			continue
		}

		// Find the preceding hyperlink
		precedingContent := content[:markerStart]
		hyperlinkMatches := hyperlinkPattern.FindAllStringSubmatch(precedingContent, -1)
		
		if len(hyperlinkMatches) == 0 {
			return nil, nil, fmt.Errorf("no preceding hyperlink found for marker %s", markerKey)
		}

		// Get the last (closest) hyperlink
		lastHyperlink := hyperlinkMatches[len(hyperlinkMatches)-1]
		oldRelID := lastHyperlink[1]

		// Create new relationship
		newRel := addHyperlinkRelationship(&updatedRels, marker.URL)

		// Update the hyperlink ID in the content
		// Find the exact position of this hyperlink
		hyperlinkPos := strings.LastIndex(precedingContent, lastHyperlink[0])
		if hyperlinkPos == -1 {
			return nil, nil, fmt.Errorf("could not find hyperlink position")
		}

		// Replace the old relationship ID with the new one
		oldHyperlinkTag := lastHyperlink[0]
		newHyperlinkTag := strings.Replace(oldHyperlinkTag, `r:id="`+oldRelID+`"`, `r:id="`+newRel.ID+`"`, 1)
		
		// Build the updated content
		content = content[:hyperlinkPos] + newHyperlinkTag + content[hyperlinkPos+len(oldHyperlinkTag):markerStart] + content[markerEnd:]
	}

	return []byte(content), updatedRels, nil
}

