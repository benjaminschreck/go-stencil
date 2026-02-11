package xml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const (
	wordprocessingMLNamespace       = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	wordprocessingMLStrictNamespace = "http://purl.oclc.org/ooxml/wordprocessingml/main"
	markupCompatibilityNamespace    = "http://schemas.openxmlformats.org/markup-compatibility/2006"
	relationshipsStrictNamespace    = "http://purl.oclc.org/ooxml/officeDocument/relationships"
)

// Paragraph represents a paragraph in the document
type Paragraph struct {
	Properties *ParagraphProperties `xml:"pPr"`
	// Attrs preserves paragraph-level attributes (e.g. w14:paraId, w:rsidR).
	Attrs []xml.Attr `xml:"-"`
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
	// Preserve paragraph-level attributes.
	if len(start.Attr) > 0 {
		p.Attrs = append([]xml.Attr(nil), start.Attr...)
	} else {
		p.Attrs = nil
	}

	// Temporary storage to check if we have hyperlinks
	var tempContent []ParagraphContent
	var tempRuns []Run
	var tempHyperlinks []Hyperlink
	useContent := false

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
				if !isWordprocessingMLElement(t) {
					if err := collectNestedParagraphContent(d, t, &tempContent, &tempRuns, &tempHyperlinks, &useContent); err != nil {
						return err
					}
					break
				}
				var run Run
				if err := d.DecodeElement(&run, &t); err != nil {
					return err
				}
				tempContent = append(tempContent, &run)
				tempRuns = append(tempRuns, run)
			case "hyperlink":
				if !isWordprocessingMLElement(t) {
					if err := collectNestedParagraphContent(d, t, &tempContent, &tempRuns, &tempHyperlinks, &useContent); err != nil {
						return err
					}
					break
				}
				var hyperlink Hyperlink
				if err := d.DecodeElement(&hyperlink, &t); err != nil {
					return err
				}
				tempContent = append(tempContent, &hyperlink)
				tempHyperlinks = append(tempHyperlinks, hyperlink)
				useContent = true
			case "proofErr":
				if !isWordprocessingMLElement(t) {
					if err := collectNestedParagraphContent(d, t, &tempContent, &tempRuns, &tempHyperlinks, &useContent); err != nil {
						return err
					}
					break
				}
				var proofErr ProofErr
				if err := d.DecodeElement(&proofErr, &t); err != nil {
					return err
				}
				tempContent = append(tempContent, &proofErr)
				useContent = true
			default:
				// Some Word wrappers (e.g. smartTag/ins/sdt) can contain runs.
				// Traverse unknown containers and collect nested paragraph content.
				if err := collectNestedParagraphContent(d, t, &tempContent, &tempRuns, &tempHyperlinks, &useContent); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				// Populate Content only when needed (e.g. hyperlinks/proofErr),
				// allowing legacy run-only paragraphs to keep existing behavior.
				if useContent {
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

func isWordprocessingMLElement(start xml.StartElement) bool {
	return start.Name.Space == "" ||
		start.Name.Space == wordprocessingMLNamespace ||
		start.Name.Space == wordprocessingMLStrictNamespace
}

func shouldSkipNestedWrapper(start xml.StartElement) bool {
	if !isWordprocessingMLElement(start) {
		return false
	}
	switch start.Name.Local {
	case "del", "moveFrom":
		return true
	default:
		return false
	}
}

func isAlternateContentElement(start xml.StartElement) bool {
	return start.Name.Space == markupCompatibilityNamespace && start.Name.Local == "AlternateContent"
}

func choiceRequirementsSupported(alternateContent xml.StartElement, choice xml.StartElement) bool {
	var requires string
	for _, attr := range choice.Attr {
		if attr.Name.Local != "Requires" {
			continue
		}
		// OOXML mc:Choice uses unqualified Requires, but be lenient.
		if attr.Name.Space == "" || attr.Name.Space == markupCompatibilityNamespace {
			requires = attr.Value
			break
		}
	}

	// Invalid/missing Requires: treat as unsupported so Fallback can be used.
	prefixes := strings.Fields(requires)
	if len(prefixes) == 0 {
		return false
	}

	choiceNamespaces := extractNamespacesFromAttrs(choice.Attr)
	alternateContentNamespaces := extractNamespacesFromAttrs(alternateContent.Attr)
	for _, prefix := range prefixes {
		if prefix == "xml" {
			continue
		}

		uri, ok := choiceNamespaces[prefix]
		if !ok {
			uri, ok = alternateContentNamespaces[prefix]
		}
		if !ok {
			uri, ok = lookupActiveParseNamespace(prefix)
		}

		if !ok || !supportedMCNamespaceURI(uri) {
			return false
		}
	}
	return true
}

func supportedMCNamespaceURI(uri string) bool {
	switch uri {
	case wordprocessingMLStrictNamespace, relationshipsStrictNamespace:
		return true
	}
	return namespaceToPrefix(uri) != uri
}

// collectNestedParagraphContent walks unknown paragraph child elements and keeps
// supported nested content instead of dropping it with Decoder.Skip().
func collectNestedParagraphContent(
	d *xml.Decoder,
	start xml.StartElement,
	tempContent *[]ParagraphContent,
	tempRuns *[]Run,
	tempHyperlinks *[]Hyperlink,
	useContent *bool,
) error {
	if shouldSkipNestedWrapper(start) {
		return d.Skip()
	}
	if isAlternateContentElement(start) {
		return collectAlternateContentBranch(d, start, tempContent, tempRuns, tempHyperlinks, useContent)
	}

	switch start.Name.Local {
	case "r":
		if !isWordprocessingMLElement(start) {
			break
		}
		var run Run
		if err := d.DecodeElement(&run, &start); err != nil {
			return err
		}
		*tempContent = append(*tempContent, &run)
		*tempRuns = append(*tempRuns, run)
		return nil
	case "hyperlink":
		if !isWordprocessingMLElement(start) {
			break
		}
		var hyperlink Hyperlink
		if err := d.DecodeElement(&hyperlink, &start); err != nil {
			return err
		}
		*tempContent = append(*tempContent, &hyperlink)
		*tempHyperlinks = append(*tempHyperlinks, hyperlink)
		*useContent = true
		return nil
	case "proofErr":
		if !isWordprocessingMLElement(start) {
			break
		}
		var proofErr ProofErr
		if err := d.DecodeElement(&proofErr, &start); err != nil {
			return err
		}
		*tempContent = append(*tempContent, &proofErr)
		*useContent = true
		return nil
	}

	for {
		token, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if err := collectNestedParagraphContent(d, t, tempContent, tempRuns, tempHyperlinks, useContent); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local && t.Name.Space == start.Name.Space {
				return nil
			}
		}
	}
}

// collectAlternateContentBranch handles mc:AlternateContent by selecting a single branch.
// OOXML semantics prefer a Choice branch when available, otherwise Fallback.
func collectAlternateContentBranch(
	d *xml.Decoder,
	start xml.StartElement,
	tempContent *[]ParagraphContent,
	tempRuns *[]Run,
	tempHyperlinks *[]Hyperlink,
	useContent *bool,
) error {
	type branchContent struct {
		content    []ParagraphContent
		runs       []Run
		hyperlinks []Hyperlink
		useContent bool
	}

	appendBranch := func(branch *branchContent) {
		if branch == nil {
			return
		}
		*tempContent = append(*tempContent, branch.content...)
		*tempRuns = append(*tempRuns, branch.runs...)
		*tempHyperlinks = append(*tempHyperlinks, branch.hyperlinks...)
		if branch.useContent {
			*useContent = true
		}
	}

	captureBranch := func(branchStart xml.StartElement) (*branchContent, error) {
		var branch branchContent
		if err := collectNestedParagraphContent(
			d,
			branchStart,
			&branch.content,
			&branch.runs,
			&branch.hyperlinks,
			&branch.useContent,
		); err != nil {
			return nil, err
		}
		return &branch, nil
	}

	var selectedChoice *branchContent
	var firstChoice *branchContent
	var fallback *branchContent

	for {
		token, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Space == markupCompatibilityNamespace && t.Name.Local == "Choice" {
				branch, err := captureBranch(t)
				if err != nil {
					return err
				}
				if firstChoice == nil {
					firstChoice = branch
				}
				if selectedChoice == nil && choiceRequirementsSupported(start, t) {
					selectedChoice = branch
				}
				continue
			}

			if t.Name.Space == markupCompatibilityNamespace && t.Name.Local == "Fallback" {
				if fallback == nil {
					branch, err := captureBranch(t)
					if err != nil {
						return err
					}
					fallback = branch
				} else {
					if err := d.Skip(); err != nil {
						return err
					}
				}
				continue
			}

			if err := collectNestedParagraphContent(d, t, tempContent, tempRuns, tempHyperlinks, useContent); err != nil {
				return err
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local && t.Name.Space == start.Name.Space {
				if selectedChoice != nil {
					appendBranch(selectedChoice)
				} else if fallback != nil {
					appendBranch(fallback)
				} else if firstChoice != nil {
					// Prevent silent data loss when no choice requirements are recognized
					// and the AlternateContent has no fallback branch.
					appendBranch(firstChoice)
				}
				return nil
			}
		}
	}
}

// MarshalXML implements custom XML marshaling for Paragraph to ensure proper namespacing
func (p Paragraph) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the paragraph element
	start.Name = xml.Name{Local: "w:p"}
	if len(p.Attrs) > 0 {
		start.Attr = append([]xml.Attr(nil), p.Attrs...)
	} else {
		start.Attr = nil
	}
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
			case *ProofErr:
				if err := e.EncodeElement(c, xml.StartElement{Name: xml.Name{Local: "w:proofErr"}}); err != nil {
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
	Style          *Style         `xml:"pStyle"`
	Tabs           *Tabs          `xml:"tabs"`
	OverflowPunct  bool           `xml:"-"` // Stored as flag
	AutoSpaceDE    bool           `xml:"-"` // Stored as flag
	AutoSpaceDN    bool           `xml:"-"` // Stored as flag
	AdjustRightInd bool           `xml:"-"` // Stored as flag
	Alignment      *Alignment     `xml:"jc"`
	Indentation    *Indentation   `xml:"ind"`
	Spacing        *Spacing       `xml:"spacing"`
	TextAlignment  *TextAlignment `xml:"-"`   // Stored as string
	RunProperties  *RunProperties `xml:"rPr"` // Default run properties for paragraph
	// RawXML stores unparsed XML elements to preserve all paragraph properties
	RawXML []RawXMLElement `xml:"-"`
	// RawXMLMarkers stores marker strings for RawXML elements (used during marshaling)
	RawXMLMarkers []string `xml:"-"`
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
	Left      string `xml:"left,attr,omitempty"`
	Right     string `xml:"right,attr,omitempty"`
	Start     string `xml:"start,attr,omitempty"`
	End       string `xml:"end,attr,omitempty"`
	FirstLine string `xml:"firstLine,attr,omitempty"`
	Hanging   string `xml:"hanging,attr,omitempty"`
}

// MarshalXML implements custom XML marshaling for Indentation
func (i Indentation) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:ind"}
	start.Attr = []xml.Attr{}
	if i.Left != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:left"}, Value: i.Left})
	}
	if i.Right != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:right"}, Value: i.Right})
	}
	if i.Start != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:start"}, Value: i.Start})
	}
	if i.End != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:end"}, Value: i.End})
	}
	if i.FirstLine != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:firstLine"}, Value: i.FirstLine})
	}
	if i.Hanging != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "w:hanging"}, Value: i.Hanging})
	}
	return e.EncodeElement(struct{}{}, start)
}

// Spacing represents paragraph spacing
type Spacing struct {
	Before   int    `xml:"before,attr,omitempty"`
	After    int    `xml:"after,attr,omitempty"`
	Line     int    `xml:"line,attr,omitempty"`
	LineRule string `xml:"lineRule,attr,omitempty"`
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

// ProofErr represents a spell/grammar proofing marker within a paragraph.
type ProofErr struct {
	Type  string     `xml:"type,attr,omitempty"`
	Attrs []xml.Attr `xml:"-"`
}

// isParagraphContent implements the ParagraphContent interface.
func (p ProofErr) isParagraphContent() {}

// UnmarshalXML preserves proofErr attributes such as w:type.
func (p *ProofErr) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if len(start.Attr) > 0 {
		p.Attrs = append([]xml.Attr(nil), start.Attr...)
	} else {
		p.Attrs = nil
	}
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			p.Type = attr.Value
			break
		}
	}
	return d.Skip()
}

// MarshalXML writes proofErr as a self-closing WordprocessingML element.
func (p ProofErr) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "w:proofErr"}
	if len(p.Attrs) > 0 {
		start.Attr = append([]xml.Attr(nil), p.Attrs...)
	} else if p.Type != "" {
		start.Attr = []xml.Attr{{Name: xml.Name{Local: "w:type"}, Value: p.Type}}
	} else {
		start.Attr = nil
	}
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
