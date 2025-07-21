package stencil

import (
	"bytes"
	"strings"
	"testing"
)

func TestHTMLInTableCells(t *testing.T) {
	// Create a simple template with HTML in table cells
	templateXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:tbl>
      <w:tr>
        <w:tc>
          <w:p>
            <w:r>
              <w:t>{{for row in items}}</w:t>
            </w:r>
          </w:p>
        </w:tc>
      </w:tr>
      <w:tr>
        <w:tc>
          <w:p>
            <w:r>
              <w:t>{{html(row.text)}}</w:t>
            </w:r>
          </w:p>
        </w:tc>
      </w:tr>
      <w:tr>
        <w:tc>
          <w:p>
            <w:r>
              <w:t>{{end}}</w:t>
            </w:r>
          </w:p>
        </w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`

	// Parse the template
	doc, err := ParseDocument(bytes.NewReader([]byte(templateXML)))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// Create test data
	data := TemplateData{
		"items": []map[string]interface{}{
			{"text": "<b>Bold text</b>"},
			{"text": "<i>Italic text</i>"},
		},
	}

	// Create render context
	ctx := &renderContext{
		ooxmlFragments: make(map[string]interface{}),
	}

	// Render the template
	rendered, err := RenderDocumentWithContext(doc, data, ctx)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Marshal to XML
	xmlBytes, err := marshalDocumentWithNamespaces(rendered)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	result := string(xmlBytes)
	
	// Debug output
	t.Logf("Rendered XML:\n%s", result)

	// Check that the HTML was processed
	if strings.Contains(result, "{{html(") {
		t.Error("Template expressions were not evaluated in table cells")
	}

	// Check for bold formatting
	if !strings.Contains(result, "<w:b/>") {
		t.Error("Bold formatting not found in output")
	}

	// Check for italic formatting
	if !strings.Contains(result, "<w:i/>") {
		t.Error("Italic formatting not found in output")
	}
}