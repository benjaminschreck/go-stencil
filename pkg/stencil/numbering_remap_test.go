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

func TestSanitizeAbstractNumberingMetadataRemovesWordIdentityFields(t *testing.T) {
	block := `<w:abstractNum w:abstractNumId="7"><w:nsid w:val="3A120F1C"/><w:multiLevelType w:val="hybridMultilevel"/><w:tmpl w:val="9BA0E964"/><w:lvl w:ilvl="0"><w:numFmt w:val="bullet"/></w:lvl></w:abstractNum>`

	got := sanitizeAbstractNumberingMetadata(block)

	if strings.Contains(got, `<w:nsid`) {
		t.Fatalf("expected sanitized abstract numbering block to remove w:nsid, got %s", got)
	}
	if strings.Contains(got, `<w:tmpl`) {
		t.Fatalf("expected sanitized abstract numbering block to remove w:tmpl, got %s", got)
	}
	if !strings.Contains(got, `<w:multiLevelType`) || !strings.Contains(got, `<w:numFmt w:val="bullet"`) {
		t.Fatalf("expected sanitized abstract numbering block to keep list definition, got %s", got)
	}
}

func TestSanitizeNumberingInstanceMetadataRemovesDurableID(t *testing.T) {
	block := `<w:num w:numId="12" w16cid:durableId="591595714"><w:abstractNumId w:val="7"/></w:num>`

	got := sanitizeNumberingInstanceMetadata(block)

	if strings.Contains(got, `w16cid:durableId=`) {
		t.Fatalf("expected sanitized numbering instance block to remove durableId, got %s", got)
	}
	if !strings.Contains(got, `<w:abstractNumId w:val="7"/>`) {
		t.Fatalf("expected sanitized numbering instance block to keep abstractNum reference, got %s", got)
	}
}
