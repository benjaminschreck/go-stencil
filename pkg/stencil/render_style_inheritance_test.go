package stencil

import "testing"

func TestRenderParagraphWithContext_MaterializesInheritedFontFromParagraphStyle(t *testing.T) {
	styleCtx := newTestStyleContextFromXML(t, `
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Standard">
    <w:rPr>
      <w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:cs="Times New Roman" w:eastAsia="Times New Roman"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Listenabsatz">
    <w:basedOn w:val="Standard"/>
  </w:style>
</w:styles>`)

	para := &Paragraph{
		Properties: &ParagraphProperties{
			Style: &Style{Val: "Listenabsatz"},
		},
		Runs: []Run{
			{
				Properties: &RunProperties{
					Size:   &Size{Val: 24},
					SizeCs: &Size{Val: 24},
				},
				Text: &Text{Content: "{{if polizei}}- {{text}}{{else}}- Ersatz{{end}}"},
			},
		},
	}

	rendered, err := RenderParagraphWithContext(para, TemplateData{
		"polizei": true,
		"text":    "Beiziehung der Ermittlungsakte",
	}, &renderContext{styles: styleCtx})
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error: %v", err)
	}

	if len(rendered.Runs) == 0 {
		t.Fatal("expected rendered paragraph to contain runs")
	}

	for i, run := range rendered.Runs {
		if run.Text == nil && run.Break == nil {
			continue
		}
		if run.Properties == nil || run.Properties.Font == nil {
			t.Fatalf("expected run %d to have a materialized font", i)
		}
		if got := run.Properties.Font.ASCII; got != "Times New Roman" {
			t.Fatalf("expected run %d font ASCII to be Times New Roman, got %q", i, got)
		}
	}
}

func TestStyleContext_WithRenderingStyles_ShadowsConflictingMainStyleFont(t *testing.T) {
	mainCtx := newTestStyleContextFromXML(t, `
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Standard">
    <w:rPr>
      <w:rFonts w:asciiTheme="minorHAnsi" w:hAnsiTheme="minorHAnsi" w:csTheme="minorBidi" w:eastAsiaTheme="minorEastAsia"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Listenabsatz">
    <w:basedOn w:val="Standard"/>
  </w:style>
</w:styles>`)

	fragmentCtx, err := mainCtx.withRenderingStyles([]byte(`
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Standard">
    <w:rPr>
      <w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:cs="Times New Roman" w:eastAsia="Times New Roman"/>
    </w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Listenabsatz">
    <w:basedOn w:val="Standard"/>
  </w:style>
</w:styles>`))
	if err != nil {
		t.Fatalf("withRenderingStyles returned error: %v", err)
	}

	para := &Paragraph{
		Properties: &ParagraphProperties{
			Style: &Style{Val: "Listenabsatz"},
		},
		Runs: []Run{
			{
				Text: &Text{Content: "Beweiszeile"},
			},
		},
	}

	fragmentCtx.materializeInheritedFonts(para)

	if len(para.Runs) != 1 || para.Runs[0].Properties == nil || para.Runs[0].Properties.Font == nil {
		t.Fatal("expected fragment paragraph run to have a materialized font")
	}
	if got := para.Runs[0].Properties.Font.ASCII; got != "Times New Roman" {
		t.Fatalf("expected fragment style inheritance to resolve to Times New Roman, got %q", got)
	}
}

func newTestStyleContextFromXML(t *testing.T, stylesXML string) *styleContext {
	t.Helper()

	ctx := &styleContext{
		existingStyles:       make(map[string]styleSignature),
		fragmentStyleMaps:    make(map[string]map[string]string),
		fragmentStylesXML:    make(map[string][]byte),
		fragmentNumberingXML: make(map[string][]byte),
		styleDefinitions:     make(map[string]styleDefinition),
	}
	if err := ctx.registerStyles([]byte(stylesXML)); err != nil {
		t.Fatalf("registerStyles returned error: %v", err)
	}
	return ctx
}
