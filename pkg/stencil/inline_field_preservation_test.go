package stencil

import (
	"strings"
	"testing"
)

func TestRenderDocumentWithContext_PreservesFieldRunsInInlineIfParagraph(t *testing.T) {
	source := &Paragraph{
		Runs: []Run{
			textRun("Bestätigung der Aktivlegitimation der "),
			textRun("{{if length(aktivseite) >= 2}}"),
			textRun("Klägerparteien"),
			textRun("{{else}}"),
			textRun("Klagepartei"),
			textRun("{{end}}"),
			textRun(" durch die Finanzierungsbank "),
			textRun("{{leasingFinanzierung.vornameName}}"),
			textRun(" – Anlage K"),
			rawRun(`<w:fldChar w:fldCharType="begin"/>`),
			rawRun(`<w:instrText xml:space="preserve"> SEQ Anlage \* ARABIC </w:instrText>`),
			rawRun(`<w:fldChar w:fldCharType="separate"/>`),
			textRun("1"),
			rawRun(`<w:fldChar w:fldCharType="end"/>`),
		},
	}

	renderedDoc, err := RenderDocumentWithContext(&Document{
		Body: &Body{
			Elements: []BodyElement{cloneParagraph(source)},
		},
	}, TemplateData{
		"aktivseite": []any{
			map[string]any{"name": "A"},
			map[string]any{"name": "B"},
		},
		"leasingFinanzierung": map[string]any{
			"vornameName": "Musterbank GmbH",
		},
	}, &renderContext{})
	if err != nil {
		t.Fatalf("RenderDocumentWithContext failed: %v", err)
	}

	if renderedDoc.Body == nil || len(renderedDoc.Body.Elements) != 1 {
		t.Fatalf("expected one rendered paragraph, got %+v", renderedDoc.Body)
	}

	renderedPara, ok := renderedDoc.Body.Elements[0].(*Paragraph)
	if !ok {
		t.Fatalf("expected rendered element to be paragraph, got %T", renderedDoc.Body.Elements[0])
	}

	if got, want := countRawRunElementMatches(source, "<w:fldChar"), countRawRunElementMatches(renderedPara, "<w:fldChar"); got != want {
		t.Fatalf("fldChar count changed: before=%d after=%d", got, want)
	}
	if got, want := countRawRunElementMatches(source, "<w:instrText"), countRawRunElementMatches(renderedPara, "<w:instrText"); got != want {
		t.Fatalf("instrText count changed: before=%d after=%d", got, want)
	}

	text := renderedPara.GetText()
	if strings.Contains(text, "{{") {
		t.Fatalf("expected inline control tags to be rendered, got text %q", text)
	}
	if !strings.Contains(text, "Klägerparteien") {
		t.Fatalf("expected rendered text to contain plural branch, got %q", text)
	}
	if !strings.Contains(text, "Musterbank GmbH") {
		t.Fatalf("expected rendered text to contain rendered bank name, got %q", text)
	}
}

func textRun(text string) Run {
	return Run{
		Text: &Text{
			Content: text,
			Space:   "preserve",
		},
	}
}

func rawRun(raw string) Run {
	return Run{
		RawXML: []RawXMLElement{
			{Content: []byte(raw)},
		},
	}
}
