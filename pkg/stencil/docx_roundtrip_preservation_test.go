package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestMarshalDocumentWithNamespaces_PreservesNumberingAndFieldsFromExampleDocx(t *testing.T) {
	docxPath := filepath.Join("..", "..", "examples", "advanced", "Beispiel Nummerierung.docx")

	docXML := mustReadDocxPart(t, docxPath, "word/document.xml")
	doc, err := ParseDocument(bytes.NewReader(docXML))
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	renderedXML, err := marshalDocumentWithNamespaces(doc)
	if err != nil {
		t.Fatalf("marshalDocumentWithNamespaces failed: %v", err)
	}

	assertPatternCountMatch(t, docXML, renderedXML, "numPr", regexp.MustCompile(`<w:numPr[\s\S]*?</w:numPr>`))
	assertPatternCountMatch(t, docXML, renderedXML, "fldChar", regexp.MustCompile(`<w:fldChar[^>]*/?>`))
	assertPatternCountMatch(t, docXML, renderedXML, "instrText", regexp.MustCompile(`<w:instrText[^>]*>[\s\S]*?</w:instrText>`))
}

func TestRenderDocumentWithContext_PreservesUntouchedExampleFieldParagraph(t *testing.T) {
	docxPath := filepath.Join("..", "..", "examples", "advanced", "Beispiel Nummerierung.docx")

	docXML := mustReadDocxPart(t, docxPath, "word/document.xml")
	doc, err := ParseDocument(bytes.NewReader(docXML))
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	source := findExampleFieldParagraph(t, doc)
	renderedDoc, err := RenderDocumentWithContext(&Document{
		Body: &Body{
			Elements: []BodyElement{cloneParagraph(source)},
		},
	}, TemplateData{}, &renderContext{})
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

	if got, want := countProofErr(source), countProofErr(renderedPara); got != want {
		t.Fatalf("proofErr count changed: before=%d after=%d", got, want)
	}
	if got, want := countRawRunElementMatches(source, "<w:fldChar"), countRawRunElementMatches(renderedPara, "<w:fldChar"); got != want {
		t.Fatalf("fldChar count changed: before=%d after=%d", got, want)
	}
	if got, want := countRawRunElementMatches(source, "<w:instrText"), countRawRunElementMatches(renderedPara, "<w:instrText"); got != want {
		t.Fatalf("instrText count changed: before=%d after=%d", got, want)
	}
	if got, want := countParagraphPropertyRawMatches(source, "<w:numPr"), countParagraphPropertyRawMatches(renderedPara, "<w:numPr"); got != want {
		t.Fatalf("numPr count changed: before=%d after=%d", got, want)
	}
}

func mustReadDocxPart(t *testing.T, docxPath, partName string) []byte {
	t.Helper()

	content, err := os.ReadFile(docxPath)
	if err != nil {
		t.Fatalf("failed to read docx %s: %v", docxPath, err)
	}

	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("failed to open docx zip %s: %v", docxPath, err)
	}

	for _, file := range reader.File {
		if file.Name != partName {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			t.Fatalf("failed to open %s in %s: %v", partName, docxPath, err)
		}
		defer func() {
			_ = rc.Close()
		}()

		part, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("failed to read %s from %s: %v", partName, docxPath, err)
		}
		return part
	}

	t.Fatalf("part %s not found in %s", partName, docxPath)
	return nil
}

func assertPatternCountMatch(t *testing.T, before, after []byte, label string, pattern *regexp.Regexp) {
	t.Helper()

	beforeCount := len(pattern.FindAll(before, -1))
	afterCount := len(pattern.FindAll(after, -1))

	if beforeCount != afterCount {
		t.Fatalf("%s count mismatch: before=%d after=%d", label, beforeCount, afterCount)
	}
}

func findExampleFieldParagraph(t *testing.T, doc *Document) *Paragraph {
	t.Helper()

	if doc == nil || doc.Body == nil {
		t.Fatal("document body is nil")
	}

	for _, elem := range doc.Body.Elements {
		para, ok := elem.(*Paragraph)
		if !ok {
			continue
		}
		if !strings.Contains(para.GetText(), "Bestätigung der") {
			continue
		}
		if countRawRunElementMatches(para, "<w:fldChar") == 0 {
			continue
		}
		return para
	}

	t.Fatal("example field paragraph not found")
	return nil
}

func countProofErr(para *Paragraph) int {
	if para == nil {
		return 0
	}

	count := 0
	for _, content := range para.Content {
		if _, ok := content.(*ProofErr); ok {
			count++
		}
	}
	return count
}

func countRawRunElementMatches(para *Paragraph, needle string) int {
	if para == nil {
		return 0
	}

	count := 0
	for _, run := range para.Runs {
		for _, raw := range run.RawXML {
			count += strings.Count(string(raw.Content), needle)
		}
	}
	return count
}

func countParagraphPropertyRawMatches(para *Paragraph, needle string) int {
	if para == nil || para.Properties == nil {
		return 0
	}

	count := 0
	for _, raw := range para.Properties.RawXML {
		count += strings.Count(string(raw.Content), needle)
	}
	return count
}
