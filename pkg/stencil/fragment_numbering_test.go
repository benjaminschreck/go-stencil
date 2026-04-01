package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"strconv"
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
	remappedNumID := strconv.Itoa(fragmentNumberingIDFloor)
	if !strings.Contains(documentXML, `<w:numId w:val="`+remappedNumID+`"></w:numId>`) &&
		!strings.Contains(documentXML, `<w:numId w:val="`+remappedNumID+`"/>`) {
		t.Fatalf("expected rendered fragment paragraph to reference remapped numId=%s, got:\n%s", remappedNumID, documentXML)
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

func TestDOCXFragmentPreservesBlankNumberedParagraphsForWord(t *testing.T) {
	templateDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:r><w:t>{{include "antraege"}}</w:t></w:r></w:p>`,
		"",
		false,
	)
	fragmentDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr></w:p>`+
			`<w:p><w:r><w:t>First entry</w:t></w:r></w:p>`+
			`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="2"/></w:numPr></w:pPr></w:p>`+
			`<w:p><w:r><w:t>Second entry</w:t></w:r></w:p>`,
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="1">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="upperRoman"/>
      <w:lvlText w:val="%1."/>
    </w:lvl>
  </w:abstractNum>
  <w:abstractNum w:abstractNumId="2">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="bullet"/>
      <w:lvlText w:val="•"/>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1">
    <w:abstractNumId w:val="1"/>
  </w:num>
  <w:num w:numId="2">
    <w:abstractNumId w:val="2"/>
  </w:num>
</w:numbering>`,
		true,
	)

	tmpl, err := ParseBytes(templateDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("antraege", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(nil)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	documentXML := readDOCXPart(t, rendered, "word/document.xml")

	if !strings.Contains(documentXML, numberedParagraphAnchor) {
		t.Fatalf("expected rendered numbered paragraph to contain invisible anchor for Word, got:\n%s", documentXML)
	}
	if !strings.Contains(documentXML, "First entry") || !strings.Contains(documentXML, "Second entry") {
		t.Fatalf("expected rendered document to keep fragment text paragraphs, got:\n%s", documentXML)
	}
}

func TestNestedDOCXFragmentNumberingDoesNotCollideWithOuterFragmentIDs(t *testing.T) {
	templateDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:r><w:t>{{include "outer"}}</w:t></w:r></w:p>`,
		templateNumberingXMLWithCount(11),
		true,
	)
	outerDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:r><w:t>{{include "inner"}}</w:t></w:r></w:p>`,
		numberingXMLWithSingleFormat(12, 12, "upperRoman", "%1."),
		true,
	)
	innerDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Inner bullet</w:t></w:r></w:p>`,
		numberingXMLWithSingleFormat(1, 1, "bullet", "•"),
		true,
	)

	tmpl, err := ParseBytes(templateDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("outer", outerDoc); err != nil {
		t.Fatalf("failed to add outer fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", innerDoc); err != nil {
		t.Fatalf("failed to add inner fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(nil)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	documentXML := readDOCXPart(t, rendered, "word/document.xml")
	numberingXML := readDOCXPart(t, rendered, "word/numbering.xml")
	expectedNumID := strconv.Itoa(fragmentNumberingIDFloor)

	if !strings.Contains(documentXML, "Inner bullet") {
		t.Fatalf("expected rendered document to contain inner fragment text, got:\n%s", documentXML)
	}
	if !strings.Contains(documentXML, `<w:numId w:val="`+expectedNumID+`"></w:numId>`) &&
		!strings.Contains(documentXML, `<w:numId w:val="`+expectedNumID+`"/>`) {
		t.Fatalf("expected inner fragment to keep remapped numId=%s, got:\n%s", expectedNumID, documentXML)
	}

	if !strings.Contains(numberingXML, `<w:num w:numId="`+expectedNumID+`">`) {
		t.Fatalf("expected numbering.xml to contain inner num definition for numId=%s, got:\n%s", expectedNumID, numberingXML)
	}
	if !strings.Contains(numberingXML, `<w:abstractNumId w:val="`+expectedNumID+`"/>`) {
		t.Fatalf("expected numbering.xml to contain inner abstractNumId=%s, got:\n%s", expectedNumID, numberingXML)
	}

	if !strings.Contains(numberingXML, `<w:numFmt w:val="bullet"/>`) {
		t.Fatalf("expected merged numbering definitions to contain bullet numbering, got:\n%s", numberingXML)
	}
}

func TestMergedNumberingKeepsAbstractNumsBeforeNums(t *testing.T) {
	templateDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:r><w:t>{{include "roman"}}</w:t></w:r></w:p><w:p><w:r><w:t>{{include "bullet"}}</w:t></w:r></w:p>`,
		decimalNumberingXML(),
		true,
	)
	romanDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Roman item</w:t></w:r></w:p>`,
		upperRomanNumberingXML(),
		true,
	)
	bulletDoc := createDOCXWithOptionalNumbering(t,
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Bullet item</w:t></w:r></w:p>`,
		`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="1">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="bullet"/>
      <w:lvlText w:val="•"/>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1">
    <w:abstractNumId w:val="1"/>
  </w:num>
</w:numbering>`,
		true,
	)

	tmpl, err := ParseBytes(templateDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("roman", romanDoc); err != nil {
		t.Fatalf("failed to add roman fragment: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("bullet", bulletDoc); err != nil {
		t.Fatalf("failed to add bullet fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(nil)
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	numberingXML := readDOCXPart(t, rendered, "word/numbering.xml")
	firstNum := strings.Index(numberingXML, `<w:num `)
	if firstNum == -1 {
		t.Fatalf("expected numbering.xml to contain <w:num>, got:\n%s", numberingXML)
	}
	if abstractAfter := strings.Index(numberingXML[firstNum:], `<w:abstractNum `); abstractAfter != -1 {
		t.Fatalf("expected numbering.xml to keep all abstractNum blocks before num blocks, got:\n%s", numberingXML)
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

func templateNumberingXMLWithCount(count int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	b.WriteString(`<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` + "\n")
	for i := 1; i <= count; i++ {
		id := strconv.Itoa(i)
		b.WriteString(`  <w:abstractNum w:abstractNumId="` + id + `">` + "\n")
		b.WriteString(`    <w:lvl w:ilvl="0">` + "\n")
		b.WriteString(`      <w:start w:val="1"/>` + "\n")
		b.WriteString(`      <w:numFmt w:val="decimal"/>` + "\n")
		b.WriteString(`      <w:lvlText w:val="%1."/>` + "\n")
		b.WriteString(`    </w:lvl>` + "\n")
		b.WriteString(`  </w:abstractNum>` + "\n")
		b.WriteString(`  <w:num w:numId="` + id + `">` + "\n")
		b.WriteString(`    <w:abstractNumId w:val="` + id + `"/>` + "\n")
		b.WriteString(`  </w:num>` + "\n")
	}
	b.WriteString(`</w:numbering>`)
	return b.String()
}

func numberingXMLWithSingleFormat(abstractID, numID int, format, levelText string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="` + strconv.Itoa(abstractID) + `">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="` + format + `"/>
      <w:lvlText w:val="` + levelText + `"/>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="` + strconv.Itoa(numID) + `">
    <w:abstractNumId w:val="` + strconv.Itoa(abstractID) + `"/>
  </w:num>
</w:numbering>`
}
