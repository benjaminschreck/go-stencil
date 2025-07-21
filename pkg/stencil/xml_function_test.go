package stencil

import (
	"testing"
)

func TestXMLFunction(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	
	// Test that XML function is registered
	xmlFn, exists := registry.GetFunction("xml")
	if !exists {
		t.Fatal("xml() function should be registered")
	}
	
	if xmlFn.Name() != "xml" {
		t.Errorf("Expected function name 'xml', got %s", xmlFn.Name())
	}
	
	if xmlFn.MinArgs() != 1 {
		t.Errorf("Expected min args 1, got %d", xmlFn.MinArgs())
	}
	
	if xmlFn.MaxArgs() != 1 {
		t.Errorf("Expected max args 1, got %d", xmlFn.MaxArgs())
	}
}

func TestXMLFunctionCall(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	xmlFn, _ := registry.GetFunction("xml")
	
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
			name:     "simple text node",
			input:    "Hello World",
			expected: true,
		},
		{
			name:     "simple XML element",
			input:    "<w:p>Hello</w:p>",
			expected: true,
		},
		{
			name:     "multiple XML elements",
			input:    "<w:p>First</w:p><w:p>Second</w:p>",
			expected: true,
		},
		{
			name:     "XML with attributes",
			input:    `<w:p w:val="test">Content</w:p>`,
			expected: true,
		},
		{
			name:     "nested XML elements",
			input:    "<w:p><w:r><w:t>Text</w:t></w:r></w:p>",
			expected: true,
		},
		{
			name:     "OOXML table fragment",
			input:    `<w:tbl xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:tr><w:tc><w:p><w:r><w:t>Cell</w:t></w:r></w:p></w:tc></w:tr></w:tbl>`,
			expected: true,
		},
		{
			name:     "XML with CDATA",
			input:    "<w:p><![CDATA[Raw content]]></w:p>",
			expected: true,
		},
		{
			name:     "self-closing tag",
			input:    "<w:br/>",
			expected: true,
		},
		{
			name:     "unclosed tag",
			input:    "<w:p>Unclosed",
			expected: false,
		},
		{
			name:     "malformed XML",
			input:    "<w:p><invalid><w:p>",
			expected: false,
		},
		{
			name:     "invalid XML characters",
			input:    "<w:p>Content with \x00 null character</w:p>",
			expected: false,
		},
		{
			name:     "mixed content with text and elements",
			input:    "Text before <w:p>Element</w:p> text after",
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := xmlFn.Call(tt.input)
			
			if tt.expected {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
					return
				}
				
				// Check that result is an OOXMLFragment containing XML fragment
				fragment, ok := result.(*OOXMLFragment)
				if !ok {
					t.Errorf("Expected *OOXMLFragment, got %T", result)
					return
				}
				
				// Check that content is XMLFragment
				xmlFragment, ok := fragment.Content.(*XMLFragment)
				if !ok {
					t.Errorf("Expected *XMLFragment in fragment content, got %T", fragment.Content)
					return
				}
				
				if len(xmlFragment.Elements) == 0 && tt.input != "" {
					t.Errorf("Expected non-empty elements for input %q", tt.input)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for input %q, got success", tt.input)
				}
			}
		})
	}
}

func TestXMLFunctionNilInput(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	xmlFn, _ := registry.GetFunction("xml")
	
	result, err := xmlFn.Call(nil)
	if err == nil {
		t.Error("Expected error with nil input")
	}
	
	if result != nil {
		t.Errorf("Expected nil result for nil input, got %v", result)
	}
}

func TestXMLFunctionArgumentValidation(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	xmlFn, _ := registry.GetFunction("xml")
	
	// Test no arguments
	_, err := xmlFn.Call()
	if err == nil {
		t.Error("Expected error with no arguments")
	}
	
	// Test too many arguments
	_, err = xmlFn.Call("test", "extra")
	if err == nil {
		t.Error("Expected error with too many arguments")
	}
	
	// Test non-string argument
	_, err = xmlFn.Call(123)
	if err == nil {
		t.Error("Expected error with non-string argument")
	}
}

func TestXMLParsingValidation(t *testing.T) {
	tests := []struct {
		name     string
		xml      string
		expected string // simplified description of expected structure
	}{
		{
			name:     "simple element",
			xml:      "<w:p>text</w:p>",
			expected: "one element with text content",
		},
		{
			name:     "element with attributes",
			xml:      `<w:p w:val="test" w:other="value">content</w:p>`,
			expected: "one element with attributes and content",
		},
		{
			name:     "nested elements",
			xml:      "<w:p><w:r>inner</w:r></w:p>",
			expected: "one element with nested element",
		},
		{
			name:     "multiple elements",
			xml:      "<w:p>first</w:p><w:p>second</w:p>",
			expected: "two separate elements",
		},
		{
			name:     "self-closing element",
			xml:      "<w:br/>",
			expected: "one self-closing element",
		},
		{
			name:     "mixed content",
			xml:      "Text <w:b>bold</w:b> more text",
			expected: "text and element content",
		},
	}
	
	registry := GetDefaultFunctionRegistry()
	xmlFn, _ := registry.GetFunction("xml")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := xmlFn.Call(tt.xml)
			if err != nil {
				t.Errorf("Expected success, got error: %v", err)
				return
			}
			
			fragment, ok := result.(*OOXMLFragment)
			if !ok {
				t.Errorf("Expected *OOXMLFragment, got %T", result)
				return
			}
			
			xmlFragment, ok := fragment.Content.(*XMLFragment)
			if !ok {
				t.Errorf("Expected *XMLFragment, got %T", fragment.Content)
				return
			}
			
			if len(xmlFragment.Elements) == 0 {
				t.Errorf("Expected at least one element, got empty elements")
				return
			}
		})
	}
}

func TestXMLFunctionErrorHandling(t *testing.T) {
	tests := []struct {
		name  string
		input string
		error string // expected error substring
	}{
		{
			name:  "unclosed tag",
			input: "<w:p>unclosed",
			error: "XML syntax error",
		},
		{
			name:  "mismatched tags",
			input: "<w:p></w:r>",
			error: "XML syntax error",
		},
		{
			name:  "invalid tag name",
			input: "<123invalid>content</123invalid>",
			error: "XML syntax error",
		},
		{
			name:  "invalid XML with ampersand",
			input: `<w:p>Content & more</w:p>`,
			error: "XML syntax error",
		},
	}
	
	registry := GetDefaultFunctionRegistry()
	xmlFn, _ := registry.GetFunction("xml")
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := xmlFn.Call(tt.input)
			
			if err == nil {
				t.Errorf("Expected error for input %q, got success", tt.input)
				return
			}
			
			if result != nil {
				t.Errorf("Expected nil result for error case, got %v", result)
			}
			
			// Check if error message is appropriate (we don't need exact match)
			if err.Error() == "" {
				t.Errorf("Expected non-empty error message")
			}
		})
	}
}