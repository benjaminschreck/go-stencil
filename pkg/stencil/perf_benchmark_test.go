package stencil

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkMarshalDocumentWithNamespaces_RawXML(b *testing.B) {
	doc := &Document{
		Attrs: []xml.Attr{
			{Name: xml.Name{Local: "xmlns:w"}, Value: "http://schemas.openxmlformats.org/wordprocessingml/2006/main"},
			{Name: xml.Name{Local: "xmlns:r"}, Value: "http://schemas.openxmlformats.org/officeDocument/2006/relationships"},
		},
		Body: &Body{
			Elements: []BodyElement{
				&Paragraph{
					Runs: []Run{
						{
							Properties: &RunProperties{Bold: &Empty{}},
							Text:       &Text{Content: "Lead"},
						},
						{
							RawXML: []RawXMLElement{
								{
									Content: []byte(`<w:drawing xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"></w:drawing>`),
								},
							},
						},
					},
				},
				&Table{
					Rows: []TableRow{
						{
							Cells: []TableCell{
								{
									Paragraphs: []Paragraph{
										{
											Runs: []Run{
												{
													Properties: &RunProperties{Italic: &Empty{}},
													Text:       &Text{Content: "Cell"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rendered := cloneDocument(doc)
		if _, err := marshalDocumentWithNamespaces(rendered); err != nil {
			b.Fatalf("marshalDocumentWithNamespaces failed: %v", err)
		}
	}
}

func BenchmarkEnsureCompiledFragmentDefinitions(b *testing.B) {
	numberingXML := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="1"><w:lvl w:ilvl="0"><w:start w:val="1"/></w:lvl></w:abstractNum>
  <w:abstractNum w:abstractNumId="2"><w:lvl w:ilvl="0"><w:start w:val="1"/></w:lvl></w:abstractNum>
  <w:num w:numId="1"><w:abstractNumId w:val="1"/></w:num>
  <w:num w:numId="2"><w:abstractNumId w:val="2"/></w:num>
</w:numbering>`)
	stylesXML := []byte(`<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style><w:pPr><w:numPr><w:numId w:val="1"/></w:numPr></w:pPr></w:style></w:styles>`)
	compiled, err := compileFragmentNumbering(numberingXML, stylesXML)
	if err != nil {
		b.Fatalf("compileFragmentNumbering failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := &numberingContext{
			xml:               defaultNumberingXML(),
			nextAbstractNumID: fragmentNumberingIDFloor,
			nextNumID:         fragmentNumberingIDFloor,
			fragmentNumMaps:   make(map[string]map[string]string),
			fragmentStylesXML: make(map[string][]byte),
		}
		if _, err := ctx.ensureCompiledFragmentDefinitions("bench-frag", compiled); err != nil {
			b.Fatalf("ensureCompiledFragmentDefinitions failed: %v", err)
		}
	}
}

func BenchmarkRenderParagraphWithPlan_ProofErrSplit(b *testing.B) {
	para := &Paragraph{
		Content: []ParagraphContent{
			&Run{Text: &Text{Content: "{{"}},
			&ProofErr{Type: "spellStart"},
			&Run{Text: &Text{Content: "customer"}},
			&ProofErr{Type: "spellEnd"},
			&Run{Text: &Text{Content: ".name}}"}},
		},
	}
	plan := compileParagraphRenderPlan(para)
	ctx := &renderContext{
		paragraphPlans: map[*Paragraph]*paragraphRenderPlan{
			para: plan,
		},
	}
	data := TemplateData{
		"customer": map[string]interface{}{"name": "Benchmark User"},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rendered, err := RenderParagraphWithContext(para, data, ctx)
		if err != nil {
			b.Fatalf("RenderParagraphWithContext failed: %v", err)
		}
		if rendered == nil {
			b.Fatal("RenderParagraphWithContext returned nil paragraph")
		}
	}
}

func BenchmarkRenderHeaderOrFooter_Template(b *testing.B) {
	partZip := new(bytes.Buffer)
	zipWriter := zip.NewWriter(partZip)
	header, err := zipWriter.Create("word/header1.xml")
	if err != nil {
		b.Fatalf("Create header part failed: %v", err)
	}
	headerXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>Hello {{name}}</w:t></w:r></w:p></w:hdr>`
	if _, err := header.Write([]byte(headerXML)); err != nil {
		b.Fatalf("Write header part failed: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		b.Fatalf("Close header zip failed: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(partZip.Bytes()), int64(partZip.Len()))
	if err != nil {
		b.Fatalf("NewReader failed: %v", err)
	}
	var headerPart *zip.File
	for _, file := range reader.File {
		if file.Name == "word/header1.xml" {
			headerPart = file
			break
		}
	}
	if headerPart == nil {
		b.Fatal("header part not found")
	}

	data := TemplateData{"name": "Benchmark Header"}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		content, err := renderHeaderOrFooter(headerPart, data, &renderContext{})
		if err != nil {
			b.Fatalf("renderHeaderOrFooter failed: %v", err)
		}
		if len(content) == 0 {
			b.Fatal("renderHeaderOrFooter returned empty content")
		}
	}
}

func BenchmarkRenderProductionShape_Synthetic(b *testing.B) {
	var includeLines strings.Builder
	for i := 0; i < 240; i++ {
		includeLines.WriteString(fmt.Sprintf("{{include \"frag_%03d\"}}\n", i))
	}

	reader := createBenchDocx(b, `{{include "root"}}`)
	tmpl, err := Prepare(reader)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 240; i++ {
		if err := tmpl.AddFragment(fmt.Sprintf("frag_%03d", i), fmt.Sprintf("Fragment %03d {{field_%03d}}", i, i)); err != nil {
			b.Fatalf("AddFragment failed: %v", err)
		}
	}
	if err := tmpl.AddFragment("root_leaf", includeLines.String()); err != nil {
		b.Fatalf("AddFragment root_leaf failed: %v", err)
	}
	if err := tmpl.AddFragment("root_4", `{{include "root_leaf"}}{{include "frag_000"}}`); err != nil {
		b.Fatalf("AddFragment root_4 failed: %v", err)
	}
	if err := tmpl.AddFragment("root_3", `{{include "root_4"}}{{include "frag_001"}}`); err != nil {
		b.Fatalf("AddFragment root_3 failed: %v", err)
	}
	if err := tmpl.AddFragment("root_2", `{{include "root_3"}}{{include "frag_002"}}`); err != nil {
		b.Fatalf("AddFragment root_2 failed: %v", err)
	}
	if err := tmpl.AddFragment("root_1", `{{include "root_2"}}{{include "frag_003"}}`); err != nil {
		b.Fatalf("AddFragment root_1 failed: %v", err)
	}
	if err := tmpl.AddFragment("root", `{{include "root_1"}}{{include "frag_004"}}`); err != nil {
		b.Fatalf("AddFragment root failed: %v", err)
	}

	data := TemplateData{}
	for i := 0; i < 700; i++ {
		data[fmt.Sprintf("field_%03d", i)] = fmt.Sprintf("value-%03d", i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		output, err := tmpl.Render(data)
		if err != nil {
			b.Fatalf("Render failed: %v", err)
		}
		if output == nil {
			b.Fatal("Render returned nil output")
		}
	}
}
