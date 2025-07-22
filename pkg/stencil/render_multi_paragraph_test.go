package stencil

import (
	"testing"
)

func TestMultiParagraphIfTimeout(t *testing.T) {
	// Minimal test to reproduce the timeout issue
	body := &Body{
		Elements: []BodyElement{
			&Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "{{if true}}"}},
				},
			},
			&Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "Content"}},
				},
			},
			&Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "{{end}}"}},
				},
			},
		},
	}

	data := TemplateData{}


	// This should not timeout
	_, err := RenderBody(body, data)
	if err != nil {
		t.Fatalf("Failed to render body: %v", err)
	}
}

func TestMultiParagraphIfWithPrefix(t *testing.T) {
	// Test with prefix text before if statement
	body := &Body{
		Elements: []BodyElement{
			&Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "Some text before {{if true}}"}},
				},
			},
			&Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "Content"}},
				},
			},
			&Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "{{end}}"}},
				},
			},
		},
	}

	data := TemplateData{}

	// This should not timeout
	_, err := RenderBody(body, data)
	if err != nil {
		t.Fatalf("Failed to render body: %v", err)
	}
}