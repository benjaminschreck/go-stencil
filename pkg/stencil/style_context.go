package stencil

import (
	"fmt"
	"regexp"
	"strings"
)

type styleSignature struct {
	styleType string
	rawXML    string
}

type styleContext struct {
	existingStyles       map[string]styleSignature
	fragmentStyleMaps    map[string]map[string]string
	fragmentStylesXML    map[string][]byte
	fragmentNumberingXML map[string][]byte
}

func newStyleContext(docxReader *DocxReader) (*styleContext, error) {
	ctx := &styleContext{
		existingStyles:       make(map[string]styleSignature),
		fragmentStyleMaps:    make(map[string]map[string]string),
		fragmentStylesXML:    make(map[string][]byte),
		fragmentNumberingXML: make(map[string][]byte),
	}

	if docxReader == nil {
		return ctx, nil
	}

	stylesXML, err := docxReader.GetPart("word/styles.xml")
	if err != nil {
		return ctx, nil
	}

	styles, err := parseStyles(stylesXML)
	if err != nil {
		return nil, err
	}

	for _, style := range styles.Styles {
		ctx.existingStyles[style.StyleID] = styleSignature{
			styleType: style.Type,
			rawXML:    normalizeStyleRawXML(style.RawXML),
		}
	}

	return ctx, nil
}

func (ctx *styleContext) ensureFragmentStyles(fragmentName string, stylesXML, numberingXML []byte) (map[string]string, []byte, []byte, error) {
	if len(stylesXML) == 0 {
		return nil, stylesXML, numberingXML, nil
	}
	if existingMap, ok := ctx.fragmentStyleMaps[fragmentName]; ok {
		return existingMap, ctx.fragmentStylesXML[fragmentName], ctx.fragmentNumberingXML[fragmentName], nil
	}

	styles, err := parseStyles(stylesXML)
	if err != nil {
		return nil, nil, nil, err
	}

	styleMap := make(map[string]string)
	for _, style := range styles.Styles {
		sig := styleSignature{
			styleType: style.Type,
			rawXML:    normalizeStyleRawXML(style.RawXML),
		}

		existingSig, exists := ctx.existingStyles[style.StyleID]
		if !exists {
			ctx.existingStyles[style.StyleID] = sig
			continue
		}
		if existingSig == sig {
			continue
		}

		newID := ctx.generateStyleID(style.StyleID, fragmentName)
		styleMap[style.StyleID] = newID
		ctx.existingStyles[newID] = sig
	}

	remappedStylesXML := stylesXML
	remappedNumberingXML := numberingXML
	if len(styleMap) > 0 {
		remappedStylesXML = remapStyleIDsInStyles(stylesXML, styleMap)
		remappedNumberingXML = remapStyleIDsInNumbering(numberingXML, styleMap)
	}

	ctx.fragmentStyleMaps[fragmentName] = styleMap
	ctx.fragmentStylesXML[fragmentName] = remappedStylesXML
	ctx.fragmentNumberingXML[fragmentName] = remappedNumberingXML

	return styleMap, remappedStylesXML, remappedNumberingXML, nil
}

func (ctx *styleContext) generateStyleID(baseID, fragmentName string) string {
	base := sanitizeStyleID(baseID + "__" + fragmentName)
	if _, exists := ctx.existingStyles[base]; !exists {
		return base
	}

	for idx := 2; ; idx++ {
		candidate := fmt.Sprintf("%s_%d", base, idx)
		if _, exists := ctx.existingStyles[candidate]; !exists {
			return candidate
		}
	}
}

func sanitizeStyleID(value string) string {
	if value == "" {
		return "FragmentStyle"
	}

	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	return b.String()
}

func normalizeStyleRawXML(raw []byte) string {
	return strings.Join(strings.Fields(string(raw)), " ")
}

func remapStyleIDsInStyles(stylesXML []byte, styleMap map[string]string) []byte {
	if len(stylesXML) == 0 || len(styleMap) == 0 {
		return stylesXML
	}

	remapped := string(stylesXML)
	for oldID, newID := range styleMap {
		remapped = replaceStyleAttribute(remapped, "styleId", oldID, newID)
		for _, tag := range []string{
			"basedOn",
			"next",
			"link",
			"pStyle",
			"rStyle",
			"tblStyle",
			"numStyleLink",
			"styleLink",
		} {
			remapped = replaceTaggedStyleValue(remapped, tag, oldID, newID)
		}
	}

	return []byte(remapped)
}

func remapStyleIDsInNumbering(numberingXML []byte, styleMap map[string]string) []byte {
	if len(numberingXML) == 0 || len(styleMap) == 0 {
		return numberingXML
	}

	remapped := string(numberingXML)
	for oldID, newID := range styleMap {
		for _, tag := range []string{"pStyle", "numStyleLink", "styleLink"} {
			remapped = replaceTaggedStyleValue(remapped, tag, oldID, newID)
		}
	}

	return []byte(remapped)
}

func replaceStyleAttribute(xmlContent, attrName, oldID, newID string) string {
	re := regexp.MustCompile(`(\bw:` + regexp.QuoteMeta(attrName) + `=")` + regexp.QuoteMeta(oldID) + `(")`)
	return re.ReplaceAllString(xmlContent, `${1}`+newID+`${2}`)
}

func replaceTaggedStyleValue(xmlContent, tagName, oldID, newID string) string {
	re := regexp.MustCompile(`(<w:` + regexp.QuoteMeta(tagName) + `\b[^>]*\bw:val=")` + regexp.QuoteMeta(oldID) + `(")`)
	return re.ReplaceAllString(xmlContent, `${1}`+newID+`${2}`)
}
