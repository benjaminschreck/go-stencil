package stencil

import (
	"strings"
	"testing"
)

func TestReplaceLinkFunctionality(t *testing.T) {
	// Create a document with a hyperlink that contains replaceLink
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
		<w:body>
			<w:p>
				<w:r>
					<w:t>Visit our </w:t>
				</w:r>
				<w:hyperlink r:id="rId4" w:history="1">
					<w:r>
						<w:t>website{{replaceLink(url)}}</w:t>
					</w:r>
				</w:hyperlink>
				<w:r>
					<w:t> for more info.</w:t>
				</w:r>
			</w:p>
		</w:body>
	</w:document>`

	// Parse the document
	doc, err := ParseDocument(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	// Create test data with a URL
	data := TemplateData{
		"url": "https://github.com/benjaminschreck/go-stencil",
	}

	// Since we're testing at a lower level, we need to simulate what would happen
	// during normal template processing. The replaceLink function generates a marker
	// that gets processed during rendering.
	
	// First, let's render the document to see what happens
	ctx := &renderContext{
		linkMarkers: make(map[string]*LinkReplacementMarker),
		ooxmlFragments: make(map[string]interface{}),
	}
	
	renderedDoc, err := RenderDocumentWithContext(doc, data, ctx)
	if err != nil {
		t.Fatalf("RenderDocumentWithContext failed: %v", err)
	}

	// Check that the hyperlink is preserved
	para := renderedDoc.Body.Elements[0].(*Paragraph)
	if len(para.Hyperlinks) != 1 {
		t.Errorf("Got %d hyperlinks, want 1", len(para.Hyperlinks))
	}

	// Check that the hyperlink text was rendered correctly (without the replaceLink marker)
	if len(para.Hyperlinks) > 0 {
		hyperlink := para.Hyperlinks[0]
		if len(hyperlink.Runs) != 1 {
			t.Errorf("Hyperlink has %d runs, want 1", len(hyperlink.Runs))
		} else {
			// The hyperlink text will contain the marker at the document level
			// This is expected - the marker gets processed later in the DOCX rendering pipeline
			gotText := hyperlink.Runs[0].Text.Content
			if !strings.HasPrefix(gotText, "website{{LINK_REPLACEMENT:") {
				t.Errorf("Hyperlink text is %q, expected to start with %q", gotText, "website{{LINK_REPLACEMENT:")
			}
		}
		
		// The hyperlink ID should remain the same for now
		if hyperlink.ID != "rId4" {
			t.Errorf("Hyperlink ID changed from rId4 to %s", hyperlink.ID)
		}
	}

	// Check if link markers were created
	if len(ctx.linkMarkers) > 0 {
		t.Logf("Link markers created: %d", len(ctx.linkMarkers))
		for key, marker := range ctx.linkMarkers {
			t.Logf("  %s: %s", key, marker.URL)
		}
	} else {
		t.Log("No link markers were created - replaceLink may not be working correctly")
	}

	// Check the full paragraph text
	// At the document level, the marker is still present
	gotText := para.GetText()
	if !strings.Contains(gotText, "{{LINK_REPLACEMENT:") {
		t.Errorf("Expected paragraph text to contain link replacement marker, got %q", gotText)
	}
	
	// Verify that hyperlinks are preserved in the structure
	if len(para.Content) != 3 {
		t.Errorf("Expected 3 content elements (run, hyperlink, run), got %d", len(para.Content))
	}
}