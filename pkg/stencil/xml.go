package stencil

// This file provides backward-compatible re-exports of types from the xml package.
// All XML-related structures and functions have been moved to pkg/stencil/xml/ for better organization.

import (
	"io"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/xml"
)

// Re-export core types and interfaces
type (
	BodyElement      = xml.BodyElement
	ParagraphContent = xml.ParagraphContent
	RawXMLElement    = xml.RawXMLElement
	Empty            = xml.Empty
	Style            = xml.Style
)

// Re-export document types
type (
	Document = xml.Document
	Body     = xml.Body
)

// Re-export paragraph types
type (
	Paragraph           = xml.Paragraph
	ParagraphProperties = xml.ParagraphProperties
	TextAlignment       = xml.TextAlignment
	Tabs                = xml.Tabs
	Tab                 = xml.Tab
	Alignment           = xml.Alignment
	Indentation         = xml.Indentation
	Spacing             = xml.Spacing
	Hyperlink           = xml.Hyperlink
)

// Re-export run types
type (
	Run            = xml.Run
	RunProperties  = xml.RunProperties
	Text           = xml.Text
	Break          = xml.Break
	Color          = xml.Color
	Size           = xml.Size
	Kern           = xml.Kern
	Lang           = xml.Lang
	Font           = xml.Font
	RunStyle       = xml.RunStyle
	UnderlineStyle = xml.UnderlineStyle
	VerticalAlign  = xml.VerticalAlign
)

// Re-export table types
type (
	Table                = xml.Table
	TableProperties      = xml.TableProperties
	TableIndentation     = xml.TableIndentation
	TableLayout          = xml.TableLayout
	TableCellMargins     = xml.TableCellMargins
	CellMargin           = xml.CellMargin
	TableLook            = xml.TableLook
	TableGrid            = xml.TableGrid
	GridColumn           = xml.GridColumn
	TableRow             = xml.TableRow
	TableRowProperties   = xml.TableRowProperties
	Height               = xml.Height
	TableCell            = xml.TableCell
	TableCellProperties  = xml.TableCellProperties
	Width                = xml.Width
	GridSpan             = xml.GridSpan
	Shading              = xml.Shading
	TableCellBorders     = xml.TableCellBorders
	BorderProperties     = xml.BorderProperties
)

// Re-export functions
var (
	ParseDocument = xml.ParseDocument
)

// ParseDocument parses a Word document XML (backward compatibility wrapper)
func parseDocument(r io.Reader) (*Document, error) {
	return xml.ParseDocument(r)
}
