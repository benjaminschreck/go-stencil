package stencil

import (
	"strings"
	"testing"
)

func TestReplaceLinkDebug(t *testing.T) {
	// Create a simple document with replaceLink in the same paragraph as a hyperlink
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
		<w:body>
			<w:p>
				<w:r>
					<w:t>Update link: </w:t>
				</w:r>
				<w:hyperlink r:id="rId4" w:history="1">
					<w:r>
						<w:t>Click here</w:t>
					</w:r>
				</w:hyperlink>
				<w:r>
					<w:t> {{replaceLink(newURL)}}</w:t>
				</w:r>
			</w:p>
		</w:body>
	</w:document>`

	// Parse the document
	doc, err := ParseDocument(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	// Create test data
	data := TemplateData{
		"newURL": "https://github.com/benjaminschreck/go-stencil",
	}

	// Create render context
	ctx := &renderContext{
		linkMarkers:    make(map[string]*LinkReplacementMarker),
		ooxmlFragments: make(map[string]interface{}),
	}

	// Render the document
	renderedDoc, err := RenderDocumentWithContext(doc, data, ctx)
	if err != nil {
		t.Fatalf("RenderDocumentWithContext failed: %v", err)
	}

	// Check the paragraph
	para := renderedDoc.Body.Elements[0].(*Paragraph)
	
	// Log the structure
	t.Logf("Paragraph has %d content elements", len(para.Content))
	for i, content := range para.Content {
		switch c := content.(type) {
		case *Run:
			if c.Text != nil {
				t.Logf("  [%d] Run: %q", i, c.Text.Content)
			}
		case *Hyperlink:
			t.Logf("  [%d] Hyperlink: ID=%s", i, c.ID)
			for j, run := range c.Runs {
				if run.Text != nil {
					t.Logf("      [%d] Run: %q", j, run.Text.Content)
				}
			}
		}
	}

	// Get the full text
	fullText := para.GetText()
	t.Logf("Full paragraph text: %q", fullText)

	// Check if link marker was created
	t.Logf("Link markers created: %d", len(ctx.linkMarkers))
	for key, marker := range ctx.linkMarkers {
		t.Logf("  %s -> %s", key, marker.URL)
	}

	// Check if the marker is in the output
	if strings.Contains(fullText, "{{LINK_REPLACEMENT:") {
		t.Logf("Link replacement marker found in text: %s", fullText)
		
		// This is expected at the document level - the marker gets processed
		// during DOCX generation when we have access to relationships
	} else {
		t.Error("No link replacement marker found in output")
	}

	// Now let's simulate what happens during DOCX generation
	// Marshal the rendered document to XML
	renderedXML, err := marshalDocumentWithNamespaces(renderedDoc)
	if err != nil {
		t.Fatalf("Failed to marshal document: %v", err)
	}

	t.Logf("Document XML length: %d bytes", len(renderedXML))
	
	// Log the full XML to see structure
	xmlStr := string(renderedXML)
	t.Logf("Full XML:\n%s", xmlStr)

	// Create test relationships
	rels := []Relationship{
		{ID: "rId1", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles", Target: "styles.xml"},
		{ID: "rId4", Type: "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink", Target: "https://example.com", TargetMode: "External"},
	}

	// Process link replacements
	processedXML, updatedRels, err := processLinkReplacements(renderedXML, ctx.linkMarkers, rels)
	if err != nil {
		t.Fatalf("processLinkReplacements failed: %v", err)
	}

	t.Logf("Processed XML length: %d bytes", len(processedXML))
	
	// Check if the marker was removed
	if strings.Contains(string(processedXML), "{{LINK_REPLACEMENT:") {
		t.Error("Link replacement marker still present after processing")
		// Find the marker location
		idx := strings.Index(string(processedXML), "{{LINK_REPLACEMENT:")
		if idx >= 0 {
			start := idx - 50
			if start < 0 {
				start = 0
			}
			end := idx + 100
			if end > len(processedXML) {
				end = len(processedXML)
			}
			t.Logf("Context: %s", string(processedXML[start:end]))
		}
	} else {
		t.Log("✓ Link replacement marker was removed")
	}

	// Check if relationships were updated
	var updatedHyperlinkRel *Relationship
	for i, rel := range updatedRels {
		if rel.Type == "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" {
			t.Logf("Hyperlink relationship %d: ID=%s, Target=%s", i, rel.ID, rel.Target)
			if rel.ID == "rId4" || strings.HasPrefix(rel.ID, "rId") {
				updatedHyperlinkRel = &updatedRels[i]
			}
		}
	}

	if updatedHyperlinkRel != nil {
		if updatedHyperlinkRel.Target != "https://github.com/benjaminschreck/go-stencil" {
			t.Errorf("Hyperlink target not updated. Got %s, want %s", 
				updatedHyperlinkRel.Target, "https://github.com/benjaminschreck/go-stencil")
		} else {
			t.Log("✓ Hyperlink target was updated correctly")
		}
	} else {
		t.Error("No hyperlink relationship found after processing")
	}
}