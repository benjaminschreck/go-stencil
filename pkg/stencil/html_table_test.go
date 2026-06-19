package stencil

import (
	"strings"
	"testing"
)

func TestHTMLFunctionTableCall(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")

	result, err := htmlFn.Call(`<table><tr><th>Name</th><th>Amount</th></tr><tr><td><b>Fee</b></td><td>12.50</td></tr></table>`)
	if err != nil {
		t.Fatalf("html() returned error: %v", err)
	}

	fragment, ok := result.(*OOXMLFragment)
	if !ok {
		t.Fatalf("html() returned %T, want *OOXMLFragment", result)
	}

	htmlBody, ok := fragment.Content.(*HTMLBody)
	if !ok {
		t.Fatalf("html() fragment content = %T, want *HTMLBody", fragment.Content)
	}
	if len(htmlBody.Elements) != 1 {
		t.Fatalf("html() body elements = %d, want 1", len(htmlBody.Elements))
	}
	htmlTable, ok := htmlBody.Elements[0].(*Table)
	if !ok {
		t.Fatalf("html() body element = %T, want *Table", htmlBody.Elements[0])
	}
	if len(htmlTable.Rows) != 2 {
		t.Fatalf("table rows = %d, want 2", len(htmlTable.Rows))
	}
	if got := strings.TrimSpace(htmlTable.Rows[1].Cells[0].GetText()); got != "Fee" {
		t.Fatalf("cell text = %q, want Fee", got)
	}
	if props := htmlTable.Rows[1].Cells[0].Paragraphs[0].Runs[0].Properties; props == nil || props.Bold == nil {
		t.Fatal("expected bold formatting in first body cell")
	}
}

func TestHTMLFunctionTableColspan(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")

	result, err := htmlFn.Call(`<table><tr><td colspan="2">Header</td></tr><tr><td>A</td><td>B</td></tr></table>`)
	if err != nil {
		t.Fatalf("html() returned error: %v", err)
	}

	fragment := result.(*OOXMLFragment)
	htmlBody := fragment.Content.(*HTMLBody)
	htmlTable := htmlBody.Elements[0].(*Table)
	firstCell := htmlTable.Rows[0].Cells[0]
	if firstCell.Properties == nil || firstCell.Properties.GridSpan == nil {
		t.Fatal("expected first cell to have grid span")
	}
	if firstCell.Properties.GridSpan.Val != 2 {
		t.Fatalf("grid span = %d, want 2", firstCell.Properties.GridSpan.Val)
	}
	if len(htmlTable.Grid.Columns) != 2 {
		t.Fatalf("grid columns = %d, want 2", len(htmlTable.Grid.Columns))
	}
}

func TestRenderBodyReplacesStandaloneHTMLTableFragment(t *testing.T) {
	body := &Body{
		Elements: []BodyElement{
			&Paragraph{
				Runs: []Run{{Text: &Text{Content: "{{html(tableHTML)}}"}}},
			},
		},
	}

	ctx := &renderContext{
		linkMarkers:    make(map[string]*LinkReplacementMarker),
		fragments:      make(map[string]*fragment),
		ooxmlFragments: make(map[string]interface{}),
	}

	rendered, err := RenderBodyWithControlStructures(body, TemplateData{
		"tableHTML": `<table><tr><td>One</td><td><i>Two</i></td></tr></table>`,
	}, ctx)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() error = %v", err)
	}

	if len(rendered.Elements) != 1 {
		t.Fatalf("rendered elements = %d, want 1", len(rendered.Elements))
	}
	table, ok := rendered.Elements[0].(*Table)
	if !ok {
		t.Fatalf("rendered element = %T, want *Table", rendered.Elements[0])
	}
	if got := table.Rows[0].Cells[0].GetText(); got != "One" {
		t.Fatalf("first cell text = %q, want One", got)
	}
	if got := table.Rows[0].Cells[1].GetText(); got != "Two" {
		t.Fatalf("second cell text = %q, want Two", got)
	}
	if props := table.Rows[0].Cells[1].Paragraphs[0].Runs[0].Properties; props == nil || props.Italic == nil {
		t.Fatal("expected italic formatting in second cell")
	}
}

func TestRenderBodyReplacesStandaloneMixedHTMLFragment(t *testing.T) {
	body := &Body{
		Elements: []BodyElement{
			&Paragraph{
				Runs: []Run{{Text: &Text{Content: "{{html(customHTML)}}"}}},
			},
		},
	}

	ctx := &renderContext{
		linkMarkers:    make(map[string]*LinkReplacementMarker),
		fragments:      make(map[string]*fragment),
		ooxmlFragments: make(map[string]interface{}),
	}

	rendered, err := RenderBodyWithControlStructures(body, TemplateData{
		"customHTML": `<p><b>Anschreiben</b><br/>Bitte beachten:</p><table><tr><th>Position</th><th>Betrag</th></tr><tr><td>Gebuehr</td><td>10,00 EUR</td></tr></table><p>Mit freundlichen Gruessen</p>`,
	}, ctx)
	if err != nil {
		t.Fatalf("RenderBodyWithControlStructures() error = %v", err)
	}

	if len(rendered.Elements) != 3 {
		t.Fatalf("rendered elements = %d, want 3", len(rendered.Elements))
	}
	firstPara, ok := rendered.Elements[0].(*Paragraph)
	if !ok {
		t.Fatalf("first rendered element = %T, want *Paragraph", rendered.Elements[0])
	}
	if got := firstPara.GetText(); got != "AnschreibenBitte beachten:" {
		t.Fatalf("first paragraph text = %q, want combined text", got)
	}
	if len(firstPara.Runs) < 3 || firstPara.Runs[1].Break == nil {
		t.Fatal("expected line break run between first paragraph text runs")
	}
	if props := firstPara.Runs[0].Properties; props == nil || props.Bold == nil {
		t.Fatal("expected bold formatting in first paragraph")
	}

	table, ok := rendered.Elements[1].(*Table)
	if !ok {
		t.Fatalf("second rendered element = %T, want *Table", rendered.Elements[1])
	}
	if got := table.Rows[1].Cells[1].GetText(); got != "10,00 EUR" {
		t.Fatalf("table amount cell = %q, want 10,00 EUR", got)
	}

	lastPara, ok := rendered.Elements[2].(*Paragraph)
	if !ok {
		t.Fatalf("third rendered element = %T, want *Paragraph", rendered.Elements[2])
	}
	if got := lastPara.GetText(); got != "Mit freundlichen Gruessen" {
		t.Fatalf("last paragraph text = %q, want greeting", got)
	}
}
