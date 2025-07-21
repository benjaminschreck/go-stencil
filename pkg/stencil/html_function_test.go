package stencil

import (
	"testing"
)

func TestHTMLFunction(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	
	// Test that HTML function is registered
	htmlFn, exists := registry.GetFunction("html")
	if !exists {
		t.Fatal("html() function should be registered")
	}
	
	if htmlFn.Name() != "html" {
		t.Errorf("Expected function name 'html', got %s", htmlFn.Name())
	}
	
	if htmlFn.MinArgs() != 1 {
		t.Errorf("Expected min args 1, got %d", htmlFn.MinArgs())
	}
	
	if htmlFn.MaxArgs() != 1 {
		t.Errorf("Expected max args 1, got %d", htmlFn.MaxArgs())
	}
}

func TestHTMLFunctionCall(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")
	
	tests := []struct {
		name     string
		input    string
		expected bool // whether it should succeed
	}{
		{
			name:     "empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "plain text",
			input:    "Hello World",
			expected: true,
		},
		{
			name:     "bold text",
			input:    "<b>Bold</b>",
			expected: true,
		},
		{
			name:     "italic text",
			input:    "<i>Italic</i>",
			expected: true,
		},
		{
			name:     "underlined text",
			input:    "<u>Underlined</u>",
			expected: true,
		},
		{
			name:     "strikethrough text",
			input:    "<s>Strikethrough</s>",
			expected: true,
		},
		{
			name:     "superscript text",
			input:    "<sup>Superscript</sup>",
			expected: true,
		},
		{
			name:     "subscript text",
			input:    "<sub>Subscript</sub>",
			expected: true,
		},
		{
			name:     "strong text",
			input:    "<strong>Strong</strong>",
			expected: true,
		},
		{
			name:     "emphasis text",
			input:    "<em>Emphasis</em>",
			expected: true,
		},
		{
			name:     "span text",
			input:    "<span>Span</span>",
			expected: true,
		},
		{
			name:     "line break",
			input:    "Line1<br>Line2",
			expected: true,
		},
		{
			name:     "line break self-closing",
			input:    "Line1<br/>Line2",
			expected: true,
		},
		{
			name:     "nested tags",
			input:    "<b><i>Bold and Italic</i></b>",
			expected: true,
		},
		{
			name:     "mixed formatting",
			input:    "Normal <b>Bold</b> and <i>Italic</i>",
			expected: true,
		},
		{
			name:     "invalid tag",
			input:    "<div>Invalid</div>",
			expected: false,
		},
		{
			name:     "malformed html",
			input:    "<b>Unclosed bold",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := htmlFn.Call(tt.input)
			
			if tt.expected {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
					return
				}
				
				// Check that result is an OOXMLFragment containing HTML runs
				fragment, ok := result.(*OOXMLFragment)
				if !ok {
					t.Errorf("Expected *OOXMLFragment, got %T", result)
					return
				}
				
				// Check that content is HTMLRuns
				htmlRuns, ok := fragment.Content.(*HTMLRuns)
				if !ok {
					t.Errorf("Expected *HTMLRuns in fragment content, got %T", fragment.Content)
					return
				}
				
				if len(htmlRuns.Runs) == 0 && tt.input != "" {
					t.Errorf("Expected non-empty runs for input %q", tt.input)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for input %q, got success", tt.input)
				}
			}
		})
	}
}

func TestHTMLFunctionNilInput(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")
	
	result, err := htmlFn.Call(nil)
	if err != nil {
		t.Errorf("Expected success with nil input, got error: %v", err)
	}
	
	if result != nil {
		t.Errorf("Expected nil result for nil input, got %v", result)
	}
}

func TestHTMLFunctionArgumentValidation(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")
	
	// Test no arguments
	_, err := htmlFn.Call()
	if err == nil {
		t.Error("Expected error with no arguments")
	}
	
	// Test too many arguments
	_, err = htmlFn.Call("test", "extra")
	if err == nil {
		t.Error("Expected error with too many arguments")
	}
}

func TestHTMLParsingBasicTags(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string // simplified description of expected structure
	}{
		{
			name:     "bold tag",
			html:     "<b>bold text</b>",
			expected: "one run with bold formatting",
		},
		{
			name:     "italic tag",
			html:     "<i>italic text</i>",
			expected: "one run with italic formatting",
		},
		{
			name:     "underline tag",
			html:     "<u>underlined text</u>",
			expected: "one run with underline formatting",
		},
		{
			name:     "strikethrough tag",
			html:     "<s>strikethrough text</s>",
			expected: "one run with strikethrough formatting",
		},
		{
			name:     "superscript tag",
			html:     "<sup>superscript text</sup>",
			expected: "one run with superscript formatting",
		},
		{
			name:     "subscript tag",
			html:     "<sub>subscript text</sub>",
			expected: "one run with subscript formatting",
		},
		{
			name:     "strong tag",
			html:     "<strong>strong text</strong>",
			expected: "one run with bold formatting",
		},
		{
			name:     "emphasis tag",
			html:     "<em>emphasis text</em>",
			expected: "one run with bold formatting",
		},
		{
			name:     "span tag",
			html:     "<span>span text</span>",
			expected: "one run with no special formatting",
		},
	}
	
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := htmlFn.Call(tt.html)
			if err != nil {
				t.Errorf("Expected success, got error: %v", err)
				return
			}
			
			fragment, ok := result.(*OOXMLFragment)
			if !ok {
				t.Errorf("Expected *OOXMLFragment, got %T", result)
				return
			}
			
			htmlRuns, ok := fragment.Content.(*HTMLRuns)
			if !ok {
				t.Errorf("Expected *HTMLRuns, got %T", fragment.Content)
				return
			}
			
			if len(htmlRuns.Runs) == 0 {
				t.Errorf("Expected at least one run, got empty runs")
				return
			}
			
			// For basic single-tag tests, we expect exactly one run
			if len(htmlRuns.Runs) != 1 {
				t.Errorf("Expected exactly one run for basic tag, got %d", len(htmlRuns.Runs))
			}
		})
	}
}

func TestHTMLParsingComplexStructures(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected int // expected number of runs
	}{
		{
			name:     "mixed formatting",
			html:     "Normal <b>bold</b> and <i>italic</i>",
			expected: 4, // "Normal ", "bold", " and ", "italic"
		},
		{
			name:     "nested tags",
			html:     "<b><i>bold italic</i></b>",
			expected: 1, // one run with both bold and italic
		},
		{
			name:     "line break",
			html:     "Line1<br>Line2",
			expected: 2, // text before break, text after break (break is inline)
		},
		{
			name:     "multiple line breaks",
			html:     "Line1<br>Line2<br>Line3",
			expected: 3, // three text segments
		},
		{
			name:     "complex nesting",
			html:     "<b>Bold <i>and italic</i> text</b>",
			expected: 3, // bold, bold+italic, bold
		},
	}
	
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := htmlFn.Call(tt.html)
			if err != nil {
				t.Errorf("Expected success, got error: %v", err)
				return
			}
			
			fragment, ok := result.(*OOXMLFragment)
			if !ok {
				t.Errorf("Expected *OOXMLFragment, got %T", result)
				return
			}
			
			htmlRuns, ok := fragment.Content.(*HTMLRuns)
			if !ok {
				t.Errorf("Expected *HTMLRuns, got %T", fragment.Content)
				return
			}
			
			if len(htmlRuns.Runs) != tt.expected {
				t.Errorf("Expected %d runs, got %d", tt.expected, len(htmlRuns.Runs))
			}
		})
	}
}