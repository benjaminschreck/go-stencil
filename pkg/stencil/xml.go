package stencil

import (
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
	// Elements maintains the order of all body elements
	Elements []BodyElement `xml:"-"`
}



// Paragraph represents a paragraph in the document
type Paragraph struct {
	Properties *ParagraphProperties `xml:"pPr"`
	Runs       []Run                `xml:"r"`
}

// isBodyElement implements the BodyElement interface
func (p Paragraph) isBodyElement() {}

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

	// Encode runs
	for _, run := range p.Runs {
		if err := e.EncodeElement(&run, xml.StartElement{Name: xml.Name{Local: "w:r"}}); err != nil {
			return err
		}
	}

	// End the paragraph element
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

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
	Font          *Font           `xml:"rFonts"`
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

// TableProperties represents table formatting properties
type TableProperties struct {
	Style *Style `xml:"tblStyle"`
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