package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"sync"
)

var (
	parseContexts sync.Map
)

type parseContext struct {
	namespaceStack []map[string]string
}

// Document represents a Word document structure
type Document struct {
	XMLName xml.Name   `xml:"document"`
	Body    *Body      `xml:"body"`
	Attrs   []xml.Attr `xml:"-"` // Preserve root element attributes (namespaces)
}

// UnmarshalXML implements custom XML unmarshaling to preserve root attributes
func (doc *Document) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Save the attributes from the root element
	doc.Attrs = start.Attr
	parseContexts.Store(d, &parseContext{
		namespaceStack: []map[string]string{extractNamespacesFromAttrs(start.Attr)},
	})
	defer parseContexts.Delete(d)

	// Use an anonymous struct to avoid recursive UnmarshalXML calls
	var temp struct {
		XMLName xml.Name `xml:"document"`
		Body    *Body    `xml:"body"`
	}

	if err := d.DecodeElement(&temp, &start); err != nil {
		return err
	}

	doc.XMLName = temp.XMLName
	doc.Body = temp.Body

	return nil
}

func parseContextForDecoder(d *xml.Decoder) *parseContext {
	ctx, ok := parseContexts.Load(d)
	if !ok {
		return nil
	}
	parseCtx, ok := ctx.(*parseContext)
	if !ok {
		return nil
	}
	return parseCtx
}

func pushParseNamespaceScope(d *xml.Decoder, attrs []xml.Attr) {
	parseCtx := parseContextForDecoder(d)
	if parseCtx == nil {
		return
	}

	current := map[string]string{}
	if n := len(parseCtx.namespaceStack); n > 0 {
		current = copyNamespaces(parseCtx.namespaceStack[n-1])
	}
	for prefix, uri := range extractNamespacesFromAttrs(attrs) {
		current[prefix] = uri
	}

	parseCtx.namespaceStack = append(parseCtx.namespaceStack, current)
}

func popParseNamespaceScope(d *xml.Decoder) {
	parseCtx := parseContextForDecoder(d)
	if parseCtx == nil || len(parseCtx.namespaceStack) == 0 {
		return
	}
	parseCtx.namespaceStack = parseCtx.namespaceStack[:len(parseCtx.namespaceStack)-1]
}

func currentParseNamespaces(d *xml.Decoder) map[string]string {
	parseCtx := parseContextForDecoder(d)
	if parseCtx == nil || len(parseCtx.namespaceStack) == 0 {
		return nil
	}
	return copyNamespaces(parseCtx.namespaceStack[len(parseCtx.namespaceStack)-1])
}

func copyNamespaces(namespaces map[string]string) map[string]string {
	if len(namespaces) == 0 {
		return map[string]string{}
	}
	dup := make(map[string]string, len(namespaces))
	for prefix, uri := range namespaces {
		dup[prefix] = uri
	}
	return dup
}

func mergeNamespaces(namespaces map[string]string, attrs []xml.Attr) map[string]string {
	merged := copyNamespaces(namespaces)
	for prefix, uri := range extractNamespacesFromAttrs(attrs) {
		merged[prefix] = uri
	}
	return merged
}

// Body represents the document body
type Body struct {
	// Elements maintains the order of all body elements
	Elements []BodyElement `xml:"-"`
	// SectionProperties at the end of the body (critical for Word compatibility)
	SectionProperties *RawXMLElement `xml:"-"`
}

// UnmarshalXML implements custom XML unmarshaling to preserve element order
func (b *Body) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	pushParseNamespaceScope(d, start.Attr)
	defer popParseNamespaceScope(d)

	// Process elements in order
	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				var para Paragraph
				if err := d.DecodeElement(&para, &t); err != nil {
					return err
				}
				b.Elements = append(b.Elements, &para)
			case "tbl":
				var table Table
				if err := d.DecodeElement(&table, &t); err != nil {
					return err
				}
				b.Elements = append(b.Elements, &table)
			case "sectPr":
				// Capture section properties as raw XML
				var raw RawXMLElement
				raw.XMLName = t.Name
				raw.Attrs = t.Attr

				// Read the entire element content as raw XML
				depth := 1
				var buf strings.Builder

				for depth > 0 {
					tok, err := d.Token()
					if err != nil {
						return err
					}

					switch tt := tok.(type) {
					case xml.StartElement:
						depth++
						buf.WriteString("<")
						if tt.Name.Space != "" {
							buf.WriteString(tt.Name.Space)
							buf.WriteString(":")
						}
						buf.WriteString(tt.Name.Local)
						for _, attr := range tt.Attr {
							buf.WriteString(" ")
							if attr.Name.Space != "" {
								buf.WriteString(attr.Name.Space)
								buf.WriteString(":")
							}
							buf.WriteString(attr.Name.Local)
							buf.WriteString("=\"")
							buf.WriteString(attr.Value)
							buf.WriteString("\"")
						}
						buf.WriteString(">")
					case xml.EndElement:
						depth--
						if depth > 0 {
							buf.WriteString("</")
							if tt.Name.Space != "" {
								buf.WriteString(tt.Name.Space)
								buf.WriteString(":")
							}
							buf.WriteString(tt.Name.Local)
							buf.WriteString(">")
						}
					case xml.CharData:
						buf.Write(tt)
					}
				}

				raw.Content = []byte(buf.String())
				b.SectionProperties = &raw
			}
		case xml.EndElement:
			if t.Name.Local == "body" {
				return nil
			}
		}
	}

	return nil
}

// MarshalXML implements custom XML marshaling to preserve element order
func (b Body) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the body element
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode elements in order
	for _, elem := range b.Elements {
		switch el := elem.(type) {
		case *Paragraph:
			if err := e.EncodeElement(el, xml.StartElement{Name: xml.Name{Local: "w:p"}}); err != nil {
				return err
			}
		case *Table:
			if err := e.EncodeElement(el, xml.StartElement{Name: xml.Name{Local: "w:tbl"}}); err != nil {
				return err
			}
		}
	}

	// Note: Section properties are handled in xml_fix.go to properly convert namespaces

	// End the body element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// ParseDocument parses a Word document XML
func ParseDocument(r io.Reader) (*Document, error) {
	decoder := xml.NewDecoder(r)

	var doc Document
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}

	return &doc, nil
}

// ExtractNamespaces returns all namespace declarations from document attributes
// Returns a map of prefix -> namespace URI
func (doc *Document) ExtractNamespaces() map[string]string {
	return extractNamespacesFromAttrs(doc.Attrs)
}

func extractNamespacesFromAttrs(attrs []xml.Attr) map[string]string {
	namespaces := make(map[string]string)

	for _, attr := range attrs {
		// Handle different forms that Go's XML parser produces:
		// Form 1: Name.Space = "xmlns", Name.Local = "prefix"
		// Form 2: Name.Local = "xmlns:prefix", Name.Space = ""
		// Form 3: Name.Local = "xmlns" (default namespace)

		if attr.Name.Space == "xmlns" {
			// Form 1: xmlns:prefix="uri"
			namespaces[attr.Name.Local] = attr.Value
		} else if attr.Name.Local == "xmlns" {
			// Form 3: xmlns="uri" (default namespace)
			namespaces[""] = attr.Value
		} else if strings.HasPrefix(attr.Name.Local, "xmlns:") {
			// Form 2: xmlns:prefix as single local name
			prefix := strings.TrimPrefix(attr.Name.Local, "xmlns:")
			namespaces[prefix] = attr.Value
		}
	}

	return namespaces
}

// MergeNamespaces adds namespace declarations to the document attributes
// If a prefix already exists, the existing declaration is preserved (first wins)
func (doc *Document) MergeNamespaces(additionalNamespaces map[string]string) {
	if len(additionalNamespaces) == 0 {
		return
	}

	// Extract existing namespace prefixes
	existingPrefixes := make(map[string]string) // prefix -> URI
	for _, attr := range doc.Attrs {
		if attr.Name.Space == "xmlns" {
			existingPrefixes[attr.Name.Local] = attr.Value
		} else if attr.Name.Local == "xmlns" {
			existingPrefixes[""] = attr.Value
		} else if strings.HasPrefix(attr.Name.Local, "xmlns:") {
			prefix := strings.TrimPrefix(attr.Name.Local, "xmlns:")
			existingPrefixes[prefix] = attr.Value
		}
	}

	// Add missing namespace declarations
	for prefix, uri := range additionalNamespaces {
		if existingURI, exists := existingPrefixes[prefix]; exists {
			// Log if there's a URI mismatch (shouldn't happen after collection phase)
			if existingURI != uri {
				// This indicates a bug in the collection phase
				// Log but continue (existing declaration wins)
				// Note: In production, this should use a logger
				// For now, we silently continue
			}
			continue // Already declared in main document
		}

		var attr xml.Attr
		if prefix == "" {
			// Default namespace: xmlns="uri"
			attr = xml.Attr{
				Name:  xml.Name{Local: "xmlns"},
				Value: uri,
			}
		} else {
			// Prefixed namespace: xmlns:prefix="uri"
			// Use the Local form (compatible with marshalDocumentWithNamespaces)
			attr = xml.Attr{
				Name:  xml.Name{Local: "xmlns:" + prefix},
				Value: uri,
			}
		}

		doc.Attrs = append(doc.Attrs, attr)
	}
}
