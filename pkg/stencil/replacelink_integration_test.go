package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReplaceLinkIntegration(t *testing.T) {
	// Create a template with a hyperlink followed by replaceLink in the same paragraph
	templateXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<w:body>
		<w:p>
			<w:r>
				<w:t>Update link: </w:t>
			</w:r>
			<w:hyperlink r:id="rId4">
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

	// Create relationships XML
	relsXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
	<Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com" TargetMode="External"/>
</Relationships>`

	// Create a minimal DOCX in memory
	docx := createMinimalDocx(map[string][]byte{
		"word/document.xml":           []byte(templateXML),
		"word/_rels/document.xml.rels": []byte(relsXML),
	})

	// Prepare the template
	tmpl, err := Prepare(bytes.NewReader(docx))
	if err != nil {
		t.Fatalf("Failed to prepare template: %v", err)
	}
	defer tmpl.Close()

	// Render with data
	data := TemplateData{
		"newURL": "https://github.com/benjaminschreck/go-stencil",
	}

	output, err := tmpl.Render(data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Read the output
	outputBytes, err := io.ReadAll(output)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	// Extract document.xml from output
	outputDoc, outputRels := extractFromDocx(t, outputBytes)

	// Check that replaceLink marker is gone
	if strings.Contains(outputDoc, "replaceLink") {
		t.Error("replaceLink marker still present in output")
		t.Logf("Document XML: %s", outputDoc)
	}

	// Check that the hyperlink still exists and uses the correct relationship
	if !strings.Contains(outputDoc, "<w:hyperlink") {
		t.Error("Hyperlink missing from output")
	}
	
	// Extract the relationship ID from the hyperlink
	hyperlinkIdx := strings.Index(outputDoc, "<w:hyperlink")
	if hyperlinkIdx >= 0 {
		endIdx := strings.Index(outputDoc[hyperlinkIdx:], ">")
		if endIdx >= 0 {
			hyperlinkTag := outputDoc[hyperlinkIdx:hyperlinkIdx+endIdx+1]
			t.Logf("Hyperlink tag: %s", hyperlinkTag)
			
			// Check which relationship ID is being used
			if strings.Contains(hyperlinkTag, "rId4") {
				t.Error("Hyperlink still uses old relationship ID (rId4)")
			}
			if strings.Contains(hyperlinkTag, "rId5") {
				t.Log("✓ Hyperlink correctly uses new relationship ID (rId5)")
			}
		}
	}

	// Parse relationships to check if URL was updated
	var foundNewURL bool
	
	lines := strings.Split(outputRels, "\n")
	for _, line := range lines {
		if strings.Contains(line, "hyperlink") {
			if strings.Contains(line, "github.com/benjaminschreck/go-stencil") {
				foundNewURL = true
				t.Logf("Found new URL in relationships: %s", line)
			}
			if strings.Contains(line, "example.com") {
				t.Logf("Old URL still exists: %s", line)
			}
		}
	}

	if !foundNewURL {
		t.Error("New URL not found in relationships")
	}

	// It's OK if the old URL still exists (as a different relationship)
	// The important thing is that the hyperlink uses the new relationship
}

func createMinimalDocx(files map[string][]byte) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	
	// Add required files for a minimal DOCX
	requiredFiles := map[string][]byte{
		"[Content_Types].xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
	<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
	<Default Extension="xml" ContentType="application/xml"/>
	<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`),
		"_rels/.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`),
	}
	
	// Add required files
	for name, content := range requiredFiles {
		f, _ := w.Create(name)
		f.Write(content)
	}
	
	// Add user-provided files
	for name, content := range files {
		f, _ := w.Create(name)
		f.Write(content)
	}
	
	w.Close()
	return buf.Bytes()
}

func extractFromDocx(t *testing.T, docxBytes []byte) (string, string) {
	r, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("Failed to read output DOCX: %v", err)
	}
	
	var documentXML, relsXML string
	
	for _, f := range r.File {
		switch f.Name {
		case "word/document.xml":
			rc, _ := f.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			documentXML = string(content)
		case "word/_rels/document.xml.rels":
			rc, _ := f.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			relsXML = string(content)
		}
	}
	
	return documentXML, relsXML
}