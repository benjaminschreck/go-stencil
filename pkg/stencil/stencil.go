package stencil

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	// ID ranges for relationship management
	MainTemplateIDRange     = 999  // Main template uses rId1 - rId999
	FragmentIDRangeSize     = 100  // Each fragment gets 100 IDs
	FragmentIDRangeStart    = 1000 // Fragments start at rId1000
	numberedParagraphAnchor = "\u200B"
)

// template represents a parsed template document (internal use)
type template struct {
	docxReader      *DocxReader
	document        *Document
	source          []byte
	fragments       map[string]*fragment
	renderResources *templateRenderResources
	closed          bool
	mu              sync.RWMutex
}

type templateRenderResources struct {
	mainNamespaces        map[string]string
	mainStylesXML         []byte
	baseNumbering         *numberingContext
	fragmentFontOverrides map[string]fragmentFontOverrides
	mergedStylesCache     map[string][]byte
	bodyPlans             map[*Body]*bodyRenderPlan
	cacheMu               sync.RWMutex
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
	numberingXML  []byte            // from word/numbering.xml
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
	usedDocxFragments      map[string]bool   // docx fragments included during this render
	numbering              *numberingContext
	fragmentFontOverrides  map[string]fragmentFontOverrides

	// Namespace collection
	collectedNamespaces map[string]string // prefix -> URI, collected from all fragments
	bodyPlans           map[*Body]*bodyRenderPlan
}

// PreparedTemplate represents a compiled template ready for rendering.
// Use Prepare() or PrepareFile() to create an instance.
type PreparedTemplate struct {
	state    *preparedTemplateState
	template *template
	closed   bool
	mu       sync.RWMutex
	registry FunctionRegistry // Function registry to use during rendering
}

type preparedTemplateState struct {
	mu       sync.Mutex
	template *template
	refs     int
}

func newPreparedTemplateState(tmpl *template) *preparedTemplateState {
	return &preparedTemplateState{
		template: tmpl,
		refs:     1,
	}
}

func (s *preparedTemplateState) retain() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.refs == 0 {
		return false
	}
	s.refs++
	return true
}

func (s *preparedTemplateState) release() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	if s.refs == 0 {
		s.mu.Unlock()
		return nil
	}
	s.refs--
	if s.refs > 0 {
		s.mu.Unlock()
		return nil
	}
	tmpl := s.template
	s.template = nil
	s.mu.Unlock()

	if tmpl != nil {
		return tmpl.Close()
	}
	return nil
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

	if _, err := tmpl.ensureRenderResources(); err != nil {
		return nil, err
	}

	return &PreparedTemplate{
		state:    newPreparedTemplateState(tmpl),
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

func hasDirectParagraphNumbering(para *Paragraph) bool {
	if para == nil || para.Properties == nil {
		return false
	}

	for _, raw := range para.Properties.RawXML {
		if raw.XMLName.Local == "numPr" {
			return true
		}
	}

	return false
}

func paragraphHasInlineContent(para *Paragraph) bool {
	if para == nil {
		return false
	}

	for _, run := range para.Runs {
		if run.Text != nil && run.Text.Content != "" {
			return true
		}
		if run.Break != nil || len(run.RawXML) > 0 {
			return true
		}
	}

	if len(para.Hyperlinks) > 0 {
		return true
	}

	for _, item := range para.Content {
		switch c := item.(type) {
		case *Run:
			if c.Text != nil && c.Text.Content != "" {
				return true
			}
			if c.Break != nil || len(c.RawXML) > 0 {
				return true
			}
		case *Hyperlink:
			return true
		}
	}

	return false
}

func ensureNumberedParagraphAnchor(para *Paragraph) {
	if !hasDirectParagraphNumbering(para) || paragraphHasInlineContent(para) {
		return
	}

	anchor := Run{
		Text: &Text{Content: numberedParagraphAnchor},
	}
	if para.Properties != nil && para.Properties.RunProperties != nil {
		anchor.Properties = para.Properties.RunProperties
	}

	para.Runs = append(para.Runs, anchor)
	if para.Content != nil {
		anchorCopy := anchor
		para.Content = append(para.Content, &anchorCopy)
	}
}

func normalizeRenderedParagraph(para *Paragraph) {
	cleanEmptyRuns(para)
	ensureNumberedParagraphAnchor(para)
}

func normalizeRenderedTable(table *Table) {
	if table == nil {
		return
	}

	for i := range table.Rows {
		for j := range table.Rows[i].Cells {
			for k := range table.Rows[i].Cells[j].Paragraphs {
				normalizeRenderedParagraph(&table.Rows[i].Cells[j].Paragraphs[k])
			}
		}
	}
}

func normalizeRenderedBodyElements(elements []BodyElement) {
	for _, elem := range elements {
		switch e := elem.(type) {
		case *Paragraph:
			normalizeRenderedParagraph(e)
		case *Table:
			normalizeRenderedTable(e)
		}
	}
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

	// Check if this header/footer contains any template markers
	// If not, return it as-is to preserve all XML structure and namespaces
	originalContent := string(content)
	if !strings.Contains(originalContent, "{{") && !strings.Contains(originalContent, "}}") {
		return content, nil
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

	// Normalize rendered paragraphs so empty Word list items still render reliably.
	for _, p := range renderedParas {
		normalizeRenderedParagraph(p)
	}
	for _, t := range renderedTables {
		normalizeRenderedTable(t)
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
	if pt == nil {
		return nil, NewTemplateError("invalid or nil template", 0, 0)
	}
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.closed || pt.template == nil {
		return nil, NewTemplateError("template is closed", 0, 0)
	}

	tmpl := pt.template
	registry := pt.registry

	// Create a copy of the data to avoid modifying the original
	renderData := make(TemplateData)
	for k, v := range data {
		renderData[k] = v
	}

	// Inject the function registry if available and not already present
	if registry != nil && renderData["__functions__"] == nil {
		renderData["__functions__"] = registry
	}

	resources, err := tmpl.ensureRenderResources()
	if err != nil {
		return nil, fmt.Errorf("failed to prepare template render resources: %w", err)
	}

	// Create render context
	numberingCtx := resources.baseNumbering.clone()

	renderCtx := &renderContext{
		linkMarkers:            make(map[string]*LinkReplacementMarker),
		fragments:              tmpl.fragments,
		fragmentStack:          make([]string, 0),
		renderDepth:            0,
		ooxmlFragments:         make(map[string]interface{}),
		fragmentMedia:          make(map[string][]byte),
		fragmentRelationships:  make([]Relationship, 0),
		fragmentIDAllocations:  make(map[string]int),
		nextFragmentIDRange:    FragmentIDRangeStart,
		fragmentResourcesAdded: make(map[string]bool),
		usedDocxFragments:      make(map[string]bool),
		numbering:              numberingCtx,
		fragmentFontOverrides:  resources.fragmentFontOverrides,
		collectedNamespaces:    make(map[string]string),
		bodyPlans:              resources.bodyPlans,
	}

	// Collect namespaces from the main template document (V5: REQUIRED)
	for prefix, uri := range resources.mainNamespaces {
		renderCtx.collectedNamespaces[prefix] = uri
	}

	// First pass: render the document with variable substitution
	renderedDoc, err := RenderDocumentWithContext(tmpl.document, renderData, renderCtx)
	if err != nil {
		return nil, WithContext(err, "rendering document", map[string]interface{}{"hasData": data != nil})
	}
	if renderedDoc != nil && renderedDoc.Body != nil {
		normalizeRenderedBodyElements(renderedDoc.Body.Elements)
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
	if renderCtx.numbering != nil && renderCtx.numbering.needsRelationship() {
		needsRelationshipUpdate = true
	}

	if needsRelationshipUpdate {
		// Get current relationships
		relsXML, err := tmpl.docxReader.GetRelationshipsXML()
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

		if renderCtx.numbering != nil && renderCtx.numbering.needsRelationship() {
			updatedRelationships = append(updatedRelationships, Relationship{
				ID:     generateNewRelationshipID(updatedRelationships),
				Type:   numberingRelationType,
				Target: "numbering.xml",
			})
		}
	}

	// Create a new DOCX with the rendered content
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	w.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestSpeed)
	})

	// Copy all parts from the original DOCX
	reader := bytes.NewReader(tmpl.source)
	zipReader, err := zip.NewReader(reader, int64(len(tmpl.source)))
	if err != nil {
		return nil, fmt.Errorf("failed to read source zip: %w", err)
	}

	renderedHeaderFooterParts := make(map[string][]byte)
	for _, file := range zipReader.File {
		if !isHeaderPartName(file.Name) && !isFooterPartName(file.Name) {
			continue
		}

		renderedPart, err := renderHeaderOrFooter(file, renderData, renderCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to render %s: %w", file.Name, err)
		}
		renderedHeaderFooterParts[file.Name] = renderedPart
	}

	// Track if we need to update Content Types for fragment media
	var contentTypes *ContentTypes
	hasFragmentMedia := len(renderCtx.fragmentMedia) > 0
	needsNumberingPartWrite := renderCtx.numbering != nil && renderCtx.numbering.modified
	needsContentTypesUpdate := hasFragmentMedia
	if renderCtx.numbering != nil && renderCtx.numbering.needsContentTypeOverride() {
		needsContentTypesUpdate = true
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
		} else if isHeaderPartName(file.Name) {
			renderedHeader, ok := renderedHeaderFooterParts[file.Name]
			if !ok {
				return nil, fmt.Errorf("missing pre-rendered header part %s", file.Name)
			}
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write(renderedHeader)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if isFooterPartName(file.Name) {
			renderedFooter, ok := renderedHeaderFooterParts[file.Name]
			if !ok {
				return nil, fmt.Errorf("missing pre-rendered footer part %s", file.Name)
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
		} else if file.Name == "word/settings.xml" {
			// Special handling for settings.xml - remove attachedTemplate reference
			// This reference points to an external template file that may not exist
			fr, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %w", file.Name, err)
			}
			settingsContent, err := io.ReadAll(fr)
			fr.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", file.Name, err)
			}

			// Remove <w:attachedTemplate r:id="..."/> tag if present
			settingsStr := string(settingsContent)
			// Match both self-closing and regular tags
			settingsStr = regexp.MustCompile(`<w:attachedTemplate[^>]*/?>`).ReplaceAllString(settingsStr, "")

			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write([]byte(settingsStr))
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if file.Name == "word/styles.xml" {
			mergedStyles := resources.stylesXMLForRender(renderCtx.numbering, tmpl.fragments, renderCtx.usedDocxFragments)

			// Write merged styles
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write(mergedStyles)
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if file.Name == "word/numbering.xml" && needsNumberingPartWrite {
			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}
			_, err = fw.Write(renderCtx.numbering.partXML())
			if err != nil {
				return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
			}
		} else if file.Name == "[Content_Types].xml" && needsContentTypesUpdate {
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
		} else if file.Name == "word/_rels/settings.xml.rels" {
			// Special handling for settings.xml.rels - filter out attachedTemplate relationships
			// These reference external template files that may not exist, causing Word to fail opening the document
			fr, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open %s: %w", file.Name, err)
			}
			relsContent, err := io.ReadAll(fr)
			fr.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", file.Name, err)
			}

			// Parse relationships
			var rels Relationships
			err = xml.Unmarshal(relsContent, &rels)
			if err != nil {
				// If parsing fails, copy the file as-is
				fw, err := w.Create(file.Name)
				if err != nil {
					return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
				}
				_, err = fw.Write(relsContent)
				if err != nil {
					return nil, fmt.Errorf("failed to write %s: %w", file.Name, err)
				}
				continue
			}

			// Filter out attachedTemplate relationships (references to .dotm template files)
			var filteredRels []Relationship
			for _, rel := range rels.Relationship {
				if !strings.Contains(rel.Type, "attachedTemplate") {
					filteredRels = append(filteredRels, rel)
				}
			}

			// Always write the file, even if empty after filtering
			// Word expects this file to exist if settings.xml exists
			rels.Relationship = filteredRels
			output, err := xml.Marshal(&rels)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal %s: %w", file.Name, err)
			}

			fw, err := w.Create(file.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", file.Name, err)
			}

			// Write XML header
			_, err = fw.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"))
			if err != nil {
				return nil, fmt.Errorf("failed to write XML header to %s: %w", file.Name, err)
			}

			_, err = fw.Write(output)
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

	if needsNumberingPartWrite && renderCtx.numbering != nil && !renderCtx.numbering.existsInTemplate {
		fw, err := w.Create("word/numbering.xml")
		if err != nil {
			return nil, fmt.Errorf("failed to create word/numbering.xml: %w", err)
		}
		_, err = fw.Write(renderCtx.numbering.partXML())
		if err != nil {
			return nil, fmt.Errorf("failed to write word/numbering.xml: %w", err)
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

		if renderCtx.numbering != nil && renderCtx.numbering.needsContentTypeOverride() {
			alreadyRegistered := false
			for _, override := range contentTypes.Overrides {
				if override.PartName == "/word/numbering.xml" {
					alreadyRegistered = true
					break
				}
			}
			if !alreadyRegistered {
				contentTypes.Overrides = append(contentTypes.Overrides, ContentTypeOverride{
					PartName:    "/word/numbering.xml",
					ContentType: numberingContentType,
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

	return pt.state.release()
}

func (pt *PreparedTemplate) isClosed() bool {
	if pt == nil {
		return true
	}
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.closed
}

func (pt *PreparedTemplate) cloneHandle() (*PreparedTemplate, bool) {
	if pt == nil {
		return nil, false
	}

	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.closed || pt.template == nil || pt.state == nil {
		return nil, false
	}
	if !pt.state.retain() {
		return nil, false
	}

	return &PreparedTemplate{
		state:    pt.state,
		template: pt.template,
		registry: pt.registry,
	}, true
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
	if pt == nil {
		return fmt.Errorf("invalid template")
	}
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.closed || pt.template == nil {
		return fmt.Errorf("template is closed")
	}

	// Lazy initialize fragments map if needed
	if pt.template.fragments == nil {
		pt.template.fragments = make(map[string]*fragment)
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
	pt.template.invalidateRenderResources()
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
	if pt == nil {
		return fmt.Errorf("invalid template")
	}
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if pt.closed || pt.template == nil {
		return fmt.Errorf("template is closed")
	}

	// Lazy initialize fragments map if needed
	if pt.template.fragments == nil {
		pt.template.fragments = make(map[string]*fragment)
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

	// Extract relationships, styles, and numbering definitions
	var relationships []Relationship
	var stylesXML []byte
	var numberingXML []byte
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
		} else if file.Name == "word/numbering.xml" {
			rc, err := file.Open()
			if err != nil {
				continue // numbering.xml is optional
			}

			numberingXML, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				numberingXML = nil
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
		numberingXML:  numberingXML,
		stylesXML:     stylesXML,
		namespaces:    namespaces,
	}

	pt.template.fragments[name] = frag
	pt.template.invalidateRenderResources()
	return nil
}

func (t *template) invalidateRenderResources() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.renderResources = nil
}

func (t *template) ensureRenderResources() (*templateRenderResources, error) {
	t.mu.RLock()
	resources := t.renderResources
	t.mu.RUnlock()
	if resources != nil {
		return resources, nil
	}

	return t.refreshRenderResources()
}

func (t *template) refreshRenderResources() (*templateRenderResources, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	mainNamespaces := make(map[string]string)
	if t.document != nil {
		for prefix, uri := range t.document.ExtractNamespaces() {
			mainNamespaces[prefix] = uri
		}
	}

	baseNumbering, err := newNumberingContext(t.docxReader)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize numbering context: %w", err)
	}

	var mainStylesXML []byte
	if t.docxReader != nil {
		if stylesXML, err := t.docxReader.GetPart("word/styles.xml"); err == nil {
			mainStylesXML = append([]byte(nil), stylesXML...)
		}
	}

	fragmentFontOverrides := buildFragmentFontOverrideMap(mainStylesXML, t.fragments)

	resources := &templateRenderResources{
		mainNamespaces:        mainNamespaces,
		mainStylesXML:         mainStylesXML,
		baseNumbering:         baseNumbering,
		fragmentFontOverrides: fragmentFontOverrides,
		mergedStylesCache:     make(map[string][]byte),
		bodyPlans:             buildTemplateBodyPlans(t.document, t.fragments),
	}
	t.renderResources = resources
	return resources, nil
}

func (r *templateRenderResources) stylesXMLForRender(numbering *numberingContext, fragments map[string]*fragment, usedDocxFragments map[string]bool) []byte {
	if r == nil {
		return nil
	}
	if len(r.mainStylesXML) == 0 {
		return nil
	}
	if len(usedDocxFragments) == 0 {
		return r.mainStylesXML
	}

	names := make([]string, 0, len(usedDocxFragments))
	for name := range usedDocxFragments {
		frag := fragments[name]
		if frag == nil || !frag.isDocx || len(frag.stylesXML) == 0 {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return r.mainStylesXML
	}

	sort.Strings(names)

	cacheKey := buildMergedStylesCacheKey(names, numbering)
	r.cacheMu.RLock()
	if cached, ok := r.mergedStylesCache[cacheKey]; ok {
		r.cacheMu.RUnlock()
		return cached
	}
	r.cacheMu.RUnlock()

	fragmentStyles := make([][]byte, 0, len(names))
	for _, name := range names {
		stylesXML := fragments[name].stylesXML
		if numbering != nil {
			if remappedStyles, ok := numbering.fragmentStylesXML[name]; ok {
				stylesXML = remappedStyles
			}
		}
		if len(stylesXML) == 0 {
			continue
		}
		fragmentStyles = append(fragmentStyles, stylesXML)
	}
	if len(fragmentStyles) == 0 {
		return r.mainStylesXML
	}

	mergedStyles, err := mergeStyles(r.mainStylesXML, fragmentStyles...)
	if err != nil {
		return r.mainStylesXML
	}

	r.cacheMu.Lock()
	if cached, ok := r.mergedStylesCache[cacheKey]; ok {
		r.cacheMu.Unlock()
		return cached
	}
	r.mergedStylesCache[cacheKey] = mergedStyles
	r.cacheMu.Unlock()

	return mergedStyles
}

func buildMergedStylesCacheKey(names []string, numbering *numberingContext) string {
	var key strings.Builder
	for _, name := range names {
		key.WriteString(name)
		if numbering != nil {
			if numMap, ok := numbering.fragmentNumMaps[name]; ok && len(numMap) > 0 {
				keys := make([]string, 0, len(numMap))
				for oldID := range numMap {
					keys = append(keys, oldID)
				}
				sort.Strings(keys)
				for _, oldID := range keys {
					key.WriteByte('|')
					key.WriteString(oldID)
					key.WriteByte('=')
					key.WriteString(numMap[oldID])
				}
			}
		}
		key.WriteByte(';')
	}
	return key.String()
}

func isHeaderPartName(name string) bool {
	return isNumberedWordPart(name, "header")
}

func isFooterPartName(name string) bool {
	return isNumberedWordPart(name, "footer")
}

func isNumberedWordPart(name, prefix string) bool {
	if !strings.HasPrefix(name, "word/"+prefix) || !strings.HasSuffix(name, ".xml") {
		return false
	}

	digits := name[len("word/"+prefix) : len(name)-len(".xml")]
	if digits == "" {
		return false
	}

	for _, ch := range digits {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return true
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

	// Clear references to allow garbage collection.
	t.fragments = nil
	t.renderResources = nil
	t.document = nil
	t.docxReader = nil
	t.source = nil

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
