package stencil

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

// BodyElement represents any element that can appear in a document body
type BodyElement interface {
	isBodyElement()
}

// Body represents the document body
type Body struct {
	// Elements maintains the order of all body elements
	Elements []BodyElement `xml:"-"`
	// SectionProperties at the end of the body (critical for Word compatibility)
	SectionProperties *RawXMLElement `xml:"-"`
}



// ParagraphContent represents any content that can appear in a paragraph
type ParagraphContent interface {
	isParagraphContent()
}

// Paragraph represents a paragraph in the document
type Paragraph struct {
	Properties *ParagraphProperties `xml:"pPr"`
	// Content maintains the order of runs and hyperlinks
	Content    []ParagraphContent   `xml:"-"`
	// Legacy fields for backward compatibility during transition
	Runs       []Run                `xml:"-"`
	Hyperlinks []Hyperlink          `xml:"-"`
}

// isBodyElement implements the BodyElement interface
func (p Paragraph) isBodyElement() {}

// UnmarshalXML implements custom XML unmarshaling to preserve element order
func (p *Paragraph) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Temporary storage to check if we have hyperlinks
	var tempContent []ParagraphContent
	var tempRuns []Run
	var tempHyperlinks []Hyperlink
	hasHyperlinks := false
	
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
			case "pPr":
				var props ParagraphProperties
				if err := d.DecodeElement(&props, &t); err != nil {
					return err
				}
				p.Properties = &props
			case "r":
				var run Run
				if err := d.DecodeElement(&run, &t); err != nil {
					return err
				}
				tempContent = append(tempContent, &run)
				tempRuns = append(tempRuns, run)
			case "hyperlink":
				var hyperlink Hyperlink
				if err := d.DecodeElement(&hyperlink, &t); err != nil {
					return err
				}
				tempContent = append(tempContent, &hyperlink)
				tempHyperlinks = append(tempHyperlinks, hyperlink)
				hasHyperlinks = true
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				// Only populate Content if we have hyperlinks
				// This allows mergeConsecutiveRuns to work on paragraphs without hyperlinks
				if hasHyperlinks {
					p.Content = tempContent
				}
				p.Runs = tempRuns
				p.Hyperlinks = tempHyperlinks
				return nil
			}
		}
	}
	
	return nil
}

// MarshalXML implements custom XML marshaling for Paragraph to ensure proper namespacing
func (p Paragraph) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the paragraph element
	start.Name = xml.Name{Local: "w:p"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode paragraph properties
	if p.Properties != nil {
		if err := e.EncodeElement(p.Properties, xml.StartElement{Name: xml.Name{Local: "w:pPr"}}); err != nil {
			return err
		}
	}

	// If we have Content, use that to preserve order
	if len(p.Content) > 0 {
		for _, content := range p.Content {
			switch c := content.(type) {
			case *Run:
				if err := e.EncodeElement(c, xml.StartElement{Name: xml.Name{Local: "w:r"}}); err != nil {
					return err
				}
			case *Hyperlink:
				if err := e.EncodeElement(c, xml.StartElement{Name: xml.Name{Local: "w:hyperlink"}}); err != nil {
					return err
				}
			}
		}
	} else {
		// Fall back to legacy fields
		for _, run := range p.Runs {
			if err := e.EncodeElement(&run, xml.StartElement{Name: xml.Name{Local: "w:r"}}); err != nil {
				return err
			}
		}

		for _, hyperlink := range p.Hyperlinks {
			if err := e.EncodeElement(&hyperlink, xml.StartElement{Name: xml.Name{Local: "w:hyperlink"}}); err != nil {
				return err
			}
		}
	}

	// End the paragraph element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// ParagraphProperties represents paragraph formatting properties
type ParagraphProperties struct {
	Style            *Style          `xml:"pStyle"`
	Tabs             *Tabs           `xml:"tabs"`
	OverflowPunct    bool            `xml:"-"` // Stored as flag
	AutoSpaceDE      bool            `xml:"-"` // Stored as flag
	AutoSpaceDN      bool            `xml:"-"` // Stored as flag
	AdjustRightInd   bool            `xml:"-"` // Stored as flag
	Alignment        *Alignment      `xml:"jc"`
	Indentation      *Indentation    `xml:"ind"`
	Spacing          *Spacing        `xml:"spacing"`
	TextAlignment    *TextAlignment  `xml:"-"` // Stored as string
	RunProperties    *RunProperties  `xml:"rPr"` // Default run properties for paragraph
	// RawXML stores unparsed XML elements to preserve all paragraph properties
	RawXML      []RawXMLElement `xml:"-"`
	// RawXMLMarkers stores marker strings for RawXML elements (used during marshaling)
	RawXMLMarkers []string      `xml:"-"`
}

// TextAlignment represents text alignment settings
type TextAlignment struct {
	Val string `xml:"val,attr"`
}

// MarshalXML implements custom XML marshaling for TextAlignment
func (t TextAlignment) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:textAlignment"}
	start.Attr = []xml.Attr{}

	if t.Val != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:val"}, Value: t.Val})
	}

	// Self-closing element
	return e.EncodeElement(struct{}{}, start)
}

// Tabs represents tab stops
type Tabs struct {
	XMLName xml.Name `xml:"tabs"`
	Tab     []Tab    `xml:"tab"`
}

// Tab represents a single tab stop
type Tab struct {
	Val string `xml:"val,attr"`
	Pos string `xml:"pos,attr"`
}

// MarshalXML implements custom XML marshaling for Tab
func (t Tab) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tab"}
	start.Attr = []xml.Attr{}

	if t.Val != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:val"}, Value: t.Val})
	}
	if t.Pos != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:pos"}, Value: t.Pos})
	}

	// Self-closing element
	return e.EncodeElement(struct{}{}, start)
}

// UnmarshalXML implements custom XML unmarshaling to preserve unknown elements
func (p *ParagraphProperties) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
			case "pStyle":
				var style Style
				if err := d.DecodeElement(&style, &t); err != nil {
					return err
				}
				p.Style = &style
			case "tabs":
				var tabs Tabs
				if err := d.DecodeElement(&tabs, &t); err != nil {
					return err
				}
				p.Tabs = &tabs
			case "jc":
				var alignment Alignment
				if err := d.DecodeElement(&alignment, &t); err != nil {
					return err
				}
				p.Alignment = &alignment
			case "ind":
				var indentation Indentation
				if err := d.DecodeElement(&indentation, &t); err != nil {
					return err
				}
				p.Indentation = &indentation
			case "spacing":
				var spacing Spacing
				if err := d.DecodeElement(&spacing, &t); err != nil {
					return err
				}
				p.Spacing = &spacing
			case "overflowPunct":
				p.OverflowPunct = true
				if err := d.Skip(); err != nil {
					return err
				}
			case "autoSpaceDE":
				p.AutoSpaceDE = true
				if err := d.Skip(); err != nil {
					return err
				}
			case "autoSpaceDN":
				p.AutoSpaceDN = true
				if err := d.Skip(); err != nil {
					return err
				}
			case "adjustRightInd":
				p.AdjustRightInd = true
				if err := d.Skip(); err != nil {
					return err
				}
			case "textAlignment":
				var textAlign TextAlignment
				if err := d.DecodeElement(&textAlign, &t); err != nil {
					return err
				}
				p.TextAlignment = &textAlign
			case "rPr":
				var runProps RunProperties
				if err := d.DecodeElement(&runProps, &t); err != nil {
					return err
				}
				p.RunProperties = &runProps
			default:
				// Preserve unknown elements as raw XML
				var raw RawXMLElement
				raw.XMLName = t.Name
				raw.Attrs = t.Attr

				// Read the entire element content as raw XML
				depth := 1
				var buf strings.Builder

				// Start with the opening tag
				buf.WriteString("<")
				if t.Name.Space != "" {
					buf.WriteString(t.Name.Space)
					buf.WriteString(":")
				}
				buf.WriteString(t.Name.Local)
				for _, attr := range t.Attr {
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

				buf.WriteString("</")
				if t.Name.Space != "" {
					buf.WriteString(t.Name.Space)
					buf.WriteString(":")
				}
				buf.WriteString(t.Name.Local)
				buf.WriteString(">")

				raw.Content = []byte(buf.String())
				p.RawXML = append(p.RawXML, raw)
			}
		case xml.EndElement:
			if t.Name.Local == "pPr" {
				return nil
			}
		}
	}

	return nil
}

// MarshalXML implements custom XML marshaling for ParagraphProperties
func (p ParagraphProperties) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:pPr"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	if p.Style != nil {
		if err := e.EncodeElement(p.Style, xml.StartElement{Name: xml.Name{Local: "w:pStyle"}}); err != nil {
			return err
		}
	}

	if p.Tabs != nil {
		if err := e.EncodeElement(p.Tabs, xml.StartElement{Name: xml.Name{Local: "w:tabs"}}); err != nil {
			return err
		}
	}

	// Output boolean flags as empty elements
	if p.OverflowPunct {
		if err := e.EncodeElement(struct{}{}, xml.StartElement{Name: xml.Name{Local: "w:overflowPunct"}}); err != nil {
			return err
		}
	}

	if p.AutoSpaceDE {
		if err := e.EncodeElement(struct{}{}, xml.StartElement{Name: xml.Name{Local: "w:autoSpaceDE"}}); err != nil {
			return err
		}
	}

	if p.AutoSpaceDN {
		if err := e.EncodeElement(struct{}{}, xml.StartElement{Name: xml.Name{Local: "w:autoSpaceDN"}}); err != nil {
			return err
		}
	}

	if p.AdjustRightInd {
		if err := e.EncodeElement(struct{}{}, xml.StartElement{Name: xml.Name{Local: "w:adjustRightInd"}}); err != nil {
			return err
		}
	}

	if p.Alignment != nil {
		if err := e.EncodeElement(p.Alignment, xml.StartElement{Name: xml.Name{Local: "w:jc"}}); err != nil {
			return err
		}
	}

	if p.Indentation != nil {
		if err := e.EncodeElement(p.Indentation, xml.StartElement{Name: xml.Name{Local: "w:ind"}}); err != nil {
			return err
		}
	}

	if p.Spacing != nil {
		if err := e.EncodeElement(p.Spacing, xml.StartElement{Name: xml.Name{Local: "w:spacing"}}); err != nil {
			return err
		}
	}

	if p.TextAlignment != nil {
		if err := e.EncodeElement(p.TextAlignment, xml.StartElement{Name: xml.Name{Local: "w:textAlignment"}}); err != nil {
			return err
		}
	}

	// Output run properties last (sets defaults for runs in this paragraph)
	if p.RunProperties != nil {
		if err := e.EncodeElement(p.RunProperties, xml.StartElement{Name: xml.Name{Local: "w:rPr"}}); err != nil {
			return err
		}
	}

	// Encode markers for raw XML elements
	for _, marker := range p.RawXMLMarkers {
		// Create a temporary element to hold the marker
		// This will be replaced later in xml_fix.go
		markerElem := struct {
			XMLName xml.Name
			Content string `xml:",chardata"`
		}{
			XMLName: xml.Name{Local: "rawXMLMarker"},
			Content: marker,
		}
		if err := e.EncodeElement(&markerElem, xml.StartElement{Name: xml.Name{Local: "rawXMLMarker"}}); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// RawXMLElement represents a raw XML element that we preserve but don't parse
type RawXMLElement struct {
	XMLName xml.Name
	Attrs   []xml.Attr
	Content []byte
}

// Run represents a run of text with common properties
type Run struct {
	Properties *RunProperties `xml:"rPr"`
	Text       *Text          `xml:"t"`
	Break      *Break         `xml:"br"`
	// RawXML stores unparsed XML elements (like drawings) to preserve them
	RawXML     []RawXMLElement `xml:"-"`
}

// isParagraphContent implements the ParagraphContent interface
func (r Run) isParagraphContent() {}

// UnmarshalXML implements custom XML unmarshaling to preserve unknown elements
func (r *Run) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
			case "rPr":
				var props RunProperties
				if err := d.DecodeElement(&props, &t); err != nil {
					return err
				}
				r.Properties = &props
			case "t":
				var text Text
				if err := d.DecodeElement(&text, &t); err != nil {
					return err
				}
				r.Text = &text
			case "br":
				var br Break
				if err := d.DecodeElement(&br, &t); err != nil {
					return err
				}
				r.Break = &br
			default:
				// Preserve unknown elements as raw XML
				var raw RawXMLElement
				raw.XMLName = t.Name
				raw.Attrs = t.Attr

				// Read the entire element content as raw XML
				depth := 1
				var buf strings.Builder

				// Start with the opening tag
				buf.WriteString("<")
				if t.Name.Space != "" {
					buf.WriteString(t.Name.Space)
					buf.WriteString(":")
				}
				buf.WriteString(t.Name.Local)
				for _, attr := range t.Attr {
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

				buf.WriteString("</")
				if t.Name.Space != "" {
					buf.WriteString(t.Name.Space)
					buf.WriteString(":")
				}
				buf.WriteString(t.Name.Local)
				buf.WriteString(">")

				raw.Content = []byte(buf.String())
				r.RawXML = append(r.RawXML, raw)
			}
		case xml.EndElement:
			if t.Name.Local == "r" {
				return nil
			}
		}
	}

	return nil
}

// MarshalXML implements custom XML marshaling for Run to ensure proper namespacing
func (r Run) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the run element
	start.Name = xml.Name{Local: "w:r"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode run properties
	if r.Properties != nil {
		if err := e.EncodeElement(r.Properties, xml.StartElement{Name: xml.Name{Local: "w:rPr"}}); err != nil {
			return err
		}
	}

	// Encode text
	if r.Text != nil {
		if err := e.EncodeElement(r.Text, xml.StartElement{Name: xml.Name{Local: "w:t"}}); err != nil {
			return err
		}
	}

	// Encode break
	if r.Break != nil {
		if err := e.Encode(r.Break); err != nil {
			return err
		}
	}

	// Note: RawXML is handled separately after initial marshaling

	// End the run element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// RunProperties represents run formatting properties
type RunProperties struct {
	Bold          *Empty          `xml:"b"`
	Italic        *Empty          `xml:"i"`
	Underline     *UnderlineStyle `xml:"u"`
	Strike        *Empty          `xml:"strike"`
	VerticalAlign *VerticalAlign  `xml:"vertAlign"`
	Color         *Color          `xml:"color"`
	Size          *Size           `xml:"sz"`
	SizeCs        *Size           `xml:"szCs"`  // Complex script size
	Kern          *Kern           `xml:"kern"`  // Character kerning
	Lang          *Lang           `xml:"lang"`  // Language settings
	Font          *Font           `xml:"rFonts"`
	Style         *RunStyle       `xml:"rStyle"`
}

// Text represents text content
type Text struct {
	XMLName xml.Name `xml:"t"`
	Space   string   `xml:"space,attr"`
	Content string   `xml:",chardata"`
}

// MarshalXML implements custom XML marshaling for Text to ensure proper namespacing
func (t Text) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:t"}
	if t.Space == "preserve" {
		// Use the predefined XML namespace
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "space"},
			Value: "preserve",
		})
	}
	return e.EncodeElement(t.Content, start)
}

// Table represents a table in the document
type Table struct {
	Properties *TableProperties `xml:"tblPr"`
	Grid       *TableGrid       `xml:"tblGrid"`
	Rows       []TableRow       `xml:"tr"`
}

// isBodyElement implements the BodyElement interface
func (t Table) isBodyElement() {}

// MarshalXML implements custom XML marshaling for Table to ensure proper namespacing
func (t Table) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the table element with w: namespace
	start.Name = xml.Name{Local: "w:tbl"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode table properties
	if t.Properties != nil {
		if err := e.EncodeElement(t.Properties, xml.StartElement{Name: xml.Name{Local: "w:tblPr"}}); err != nil {
			return err
		}
	}

	// Encode table grid
	if t.Grid != nil {
		if err := e.EncodeElement(t.Grid, xml.StartElement{Name: xml.Name{Local: "w:tblGrid"}}); err != nil {
			return err
		}
	}

	// Encode rows
	for _, row := range t.Rows {
		if err := e.EncodeElement(&row, xml.StartElement{Name: xml.Name{Local: "w:tr"}}); err != nil {
			return err
		}
	}

	// End the table element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
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

// TableRow represents a row in a table
type TableRow struct {
	Properties *TableRowProperties `xml:"trPr"`
	Cells      []TableCell         `xml:"tc"`
}

// MarshalXML implements custom XML marshaling for TableRow to ensure proper namespacing
func (r TableRow) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the row element
	start.Name = xml.Name{Local: "w:tr"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode row properties
	if r.Properties != nil {
		if err := e.EncodeElement(r.Properties, xml.StartElement{Name: xml.Name{Local: "w:trPr"}}); err != nil {
			return err
		}
	}

	// Encode cells
	for _, cell := range r.Cells {
		if err := e.EncodeElement(&cell, xml.StartElement{Name: xml.Name{Local: "w:tc"}}); err != nil {
			return err
		}
	}

	// End the row element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// TableCell represents a cell in a table
type TableCell struct {
	Properties *TableCellProperties `xml:"tcPr"`
	Paragraphs []Paragraph          `xml:"p"`
}

// MarshalXML implements custom XML marshaling for TableCell to ensure proper namespacing
func (c TableCell) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the cell element
	start.Name = xml.Name{Local: "w:tc"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode cell properties
	if c.Properties != nil {
		if err := e.EncodeElement(c.Properties, xml.StartElement{Name: xml.Name{Local: "w:tcPr"}}); err != nil {
			return err
		}
	}

	// Encode paragraphs
	for _, para := range c.Paragraphs {
		if err := e.EncodeElement(&para, xml.StartElement{Name: xml.Name{Local: "w:p"}}); err != nil {
			return err
		}
	}

	// End the cell element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// Empty represents an empty element (used for boolean properties)
type Empty struct{}

// Style represents a style reference
type Style struct {
	Val string `xml:"val,attr"`
}

// MarshalXML implements custom XML marshaling for Style
func (s Style) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// The element name depends on the context (pStyle, tblStyle, etc.)
	// so we keep the provided name
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:val"}, Value: s.Val},
	}
	return e.EncodeElement(struct{}{}, start)
}

// Alignment represents text alignment
type Alignment struct {
	Val string `xml:"val,attr"`
}

// MarshalXML implements custom XML marshaling for Alignment
func (a Alignment) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:jc"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:val"}, Value: a.Val},
	}
	return e.EncodeElement(struct{}{}, start)
}

// Indentation represents paragraph indentation
type Indentation struct {
	Left  int `xml:"left,attr"`
	Right int `xml:"right,attr"`
}

// MarshalXML implements custom XML marshaling for Indentation
func (i Indentation) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:ind"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:left"}, Value: fmt.Sprintf("%d", i.Left)},
		{Name: xml.Name{Local: "w:right"}, Value: fmt.Sprintf("%d", i.Right)},
	}
	return e.EncodeElement(struct{}{}, start)
}

// Spacing represents paragraph spacing
type Spacing struct {
	Before     int    `xml:"before,attr,omitempty"`
	After      int    `xml:"after,attr,omitempty"`
	Line       int    `xml:"line,attr,omitempty"`
	LineRule   string `xml:"lineRule,attr,omitempty"`
}

// MarshalXML implements custom XML marshaling for Spacing
func (s Spacing) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:spacing"}
	start.Attr = []xml.Attr{}

	if s.Before != 0 {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:before"}, Value: fmt.Sprintf("%d", s.Before)})
	}
	if s.After != 0 {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:after"}, Value: fmt.Sprintf("%d", s.After)})
	}
	if s.Line != 0 {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:line"}, Value: fmt.Sprintf("%d", s.Line)})
	}
	if s.LineRule != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:lineRule"}, Value: s.LineRule})
	}

	// Self-closing element
	return e.EncodeElement(struct{}{}, start)
}

// Color represents text color
type Color struct {
	Val string `xml:"val,attr"`
}

// Size represents font size
type Size struct {
	Val int `xml:"val,attr"`
}

// MarshalXML implements custom XML marshaling for Size
func (s Size) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Ensure the element has the w: prefix if it doesn't already
	if !strings.HasPrefix(start.Name.Local, "w:") {
		start.Name.Local = "w:" + start.Name.Local
	}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:val"}, Value: fmt.Sprintf("%d", s.Val)},
	}
	return e.EncodeElement(struct{}{}, start)
}

// Kern represents character kerning
type Kern struct {
	Val int `xml:"val,attr"`
}

// MarshalXML implements custom XML marshaling for Kern
func (k Kern) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:kern"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:val"}, Value: fmt.Sprintf("%d", k.Val)},
	}
	return e.EncodeElement(struct{}{}, start)
}

// Lang represents language settings
type Lang struct {
	Val      string `xml:"val,attr,omitempty"`
	EastAsia string `xml:"eastAsia,attr,omitempty"`
	Bidi     string `xml:"bidi,attr,omitempty"`
}

// MarshalXML implements custom XML marshaling for Lang
func (l Lang) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:lang"}
	start.Attr = []xml.Attr{}

	if l.Val != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:val"}, Value: l.Val})
	}
	if l.EastAsia != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:eastAsia"}, Value: l.EastAsia})
	}
	if l.Bidi != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:bidi"}, Value: l.Bidi})
	}

	return e.EncodeElement(struct{}{}, start)
}

// Font represents font information
type Font struct {
	ASCII string `xml:"ascii,attr"`
}

// RunStyle represents a run style reference
type RunStyle struct {
	Val string `xml:"val,attr"`
}

// MarshalXML implements custom XML marshaling for RunStyle
func (s RunStyle) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:rStyle"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:val"}, Value: s.Val},
	}
	return e.EncodeElement(struct{}{}, start)
}

// UnderlineStyle represents underline formatting
type UnderlineStyle struct {
	Val string `xml:"val,attr"`
}

// VerticalAlign represents vertical text alignment (superscript/subscript)
type VerticalAlign struct {
	Val string `xml:"val,attr"`
}

// MarshalXML implements custom XML marshaling for VerticalAlign
func (v VerticalAlign) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:val"}, Value: v.Val},
	}
	return e.EncodeElement(struct{}{}, start)
}

// Break represents a line break
type Break struct {
	Type string `xml:"type,attr,omitempty"`
}

// Hyperlink represents a hyperlink in the document
type Hyperlink struct {
	ID      string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
	History string `xml:"history,attr,omitempty"`
	Runs    []Run  `xml:"r"`
}

// isParagraphContent implements the ParagraphContent interface
func (h Hyperlink) isParagraphContent() {}

// MarshalXML implements custom XML marshaling for Hyperlink to ensure proper namespacing
func (h Hyperlink) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the hyperlink element
	start.Name = xml.Name{Local: "w:hyperlink"}
	
	// Add attributes
	start.Attr = []xml.Attr{}
	if h.ID != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Space: "http://schemas.openxmlformats.org/officeDocument/2006/relationships", Local: "id"},
			Value: h.ID,
		})
	}
	if h.History != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "w:history"},
			Value: h.History,
		})
	}

	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode runs
	for _, run := range h.Runs {
		if err := e.EncodeElement(&run, xml.StartElement{Name: xml.Name{Local: "w:r"}}); err != nil {
			return err
		}
	}

	// End the hyperlink element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// MarshalXML implements xml.Marshaler to ensure Break is self-closing
func (b *Break) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Use w:br since the w namespace is already defined in the document
	start.Name = xml.Name{
		Local: "w:br",
	}
	// Clear any attributes that might have been set
	start.Attr = nil
	if b.Type != "" {
		start.Attr = append(start.Attr, xml.Attr{
			Name:  xml.Name{Local: "w:type"},
			Value: b.Type,
		})
	}
	// Encode as an empty element (self-closing)
	return e.EncodeElement(struct{}{}, start)
}

// TableProperties represents table formatting properties
type TableProperties struct {
	Style       *Style            `xml:"tblStyle"`
	Width       *Width            `xml:"tblW"`
	Indentation *TableIndentation `xml:"tblInd"`
	Layout      *TableLayout      `xml:"tblLayout"`
	CellMargins *TableCellMargins `xml:"tblCellMar"`
	Look        *TableLook        `xml:"tblLook"`
}

// TableIndentation represents table indentation from margin
type TableIndentation struct {
	Width int    `xml:"w,attr"`
	Type  string `xml:"type,attr"`
}

// TableLayout represents table layout mode
type TableLayout struct {
	Type string `xml:"type,attr"`
}

// TableCellMargins represents default cell margins for a table
type TableCellMargins struct {
	Left  *CellMargin `xml:"left"`
	Right *CellMargin `xml:"right"`
	Top   *CellMargin `xml:"top"`
	Bottom *CellMargin `xml:"bottom"`
}

// CellMargin represents a single cell margin
type CellMargin struct {
	Width int    `xml:"w,attr"`
	Type  string `xml:"type,attr"`
}

// TableLook represents table style options
type TableLook struct {
	Val         string `xml:"val,attr,omitempty"`
	FirstRow    string `xml:"firstRow,attr,omitempty"`
	LastRow     string `xml:"lastRow,attr,omitempty"`
	FirstColumn string `xml:"firstColumn,attr,omitempty"`
	LastColumn  string `xml:"lastColumn,attr,omitempty"`
	NoHBand     string `xml:"noHBand,attr,omitempty"`
	NoVBand     string `xml:"noVBand,attr,omitempty"`
}

// MarshalXML implements custom XML marshaling for TableIndentation
func (t TableIndentation) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tblInd"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:w"}, Value: fmt.Sprintf("%d", t.Width)},
		{Name: xml.Name{Local: "w:type"}, Value: t.Type},
	}
	return e.EncodeElement(struct{}{}, start)
}

// MarshalXML implements custom XML marshaling for TableLayout
func (t TableLayout) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tblLayout"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:type"}, Value: t.Type},
	}
	return e.EncodeElement(struct{}{}, start)
}

// MarshalXML implements custom XML marshaling for TableCellMargins
func (m TableCellMargins) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tblCellMar"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	if m.Left != nil {
		if err := e.EncodeElement(m.Left, xml.StartElement{Name: xml.Name{Local: "w:left"}}); err != nil {
			return err
		}
	}
	if m.Right != nil {
		if err := e.EncodeElement(m.Right, xml.StartElement{Name: xml.Name{Local: "w:right"}}); err != nil {
			return err
		}
	}
	if m.Top != nil {
		if err := e.EncodeElement(m.Top, xml.StartElement{Name: xml.Name{Local: "w:top"}}); err != nil {
			return err
		}
	}
	if m.Bottom != nil {
		if err := e.EncodeElement(m.Bottom, xml.StartElement{Name: xml.Name{Local: "w:bottom"}}); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// MarshalXML implements custom XML marshaling for CellMargin
func (m CellMargin) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:w"}, Value: fmt.Sprintf("%d", m.Width)},
		{Name: xml.Name{Local: "w:type"}, Value: m.Type},
	}
	return e.EncodeElement(struct{}{}, start)
}

// MarshalXML implements custom XML marshaling for TableLook
func (t TableLook) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tblLook"}
	start.Attr = []xml.Attr{}

	if t.Val != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:val"}, Value: t.Val})
	}
	if t.FirstRow != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:firstRow"}, Value: t.FirstRow})
	}
	if t.LastRow != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:lastRow"}, Value: t.LastRow})
	}
	if t.FirstColumn != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:firstColumn"}, Value: t.FirstColumn})
	}
	if t.LastColumn != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:lastColumn"}, Value: t.LastColumn})
	}
	if t.NoHBand != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:noHBand"}, Value: t.NoHBand})
	}
	if t.NoVBand != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:noVBand"}, Value: t.NoVBand})
	}

	return e.EncodeElement(struct{}{}, start)
}

// MarshalXML implements custom XML marshaling for TableProperties
func (p TableProperties) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tblPr"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode style if present
	if p.Style != nil {
		if err := e.EncodeElement(p.Style, xml.StartElement{Name: xml.Name{Local: "w:tblStyle"}}); err != nil {
			return err
		}
	}

	// Encode width if present
	if p.Width != nil {
		if err := e.EncodeElement(p.Width, xml.StartElement{Name: xml.Name{Local: "w:tblW"}}); err != nil {
			return err
		}
	}

	// Encode indentation if present
	if p.Indentation != nil {
		if err := e.EncodeElement(p.Indentation, xml.StartElement{Name: xml.Name{Local: "w:tblInd"}}); err != nil {
			return err
		}
	}

	// Encode layout if present
	if p.Layout != nil {
		if err := e.EncodeElement(p.Layout, xml.StartElement{Name: xml.Name{Local: "w:tblLayout"}}); err != nil {
			return err
		}
	}

	// Encode cell margins if present
	if p.CellMargins != nil {
		if err := e.EncodeElement(p.CellMargins, xml.StartElement{Name: xml.Name{Local: "w:tblCellMar"}}); err != nil {
			return err
		}
	}

	// Encode table look if present
	if p.Look != nil {
		if err := e.EncodeElement(p.Look, xml.StartElement{Name: xml.Name{Local: "w:tblLook"}}); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// TableGrid represents table column definitions
type TableGrid struct {
	Columns []GridColumn `xml:"gridCol"`
}

// MarshalXML implements custom XML marshaling for TableGrid
func (g TableGrid) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tblGrid"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode columns
	for _, col := range g.Columns {
		if err := e.EncodeElement(&col, xml.StartElement{Name: xml.Name{Local: "w:gridCol"}}); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// GridColumn represents a table column
type GridColumn struct {
	Width int `xml:"w,attr"`
}

// MarshalXML implements custom XML marshaling for GridColumn
func (g GridColumn) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:gridCol"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:w"}, Value: fmt.Sprintf("%d", g.Width)},
	}
	// Self-closing element
	return e.EncodeElement(struct{}{}, start)
}

// TableRowProperties represents row properties
type TableRowProperties struct {
	CantSplit bool    `xml:"-"` // Prevent row from splitting across pages
	Height    *Height `xml:"trHeight"`
}

// UnmarshalXML implements custom XML unmarshaling for TableRowProperties
func (p *TableRowProperties) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
			case "cantSplit":
				p.CantSplit = true
				if err := d.Skip(); err != nil {
					return err
				}
			case "trHeight":
				var height Height
				if err := d.DecodeElement(&height, &t); err != nil {
					return err
				}
				p.Height = &height
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "trPr" {
				return nil
			}
		}
	}
	return nil
}

// MarshalXML implements custom XML marshaling for TableRowProperties
func (p TableRowProperties) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:trPr"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode cantSplit if true
	if p.CantSplit {
		if err := e.EncodeElement(struct{}{}, xml.StartElement{Name: xml.Name{Local: "w:cantSplit"}}); err != nil {
			return err
		}
	}

	// Encode height if present
	if p.Height != nil {
		if err := e.EncodeElement(p.Height, xml.StartElement{Name: xml.Name{Local: "w:trHeight"}}); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// Height represents row height
type Height struct {
	Val int `xml:"val,attr"`
}

// MarshalXML implements custom XML marshaling for Height
func (h Height) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:val"}, Value: fmt.Sprintf("%d", h.Val)},
	}
	// Self-closing element
	return e.EncodeElement(struct{}{}, start)
}

// TableCellProperties represents cell properties
type TableCellProperties struct {
	Width    *Width         `xml:"tcW"`
	VAlign   *VerticalAlign `xml:"vAlign"`
	GridSpan *GridSpan      `xml:"gridSpan"`
	Shading  *Shading       `xml:"shd"`
}

// MarshalXML implements custom XML marshaling for TableCellProperties
func (p TableCellProperties) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tcPr"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode width if present
	if p.Width != nil {
		if err := e.EncodeElement(p.Width, xml.StartElement{Name: xml.Name{Local: "w:tcW"}}); err != nil {
			return err
		}
	}

	// Encode vertical alignment if present
	if p.VAlign != nil {
		if err := e.EncodeElement(p.VAlign, xml.StartElement{Name: xml.Name{Local: "w:vAlign"}}); err != nil {
			return err
		}
	}

	// Encode grid span if present
	if p.GridSpan != nil {
		if err := e.EncodeElement(p.GridSpan, xml.StartElement{Name: xml.Name{Local: "w:gridSpan"}}); err != nil {
			return err
		}
	}

	// Encode shading if present
	if p.Shading != nil {
		if err := e.EncodeElement(p.Shading, xml.StartElement{Name: xml.Name{Local: "w:shd"}}); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// Width represents width settings
type Width struct {
	Type string `xml:"type,attr"`
	Val  int    `xml:"w,attr"`
}

// GridSpan represents cell column span
type GridSpan struct {
	Val int `xml:"val,attr"`
}

// Shading represents cell or paragraph shading
type Shading struct {
	Val   string `xml:"val,attr,omitempty"`
	Color string `xml:"color,attr,omitempty"`
	Fill  string `xml:"fill,attr,omitempty"`
}

// MarshalXML implements custom XML marshaling for Shading
func (s Shading) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:shd"}
	start.Attr = []xml.Attr{}

	if s.Val != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:val"}, Value: s.Val})
	}
	if s.Color != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:color"}, Value: s.Color})
	}
	if s.Fill != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:fill"}, Value: s.Fill})
	}

	// Self-closing element
	return e.EncodeElement(struct{}{}, start)
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

// GetText returns the text content of a run
func (r *Run) GetText() string {
	if r.Text == nil {
		return ""
	}
	return r.Text.Content
}

// GetText returns the concatenated text of all runs in a paragraph
func (p *Paragraph) GetText() string {
	var texts []string
	
	// If we have Content, use that to preserve order
	if len(p.Content) > 0 {
		for _, content := range p.Content {
			switch c := content.(type) {
			case *Run:
				if text := c.GetText(); text != "" {
					texts = append(texts, text)
				}
			case *Hyperlink:
				if text := c.GetText(); text != "" {
					texts = append(texts, text)
				}
			}
		}
		return strings.Join(texts, "")
	}
	
	// Fall back to legacy fields
	for _, run := range p.Runs {
		if text := run.GetText(); text != "" {
			texts = append(texts, text)
		}
	}
	// Also include text from hyperlinks
	for _, hyperlink := range p.Hyperlinks {
		if text := hyperlink.GetText(); text != "" {
			texts = append(texts, text)
		}
	}
	
	return strings.Join(texts, "")
}

// GetText returns the concatenated text of all runs in a hyperlink
func (h *Hyperlink) GetText() string {
	var texts []string
	for _, run := range h.Runs {
		if text := run.GetText(); text != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "")
}

// GetText returns the concatenated text of all paragraphs in a cell
func (c *TableCell) GetText() string {
	var texts []string
	for _, para := range c.Paragraphs {
		if text := para.GetText(); text != "" {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "\n")
}