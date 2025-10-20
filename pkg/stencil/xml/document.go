package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

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

// Body represents the document body
type Body struct {
	// Elements maintains the order of all body elements
	Elements []BodyElement `xml:"-"`
	// SectionProperties at the end of the body (critical for Word compatibility)
	SectionProperties *RawXMLElement `xml:"-"`
}

// UnmarshalXML implements custom XML unmarshaling to preserve element order
func (b *Body) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
