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
	styleDefinitions     map[string]styleDefinition
}

type styleDefinition struct {
	styleType     string
	basedOn       string
	runProperties *RunProperties
}

func newStyleContext(docxReader *DocxReader) (*styleContext, error) {
	ctx := &styleContext{
		existingStyles:       make(map[string]styleSignature),
		fragmentStyleMaps:    make(map[string]map[string]string),
		fragmentStylesXML:    make(map[string][]byte),
		fragmentNumberingXML: make(map[string][]byte),
		styleDefinitions:     make(map[string]styleDefinition),
	}

	if docxReader == nil {
		return ctx, nil
	}

	stylesXML, err := docxReader.GetPart("word/styles.xml")
	if err != nil {
		return ctx, nil
	}

	if err := ctx.registerStyles(stylesXML); err != nil {
		return nil, err
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

	if err := ctx.registerStyleDefinitions(remappedStylesXML); err != nil {
		return nil, nil, nil, err
	}

	ctx.fragmentStyleMaps[fragmentName] = styleMap
	ctx.fragmentStylesXML[fragmentName] = remappedStylesXML
	ctx.fragmentNumberingXML[fragmentName] = remappedNumberingXML

	return styleMap, remappedStylesXML, remappedNumberingXML, nil
}

func (ctx *styleContext) registerStyles(stylesXML []byte) error {
	styles, err := parseStyles(stylesXML)
	if err != nil {
		return err
	}

	for _, style := range styles.Styles {
		ctx.existingStyles[style.StyleID] = styleSignature{
			styleType: style.Type,
			rawXML:    normalizeStyleRawXML(style.RawXML),
		}
		ctx.styleDefinitions[style.StyleID] = buildStyleDefinition(style)
	}

	return nil
}

func (ctx *styleContext) registerStyleDefinitions(stylesXML []byte) error {
	if len(stylesXML) == 0 {
		return nil
	}

	styles, err := parseStyles(stylesXML)
	if err != nil {
		return err
	}

	for _, style := range styles.Styles {
		ctx.styleDefinitions[style.StyleID] = buildStyleDefinition(style)
	}

	return nil
}

func buildStyleDefinition(style DocumentStyle) styleDefinition {
	def := styleDefinition{
		styleType: style.Type,
	}
	if style.BasedOn != nil {
		def.basedOn = style.BasedOn.Val
	}
	def.runProperties = mergedStyleRunProperties(style)
	return def
}

func mergedStyleRunProperties(style DocumentStyle) *RunProperties {
	var merged *RunProperties
	if style.ParagraphProperties != nil && style.ParagraphProperties.RunProperties != nil {
		merged = cloneRunProperties(style.ParagraphProperties.RunProperties)
	}
	if style.RunProperties != nil {
		merged = mergeRunProperties(merged, style.RunProperties)
	}
	return merged
}

func (ctx *styleContext) withRenderingStyles(stylesXML []byte) (*styleContext, error) {
	if ctx == nil || len(stylesXML) == 0 {
		return ctx, nil
	}

	child := &styleContext{
		existingStyles:       ctx.existingStyles,
		fragmentStyleMaps:    ctx.fragmentStyleMaps,
		fragmentStylesXML:    ctx.fragmentStylesXML,
		fragmentNumberingXML: ctx.fragmentNumberingXML,
		styleDefinitions:     cloneStyleDefinitions(ctx.styleDefinitions),
	}
	if err := child.registerStyleDefinitions(stylesXML); err != nil {
		return nil, err
	}
	return child, nil
}

func cloneStyleDefinitions(src map[string]styleDefinition) map[string]styleDefinition {
	if len(src) == 0 {
		return make(map[string]styleDefinition)
	}

	cloned := make(map[string]styleDefinition, len(src))
	for id, def := range src {
		cloned[id] = styleDefinition{
			styleType:     def.styleType,
			basedOn:       def.basedOn,
			runProperties: cloneRunProperties(def.runProperties),
		}
	}
	return cloned
}

func (ctx *styleContext) materializeInheritedFonts(para *Paragraph) {
	if ctx == nil || para == nil {
		return
	}

	if len(para.Content) > 0 {
		for _, content := range para.Content {
			switch c := content.(type) {
			case *Run:
				ctx.materializeInheritedFontForRun(para, c)
			case *Hyperlink:
				for i := range c.Runs {
					ctx.materializeInheritedFontForRun(para, &c.Runs[i])
				}
			}
		}
	}

	for i := range para.Runs {
		ctx.materializeInheritedFontForRun(para, &para.Runs[i])
	}
	for i := range para.Hyperlinks {
		for j := range para.Hyperlinks[i].Runs {
			ctx.materializeInheritedFontForRun(para, &para.Hyperlinks[i].Runs[j])
		}
	}
}

func (ctx *styleContext) materializeInheritedFontForRun(para *Paragraph, run *Run) {
	if run == nil {
		return
	}
	if run.Properties != nil && run.Properties.Font != nil {
		return
	}

	font := ctx.resolveEffectiveRunFont(para, run)
	if font == nil {
		return
	}

	props := cloneRunProperties(run.Properties)
	if props == nil {
		props = &RunProperties{}
	}
	props.Font = cloneFont(font)
	run.Properties = props
}

func (ctx *styleContext) resolveEffectiveRunFont(para *Paragraph, run *Run) *Font {
	effective := ctx.resolveParagraphRunProperties(para)
	if run != nil && run.Properties != nil && run.Properties.Style != nil {
		effective = mergeRunProperties(effective, ctx.resolveStyleRunProperties(run.Properties.Style.Val, make(map[string]bool)))
	}
	if run != nil && run.Properties != nil {
		effective = mergeRunProperties(effective, run.Properties)
	}
	if effective == nil || effective.Font == nil {
		return nil
	}
	return cloneFont(effective.Font)
}

func (ctx *styleContext) resolveParagraphRunProperties(para *Paragraph) *RunProperties {
	if para == nil {
		return nil
	}

	var effective *RunProperties
	if para.Properties != nil && para.Properties.Style != nil {
		effective = ctx.resolveStyleRunProperties(para.Properties.Style.Val, make(map[string]bool))
	}
	if para.Properties != nil && para.Properties.RunProperties != nil {
		effective = mergeRunProperties(effective, para.Properties.RunProperties)
	}
	return effective
}

func (ctx *styleContext) resolveStyleRunProperties(styleID string, seen map[string]bool) *RunProperties {
	if ctx == nil || styleID == "" {
		return nil
	}
	if seen[styleID] {
		return nil
	}

	def, ok := ctx.styleDefinitions[styleID]
	if !ok {
		return nil
	}

	seen[styleID] = true
	defer delete(seen, styleID)

	effective := ctx.resolveStyleRunProperties(def.basedOn, seen)
	return mergeRunProperties(effective, def.runProperties)
}

func cloneRunProperties(src *RunProperties) *RunProperties {
	if src == nil {
		return nil
	}

	cloned := *src
	if src.Font != nil {
		cloned.Font = cloneFont(src.Font)
	}
	if src.Color != nil {
		colorCopy := *src.Color
		cloned.Color = &colorCopy
	}
	if src.Size != nil {
		sizeCopy := *src.Size
		cloned.Size = &sizeCopy
	}
	if src.SizeCs != nil {
		sizeCsCopy := *src.SizeCs
		cloned.SizeCs = &sizeCsCopy
	}
	if src.Kern != nil {
		kernCopy := *src.Kern
		cloned.Kern = &kernCopy
	}
	if src.Lang != nil {
		langCopy := *src.Lang
		cloned.Lang = &langCopy
	}
	if src.Style != nil {
		styleCopy := *src.Style
		cloned.Style = &styleCopy
	}
	if src.Underline != nil {
		underlineCopy := *src.Underline
		cloned.Underline = &underlineCopy
	}
	if src.VerticalAlign != nil {
		verticalAlignCopy := *src.VerticalAlign
		cloned.VerticalAlign = &verticalAlignCopy
	}
	return &cloned
}

func mergeRunProperties(base, override *RunProperties) *RunProperties {
	if override == nil {
		return cloneRunProperties(base)
	}
	if base == nil {
		return cloneRunProperties(override)
	}

	merged := cloneRunProperties(base)
	if override.Bold != nil {
		merged.Bold = override.Bold
	}
	if override.BoldCs != nil {
		merged.BoldCs = override.BoldCs
	}
	if override.Italic != nil {
		merged.Italic = override.Italic
	}
	if override.ItalicCs != nil {
		merged.ItalicCs = override.ItalicCs
	}
	if override.Underline != nil {
		underlineCopy := *override.Underline
		merged.Underline = &underlineCopy
	}
	if override.Strike != nil {
		merged.Strike = override.Strike
	}
	if override.VerticalAlign != nil {
		verticalAlignCopy := *override.VerticalAlign
		merged.VerticalAlign = &verticalAlignCopy
	}
	if override.Color != nil {
		colorCopy := *override.Color
		merged.Color = &colorCopy
	}
	if override.Size != nil {
		sizeCopy := *override.Size
		merged.Size = &sizeCopy
	}
	if override.SizeCs != nil {
		sizeCsCopy := *override.SizeCs
		merged.SizeCs = &sizeCsCopy
	}
	if override.Kern != nil {
		kernCopy := *override.Kern
		merged.Kern = &kernCopy
	}
	if override.Lang != nil {
		langCopy := *override.Lang
		merged.Lang = &langCopy
	}
	if override.Font != nil {
		merged.Font = cloneFont(override.Font)
	}
	if override.Style != nil {
		styleCopy := *override.Style
		merged.Style = &styleCopy
	}
	return merged
}

func cloneFont(src *Font) *Font {
	if src == nil {
		return nil
	}
	fontCopy := *src
	return &fontCopy
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
