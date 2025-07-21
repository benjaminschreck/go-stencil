package stencil

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestFindHyperlinks(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected []hyperlinkInfo
	}{
		{
			name: "Single hyperlink",
			xml: `<w:p>
				<w:r><w:t>Before link</w:t></w:r>
				<w:hyperlink r:id="rId2">
					<w:r><w:t>Link text</w:t></w:r>
				</w:hyperlink>
				<w:r><w:t>After link</w:t></w:r>
			</w:p>`,
			expected: []hyperlinkInfo{
				{RelationshipID: "rId2"},
			},
		},
		{
			name: "Multiple hyperlinks",
			xml: `<w:p>
				<w:hyperlink r:id="rId2">
					<w:r><w:t>First link</w:t></w:r>
				</w:hyperlink>
				<w:r><w:t> and </w:t></w:r>
				<w:hyperlink r:id="rId3">
					<w:r><w:t>Second link</w:t></w:r>
				</w:hyperlink>
			</w:p>`,
			expected: []hyperlinkInfo{
				{RelationshipID: "rId2"},
				{RelationshipID: "rId3"},
			},
		},
		{
			name: "Hyperlink with attributes",
			xml: `<w:p>
				<w:hyperlink r:id="rId4" w:tgtFrame="_blank" w:tooltip="Click here">
					<w:r><w:t>Complex link</w:t></w:r>
				</w:hyperlink>
			</w:p>`,
			expected: []hyperlinkInfo{
				{RelationshipID: "rId4"},
			},
		},
		{
			name:     "No hyperlinks",
			xml:      `<w:p><w:r><w:t>Just plain text</w:t></w:r></w:p>`,
			expected: []hyperlinkInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := xml.NewDecoder(strings.NewReader(tt.xml))
			hyperlinks := findHyperlinks(decoder)

			if len(hyperlinks) != len(tt.expected) {
				t.Errorf("Expected %d hyperlinks, got %d", len(tt.expected), len(hyperlinks))
				return
			}

			for i, h := range hyperlinks {
				if h.RelationshipID != tt.expected[i].RelationshipID {
					t.Errorf("Hyperlink %d: expected relationship ID %s, got %s",
						i, tt.expected[i].RelationshipID, h.RelationshipID)
				}
			}
		})
	}
}

func TestParseHyperlinkRelationships(t *testing.T) {
	relationshipsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
	<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="http://example.com" TargetMode="External"/>
	<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://golang.org" TargetMode="External"/>
</Relationships>`

	rels := parseRelationships([]byte(relationshipsXML))

	linkRels := filterHyperlinkRelationships(rels)
	if len(linkRels) != 2 {
		t.Errorf("Expected 2 hyperlink relationships, got %d", len(linkRels))
	}

	expectedLinks := map[string]string{
		"rId2": "http://example.com",
		"rId3": "https://golang.org",
	}

	for _, rel := range linkRels {
		expected, ok := expectedLinks[rel.ID]
		if !ok {
			t.Errorf("Unexpected relationship ID: %s", rel.ID)
			continue
		}
		if rel.Target != expected {
			t.Errorf("Relationship %s: expected target %s, got %s", rel.ID, expected, rel.Target)
		}
		if rel.TargetMode != "External" {
			t.Errorf("Relationship %s: expected TargetMode 'External', got %s", rel.ID, rel.TargetMode)
		}
	}
}

func TestGenerateNewRelationshipID(t *testing.T) {
	existingRels := []Relationship{
		{ID: "rId1"},
		{ID: "rId2"},
		{ID: "rId3"},
		{ID: "rId5"}, // Gap in numbering
	}

	newID := generateNewRelationshipID(existingRels)
	if newID != "rId6" {
		t.Errorf("Expected 'rId6', got '%s'", newID)
	}

	// Test with empty relationships
	emptyRels := []Relationship{}
	newID = generateNewRelationshipID(emptyRels)
	if newID != "rId1" {
		t.Errorf("Expected 'rId1' for empty relationships, got '%s'", newID)
	}
}

func TestAddHyperlinkRelationship(t *testing.T) {
	rels := []Relationship{
		{ID: "rId1", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles", Target: "styles.xml"},
	}

	newRel := addHyperlinkRelationship(&rels, "https://example.com")

	if len(rels) != 2 {
		t.Errorf("Expected 2 relationships after addition, got %d", len(rels))
	}

	if newRel.ID != "rId2" {
		t.Errorf("Expected new relationship ID 'rId2', got '%s'", newRel.ID)
	}

	if newRel.Type != hyperlinkRelationType {
		t.Errorf("Expected hyperlink relationship type, got '%s'", newRel.Type)
	}

	if newRel.Target != "https://example.com" {
		t.Errorf("Expected target 'https://example.com', got '%s'", newRel.Target)
	}

	if newRel.TargetMode != "External" {
		t.Errorf("Expected TargetMode 'External', got '%s'", newRel.TargetMode)
	}
}

func TestUpdateHyperlinkID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		oldID       string
		newID       string
		expected    string
		shouldError bool
	}{
		{
			name:     "Simple hyperlink update",
			input:    `<w:hyperlink r:id="rId2"><w:r><w:t>Link</w:t></w:r></w:hyperlink>`,
			oldID:    "rId2",
			newID:    "rId5",
			expected: `<w:hyperlink r:id="rId5"><w:r><w:t>Link</w:t></w:r></w:hyperlink>`,
		},
		{
			name:     "Hyperlink with multiple attributes",
			input:    `<w:hyperlink r:id="rId3" w:tgtFrame="_blank"><w:r><w:t>Link</w:t></w:r></w:hyperlink>`,
			oldID:    "rId3",
			newID:    "rId6",
			expected: `<w:hyperlink r:id="rId6" w:tgtFrame="_blank"><w:r><w:t>Link</w:t></w:r></w:hyperlink>`,
		},
		{
			name:        "Non-matching ID",
			input:       `<w:hyperlink r:id="rId2"><w:r><w:t>Link</w:t></w:r></w:hyperlink>`,
			oldID:       "rId3",
			newID:       "rId5",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := updateHyperlinkID(tt.input, tt.oldID, tt.newID)
			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}