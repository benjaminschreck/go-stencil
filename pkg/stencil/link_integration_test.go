package stencil

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestLinkReplacementIntegration(t *testing.T) {
	// Create a test document with a hyperlink
	docXML := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<w:body>
		<w:p>
			<w:r><w:t>Visit our website at </w:t></w:r>
			<w:hyperlink r:id="rId2">
				<w:r><w:t>example.com</w:t></w:r>
			</w:hyperlink>
			<w:r><w:t> for more info. {{replaceLink(newUrl)}}</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`

	// Create test relationships
	_ = `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
	<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="http://example.com" TargetMode="External"/>
</Relationships>`

	// Parse the document
	doc := &Document{}
	if err := xml.Unmarshal([]byte(docXML), doc); err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// We'll pass relationships separately to the processing function

	// Create template data
	data := TemplateData{
		"newUrl": "https://new-example.com",
	}

	// Create render context
	ctx := &renderContext{
		imageMarkers: make(map[string]*imageReplacementMarker),
		linkMarkers:  make(map[string]*LinkReplacementMarker),
	}

	// Render the document
	rendered, err := RenderDocumentWithContext(doc, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render document: %v", err)
	}

	// Convert back to XML
	renderedXML, err := xml.Marshal(rendered)
	if err != nil {
		t.Fatalf("Failed to marshal rendered document: %v", err)
	}

	// Check that we collected link markers
	if len(ctx.linkMarkers) != 1 {
		t.Errorf("Expected 1 link marker, got %d", len(ctx.linkMarkers))
	}

	// Check the link marker content
	for _, marker := range ctx.linkMarkers {
		if marker.URL != "https://new-example.com" {
			t.Errorf("Expected URL 'https://new-example.com', got '%s'", marker.URL)
		}
	}

	// The rendered XML should contain a link replacement marker
	if !strings.Contains(string(renderedXML), "{{LINK_REPLACEMENT:") {
		t.Error("Rendered XML should contain link replacement marker")
	}
}

func TestProcessLinkReplacements(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<w:body>
		<w:p>
			<w:r><w:t>Check out </w:t></w:r>
			<w:hyperlink r:id="rId2">
				<w:r><w:t>our site</w:t></w:r>
			</w:hyperlink>
			<w:r><w:t>{{LINK_REPLACEMENT:link_0}}</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`

	linkMarkers := map[string]*LinkReplacementMarker{
		"link_0": {URL: "https://updated-site.com"},
	}

	// Initial relationships
	rels := []Relationship{
		{ID: "rId1", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles", Target: "styles.xml"},
		{ID: "rId2", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink", Target: "http://old-site.com", TargetMode: "External"},
	}

	// Process link replacements
	updatedXML, updatedRels, err := processLinkReplacements([]byte(xmlContent), linkMarkers, rels)
	if err != nil {
		t.Fatalf("Failed to process link replacements: %v", err)
	}

	// Check that the marker was removed
	if strings.Contains(string(updatedXML), "{{LINK_REPLACEMENT:") {
		t.Error("Link replacement marker should have been removed")
	}

	// Check that the hyperlink ID was updated
	if !strings.Contains(string(updatedXML), "r:id=\"rId3\"") {
		t.Error("Hyperlink should have been updated to new relationship ID")
	}

	// Check that new relationship was added
	if len(updatedRels) != 3 {
		t.Errorf("Expected 3 relationships after update, got %d", len(updatedRels))
	}

	// Find the new relationship
	var newRel *Relationship
	for i, rel := range updatedRels {
		if rel.Target == "https://updated-site.com" {
			newRel = &updatedRels[i]
			break
		}
	}

	if newRel == nil {
		t.Fatal("New relationship not found")
	}

	if newRel.Type != hyperlinkRelationType {
		t.Errorf("New relationship has wrong type: %s", newRel.Type)
	}

	if newRel.TargetMode != "External" {
		t.Errorf("New relationship should have External target mode, got: %s", newRel.TargetMode)
	}
}

func TestMultipleLinkReplacements(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
            xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<w:body>
		<w:p>
			<w:hyperlink r:id="rId2">
				<w:r><w:t>First link</w:t></w:r>
			</w:hyperlink>
			<w:r><w:t>{{LINK_REPLACEMENT:link_0}}</w:t></w:r>
		</w:p>
		<w:p>
			<w:hyperlink r:id="rId3">
				<w:r><w:t>Second link</w:t></w:r>
			</w:hyperlink>
			<w:r><w:t>{{LINK_REPLACEMENT:link_1}}</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`

	linkMarkers := map[string]*LinkReplacementMarker{
		"link_0": {URL: "https://first-new.com"},
		"link_1": {URL: "https://second-new.com"},
	}

	rels := []Relationship{
		{ID: "rId1", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles", Target: "styles.xml"},
		{ID: "rId2", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink", Target: "http://first.com", TargetMode: "External"},
		{ID: "rId3", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink", Target: "http://second.com", TargetMode: "External"},
	}

	updatedXML, updatedRels, err := processLinkReplacements([]byte(xmlContent), linkMarkers, rels)
	if err != nil {
		t.Fatalf("Failed to process link replacements: %v", err)
	}

	// Check that all markers were removed
	if strings.Contains(string(updatedXML), "{{LINK_REPLACEMENT:") {
		t.Error("All link replacement markers should have been removed")
	}

	// Check that we have the right number of relationships
	if len(updatedRels) != 5 { // 1 style + 2 original links + 2 new links
		t.Errorf("Expected 5 relationships, got %d", len(updatedRels))
	}

	// Verify both new URLs exist in relationships
	foundFirst := false
	foundSecond := false
	for _, rel := range updatedRels {
		if rel.Target == "https://first-new.com" {
			foundFirst = true
		}
		if rel.Target == "https://second-new.com" {
			foundSecond = true
		}
	}

	if !foundFirst {
		t.Error("First new URL not found in relationships")
	}
	if !foundSecond {
		t.Error("Second new URL not found in relationships")
	}
}