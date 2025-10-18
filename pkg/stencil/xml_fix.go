package stencil

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
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
		"http://schemas.openxmlformats.org/wordprocessingml/2006/main":                    "w",
		"http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing":          "wp",
		"http://schemas.openxmlformats.org/drawingml/2006/main":                           "a",
		"http://schemas.openxmlformats.org/drawingml/2006/picture":                        "pic",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing":             "wp14",
		"http://schemas.microsoft.com/office/drawing/2010/main":                           "a14",
		"http://schemas.openxmlformats.org/officeDocument/2006/relationships":             "r",
		"http://www.w3.org/XML/1998/namespace":                                            "xml",
		"http://schemas.openxmlformats.org/markup-compatibility/2006":                     "mc",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas":              "wpc",
		"http://schemas.microsoft.com/office/drawing/2014/chartex":                        "cx",
		"http://schemas.microsoft.com/office/drawing/2015/9/8/chartex":                    "cx1",
		"http://schemas.microsoft.com/office/drawing/2015/10/21/chartex":                  "cx2",
		"http://schemas.microsoft.com/office/drawing/2016/5/9/chartex":                    "cx3",
		"http://schemas.microsoft.com/office/drawing/2016/5/10/chartex":                   "cx4",
		"http://schemas.microsoft.com/office/drawing/2016/5/11/chartex":                   "cx5",
		"http://schemas.microsoft.com/office/drawing/2016/5/12/chartex":                   "cx6",
		"http://schemas.microsoft.com/office/drawing/2016/5/13/chartex":                   "cx7",
		"http://schemas.microsoft.com/office/drawing/2016/5/14/chartex":                   "cx8",
		"http://schemas.microsoft.com/office/drawing/2016/ink":                            "aink",
		"http://schemas.microsoft.com/office/drawing/2017/model3d":                        "am3d",
		"urn:schemas-microsoft-com:office:office":                                         "o",
		"http://schemas.microsoft.com/office/2019/extlst":                                 "oel",
		"http://schemas.openxmlformats.org/officeDocument/2006/math":                      "m",
		"urn:schemas-microsoft-com:vml":                                                   "v",
		"urn:schemas-microsoft-com:office:word":                                           "w10",
		"http://schemas.microsoft.com/office/word/2010/wordml":                            "w14",
		"http://schemas.microsoft.com/office/word/2012/wordml":                            "w15",
		"http://schemas.microsoft.com/office/word/2018/wordml/cex":                        "w16cex",
		"http://schemas.microsoft.com/office/word/2016/wordml/cid":                        "w16cid",
		"http://schemas.microsoft.com/office/word/2018/wordml":                            "w16",
		"http://schemas.microsoft.com/office/word/2023/wordml/word16du":                   "w16du",
		"http://schemas.microsoft.com/office/word/2020/wordml/sdtdatahash":                "w16sdtdh",
		"http://schemas.microsoft.com/office/word/2024/wordml/sdtformatlock":              "w16sdtfl",
		"http://schemas.microsoft.com/office/word/2015/wordml/symex":                      "w16se",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingGroup":               "wpg",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingInk":                 "wpi",
		"http://schemas.microsoft.com/office/word/2006/wordml":                            "wne",
		"http://schemas.microsoft.com/office/word/2010/wordprocessingShape":               "wps",
	}

	if prefix, ok := prefixMap[uri]; ok {
		return prefix
	}
	// Return the URI as-is if no mapping found (shouldn't happen but safe fallback)
	return uri
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
	
	// Add w: prefix to all elements
	xmlStr = strings.ReplaceAll(xmlStr, "<document>", `<w:document>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</document>", `</w:document>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<body>", `<w:body>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</body>", `</w:body>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<p>", `<w:p>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</p>", `</w:p>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<r>", `<w:r>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</r>", `</w:r>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<t ", `<w:t `)
	xmlStr = strings.ReplaceAll(xmlStr, "<t>", `<w:t>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</t>", `</w:t>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<br>", `<w:br/>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</br>", ``)
	xmlStr = strings.ReplaceAll(xmlStr, "<br/>", `<w:br/>`)
	
	// Handle table elements
	xmlStr = strings.ReplaceAll(xmlStr, "<tbl>", `<w:tbl>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</tbl>", `</w:tbl>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<tr>", `<w:tr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</tr>", `</w:tr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<tc>", `<w:tc>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</tc>", `</w:tc>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<tblPr>", `<w:tblPr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</tblPr>", `</w:tblPr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<tblGrid>", `<w:tblGrid>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</tblGrid>", `</w:tblGrid>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<gridCol ", `<w:gridCol `)
	xmlStr = strings.ReplaceAll(xmlStr, "<gridCol/>", `<w:gridCol/>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<tcPr>", `<w:tcPr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</tcPr>", `</w:tcPr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<trPr>", `<w:trPr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</trPr>", `</w:trPr>`)
	
	// Handle properties
	xmlStr = strings.ReplaceAll(xmlStr, "<pPr>", `<w:pPr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</pPr>", `</w:pPr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<rPr>", `<w:rPr>`)
	xmlStr = strings.ReplaceAll(xmlStr, "</rPr>", `</w:rPr>`)
	
	// Handle formatting elements in run properties
	xmlStr = strings.ReplaceAll(xmlStr, "<b></b>", `<w:b/>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<b/>", `<w:b/>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<i></i>", `<w:i/>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<i/>", `<w:i/>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<u ", `<w:u `)
	xmlStr = strings.ReplaceAll(xmlStr, "</u>", `</w:u>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<strike></strike>", `<w:strike/>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<strike/>", `<w:strike/>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<vertAlign ", `<w:vertAlign `)
	xmlStr = strings.ReplaceAll(xmlStr, "</vertAlign>", `</w:vertAlign>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<color ", `<w:color `)
	xmlStr = strings.ReplaceAll(xmlStr, "</color>", `</w:color>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<sz ", `<w:sz `)
	xmlStr = strings.ReplaceAll(xmlStr, "</sz>", `</w:sz>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<lang ", `<w:lang `)
	xmlStr = strings.ReplaceAll(xmlStr, "</lang>", `</w:lang>`)
	
	// Handle style elements
	xmlStr = strings.ReplaceAll(xmlStr, "<pStyle ", `<w:pStyle `)
	xmlStr = strings.ReplaceAll(xmlStr, "</pStyle>", `</w:pStyle>`)
	// Remove empty pStyle elements
	xmlStr = strings.ReplaceAll(xmlStr, `<w:pStyle w:val=""></w:pStyle>`, ``)
	xmlStr = strings.ReplaceAll(xmlStr, "<rFonts ", `<w:rFonts `)
	xmlStr = strings.ReplaceAll(xmlStr, "</rFonts>", `</w:rFonts>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<tblStyle ", `<w:tblStyle `)
	xmlStr = strings.ReplaceAll(xmlStr, "</tblStyle>", `</w:tblStyle>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<tcW ", `<w:tcW `)
	xmlStr = strings.ReplaceAll(xmlStr, "</tcW>", `</w:tcW>`)
	xmlStr = strings.ReplaceAll(xmlStr, "<gridCol ", `<w:gridCol `)
	xmlStr = strings.ReplaceAll(xmlStr, "</gridCol>", `</w:gridCol>`)
	
	// Handle attributes
	// Don't add xml: prefix here since MarshalXML already handles it properly
	// xmlStr = strings.ReplaceAll(xmlStr, `space="preserve"`, `xml:space="preserve"`)
	xmlStr = strings.ReplaceAll(xmlStr, `space=""`, ``)
	
	// Remove empty property elements that might cause issues
	xmlStr = strings.ReplaceAll(xmlStr, `<w:pPr></w:pPr>`, ``)
	xmlStr = strings.ReplaceAll(xmlStr, `<w:rPr></w:rPr>`, ``)

	// Fix attribute namespaces on our marshaled XML (while markers are still in place)
	xmlStr = strings.ReplaceAll(xmlStr, ` val="`, ` w:val="`)
	xmlStr = strings.ReplaceAll(xmlStr, ` type="`, ` w:type="`)
	xmlStr = strings.ReplaceAll(xmlStr, ` w="`, ` w:w="`)
	xmlStr = strings.ReplaceAll(xmlStr, ` ascii="`, ` w:ascii="`)
	xmlStr = strings.ReplaceAll(xmlStr, ` before="`, ` w:before="`)
	xmlStr = strings.ReplaceAll(xmlStr, ` after="`, ` w:after="`)

	// Now replace markers with actual raw XML (AFTER attribute fixing)
	// IMPORTANT: Raw XML elements (like drawings) must be siblings of <w:t>, not children
	// So we need to replace <w:t>marker</w:t> with just the raw XML (no text wrapper)
	for marker, rawXML := range rawXMLMap {
		if strings.Contains(xmlStr, marker) {
			// Convert full namespace URIs to namespace prefixes in the raw XML
			cleanedXML := string(rawXML)
			// For elements: <uri:name> becomes <prefix:name>
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/wordprocessingml/2006/main:", "w:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing:", "wp:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/drawingml/2006/main:", "a:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/drawingml/2006/picture:", "pic:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing:", "wp14:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.microsoft.com/office/drawing/2010/main:", "a14:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/officeDocument/2006/relationships:", "r:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://www.w3.org/XML/1998/namespace:", "xml:")
			// For attributes: space + uri: + name becomes space + prefix: + name
			// Note: attributes have a space before the namespace URI
			cleanedXML = strings.ReplaceAll(cleanedXML, " http://schemas.openxmlformats.org/wordprocessingml/2006/main:", " w:")
			cleanedXML = strings.ReplaceAll(cleanedXML, " http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing:", " wp:")
			cleanedXML = strings.ReplaceAll(cleanedXML, " http://schemas.openxmlformats.org/drawingml/2006/main:", " a:")
			cleanedXML = strings.ReplaceAll(cleanedXML, " http://schemas.openxmlformats.org/drawingml/2006/picture:", " pic:")
			cleanedXML = strings.ReplaceAll(cleanedXML, " http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing:", " wp14:")
			cleanedXML = strings.ReplaceAll(cleanedXML, " http://schemas.microsoft.com/office/drawing/2010/main:", " a14:")
			cleanedXML = strings.ReplaceAll(cleanedXML, " http://schemas.openxmlformats.org/officeDocument/2006/relationships:", " r:")
			cleanedXML = strings.ReplaceAll(cleanedXML, " http://www.w3.org/XML/1998/namespace:", " xml:")

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
		bodyContent := xmlStr[start+len("<w:document>"):end]
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
				sectContent := string(doc.Body.SectionProperties.Content)

				// Convert namespace URIs to prefixes (same as we do for raw XML elements)
				sectContent = strings.ReplaceAll(sectContent, "http://schemas.openxmlformats.org/wordprocessingml/2006/main:", "w:")
				sectContent = strings.ReplaceAll(sectContent, "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing:", "wp:")
				sectContent = strings.ReplaceAll(sectContent, "http://schemas.openxmlformats.org/drawingml/2006/main:", "a:")
				sectContent = strings.ReplaceAll(sectContent, "http://schemas.openxmlformats.org/drawingml/2006/picture:", "pic:")
				sectContent = strings.ReplaceAll(sectContent, "http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing:", "wp14:")
				sectContent = strings.ReplaceAll(sectContent, "http://schemas.microsoft.com/office/drawing/2010/main:", "a14:")
				sectContent = strings.ReplaceAll(sectContent, "http://schemas.openxmlformats.org/officeDocument/2006/relationships:", "r:")
				sectContent = strings.ReplaceAll(sectContent, "http://www.w3.org/XML/1998/namespace:", "xml:")

				// For attributes (with leading space)
				sectContent = strings.ReplaceAll(sectContent, " http://schemas.openxmlformats.org/wordprocessingml/2006/main:", " w:")
				sectContent = strings.ReplaceAll(sectContent, " http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing:", " wp:")
				sectContent = strings.ReplaceAll(sectContent, " http://schemas.openxmlformats.org/drawingml/2006/main:", " a:")
				sectContent = strings.ReplaceAll(sectContent, " http://schemas.openxmlformats.org/drawingml/2006/picture:", " pic:")
				sectContent = strings.ReplaceAll(sectContent, " http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing:", " wp14:")
				sectContent = strings.ReplaceAll(sectContent, " http://schemas.microsoft.com/office/drawing/2010/main:", " a14:")
				sectContent = strings.ReplaceAll(sectContent, " http://schemas.openxmlformats.org/officeDocument/2006/relationships:", " r:")
				sectContent = strings.ReplaceAll(sectContent, " http://www.w3.org/XML/1998/namespace:", " xml:")

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