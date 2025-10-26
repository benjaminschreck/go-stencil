// test_helpers.go contains functions that are exposed only for testing purposes.
// These should not be used in production code.

package stencil

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// TestTemplate is a wrapper around the internal template type for testing
type TestTemplate struct {
	*template
}

// Parse creates a test template from a string (for testing only)
func Parse(filename string, content string) (*TestTemplate, error) {
	// Create a simple DOCX in memory
	docxBytes := createSimpleDOCXBytes(content)
	
	// Parse it
	reader := bytes.NewReader(docxBytes)
	docxReader, err := NewDocxReader(reader, int64(len(docxBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse DOCX: %w", err)
	}
	
	// Get document.xml
	docXML, err := docxReader.GetDocumentXML()
	if err != nil {
		return nil, fmt.Errorf("failed to get document.xml: %w", err)
	}
	
	// Parse document
	doc, err := ParseDocument(bytes.NewReader([]byte(docXML)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}
	
	return &TestTemplate{
		template: &template{
			docxReader: docxReader,
			document:   doc,
			source:     docxBytes,
			fragments:  make(map[string]*fragment),
		},
	}, nil
}

// ParseBytes creates a test template from DOCX bytes (for testing only)
func ParseBytes(docxBytes []byte) (*TestTemplate, error) {
	reader := bytes.NewReader(docxBytes)
	docxReader, err := NewDocxReader(reader, int64(len(docxBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse DOCX: %w", err)
	}
	
	// Get document.xml
	docXML, err := docxReader.GetDocumentXML()
	if err != nil {
		return nil, fmt.Errorf("failed to get document.xml: %w", err)
	}
	
	// Parse document
	doc, err := ParseDocument(bytes.NewReader([]byte(docXML)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}
	
	return &TestTemplate{
		template: &template{
			docxReader: docxReader,
			document:   doc,
			source:     docxBytes,
			fragments:  make(map[string]*fragment),
		},
	}, nil
}

// AddFragment adds a text fragment that can be included using {{include "name"}}
func (t *TestTemplate) AddFragment(name string, content string) error {
	// Parse the fragment content as a document
	parsed, err := ParseDocument(strings.NewReader(wrapInDocumentXML(content)))
	if err != nil {
		return fmt.Errorf("failed to parse fragment content: %w", err)
	}
	
	frag := &fragment{
		name:    name,
		content: content,
		parsed:  parsed,
		isDocx:  false,
	}
	
	t.fragments[name] = frag
	return nil
}

// AddFragmentFromBytes adds a DOCX fragment from raw bytes
func (t *TestTemplate) AddFragmentFromBytes(name string, docxBytes []byte) error {
	// Parse the DOCX fragment
	reader := bytes.NewReader(docxBytes)
	docxReader, err := NewDocxReader(reader, int64(len(docxBytes)))
	if err != nil {
		return fmt.Errorf("failed to parse fragment DOCX: %w", err)
	}
	
	// Get document.xml from the fragment
	docXML, err := docxReader.GetDocumentXML()
	if err != nil {
		return fmt.Errorf("failed to get fragment document.xml: %w", err)
	}
	
	// Parse the document
	doc, err := ParseDocument(bytes.NewReader([]byte(docXML)))
	if err != nil {
		return fmt.Errorf("failed to parse fragment document: %w", err)
	}

	// Extract styles.xml from fragment if it exists
	var stylesXML []byte
	zipReader, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err == nil {
		for _, file := range zipReader.File {
			if file.Name == "word/styles.xml" {
				rc, _ := file.Open()
				if rc != nil {
					stylesXML, _ = io.ReadAll(rc)
					rc.Close()
				}
				break
			}
		}
	}

	frag := &fragment{
		name:      name,
		parsed:    doc,
		isDocx:    true,
		docxData:  docxBytes,
		stylesXML: stylesXML,
	}

	t.fragments[name] = frag
	return nil
}

// Render renders the template with the given data (for simple text templates)
func (t *TestTemplate) Render(data map[string]interface{}) (string, error) {
	// Extract the text content from the document
	var content strings.Builder
	extractTextFromDocument(t.document, &content)
	
	// Process the template with fragments
	return ProcessTemplateWithFragments(content.String(), data, t.fragments)
}

// RenderToBytes renders the template to DOCX bytes
func (t *TestTemplate) RenderToBytes(data map[string]interface{}) ([]byte, error) {
	// Create a PreparedTemplate wrapper
	pt := &PreparedTemplate{template: t.template}
	
	// Use the full Render method
	reader, err := pt.Render(TemplateData(data))
	if err != nil {
		return nil, err
	}
	
	// Read all bytes
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, reader)
	if err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// Close releases resources held by the template
func (t *TestTemplate) Close() error {
	return t.template.Close()
}

// createDOCXWithNamespaces creates a DOCX with specified namespace declarations
func createDOCXWithNamespaces(namespaces map[string]string) []byte {
	return createDOCXWithContent("Test content", namespaces)
}

// createDOCXWithContent creates a DOCX with content and namespace declarations
func createDOCXWithContent(content string, namespaces map[string]string) []byte {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
	w, _ := zipWriter.Create("[Content_Types].xml")
	w.Write([]byte(contentTypes))

	// Add _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`
	w, _ = zipWriter.Create("_rels/.rels")
	w.Write([]byte(rels))

	// Add word/_rels/document.xml.rels
	docRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`
	w, _ = zipWriter.Create("word/_rels/document.xml.rels")
	w.Write([]byte(docRels))

	// Build namespace declarations
	nsDecls := `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`
	for prefix, uri := range namespaces {
		if prefix == "" {
			nsDecls += fmt.Sprintf(` xmlns="%s"`, uri)
		} else {
			nsDecls += fmt.Sprintf(` xmlns:%s="%s"`, prefix, uri)
		}
	}

	// Add word/document.xml with custom namespaces
	doc := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document %s>
  <w:body>
    <w:p>
      <w:r>
        <w:t>%s</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`, nsDecls, content)
	w, _ = zipWriter.Create("word/document.xml")
	w.Write([]byte(doc))

	zipWriter.Close()
	return buf.Bytes()
}