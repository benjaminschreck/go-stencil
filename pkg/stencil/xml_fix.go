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

// marshalDocumentWithNamespaces marshals a document with proper namespaces
func marshalDocumentWithNamespaces(doc *Document) ([]byte, error) {
	// First, collect all raw XML elements indexed by a unique marker
	rawXMLMap := make(map[string][]byte)
	markerIndex := 0

	// Walk through all elements and replace RawXML with markers
	for _, elem := range doc.Body.Elements {
		switch el := elem.(type) {
		case *Paragraph:
			// Process paragraph properties RawXML
			if el.Properties != nil && len(el.Properties.RawXML) > 0 {
				// Create markers for each raw XML element
				el.Properties.RawXMLMarkers = make([]string, len(el.Properties.RawXML))
				for i, raw := range el.Properties.RawXML {
					marker := fmt.Sprintf("__PARA_PROP_MARKER_%d__", markerIndex)
					rawXMLMap[marker] = raw.Content
					el.Properties.RawXMLMarkers[i] = marker
					markerIndex++
				}
			}

			// Check both Content and Runs for RawXML
			// Process Content first (if it exists)
			if len(el.Content) > 0 {
				for _, content := range el.Content {
					if run, ok := content.(*Run); ok {
						if len(run.RawXML) > 0 {
							// Save the raw XML and create a marker
							for _, raw := range run.RawXML {
								marker := fmt.Sprintf("__RAW_XML_MARKER_%d__", markerIndex)
								rawXMLMap[marker] = raw.Content
								markerIndex++

								// Insert marker as text in the run
								if run.Text == nil {
									run.Text = &Text{Content: marker}
								} else {
									run.Text.Content += marker
								}
							}
							// Clear RawXML to avoid issues during marshaling
							run.RawXML = nil
						}
					}
				}
			}

			// Also process the Runs slice (legacy field)
			for i := range el.Runs {
				run := &el.Runs[i]
				if len(run.RawXML) > 0 {
					// Save the raw XML and create a marker
					for _, raw := range run.RawXML {
						marker := fmt.Sprintf("__RAW_XML_MARKER_%d__", markerIndex)
						rawXMLMap[marker] = raw.Content
						markerIndex++

						// Insert marker as text in the run
						if run.Text == nil {
							run.Text = &Text{Content: marker}
						} else {
							run.Text.Content += marker
						}
					}
					// Clear RawXML to avoid issues during marshaling
					run.RawXML = nil
				}
			}
		}
	}

	// First, marshal to get the basic structure
	data, err := xml.Marshal(doc)
	if err != nil {
		return nil, err
	}

	// Convert to string for processing
	xmlStr := string(data)

	// Convert marshaled XML tags and attributes in a couple of single-pass rewrites
	xmlStr = marshalDocumentTagReplacer.Replace(xmlStr)
	xmlStr = marshalDocumentCleanupReplacer.Replace(xmlStr)

	// Now replace markers with actual raw XML (AFTER attribute fixing)
	// IMPORTANT: Raw XML elements (like drawings) must be siblings of <w:t>, not children
	// So we need to replace <w:t>marker</w:t> with just the raw XML (no text wrapper)
	for marker, rawXML := range rawXMLMap {
		if strings.Contains(xmlStr, marker) {
			// Convert full namespace URIs to namespace prefixes in the raw XML
			cleanedXML := convertNamespaceURIsToPrefix(string(rawXML))

			// Check if this is a paragraph property marker
			if strings.HasPrefix(marker, "__PARA_PROP_MARKER_") {
				// Replace the entire <rawXMLMarker>marker</rawXMLMarker> pattern with the cleaned XML
				markerElement := fmt.Sprintf("<rawXMLMarker>%s</rawXMLMarker>", marker)
				xmlStr = strings.ReplaceAll(xmlStr, markerElement, cleanedXML)
			} else {
				// Replace the entire <w:t>marker</w:t> pattern with just the cleaned XML
				// This ensures drawings are siblings of text elements, not children
				textWithMarker := fmt.Sprintf("<w:t>%s</w:t>", marker)
				textWithMarkerPreserve := fmt.Sprintf(`<w:t xml:space="preserve">%s</w:t>`, marker)

				if strings.Contains(xmlStr, textWithMarker) {
					xmlStr = strings.ReplaceAll(xmlStr, textWithMarker, cleanedXML)
				} else if strings.Contains(xmlStr, textWithMarkerPreserve) {
					xmlStr = strings.ReplaceAll(xmlStr, textWithMarkerPreserve, cleanedXML)
				} else {
					// Fallback: marker might be part of text with other content
					// In this case, just replace the marker (but this may still cause issues)
					xmlStr = strings.ReplaceAll(xmlStr, marker, cleanedXML)
				}
			}
		}
	}

	// Normalize ad-hoc namespace prefixes produced by encoding/xml for preserved attributes.
	// We keep the original document-level declarations and rewrite element-local prefixes.
	xmlStr = marshalDocumentCleanupReplacer.Replace(xmlStr)
	xmlStr = rewritePrefixInsideTags(xmlStr, "main:", "w:")
	xmlStr = rewritePrefixInsideTags(xmlStr, "wordml:", "w14:")

	// Add proper document declaration and root element with namespaces
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buf.WriteString("\n")

	// Build root element with preserved namespace attributes
	buf.WriteString("<w:document")

	// Add all preserved namespace attributes from the original document
	if len(doc.Attrs) > 0 {
		for _, attr := range doc.Attrs {
			// Skip the default xmlns declaration since we're using w:document
			if attr.Name.Local == "xmlns" && attr.Name.Space == "" {
				continue
			}
			buf.WriteString(" ")
			if attr.Name.Space != "" {
				// Convert namespace URI to prefix
				prefix := namespaceURIToPrefix(attr.Name.Space)
				buf.WriteString(prefix)
				buf.WriteString(":")
			}
			buf.WriteString(attr.Name.Local)
			buf.WriteString(`="`)
			buf.WriteString(attr.Value)
			buf.WriteString(`"`)
		}
	} else {
		// Fallback to minimal namespaces if no attributes preserved
		buf.WriteString(` xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`)
		buf.WriteString(` xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"`)
		buf.WriteString(` xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"`)
		buf.WriteString(` xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`)
		buf.WriteString(` xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"`)
		buf.WriteString(` xmlns:wp14="http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing"`)
		buf.WriteString(` xmlns:a14="http://schemas.microsoft.com/office/drawing/2010/main"`)
	}

	buf.WriteString(">")

	// Extract the body content (remove the outer document tags we added)
	start := strings.Index(xmlStr, "<w:document>")
	end := strings.LastIndex(xmlStr, "</w:document>")
	if start >= 0 && end > start {
		bodyContent := xmlStr[start+len("<w:document>") : end]
		// Trim any whitespace/newlines that might cause issues
		bodyContent = strings.TrimSpace(bodyContent)
		// Ensure body content starts with an element tag
		if !strings.HasPrefix(bodyContent, "<") {
			return nil, fmt.Errorf("invalid body content: doesn't start with '<': %s", bodyContent[:min(50, len(bodyContent))])
		}

		// Insert section properties before </w:body> if present
		if doc.Body != nil && doc.Body.SectionProperties != nil {
			// Find the closing </w:body> tag
			bodyEndTag := "</w:body>"
			bodyEndIdx := strings.LastIndex(bodyContent, bodyEndTag)
			if bodyEndIdx >= 0 {
				// Build the section properties XML with namespace conversion
				var sectBuf bytes.Buffer
				sectBuf.WriteString("<w:sectPr")

				// Add attributes
				for _, attr := range doc.Body.SectionProperties.Attrs {
					sectBuf.WriteString(" w:")
					sectBuf.WriteString(attr.Name.Local)
					sectBuf.WriteString(`="`)
					sectBuf.WriteString(attr.Value)
					sectBuf.WriteString(`"`)
				}
				sectBuf.WriteString(">")

				// Add content with namespace conversion
				sectContent := convertNamespaceURIsToPrefix(string(doc.Body.SectionProperties.Content))

				sectBuf.WriteString(sectContent)
				sectBuf.WriteString("</w:sectPr>")

				// Insert section properties before </w:body>
				bodyContent = bodyContent[:bodyEndIdx] + sectBuf.String() + bodyContent[bodyEndIdx:]
			}
		}

		buf.WriteString(bodyContent)
	}

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
