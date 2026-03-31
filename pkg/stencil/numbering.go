package stencil

import (
	"fmt"
	"hash/fnv"
	"regexp"
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
	numberingNumPicBlockRegex   = regexp.MustCompile(`(?s)<w:numPicBullet\b.*?</w:numPicBullet>`)
	numberingAbstractBlockRegex = regexp.MustCompile(`(?s)<w:abstractNum\b.*?</w:abstractNum>`)
	numberingNumBlockRegex      = regexp.MustCompile(`(?s)<w:num\b.*?</w:num>`)
	numberingCleanupRegex       = regexp.MustCompile(`(?s)<w:numIdMacAtCleanup\b[^>]*/?>`)
	numberingAbstractIDRegex    = regexp.MustCompile(`\bw:abstractNumId="(\d+)"`)
	numberingNumIDRegex         = regexp.MustCompile(`<w:num\b[^>]*\bw:numId="(\d+)"`)
	numberingNumRefRegex        = regexp.MustCompile(`<w:abstractNumId\b[^>]*\bw:val="(\d+)"\s*/?>`)
	numberingNSIDRegex          = regexp.MustCompile(`<w:nsid\b[^>]*\bw:val="([0-9A-Fa-f]{8})"\s*/?>`)
	numberingTmplRegex          = regexp.MustCompile(`<w:tmpl\b[^>]*\bw:val="([0-9A-Fa-f]{8})"\s*/?>`)
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
	usedSignatures     map[string]bool
}

func newNumberingContext(docxReader *DocxReader) (*numberingContext, error) {
	ctx := &numberingContext{
		xml:               defaultNumberingXML(),
		nextAbstractNumID: 1,
		nextNumID:         1,
		fragmentNumMaps:   make(map[string]map[string]string),
		fragmentStylesXML: make(map[string][]byte),
		usedSignatures:    make(map[string]bool),
	}

	if docxReader == nil {
		return ctx, nil
	}

	if numberingXML, err := docxReader.GetPart("word/numbering.xml"); err == nil {
		ctx.xml = string(numberingXML)
		ctx.existsInTemplate = true
		ctx.nextAbstractNumID = maxNumberingID(numberingXML, numberingAbstractIDRegex) + 1
		ctx.nextNumID = maxNumberingID(numberingXML, regexp.MustCompile(`\bw:numId="(\d+)"`)) + 1
		ctx.collectExistingSignatures(ctx.xml)
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
	remappedAbstractBlocks := make([]string, 0, len(abstractBlocks))
	remappedNumBlocks := make([]string, 0, len(numBlocks))

	for _, block := range abstractBlocks {
		oldID, ok := extractNumberingMatch(block, numberingAbstractIDRegex)
		if !ok {
			return nil, fmt.Errorf("fragment %s has abstractNum without w:abstractNumId", fragmentName)
		}
		newID := strconv.Itoa(ctx.nextAbstractNumID)
		ctx.nextAbstractNumID++
		abstractMap[oldID] = newID
		remappedBlock := numberingAbstractIDRegex.ReplaceAllString(block, `w:abstractNumId="`+newID+`"`)
		remappedBlock = ctx.ensureUniqueAbstractSignature(fragmentName, newID, remappedBlock)
		remappedAbstractBlocks = append(remappedAbstractBlocks, remappedBlock)
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

		remappedNumBlocks = append(remappedNumBlocks, remapped)
		numMap[oldNumID] = newNumID
	}

	ctx.xml = insertNumberingDefinitions(ctx.xml, remappedAbstractBlocks, remappedNumBlocks)
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
	return []byte(normalizeNumberingXML(ctx.xml))
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

func insertNumberingDefinitions(xmlContent string, abstractBlocks, numBlocks []string) string {
	if len(abstractBlocks) == 0 && len(numBlocks) == 0 {
		return xmlContent
	}

	withAbstracts := insertNumberingAbstractBlocks(xmlContent, abstractBlocks)
	return insertNumberingNumBlocks(withAbstracts, numBlocks)
}

func insertNumberingAbstractBlocks(xmlContent string, blocks []string) string {
	if len(blocks) == 0 {
		return xmlContent
	}

	inserted := strings.Join(blocks, "\n")
	firstNumIdx := strings.Index(xmlContent, "<w:num ")
	if firstNumIdx != -1 {
		return xmlContent[:firstNumIdx] + inserted + "\n" + xmlContent[firstNumIdx:]
	}

	closingIdx := strings.LastIndex(xmlContent, numberingCloseTag)
	if closingIdx == -1 {
		return xmlContent
	}
	return xmlContent[:closingIdx] + inserted + xmlContent[closingIdx:]
}

func insertNumberingNumBlocks(xmlContent string, blocks []string) string {
	if len(blocks) == 0 {
		return xmlContent
	}

	inserted := strings.Join(blocks, "\n")
	closingIdx := strings.LastIndex(xmlContent, numberingCloseTag)
	if closingIdx == -1 {
		return xmlContent
	}
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
	for oldID, newID := range numMap {
		remapped = strings.ReplaceAll(remapped, `w:numId w:val="`+oldID+`"`, `w:numId w:val="`+newID+`"`)
	}
	return []byte(remapped)
}

func normalizeNumberingXML(xmlContent string) string {
	startTag := numberingStartTagRegex.FindString(xmlContent)
	if startTag == "" {
		return xmlContent
	}

	startIdx := strings.Index(xmlContent, startTag)
	if startIdx == -1 {
		return xmlContent
	}
	innerStart := startIdx + len(startTag)
	closingIdx := strings.LastIndex(xmlContent, numberingCloseTag)
	if closingIdx == -1 || closingIdx < innerStart {
		return xmlContent
	}

	inner := xmlContent[innerStart:closingIdx]
	numPicBlocks := numberingNumPicBlockRegex.FindAllString(inner, -1)
	abstractBlocks := numberingAbstractBlockRegex.FindAllString(inner, -1)
	numBlocks := numberingNumBlockRegex.FindAllString(inner, -1)
	cleanupBlocks := numberingCleanupRegex.FindAllString(inner, -1)

	remaining := inner
	for _, re := range []*regexp.Regexp{
		numberingNumPicBlockRegex,
		numberingAbstractBlockRegex,
		numberingNumBlockRegex,
		numberingCleanupRegex,
	} {
		remaining = re.ReplaceAllString(remaining, "")
	}
	remaining = strings.TrimSpace(remaining)

	var ordered []string
	ordered = append(ordered, numPicBlocks...)
	ordered = append(ordered, abstractBlocks...)
	ordered = append(ordered, numBlocks...)
	ordered = append(ordered, cleanupBlocks...)
	if remaining != "" {
		ordered = append(ordered, remaining)
	}

	body := strings.Join(ordered, "\n")
	if body != "" {
		body = "\n" + body
	}
	return xmlContent[:innerStart] + body + xmlContent[closingIdx:]
}

func (ctx *numberingContext) collectExistingSignatures(xmlContent string) {
	abstractBlocks := numberingAbstractBlockRegex.FindAllString(xmlContent, -1)
	for _, block := range abstractBlocks {
		nsid, okNSID := extractNumberingMatch(block, numberingNSIDRegex)
		tmpl, okTmpl := extractNumberingMatch(block, numberingTmplRegex)
		if !okNSID || !okTmpl {
			continue
		}
		ctx.usedSignatures[numberingSignatureKey(nsid, tmpl)] = true
	}
}

func (ctx *numberingContext) ensureUniqueAbstractSignature(fragmentName, abstractID, block string) string {
	nsid, okNSID := extractNumberingMatch(block, numberingNSIDRegex)
	tmpl, okTmpl := extractNumberingMatch(block, numberingTmplRegex)
	if okNSID && okTmpl {
		key := numberingSignatureKey(strings.ToUpper(nsid), strings.ToUpper(tmpl))
		if !ctx.usedSignatures[key] {
			ctx.usedSignatures[key] = true
			return block
		}
	}

	for salt := 0; ; salt++ {
		newNSID := numberedHexSignature(fragmentName, abstractID, "nsid", salt)
		newTmpl := numberedHexSignature(fragmentName, abstractID, "tmpl", salt)
		key := numberingSignatureKey(newNSID, newTmpl)
		if ctx.usedSignatures[key] {
			continue
		}
		ctx.usedSignatures[key] = true
		block = replaceOrInsertNumberingTag(block, "nsid", newNSID, numberingNSIDRegex)
		block = replaceOrInsertNumberingTag(block, "tmpl", newTmpl, numberingTmplRegex)
		return block
	}
}

func numberingSignatureKey(nsid, tmpl string) string {
	return nsid + ":" + tmpl
}

func numberedHexSignature(fragmentName, abstractID, kind string, salt int) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fragmentName))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(abstractID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(kind))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.Itoa(salt)))
	return strings.ToUpper(fmt.Sprintf("%08X", h.Sum32()))
}

func replaceOrInsertNumberingTag(block, tag, value string, re *regexp.Regexp) string {
	replacement := `<w:` + tag + ` w:val="` + value + `"/>`
	if re.MatchString(block) {
		return re.ReplaceAllString(block, replacement)
	}
	insertAfter := strings.Index(block, ">")
	if insertAfter == -1 {
		return block
	}
	insertAfter++
	return block[:insertAfter] + replacement + block[insertAfter:]
}
