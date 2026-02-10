package stencil

import "testing"

func TestRenderParagraphWithContext_ProofErrSplitVariable(t *testing.T) {
	para := &Paragraph{
		Content: []ParagraphContent{
			&Run{Text: &Text{Content: "{{"}},
			&ProofErr{Type: "spellStart"},
			&Run{Text: &Text{Content: "name"}},
			&ProofErr{Type: "spellEnd"},
			&Run{Text: &Text{Content: "}}"}},
		},
	}

	rendered, err := RenderParagraphWithContext(para, TemplateData{
		"name": "Alice",
	}, &renderContext{})
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error: %v", err)
	}

	if got := rendered.GetText(); got != "Alice" {
		t.Fatalf("rendered text = %q, want %q", got, "Alice")
	}
}
