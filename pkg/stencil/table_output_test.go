package stencil

import (
	"strings"
	"testing"
)

func TestTableInOutput(t *testing.T) {
	// Create a simple document with a table
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>Before table</w:t>
			</w:r>
		</w:p>
		<w:tbl>
			<w:tblPr>
				<w:tblStyle w:val="TableGrid"/>
			</w:tblPr>
			<w:tblGrid>
				<w:gridCol w:w="2000"/>
				<w:gridCol w:w="2000"/>
			</w:tblGrid>
			<w:tr>
				<w:tc>
					<w:p>
						<w:r>
							<w:t>Cell 1</w:t>
						</w:r>
					</w:p>
				</w:tc>
				<w:tc>
					<w:p>
						<w:r>
							<w:t>Cell 2</w:t>
						</w:r>
					</w:p>
				</w:tc>
			</w:tr>
		</w:tbl>
		<w:p>
			<w:r>
				<w:t>After table</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`

	// Parse the document
	doc, err := ParseDocument(strings.NewReader(docXML))
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Check that table was parsed
	if len(doc.Body.Elements) != 3 {
		t.Errorf("Expected 3 body elements, got %d", len(doc.Body.Elements))
	}

	// Check that second element is a table
	if _, ok := doc.Body.Elements[1].(*Table); !ok {
		t.Errorf("Expected second element to be a table, got %T", doc.Body.Elements[1])
	}

	// Render the document
	rendered, err := RenderDocument(doc, TemplateData{})
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Check that table is preserved after rendering
	if len(rendered.Body.Elements) != 3 {
		t.Errorf("Expected 3 rendered body elements, got %d", len(rendered.Body.Elements))
	}

	if _, ok := rendered.Body.Elements[1].(*Table); !ok {
		t.Errorf("Expected second rendered element to be a table, got %T", rendered.Body.Elements[1])
	}

	// Now marshal to XML using the internal function
	xmlBytes, err := marshalDocumentWithNamespaces(rendered)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	xmlStr := string(xmlBytes)
	
	// Check that table is in the output
	if !strings.Contains(xmlStr, "<w:tbl>") {
		t.Errorf("Expected <w:tbl> in output, but not found")
		t.Logf("Output XML:\n%s", xmlStr)
	}

	// Also check that we have the correct structure
	if !strings.Contains(xmlStr, "Cell 1") {
		t.Errorf("Expected 'Cell 1' in output, but not found")
	}
	if !strings.Contains(xmlStr, "Cell 2") {
		t.Errorf("Expected 'Cell 2' in output, but not found")
	}
}

func TestTableWithForLoop(t *testing.T) {
	// Create a document with a table containing a for loop
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>Items:</w:t>
			</w:r>
		</w:p>
		<w:tbl>
			<w:tblPr>
				<w:tblStyle w:val="TableGrid"/>
			</w:tblPr>
			<w:tblGrid>
				<w:gridCol w:w="2000"/>
				<w:gridCol w:w="2000"/>
			</w:tblGrid>
			<w:tr>
				<w:tc>
					<w:p>
						<w:r>
							<w:t>{{for item in items}}</w:t>
						</w:r>
					</w:p>
				</w:tc>
				<w:tc>
					<w:p>
						<w:r>
							<w:t></w:t>
						</w:r>
					</w:p>
				</w:tc>
			</w:tr>
			<w:tr>
				<w:tc>
					<w:p>
						<w:r>
							<w:t>{{item}}</w:t>
						</w:r>
					</w:p>
				</w:tc>
				<w:tc>
					<w:p>
						<w:r>
							<w:t>Value</w:t>
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
				<w:tc>
					<w:p>
						<w:r>
							<w:t></w:t>
						</w:r>
					</w:p>
				</w:tc>
			</w:tr>
		</w:tbl>
	</w:body>
</w:document>`

	// Parse the document
	doc, err := ParseDocument(strings.NewReader(docXML))
	if err != nil {
		t.Fatalf("Failed to parse document: %v", err)
	}

	// Render with test data
	data := TemplateData{
		"items": []string{"Item 1", "Item 2", "Item 3"},
	}

	rendered, err := RenderDocument(doc, data)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Should still have a table
	hasTable := false
	for _, elem := range rendered.Body.Elements {
		if _, ok := elem.(*Table); ok {
			hasTable = true
			break
		}
	}

	if !hasTable {
		t.Errorf("Expected table in rendered output, but none found")
		t.Logf("Rendered elements: %d", len(rendered.Body.Elements))
		for i, elem := range rendered.Body.Elements {
			t.Logf("  Element %d: %T", i, elem)
		}
	}

	// Marshal to XML
	xmlBytes, err := marshalDocumentWithNamespaces(rendered)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	xmlStr := string(xmlBytes)
	
	// Check that table is in the output
	if !strings.Contains(xmlStr, "<w:tbl>") {
		t.Errorf("Expected <w:tbl> in output, but not found")
	}

	// Check that all items are in the output
	for _, item := range []string{"Item 1", "Item 2", "Item 3"} {
		if !strings.Contains(xmlStr, item) {
			t.Errorf("Expected '%s' in output, but not found", item)
		}
	}
}

// TestFullDocxPipeline tests the full DOCX creation pipeline
func TestFullDocxPipeline(t *testing.T) {
	// Create a temp DOCX file with a table
	engine := New()
	defer engine.Close()

	// Create a minimal DOCX structure in memory
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>Title</w:t>
			</w:r>
		</w:p>
		<w:tbl>
			<w:tblPr>
				<w:tblStyle w:val="TableGrid"/>
			</w:tblPr>
			<w:tblGrid>
				<w:gridCol w:w="2000"/>
			</w:tblGrid>
			<w:tr>
				<w:tc>
					<w:p>
						<w:r>
							<w:t>Test Cell</w:t>
						</w:r>
					</w:p>
				</w:tc>
			</w:tr>
		</w:tbl>
	</w:body>
</w:document>`

	// Parse and render
	doc, err := ParseDocument(strings.NewReader(docXML))
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	rendered, err := RenderDocument(doc, TemplateData{})
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Marshal with namespaces
	xmlBytes, err := marshalDocumentWithNamespaces(rendered)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	xmlStr := string(xmlBytes)
	if !strings.Contains(xmlStr, "<w:tbl>") {
		t.Errorf("Table missing from marshaled output")
		t.Logf("Output:\n%s", xmlStr)
	}
}