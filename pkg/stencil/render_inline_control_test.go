package stencil

import (
	"testing"
)

// TestProcessTemplateTextWithIf tests if statements in inline template text
func TestProcessTemplateTextWithIf(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		data     TemplateData
		expected string
	}{
		{
			name:     "Simple if - true condition",
			text:     "{{if x > 5}}yes{{end}}",
			data:     TemplateData{"x": 10},
			expected: "yes",
		},
		{
			name:     "Simple if - false condition",
			text:     "{{if x > 5}}yes{{end}}",
			data:     TemplateData{"x": 3},
			expected: "",
		},
		{
			name:     "If with else - true condition",
			text:     "{{if x > 5}}yes{{else}}no{{end}}",
			data:     TemplateData{"x": 10},
			expected: "yes",
		},
		{
			name:     "If with else - false condition",
			text:     "{{if x > 5}}yes{{else}}no{{end}}",
			data:     TemplateData{"x": 3},
			expected: "no",
		},
		{
			name:     "If with elsif - first true",
			text:     "{{if x > 10}}big{{elsif x > 5}}medium{{else}}small{{end}}",
			data:     TemplateData{"x": 15},
			expected: "big",
		},
		{
			name:     "If with elsif - second true",
			text:     "{{if x > 10}}big{{elsif x > 5}}medium{{else}}small{{end}}",
			data:     TemplateData{"x": 7},
			expected: "medium",
		},
		{
			name:     "If with elsif - else",
			text:     "{{if x > 10}}big{{elsif x > 5}}medium{{else}}small{{end}}",
			data:     TemplateData{"x": 3},
			expected: "small",
		},
		{
			name:     "If with text before and after",
			text:     "Value: {{if x > 5}}high{{else}}low{{end}}!",
			data:     TemplateData{"x": 10},
			expected: "Value: high!",
		},
		{
			name:     "Multiple ifs",
			text:     "{{if a}}A{{end}} {{if b}}B{{end}}",
			data:     TemplateData{"a": true, "b": true},
			expected: "A B",
		},
		{
			name:     "Nested if",
			text:     "{{if x > 5}}outer{{if x > 10}}inner{{end}}{{end}}",
			data:     TemplateData{"x": 15},
			expected: "outerinner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processTemplateText(tt.text, tt.data)
			if err != nil {
				t.Errorf("processTemplateText() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("processTemplateText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestProcessTemplateTextWithUnless tests unless statements in inline template text
func TestProcessTemplateTextWithUnless(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		data     TemplateData
		expected string
	}{
		{
			name:     "Simple unless - false condition (execute)",
			text:     "{{unless x > 5}}yes{{end}}",
			data:     TemplateData{"x": 3},
			expected: "yes",
		},
		{
			name:     "Simple unless - true condition (skip)",
			text:     "{{unless x > 5}}yes{{end}}",
			data:     TemplateData{"x": 10},
			expected: "",
		},
		{
			name:     "Unless with else - false condition",
			text:     "{{unless x > 5}}low{{else}}high{{end}}",
			data:     TemplateData{"x": 3},
			expected: "low",
		},
		{
			name:     "Unless with else - true condition",
			text:     "{{unless x > 5}}low{{else}}high{{end}}",
			data:     TemplateData{"x": 10},
			expected: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processTemplateText(tt.text, tt.data)
			if err != nil {
				t.Errorf("processTemplateText() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("processTemplateText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestProcessTemplateTextLoopBody tests if statements in loop bodies (the original bug)
// These tests simulate the loop body processing (without the {{for}}...{{end}} wrapper)
func TestProcessTemplateTextLoopBody(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		data     TemplateData
		expected string
	}{
		{
			name:     "If with index - first iteration (i=0)",
			text:     "{{if i > 0}}, {{end}}{{item}}",
			data:     TemplateData{"i": 0, "item": "A"},
			expected: "A",
		},
		{
			name:     "If with index - second iteration (i=1)",
			text:     "{{if i > 0}}, {{end}}{{item}}",
			data:     TemplateData{"i": 1, "item": "B"},
			expected: ", B",
		},
		{
			name:     "Unless with index - first iteration (i=0)",
			text:     "{{unless i == 0}}, {{end}}{{item}}",
			data:     TemplateData{"i": 0, "item": "A"},
			expected: "A",
		},
		{
			name:     "Unless with index - second iteration (i=1)",
			text:     "{{unless i == 0}}, {{end}}{{item}}",
			data:     TemplateData{"i": 1, "item": "B"},
			expected: ", B",
		},
		{
			name: "If with object access",
			text: "{{if i > 0}}, {{end}}{{sale.region}}",
			data: TemplateData{
				"i":    1,
				"sale": map[string]interface{}{"region": "North"},
			},
			expected: ", North",
		},
		{
			name:     "If with multiple conditions",
			text:     "{{if i > 0 & i < 3}}|{{end}}{{item}}",
			data:     TemplateData{"i": 1, "item": "B"},
			expected: "|B",
		},
		{
			name:     "If-else - first iteration",
			text:     "{{if i == 0}}[{{item}}]{{else}}({{item}}){{end}}",
			data:     TemplateData{"i": 0, "item": "A"},
			expected: "[A]",
		},
		{
			name:     "If-else - second iteration",
			text:     "{{if i == 0}}[{{item}}]{{else}}({{item}}){{end}}",
			data:     TemplateData{"i": 1, "item": "B"},
			expected: "(B)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processTemplateText(tt.text, tt.data)
			if err != nil {
				t.Errorf("processTemplateText() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("processTemplateText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestProcessTemplateTextWithVariables tests that variables still work correctly
func TestProcessTemplateTextWithVariables(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		data     TemplateData
		expected string
	}{
		{
			name:     "Simple variable",
			text:     "Hello {{name}}!",
			data:     TemplateData{"name": "World"},
			expected: "Hello World!",
		},
		{
			name:     "Multiple variables",
			text:     "{{first}} {{last}}",
			data:     TemplateData{"first": "John", "last": "Doe"},
			expected: "John Doe",
		},
		{
			name:     "Nested field access",
			text:     "{{customer.name}}",
			data:     TemplateData{"customer": map[string]interface{}{"name": "Alice"}},
			expected: "Alice",
		},
		{
			name:     "Expression",
			text:     "{{price * 1.2}}",
			data:     TemplateData{"price": 10},
			expected: "12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processTemplateText(tt.text, tt.data)
			if err != nil {
				t.Errorf("processTemplateText() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("processTemplateText() = %q, want %q", result, tt.expected)
			}
		})
	}
}
