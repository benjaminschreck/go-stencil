package xml

import (
	"encoding/xml"
	"testing"
)

func TestDocumentExtractNamespaces(t *testing.T) {
	tests := []struct {
		name     string
		attrs    []xml.Attr
		expected map[string]string
	}{
		{
			name: "Standard namespaces",
			attrs: []xml.Attr{
				{Name: xml.Name{Local: "xmlns:w"}, Value: "http://w"},
				{Name: xml.Name{Local: "xmlns:r"}, Value: "http://r"},
			},
			expected: map[string]string{
				"w": "http://w",
				"r": "http://r",
			},
		},
		{
			name: "Namespace with Space attribute",
			attrs: []xml.Attr{
				{Name: xml.Name{Space: "xmlns", Local: "w14"}, Value: "http://w14"},
			},
			expected: map[string]string{
				"w14": "http://w14",
			},
		},
		{
			name: "Default namespace",
			attrs: []xml.Attr{
				{Name: xml.Name{Local: "xmlns"}, Value: "http://default"},
			},
			expected: map[string]string{
				"": "http://default",
			},
		},
		{
			name: "Mixed forms",
			attrs: []xml.Attr{
				{Name: xml.Name{Local: "xmlns:w"}, Value: "http://w"},
				{Name: xml.Name{Space: "xmlns", Local: "w14"}, Value: "http://w14"},
				{Name: xml.Name{Local: "xmlns"}, Value: "http://default"},
			},
			expected: map[string]string{
				"w":   "http://w",
				"w14": "http://w14",
				"":    "http://default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{Attrs: tt.attrs}
			result := doc.ExtractNamespaces()

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d namespaces, got %d", len(tt.expected), len(result))
			}

			for prefix, uri := range tt.expected {
				if result[prefix] != uri {
					t.Errorf("Expected %s=%s, got %s", prefix, uri, result[prefix])
				}
			}
		})
	}
}

func TestDocumentMergeNamespaces(t *testing.T) {
	tests := []struct {
		name        string
		existing    []xml.Attr
		additional  map[string]string
		expectCount int
		expectAttrs []string
	}{
		{
			name: "Merge new namespaces",
			existing: []xml.Attr{
				{Name: xml.Name{Local: "xmlns:w"}, Value: "http://w"},
			},
			additional: map[string]string{
				"w14":  "http://w14",
				"wp14": "http://wp14",
			},
			expectCount: 3,
			expectAttrs: []string{"xmlns:w", "xmlns:w14", "xmlns:wp14"},
		},
		{
			name: "Don't duplicate existing",
			existing: []xml.Attr{
				{Name: xml.Name{Local: "xmlns:w14"}, Value: "http://w14"},
			},
			additional: map[string]string{
				"w14": "http://different", // Should be ignored
			},
			expectCount: 1,
			expectAttrs: []string{"xmlns:w14"},
		},
		{
			name:     "Default namespace",
			existing: []xml.Attr{},
			additional: map[string]string{
				"": "http://default",
			},
			expectCount: 1,
			expectAttrs: []string{"xmlns"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &Document{Attrs: tt.existing}
			doc.MergeNamespaces(tt.additional)

			if len(doc.Attrs) != tt.expectCount {
				t.Errorf("Expected %d attrs, got %d", tt.expectCount, len(doc.Attrs))
			}

			for _, expected := range tt.expectAttrs {
				found := false
				for _, attr := range doc.Attrs {
					attrName := attr.Name.Local
					if attr.Name.Space != "" {
						attrName = attr.Name.Space + ":" + attr.Name.Local
					}
					if attrName == expected || attr.Name.Local == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected attribute %s not found", expected)
				}
			}
		})
	}
}
