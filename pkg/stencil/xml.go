package stencil

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Document represents a Word document structure
type Document struct {
	XMLName xml.Name `xml:"document"`
	Body    *Body    `xml:"body"`
}

// BodyElement represents any element that can appear in a document body
type BodyElement interface {
	isBodyElement()
}

// Body represents the document body
type Body struct {
	// Legacy fields for backward compatibility (deprecated)
	Paragraphs []Paragraph `xml:"p"`
	Tables     []Table     `xml:"tbl"`
	// Alternative tags with namespace prefix
	WParagraphs []Paragraph `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main p"`
	WTables     []Table     `xml:"http://schemas.openxmlformats.org/wordprocessingml/2006/main tbl"`
	
	// New field that maintains element order
	Elements []BodyElement `xml:"-"`
}

// Paragraph represents a paragraph in the document
type Paragraph struct {
	Properties *ParagraphProperties `xml:"pPr"`
	Runs       []Run                `xml:"r"`
}

// isBodyElement implements the BodyElement interface
func (p Paragraph) isBodyElement() {}

// ParagraphProperties represents paragraph formatting properties
type ParagraphProperties struct {
	Style       *Style       `xml:"pStyle"`
	Alignment   *Alignment   `xml:"jc"`
	Indentation *Indentation `xml:"ind"`
	Spacing     *Spacing     `xml:"spacing"`
}

// Run represents a run of text with common properties
type Run struct {
	Properties *RunProperties `xml:"rPr"`
	Text       *Text          `xml:"t"`
	Break      *Break         `xml:"br"`
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
	Font          *Font           `xml:"rFonts"`
}

// Text represents text content
type Text struct {
	XMLName xml.Name `xml:"t"`
	Space   string   `xml:"space,attr"`
	Content string   `xml:",chardata"`
}

// Table represents a table in the document
type Table struct {
	Properties *TableProperties `xml:"tblPr"`
	Grid       *TableGrid       `xml:"tblGrid"`
	Rows       []TableRow       `xml:"tr"`
}

// isBodyElement implements the BodyElement interface
func (t Table) isBodyElement() {}

// UnmarshalXML implements custom XML unmarshaling to preserve element order
func (b *Body) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// First, use default unmarshaling to populate legacy fields
	type bodyAlias Body
	alias := (*bodyAlias)(b)
	
	// Collect the raw XML content
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	
	// Encode the start element
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	
	// Copy all tokens until we reach the end element
	depth := 1
	for depth > 0 {
		token, err := d.Token()
		if err != nil {
			return err
		}
		
		if err := encoder.EncodeToken(token); err != nil {
			return err
		}
		
		switch token.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		}
	}
	
	if err := encoder.Flush(); err != nil {
		return err
	}
	
	// Now unmarshal the buffered content using default unmarshaling
	if err := xml.Unmarshal(buf.Bytes(), alias); err != nil {
		return err
	}
	
	// Parse again to maintain element order
	decoder := xml.NewDecoder(bytes.NewReader(buf.Bytes()))
	
	// Skip the body start element
	if _, err := decoder.Token(); err != nil {
		return err
	}
	
	// Process elements in order
	for {
		token, err := decoder.Token()
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
				if err := decoder.DecodeElement(&para, &t); err != nil {
					return err
				}
				b.Elements = append(b.Elements, para)
			case "tbl":
				var table Table
				if err := decoder.DecodeElement(&table, &t); err != nil {
					return err
				}
				b.Elements = append(b.Elements, table)
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
	
	// If we have ordered elements, use them
	if len(b.Elements) > 0 {
		for _, elem := range b.Elements {
			switch el := elem.(type) {
			case Paragraph:
				if err := e.EncodeElement(el, xml.StartElement{Name: xml.Name{Local: "w:p"}}); err != nil {
					return err
				}
			case Table:
				if err := e.EncodeElement(el, xml.StartElement{Name: xml.Name{Local: "w:tbl"}}); err != nil {
					return err
				}
			}
		}
	} else {
		// Fall back to legacy fields for backward compatibility
		// Encode paragraphs
		for _, p := range b.Paragraphs {
			if err := e.EncodeElement(p, xml.StartElement{Name: xml.Name{Local: "w:p"}}); err != nil {
				return err
			}
		}
		for _, p := range b.WParagraphs {
			if err := e.EncodeElement(p, xml.StartElement{Name: xml.Name{Local: "w:p"}}); err != nil {
				return err
			}
		}
		
		// Encode tables
		for _, t := range b.Tables {
			if err := e.EncodeElement(t, xml.StartElement{Name: xml.Name{Local: "w:tbl"}}); err != nil {
				return err
			}
		}
		for _, t := range b.WTables {
			if err := e.EncodeElement(t, xml.StartElement{Name: xml.Name{Local: "w:tbl"}}); err != nil {
				return err
			}
		}
	}
	
	// End the body element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// TableRow represents a row in a table
type TableRow struct {
	Properties *TableRowProperties `xml:"trPr"`
	Cells      []TableCell         `xml:"tc"`
}

// TableCell represents a cell in a table
type TableCell struct {
	Properties *TableCellProperties `xml:"tcPr"`
	Paragraphs []Paragraph          `xml:"p"`
}

// Empty represents an empty element (used for boolean properties)
type Empty struct{}

// Style represents a style reference
type Style struct {
	Val string `xml:"val,attr"`
}

// Alignment represents text alignment
type Alignment struct {
	Val string `xml:"val,attr"`
}

// Indentation represents paragraph indentation
type Indentation struct {
	Left  int `xml:"left,attr"`
	Right int `xml:"right,attr"`
}

// Spacing represents paragraph spacing
type Spacing struct {
	Before int `xml:"before,attr"`
	After  int `xml:"after,attr"`
}

// Color represents text color
type Color struct {
	Val string `xml:"val,attr"`
}

// Size represents font size
type Size struct {
	Val int `xml:"val,attr"`
}

// Font represents font information
type Font struct {
	ASCII string `xml:"ascii,attr"`
}

// UnderlineStyle represents underline formatting
type UnderlineStyle struct {
	Val string `xml:"val,attr"`
}

// VerticalAlign represents vertical text alignment (superscript/subscript)
type VerticalAlign struct {
	Val string `xml:"val,attr"`
}

// Break represents a line break
type Break struct {
	Type string `xml:"type,attr"`
}

// TableProperties represents table formatting properties
type TableProperties struct {
	Style *Style `xml:"tblStyle"`
}

// TableGrid represents table column definitions
type TableGrid struct {
	Columns []GridColumn `xml:"gridCol"`
}

// GridColumn represents a table column
type GridColumn struct {
	Width int `xml:"w,attr"`
}

// TableRowProperties represents row properties
type TableRowProperties struct {
	Height *Height `xml:"trHeight"`
}

// Height represents row height
type Height struct {
	Val int `xml:"val,attr"`
}

// TableCellProperties represents cell properties
type TableCellProperties struct {
	Width    *Width    `xml:"tcW"`
	GridSpan *GridSpan `xml:"gridSpan"`
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
	for _, run := range p.Runs {
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