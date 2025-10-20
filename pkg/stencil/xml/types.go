package xml

import (
	"encoding/xml"
)

// BodyElement represents any element that can appear in a document body
type BodyElement interface {
	isBodyElement()
}

// ParagraphContent represents any content that can appear in a paragraph
type ParagraphContent interface {
	isParagraphContent()
}

// RawXMLElement represents a raw XML element that we preserve but don't parse
type RawXMLElement struct {
	XMLName xml.Name
	Attrs   []xml.Attr
	Content []byte
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
