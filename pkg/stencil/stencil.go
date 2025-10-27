package stencil

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	// ID ranges for relationship management
	MainTemplateIDRange  = 999  // Main template uses rId1 - rId999
	FragmentIDRangeSize  = 100  // Each fragment gets 100 IDs
	FragmentIDRangeStart = 1000 // Fragments start at rId1000
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
	name          string
	content       string
	parsed        *Document
	isDocx        bool
	docxData      []byte
	mediaFiles    map[string][]byte // filename -> content
	relationships []Relationship    // from word/_rels/document.xml.rels
	stylesXML     []byte            // from word/styles.xml
	namespaces    map[string]string // prefix -> URI, extracted from fragment document
}

// renderContext holds the context during rendering (internal use)
type renderContext struct {
	linkMarkers    map[string]*LinkReplacementMarker
	fragments      map[string]*fragment
	fragmentStack  []string               // Track fragment inclusion stack for circular reference detection
	renderDepth    int                    // Track render depth to prevent excessive nesting
	ooxmlFragments map[string]interface{} // Store OOXML fragments for later processing

	// Fragment resource tracking
	fragmentMedia          map[string][]byte // remapped filename -> content
	fragmentRelationships  []Relationship    // relationships to add
	fragmentIDAllocations  map[string]int    // fragment name -> allocated range start
	nextFragmentIDRange    int               // next available range start
	fragmentResourcesAdded map[string]bool   // fragment name -> already added

	// Namespace collection
	collectedNamespaces map[string]string // prefix -> URI, collected from all fragments
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

// cleanEmptyRuns removes empty run elements from a paragraph
// Empty runs (with no text content) can cause Word to fail opening the document
func cleanEmptyRuns(para *Paragraph) {
	if para == nil {
		return
	}

	// Filter out runs that have no text or empty text
	var nonEmptyRuns []Run
	for _, run := range para.Runs {
		// Determine if this run should be kept
		keepRun := false

		// Keep run if it has text content (even if just whitespace that should be preserved)
		if run.Text != nil && run.Text.Content != "" {
			keepRun = true
		}

		// Keep runs with breaks or raw XML elements (like drawings) even if they have no text
		if run.Break != nil || len(run.RawXML) > 0 {
			keepRun = true
		}

		// Skip runs with only properties - they can cause issues in headers/footers
		// Word will inherit paragraph properties anyway
		if !keepRun && run.Properties != nil {
			keepRun = false
		}

		if keepRun {
			nonEmptyRuns = append(nonEmptyRuns, run)
		}
		// Otherwise skip completely empty runs
	}

	para.Runs = nonEmptyRuns
}

// renderHeaderOrFooter processes a header or footer XML file with template rendering
func renderHeaderOrFooter(file *zip.File, data TemplateData, ctx *renderContext) ([]byte, error) {
	// Read the original header/footer XML
	fr, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", file.Name, err)
	}
	defer fr.Close()

	content, err := io.ReadAll(fr)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", file.Name, err)
	}

	// Parse using a generic structure that handles both <w:hdr> and <w:ftr>
	// These have the same structure as document body - just paragraphs and tables
	var headerFooter struct {
		XMLName    xml.Name     `xml:""`
		Attrs      []xml.Attr   `xml:",any,attr"`
		Paragraphs []*Paragraph `xml:"p"`
		Tables     []*Table     `xml:"tbl"`
	}

	if err := xml.Unmarshal(content, &headerFooter); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", file.Name, err)
	}

	// Convert paragraphs and tables to BodyElements
	var elements []BodyElement
	for _, p := range headerFooter.Paragraphs {
		elements = append(elements, p)
	}
	for _, t := range headerFooter.Tables {
		elements = append(elements, t)
	}

	// Render the elements with context
	renderedElements, err := renderElementsWithContext(elements, data, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to render elements in %s: %w", file.Name, err)
	}

	// Separate rendered elements back into paragraphs and tables
	var renderedParas []*Paragraph
	var renderedTables []*Table
	for _, elem := range renderedElements {
		switch e := elem.(type) {
		case *Paragraph:
			renderedParas = append(renderedParas, e)
		case *Table:
			renderedTables = append(renderedTables, e)
		}
	}

	// Clean empty runs from paragraphs to avoid Word corruption issues
	for _, p := range renderedParas {
		cleanEmptyRuns(p)
	}

	// Also clean empty runs from table cells
	for _, t := range renderedTables {
		for i := range t.Rows {
			for j := range t.Rows[i].Cells {
				for k := range t.Rows[i].Cells[j].Paragraphs {
					cleanEmptyRuns(&t.Rows[i].Cells[j].Paragraphs[k])
				}
			}
		}
	}

	headerFooter.Paragraphs = renderedParas
	headerFooter.Tables = renderedTables

	// Create a map to track runs with RawXML that need to be injected
	rawXMLMap := make(map[string][]byte)
	markerIndex := 0

	// Replace RawXML with markers before marshaling
	for _, p := range renderedParas {
		for i := range p.Runs {
			if len(p.Runs[i].RawXML) > 0 {
				// Create a marker
				marker := fmt.Sprintf("__RAWXML_MARKER_%d__", markerIndex)
				markerIndex++

				// Collect all RawXML content
				var rawContent bytes.Buffer
				for _, raw := range p.Runs[i].RawXML {
					rawContent.Write(raw.Content)
				}

				// Store the raw XML
				rawXMLMap[marker] = rawContent.Bytes()

				// Replace RawXML with a text marker
				p.Runs[i].Text = &Text{Content: marker}
				p.Runs[i].RawXML = nil
			}
		}
	}

	// Marshal the body elements
	var bodyXML bytes.Buffer
	encoder := xml.NewEncoder(&bodyXML)
	for _, p := range renderedParas {
		if err := encoder.Encode(p); err != nil {
			return nil, fmt.Errorf("failed to encode paragraph: %w", err)
		}
	}
	for _, t := range renderedTables {
		if err := encoder.Encode(t); err != nil {
			return nil, fmt.Errorf("failed to encode table: %w", err)
		}
	}
	encoder.Flush()

	// Replace markers with actual RawXML
	bodyXMLStr := bodyXML.String()
	for marker, rawXML := range rawXMLMap {
		// The marker will be inside <w:t>marker</w:t>, replace the entire text element
		bodyXMLStr = strings.Replace(bodyXMLStr,
			fmt.Sprintf("<w:t>%s</w:t>", marker),
			string(rawXML),
			1)
	}
	bodyXML.Reset()
	bodyXML.WriteString(bodyXMLStr)

	// Extract the opening tag with namespaces from the original content
	contentStr := string(content)

	// Skip the XML declaration if present
	xmlDeclEnd := strings.Index(contentStr, "?>")
	searchStart := 0
	if xmlDeclEnd != -1 && strings.HasPrefix(strings.TrimSpace(contentStr), "<?xml") {
		searchStart = xmlDeclEnd + 2
	}

	// Find the opening root tag (starts after XML declaration)
	rootTagStart := strings.Index(contentStr[searchStart:], "<")
	if rootTagStart == -1 {
		return nil, fmt.Errorf("malformed XML: no root tag found")
	}
	rootTagStart += searchStart

	// Find the end of the opening tag
	openTagEnd := strings.Index(contentStr[rootTagStart:], ">")
	if openTagEnd == -1 {
		return nil, fmt.Errorf("malformed XML: no opening tag end found")
	}
	openTagEnd += rootTagStart

	// Find where the closing tag starts
	var closingTag string
	if strings.Contains(contentStr, "</w:hdr>") {
		closingTag = "</w:hdr>"
	} else if strings.Contains(contentStr, "</w:ftr>") {
		closingTag = "</w:ftr>"
	} else {
		return nil, fmt.Errorf("unknown root element type")
	}

	// Reconstruct: XML declaration + original opening tag + rendered body + closing tag
	result := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	result = append(result, content[rootTagStart:openTagEnd+1]...) // Opening tag with namespaces
	result = append(result, bodyXML.Bytes()...)                    // Rendered body
	result = append(result, []byte(closingTag)...)                 // Closing tag

	return result, nil
}

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
		linkMarkers:            make(map[string]*LinkReplacementMarker),
		fragments:              pt.template.fragments,
		fragmentStack:          make([]string, 0),
		renderDepth:            0,
		ooxmlFragments:         make(map[string]interface{}),
		fragmentMedia:          make(map[string][]byte),
		fragmentRelationships:  make([]Relationship, 0),
		fragmentIDAllocations:  make(map[string]int),
		nextFragmentIDRange:    FragmentIDRangeStart,
		fragmentResourcesAdded: make(map[string]bool),
		collectedNamespaces:    make(map[string]string),
	}

	// Collect namespaces from the main template document (V5: REQUIRED)
	if pt.template.document != nil {
		mainNamespaces := pt.template.document.ExtractNamespaces()
		for prefix, uri := range mainNamespaces {
			renderCtx.collectedNamespaces[prefix] = uri
		}
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

	// V5: Merge collected namespaces from fragments into main document
	if len(renderCtx.collectedNamespaces) > 0 {
		renderedDoc.MergeNamespaces(renderCtx.collectedNamespaces)
	}

	// Convert the rendered document back to XML with proper namespaces
	// NOTE: This uses the existing marshalDocumentWithNamespaces which writes doc.Attrs!
	renderedXML, err := marshalDocumentWithNamespaces(renderedDoc)
	if err != nil {
		return nil, NewDocumentError("marshal", "rendered document", err)
	}

	// Process link replacements and fragment relationships
	var updatedRelationships []Relationship
	needsRelationshipUpdate := len(renderCtx.linkMarkers) > 0 || len(renderCtx.fragmentRelationships) > 0

	if needsRelationshipUpdate {
		// Get current relationships
		relsXML, err := pt.template.docxReader.GetRelationshipsXML()
		if err != nil {
			return nil, NewDocumentError("extract", "relationships", err)
		}
		currentRels := parseRelationships([]byte(relsXML))

		// Process link replacements if any
		if len(renderCtx.linkMarkers) > 0 {
			renderedXML, updatedRelationships, err = processLinkReplacements(renderedXML, renderCtx.linkMarkers, currentRels)
			if err != nil {
				return nil, fmt.Errorf("failed to process link replacements: %w", err)
			}
		} else {
			updatedRelationships = currentRels
		}

		// Add fragment relationships
		if len(renderCtx.fragmentRelationships) > 0 {
			updatedRelationships = append(updatedRelationships, renderCtx.fragmentRelationships...)
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

	// Track if we need to update Content Types for fragment media
	var contentTypes *ContentTypes
	hasFragmentMedia := len(renderCtx.fragmentMedia) > 0

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
		} else if matched, _ := regexp.MatchString(`^word/header\d+\.xml$`, file.Name); matched {
			// Process header files
			renderedHeader, err := renderHeaderOrFooter(file, renderData, renderCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to render %s: %w", file.Name, err)
			}
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write(renderedHeader)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if matched, _ := regexp.MatchString(`^word/footer\d+\.xml$`, file.Name); matched {
			// Process footer files
			renderedFooter, err := renderHeaderOrFooter(file, renderData, renderCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to render %s: %w", file.Name, err)
			}
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write(renderedFooter)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if file.Name == "word/_rels/document.xml.rels" && len(updatedRelationships) > 0 {
			// Update relationships if we have link replacements or fragment relationships
			// Use Marshal (not MarshalIndent) to produce compact XML like the original
			output, err := xml.Marshal(&Relationships{
				Namespace:    "http://schemas.openxmlformats.org/package/2006/relationships",
				Relationship: updatedRelationships,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to marshal relationships: %w", err)
			}

			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}

			// Write XML header with standalone="yes" (required by Word)
			_, err = fw.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"))
			if err != nil {
				return nil, fmt.Errorf("failed to write XML header to %s: %w", file.Name, err)
			}

			// Write relationships XML
			_, err = fw.Write(output)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if file.Name == "word/styles.xml" {
			// Merge styles from fragments
			fr, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %w", file.Name, err)
			}
			mainStyles, err := io.ReadAll(fr)
			fr.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", file.Name, err)
			}

			// Collect fragment styles
			var fragmentStyles [][]byte
			for _, frag := range pt.template.fragments {
				if frag.isDocx && len(frag.stylesXML) > 0 {
					fragmentStyles = append(fragmentStyles, frag.stylesXML)
				}
			}

			// Merge styles if we have fragments with styles
			mergedStyles := mainStyles
			if len(fragmentStyles) > 0 {
				mergedStyles, err = mergeStyles(mainStyles, fragmentStyles...)
				if err != nil {
					// If merge fails, use original styles
					mergedStyles = mainStyles
				}
			}

			// Write merged styles
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write(mergedStyles)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if file.Name == "[Content_Types].xml" && hasFragmentMedia {
			// Parse Content Types to potentially update it
			fr, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %w", file.Name, err)
			}
			ctContent, err := io.ReadAll(fr)
			fr.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", file.Name, err)
			}

			contentTypes = &ContentTypes{}
			err = xml.Unmarshal(ctContent, contentTypes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", file.Name, err)
			}

			// We'll write this later after ensuring PNG is registered
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

	// Add fragment media files
	for filename, content := range renderCtx.fragmentMedia {
		fw, err := w.Create("word/media/" + filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create fragment media file %s: %w", filename, err)
		}
		_, err = fw.Write(content)
		if err != nil {
			return nil, fmt.Errorf("failed to write fragment media file %s: %w", filename, err)
		}
	}

	// Write updated Content Types if we added fragment media
	if contentTypes != nil {
		// Collect all extensions from fragment media
		mediaExtensions := make(map[string]bool)
		for filename := range renderCtx.fragmentMedia {
			ext := strings.TrimPrefix(filepath.Ext(filename), ".")
			if ext != "" {
				mediaExtensions[strings.ToLower(ext)] = true
			}
		}

		// Build map of already registered extensions
		registeredExtensions := make(map[string]bool)
		for _, def := range contentTypes.Defaults {
			registeredExtensions[strings.ToLower(def.Extension)] = true
		}

		// Add missing extensions with their content types
		extensionContentTypes := map[string]string{
			"png":  "image/png",
			"jpg":  "image/jpeg",
			"jpeg": "image/jpeg",
			"gif":  "image/gif",
			"bmp":  "image/bmp",
			"tiff": "image/tiff",
			"tif":  "image/tiff",
			"svg":  "image/svg+xml",
			"webp": "image/webp",
			"emf":  "image/x-emf",
			"wmf":  "image/x-wmf",
		}

		for ext := range mediaExtensions {
			if !registeredExtensions[ext] {
				contentType, ok := extensionContentTypes[ext]
				if !ok {
					// Default to generic image type for unknown extensions
					contentType = "image/" + ext
				}
				contentTypes.Defaults = append(contentTypes.Defaults, ContentTypeDefault{
					Extension:   ext,
					ContentType: contentType,
				})
			}
		}

		// Set namespace if not present
		if contentTypes.Namespace == "" {
			contentTypes.Namespace = "http://schemas.openxmlformats.org/package/2006/content-types"
		}

		// Marshal and write Content Types
		output, err := xml.Marshal(contentTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Content Types: %w", err)
		}

		fw, err := w.Create("[Content_Types].xml")
		if err != nil {
			return nil, fmt.Errorf("failed to create [Content_Types].xml: %w", err)
		}

		_, err = fw.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"))
		if err != nil {
			return nil, fmt.Errorf("failed to write XML header to [Content_Types].xml: %w", err)
		}

		_, err = fw.Write(output)
		if err != nil {
			return nil, fmt.Errorf("failed to write [Content_Types].xml: %w", err)
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
// etc. The fragment should be a complete DOCX file.
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

	// Extract namespaces from parsed document
	namespaces := doc.ExtractNamespaces()

	// Extract media files and relationships from fragment ZIP
	zipReader, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		return fmt.Errorf("failed to read fragment as ZIP: %w", err)
	}

	// Extract media files
	mediaFiles := make(map[string][]byte)
	for _, file := range zipReader.File {
		if strings.HasPrefix(file.Name, "word/media/") {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open media file %s: %w", file.Name, err)
			}

			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return fmt.Errorf("failed to read media file %s: %w", file.Name, err)
			}

			// Store with path relative to word/ (e.g., "media/image1.png")
			relativePath := strings.TrimPrefix(file.Name, "word/")
			mediaFiles[relativePath] = content
		}
	}

	// Extract relationships and styles
	var relationships []Relationship
	var stylesXML []byte
	for _, file := range zipReader.File {
		if file.Name == "word/_rels/document.xml.rels" {
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open relationships: %w", err)
			}

			relsData, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return fmt.Errorf("failed to read relationships: %w", err)
			}

			var rels Relationships
			err = xml.Unmarshal(relsData, &rels)
			if err != nil {
				return fmt.Errorf("failed to parse relationships: %w", err)
			}

			relationships = rels.Relationship
		} else if file.Name == "word/styles.xml" {
			rc, err := file.Open()
			if err != nil {
				continue // styles.xml is optional
			}

			stylesXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				stylesXML = nil
			}
		}
	}

	frag := &fragment{
		name:          name,
		parsed:        doc,
		isDocx:        true,
		docxData:      docxBytes,
		mediaFiles:    mediaFiles,
		relationships: relationships,
		stylesXML:     stylesXML,
		namespaces:    namespaces,
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
