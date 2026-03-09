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

func TestRenderParagraphWithContext_ProofErrInlineIfPreservesBoldBranchFormatting(t *testing.T) {
	bold := &RunProperties{
		Bold:   &Empty{},
		BoldCs: &Empty{},
	}

	para := &Paragraph{
		Content: []ParagraphContent{
			&Run{Text: &Text{Content: "{{"}},
			&ProofErr{Type: "spellStart"},
			&Run{Text: &Text{Content: "if"}},
			&ProofErr{Type: "spellEnd"},
			&Run{Text: &Text{Content: " showAdvance}}"}},
			&Run{
				Properties: bold,
				Text:       &Text{Content: "Schmerzensgeldvorschuss:"},
			},
			&Run{Text: &Text{Content: "{{else}}"}},
			&Run{
				Properties: bold,
				Text:       &Text{Content: "Schmerzensgeld:"},
			},
			&Run{Text: &Text{Content: "{{end}}"}},
		},
	}

	rendered, err := RenderParagraphWithContext(para, TemplateData{
		"showAdvance": true,
	}, &renderContext{})
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error: %v", err)
	}

	if got := rendered.GetText(); got != "Schmerzensgeldvorschuss:" {
		t.Fatalf("rendered text = %q, want %q", got, "Schmerzensgeldvorschuss:")
	}
	if len(rendered.Runs) != 1 {
		t.Fatalf("expected 1 rendered run, got %d", len(rendered.Runs))
	}
	if rendered.Runs[0].Properties == nil || rendered.Runs[0].Properties.Bold == nil {
		t.Fatalf("expected bold formatting on rendered branch, got %+v", rendered.Runs[0].Properties)
	}

	renderedElse, err := RenderParagraphWithContext(para, TemplateData{
		"showAdvance": false,
	}, &renderContext{})
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error for else branch: %v", err)
	}

	if got := renderedElse.GetText(); got != "Schmerzensgeld:" {
		t.Fatalf("rendered else text = %q, want %q", got, "Schmerzensgeld:")
	}
	if len(renderedElse.Runs) != 1 {
		t.Fatalf("expected 1 rendered else run, got %d", len(renderedElse.Runs))
	}
	if renderedElse.Runs[0].Properties == nil || renderedElse.Runs[0].Properties.Bold == nil {
		t.Fatalf("expected bold formatting on rendered else branch, got %+v", renderedElse.Runs[0].Properties)
	}
}
