package stencil

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
)

type compiledFragmentMetadata struct {
	numbering *compiledFragmentNumbering
}

type compiledFragmentNumbering struct {
	rootAttributes []numberingAttribute
	abstracts      []compiledAbstractNumbering
	nums           []compiledInstanceNumbering
	stylesTemplate *compiledStylesNumberingTemplate
}

type numberingAttribute struct {
	name  string
	value string
}

type compiledAbstractNumbering struct {
	oldID    string
	template string
}

type compiledInstanceNumbering struct {
	oldID         string
	oldAbstractID string
	template      string
}

type compiledStylesNumberingTemplate struct {
	content      string
	placeholders map[string]string
}

func compileFragmentMetadata(frag *fragment) error {
	if frag == nil {
		return nil
	}

	meta := &compiledFragmentMetadata{}
	if len(frag.numberingXML) > 0 {
		compiled, err := compileFragmentNumbering(frag.numberingXML, frag.stylesXML)
		if err != nil {
			return err
		}
		meta.numbering = compiled
	}

	frag.compiled = meta
	return nil
}

func compileFragmentNumbering(numberingXML, stylesXML []byte) (*compiledFragmentNumbering, error) {
	fragmentXML := string(numberingXML)
	result := &compiledFragmentNumbering{
		rootAttributes: extractCompiledNumberingAttributes(fragmentXML),
	}

	abstractBlocks := numberingAbstractBlockRegex.FindAllString(fragmentXML, -1)
	numBlocks := numberingNumBlockRegex.FindAllString(fragmentXML, -1)

	if len(abstractBlocks) == 0 && len(numBlocks) == 0 {
		return result, nil
	}

	oldNumIDs := make([]string, 0, len(numBlocks))
	for _, block := range abstractBlocks {
		oldID, ok := extractNumberingMatch(block, numberingAbstractIDRegex)
		if !ok {
			return nil, fmt.Errorf("fragment numbering has abstractNum without w:abstractNumId")
		}

		template := numberingAbstractIDRegex.ReplaceAllString(block, `w:abstractNumId="__GO_STENCIL_ABSTRACT_ID__"`)
		template = sanitizeAbstractNumberingMetadata(template)

		result.abstracts = append(result.abstracts, compiledAbstractNumbering{
			oldID:    oldID,
			template: template,
		})
	}

	for _, block := range numBlocks {
		oldNumID, ok := extractNumberingMatch(block, numberingNumIDRegex)
		if !ok {
			return nil, fmt.Errorf("fragment numbering has num without w:numId")
		}

		template := numberingNumIDRegex.ReplaceAllString(block, `<w:num w:numId="__GO_STENCIL_NUM_ID__"`)
		template = sanitizeNumberingInstanceMetadata(template)
		oldAbstractID, _ := extractNumberingMatch(template, numberingNumRefRegex)
		if oldAbstractID != "" {
			template = numberingNumRefRegex.ReplaceAllString(template, `<w:abstractNumId w:val="__GO_STENCIL_ABSTRACT_REF__"/>`)
		}

		result.nums = append(result.nums, compiledInstanceNumbering{
			oldID:         oldNumID,
			oldAbstractID: oldAbstractID,
			template:      template,
		})
		oldNumIDs = append(oldNumIDs, oldNumID)
	}

	if len(stylesXML) > 0 && len(oldNumIDs) > 0 {
		result.stylesTemplate = compileStylesNumberingTemplate(stylesXML, oldNumIDs)
	}

	return result, nil
}

func extractCompiledNumberingAttributes(fragmentXML string) []numberingAttribute {
	fragmentTag := numberingStartTagRegex.FindString(fragmentXML)
	if fragmentTag == "" {
		return nil
	}

	attrs := make([]numberingAttribute, 0, 8)
	for _, match := range numberingAttrRegex.FindAllStringSubmatch(fragmentTag, -1) {
		if len(match) < 3 {
			continue
		}
		attrs = append(attrs, numberingAttribute{name: match[1], value: match[2]})
	}
	return attrs
}

func compileStylesNumberingTemplate(stylesXML []byte, oldNumIDs []string) *compiledStylesNumberingTemplate {
	keys := append([]string(nil), oldNumIDs...)
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) == len(keys[j]) {
			return keys[i] < keys[j]
		}
		return len(keys[i]) > len(keys[j])
	})

	content := string(stylesXML)
	placeholders := make(map[string]string, len(keys))
	for idx, oldID := range keys {
		placeholder := fmt.Sprintf("__GO_STENCIL_STYLE_NUM_%d__", idx)
		content = strings.ReplaceAll(content, `w:numId w:val="`+oldID+`"`, placeholder)
		placeholders[oldID] = placeholder
	}

	return &compiledStylesNumberingTemplate{
		content:      content,
		placeholders: placeholders,
	}
}

func (tmpl *compiledStylesNumberingTemplate) render(numMap map[string]string) []byte {
	if tmpl == nil {
		return nil
	}

	rendered := tmpl.content
	keys := make([]string, 0, len(numMap))
	for oldID := range numMap {
		keys = append(keys, oldID)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) == len(keys[j]) {
			return keys[i] < keys[j]
		}
		return len(keys[i]) > len(keys[j])
	})
	for _, oldID := range keys {
		placeholder, ok := tmpl.placeholders[oldID]
		if !ok {
			continue
		}
		rendered = strings.ReplaceAll(rendered, placeholder, `w:numId w:val="`+numMap[oldID]+`"`)
	}
	return []byte(rendered)
}

func marshalXMLAttrs(attrs []xml.Attr) []numberingAttribute {
	compiled := make([]numberingAttribute, 0, len(attrs))
	for _, attr := range attrs {
		name := attr.Name.Local
		if attr.Name.Space != "" {
			name = namespaceURIToPrefix(attr.Name.Space) + ":" + attr.Name.Local
		}
		compiled = append(compiled, numberingAttribute{name: name, value: attr.Value})
	}
	return compiled
}
