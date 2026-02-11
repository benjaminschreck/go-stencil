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
				if len(doc.Body.Elements) != 1 {
					t.Errorf("expected 1 element, got %d", len(doc.Body.Elements))
					return
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Errorf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
					return
				}
				if len(para.Runs) != 1 {
					t.Errorf("expected 1 run, got %d", len(para.Runs))
				}
				if para.Runs[0].Text.Content != "Hello World" {
					t.Errorf("expected 'Hello World', got '%s'", para.Runs[0].Text.Content)
				}
			},
		},
		{
			name: "parse strict wordprocessing namespace paragraph",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://purl.oclc.org/ooxml/wordprocessingml/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>Hello Strict</w:t>
			</w:r>
			<w:proofErr w:type="spellStart"/>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if doc == nil || doc.Body == nil || len(doc.Body.Elements) != 1 {
					t.Fatalf("expected one paragraph in strict document")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if len(para.Runs) != 1 {
					t.Fatalf("expected 1 run, got %d", len(para.Runs))
				}
				if para.Runs[0].Text == nil || para.Runs[0].Text.Content != "Hello Strict" {
					t.Fatalf("unexpected strict run text: %+v", para.Runs[0].Text)
				}
				if para.GetText() != "Hello Strict" {
					t.Fatalf("unexpected strict paragraph text: %q", para.GetText())
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
				if len(doc.Body.Elements) != 1 {
					t.Errorf("expected 1 element, got %d", len(doc.Body.Elements))
					return
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Errorf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
					return
				}
				if len(para.Runs) != 2 {
					t.Errorf("expected 2 runs, got %d", len(para.Runs))
				}
				if para.Runs[0].Text.Content != "Hello " {
					t.Errorf("expected 'Hello ', got '%s'", para.Runs[0].Text.Content)
				}
				if para.Runs[1].Text.Content != "{{name}}" {
					t.Errorf("expected '{{name}}', got '%s'", para.Runs[1].Text.Content)
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
				if len(doc.Body.Elements) != 2 {
					t.Errorf("expected 2 elements, got %d", len(doc.Body.Elements))
				}
				// Verify both elements are paragraphs
				for i, elem := range doc.Body.Elements {
					if _, ok := elem.(*Paragraph); !ok {
						t.Errorf("expected element %d to be *Paragraph, got %T", i, elem)
					}
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
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if len(para.Runs) == 0 {
					t.Fatal("expected at least one run")
				}
				text := para.Runs[0].Text
				if text.Space != "preserve" {
					t.Errorf("expected space='preserve', got '%s'", text.Space)
				}
				if text.Content != "  Spaced  " {
					t.Errorf("expected '  Spaced  ', got '%s'", text.Content)
				}
			},
		},
		{
			name: "parse run nested in unknown paragraph wrapper",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:smartTag w:uri="urn:test" w:element="token">
				<w:r>
					<w:t>Hello from wrapper</w:t>
				</w:r>
			</w:smartTag>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if len(para.Runs) != 1 {
					t.Fatalf("expected 1 run from wrapper content, got %d", len(para.Runs))
				}
				if para.Runs[0].Text == nil || para.Runs[0].Text.Content != "Hello from wrapper" {
					t.Fatalf("unexpected run text: %+v", para.Runs[0].Text)
				}
			},
		},
		{
			name: "do not parse non-wordprocessing runs inside wrappers",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:m="http://schemas.openxmlformats.org/officeDocument/2006/math">
	<w:body>
		<w:p>
			<m:oMath>
				<m:r>
					<m:t>x</m:t>
				</m:r>
			</m:oMath>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if len(para.Runs) != 0 {
					t.Fatalf("expected no paragraph runs from math wrapper, got %d", len(para.Runs))
				}
				if text := para.GetText(); text != "" {
					t.Fatalf("expected empty paragraph text, got %q", text)
				}
			},
		},
		{
			name: "skip deleted revision wrapper content",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:del w:id="1" w:author="tester">
				<w:r><w:t>deleted</w:t></w:r>
			</w:del>
			<w:r><w:t>kept</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "kept" {
					t.Fatalf("expected only kept text, got %q", text)
				}
			},
		},
		{
			name: "skip moveFrom revision wrapper content",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:moveFrom w:id="1" w:author="tester">
				<w:r><w:t>moved-from</w:t></w:r>
			</w:moveFrom>
			<w:r><w:t>kept</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "kept" {
					t.Fatalf("expected only kept text, got %q", text)
				}
			},
		},
		{
			name: "skip deleted revision wrapper content in strict namespace",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://purl.oclc.org/ooxml/wordprocessingml/main">
	<w:body>
		<w:p>
			<w:del w:id="1" w:author="tester">
				<w:r><w:t>deleted</w:t></w:r>
			</w:del>
			<w:r><w:t>kept</w:t></w:r>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "kept" {
					t.Fatalf("expected only kept text, got %q", text)
				}
			},
		},
		{
			name: "alternate content selects one branch only",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">
	<w:body>
		<w:p>
			<mc:AlternateContent>
				<mc:Choice Requires="w">
					<w:r><w:t>choice</w:t></w:r>
				</mc:Choice>
				<mc:Fallback>
					<w:r><w:t>fallback</w:t></w:r>
				</mc:Fallback>
			</mc:AlternateContent>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "choice" {
					t.Fatalf("expected choice branch only, got %q", text)
				}
			},
		},
		{
			name: "alternate content uses fallback when choice requires unsupported prefix",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">
	<w:body>
		<w:p>
			<mc:AlternateContent>
				<mc:Choice Requires="x16">
					<w:r><w:t>choice</w:t></w:r>
				</mc:Choice>
				<mc:Fallback>
					<w:r><w:t>fallback</w:t></w:r>
				</mc:Fallback>
			</mc:AlternateContent>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "fallback" {
					t.Fatalf("expected fallback branch, got %q", text)
				}
			},
		},
		{
			name: "alternate content keeps choice when no fallback exists",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">
	<w:body>
		<w:p>
			<mc:AlternateContent>
				<mc:Choice Requires="x16">
					<w:r><w:t>choice</w:t></w:r>
				</mc:Choice>
			</mc:AlternateContent>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "choice" {
					t.Fatalf("expected choice branch to avoid data loss, got %q", text)
				}
			},
		},
		{
			name: "alternate content accepts common word versioned prefix",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">
	<w:body>
		<w:p>
			<mc:AlternateContent>
				<mc:Choice Requires="w14">
					<w:r><w:t>choice</w:t></w:r>
				</mc:Choice>
				<mc:Fallback>
					<w:r><w:t>fallback</w:t></w:r>
				</mc:Fallback>
			</mc:AlternateContent>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "choice" {
					t.Fatalf("expected w14 choice branch, got %q", text)
				}
			},
		},
		{
			name: "alternate content resolves alias prefix by namespace",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:w0="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">
	<w:body>
		<w:p>
			<mc:AlternateContent>
				<mc:Choice Requires="w0">
					<w:r><w:t>choice</w:t></w:r>
				</mc:Choice>
				<mc:Fallback>
					<w:r><w:t>fallback</w:t></w:r>
				</mc:Fallback>
			</mc:AlternateContent>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "choice" {
					t.Fatalf("expected choice branch for alias prefix, got %q", text)
				}
			},
		},
		{
			name: "alternate content resolves alias prefix declared on alternate content",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">
	<w:body>
		<w:p>
			<mc:AlternateContent xmlns:w0="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
				<mc:Choice Requires="w0">
					<w:r><w:t>choice</w:t></w:r>
				</mc:Choice>
				<mc:Fallback>
					<w:r><w:t>fallback</w:t></w:r>
				</mc:Fallback>
			</mc:AlternateContent>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "choice" {
					t.Fatalf("expected choice branch for alternate-content alias prefix, got %q", text)
				}
			},
		},
		{
			name: "alternate content uses fallback for unknown word-like prefix",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006">
	<w:body>
		<w:p>
			<mc:AlternateContent>
				<mc:Choice Requires="w99">
					<w:r><w:t>choice</w:t></w:r>
				</mc:Choice>
				<mc:Fallback>
					<w:r><w:t>fallback</w:t></w:r>
				</mc:Fallback>
			</mc:AlternateContent>
		</w:p>
	</w:body>
</w:document>`,
			wantErr: false,
			check: func(t *testing.T, doc *Document) {
				if len(doc.Body.Elements) == 0 {
					t.Fatal("expected at least one element")
				}
				para, ok := doc.Body.Elements[0].(*Paragraph)
				if !ok {
					t.Fatalf("expected first element to be *Paragraph, got %T", doc.Body.Elements[0])
				}
				if text := para.GetText(); text != "fallback" {
					t.Fatalf("expected fallback branch, got %q", text)
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
