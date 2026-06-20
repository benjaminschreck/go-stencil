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

func TestHTMLFunctionTableProperties(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")

	result, err := htmlFn.Call(`<table style="width:100%; border:2px dashed #336699;"><tr><th style="width:120px; background-color:#eeeeee;">Name</th><th style="text-align:right; width:80px;">Amount</th></tr><tr><td style="background-color:#ffeecc;">Fee</td><td style="text-align:right;">12.50</td></tr></table>`)
	if err != nil {
		t.Fatalf("html() returned error: %v", err)
	}

	fragment := result.(*OOXMLFragment)
	htmlBody := fragment.Content.(*HTMLBody)
	table := htmlBody.Elements[0].(*Table)

	if table.Properties == nil || table.Properties.Width == nil {
		t.Fatal("expected table width")
	}
	if table.Properties.Width.Type != "pct" || table.Properties.Width.Val != 5000 {
		t.Fatalf("table width = %#v, want pct 5000", table.Properties.Width)
	}
	if table.Properties.Borders == nil || table.Properties.Borders.Top == nil {
		t.Fatal("expected table borders")
	}
	if table.Properties.Borders.Top.Val != "dashed" || table.Properties.Borders.Top.Sz != "16" || table.Properties.Borders.Top.Color != "336699" {
		t.Fatalf("top border = %#v, want dashed 16 336699", table.Properties.Borders.Top)
	}
	if len(table.Grid.Columns) != 2 {
		t.Fatalf("grid columns = %d, want 2", len(table.Grid.Columns))
	}
	if table.Grid.Columns[0].Width != 1800 || table.Grid.Columns[1].Width != 1200 {
		t.Fatalf("grid widths = %v, want [1800 1200]", table.Grid.Columns)
	}

	headerName := table.Rows[0].Cells[0]
	if headerName.Properties == nil || headerName.Properties.Shading == nil {
		t.Fatal("expected header cell shading")
	}
	if headerName.Properties.Shading.Fill != "EEEEEE" {
		t.Fatalf("header shading = %q, want EEEEEE", headerName.Properties.Shading.Fill)
	}
	if props := headerName.Paragraphs[0].Runs[0].Properties; props == nil || props.Bold == nil {
		t.Fatal("expected th content to be bold")
	}

	headerAmount := table.Rows[0].Cells[1]
	if headerAmount.Paragraphs[0].Properties == nil || headerAmount.Paragraphs[0].Properties.Alignment == nil {
		t.Fatal("expected header amount alignment")
	}
	if headerAmount.Paragraphs[0].Properties.Alignment.Val != "right" {
		t.Fatalf("header alignment = %q, want right", headerAmount.Paragraphs[0].Properties.Alignment.Val)
	}

	bodyName := table.Rows[1].Cells[0]
	if bodyName.Properties == nil || bodyName.Properties.Shading == nil || bodyName.Properties.Shading.Fill != "FFEECC" {
		t.Fatalf("body cell shading = %#v, want FFEECC", bodyName.Properties)
	}
	bodyAmount := table.Rows[1].Cells[1]
	if bodyAmount.Paragraphs[0].Properties == nil || bodyAmount.Paragraphs[0].Properties.Alignment == nil || bodyAmount.Paragraphs[0].Properties.Alignment.Val != "right" {
		t.Fatalf("body amount paragraph properties = %#v, want right alignment", bodyAmount.Paragraphs[0].Properties)
	}
}

func TestHTMLFunctionTableNoBorders(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	htmlFn, _ := registry.GetFunction("html")

	result, err := htmlFn.Call(`<table style="border:none;"><tr><td style="border:0;">No border</td></tr></table>`)
	if err != nil {
		t.Fatalf("html() returned error: %v", err)
	}

	fragment := result.(*OOXMLFragment)
	table := fragment.Content.(*HTMLBody).Elements[0].(*Table)
	if table.Properties == nil || table.Properties.Borders == nil || table.Properties.Borders.Top == nil {
		t.Fatal("expected nil table border markers")
	}
	if table.Properties.Borders.Top.Val != "nil" {
		t.Fatalf("table top border = %q, want nil", table.Properties.Borders.Top.Val)
	}
	cell := table.Rows[0].Cells[0]
	if cell.Properties == nil || cell.Properties.TcBorders == nil || cell.Properties.TcBorders.Top == nil {
		t.Fatal("expected nil cell border markers")
	}
	if cell.Properties.TcBorders.Top.Val != "nil" {
		t.Fatalf("cell top border = %q, want nil", cell.Properties.TcBorders.Top.Val)
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
