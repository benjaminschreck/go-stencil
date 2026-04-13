package stencil

import "encoding/xml"

type fragmentFontOverrides struct {
	paragraphStyles map[string]Font
	runStyles       map[string]Font
}

type styleFontCatalog struct {
	styles map[string]styleFontDefinition
}

type styleFontDefinition struct {
	Type    string
	BasedOn string
	Font    *Font
}

type styleFontStylesXML struct {
	Styles []styleFontStyleXML `xml:"style"`
}

type styleFontStyleXML struct {
	StyleID  string            `xml:"styleId,attr"`
	Type     string            `xml:"type,attr"`
	BasedOn  *styleRefXML      `xml:"basedOn"`
	RunProps *styleRunPropsXML `xml:"rPr"`
}

type styleRefXML struct {
	Val string `xml:"val,attr"`
}

type styleRunPropsXML struct {
	Font *Font `xml:"rFonts"`
}

func buildFragmentFontOverride(mainStylesXML, fragmentStylesXML []byte) (fragmentFontOverrides, bool) {
	if len(mainStylesXML) == 0 || len(fragmentStylesXML) == 0 {
		return fragmentFontOverrides{}, false
	}
	mainCatalog, err := parseStyleFontCatalog(mainStylesXML)
	if err != nil {
		return fragmentFontOverrides{}, false
	}
	fragmentCatalog, err := parseStyleFontCatalog(fragmentStylesXML)
	if err != nil {
		return fragmentFontOverrides{}, false
	}

	conflict := fragmentFontOverrides{
		paragraphStyles: make(map[string]Font),
		runStyles:       make(map[string]Font),
	}

	for styleID, fragmentStyle := range fragmentCatalog.styles {
		mainStyle, exists := mainCatalog.styles[styleID]
		if !exists || mainStyle.Type != fragmentStyle.Type {
			continue
		}

		fragmentFont := fragmentCatalog.effectiveFont(styleID)
		if fragmentFont == nil {
			continue
		}

		mainFont := mainCatalog.effectiveFont(styleID)
		if fontsEqual(mainFont, fragmentFont) {
			continue
		}

		switch fragmentStyle.Type {
		case "paragraph":
			conflict.paragraphStyles[styleID] = *cloneFont(fragmentFont)
		case "character":
			conflict.runStyles[styleID] = *cloneFont(fragmentFont)
		}
	}

	if len(conflict.paragraphStyles) == 0 && len(conflict.runStyles) == 0 {
		return fragmentFontOverrides{}, false
	}
	return conflict, true
}

func parseStyleFontCatalog(stylesXML []byte) (*styleFontCatalog, error) {
	var parsed styleFontStylesXML
	if err := xml.Unmarshal(stylesXML, &parsed); err != nil {
		return nil, err
	}

	catalog := &styleFontCatalog{styles: make(map[string]styleFontDefinition)}
	for _, style := range parsed.Styles {
		if style.StyleID == "" {
			continue
		}
		def := styleFontDefinition{
			Type: style.Type,
		}
		if style.BasedOn != nil {
			def.BasedOn = style.BasedOn.Val
		}
		if style.RunProps != nil && style.RunProps.Font != nil {
			def.Font = cloneFont(style.RunProps.Font)
		}
		catalog.styles[style.StyleID] = def
	}

	return catalog, nil
}

func (c *styleFontCatalog) effectiveFont(styleID string) *Font {
	if c == nil || styleID == "" {
		return nil
	}
	return c.resolveEffectiveFont(styleID, make(map[string]bool))
}

func (c *styleFontCatalog) resolveEffectiveFont(styleID string, seen map[string]bool) *Font {
	if seen[styleID] {
		return nil
	}
	seen[styleID] = true

	style, ok := c.styles[styleID]
	if !ok {
		return nil
	}

	var base *Font
	if style.BasedOn != "" {
		base = c.resolveEffectiveFont(style.BasedOn, seen)
	}
	return mergeFonts(base, style.Font)
}

func mergeFonts(base, override *Font) *Font {
	if base == nil && override == nil {
		return nil
	}

	merged := Font{}
	if base != nil {
		merged = *cloneFont(base)
	}
	if override == nil {
		return &merged
	}

	if override.ASCII != "" {
		merged.ASCII = override.ASCII
	}
	if override.HAnsi != "" {
		merged.HAnsi = override.HAnsi
	}
	if override.CS != "" {
		merged.CS = override.CS
	}
	if override.EastAsia != "" {
		merged.EastAsia = override.EastAsia
	}
	if override.ASCIITheme != "" {
		merged.ASCIITheme = override.ASCIITheme
	}
	if override.HAnsiTheme != "" {
		merged.HAnsiTheme = override.HAnsiTheme
	}
	if override.CSTheme != "" {
		merged.CSTheme = override.CSTheme
	}
	if override.EastAsiaTheme != "" {
		merged.EastAsiaTheme = override.EastAsiaTheme
	}
	if override.Hint != "" {
		merged.Hint = override.Hint
	}

	return &merged
}

func cloneFont(font *Font) *Font {
	if font == nil {
		return nil
	}
	cloned := *font
	return &cloned
}

func fontsEqual(a, b *Font) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return a.ASCII == b.ASCII &&
		a.HAnsi == b.HAnsi &&
		a.CS == b.CS &&
		a.EastAsia == b.EastAsia &&
		a.ASCIITheme == b.ASCIITheme &&
		a.HAnsiTheme == b.HAnsiTheme &&
		a.CSTheme == b.CSTheme &&
		a.EastAsiaTheme == b.EastAsiaTheme &&
		a.Hint == b.Hint
}

func applyFragmentFontOverrides(elements []BodyElement, fragmentName string, ctx *renderContext) {
	if ctx == nil || len(elements) == 0 {
		return
	}

	override, ok := ctx.fragmentFontOverrides[fragmentName]
	if !ok {
		return
	}

	for _, element := range elements {
		applyElementFontOverrides(element, override)
	}
}

func applyElementFontOverrides(element BodyElement, override fragmentFontOverrides) {
	switch el := element.(type) {
	case *Paragraph:
		applyParagraphFontOverrides(el, override)
	case *Table:
		for rowIdx := range el.Rows {
			for cellIdx := range el.Rows[rowIdx].Cells {
				for paraIdx := range el.Rows[rowIdx].Cells[cellIdx].Paragraphs {
					applyParagraphFontOverrides(&el.Rows[rowIdx].Cells[cellIdx].Paragraphs[paraIdx], override)
				}
			}
		}
	}
}

func applyParagraphFontOverrides(para *Paragraph, override fragmentFontOverrides) {
	if para == nil {
		return
	}

	var paragraphFont *Font
	if para.Properties != nil && para.Properties.Style != nil {
		if font, ok := override.paragraphStyles[para.Properties.Style.Val]; ok {
			paragraphFont = cloneFont(&font)
			if para.Properties.RunProperties == nil {
				para.Properties.RunProperties = &RunProperties{}
			}
			if para.Properties.RunProperties.Font == nil {
				para.Properties.RunProperties.Font = cloneFont(paragraphFont)
			}
		}
	}

	for idx := range para.Runs {
		applyRunFontOverrides(&para.Runs[idx], override, paragraphFont)
	}
	for idx := range para.Hyperlinks {
		for runIdx := range para.Hyperlinks[idx].Runs {
			applyRunFontOverrides(&para.Hyperlinks[idx].Runs[runIdx], override, paragraphFont)
		}
	}
	for _, content := range para.Content {
		switch item := content.(type) {
		case *Run:
			applyRunFontOverrides(item, override, paragraphFont)
		case *Hyperlink:
			for runIdx := range item.Runs {
				applyRunFontOverrides(&item.Runs[runIdx], override, paragraphFont)
			}
		}
	}
}

func applyRunFontOverrides(run *Run, override fragmentFontOverrides, paragraphFont *Font) {
	if run == nil {
		return
	}

	if paragraphFont != nil {
		ensureRunFont(run, paragraphFont)
	}

	if run.Properties != nil && run.Properties.Style != nil {
		if font, ok := override.runStyles[run.Properties.Style.Val]; ok {
			ensureRunFont(run, &font)
		}
	}
}

func ensureRunFont(run *Run, font *Font) {
	if run == nil || font == nil {
		return
	}
	if run.Properties == nil {
		run.Properties = &RunProperties{}
	}
	if run.Properties.Font != nil {
		return
	}
	run.Properties.Font = cloneFont(font)
}
