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

func TestMarshalDocumentWithNamespaces_DoesNotRewriteLiteralPrefixText(t *testing.T) {
	doc := &Document{
		Body: &Body{
			Elements: []BodyElement{
				&Paragraph{
					Runs: []Run{
						{Text: &Text{Content: "main:demo wordml:demo"}},
					},
				},
			},
		},
	}

	output, err := marshalDocumentWithNamespaces(doc)
	if err != nil {
		t.Fatalf("Marshaling failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "main:demo wordml:demo") {
		t.Fatalf("expected literal text to be preserved, got:\n%s", outputStr)
	}
}

func TestRewritePrefixInsideTags_DoesNotRewriteQuotedAttributeValues(t *testing.T) {
	in := `<w:p w:custom="main:demo" main:foo="1"><w:r><w:t>ok</w:t></w:r></w:p>`
	out := rewritePrefixInsideTags(in, "main:", "w:")

	if !strings.Contains(out, `w:custom="main:demo"`) {
		t.Fatalf("expected quoted attribute value to stay unchanged, got: %s", out)
	}
	if !strings.Contains(out, `w:foo="1"`) {
		t.Fatalf("expected attribute name prefix rewrite, got: %s", out)
	}
}

func TestMarshalDocumentWithNamespaces_DoesNotRewriteAttributeValues(t *testing.T) {
	doc := &Document{
		Body: &Body{
			Elements: []BodyElement{
				&Paragraph{
					Attrs: []xml.Attr{
						{
							Name:  xml.Name{Space: "http://schemas.openxmlformats.org/wordprocessingml/2006/main", Local: "custom"},
							Value: "main:demo",
						},
					},
					Runs: []Run{
						{Text: &Text{Content: "ok"}},
					},
				},
			},
		},
	}

	output, err := marshalDocumentWithNamespaces(doc)
	if err != nil {
		t.Fatalf("Marshaling failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, `w:custom="main:demo"`) {
		t.Fatalf("expected attribute value to stay unchanged, got:\n%s", outputStr)
	}
	if strings.Contains(outputStr, `w:custom="w:demo"`) {
		t.Fatalf("attribute value was incorrectly rewritten, got:\n%s", outputStr)
	}
}

func TestRewritePrefixInsideTags_DoesNotRewriteCDATACommentsOrPI(t *testing.T) {
	in := `<w:p><![CDATA[main:demo]]><!-- main:comment --><?pi main:proc?><main:r main:foo="1"><w:t>ok</w:t></main:r></w:p>`
	out := rewritePrefixInsideTags(in, "main:", "w:")

	if !strings.Contains(out, "<![CDATA[main:demo]]>") {
		t.Fatalf("expected CDATA payload unchanged, got: %s", out)
	}
	if !strings.Contains(out, "<!-- main:comment -->") {
		t.Fatalf("expected comment payload unchanged, got: %s", out)
	}
	if !strings.Contains(out, "<?pi main:proc?>") {
		t.Fatalf("expected processing instruction payload unchanged, got: %s", out)
	}
	if !strings.Contains(out, "<w:r w:foo=\"1\">") {
		t.Fatalf("expected tag/attribute prefixes to be rewritten, got: %s", out)
	}
}
