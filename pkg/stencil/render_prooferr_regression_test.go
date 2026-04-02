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

func TestRenderParagraphWithContext_InlineIfWithTextFragmentPreservesBranchFormatting(t *testing.T) {
	bold := &RunProperties{
		Bold:   &Empty{},
		BoldCs: &Empty{},
	}

	para := &Paragraph{
		Runs: []Run{
			{Text: &Text{Content: "{{if show}}"}},
			{
				Properties: bold,
				Text:       &Text{Content: "{{include \"frag\"}}"},
			},
			{Text: &Text{Content: "{{else}}"}},
			{
				Properties: bold,
				Text:       &Text{Content: "Fallback"},
			},
			{Text: &Text{Content: "{{end}}"}},
		},
	}

	ctx := &renderContext{
		fragments: map[string]*fragment{
			"frag": {
				content: "Schmerzensgeld {{name}}",
			},
		},
		fragmentStack:  make([]string, 0),
		ooxmlFragments: make(map[string]interface{}),
	}

	rendered, err := RenderParagraphWithContext(para, TemplateData{
		"show": true,
		"name": "Alice",
	}, ctx)
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error: %v", err)
	}

	if got := rendered.GetText(); got != "Schmerzensgeld Alice" {
		t.Fatalf("rendered text = %q, want %q", got, "Schmerzensgeld Alice")
	}
	if len(rendered.Runs) != 1 {
		t.Fatalf("expected 1 rendered run, got %d", len(rendered.Runs))
	}
	if rendered.Runs[0].Properties == nil || rendered.Runs[0].Properties.Bold == nil {
		t.Fatalf("expected bold formatting on rendered fragment branch, got %+v", rendered.Runs[0].Properties)
	}

	renderedElse, err := RenderParagraphWithContext(para, TemplateData{
		"show": false,
		"name": "Alice",
	}, ctx)
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error for else branch: %v", err)
	}

	if got := renderedElse.GetText(); got != "Fallback" {
		t.Fatalf("rendered else text = %q, want %q", got, "Fallback")
	}
	if len(renderedElse.Runs) != 1 {
		t.Fatalf("expected 1 rendered else run, got %d", len(renderedElse.Runs))
	}
	if renderedElse.Runs[0].Properties == nil || renderedElse.Runs[0].Properties.Bold == nil {
		t.Fatalf("expected bold formatting on rendered else branch, got %+v", renderedElse.Runs[0].Properties)
	}
}

func TestRenderParagraphWithContext_InlineIfWithBraceSplitEndPreservesBranchFormatting(t *testing.T) {
	bold := &RunProperties{
		Bold:   &Empty{},
		BoldCs: &Empty{},
	}

	para := &Paragraph{
		Runs: []Run{
			{Text: &Text{Content: "{{if show}}"}},
			{
				Properties: bold,
				Text:       &Text{Content: "Visible"},
			},
			{
				Properties: &RunProperties{Italic: &Empty{}},
				Text:       &Text{Content: "{"},
			},
			{
				Properties: &RunProperties{Underline: &UnderlineStyle{Val: "single"}},
				Text:       &Text{Content: "{end}}"},
			},
		},
	}

	rendered, err := RenderParagraphWithContext(para, TemplateData{
		"show": true,
	}, &renderContext{})
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error: %v", err)
	}

	if got := rendered.GetText(); got != "Visible" {
		t.Fatalf("rendered text = %q, want %q", got, "Visible")
	}
	if len(rendered.Runs) != 1 {
		t.Fatalf("expected 1 rendered run, got %d", len(rendered.Runs))
	}
	if rendered.Runs[0].Properties == nil || rendered.Runs[0].Properties.Bold == nil {
		t.Fatalf("expected bold formatting on rendered branch, got %+v", rendered.Runs[0].Properties)
	}

	renderedHidden, err := RenderParagraphWithContext(para, TemplateData{
		"show": false,
	}, &renderContext{})
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error for hidden branch: %v", err)
	}

	if got := renderedHidden.GetText(); got != "" {
		t.Fatalf("rendered hidden text = %q, want empty", got)
	}
}

func TestRenderParagraphWithContext_ProofErrInlineForPreservesBoldBodyFormatting(t *testing.T) {
	bold := &RunProperties{
		Bold:   &Empty{},
		BoldCs: &Empty{},
	}

	para := &Paragraph{
		Runs: []Run{
			{Text: &Text{Content: "{{for party in aktivseite}}"}},
			{
				Properties: bold,
				Text:       &Text{Content: "{{party.vornameName}}, {{party.strasse}}"},
			},
			{Text: &Text{Content: "{{end}}"}},
		},
	}

	rendered, err := RenderParagraphWithContext(para, TemplateData{
		"aktivseite": []any{
			map[string]any{
				"vornameName": "Bolt",
				"strasse":     "Hauptstrasse 1",
			},
		},
	}, &renderContext{})
	if err != nil {
		t.Fatalf("RenderParagraphWithContext returned error: %v", err)
	}

	if got := rendered.GetText(); got != "Bolt, Hauptstrasse 1" {
		t.Fatalf("rendered text = %q, want %q", got, "Bolt, Hauptstrasse 1")
	}
	if len(rendered.Runs) == 0 {
		t.Fatal("expected rendered runs, got none")
	}

	for i, run := range rendered.Runs {
		if run.Text == nil || run.Text.Content == "" {
			continue
		}
		if run.Properties == nil || run.Properties.Bold == nil {
			t.Fatalf("expected bold formatting on rendered run %d, got %+v", i, run.Properties)
		}
	}
}
