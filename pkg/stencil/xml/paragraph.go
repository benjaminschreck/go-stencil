package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Paragraph represents a paragraph in the document
type Paragraph struct {
	Properties *ParagraphProperties `xml:"pPr"`
	// Content maintains the order of runs and hyperlinks
	Content []ParagraphContent `xml:"-"`
	// Legacy fields for backward compatibility during transition
	Runs       []Run       `xml:"-"`
	Hyperlinks []Hyperlink `xml:"-"`
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
