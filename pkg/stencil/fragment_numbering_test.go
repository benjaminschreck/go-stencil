package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"
)

func TestDOCXFragmentPreservesNumberingDefinitions(t *testing.T) {
	templateDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:r><w:t>{{include "roman"}}</w:t></w:r></w:p>`,
		decimalNumberingXML(),
		true,
	)
	fragmentDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Roman item</w:t></w:r></w:p>`,
		upperRomanNumberingXML(),
		true,
	)

	tmpl, err := ParseBytes(templateDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("roman", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(nil)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	documentXML := readDOCXPart(t, rendered, "word/document.xml")
	numberingXML := readDOCXPart(t, rendered, "word/numbering.xml")

	if !strings.Contains(numberingXML, `w:numFmt w:val="upperRoman"`) {
		t.Fatalf("expected merged numbering.xml to contain upperRoman format, got:\n%s", numberingXML)
	}
	if strings.Contains(documentXML, `<w:numId w:val="1"/>`) {
		t.Fatalf("expected fragment numbering to be remapped away from template numId=1, got:\n%s", documentXML)
	}
	if !strings.Contains(documentXML, `<w:numId w:val="2"></w:numId>`) && !strings.Contains(documentXML, `<w:numId w:val="2"/>`) {
		t.Fatalf("expected rendered fragment paragraph to reference remapped numId=2, got:\n%s", documentXML)
	}
}

func TestDOCXFragmentAddsNumberingPartWhenTemplateHasNone(t *testing.T) {
	templateDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:r><w:t>{{include "roman"}}</w:t></w:r></w:p>`,
		"",
		false,
	)
	fragmentDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Roman item</w:t></w:r></w:p>`,
		lowerRomanNumberingXML(),
		true,
	)

	tmpl, err := ParseBytes(templateDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("roman", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(nil)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	numberingXML := readDOCXPart(t, rendered, "word/numbering.xml")
	relsXML := readDOCXPart(t, rendered, "word/_rels/document.xml.rels")
	contentTypesXML := readDOCXPart(t, rendered, "[Content_Types].xml")

	if !strings.Contains(numberingXML, `w:numFmt w:val="lowerRoman"`) {
		t.Fatalf("expected generated numbering.xml to contain lowerRoman format, got:\n%s", numberingXML)
	}
	if !strings.Contains(relsXML, numberingRelationType) {
		t.Fatalf("expected document relationships to include numbering relationship, got:\n%s", relsXML)
	}
	if !strings.Contains(contentTypesXML, numberingContentType) {
		t.Fatalf("expected content types to include numbering override, got:\n%s", contentTypesXML)
	}
}

func TestDOCXFragmentPreservesRomanStyleBasedNumberingWhenStyleIDCollides(t *testing.T) {
	templateDoc := createDOCXWithOptionalStylesAndNumbering(t,
		`<w:p><w:r><w:t>{{include "frag"}}</w:t></w:r></w:p>`,
		paragraphListStyleXML("ListStyle", "Arial", "1"),
		decimalNumberingXML(),
		true,
		true,
	)
	fragmentDoc := createDOCXWithOptionalStylesAndNumbering(t,
		`<w:p><w:pPr><w:pStyle w:val="ListStyle"/></w:pPr><w:r><w:t>Roman item</w:t></w:r></w:p>`,
		paragraphListStyleXML("ListStyle", "Times New Roman", "1"),
		upperRomanNumberingXML(),
		true,
		true,
	)

	tmpl, err := ParseBytes(templateDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(nil)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	documentXML := readDOCXPart(t, rendered, "word/document.xml")
	stylesXML := readDOCXPart(t, rendered, "word/styles.xml")
	numberingXML := readDOCXPart(t, rendered, "word/numbering.xml")

	if !strings.Contains(numberingXML, `w:numFmt w:val="upperRoman"`) {
		t.Fatalf("expected merged numbering.xml to contain upperRoman format, got:\n%s", numberingXML)
	}
	if !strings.Contains(stylesXML, `w:styleId="ListStyle__frag"`) {
		t.Fatalf("expected fragment style to be remapped, got:\n%s", stylesXML)
	}

	doc, err := ParseDocument(bytes.NewReader([]byte(documentXML)))
	if err != nil {
		t.Fatalf("failed to parse rendered document.xml: %v", err)
	}
	para := findParagraphByText(t, doc, "Roman item")
	if para.Properties == nil || para.Properties.Style == nil {
		t.Fatalf("expected rendered paragraph to keep a paragraph style, got %+v", para.Properties)
	}
	if para.Properties.Style.Val != "ListStyle__frag" {
		t.Fatalf("expected rendered paragraph style to be remapped, got %q", para.Properties.Style.Val)
	}

	styles, err := parseStyles([]byte(stylesXML))
	if err != nil {
		t.Fatalf("failed to parse styles.xml: %v", err)
	}
	style := findStyleByID(t, styles, "ListStyle__frag")
	if !strings.Contains(string(style.RawXML), `w:rFonts w:ascii="Times New Roman"`) {
		t.Fatalf("expected remapped fragment style to preserve Times New Roman, got:\n%s", string(style.RawXML))
	}
	if !strings.Contains(string(style.RawXML), `<w:numId w:val="2"/>`) {
		t.Fatalf("expected remapped fragment style to reference remapped numId=2, got:\n%s", string(style.RawXML))
	}
}

func TestDOCXFragmentPreservesBulletStyleBasedNumberingWhenStyleIDCollides(t *testing.T) {
	templateDoc := createDOCXWithOptionalStylesAndNumbering(t,
		`<w:p><w:r><w:t>{{include "frag"}}</w:t></w:r></w:p>`,
		paragraphListStyleXML("ListStyle", "Arial", "1"),
		decimalNumberingXML(),
		true,
		true,
	)
	fragmentDoc := createDOCXWithOptionalStylesAndNumbering(t,
		`<w:p><w:pPr><w:pStyle w:val="ListStyle"/></w:pPr><w:r><w:t>Bullet item</w:t></w:r></w:p>`,
		paragraphListStyleXML("ListStyle", "Times New Roman", "1"),
		bulletNumberingXML(),
		true,
		true,
	)

	tmpl, err := ParseBytes(templateDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(nil)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	numberingXML := readDOCXPart(t, rendered, "word/numbering.xml")
	stylesXML := readDOCXPart(t, rendered, "word/styles.xml")

	if !strings.Contains(numberingXML, `w:numFmt w:val="bullet"`) {
		t.Fatalf("expected merged numbering.xml to contain bullet format, got:\n%s", numberingXML)
	}

	styles, err := parseStyles([]byte(stylesXML))
	if err != nil {
		t.Fatalf("failed to parse styles.xml: %v", err)
	}
	style := findStyleByID(t, styles, "ListStyle__frag")
	if !strings.Contains(string(style.RawXML), `<w:numId w:val="2"/>`) {
		t.Fatalf("expected remapped fragment style to reference remapped numId=2, got:\n%s", string(style.RawXML))
	}
}

func TestDOCXFragmentRenumbersDuplicateNumberingSignatures(t *testing.T) {
	templateDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:r><w:t>{{include "frag"}}</w:t></w:r></w:p>`,
		signedNumberingXML("bullet", "•", "AAAABBBB", "CCCCDDDD", `<w:rPr><w:rFonts w:ascii="Symbol" w:hAnsi="Symbol"/></w:rPr>`),
		true,
	)
	fragmentDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Bullet item</w:t></w:r></w:p>`,
		signedNumberingXML("upperRoman", "%1.", "AAAABBBB", "CCCCDDDD", ""),
		true,
	)

	tmpl, err := ParseBytes(templateDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(nil)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	numberingXML := readDOCXPart(t, rendered, "word/numbering.xml")
	assertUniqueNumberingSignatures(t, numberingXML)
	assertNumberingBlockOrder(t, numberingXML)
	if !strings.Contains(numberingXML, `w:numFmt w:val="upperRoman"`) {
		t.Fatalf("expected merged numbering.xml to contain upperRoman format, got:\n%s", numberingXML)
	}
}

func createDOCXWithOptionalNumbering(t *testing.T, bodyXML, numberingXML string, includeNumberingRelationship bool) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	rels, err := w.Create("_rels/.rels")
	if err != nil {
		t.Fatalf("failed to create _rels/.rels: %v", err)
	}
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	wordRels, err := w.Create("word/_rels/document.xml.rels")
	if err != nil {
		t.Fatalf("failed to create word/_rels/document.xml.rels: %v", err)
	}
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	if includeNumberingRelationship && numberingXML != "" {
		io.WriteString(wordRels, `
  <Relationship Id="rId1" Type="`+numberingRelationType+`" Target="numbering.xml"/>`)
	}
	io.WriteString(wordRels, `
</Relationships>`)

	doc, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatalf("failed to create word/document.xml: %v", err)
	}
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>`+bodyXML+`
  </w:body>
</w:document>`)

	if numberingXML != "" {
		numberingPart, err := w.Create("word/numbering.xml")
		if err != nil {
			t.Fatalf("failed to create word/numbering.xml: %v", err)
		}
		io.WriteString(numberingPart, numberingXML)
	}

	contentTypes, err := w.Create("[Content_Types].xml")
	if err != nil {
		t.Fatalf("failed to create [Content_Types].xml: %v", err)
	}
	io.WriteString(contentTypes, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`)
	if numberingXML != "" {
		io.WriteString(contentTypes, `
  <Override PartName="/word/numbering.xml" ContentType="`+numberingContentType+`"/>`)
	}
	io.WriteString(contentTypes, `
</Types>`)

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return buf.Bytes()
}

func createDOCXWithOptionalStylesAndNumbering(t *testing.T, bodyXML, stylesXML, numberingXML string, includeStylesRelationship, includeNumberingRelationship bool) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	rels, err := w.Create("_rels/.rels")
	if err != nil {
		t.Fatalf("failed to create _rels/.rels: %v", err)
	}
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	wordRels, err := w.Create("word/_rels/document.xml.rels")
	if err != nil {
		t.Fatalf("failed to create word/_rels/document.xml.rels: %v", err)
	}
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	if includeStylesRelationship && stylesXML != "" {
		io.WriteString(wordRels, `
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`)
	}
	if includeNumberingRelationship && numberingXML != "" {
		io.WriteString(wordRels, `
  <Relationship Id="rId2" Type="`+numberingRelationType+`" Target="numbering.xml"/>`)
	}
	io.WriteString(wordRels, `
</Relationships>`)

	doc, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatalf("failed to create word/document.xml: %v", err)
	}
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>`+bodyXML+`
  </w:body>
</w:document>`)

	if stylesXML != "" {
		stylesPart, err := w.Create("word/styles.xml")
		if err != nil {
			t.Fatalf("failed to create word/styles.xml: %v", err)
		}
		io.WriteString(stylesPart, stylesXML)
	}

	if numberingXML != "" {
		numberingPart, err := w.Create("word/numbering.xml")
		if err != nil {
			t.Fatalf("failed to create word/numbering.xml: %v", err)
		}
		io.WriteString(numberingPart, numberingXML)
	}

	contentTypes, err := w.Create("[Content_Types].xml")
	if err != nil {
		t.Fatalf("failed to create [Content_Types].xml: %v", err)
	}
	io.WriteString(contentTypes, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`)
	if stylesXML != "" {
		io.WriteString(contentTypes, `
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`)
	}
	if numberingXML != "" {
		io.WriteString(contentTypes, `
  <Override PartName="/word/numbering.xml" ContentType="`+numberingContentType+`"/>`)
	}
	io.WriteString(contentTypes, `
</Types>`)

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return buf.Bytes()
}

func readDOCXPart(t *testing.T, docxBytes []byte, partName string) string {
	t.Helper()

	r, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		t.Fatalf("failed to read rendered DOCX: %v", err)
	}

	for _, file := range r.File {
		if file.Name != partName {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("failed to open %s: %v", partName, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("failed to read %s: %v", partName, err)
		}
		return string(data)
	}

	t.Fatalf("part %s not found in rendered DOCX", partName)
	return ""
}

func decimalNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="1">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="decimal"/>
      <w:lvlText w:val="%1."/>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1">
    <w:abstractNumId w:val="1"/>
  </w:num>
</w:numbering>`
}

func upperRomanNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="1">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="upperRoman"/>
      <w:lvlText w:val="%1."/>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1">
    <w:abstractNumId w:val="1"/>
  </w:num>
</w:numbering>`
}

func lowerRomanNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="1">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="lowerRoman"/>
      <w:lvlText w:val="%1."/>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1">
    <w:abstractNumId w:val="1"/>
  </w:num>
</w:numbering>`
}

func bulletNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="1">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="bullet"/>
      <w:lvlText w:val="•"/>
      <w:rPr><w:rFonts w:ascii="Symbol" w:hAnsi="Symbol"/></w:rPr>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1">
    <w:abstractNumId w:val="1"/>
  </w:num>
</w:numbering>`
}

func signedNumberingXML(numFmt, lvlText, nsid, tmpl, extraLevelXML string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="1">
    <w:nsid w:val="` + nsid + `"/>
    <w:tmpl w:val="` + tmpl + `"/>
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="` + numFmt + `"/>
      <w:lvlText w:val="` + lvlText + `"/>` + extraLevelXML + `
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1">
    <w:abstractNumId w:val="1"/>
  </w:num>
</w:numbering>`
}

func paragraphListStyleXML(styleID, fontName, numID string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="` + styleID + `">
    <w:name w:val="` + styleID + `"/>
    <w:pPr>
      <w:numPr>
        <w:ilvl w:val="0"/>
        <w:numId w:val="` + numID + `"/>
      </w:numPr>
    </w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="` + fontName + `" w:hAnsi="` + fontName + `" w:cs="` + fontName + `"/>
    </w:rPr>
  </w:style>
</w:styles>`
}

func findParagraphByText(t *testing.T, doc *Document, text string) *Paragraph {
	t.Helper()

	if doc == nil || doc.Body == nil {
		t.Fatal("document body is nil")
	}

	for _, elem := range doc.Body.Elements {
		para, ok := elem.(*Paragraph)
		if !ok {
			continue
		}
		if para.GetText() == text {
			return para
		}
	}

	t.Fatalf("paragraph %q not found", text)
	return nil
}

func findStyleByID(t *testing.T, styles *Styles, styleID string) DocumentStyle {
	t.Helper()

	if styles == nil {
		t.Fatal("styles are nil")
	}

	for _, style := range styles.Styles {
		if style.StyleID == styleID {
			return style
		}
	}

	t.Fatalf("style %q not found", styleID)
	return DocumentStyle{}
}

func assertUniqueNumberingSignatures(t *testing.T, numberingXML string) {
	t.Helper()

	abstracts := numberingAbstractBlockRegex.FindAllString(numberingXML, -1)
	signatures := make(map[string]bool)
	for _, block := range abstracts {
		nsid, okNSID := extractNumberingMatch(block, regexp.MustCompile(`<w:nsid\b[^>]*\bw:val="([0-9A-Fa-f]{8})"\s*/?>`))
		tmpl, okTmpl := extractNumberingMatch(block, regexp.MustCompile(`<w:tmpl\b[^>]*\bw:val="([0-9A-Fa-f]{8})"\s*/?>`))
		if !okNSID || !okTmpl {
			continue
		}
		signature := strings.ToUpper(nsid) + ":" + strings.ToUpper(tmpl)
		if signatures[signature] {
			t.Fatalf("expected numbering signatures to be unique, duplicate %s in:\n%s", signature, numberingXML)
		}
		signatures[signature] = true
	}
}

func assertNumberingBlockOrder(t *testing.T, numberingXML string) {
	t.Helper()

	seenNum := false
	for _, match := range regexp.MustCompile(`<w:(abstractNum|num)\b`).FindAllStringSubmatch(numberingXML, -1) {
		switch match[1] {
		case "num":
			seenNum = true
		case "abstractNum":
			if seenNum {
				t.Fatalf("expected all abstractNum blocks before any num block, got:\n%s", numberingXML)
			}
		}
	}
}
