package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

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

// GetText returns the text content of a run
func (r *Run) GetText() string {
	if r.Text == nil {
		return ""
	}
	return r.Text.Content
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

// Break represents a line break
type Break struct {
	Type string `xml:"type,attr,omitempty"`
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
