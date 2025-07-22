package stencil

import (
	"testing"
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
				text := getParagraphText(para)
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
			gotType, gotContent := detectControlStructure(tt.paragraph)
			if gotType != tt.wantType {
				t.Errorf("detectControlStructure() type = %v, want %v", gotType, tt.wantType)
			}
			if gotContent != tt.wantContent {
				t.Errorf("detectControlStructure() content = %v, want %v", gotContent, tt.wantContent)
			}
		})
	}
}