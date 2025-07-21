// test_helpers.go contains functions that are exposed only for testing purposes.
// These should not be used in production code.

package stencil

import (
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
	
	frag := &fragment{
		name:     name,
		parsed:   doc,
		isDocx:   true,
		docxData: docxBytes,
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