package stencil

import (
	"testing"
)

func TestRenderBodyWithControlStructures(t *testing.T) {
	tests := []struct {
		name     string
		body     *Body
		data     TemplateData
		wantText []string // Expected text content from paragraphs
	}{
		{
			name: "inline for loop",
			body: &Body{
				Paragraphs: []Paragraph{
					{
						Runs: []Run{
							{
								Text: &Text{
									Content: "Items: {{for item in items}} - {{item.name}}{{end}}",
								},
							},
						},
					},
				},
			},
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
			body: &Body{
				Paragraphs: []Paragraph{
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
				},
			},
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
			body: &Body{
				Paragraphs: []Paragraph{
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
				},
			},
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
			for _, para := range rendered.Paragraphs {
				text := getParagraphText(&para)
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

			for i, want := range tt.wantText {
				if i < len(gotText) && gotText[i] != want {
					t.Errorf("Paragraph %d: got %q, want %q", i, gotText[i], want)
				}
			}
		})
	}
}

func TestDetectControlStructure(t *testing.T) {
	tests := []struct {
		name         string
		para         *Paragraph
		wantType     string
		wantContent  string
	}{
		{
			name: "for loop",
			para: &Paragraph{
				Runs: []Run{
					{
						Text: &Text{
							Content: "{{for item in items}}",
						},
					},
				},
			},
			wantType:    "for",
			wantContent: "item in items",
		},
		{
			name: "inline for loop",
			para: &Paragraph{
				Runs: []Run{
					{
						Text: &Text{
							Content: "List: {{for x in list}} {{x}}{{end}}",
						},
					},
				},
			},
			wantType:    "inline-for",
			wantContent: "List: {{for x in list}} {{x}}{{end}}",
		},
		{
			name: "end marker",
			para: &Paragraph{
				Runs: []Run{
					{
						Text: &Text{
							Content: "{{end}}",
						},
					},
				},
			},
			wantType:    "end",
			wantContent: "",
		},
		{
			name: "regular paragraph",
			para: &Paragraph{
				Runs: []Run{
					{
						Text: &Text{
							Content: "This is regular text with {{variable}}",
						},
					},
				},
			},
			wantType:    "",
			wantContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotContent := detectControlStructure(tt.para)
			if gotType != tt.wantType {
				t.Errorf("detectControlStructure() type = %v, want %v", gotType, tt.wantType)
			}
			if gotContent != tt.wantContent {
				t.Errorf("detectControlStructure() content = %v, want %v", gotContent, tt.wantContent)
			}
		})
	}
}