package stencil

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

func TestMergeConsecutiveRuns_DoesNotMergeDifferentRunAttrs(t *testing.T) {
	para := &Paragraph{
		Runs: []Run{
			{
				Properties: &RunProperties{},
				Attrs:      []xml.Attr{{Name: xml.Name{Local: "w:rsidRPr"}, Value: "AAA"}},
				Text:       &Text{Content: "A"},
			},
			{
				Properties: &RunProperties{},
				Attrs:      []xml.Attr{{Name: xml.Name{Local: "w:rsidRPr"}, Value: "BBB"}},
				Text:       &Text{Content: "B"},
			},
		},
	}

	render.MergeConsecutiveRuns(para)

	if len(para.Runs) != 2 {
		t.Fatalf("expected 2 runs after merge, got %d", len(para.Runs))
	}
	if para.Runs[0].Text == nil || para.Runs[0].Text.Content != "A" {
		t.Fatalf("unexpected first run content: %+v", para.Runs[0].Text)
	}
	if para.Runs[1].Text == nil || para.Runs[1].Text.Content != "B" {
		t.Fatalf("unexpected second run content: %+v", para.Runs[1].Text)
	}
}

func TestMergeConsecutiveRunsWithContent_PreservesProofErr(t *testing.T) {
	para := &Paragraph{
		Content: []ParagraphContent{
			&Run{Text: &Text{Content: "{{"}},
			&ProofErr{Type: "spellStart"},
			&Run{Text: &Text{Content: "include"}},
			&ProofErr{Type: "spellEnd"},
			&Run{Text: &Text{Content: "}}"}},
		},
	}

	render.MergeConsecutiveRuns(para)

	if len(para.Content) != 5 {
		t.Fatalf("expected 5 content items, got %d", len(para.Content))
	}
	if _, ok := para.Content[1].(*ProofErr); !ok {
		t.Fatalf("expected proofErr at content index 1, got %T", para.Content[1])
	}
	if _, ok := para.Content[3].(*ProofErr); !ok {
		t.Fatalf("expected proofErr at content index 3, got %T", para.Content[3])
	}
}

func TestCloneParagraph_PreservesAndCopiesAttrs(t *testing.T) {
	original := &Paragraph{
		Attrs: []xml.Attr{
			{Name: xml.Name{Local: "w14:paraId"}, Value: "AAAA1111"},
		},
		Runs: []Run{
			{
				Attrs: []xml.Attr{
					{Name: xml.Name{Local: "w:rsidRPr"}, Value: "00ABCDEF"},
				},
				Text: &Text{Content: "text"},
			},
		},
	}

	cloned := cloneParagraph(original)
	if cloned == nil {
		t.Fatal("expected cloned paragraph, got nil")
	}
	if len(cloned.Attrs) != 1 || cloned.Attrs[0].Value != "AAAA1111" {
		t.Fatalf("expected paragraph attrs to be cloned, got %+v", cloned.Attrs)
	}
	if len(cloned.Runs) != 1 || len(cloned.Runs[0].Attrs) != 1 || cloned.Runs[0].Attrs[0].Value != "00ABCDEF" {
		t.Fatalf("expected run attrs to be cloned, got %+v", cloned.Runs)
	}

	original.Attrs[0].Value = "CHANGED"
	original.Runs[0].Attrs[0].Value = "CHANGED_RUN"

	if cloned.Attrs[0].Value != "AAAA1111" {
		t.Fatalf("clone paragraph attrs changed unexpectedly: %+v", cloned.Attrs)
	}
	if cloned.Runs[0].Attrs[0].Value != "00ABCDEF" {
		t.Fatalf("clone run attrs changed unexpectedly: %+v", cloned.Runs[0].Attrs)
	}
}

func TestMarshalDocumentWithNamespaces_WritesProofErr(t *testing.T) {
	doc := &Document{
		Body: &Body{
			Elements: []BodyElement{
				&Paragraph{
					Content: []ParagraphContent{
						&Run{Text: &Text{Content: "{{"}},
						&ProofErr{Type: "spellStart"},
						&Run{Text: &Text{Content: "include"}},
						&ProofErr{Type: "spellEnd"},
						&Run{Text: &Text{Content: "}}"}},
					},
				},
			},
		},
	}

	out, err := marshalDocumentWithNamespaces(doc)
	if err != nil {
		t.Fatalf("marshalDocumentWithNamespaces failed: %v", err)
	}
	xmlOut := string(out)
	if !strings.Contains(xmlOut, "<w:proofErr") {
		t.Fatalf("expected output to contain <w:proofErr, got: %s", xmlOut)
	}
	if !strings.Contains(xmlOut, `w:type="spellStart"`) || !strings.Contains(xmlOut, `w:type="spellEnd"`) {
		t.Fatalf("expected output to contain proofErr types, got: %s", xmlOut)
	}
}
