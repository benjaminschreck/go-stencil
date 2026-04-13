package stencil

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

var (
	namespaceURIToPrefixReplacer = strings.NewReplacer(
		// Core Word namespaces
		"http://schemas.openxmlformats.org/wordprocessingml/2006/main:", "w:",
		"http://schemas.openxmlformats.org/officeDocument/2006/relationships:", "r:",
		"http://schemas.openxmlformats.org/officeDocument/2006/math:", "m:",
		"http://www.w3.org/XML/1998/namespace:", "xml:",
		// Drawing namespaces
		"http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing:", "wp:",
		"http://schemas.openxmlformats.org/drawingml/2006/main:", "a:",
		"http://schemas.openxmlformats.org/drawingml/2006/picture:", "pic:",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing:", "wp14:",
		"http://schemas.microsoft.com/office/drawing/2010/main:", "a14:",
		// VML namespaces
		"urn:schemas-microsoft-com:vml:", "v:",
		"urn:schemas-microsoft-com:office:office:", "o:",
		"urn:schemas-microsoft-com:office:word:", "w10:",
		// Markup compatibility namespace
		"http://schemas.openxmlformats.org/markup-compatibility/2006:", "mc:",
		// Word processing shapes and canvas
		"http://schemas.microsoft.com/office/word/2010/wordprocessingShape:", "wps:",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas:", "wpc:",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingGroup:", "wpg:",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingInk:", "wpi:",
		// Extended Word namespaces
		"http://schemas.microsoft.com/office/word/2010/wordml:", "w14:",
		"http://schemas.microsoft.com/office/word/2012/wordml:", "w15:",
		"http://schemas.microsoft.com/office/word/2015/wordml/symex:", "w16se:",
		"http://schemas.microsoft.com/office/word/2016/wordml/cid:", "w16cid:",
		"http://schemas.microsoft.com/office/word/2018/wordml:", "w16:",
		"http://schemas.microsoft.com/office/word/2018/wordml/cex:", "w16cex:",
		"http://schemas.microsoft.com/office/word/2020/wordml/sdtdatahash:", "w16sdtdh:",
		"http://schemas.microsoft.com/office/word/2024/wordml/sdtformatlock:", "w16sdtfl:",
		"http://schemas.microsoft.com/office/word/2023/wordml/word16du:", "w16du:",
		"http://schemas.microsoft.com/office/word/2006/wordml:", "wne:",
		// Chart namespaces
		"http://schemas.microsoft.com/office/drawing/2014/chartex:", "cx:",
		"http://schemas.microsoft.com/office/drawing/2015/9/8/chartex:", "cx1:",
		"http://schemas.microsoft.com/office/drawing/2015/10/21/chartex:", "cx2:",
		"http://schemas.microsoft.com/office/drawing/2016/5/9/chartex:", "cx3:",
		"http://schemas.microsoft.com/office/drawing/2016/5/10/chartex:", "cx4:",
		"http://schemas.microsoft.com/office/drawing/2016/5/11/chartex:", "cx5:",
		"http://schemas.microsoft.com/office/drawing/2016/5/12/chartex:", "cx6:",
		"http://schemas.microsoft.com/office/drawing/2016/5/13/chartex:", "cx7:",
		"http://schemas.microsoft.com/office/drawing/2016/5/14/chartex:", "cx8:",
		// Other drawing namespaces
		"http://schemas.microsoft.com/office/drawing/2016/ink:", "aink:",
		"http://schemas.microsoft.com/office/drawing/2017/model3d:", "am3d:",
		// Office extension namespaces
		"http://schemas.microsoft.com/office/2019/extlst:", "oel:",
	)
	marshalDocumentTagReplacer = strings.NewReplacer(
		"<document>", `<w:document>`,
		"</document>", `</w:document>`,
		"<body>", `<w:body>`,
		"</body>", `</w:body>`,
		"<p>", `<w:p>`,
		"<p ", `<w:p `,
		"</p>", `</w:p>`,
		"<r>", `<w:r>`,
		"<r ", `<w:r `,
		"</r>", `</w:r>`,
		"<t ", `<w:t `,
		"<t>", `<w:t>`,
		"</t>", `</w:t>`,
		"<br/>", `<w:br/>`,
		"<br>", `<w:br/>`,
		"</br>", ``,
		"<tbl>", `<w:tbl>`,
		"</tbl>", `</w:tbl>`,
		"<tr>", `<w:tr>`,
		"</tr>", `</w:tr>`,
		"<tc>", `<w:tc>`,
		"</tc>", `</w:tc>`,
		"<tblPr>", `<w:tblPr>`,
		"</tblPr>", `</w:tblPr>`,
		"<tblGrid>", `<w:tblGrid>`,
		"</tblGrid>", `</w:tblGrid>`,
		"<gridCol/>", `<w:gridCol/>`,
		"<gridCol ", `<w:gridCol `,
		"</gridCol>", `</w:gridCol>`,
		"<tcPr>", `<w:tcPr>`,
		"</tcPr>", `</w:tcPr>`,
		"<trPr>", `<w:trPr>`,
		"</trPr>", `</w:trPr>`,
		"<pPr>", `<w:pPr>`,
		"</pPr>", `</w:pPr>`,
		"<rPr>", `<w:rPr>`,
		"</rPr>", `</w:rPr>`,
		"<b></b>", `<w:b/>`,
		"<b/>", `<w:b/>`,
		"<bCs></bCs>", `<w:bCs/>`,
		"<bCs/>", `<w:bCs/>`,
		"<i></i>", `<w:i/>`,
		"<i/>", `<w:i/>`,
		"<iCs></iCs>", `<w:iCs/>`,
		"<iCs/>", `<w:iCs/>`,
		"<u ", `<w:u `,
		"</u>", `</w:u>`,
		"<strike></strike>", `<w:strike/>`,
		"<strike/>", `<w:strike/>`,
		"<vertAlign ", `<w:vertAlign `,
		"</vertAlign>", `</w:vertAlign>`,
		"<color ", `<w:color `,
		"</color>", `</w:color>`,
		"<sz ", `<w:sz `,
		"</sz>", `</w:sz>`,
		"<lang ", `<w:lang `,
		"</lang>", `</w:lang>`,
		"<pStyle ", `<w:pStyle `,
		"</pStyle>", `</w:pStyle>`,
		"<rFonts ", `<w:rFonts `,
		"</rFonts>", `</w:rFonts>`,
		"<tblStyle ", `<w:tblStyle `,
		"</tblStyle>", `</w:tblStyle>`,
		"<tcW ", `<w:tcW `,
		"</tcW>", `</w:tcW>`,
		"<shd ", `<w:shd `,
		"</shd>", `</w:shd>`,
		`space=""`, ``,
		` val="`, ` w:val="`,
		` type="`, ` w:type="`,
		` w="`, ` w:w="`,
		` ascii="`, ` w:ascii="`,
		` before="`, ` w:before="`,
		` after="`, ` w:after="`,
		` color="`, ` w:color="`,
		` fill="`, ` w:fill="`,
		` themeFill="`, ` w:themeFill="`,
	)
	marshalDocumentCleanupReplacer = strings.NewReplacer(
		`<w:pStyle w:val=""></w:pStyle>`, ``,
		`<w:pPr></w:pPr>`, ``,
		`<w:rPr></w:rPr>`, ``,
		` xmlns:main="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`, ``,
		` xmlns:wordml="http://schemas.microsoft.com/office/word/2010/wordml"`, ``,
	)
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// namespaceURIToPrefix converts a full namespace URI to its prefix
func namespaceURIToPrefix(uri string) string {
	prefixMap := map[string]string{
		"http://schemas.openxmlformats.org/wordprocessingml/2006/main":           "w",
		"http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing": "wp",
		"http://schemas.openxmlformats.org/drawingml/2006/main":                  "a",
		"http://schemas.openxmlformats.org/drawingml/2006/picture":               "pic",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing":    "wp14",
		"http://schemas.microsoft.com/office/drawing/2010/main":                  "a14",
		"http://schemas.openxmlformats.org/officeDocument/2006/relationships":    "r",
		"http://www.w3.org/XML/1998/namespace":                                   "xml",
		"http://schemas.openxmlformats.org/markup-compatibility/2006":            "mc",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas":     "wpc",
		"http://schemas.microsoft.com/office/drawing/2014/chartex":               "cx",
		"http://schemas.microsoft.com/office/drawing/2015/9/8/chartex":           "cx1",
		"http://schemas.microsoft.com/office/drawing/2015/10/21/chartex":         "cx2",
		"http://schemas.microsoft.com/office/drawing/2016/5/9/chartex":           "cx3",
		"http://schemas.microsoft.com/office/drawing/2016/5/10/chartex":          "cx4",
		"http://schemas.microsoft.com/office/drawing/2016/5/11/chartex":          "cx5",
		"http://schemas.microsoft.com/office/drawing/2016/5/12/chartex":          "cx6",
		"http://schemas.microsoft.com/office/drawing/2016/5/13/chartex":          "cx7",
		"http://schemas.microsoft.com/office/drawing/2016/5/14/chartex":          "cx8",
		"http://schemas.microsoft.com/office/drawing/2016/ink":                   "aink",
		"http://schemas.microsoft.com/office/drawing/2017/model3d":               "am3d",
		"urn:schemas-microsoft-com:office:office":                                "o",
		"http://schemas.microsoft.com/office/2019/extlst":                        "oel",
		"http://schemas.openxmlformats.org/officeDocument/2006/math":             "m",
		"urn:schemas-microsoft-com:vml":                                          "v",
		"urn:schemas-microsoft-com:office:word":                                  "w10",
		"http://schemas.microsoft.com/office/word/2010/wordml":                   "w14",
		"http://schemas.microsoft.com/office/word/2012/wordml":                   "w15",
		"http://schemas.microsoft.com/office/word/2018/wordml/cex":               "w16cex",
		"http://schemas.microsoft.com/office/word/2016/wordml/cid":               "w16cid",
		"http://schemas.microsoft.com/office/word/2018/wordml":                   "w16",
		"http://schemas.microsoft.com/office/word/2023/wordml/word16du":          "w16du",
		"http://schemas.microsoft.com/office/word/2020/wordml/sdtdatahash":       "w16sdtdh",
		"http://schemas.microsoft.com/office/word/2024/wordml/sdtformatlock":     "w16sdtfl",
		"http://schemas.microsoft.com/office/word/2015/wordml/symex":             "w16se",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingGroup":      "wpg",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingInk":        "wpi",
		"http://schemas.microsoft.com/office/word/2006/wordml":                   "wne",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingShape":      "wps",
	}

	if prefix, ok := prefixMap[uri]; ok {
		return prefix
	}
	// Return the URI as-is if no mapping found (shouldn't happen but safe fallback)
	return uri
}

// convertNamespaceURIsToPrefix converts all namespace URIs in an XML string to their prefixes
func convertNamespaceURIsToPrefix(xml string) string {
	return namespaceURIToPrefixReplacer.Replace(xml)
}

func prepareDocumentRawXML(doc *Document) map[string][]byte {
	rawXMLMap := make(map[string][]byte)
	markerIndex := 0

	if doc == nil || doc.Body == nil {
		return rawXMLMap
	}

	for _, elem := range doc.Body.Elements {
		prepareBodyElementRawXML(elem, rawXMLMap, &markerIndex)
	}

	return rawXMLMap
}

func prepareBodyElementRawXML(elem BodyElement, rawXMLMap map[string][]byte, markerIndex *int) {
	switch e := elem.(type) {
	case *Paragraph:
		prepareParagraphRawXML(e, rawXMLMap, markerIndex)
	case *Table:
		for rowIdx := range e.Rows {
			for cellIdx := range e.Rows[rowIdx].Cells {
				for paraIdx := range e.Rows[rowIdx].Cells[cellIdx].Paragraphs {
					prepareParagraphRawXML(&e.Rows[rowIdx].Cells[cellIdx].Paragraphs[paraIdx], rawXMLMap, markerIndex)
				}
			}
		}
	}
}

func prepareParagraphRawXML(para *Paragraph, rawXMLMap map[string][]byte, markerIndex *int) {
	if para == nil {
		return
	}

	if para.Properties != nil && len(para.Properties.RawXML) > 0 {
		para.Properties.RawXMLMarkers = make([]string, len(para.Properties.RawXML))
		for i, raw := range para.Properties.RawXML {
			marker := fmt.Sprintf("__PARA_PROP_MARKER_%d__", *markerIndex)
			rawXMLMap[marker] = raw.Content
			para.Properties.RawXMLMarkers[i] = marker
			*markerIndex = *markerIndex + 1
		}
	}

	if len(para.Content) > 0 {
		for _, content := range para.Content {
			switch c := content.(type) {
			case *Run:
				prepareRunRawXML(c, rawXMLMap, markerIndex)
			case *Hyperlink:
				for runIdx := range c.Runs {
					prepareRunRawXML(&c.Runs[runIdx], rawXMLMap, markerIndex)
				}
			}
		}
		return
	}

	for runIdx := range para.Runs {
		prepareRunRawXML(&para.Runs[runIdx], rawXMLMap, markerIndex)
	}
	for linkIdx := range para.Hyperlinks {
		for runIdx := range para.Hyperlinks[linkIdx].Runs {
			prepareRunRawXML(&para.Hyperlinks[linkIdx].Runs[runIdx], rawXMLMap, markerIndex)
		}
	}
}

func prepareRunRawXML(run *Run, rawXMLMap map[string][]byte, markerIndex *int) {
	if run == nil || len(run.RawXML) == 0 {
		return
	}

	for _, raw := range run.RawXML {
		marker := fmt.Sprintf("__RAW_XML_MARKER_%d__", *markerIndex)
		rawXMLMap[marker] = raw.Content
		*markerIndex = *markerIndex + 1
		if run.Text == nil {
			run.Text = &Text{Content: marker}
		} else {
			run.Text.Content += marker
		}
	}
	run.RawXML = nil
}

func encodeXMLChunk(value interface{}, start xml.StartElement, rawXMLMap map[string][]byte) ([]byte, error) {
	bufBytes := getBufferBytes()
	temp := bytes.NewBuffer(bufBytes[:0])
	encoder := xml.NewEncoder(temp)
	if err := encoder.EncodeElement(value, start); err != nil {
		putBufferBytes(temp.Bytes())
		return nil, err
	}
	if err := encoder.Flush(); err != nil {
		putBufferBytes(temp.Bytes())
		return nil, err
	}

	chunk := postProcessXMLChunk(temp.Bytes(), rawXMLMap)
	putBufferBytes(temp.Bytes())
	return []byte(chunk), nil
}

func postProcessXMLChunk(chunk []byte, rawXMLMap map[string][]byte) string {
	xmlChunk := string(chunk)
	xmlChunk = marshalDocumentTagReplacer.Replace(xmlChunk)
	xmlChunk = marshalDocumentCleanupReplacer.Replace(xmlChunk)
	xmlChunk = rewritePrefixInsideTags(xmlChunk, "main:", "w:")
	xmlChunk = rewritePrefixInsideTags(xmlChunk, "wordml:", "w14:")

	for marker, rawXML := range rawXMLMap {
		if !strings.Contains(xmlChunk, marker) {
			continue
		}

		cleanedXML := convertNamespaceURIsToPrefix(string(rawXML))
		if strings.HasPrefix(marker, "__PARA_PROP_MARKER_") {
			markerElement := fmt.Sprintf("<rawXMLMarker>%s</rawXMLMarker>", marker)
			xmlChunk = strings.ReplaceAll(xmlChunk, markerElement, cleanedXML)
			continue
		}

		textWithMarker := fmt.Sprintf("<w:t>%s</w:t>", marker)
		textWithMarkerPreserve := fmt.Sprintf(`<w:t xml:space="preserve">%s</w:t>`, marker)
		if strings.Contains(xmlChunk, textWithMarker) {
			xmlChunk = strings.ReplaceAll(xmlChunk, textWithMarker, cleanedXML)
		} else if strings.Contains(xmlChunk, textWithMarkerPreserve) {
			xmlChunk = strings.ReplaceAll(xmlChunk, textWithMarkerPreserve, cleanedXML)
		} else {
			xmlChunk = strings.ReplaceAll(xmlChunk, marker, cleanedXML)
		}
	}

	return xmlChunk
}

func appendRootAttributes(buf *bytes.Buffer, attrs []xml.Attr) {
	if len(attrs) == 0 {
		buf.WriteString(` xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`)
		buf.WriteString(` xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"`)
		buf.WriteString(` xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"`)
		buf.WriteString(` xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`)
		buf.WriteString(` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"`)
		buf.WriteString(` xmlns:wp14="http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing"`)
		buf.WriteString(` xmlns:a14="http://schemas.microsoft.com/office/drawing/2010/main"`)
		return
	}

	for _, attr := range attrs {
		if attr.Name.Local == "xmlns" && attr.Name.Space == "" {
			continue
		}
		buf.WriteString(" ")
		if attr.Name.Space != "" {
			buf.WriteString(namespaceURIToPrefix(attr.Name.Space))
			buf.WriteString(":")
		}
		buf.WriteString(attr.Name.Local)
		buf.WriteString(`="`)
		buf.WriteString(attr.Value)
		buf.WriteString(`"`)
	}
}

func writeSectionPropertiesXML(buf *bytes.Buffer, raw *RawXMLElement) {
	if raw == nil {
		return
	}

	buf.WriteString("<w:sectPr")
	for _, attr := range raw.Attrs {
		buf.WriteString(" ")
		if attr.Name.Space != "" {
			buf.WriteString(namespaceURIToPrefix(attr.Name.Space))
			buf.WriteString(":")
		} else {
			buf.WriteString("w:")
		}
		buf.WriteString(attr.Name.Local)
		buf.WriteString(`="`)
		buf.WriteString(attr.Value)
		buf.WriteString(`"`)
	}
	buf.WriteString(">")
	buf.WriteString(convertNamespaceURIsToPrefix(string(raw.Content)))
	buf.WriteString("</w:sectPr>")
}

// marshalDocumentWithNamespaces marshals a document with proper namespaces
func marshalDocumentWithNamespaces(doc *Document) ([]byte, error) {
	rawXMLMap := prepareDocumentRawXML(doc)

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buf.WriteString("\n")
	buf.WriteString("<w:document")
	if doc != nil {
		appendRootAttributes(&buf, doc.Attrs)
	} else {
		appendRootAttributes(&buf, nil)
	}
	buf.WriteString(">")
	buf.WriteString("<w:body>")

	if doc != nil && doc.Body != nil {
		for _, elem := range doc.Body.Elements {
			switch el := elem.(type) {
			case *Paragraph:
				chunk, err := encodeXMLChunk(el, xml.StartElement{Name: xml.Name{Local: "w:p"}}, rawXMLMap)
				if err != nil {
					return nil, err
				}
				buf.Write(chunk)
			case *Table:
				chunk, err := encodeXMLChunk(el, xml.StartElement{Name: xml.Name{Local: "w:tbl"}}, rawXMLMap)
				if err != nil {
					return nil, err
				}
				buf.Write(chunk)
			}
		}
		writeSectionPropertiesXML(&buf, doc.Body.SectionProperties)
	}

	buf.WriteString(`</w:body>`)
	buf.WriteString(`</w:document>`)
	return buf.Bytes(), nil
}

// rewritePrefixInsideTags rewrites a namespace prefix only inside XML tag
// markup (element/attribute names), never inside text nodes.
func rewritePrefixInsideTags(input, fromPrefix, toPrefix string) string {
	if fromPrefix == "" || fromPrefix == toPrefix || !strings.Contains(input, fromPrefix) {
		return input
	}

	var out strings.Builder
	out.Grow(len(input))

	inTag := false
	var quote byte
	for i := 0; i < len(input); {
		ch := input[i]
		if !inTag {
			if ch == '<' {
				// Preserve XML declarations/PI/comments/CDATA/doctype as-is.
				if strings.HasPrefix(input[i:], "<![CDATA[") {
					end := strings.Index(input[i+9:], "]]>")
					if end < 0 {
						out.WriteString(input[i:])
						return out.String()
					}
					end += i + 9 + 3
					out.WriteString(input[i:end])
					i = end
					continue
				}
				if strings.HasPrefix(input[i:], "<!--") {
					end := strings.Index(input[i+4:], "-->")
					if end < 0 {
						out.WriteString(input[i:])
						return out.String()
					}
					end += i + 4 + 3
					out.WriteString(input[i:end])
					i = end
					continue
				}
				if strings.HasPrefix(input[i:], "<?") {
					end := strings.Index(input[i+2:], "?>")
					if end < 0 {
						out.WriteString(input[i:])
						return out.String()
					}
					end += i + 2 + 2
					out.WriteString(input[i:end])
					i = end
					continue
				}
				if strings.HasPrefix(input[i:], "<!") {
					end := strings.Index(input[i+2:], ">")
					if end < 0 {
						out.WriteString(input[i:])
						return out.String()
					}
					end += i + 2 + 1
					out.WriteString(input[i:end])
					i = end
					continue
				}

				inTag = true
				out.WriteByte(ch)
				i++
				continue
			}
			out.WriteByte(ch)
			i++
			continue
		}

		if quote != 0 {
			out.WriteByte(ch)
			if ch == quote {
				quote = 0
			}
			i++
			continue
		}

		if ch == '"' || ch == '\'' {
			quote = ch
			out.WriteByte(ch)
			i++
			continue
		}

		if ch == '>' {
			inTag = false
			out.WriteByte(ch)
			i++
			continue
		}

		if strings.HasPrefix(input[i:], fromPrefix) {
			out.WriteString(toPrefix)
			i += len(fromPrefix)
			continue
		}

		out.WriteByte(ch)
		i++
	}

	return out.String()
}
