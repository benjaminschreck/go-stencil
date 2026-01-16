package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

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

// TableProperties represents table formatting properties
type TableProperties struct {
	Style       *Style            `xml:"tblStyle"`
	Width       *Width            `xml:"tblW"`
	Indentation *TableIndentation `xml:"tblInd"`
	Borders     *TableBorders     `xml:"tblBorders"`
	Layout      *TableLayout      `xml:"tblLayout"`
	CellMargins *TableCellMargins `xml:"tblCellMar"`
	Look        *TableLook        `xml:"tblLook"`
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

	// Encode borders if present
	if p.Borders != nil {
		if err := e.EncodeElement(p.Borders, xml.StartElement{Name: xml.Name{Local: "w:tblBorders"}}); err != nil {
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

// TableIndentation represents table indentation from margin
type TableIndentation struct {
	Width int    `xml:"w,attr"`
	Type  string `xml:"type,attr"`
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

// TableLayout represents table layout mode
type TableLayout struct {
	Type string `xml:"type,attr"`
}

// MarshalXML implements custom XML marshaling for TableLayout
func (t TableLayout) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tblLayout"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:type"}, Value: t.Type},
	}
	return e.EncodeElement(struct{}{}, start)
}

// TableCellMargins represents default cell margins for a table
type TableCellMargins struct {
	Left  *CellMargin `xml:"left"`
	Right *CellMargin `xml:"right"`
	Top   *CellMargin `xml:"top"`
	Bottom *CellMargin `xml:"bottom"`
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

// CellMargin represents a single cell margin
type CellMargin struct {
	Width int    `xml:"w,attr"`
	Type  string `xml:"type,attr"`
}

// MarshalXML implements custom XML marshaling for CellMargin
func (m CellMargin) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "w:w"}, Value: fmt.Sprintf("%d", m.Width)},
		{Name: xml.Name{Local: "w:type"}, Value: m.Type},
	}
	return e.EncodeElement(struct{}{}, start)
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

// TableCellProperties represents cell properties
type TableCellProperties struct {
	Width     *Width          `xml:"tcW"`
	VAlign    *VerticalAlign  `xml:"vAlign"`
	GridSpan  *GridSpan       `xml:"gridSpan"`
	Shading   *Shading        `xml:"shd"`
	TcBorders *TableCellBorders `xml:"tcBorders"`
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

	// Encode table cell borders if present
	if p.TcBorders != nil {
		if err := e.EncodeElement(p.TcBorders, xml.StartElement{Name: xml.Name{Local: "w:tcBorders"}}); err != nil {
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
	Val       string `xml:"val,attr,omitempty"`
	Color     string `xml:"color,attr,omitempty"`
	Fill      string `xml:"fill,attr,omitempty"`
	ThemeFill string `xml:"themeFill,attr,omitempty"`
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
	if s.ThemeFill != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:themeFill"}, Value: s.ThemeFill})
	}

	// Self-closing element
	return e.EncodeElement(struct{}{}, start)
}

// TableBorders represents borders for a table (w:tblBorders)
// This includes inner borders (insideH, insideV) in addition to outer borders
type TableBorders struct {
	Top     *BorderProperties `xml:"top"`
	Left    *BorderProperties `xml:"left"`
	Bottom  *BorderProperties `xml:"bottom"`
	Right   *BorderProperties `xml:"right"`
	InsideH *BorderProperties `xml:"insideH"`
	InsideV *BorderProperties `xml:"insideV"`
}

// MarshalXML implements custom XML marshaling for TableBorders
func (b TableBorders) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tblBorders"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode each border if present (order matters in Word XML)
	if b.Top != nil {
		if err := e.EncodeElement(b.Top, xml.StartElement{Name: xml.Name{Local: "w:top"}}); err != nil {
			return err
		}
	}
	if b.Left != nil {
		if err := e.EncodeElement(b.Left, xml.StartElement{Name: xml.Name{Local: "w:left"}}); err != nil {
			return err
		}
	}
	if b.Bottom != nil {
		if err := e.EncodeElement(b.Bottom, xml.StartElement{Name: xml.Name{Local: "w:bottom"}}); err != nil {
			return err
		}
	}
	if b.Right != nil {
		if err := e.EncodeElement(b.Right, xml.StartElement{Name: xml.Name{Local: "w:right"}}); err != nil {
			return err
		}
	}
	if b.InsideH != nil {
		if err := e.EncodeElement(b.InsideH, xml.StartElement{Name: xml.Name{Local: "w:insideH"}}); err != nil {
			return err
		}
	}
	if b.InsideV != nil {
		if err := e.EncodeElement(b.InsideV, xml.StartElement{Name: xml.Name{Local: "w:insideV"}}); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// TableCellBorders represents borders for a table cell
type TableCellBorders struct {
	Top    *BorderProperties `xml:"top"`
	Bottom *BorderProperties `xml:"bottom"`
	Left   *BorderProperties `xml:"left"`
	Right  *BorderProperties `xml:"right"`
}

// MarshalXML implements custom XML marshaling for TableCellBorders
func (b TableCellBorders) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:tcBorders"}
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode each border if present
	if b.Top != nil {
		if err := e.EncodeElement(b.Top, xml.StartElement{Name: xml.Name{Local: "w:top"}}); err != nil {
			return err
		}
	}
	if b.Bottom != nil {
		if err := e.EncodeElement(b.Bottom, xml.StartElement{Name: xml.Name{Local: "w:bottom"}}); err != nil {
			return err
		}
	}
	if b.Left != nil {
		if err := e.EncodeElement(b.Left, xml.StartElement{Name: xml.Name{Local: "w:left"}}); err != nil {
			return err
		}
	}
	if b.Right != nil {
		if err := e.EncodeElement(b.Right, xml.StartElement{Name: xml.Name{Local: "w:right"}}); err != nil {
			return err
		}
	}

	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// BorderProperties represents border styling
type BorderProperties struct {
	Val        string `xml:"val,attr,omitempty"`
	Sz         string `xml:"sz,attr,omitempty"`
	Space      string `xml:"space,attr,omitempty"`
	Color      string `xml:"color,attr,omitempty"`
	ThemeColor string `xml:"themeColor,attr,omitempty"`
	ThemeShade string `xml:"themeShade,attr,omitempty"`
}

// MarshalXML implements custom XML marshaling for BorderProperties
func (b BorderProperties) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Attr = []xml.Attr{}

	if b.Val != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:val"}, Value: b.Val})
	}
	if b.Sz != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:sz"}, Value: b.Sz})
	}
	if b.Space != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:space"}, Value: b.Space})
	}
	if b.Color != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:color"}, Value: b.Color})
	}
	if b.ThemeColor != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:themeColor"}, Value: b.ThemeColor})
	}
	if b.ThemeShade != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:themeShade"}, Value: b.ThemeShade})
	}

	// Self-closing element
	return e.EncodeElement(struct{}{}, start)
}
