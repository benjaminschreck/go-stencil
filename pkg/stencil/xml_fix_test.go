package stencil

import (
	"encoding/xml"
	"strings"
	"testing"
)

func TestExistingMarshalingPreservesAttrs(t *testing.T) {
	// Create document with namespace attributes
	doc := &Document{
		XMLName: xml.Name{Local: "document"},
		Attrs: []xml.Attr{
			{Name: xml.Name{Local: "xmlns:w"}, Value: "http://main"},
			{Name: xml.Name{Local: "xmlns:w14"}, Value: "http://w14"},
			{Name: xml.Name{Local: "xmlns:wp14"}, Value: "http://wp14"},
		},
		Body: &Body{
			Elements: []BodyElement{
				&Paragraph{
					Runs: []Run{
						{Text: &Text{Content: "Test"}},
					},
				},
			},
		},
	}

	// Use existing marshalDocumentWithNamespaces
	output, err := marshalDocumentWithNamespaces(doc)
	if err != nil {
		t.Fatalf("Marshaling failed: %v", err)
	}

	outputStr := string(output)

	// Verify all namespaces are present
	requiredNamespaces := []string{
		`xmlns:w="http://main"`,
		`xmlns:w14="http://w14"`,
		`xmlns:wp14="http://wp14"`,
	}

	for _, ns := range requiredNamespaces {
		if !strings.Contains(outputStr, ns) {
			t.Errorf("Output missing namespace: %s\nOutput:\n%s", ns, outputStr)
		}
	}

	maxLen := 200
	if len(outputStr) < maxLen {
		maxLen = len(outputStr)
	}
	t.Logf("✅ Existing marshaling preserves Attrs:\n%s", outputStr[:maxLen])
}
