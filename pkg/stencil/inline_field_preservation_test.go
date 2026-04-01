package stencil

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderDocumentWithContext_PreservesFieldRunsInInlineIfParagraph(t *testing.T) {
	docxPath := filepath.Join("..", "..", "examples", "klage", "fragments", "vorlage.docx")

	docXML := mustReadDocxPart(t, docxPath, "word/document.xml")
	doc, err := ParseDocument(bytes.NewReader(docXML))
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	source := findFieldParagraphByText(t, doc, "Bestätigung der Aktivlegitimation der")
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

func findFieldParagraphByText(t *testing.T, doc *Document, needle string) *Paragraph {
	t.Helper()

	if doc == nil || doc.Body == nil {
		t.Fatal("document body is nil")
	}

	for _, elem := range doc.Body.Elements {
		para, ok := elem.(*Paragraph)
		if !ok {
			continue
		}
		if !strings.Contains(para.GetText(), needle) {
			continue
		}
		if countRawRunElementMatches(para, "<w:fldChar") == 0 {
			continue
		}
		return para
	}

	t.Fatalf("field paragraph containing %q not found", needle)
	return nil
}
