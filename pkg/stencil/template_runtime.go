package stencil

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

func cloneFragmentMap(src map[string]*fragment) map[string]*fragment {
	if len(src) == 0 {
		return make(map[string]*fragment)
	}
	cloned := make(map[string]*fragment, len(src))
	for name, frag := range src {
		cloned[name] = frag
	}
	return cloned
}

func (t *template) snapshotFragments() map[string]*fragment {
	if t == nil {
		return make(map[string]*fragment)
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return cloneFragmentMap(t.fragments)
}

func cloneBodyPlanMap(src map[*Body]*bodyRenderPlan) map[*Body]*bodyRenderPlan {
	if len(src) == 0 {
		return make(map[*Body]*bodyRenderPlan)
	}
	cloned := make(map[*Body]*bodyRenderPlan, len(src))
	for body, plan := range src {
		cloned[body] = plan
	}
	return cloned
}

func cloneParagraphPlanMap(src map[*Paragraph]*paragraphRenderPlan) map[*Paragraph]*paragraphRenderPlan {
	if len(src) == 0 {
		return make(map[*Paragraph]*paragraphRenderPlan)
	}
	cloned := make(map[*Paragraph]*paragraphRenderPlan, len(src))
	for para, plan := range src {
		cloned[para] = plan
	}
	return cloned
}

func cloneFragmentFontOverrideMap(src map[string]fragmentFontOverrides) map[string]fragmentFontOverrides {
	if len(src) == 0 {
		return make(map[string]fragmentFontOverrides)
	}
	cloned := make(map[string]fragmentFontOverrides, len(src))
	for name, override := range src {
		cloned[name] = override
	}
	return cloned
}

func (t *template) resolveFragment(name string) (*fragment, error) {
	t.mu.RLock()
	if frag, ok := t.fragments[name]; ok {
		t.mu.RUnlock()
		return frag, nil
	}
	resolver := t.fragmentResolver
	t.mu.RUnlock()

	if resolver == nil {
		return nil, nil
	}

	content, err := resolver.ResolveFragment(name)
	if err != nil {
		return nil, err
	}
	if content == nil {
		return nil, nil
	}

	frag, err := newResolvedFragment(name, content)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if existing, ok := t.fragments[name]; ok {
		return existing, nil
	}
	if t.resolverMisses != nil {
		delete(t.resolverMisses, name)
	}
	if t.fragments == nil {
		t.fragments = make(map[string]*fragment)
	}
	t.fragments[name] = frag
	return frag, nil
}

func buildStaticPartCache(reader *DocxReader) (map[string][]byte, map[string]bool, error) {
	staticParts := make(map[string][]byte)
	dynamicParts := make(map[string]bool)
	if reader == nil {
		return staticParts, dynamicParts, nil
	}

	candidateParts := []string{"word/document.xml"}
	for _, name := range reader.ListParts() {
		if isHeaderPartName(name) || isFooterPartName(name) {
			candidateParts = append(candidateParts, name)
		}
	}

	for _, name := range candidateParts {
		content, err := reader.GetPart(name)
		if err != nil {
			return nil, nil, err
		}
		if partHasPotentialTemplateMarkers(name, content) {
			dynamicParts[name] = true
			continue
		}
		staticParts[name] = append([]byte(nil), content...)
	}

	return staticParts, dynamicParts, nil
}

func partHasPotentialTemplateMarkers(partName string, content []byte) bool {
	if bytes.Contains(content, []byte("{{")) || bytes.Contains(content, []byte("}}")) {
		return true
	}
	switch {
	case partName == "word/document.xml":
		doc, err := ParseDocument(bytes.NewReader(content))
		if err != nil {
			return true
		}
		return bodyHasPotentialTemplateMarkers(doc.Body)
	case isHeaderPartName(partName), isFooterPartName(partName):
		var headerFooter struct {
			Paragraphs []*Paragraph `xml:"p"`
			Tables     []*Table     `xml:"tbl"`
		}
		if err := xml.Unmarshal(content, &headerFooter); err != nil {
			return true
		}
		return elementsHavePotentialTemplateMarkers(paragraphsAndTablesToElements(headerFooter.Paragraphs, headerFooter.Tables))
	default:
		return false
	}
}

func paragraphsAndTablesToElements(paragraphs []*Paragraph, tables []*Table) []BodyElement {
	elements := make([]BodyElement, 0, len(paragraphs)+len(tables))
	for _, para := range paragraphs {
		elements = append(elements, para)
	}
	for _, table := range tables {
		elements = append(elements, table)
	}
	return elements
}

func bodyHasPotentialTemplateMarkers(body *Body) bool {
	if body == nil {
		return false
	}
	return elementsHavePotentialTemplateMarkers(body.Elements)
}

func elementsHavePotentialTemplateMarkers(elements []BodyElement) bool {
	for _, elem := range elements {
		switch e := elem.(type) {
		case *Paragraph:
			if paragraphHasPotentialTemplateMarkers(e) {
				return true
			}
		case *Table:
			if tableHasPotentialTemplateMarkers(e) {
				return true
			}
		}
	}
	return false
}

func paragraphHasPotentialTemplateMarkers(para *Paragraph) bool {
	if para == nil {
		return false
	}
	text := para.GetText()
	return strings.Contains(text, "{{") || strings.Contains(text, "}}")
}

func tableHasPotentialTemplateMarkers(table *Table) bool {
	if table == nil {
		return false
	}
	for _, row := range table.Rows {
		for _, cell := range row.Cells {
			for i := range cell.Paragraphs {
				if paragraphHasPotentialTemplateMarkers(&cell.Paragraphs[i]) {
					return true
				}
			}
		}
	}
	return false
}

func resolveFragmentByName(name string, ctx *renderContext) (*fragment, error) {
	if ctx == nil {
		return nil, fmt.Errorf("render context is nil")
	}
	if frag, ok := ctx.fragments[name]; ok {
		return frag, nil
	}
	if ctx.template == nil {
		return nil, nil
	}

	frag, err := ctx.template.resolveFragment(name)
	if err != nil {
		return nil, err
	}
	if frag != nil {
		ctx.fragments[name] = frag
	}
	return frag, nil
}
