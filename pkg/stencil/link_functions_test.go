package stencil

import (
	"testing"
)

func TestReplaceLinkFunction(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "Valid URL",
			args:     []interface{}{"https://example.com"},
			expected: LinkReplacementMarker{URL: "https://example.com"},
			wantErr:  false,
		},
		{
			name:     "URL with spaces (should be trimmed)",
			args:     []interface{}{" https://example.com "},
			expected: LinkReplacementMarker{URL: "https://example.com"},
			wantErr:  false,
		},
		{
			name:     "Empty URL",
			args:     []interface{}{""},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "No arguments",
			args:     []interface{}{},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Too many arguments",
			args:     []interface{}{"https://example.com", "extra"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Non-string argument",
			args:     []interface{}{123},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "HTTP URL",
			args:     []interface{}{"http://example.com"},
			expected: LinkReplacementMarker{URL: "http://example.com"},
			wantErr:  false,
		},
		{
			name:     "Relative URL",
			args:     []interface{}{"/path/to/page"},
			expected: LinkReplacementMarker{URL: "/path/to/page"},
			wantErr:  false,
		},
		{
			name:     "Email link",
			args:     []interface{}{"mailto:test@example.com"},
			expected: LinkReplacementMarker{URL: "mailto:test@example.com"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := replaceLinkFunc(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceLinkFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				marker, ok := result.(LinkReplacementMarker)
				if !ok {
					t.Errorf("Expected LinkReplacementMarker, got %T", result)
					return
				}
				expectedMarker := tt.expected.(LinkReplacementMarker)
				if marker.URL != expectedMarker.URL {
					t.Errorf("Expected URL %s, got %s", expectedMarker.URL, marker.URL)
				}
			}
		})
	}
}

func TestLinkReplacementMarkerInterface(t *testing.T) {
	marker := LinkReplacementMarker{URL: "https://example.com"}
	
	// Test that it implements the marker interface
	if !marker.isMarker() {
		t.Error("linkReplacementMarker should implement marker interface")
	}
}

func TestLinkFunctionRegistration(t *testing.T) {
	// Create a new registry
	registry := NewFunctionRegistry()
	
	// Register link functions
	registerLinkFunctions(registry)
	
	// Check that replaceLink is registered
	fn, ok := registry.GetFunction("replaceLink")
	if !ok {
		t.Error("replaceLink function not registered")
	}
	
	if fn == nil {
		t.Error("replaceLink function is nil")
	}
}