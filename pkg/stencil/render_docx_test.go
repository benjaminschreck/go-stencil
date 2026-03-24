package stencil

import (
	"testing"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

// Helper function to create a Body with Elements from paragraphs
func createBodyWithParagraphs(paragraphs []Paragraph) *Body {
	body := &Body{
		Elements: make([]BodyElement, len(paragraphs)),
	}
	for i, para := range paragraphs {
		p := para // Create a copy to get a new address
		body.Elements[i] = &p
	}
	return body
}

func TestRenderBodyWithControlStructures(t *testing.T) {
	tests := []struct {
		name     string
		body     *Body
		data     TemplateData
		wantText []string // Expected text content from paragraphs
	}{
		{
			name: "inline for loop",
			body: createBodyWithParagraphs([]Paragraph{
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "Items: {{for item in items}} - {{item.name}}{{end}}",
							},
						},
					},
				},
			}),
			data: TemplateData{
				"items": []map[string]interface{}{
					{"name": "Item 1"},
					{"name": "Item 2"},
					{"name": "Item 3"},
				},
			},
			wantText: []string{
				"Items:  - Item 1 - Item 2 - Item 3",
			},
		},
		{
			name: "multi-paragraph for loop",
			body: createBodyWithParagraphs([]Paragraph{
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "{{for item in items}}",
							},
						},
					},
				},
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "- {{item.name}}: {{item.status}}",
							},
						},
					},
				},
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "{{end}}",
							},
						},
					},
				},
			}),
			data: TemplateData{
				"items": []map[string]interface{}{
					{"name": "Task 1", "status": "Complete"},
					{"name": "Task 2", "status": "Pending"},
				},
			},
			wantText: []string{
				"- Task 1: Complete",
				"- Task 2: Pending",
			},
		},
		{
			name: "for loop with surrounding content",
			body: createBodyWithParagraphs([]Paragraph{
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "Tasks:",
							},
						},
					},
				},
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "{{for task in tasks}}",
							},
						},
					},
				},
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "• {{task}}",
							},
						},
					},
				},
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "{{end}}",
							},
						},
					},
				},
				{
					Runs: []Run{
						{
							Text: &Text{
								Content: "End of tasks.",
							},
						},
					},
				},
			}),
			data: TemplateData{
				"tasks": []string{"Write code", "Test code", "Deploy code"},
			},
			wantText: []string{
				"Tasks:",
				"• Write code",
				"• Test code",
				"• Deploy code",
				"End of tasks.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered, err := RenderBodyWithControlStructures(tt.body, tt.data, nil)
			if err != nil {
				t.Fatalf("RenderBodyWithControlStructures() error = %v", err)
			}

			// Extract text from rendered paragraphs
			var gotText []string
			for _, elem := range rendered.Elements {
				para, ok := elem.(*Paragraph)
				if !ok {
					continue
				}
				text := render.GetParagraphText(para)
				if text != "" {
					gotText = append(gotText, text)
				}
			}

			// Compare
			if len(gotText) != len(tt.wantText) {
				t.Errorf("Got %d paragraphs, want %d", len(gotText), len(tt.wantText))
				t.Errorf("Got: %v", gotText)
				t.Errorf("Want: %v", tt.wantText)
				return
			}

			for i, got := range gotText {
				if got != tt.wantText[i] {
					t.Errorf("Paragraph %d: got %q, want %q", i, got, tt.wantText[i])
				}
			}
		})
	}
}

func TestRenderBodyWithControlStructuresIfSplitAcrossRuns(t *testing.T) {
	body := &Body{
		Elements: []BodyElement{
			&Paragraph{
				Content: []ParagraphContent{
					&Run{Text: &Text{Content: "{{"}},
					&ProofErr{Type: "spellStart"},
					&Run{Text: &Text{Content: "if"}},
					&ProofErr{Type: "spellEnd"},
					&Run{Text: &Text{Content: " gegner."}},
					&Run{Text: &Text{Content: "name}}"}},
				},
			},
			&Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "Unfallgegner: {{gegner.vornameName}}"}},
				},
			},
			&Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "{{end}}"}},
				},
			},
		},
	}

	data := TemplateData{
		"gegner": map[string]interface{}{
			"name":        "Iris Schulz",
			"vornameName": "Iris Schulz",
		},
	}

	rendered, err := RenderBodyWithControlStructures(body, data, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() error = %v", err)
	}

	var gotText []string
	for _, elem := range rendered.Elements {
		para, ok := elem.(*Paragraph)
		if !ok {
			continue
		}
		text := render.GetParagraphText(para)
		if text != "" {
			gotText = append(gotText, text)
		}
	}

	if len(gotText) != 1 {
		t.Fatalf("expected 1 non-empty paragraph, got %d (%v)", len(gotText), gotText)
	}
	if gotText[0] != "Unfallgegner: Iris Schulz" {
		t.Fatalf("unexpected text: %q", gotText[0])
	}
}

func TestDetectControlStructure(t *testing.T) {
	tests := []struct {
		name        string
		paragraph   *Paragraph
		wantType    string
		wantContent string
	}{
		{
			name: "for loop",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "{{for item in items}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "for",
			wantContent: "item in items",
		},
		{
			name: "inline for loop",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "List: {{for x in list}} {{x}}{{end}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "inline-for",
			wantContent: "List: {{for x in list}} {{x}}{{end}}",
		},
		{
			name: "top-level if with nested inline for",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "{{if fahrerGegner}}{{for passivpartei in passivseite}}{{if passivpartei.nameAdresse == fahrerGegner.nameAdresse}}{{if passivpartei.anrede == „Frau“}}die Beklagtenpartei zu {{passivpartei.index}}) als Fahrerin {{else}}der Beklage zu {{passivpartei.index}}) als Fahrer {{end}}{{end}}{{end}}{{else}}der Fahrer {{end}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "",
			wantContent: "",
		},
		{
			name: "block if with nested inline for in opening paragraph",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "{{if show}}{{for _ in empty}}{{end}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "if",
			wantContent: "show",
		},
		{
			name: "block for with nested inline if in opening paragraph",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "{{for item in items}}{{if showHeader}}{{end}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "for",
			wantContent: "item in items",
		},
		{
			name: "block unless with nested inline if and for in opening paragraph",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "{{unless hide}}{{if showHeader}}{{end}}{{for _ in empty}}{{end}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "unless",
			wantContent: "hide",
		},
		{
			name: "block if with multiple nested inline controls in opening paragraph",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "{{if show}}{{unless suppress}}{{end}}{{for _ in empty}}{{if nested}}{{end}}{{end}}{{if extra}}{{end}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "if",
			wantContent: "show",
		},
		{
			name: "block for with multiple nested inline controls in opening paragraph",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "{{for item in items}}{{if showHeader}}{{end}}{{unless skipBody}}{{end}}{{for _ in empty}}{{end}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "for",
			wantContent: "item in items",
		},
		{
			name: "end marker",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "{{end}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "end",
			wantContent: "",
		},
		{
			name: "regular paragraph",
			paragraph: func() *Paragraph {
				p := createBodyWithParagraphs([]Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "This is regular text with {{variable}}",
								},
							},
						},
					},
				}).Elements[0].(*Paragraph)
				return p
			}(),
			wantType:    "",
			wantContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotContent := render.DetectControlStructure(tt.paragraph)
			if gotType != tt.wantType {
				t.Errorf("render.DetectControlStructure() type = %v, want %v", gotType, tt.wantType)
			}
			if gotContent != tt.wantContent {
				t.Errorf("render.DetectControlStructure() content = %v, want %v", gotContent, tt.wantContent)
			}
		})
	}
}

func TestRenderBodyWithControlStructuresNestedInlineForInsideIf(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{if fahrerGegner}}{{for passivpartei in passivseite}}{{if passivpartei.nameAdresse == fahrerGegner.nameAdresse}}{{if passivpartei.anrede == „Frau“}}die Beklagtenpartei zu {{passivpartei.index}}) als Fahrerin {{else}}der Beklage zu {{passivpartei.index}}) als Fahrer {{end}}{{end}}{{end}}{{else}}der Fahrer {{end}}",
					},
				},
			},
		},
	})

	data := TemplateData{
		"fahrerGegner": map[string]interface{}{
			"nameAdresse": "addr-1",
		},
		"passivseite": []interface{}{
			map[string]interface{}{
				"nameAdresse": "addr-1",
				"anrede":      "Frau",
				"index":       1,
			},
		},
	}

	rendered, err := RenderBodyWithControlStructures(body, data, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() error = %v", err)
	}

	if len(rendered.Elements) != 1 {
		t.Fatalf("expected 1 rendered element, got %d", len(rendered.Elements))
	}

	para, ok := rendered.Elements[0].(*Paragraph)
	if !ok {
		t.Fatalf("expected paragraph element, got %T", rendered.Elements[0])
	}

	got := render.GetParagraphText(para)
	want := "die Beklagtenpartei zu 1) als Fahrerin "
	if got != want {
		t.Fatalf("unexpected text: got %q want %q", got, want)
	}
}

func collectRenderedParagraphTexts(t *testing.T, body *Body) []string {
	t.Helper()

	var got []string
	for _, elem := range body.Elements {
		para, ok := elem.(*Paragraph)
		if !ok {
			t.Fatalf("expected paragraph element, got %T", elem)
		}
		got = append(got, render.GetParagraphText(para))
	}

	return got
}

func TestRenderBodyWithControlStructuresBlockIfOpeningParagraphContainsNestedInlineFor(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{if show}}{{for _ in empty}}{{end}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "Visible",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{end}}",
					},
				},
			},
		},
	})

	renderedHidden, err := RenderBodyWithControlStructures(body, TemplateData{
		"show":  false,
		"empty": []interface{}{},
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() hidden error = %v", err)
	}
	if len(renderedHidden.Elements) != 0 {
		t.Fatalf("expected no rendered elements when outer if is false, got %d", len(renderedHidden.Elements))
	}

	renderedVisible, err := RenderBodyWithControlStructures(body, TemplateData{
		"show":  true,
		"empty": []interface{}{},
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() visible error = %v", err)
	}
	if len(renderedVisible.Elements) != 1 {
		t.Fatalf("expected 1 rendered element when outer if is true, got %d", len(renderedVisible.Elements))
	}

	para, ok := renderedVisible.Elements[0].(*Paragraph)
	if !ok {
		t.Fatalf("expected paragraph element, got %T", renderedVisible.Elements[0])
	}
	if got := render.GetParagraphText(para); got != "Visible" {
		t.Fatalf("unexpected visible text: got %q want %q", got, "Visible")
	}
}

func TestRenderBodyWithControlStructuresBlockForOpeningParagraphContainsNestedInlineIf(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{for item in items}}{{if showHeader}}{{end}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "Item: {{item}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{end}}",
					},
				},
			},
		},
	})

	rendered, err := RenderBodyWithControlStructures(body, TemplateData{
		"items":      []string{"A", "B"},
		"showHeader": false,
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() error = %v", err)
	}

	got := collectRenderedParagraphTexts(t, rendered)
	want := []string{"Item: A", "Item: B"}
	if len(got) != len(want) {
		t.Fatalf("expected %d rendered elements, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected paragraph %d text: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestRenderBodyWithControlStructuresBlockUnlessOpeningParagraphContainsNestedInlineCombinations(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{unless hide}}{{if showHeader}}{{end}}{{for _ in empty}}{{end}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "Shown",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{else}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "Hidden",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{end}}",
					},
				},
			},
		},
	})

	renderedShown, err := RenderBodyWithControlStructures(body, TemplateData{
		"hide":       false,
		"showHeader": true,
		"empty":      []interface{}{},
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() shown error = %v", err)
	}
	if got := collectRenderedParagraphTexts(t, renderedShown); len(got) != 1 || got[0] != "Shown" {
		t.Fatalf("unexpected shown branch paragraphs: %v", got)
	}

	renderedHidden, err := RenderBodyWithControlStructures(body, TemplateData{
		"hide":       true,
		"showHeader": true,
		"empty":      []interface{}{},
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() hidden error = %v", err)
	}
	if got := collectRenderedParagraphTexts(t, renderedHidden); len(got) != 1 || got[0] != "Hidden" {
		t.Fatalf("unexpected hidden branch paragraphs: %v", got)
	}
}

func TestRenderBodyWithControlStructuresBlockIfOpeningParagraphContainsMultipleNestedInlineControls(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{if show}}{{unless suppress}}{{end}}{{for _ in empty}}{{if nested}}{{end}}{{end}}{{if extra}}{{end}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "Then branch",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{else}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "Else branch",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{end}}",
					},
				},
			},
		},
	})

	renderedThen, err := RenderBodyWithControlStructures(body, TemplateData{
		"show":     true,
		"suppress": false,
		"empty":    []interface{}{},
		"nested":   false,
		"extra":    false,
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() then error = %v", err)
	}
	if got := collectRenderedParagraphTexts(t, renderedThen); len(got) != 1 || got[0] != "Then branch" {
		t.Fatalf("unexpected then branch paragraphs: %v", got)
	}

	renderedElse, err := RenderBodyWithControlStructures(body, TemplateData{
		"show":     false,
		"suppress": false,
		"empty":    []interface{}{},
		"nested":   true,
		"extra":    true,
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() else error = %v", err)
	}
	if got := collectRenderedParagraphTexts(t, renderedElse); len(got) != 1 || got[0] != "Else branch" {
		t.Fatalf("unexpected else branch paragraphs: %v", got)
	}
}

func TestRenderBodyWithControlStructuresElseIfAndTailInSameParagraph(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{if first}}First",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{elseif second}}{{for i, item in items}}{{if i > 0}}, {{end}}{{item}}{{end}}{{else}}Fallback{{end}}{{if showTail}} tail{{end}}",
					},
				},
			},
		},
	})

	renderedElseIf, err := RenderBodyWithControlStructures(body, TemplateData{
		"first":    false,
		"second":   true,
		"items":    []string{"A", "B", "C"},
		"showTail": true,
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() elsif error = %v", err)
	}

	if got := collectRenderedParagraphTexts(t, renderedElseIf); len(got) != 1 || got[0] != "A, B, C tail" {
		t.Fatalf("unexpected elsif branch paragraphs: %v", got)
	}

	renderedElse, err := RenderBodyWithControlStructures(body, TemplateData{
		"first":    false,
		"second":   false,
		"items":    []string{"A", "B", "C"},
		"showTail": true,
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() else error = %v", err)
	}

	if got := collectRenderedParagraphTexts(t, renderedElse); len(got) != 1 || got[0] != "Fallback tail" {
		t.Fatalf("unexpected else branch paragraphs: %v", got)
	}
}

func TestRenderBodyWithControlStructuresBlockForOpeningParagraphContainsMultipleNestedInlineControls(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{for item in items}}{{if showHeader}}{{end}}{{unless skipBody}}{{end}}{{for _ in empty}}{{end}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{if item.active}}Item: {{item.name}}{{else}}Inactive: {{item.name}}{{end}}",
					},
				},
			},
		},
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: "{{end}}",
					},
				},
			},
		},
	})

	rendered, err := RenderBodyWithControlStructures(body, TemplateData{
		"items": []interface{}{
			map[string]interface{}{"name": "A", "active": true},
			map[string]interface{}{"name": "B", "active": false},
			map[string]interface{}{"name": "C", "active": true},
		},
		"showHeader": false,
		"skipBody":   false,
		"empty":      []interface{}{},
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() error = %v", err)
	}

	got := collectRenderedParagraphTexts(t, rendered)
	want := []string{"Item: A", "Inactive: B", "Item: C"}
	if len(got) != len(want) {
		t.Fatalf("expected %d rendered elements, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected paragraph %d text: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestRenderBodyWithControlStructuresAllControlsNestedInSameParagraph(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: `{{for i, item in items}}{{if i > 0}} | {{end}}{{if item.kind == "vip"}}VIP {{item.name}}{{elseif item.kind == "std"}}{{unless item.skip}}STD {{item.name}}{{else}}SKIP {{item.name}}{{end}}{{else}}OTHER {{item.name}}{{end}}{{end}}`,
					},
				},
			},
		},
	})

	rendered, err := RenderBodyWithControlStructures(body, TemplateData{
		"items": []interface{}{
			map[string]interface{}{"name": "Alice", "kind": "vip", "skip": false},
			map[string]interface{}{"name": "Bob", "kind": "std", "skip": false},
			map[string]interface{}{"name": "Cara", "kind": "std", "skip": true},
			map[string]interface{}{"name": "Dora", "kind": "other", "skip": false},
		},
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() error = %v", err)
	}

	if got := collectRenderedParagraphTexts(t, rendered); len(got) != 1 || got[0] != "VIP Alice | STD Bob | SKIP Cara | OTHER Dora" {
		t.Fatalf("unexpected nested same-paragraph output: %v", got)
	}
}

func TestRenderBodyWithControlStructuresForNestedInIfSameParagraph(t *testing.T) {
	body := createBodyWithParagraphs([]Paragraph{
		{
			Runs: []Run{
				{
					Text: &Text{
						Content: `{{if show}}{{for i, item in items}}{{if i > 0}} | {{end}}{{if i == 0}}FIRST {{item.name}}{{elseif item.active}}ON {{item.name}}{{else}}OFF {{item.name}}{{end}}{{end}}{{else}}Hidden{{end}}`,
					},
				},
			},
		},
	})

	renderedShown, err := RenderBodyWithControlStructures(body, TemplateData{
		"show": true,
		"items": []interface{}{
			map[string]interface{}{"name": "A", "active": true},
			map[string]interface{}{"name": "B", "active": false},
			map[string]interface{}{"name": "C", "active": true},
		},
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() shown error = %v", err)
	}

	if got := collectRenderedParagraphTexts(t, renderedShown); len(got) != 1 || got[0] != "FIRST A | OFF B | ON C" {
		t.Fatalf("unexpected shown same-paragraph if/for/if output: %v", got)
	}

	renderedHidden, err := RenderBodyWithControlStructures(body, TemplateData{
		"show":  false,
		"items": []interface{}{},
	}, nil)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() hidden error = %v", err)
	}

	if got := collectRenderedParagraphTexts(t, renderedHidden); len(got) != 1 || got[0] != "Hidden" {
		t.Fatalf("unexpected hidden same-paragraph if/for/if output: %v", got)
	}
}
