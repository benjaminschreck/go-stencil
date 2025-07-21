package stencil

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Token
	}{
		{
			name:  "plain text",
			input: "Hello World",
			want: []Token{
				{Type: TokenText, Value: "Hello World"},
			},
		},
		{
			name:  "simple variable",
			input: "Hello {{name}}!",
			want: []Token{
				{Type: TokenText, Value: "Hello "},
				{Type: TokenVariable, Value: "name"},
				{Type: TokenText, Value: "!"},
			},
		},
		{
			name:  "multiple variables",
			input: "{{greeting}} {{name}}, you have {{count}} messages",
			want: []Token{
				{Type: TokenVariable, Value: "greeting"},
				{Type: TokenText, Value: " "},
				{Type: TokenVariable, Value: "name"},
				{Type: TokenText, Value: ", you have "},
				{Type: TokenVariable, Value: "count"},
				{Type: TokenText, Value: " messages"},
			},
		},
		{
			name:  "if statement",
			input: "{{if condition}}Show this{{end}}",
			want: []Token{
				{Type: TokenIf, Value: "condition"},
				{Type: TokenText, Value: "Show this"},
				{Type: TokenEnd, Value: ""},
			},
		},
		{
			name:  "if-else statement",
			input: "{{if isPremium}}Premium content{{else}}Regular content{{end}}",
			want: []Token{
				{Type: TokenIf, Value: "isPremium"},
				{Type: TokenText, Value: "Premium content"},
				{Type: TokenElse, Value: ""},
				{Type: TokenText, Value: "Regular content"},
				{Type: TokenEnd, Value: ""},
			},
		},
		{
			name:  "for loop",
			input: "{{for item in items}}Item: {{item.name}}{{end}}",
			want: []Token{
				{Type: TokenFor, Value: "item in items"},
				{Type: TokenText, Value: "Item: "},
				{Type: TokenVariable, Value: "item.name"},
				{Type: TokenEnd, Value: ""},
			},
		},
		{
			name:  "nested structures",
			input: "{{for user in users}}{{if user.active}}{{user.name}}{{end}}{{end}}",
			want: []Token{
				{Type: TokenFor, Value: "user in users"},
				{Type: TokenIf, Value: "user.active"},
				{Type: TokenVariable, Value: "user.name"},
				{Type: TokenEnd, Value: ""},
				{Type: TokenEnd, Value: ""},
			},
		},
		{
			name:  "elsif statement",
			input: "{{if x > 10}}Large{{elsif x > 5}}Medium{{else}}Small{{end}}",
			want: []Token{
				{Type: TokenIf, Value: "x > 10"},
				{Type: TokenText, Value: "Large"},
				{Type: TokenElsif, Value: "x > 5"},
				{Type: TokenText, Value: "Medium"},
				{Type: TokenElse, Value: ""},
				{Type: TokenText, Value: "Small"},
				{Type: TokenEnd, Value: ""},
			},
		},
		{
			name:  "unless statement",
			input: "{{unless hidden}}Visible content{{end}}",
			want: []Token{
				{Type: TokenUnless, Value: "hidden"},
				{Type: TokenText, Value: "Visible content"},
				{Type: TokenEnd, Value: ""},
			},
		},
		{
			name:  "empty template tags",
			input: "{{}}",
			want: []Token{
				{Type: TokenText, Value: "{{}}"},
			},
		},
		{
			name:  "unclosed template tag",
			input: "Hello {{name",
			want: []Token{
				{Type: TokenText, Value: "Hello {{name"},
			},
		},
		{
			name:  "whitespace in tags",
			input: "{{ name }} and {{ if condition }}",
			want: []Token{
				{Type: TokenVariable, Value: "name"},
				{Type: TokenText, Value: " and "},
				{Type: TokenIf, Value: "condition"},
			},
		},
		{
			name:  "expression in variable",
			input: "Total: {{price * quantity}}",
			want: []Token{
				{Type: TokenText, Value: "Total: "},
				{Type: TokenVariable, Value: "price * quantity"},
			},
		},
		{
			name:  "function call",
			input: "{{uppercase(name)}}",
			want: []Token{
				{Type: TokenVariable, Value: "uppercase(name)"},
			},
		},
		{
			name:  "pageBreak",
			input: "Page 1{{pageBreak}}Page 2",
			want: []Token{
				{Type: TokenText, Value: "Page 1"},
				{Type: TokenPageBreak, Value: ""},
				{Type: TokenText, Value: "Page 2"},
			},
		},
		{
			name:  "include directive",
			input: "{{include header}}Content{{include footer}}",
			want: []Token{
				{Type: TokenInclude, Value: "header"},
				{Type: TokenText, Value: "Content"},
				{Type: TokenInclude, Value: "footer"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tokenize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindTemplateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "find variables",
			input: "Hello {{name}}, your balance is {{balance}}",
			want:  []string{"{{name}}", "{{balance}}"},
		},
		{
			name:  "find control structures",
			input: "{{if premium}}Welcome!{{else}}Sign up{{end}}",
			want:  []string{"{{if premium}}", "{{else}}", "{{end}}"},
		},
		{
			name:  "no tokens",
			input: "Plain text without any templates",
			want:  []string{},
		},
		{
			name:  "unclosed token ignored",
			input: "Hello {{name}} and {{unclosed",
			want:  []string{"{{name}}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindTemplateTokens(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindTemplateTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}