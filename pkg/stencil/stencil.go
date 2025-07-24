package stencil

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"sync"
)

// template represents a parsed template document (internal use)
type template struct {
	docxReader *DocxReader
	document   *Document
	source     []byte
	fragments  map[string]*fragment
	closed     bool
	mu         sync.RWMutex
}

// fragment represents a reusable template fragment (internal use)
type fragment struct {
	name     string
	content  string
	parsed   *Document
	isDocx   bool
	docxData []byte
}

// renderContext holds the context during rendering (internal use)
type renderContext struct {
	linkMarkers      map[string]*LinkReplacementMarker
	fragments        map[string]*fragment
	fragmentStack    []string // Track fragment inclusion stack for circular reference detection
	renderDepth      int      // Track render depth to prevent excessive nesting
	ooxmlFragments   map[string]interface{} // Store OOXML fragments for later processing
}

// PreparedTemplate represents a compiled template ready for rendering.
// Use Prepare() or PrepareFile() to create an instance.
type PreparedTemplate struct {
	template *template
	closed   bool
	mu       sync.Mutex
	registry FunctionRegistry // Function registry to use during rendering
}

// TemplateData represents the data context for rendering templates.
// It's a map of key-value pairs where values can be strings, numbers,
// booleans, slices, maps, or any other type that can be accessed
// in template expressions.
//
// Example:
//
//	data := TemplateData{
//	    "name": "John Doe",
//	    "age": 30,
//	    "items": []map[string]interface{}{
//	        {"name": "Item 1", "price": 19.99},
//	        {"name": "Item 2", "price": 29.99},
//	    },
//	}
type TemplateData map[string]interface{}

// prepare is the internal implementation of template preparation
func prepare(r io.Reader) (*PreparedTemplate, error) {
	// Read the entire content into memory
	buf := new(bytes.Buffer)
	size, err := buf.ReadFrom(r)
	if err != nil {
		return nil, NewDocumentError("read", "", err)
	}
	
	// Create a copy of the buffer for DocxReader
	source := buf.Bytes()
	reader := bytes.NewReader(source)
	
	// Parse as DOCX
	docxReader, err := NewDocxReader(reader, size)
	if err != nil {
		return nil, NewDocumentError("parse", "DOCX", err)
	}
	
	// Parse document.xml
	docXML, err := docxReader.GetDocumentXML()
	if err != nil {
		return nil, NewDocumentError("extract", "document.xml", err)
	}
	
	doc, err := ParseDocument(bytes.NewReader([]byte(docXML)))
	if err != nil {
		return nil, NewParseError("document structure", "", 0)
	}
	
	
	tmpl := &template{
		docxReader: docxReader,
		document:   doc,
		source:     source,
		fragments:  make(map[string]*fragment),
	}
	
	return &PreparedTemplate{
		template: tmpl,
	}, nil
}



// Render executes the template with the given data and returns a reader
// containing the rendered DOCX file.
//
// The data parameter should contain all variables referenced in the template.
// Missing variables will be replaced with empty strings.
//
// Example:
//
//	data := TemplateData{
//	    "name": "John Doe",
//	    "date": time.Now(),
//	}
//	reader, err := template.Render(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (pt *PreparedTemplate) Render(data TemplateData) (io.Reader, error) {
	if pt == nil || pt.template == nil {
		return nil, NewTemplateError("invalid or nil template", 0, 0)
	}
	
	// Create a copy of the data to avoid modifying the original
	renderData := make(TemplateData)
	for k, v := range data {
		renderData[k] = v
	}
	
	// Inject the function registry if available and not already present
	if pt.registry != nil && renderData["__functions__"] == nil {
		renderData["__functions__"] = pt.registry
	}
	
	// Create render context
	renderCtx := &renderContext{
		linkMarkers:      make(map[string]*LinkReplacementMarker),
		fragments:        pt.template.fragments,
		fragmentStack:    make([]string, 0),
		renderDepth:      0,
		ooxmlFragments:   make(map[string]interface{}),
	}
	
	// First pass: render the document with variable substitution
	renderedDoc, err := RenderDocumentWithContext(pt.template.document, renderData, renderCtx)
	if err != nil {
		return nil, WithContext(err, "rendering document", map[string]interface{}{"hasData": data != nil})
	}
	
	// Process table row markers (hideRow() functions)
	err = ProcessTableRowMarkers(renderedDoc)
	if err != nil {
		return nil, WithContext(err, "processing table row markers", nil)
	}
	
	// Process table column markers (hideColumn() functions)
	err = ProcessTableColumnMarkers(renderedDoc)
	if err != nil {
		return nil, WithContext(err, "processing table column markers", nil)
	}
	
	// Convert the rendered document back to XML with proper namespaces
	renderedXML, err := marshalDocumentWithNamespaces(renderedDoc)
	if err != nil {
		return nil, NewDocumentError("marshal", "rendered document", err)
	}
	
	// Process link replacements if any
	var updatedRelationships []Relationship
	if len(renderCtx.linkMarkers) > 0 {
		// Get current relationships
		relsXML, err := pt.template.docxReader.GetRelationshipsXML()
		if err != nil {
			return nil, NewDocumentError("extract", "relationships", err)
		}
		currentRels := parseRelationships([]byte(relsXML))
		
		// Process link replacements
		renderedXML, updatedRelationships, err = processLinkReplacements(renderedXML, renderCtx.linkMarkers, currentRels)
		if err != nil {
			return nil, fmt.Errorf("failed to process link replacements: %w", err)
		}
	}
	
	// Create a new DOCX with the rendered content
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	
	// Copy all parts from the original DOCX
	reader := bytes.NewReader(pt.template.source)
	zipReader, err := zip.NewReader(reader, int64(len(pt.template.source)))
	if err != nil {
		return nil, fmt.Errorf("failed to read source zip: %w", err)
	}
	
	for _, file := range zipReader.File {
		// Special handling for document.xml
		if file.Name == "word/document.xml" {
			// Write the rendered document XML
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write(renderedXML)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if file.Name == "word/_rels/document.xml.rels" && len(updatedRelationships) > 0 {
			// Update relationships if we have link replacements
			output, err := xml.MarshalIndent(&Relationships{
				Namespace: "http://schemas.openxmlformats.org/package/2006/relationships",
				Relationship: updatedRelationships,
			}, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal relationships: %w", err)
			}
			rels := output
			
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write(rels)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else {
			// Copy other files as-is
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			
			fr, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %w", file.Name, err)
			}
			
			_, err = io.Copy(fw, fr)
			fr.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to copy %s: %w", file.Name, err)
			}
		}
	}
	
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}
	
	return bytes.NewReader(buf.Bytes()), nil
}

// Close releases any resources held by the prepared template.
// After calling Close, the template should not be used.
func (pt *PreparedTemplate) Close() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.closed {
		return nil
	}

	pt.closed = true

	// Close the underlying template
	if pt.template != nil {
		return pt.template.Close()
	}

	return nil
}


// AddFragment adds a text fragment that can be included in the template
// using the {{include "name"}} syntax.
//
// Fragments are useful for reusable content like headers, footers, or
// standard paragraphs. The content should be plain text; it will be
// wrapped in appropriate DOCX structure automatically.
//
// Example:
//
//	err := template.AddFragment("disclaimer", "This is confidential information.")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Then in your template: {{include "disclaimer"}}
func (pt *PreparedTemplate) AddFragment(name string, content string) error {
	if pt == nil || pt.template == nil {
		return fmt.Errorf("invalid template")
	}
	
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
	
	pt.template.fragments[name] = frag
	return nil
}

// AddFragmentFromBytes adds a DOCX fragment from raw bytes.
// This allows including pre-formatted DOCX content with styling, tables,
// images, etc. The fragment should be a complete DOCX file.
//
// Example:
//
//	fragmentBytes, err := os.ReadFile("header.docx")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = template.AddFragmentFromBytes("header", fragmentBytes)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Then in your template: {{include "header"}}
func (pt *PreparedTemplate) AddFragmentFromBytes(name string, docxBytes []byte) error {
	if pt == nil || pt.template == nil {
		return fmt.Errorf("invalid template")
	}
	
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
	
	pt.template.fragments[name] = frag
	return nil
}

// wrapInDocumentXML wraps plain text content in minimal document XML structure
func wrapInDocumentXML(content string) string {
	// Escape XML special characters
	content = strings.ReplaceAll(content, "&", "&amp;")
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	content = strings.ReplaceAll(content, "\"", "&quot;")
	content = strings.ReplaceAll(content, "'", "&apos;")
	
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>` + content + `</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`
}


// extractTextFromDocument extracts all text content from a document
func extractTextFromDocument(doc *Document, result *strings.Builder) {
	if doc.Body != nil {
		extractTextFromBody(doc.Body, result)
	}
}

// extractTextFromBody extracts text from document body
func extractTextFromBody(body *Body, result *strings.Builder) {
	for _, elem := range body.Elements {
		if para, ok := elem.(*Paragraph); ok {
			extractTextFromParagraph(para, result)
		}
	}
}

// extractTextFromParagraph extracts text from a paragraph
func extractTextFromParagraph(para *Paragraph, result *strings.Builder) {
	for _, run := range para.Runs {
		if run.Text != nil {
			result.WriteString(run.Text.Content)
		}
	}
}

// Close releases resources held by the template
func (t *template) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if t.closed {
		return nil
	}
	
	t.closed = true
	
	// Clear references to allow garbage collection
	// Note: We keep source as it may be needed for rendering
	t.fragments = nil
	
	// Close the DocxReader if it has a Close method
	// Note: We keep docxReader as it may be needed for rendering
	
	return nil
}





// createSimpleDOCXBytes creates a minimal DOCX file with the given content
func createSimpleDOCXBytes(content string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	
	// Add _rels/.rels
	rels, _ := w.Create("_rels/.rels")
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)
	
	// Add word/_rels/document.xml.rels
	wordRels, _ := w.Create("word/_rels/document.xml.rels")
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`)
	
	// Add word/document.xml
	doc, _ := w.Create("word/document.xml")
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>`+content+`</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`)
	
	// Add [Content_Types].xml
	ct, _ := w.Create("[Content_Types].xml")
	io.WriteString(ct, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`)
	
	w.Close()
	return buf.Bytes()
}

