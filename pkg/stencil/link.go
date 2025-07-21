package stencil

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

const hyperlinkRelationType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink"

type hyperlinkInfo struct {
	RelationshipID string
}

type linkReplacement struct {
	OldRelID string
	NewRelID string
	URL      string
}

func findHyperlinks(decoder *xml.Decoder) []hyperlinkInfo {
	var hyperlinks []hyperlinkInfo
	
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		
		switch t := token.(type) {
		case xml.StartElement:
			// Check if this is a w:hyperlink element
			if t.Name.Local == "hyperlink" {
				// Look for r:id attribute
				for _, attr := range t.Attr {
					if attr.Name.Local == "id" {
						hyperlinks = append(hyperlinks, hyperlinkInfo{
							RelationshipID: attr.Value,
						})
						break
					}
				}
			}
		}
	}
	
	return hyperlinks
}

func parseRelationships(data []byte) []Relationship {
	var rels Relationships
	if err := xml.Unmarshal(data, &rels); err != nil {
		return nil
	}
	return rels.Relationship
}

func filterHyperlinkRelationships(rels []Relationship) []Relationship {
	var linkRels []Relationship
	for _, rel := range rels {
		if rel.Type == hyperlinkRelationType {
			linkRels = append(linkRels, rel)
		}
	}
	return linkRels
}

func generateNewRelationshipID(rels []Relationship) string {
	maxID := 0
	for _, rel := range rels {
		if strings.HasPrefix(rel.ID, "rId") {
			numStr := strings.TrimPrefix(rel.ID, "rId")
			if num, err := strconv.Atoi(numStr); err == nil && num > maxID {
				maxID = num
			}
		}
	}
	return fmt.Sprintf("rId%d", maxID+1)
}

func addHyperlinkRelationship(rels *[]Relationship, url string) Relationship {
	newRel := Relationship{
		ID:         generateNewRelationshipID(*rels),
		Type:       hyperlinkRelationType,
		Target:     url,
		TargetMode: "External",
	}
	*rels = append(*rels, newRel)
	return newRel
}

func updateHyperlinkID(xmlContent string, oldID, newID string) (string, error) {
	// Use simple string replacement for hyperlink ID updates
	oldPattern := `r:id="` + oldID + `"`
	newPattern := `r:id="` + newID + `"`
	
	if !strings.Contains(xmlContent, oldPattern) {
		return "", fmt.Errorf("hyperlink with ID %s not found", oldID)
	}
	
	// Replace the old ID with the new ID
	result := strings.Replace(xmlContent, oldPattern, newPattern, -1)
	
	return result, nil
}

