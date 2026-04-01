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
	// Keep remapped fragment numbering IDs in a high reserved range so nested
	// fragments cannot accidentally collide with low fragment-local IDs from an
	// outer fragment that are still waiting to be remapped.
	fragmentNumberingIDFloor = 1024
)

var (
	numberingStartTagRegex      = regexp.MustCompile(`(?s)<w:numbering\b[^>]*>`)
	numberingAttrRegex          = regexp.MustCompile(`\s+([A-Za-z_:][A-Za-z0-9_.:-]*)="([^"]*)"`)
	numberingAbstractBlockRegex = regexp.MustCompile(`(?s)<w:abstractNum\b.*?</w:abstractNum>`)
	numberingNumBlockRegex      = regexp.MustCompile(`(?s)<w:num\b.*?</w:num>`)
	numberingAbstractIDRegex    = regexp.MustCompile(`\bw:abstractNumId="(\d+)"`)
	numberingNumIDRegex         = regexp.MustCompile(`<w:num\b[^>]*\bw:numId="(\d+)"`)
	numberingNumRefRegex        = regexp.MustCompile(`<w:abstractNumId\b[^>]*\bw:val="(\d+)"\s*/?>`)
	numberingNSIDRegex          = regexp.MustCompile(`(?s)<w:nsid\b[^>]*/>`)
	numberingTemplateRegex      = regexp.MustCompile(`(?s)<w:tmpl\b[^>]*/>`)
	numberingDurableIDRegex     = regexp.MustCompile(`\s+w16cid:durableId="[^"]*"`)
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

	if ctx.nextAbstractNumID < fragmentNumberingIDFloor {
		ctx.nextAbstractNumID = fragmentNumberingIDFloor
	}
	if ctx.nextNumID < fragmentNumberingIDFloor {
		ctx.nextNumID = fragmentNumberingIDFloor
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
	appendedAbstracts := make([]string, 0, len(abstractBlocks))
	appendedNums := make([]string, 0, len(numBlocks))

	for _, block := range abstractBlocks {
		oldID, ok := extractNumberingMatch(block, numberingAbstractIDRegex)
		if !ok {
			return nil, fmt.Errorf("fragment %s has abstractNum without w:abstractNumId", fragmentName)
		}
		newID := strconv.Itoa(ctx.nextAbstractNumID)
		ctx.nextAbstractNumID++
		abstractMap[oldID] = newID

		remapped := numberingAbstractIDRegex.ReplaceAllString(block, `w:abstractNumId="`+newID+`"`)
		remapped = sanitizeAbstractNumberingMetadata(remapped)
		appendedAbstracts = append(appendedAbstracts, remapped)
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
		remapped = sanitizeNumberingInstanceMetadata(remapped)
		if oldAbstractID, ok := extractNumberingMatch(remapped, numberingNumRefRegex); ok {
			if newAbstractID, exists := abstractMap[oldAbstractID]; exists {
				remapped = numberingNumRefRegex.ReplaceAllString(remapped, `<w:abstractNumId w:val="`+newAbstractID+`"/>`)
			}
		}

		appendedNums = append(appendedNums, remapped)
		numMap[oldNumID] = newNumID
	}

	ctx.xml = insertNumberingBlocks(ctx.xml, appendedAbstracts, appendedNums)
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

func insertNumberingBlocks(xmlContent string, abstractBlocks, numBlocks []string) string {
	if len(abstractBlocks) == 0 && len(numBlocks) == 0 {
		return xmlContent
	}

	closingIdx := strings.LastIndex(xmlContent, numberingCloseTag)
	if closingIdx == -1 {
		return xmlContent
	}

	startTag := numberingStartTagRegex.FindString(xmlContent)
	if startTag == "" {
		return xmlContent
	}
	startIdx := strings.Index(xmlContent, startTag)
	if startIdx == -1 {
		return xmlContent
	}

	innerStart := startIdx + len(startTag)
	if innerStart > closingIdx {
		return xmlContent
	}

	inner := xmlContent[innerStart:closingIdx]
	existingAbstracts := numberingAbstractBlockRegex.FindAllString(inner, -1)
	existingNums := numberingNumBlockRegex.FindAllString(inner, -1)

	firstBlockStart := len(inner)
	lastBlockEnd := 0
	for _, loc := range numberingAbstractBlockRegex.FindAllStringIndex(inner, -1) {
		if loc[0] < firstBlockStart {
			firstBlockStart = loc[0]
		}
		if loc[1] > lastBlockEnd {
			lastBlockEnd = loc[1]
		}
	}
	for _, loc := range numberingNumBlockRegex.FindAllStringIndex(inner, -1) {
		if loc[0] < firstBlockStart {
			firstBlockStart = loc[0]
		}
		if loc[1] > lastBlockEnd {
			lastBlockEnd = loc[1]
		}
	}

	leading := inner
	trailing := ""
	if firstBlockStart != len(inner) {
		leading = inner[:firstBlockStart]
		trailing = inner[lastBlockEnd:]
	}

	ordered := make([]string, 0, len(existingAbstracts)+len(abstractBlocks)+len(existingNums)+len(numBlocks))
	ordered = append(ordered, existingAbstracts...)
	ordered = append(ordered, abstractBlocks...)
	ordered = append(ordered, existingNums...)
	ordered = append(ordered, numBlocks...)

	var rebuilt strings.Builder
	rebuilt.Grow(len(xmlContent) + 128*len(ordered))
	rebuilt.WriteString(xmlContent[:innerStart])
	rebuilt.WriteString(leading)
	if len(ordered) > 0 {
		rebuilt.WriteString(strings.Join(ordered, "\n"))
	}
	rebuilt.WriteString(trailing)
	rebuilt.WriteString(xmlContent[closingIdx:])
	return rebuilt.String()
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

func sanitizeAbstractNumberingMetadata(block string) string {
	block = numberingNSIDRegex.ReplaceAllString(block, "")
	block = numberingTemplateRegex.ReplaceAllString(block, "")
	return block
}

func sanitizeNumberingInstanceMetadata(block string) string {
	return numberingDurableIDRegex.ReplaceAllString(block, "")
}
