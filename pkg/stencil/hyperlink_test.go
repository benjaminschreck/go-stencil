package stencil

import (
	"strings"
	"testing"
)

func TestHyperlinkParsing(t *testing.T) {
	tests := []struct {
		name        string
		xml         string
		wantRuns    int
		wantHyperlinks int
		wantText    string
	}{
		{
			name: "paragraph with hyperlink",
			xml: `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
				<w:body>
					<w:p>
						<w:r>
							<w:t>Visit our </w:t>
						</w:r>
						<w:hyperlink r:id="rId4" w:history="1">
							<w:r>
								<w:t>website</w:t>
							</w:r>
						</w:hyperlink>
						<w:r>
							<w:t> for more info.</w:t>
						</w:r>
					</w:p>
				</w:body>
			</w:document>`,
			wantRuns:    2,
			wantHyperlinks: 1,
			wantText:    "Visit our website for more info.",
		},
		{
			name: "paragraph with multiple hyperlinks",
			xml: `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
				<w:body>
					<w:p>
						<w:r>
							<w:t>Check </w:t>
						</w:r>
						<w:hyperlink r:id="rId1">
							<w:r>
								<w:t>GitHub</w:t>
							</w:r>
						</w:hyperlink>
						<w:r>
							<w:t> and </w:t>
						</w:r>
						<w:hyperlink r:id="rId2">
							<w:r>
								<w:t>Documentation</w:t>
							</w:r>
						</w:hyperlink>
					</w:p>
				</w:body>
			</w:document>`,
			wantRuns:    2,
			wantHyperlinks: 2,
			wantText:    "Check GitHub and Documentation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseDocument(strings.NewReader(tt.xml))
			if err != nil {
				t.Fatalf("ParseDocument failed: %v", err)
			}

			if doc.Body == nil || len(doc.Body.Elements) == 0 {
				t.Fatal("No body elements found")
			}

			para, ok := doc.Body.Elements[0].(*Paragraph)
			if !ok {
				t.Fatal("First element is not a paragraph")
			}

			// Check runs count
			if len(para.Runs) != tt.wantRuns {
				t.Errorf("Got %d runs, want %d", len(para.Runs), tt.wantRuns)
			}

			// Check hyperlinks count
			if len(para.Hyperlinks) != tt.wantHyperlinks {
				t.Errorf("Got %d hyperlinks, want %d", len(para.Hyperlinks), tt.wantHyperlinks)
			}

			// Check content preservation
			if len(para.Content) != tt.wantRuns+tt.wantHyperlinks {
				t.Errorf("Got %d content elements, want %d", len(para.Content), tt.wantRuns+tt.wantHyperlinks)
			}

			// Check text
			gotText := para.GetText()
			if gotText != tt.wantText {
				t.Errorf("Got text %q, want %q", gotText, tt.wantText)
			}
		})
	}
}

func TestHyperlinkRendering(t *testing.T) {
	xml := `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
		<w:body>
			<w:p>
				<w:r>
					<w:t xml:space="preserve">Hello {{name}}, visit </w:t>
				</w:r>
				<w:hyperlink r:id="rId4" w:history="1">
					<w:r>
						<w:t>{{siteName}}</w:t>
					</w:r>
				</w:hyperlink>
				<w:r>
					<w:t xml:space="preserve"> today!</w:t>
				</w:r>
			</w:p>
		</w:body>
	</w:document>`

	doc, err := ParseDocument(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}
	
	// Debug: Check what was parsed
	para := doc.Body.Elements[0].(*Paragraph)
	t.Logf("Parsed paragraph has %d Content elements, %d Runs, %d Hyperlinks", 
		len(para.Content), len(para.Runs), len(para.Hyperlinks))
	
	for i, content := range para.Content {
		switch c := content.(type) {
		case *Run:
			t.Logf("Parsed Content[%d]: Run with text %q", i, c.GetText())
		case *Hyperlink:
			t.Logf("Parsed Content[%d]: Hyperlink with text %q", i, c.GetText())
		}
	}

	data := TemplateData{
		"name":     "John",
		"siteName": "GitHub",
	}

	// Create a new render context for debugging
	ctx := &renderContext{}
	rendered, err := RenderDocumentWithContext(doc, data, ctx)
	if err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	renderedPara := rendered.Body.Elements[0].(*Paragraph)
	
	// Check that hyperlink is preserved
	if len(renderedPara.Hyperlinks) != 1 {
		t.Errorf("Got %d hyperlinks after render, want 1", len(renderedPara.Hyperlinks))
	}

	// Check that variables in hyperlink are rendered
	if len(renderedPara.Hyperlinks) > 0 {
		hyperlink := renderedPara.Hyperlinks[0]
		if len(hyperlink.Runs) != 1 {
			t.Errorf("Hyperlink has %d runs, want 1", len(hyperlink.Runs))
		} else if hyperlink.Runs[0].Text.Content != "GitHub" {
			t.Errorf("Hyperlink text is %q, want %q", hyperlink.Runs[0].Text.Content, "GitHub")
		}
	}

	// Debug: Show what's in the paragraph
	t.Logf("Rendered paragraph has %d Content elements, %d Runs, %d Hyperlinks", 
		len(renderedPara.Content), len(renderedPara.Runs), len(renderedPara.Hyperlinks))
	
	for i, content := range renderedPara.Content {
		switch c := content.(type) {
		case *Run:
			t.Logf("Content[%d]: Run with text %q", i, c.GetText())
		case *Hyperlink:
			t.Logf("Content[%d]: Hyperlink with text %q", i, c.GetText())
		}
	}
	
	// Debug: Show legacy arrays
	t.Logf("Legacy Runs:")
	for i, run := range renderedPara.Runs {
		t.Logf("  Run[%d]: %q", i, run.GetText())
	}
	t.Logf("Legacy Hyperlinks:")
	for i, hyperlink := range renderedPara.Hyperlinks {
		t.Logf("  Hyperlink[%d]: %q", i, hyperlink.GetText())
	}
	
	// Check full text
	gotText := renderedPara.GetText()
	wantText := "Hello John, visit GitHub today!"
	if gotText != wantText {
		t.Errorf("Got text %q, want %q", gotText, wantText)
	}
}