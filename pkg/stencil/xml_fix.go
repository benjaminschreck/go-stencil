package stencil

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// marshalDocumentWithNamespaces marshals a document with proper namespaces
func marshalDocumentWithNamespaces(doc *Document) ([]byte, error) {
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
	
	// Add proper document declaration and root element with namespaces
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	buf.WriteString("\n")
	buf.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	
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