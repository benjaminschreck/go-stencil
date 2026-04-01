package stencil

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	numberingRelationType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering"
	numberingContentType  = "application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"
	numberingMainNS       = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
)

var (
	numberingStartTagRegex      = regexp.MustCompile(`(?s)<w:numbering\b[^>]*>`)
	numberingAttrRegex          = regexp.MustCompile(`\s+([A-Za-z_:][A-Za-z0-9_.:-]*)="([^"]*)"`)
	numberingAbstractBlockRegex = regexp.MustCompile(`(?s)<w:abstractNum\b.*?</w:abstractNum>`)
	numberingNumBlockRegex      = regexp.MustCompile(`(?s)<w:num\b.*?</w:num>`)
	numberingAbstractIDRegex    = regexp.MustCompile(`\bw:abstractNumId="(\d+)"`)
	numberingNumIDRegex         = regexp.MustCompile(`<w:num\b[^>]*\bw:numId="(\d+)"`)
	numberingNumRefRegex        = regexp.MustCompile(`<w:abstractNumId\b[^>]*\bw:val="(\d+)"\s*/?>`)
	numberingCloseTag           = "</w:numbering>"
)

type numberingContext struct {
	xml                string
	existsInTemplate   bool
	relationshipExists bool
	modified           bool
	nextAbstractNumID  int
	nextNumID          int
	fragmentNumMaps    map[string]map[string]string
	fragmentStylesXML  map[string][]byte
}

func newNumberingContext(docxReader *DocxReader) (*numberingContext, error) {
	ctx := &numberingContext{
		xml:               defaultNumberingXML(),
		nextAbstractNumID: 1,
		nextNumID:         1,
		fragmentNumMaps:   make(map[string]map[string]string),
		fragmentStylesXML: make(map[string][]byte),
	}

	if docxReader == nil {
		return ctx, nil
	}

	if numberingXML, err := docxReader.GetPart("word/numbering.xml"); err == nil {
		ctx.xml = string(numberingXML)
		ctx.existsInTemplate = true
		ctx.nextAbstractNumID = maxNumberingID(numberingXML, numberingAbstractIDRegex) + 1
		ctx.nextNumID = maxNumberingID(numberingXML, regexp.MustCompile(`\bw:numId="(\d+)"`)) + 1
	}

	rels, err := docxReader.GetRelationships("word/document.xml")
	if err != nil {
		return nil, err
	}
	for _, rel := range rels {
		if rel.Type == numberingRelationType || rel.Target == "numbering.xml" {
			ctx.relationshipExists = true
			break
		}
	}

	return ctx, nil
}

func (ctx *numberingContext) ensureFragmentDefinitions(fragmentName string, numberingXML, stylesXML []byte) (map[string]string, error) {
	if len(numberingXML) == 0 {
		return nil, nil
	}
	if existingMap, ok := ctx.fragmentNumMaps[fragmentName]; ok {
		return existingMap, nil
	}

	fragmentXML := string(numberingXML)
	ctx.xml = mergeNumberingRootAttributes(ctx.xml, fragmentXML)

	abstractBlocks := numberingAbstractBlockRegex.FindAllString(fragmentXML, -1)
	numBlocks := numberingNumBlockRegex.FindAllString(fragmentXML, -1)
	if len(abstractBlocks) == 0 && len(numBlocks) == 0 {
		ctx.fragmentNumMaps[fragmentName] = map[string]string{}
		return ctx.fragmentNumMaps[fragmentName], nil
	}

	abstractMap := make(map[string]string)
	appendedBlocks := make([]string, 0, len(abstractBlocks)+len(numBlocks))

	for _, block := range abstractBlocks {
		oldID, ok := extractNumberingMatch(block, numberingAbstractIDRegex)
		if !ok {
			return nil, fmt.Errorf("fragment %s has abstractNum without w:abstractNumId", fragmentName)
		}
		newID := strconv.Itoa(ctx.nextAbstractNumID)
		ctx.nextAbstractNumID++
		abstractMap[oldID] = newID
		appendedBlocks = append(appendedBlocks, numberingAbstractIDRegex.ReplaceAllString(block, `w:abstractNumId="`+newID+`"`))
	}

	numMap := make(map[string]string)
	for _, block := range numBlocks {
		oldNumID, ok := extractNumberingMatch(block, numberingNumIDRegex)
		if !ok {
			return nil, fmt.Errorf("fragment %s has num without w:numId", fragmentName)
		}
		newNumID := strconv.Itoa(ctx.nextNumID)
		ctx.nextNumID++

		remapped := numberingNumIDRegex.ReplaceAllString(block, `<w:num w:numId="`+newNumID+`"`)
		if oldAbstractID, ok := extractNumberingMatch(remapped, numberingNumRefRegex); ok {
			if newAbstractID, exists := abstractMap[oldAbstractID]; exists {
				remapped = numberingNumRefRegex.ReplaceAllString(remapped, `<w:abstractNumId w:val="`+newAbstractID+`"/>`)
			}
		}

		appendedBlocks = append(appendedBlocks, remapped)
		numMap[oldNumID] = newNumID
	}

	ctx.xml = insertNumberingBlocks(ctx.xml, appendedBlocks)
	ctx.modified = true
	ctx.fragmentNumMaps[fragmentName] = numMap
	if len(stylesXML) > 0 && len(numMap) > 0 {
		ctx.fragmentStylesXML[fragmentName] = remapStylesNumberingIDs(stylesXML, numMap)
	}

	return numMap, nil
}

func (ctx *numberingContext) needsRelationship() bool {
	return ctx.modified && !ctx.relationshipExists
}

func (ctx *numberingContext) needsContentTypeOverride() bool {
	return ctx.modified && !ctx.existsInTemplate
}

func (ctx *numberingContext) partXML() []byte {
	return []byte(ctx.xml)
}

func defaultNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="` + numberingMainNS + `"></w:numbering>`
}

func maxNumberingID(xmlContent []byte, re *regexp.Regexp) int {
	maxID := 0
	matches := re.FindAllSubmatch(xmlContent, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		id, err := strconv.Atoi(string(match[1]))
		if err == nil && id > maxID {
			maxID = id
		}
	}
	return maxID
}

func extractNumberingMatch(content string, re *regexp.Regexp) (string, bool) {
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return "", false
	}
	return match[1], true
}

func insertNumberingBlocks(xmlContent string, blocks []string) string {
	if len(blocks) == 0 {
		return xmlContent
	}
	closingIdx := strings.LastIndex(xmlContent, numberingCloseTag)
	if closingIdx == -1 {
		return xmlContent
	}

	inserted := strings.Join(blocks, "\n")
	return xmlContent[:closingIdx] + inserted + xmlContent[closingIdx:]
}

func mergeNumberingRootAttributes(baseXML, fragmentXML string) string {
	baseTag := numberingStartTagRegex.FindString(baseXML)
	fragmentTag := numberingStartTagRegex.FindString(fragmentXML)
	if baseTag == "" || fragmentTag == "" {
		return baseXML
	}

	existingAttrs := make(map[string]bool)
	for _, match := range numberingAttrRegex.FindAllStringSubmatch(baseTag, -1) {
		existingAttrs[match[1]] = true
	}

	additions := make([]string, 0)
	for _, match := range numberingAttrRegex.FindAllStringSubmatch(fragmentTag, -1) {
		if existingAttrs[match[1]] {
			continue
		}
		existingAttrs[match[1]] = true
		additions = append(additions, fmt.Sprintf(`%s="%s"`, match[1], match[2]))
	}
	if len(additions) == 0 {
		return baseXML
	}

	updatedTag := strings.TrimSuffix(baseTag, ">") + " " + strings.Join(additions, " ") + ">"
	return strings.Replace(baseXML, baseTag, updatedTag, 1)
}

func remapStylesNumberingIDs(stylesXML []byte, numMap map[string]string) []byte {
	remapped := string(stylesXML)
	remapped = replaceNumberingIDReferences(remapped, func(id string) string {
		return `w:numId w:val="` + id + `"`
	}, numMap)
	return []byte(remapped)
}

func replaceNumberingIDReferences(content string, pattern func(string) string, numIDMap map[string]string) string {
	if len(numIDMap) == 0 || content == "" {
		return content
	}

	type numberedReplacement struct {
		oldID       string
		newID       string
		oldPattern  string
		newPattern  string
		placeholder string
	}

	keys := make([]string, 0, len(numIDMap))
	for oldID := range numIDMap {
		keys = append(keys, oldID)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) == len(keys[j]) {
			return keys[i] < keys[j]
		}
		return len(keys[i]) > len(keys[j])
	})

	replacements := make([]numberedReplacement, 0, len(keys))
	for idx, oldID := range keys {
		newID := numIDMap[oldID]
		replacements = append(replacements, numberedReplacement{
			oldID:       oldID,
			newID:       newID,
			oldPattern:  pattern(oldID),
			newPattern:  pattern(newID),
			placeholder: fmt.Sprintf("__GO_STENCIL_NUM_%d__", idx),
		})
	}

	remapped := content
	for _, replacement := range replacements {
		remapped = strings.ReplaceAll(remapped, replacement.oldPattern, replacement.placeholder)
	}
	for _, replacement := range replacements {
		remapped = strings.ReplaceAll(remapped, replacement.placeholder, replacement.newPattern)
	}

	return remapped
}
