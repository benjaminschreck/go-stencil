package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

func TestDOCXFragmentStyleConflictUsesFragmentFont(t *testing.T) {
	mainDoc := createDOCXWithCustomStylesAndBody(t, `
<w:p>
  <w:pPr><w:pStyle w:val="Listenabsatz"/></w:pPr>
  <w:r><w:t>Main item</w:t></w:r>
</w:p>
<w:p>
  <w:r><w:t>{{include "frag"}}</w:t></w:r>
</w:p>`, `
<w:style w:type="paragraph" w:styleId="Standard">
  <w:rPr><w:rFonts w:ascii="Aptos" w:hAnsi="Aptos" w:cs="Aptos"/></w:rPr>
</w:style>
<w:style w:type="paragraph" w:styleId="Listenabsatz">
  <w:basedOn w:val="Standard"/>
  <w:rPr><w:rFonts w:ascii="Aptos" w:hAnsi="Aptos" w:cs="Aptos"/></w:rPr>
</w:style>`)

	fragmentDoc := createDOCXWithCustomStylesAndBody(t, `
<w:p>
  <w:pPr><w:pStyle w:val="Listenabsatz"/></w:pPr>
  <w:r><w:t>Frag item</w:t></w:r>
</w:p>`, `
<w:style w:type="paragraph" w:styleId="Standard">
  <w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:cs="Times New Roman"/></w:rPr>
</w:style>
<w:style w:type="paragraph" w:styleId="Listenabsatz">
  <w:basedOn w:val="Standard"/>
</w:style>`)

	tmpl, err := ParseBytes(mainDoc)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("frag", fragmentDoc); err != nil {
		t.Fatalf("failed to add fragment: %v", err)
	}

	rendered, err := tmpl.RenderToBytes(TemplateData{})
	if err != nil {
		t.Fatalf("failed to render template: %v", err)
	}

	doc, err := ParseDocument(bytes.NewReader([]byte(extractDocumentXMLFromDOCX(t, rendered))))
	if err != nil {
		t.Fatalf("failed to parse rendered document.xml: %v", err)
	}

	var mainPara, fragPara *Paragraph
	for _, element := range doc.Body.Elements {
		para, ok := element.(*Paragraph)
		if !ok {
			continue
		}
		switch para.GetText() {
		case "Main item":
			mainPara = para
		case "Frag item":
			fragPara = para
		}
	}

	if mainPara == nil || fragPara == nil {
		t.Fatalf("expected both main and fragment paragraphs, got main=%v fragment=%v", mainPara != nil, fragPara != nil)
	}

	if mainPara.Properties != nil && mainPara.Properties.RunProperties != nil && mainPara.Properties.RunProperties.Font != nil && mainPara.Properties.RunProperties.Font.ASCII == "Times New Roman" {
		t.Fatalf("main paragraph unexpectedly received fragment font")
	}

	if fragPara.Properties == nil || fragPara.Properties.RunProperties == nil || fragPara.Properties.RunProperties.Font == nil {
		t.Fatalf("expected fragment paragraph to receive explicit font in pPr/rPr")
	}
	if got := fragPara.Properties.RunProperties.Font.ASCII; got != "Times New Roman" {
		t.Fatalf("fragment paragraph font = %q, want Times New Roman", got)
	}
	if len(fragPara.Runs) == 0 || fragPara.Runs[0].Properties == nil || fragPara.Runs[0].Properties.Font == nil {
		t.Fatalf("expected fragment run to receive explicit font")
	}
	if got := fragPara.Runs[0].Properties.Font.ASCII; got != "Times New Roman" {
		t.Fatalf("fragment run font = %q, want Times New Roman", got)
	}
}

func createDOCXWithCustomStylesAndBody(t *testing.T, bodyXML, stylesXML string) []byte {
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
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`)

	styles, err := w.Create("word/styles.xml")
	if err != nil {
		t.Fatalf("failed to create word/styles.xml: %v", err)
	}
	io.WriteString(styles, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`+stylesXML+`</w:styles>`)

	doc, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatalf("failed to create word/document.xml: %v", err)
	}
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>`+bodyXML+`
  </w:body>
</w:document>`)

	ct, err := w.Create("[Content_Types].xml")
	if err != nil {
		t.Fatalf("failed to create [Content_Types].xml: %v", err)
	}
	io.WriteString(ct, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`)

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return buf.Bytes()
}
