package stencil

import (
	"strings"
	"testing"
)

func TestParseDocument(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		wantErr bool
		check   func(t *testing.T, doc *Document)
	}{
		{
			name: "parse simple paragraph",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>Hello World</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if doc == nil {
					t.Fatal("expected non-nil document")
				}
				if doc.Body == nil {
					t.Fatal("expected non-nil body")
				}
				if len(doc.Body.Paragraphs) != 1 {
					t.Errorf("expected 1 paragraph, got %d", len(doc.Body.Paragraphs))
				}
				if len(doc.Body.Paragraphs[0].Runs) != 1 {
					t.Errorf("expected 1 run, got %d", len(doc.Body.Paragraphs[0].Runs))
				}
				if doc.Body.Paragraphs[0].Runs[0].Text.Content != "Hello World" {
					t.Errorf("expected 'Hello World', got '%s'", doc.Body.Paragraphs[0].Runs[0].Text.Content)
				}
			},
		},
		{
			name: "parse paragraph with multiple runs",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>Hello </w:t>
			</w:r>
			<w:r>
				<w:t>{{name}}</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Paragraphs) != 1 {
					t.Errorf("expected 1 paragraph, got %d", len(doc.Body.Paragraphs))
				}
				if len(doc.Body.Paragraphs[0].Runs) != 2 {
					t.Errorf("expected 2 runs, got %d", len(doc.Body.Paragraphs[0].Runs))
				}
				if doc.Body.Paragraphs[0].Runs[0].Text.Content != "Hello " {
					t.Errorf("expected 'Hello ', got '%s'", doc.Body.Paragraphs[0].Runs[0].Text.Content)
				}
				if doc.Body.Paragraphs[0].Runs[1].Text.Content != "{{name}}" {
					t.Errorf("expected '{{name}}', got '%s'", doc.Body.Paragraphs[0].Runs[1].Text.Content)
				}
			},
		},
		{
			name: "parse multiple paragraphs",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>First paragraph</w:t>
			</w:r>
		</w:p>
		<w:p>
			<w:r>
				<w:t>Second paragraph</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Paragraphs) != 2 {
					t.Errorf("expected 2 paragraphs, got %d", len(doc.Body.Paragraphs))
				}
			},
		},
		{
			name: "parse text with xml:space preserve",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t xml:space="preserve">  Spaced  </w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				text := doc.Body.Paragraphs[0].Runs[0].Text
				if text.Space != "preserve" {
					t.Errorf("expected space='preserve', got '%s'", text.Space)
				}
				if text.Content != "  Spaced  " {
					t.Errorf("expected '  Spaced  ', got '%s'", text.Content)
				}
			},
		},
		{
			name:    "parse invalid XML",
			xml:     `<invalid>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.xml)
			doc, err := ParseDocument(reader)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDocument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(t, doc)
			}
		})
	}
}

func TestRun_GetText(t *testing.T) {
	tests := []struct {
		name string
		run  Run
		want string
	}{
		{
			name: "simple text",
			run: Run{
				Text: &Text{Content: "Hello"},
			},
			want: "Hello",
		},
		{
			name: "empty run",
			run:  Run{},
			want: "",
		},
		{
			name: "nil text",
			run: Run{
				Text: nil,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.run.GetText(); got != tt.want {
				t.Errorf("Run.GetText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParagraph_GetText(t *testing.T) {
	tests := []struct {
		name string
		para Paragraph
		want string
	}{
		{
			name: "single run",
			para: Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "Hello World"}},
				},
			},
			want: "Hello World",
		},
		{
			name: "multiple runs",
			para: Paragraph{
				Runs: []Run{
					{Text: &Text{Content: "Hello "}},
					{Text: &Text{Content: "{{name}}"}},
					{Text: &Text{Content: "!"}},
				},
			},
			want: "Hello {{name}}!",
		},
		{
			name: "empty paragraph",
			para: Paragraph{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.para.GetText(); got != tt.want {
				t.Errorf("Paragraph.GetText() = %v, want %v", got, tt.want)
			}
		})
	}
}