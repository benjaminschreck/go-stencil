package stencil

import (
	"strings"
	"testing"
)

func TestUpdateRawXMLNumberingIDsDoesNotCascadeIntoNewIDs(t *testing.T) {
	raw := &RawXMLElement{
		Content: []byte(`<w:numPr><w:ilvl w:val="0"/><w:numId w:val="5"/></w:numPr>`),
	}

	updateRawXMLNumberingIDs(raw, map[string]string{
		"5":  "27",
		"27": "49",
	})

	got := string(raw.Content)
	if !strings.Contains(got, `w:numId w:val="27"`) {
		t.Fatalf("expected remapped raw XML to contain numId=27, got %s", got)
	}
	if strings.Contains(got, `w:numId w:val="49"`) {
		t.Fatalf("expected remapped raw XML not to cascade into numId=49, got %s", got)
	}
}

func TestRemapStylesNumberingIDsDoesNotCascadeIntoNewIDs(t *testing.T) {
	stylesXML := []byte(`<w:style><w:pPr><w:numPr><w:numId w:val="5"/></w:numPr></w:pPr></w:style>`)

	got := string(remapStylesNumberingIDs(stylesXML, map[string]string{
		"5":  "27",
		"27": "49",
	}))

	if !strings.Contains(got, `w:numId w:val="27"`) {
		t.Fatalf("expected remapped styles XML to contain numId=27, got %s", got)
	}
	if strings.Contains(got, `w:numId w:val="49"`) {
		t.Fatalf("expected remapped styles XML not to cascade into numId=49, got %s", got)
	}
}
