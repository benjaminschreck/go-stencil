package stencil

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

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
	
	// Fix attribute namespaces
	xmlStr = strings.ReplaceAll(xmlStr, ` val="`, ` w:val="`)
	xmlStr = strings.ReplaceAll(xmlStr, ` type="`, ` w:type="`)
	xmlStr = strings.ReplaceAll(xmlStr, ` w="`, ` w:w="`)
	xmlStr = strings.ReplaceAll(xmlStr, ` ascii="`, ` w:ascii="`)

	// Replace markers with actual raw XML
	// Handle the case where markers might be concatenated in a single <w:t> element
	// by replacing just the marker text, not the entire <w:t>marker</w:t> pattern
	for marker, rawXML := range rawXMLMap {
		if strings.Contains(xmlStr, marker) {
			// Convert full namespace URIs to namespace prefixes in the raw XML
			cleanedXML := string(rawXML)
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/wordprocessingml/2006/main:", "w:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing:", "wp:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/drawingml/2006/main:", "a:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/drawingml/2006/picture:", "pic:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing:", "wp14:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.microsoft.com/office/drawing/2010/main:", "a14:")
			cleanedXML = strings.ReplaceAll(cleanedXML, "http://schemas.openxmlformats.org/officeDocument/2006/relationships:", "r:")

			xmlStr = strings.ReplaceAll(xmlStr, marker, cleanedXML)
		}
	}

	// Add proper document declaration and root element with namespaces
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
		`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
		`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
		`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture" ` +
		`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
		`xmlns:wp14="http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing" ` +
		`xmlns:a14="http://schemas.microsoft.com/office/drawing/2010/main">`)

	// Extract the body content (remove the outer document tags we added)
	start := strings.Index(xmlStr, "<w:document>")
	end := strings.LastIndex(xmlStr, "</w:document>")
	if start >= 0 && end > start {
		bodyContent := xmlStr[start+len("<w:document>"):end]
		buf.WriteString(bodyContent)
	}

	buf.WriteString(`</w:document>`)

	return buf.Bytes(), nil
}