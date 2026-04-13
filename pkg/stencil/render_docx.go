package stencil

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

// renderElementsWithContext renders a slice of elements with the given context
// This function DOES process control structures to support nested loops and conditionals
func renderElementsWithContext(elements []BodyElement, data TemplateData, ctx *renderContext) ([]BodyElement, error) {
	body := &Body{Elements: elements}
	return renderBodyElementRange(body, compileBodyRenderPlan(body), 0, len(elements), data, ctx)
}

// getFragmentKeys returns the keys of fragments in the context for debugging
func getFragmentKeys(ctx *renderContext) []string {
	if ctx == nil || ctx.ooxmlFragments == nil {
		return []string{}
	}
	keys := make([]string, 0, len(ctx.ooxmlFragments))
	for k := range ctx.ooxmlFragments {
		keys = append(keys, k)
	}
	return keys
}

var docxFragmentMarkerRegex = regexp.MustCompile(`__DOCX_FRAGMENT__(.+?)__`)

func expandDOCXFragmentParagraph(
	renderedPara *Paragraph,
	data TemplateData,
	ctx *renderContext,
	renderFragment func(*fragment) ([]BodyElement, error),
) ([]BodyElement, bool, error) {
	if renderedPara == nil {
		return nil, false, nil
	}

	paraText := renderedPara.GetText()
	matches := docxFragmentMarkerRegex.FindAllStringSubmatchIndex(paraText, -1)
	if len(matches) == 0 {
		return nil, false, nil
	}
	if ctx == nil || ctx.ooxmlFragments == nil {
		return nil, false, fmt.Errorf("fragment marker found but fragments not in context")
	}

	result := make([]BodyElement, 0, len(matches)+1)
	var currentPara *Paragraph
	currentParaAttached := false

	if prefixRuns := extractRunsBetweenTextOffsets(renderedPara.Runs, 0, matches[0][0]); len(prefixRuns) > 0 {
		currentPara = newParagraphWithRunsLike(renderedPara, prefixRuns)
	}

	for matchIdx, match := range matches {
		fragmentName := paraText[match[2]:match[3]]
		markerKey := fmt.Sprintf("__DOCX_FRAGMENT__%s__", fragmentName)

		fragValue, ok := ctx.ooxmlFragments[markerKey]
		if !ok {
			return nil, false, fmt.Errorf("fragment marker %s found but fragment not in context (available: %v)", fragmentName, getFragmentKeys(ctx))
		}

		frag, ok := fragValue.(*fragment)
		if !ok {
			return nil, false, fmt.Errorf("fragment marker %s resolved to unexpected type %T", fragmentName, fragValue)
		}
		if ctx != nil && frag.isDocx {
			ctx.usedDocxFragments[fragmentName] = true
		}
		if frag.parsed == nil || frag.parsed.Body == nil {
			return nil, false, fmt.Errorf("fragment %s has no parsed body", fragmentName)
		}

		fragmentElements, err := renderFragment(frag)
		if err != nil {
			return nil, false, fmt.Errorf("failed to render fragment %s: %w", fragmentName, err)
		}
		applyFragmentFontOverrides(fragmentElements, fragmentName, ctx)

		if len(fragmentElements) > 0 {
			mergedIntoAttachedPara := currentParaAttached && currentPara != nil
			if firstPara, ok := fragmentElements[0].(*Paragraph); ok {
				if currentPara != nil {
					if currentParaAttached {
						currentPara.Runs = append(currentPara.Runs, firstPara.Runs...)
						fragmentElements = fragmentElements[1:]
					} else {
						firstPara.Runs = append(currentPara.Runs, firstPara.Runs...)
					}
					currentPara = nil
					currentParaAttached = false
				}
			} else if currentPara != nil && !currentParaAttached {
				result = append(result, currentPara)
				currentPara = nil
			}

			result = append(result, fragmentElements...)

			if len(fragmentElements) > 0 {
				if lastPara, ok := fragmentElements[len(fragmentElements)-1].(*Paragraph); ok {
					currentPara = lastPara
					currentParaAttached = true
				} else {
					currentPara = nil
					currentParaAttached = false
				}
			} else if mergedIntoAttachedPara {
				currentPara = result[len(result)-1].(*Paragraph)
				currentParaAttached = true
			}
		}

		segmentStart := match[1]
		segmentEnd := len(paraText)
		if matchIdx+1 < len(matches) {
			segmentEnd = matches[matchIdx+1][0]
		}

		if betweenRuns := extractRunsBetweenTextOffsets(renderedPara.Runs, segmentStart, segmentEnd); len(betweenRuns) > 0 {
			if currentPara == nil {
				currentPara = newParagraphWithRunsLike(renderedPara, betweenRuns)
				currentParaAttached = false
			} else {
				currentPara.Runs = append(currentPara.Runs, betweenRuns...)
			}
		}
	}

	if currentPara != nil && !currentParaAttached {
		result = append(result, currentPara)
	}

	return result, true, nil
}

func newParagraphWithRunsLike(base *Paragraph, runs []Run) *Paragraph {
	if len(runs) == 0 {
		return nil
	}

	para := &Paragraph{
		Properties: base.Properties,
		Attrs:      base.Attrs,
		Runs:       cloneRunsForLegacyRendering(runs),
	}
	return para
}

func extractRunsBetweenTextOffsets(runs []Run, start, end int) []Run {
	if start >= end || len(runs) == 0 {
		return nil
	}

	var extracted []Run
	textPos := 0

	for _, run := range runs {
		if run.Text == nil {
			continue
		}

		runStart := textPos
		runEnd := textPos + len(run.Text.Content)
		textPos = runEnd

		if end <= runStart || start >= runEnd {
			continue
		}

		segmentStart := max(start-runStart, 0)
		segmentEnd := min(end-runStart, len(run.Text.Content))
		if segmentStart >= segmentEnd {
			continue
		}

		clonedRun := *cloneRun(&run)
		textCopy := *clonedRun.Text
		textCopy.Content = run.Text.Content[segmentStart:segmentEnd]
		clonedRun.Text = &textCopy
		extracted = append(extracted, clonedRun)
	}

	return extracted
}

func planEntryAt(plan *bodyRenderPlan, idx int) bodyRenderPlanEntry {
	if plan == nil || idx < 0 || idx >= len(plan.entries) {
		return bodyRenderPlanEntry{endIdx: -1}
	}
	return plan.entries[idx]
}

func fallbackFindMatchingEnd(elements []BodyElement, startIdx int) (int, error) {
	return render.FindMatchingEndInElements(elements, startIdx)
}

func fallbackFindIfStructure(elements []BodyElement, startIdx int) (int, []render.ElseBranch, error) {
	return render.FindIfStructureInElements(elements, startIdx)
}

func branchBodiesForEntry(entry bodyRenderPlanEntry, endIdx int) []render.ElseBranch {
	if len(entry.branches) == 0 {
		return nil
	}

	branches := make([]render.ElseBranch, 0, len(entry.branches))
	for _, branch := range entry.branches {
		branches = append(branches, render.ElseBranch{
			Index:      branch.index,
			BranchType: branch.branchType,
			Condition:  branch.condition,
		})
	}
	if endIdx < 0 {
		return nil
	}
	return branches
}

func renderBodyElementRange(body *Body, plan *bodyRenderPlan, start, end int, data TemplateData, ctx *renderContext) ([]BodyElement, error) {
	if body == nil {
		return nil, nil
	}
	if start < 0 {
		start = 0
	}
	if end > len(body.Elements) {
		end = len(body.Elements)
	}

	result := make([]BodyElement, 0, max(end-start, 0))

	for i := start; i < end; {
		elem := body.Elements[i]
		entry := planEntryAt(plan, i)

		switch el := elem.(type) {
		case *Paragraph:
			controlType := entry.controlType
			controlContent := entry.controlContent
			if controlType == "" && plan == nil {
				controlType, controlContent = render.DetectControlStructure(el)
			}

			switch controlType {
			case "inline-for":
				renderedParas, err := renderInlineForLoop(el, controlContent, data, ctx)
				if err != nil {
					return nil, err
				}
				for _, p := range renderedParas {
					result = append(result, &p)
				}
				i++

			case "for":
				forNode := entry.forNode
				if forNode == nil {
					var err error
					forNode, err = parseForSyntax(controlContent)
					if err != nil {
						return nil, fmt.Errorf("invalid for syntax: %w", err)
					}
				}

				endIdx := entry.endIdx
				if endIdx < 0 {
					var err error
					endIdx, err = fallbackFindMatchingEnd(body.Elements, i)
					if err != nil {
						return nil, fmt.Errorf("no matching {{end}} for {{for}} at element %d", i)
					}
				}

				collection, err := forNode.Collection.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate collection: %w", err)
				}

				items, err := toSlice(collection)
				if err != nil {
					return nil, fmt.Errorf("failed to convert collection to slice: %w", err)
				}

				for idx, item := range items {
					loopData := newChildTemplateData(data, 2)
					loopData[forNode.Variable] = item
					if forNode.IndexVar != "" {
						loopData[forNode.IndexVar] = idx
					}

					loopRendered, err := renderBodyElementRange(body, plan, i+1, endIdx, loopData, ctx)
					if err != nil {
						return nil, err
					}
					result = append(result, loopRendered...)
				}

				i = endIdx + 1

			case "if":
				endIdx := entry.endIdx
				branches := branchBodiesForEntry(entry, endIdx)
				if endIdx < 0 {
					var err error
					endIdx, branches, err = fallbackFindIfStructure(body.Elements, i)
					if err != nil {
						return nil, fmt.Errorf("no matching {{end}} for {{if}} at element %d: %w", i, err)
					}
				}

				expr := entry.conditionExpr
				if expr == nil {
					var err error
					expr, err = ParseExpression(controlContent)
					if err != nil {
						return nil, fmt.Errorf("failed to parse if condition: %w", err)
					}
				}

				prefixRuns := extractPrefixRunsBeforeControlMarker(el.Runs, "{{if ")

				condValue, err := expr.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate if condition: %w", err)
				}

				branchElements, err := renderSelectedIfBranch(body, plan, el, i, endIdx, branches, prefixRuns, isTruthy(condValue), data, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, branchElements...)
				i = endIdx + 1

			case "unless":
				endIdx := entry.endIdx
				branches := branchBodiesForEntry(entry, endIdx)
				if endIdx < 0 {
					var err error
					endIdx, branches, err = fallbackFindIfStructure(body.Elements, i)
					if err != nil {
						return nil, fmt.Errorf("no matching {{end}} for {{unless}} at element %d", i)
					}
				}

				expr := entry.conditionExpr
				if expr == nil {
					var err error
					expr, err = ParseExpression(controlContent)
					if err != nil {
						return nil, fmt.Errorf("failed to parse unless condition: %w", err)
					}
				}

				condValue, err := expr.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate unless condition: %w", err)
				}

				if !isTruthy(condValue) {
					branchEnd := endIdx
					if len(branches) > 0 && branches[0].BranchType == "else" {
						branchEnd = branches[0].Index
					}
					branchElements, err := renderBodyElementRange(body, plan, i+1, branchEnd, data, ctx)
					if err != nil {
						return nil, err
					}
					result = append(result, branchElements...)
				} else if len(branches) > 0 && branches[0].BranchType == "else" {
					branchElements, err := renderBodyElementRange(body, plan, branches[0].Index+1, endIdx, data, ctx)
					if err != nil {
						return nil, err
					}
					result = append(result, branchElements...)
				}
				i = endIdx + 1

			case "include":
				if ctx == nil || ctx.fragments == nil {
					return nil, fmt.Errorf("fragments not available in render context")
				}

				expr := entry.includeExpr
				if expr == nil {
					var err error
					expr, err = ParseExpression(controlContent)
					if err != nil {
						return nil, fmt.Errorf("failed to parse include expression: %w", err)
					}
				}

				fragmentNameValue, err := expr.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate fragment name: %w", err)
				}

				fragmentName, ok := fragmentNameValue.(string)
				if !ok {
					return nil, fmt.Errorf("fragment name must be a string, got %T", fragmentNameValue)
				}

				frag, exists := ctx.fragments[fragmentName]
				if !exists {
					return nil, fmt.Errorf("fragment not found: %s", fragmentName)
				}

				fragmentElements, err := renderIncludedFragment(fragmentName, frag, data, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, fragmentElements...)
				i++

			case "end":
				return nil, fmt.Errorf("unmatched {{end}} at element %d", i)

			default:
				renderedPara, err := RenderParagraphWithContext(el, data, ctx)
				if err != nil {
					return nil, err
				}

				fragmentElements, handled, err := expandDOCXFragmentParagraph(renderedPara, data, ctx, func(fragment *fragment) ([]BodyElement, error) {
					fragmentBody, err := RenderBodyWithControlStructures(fragment.parsed.Body, data, ctx)
					if err != nil {
						return nil, err
					}
					return fragmentBody.Elements, nil
				})
				if err != nil {
					return nil, err
				}
				if handled {
					result = append(result, fragmentElements...)
					i++
					continue
				}

				result = append(result, renderedPara)
				i++
			}

		case *Table:
			table := cloneTable(el)
			renderedTable, err := RenderTableWithControlStructures(table, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, renderedTable)
			i++

		default:
			result = append(result, elem)
			i++
		}
	}

	return result, nil
}

func renderSelectedIfBranch(body *Body, plan *bodyRenderPlan, openingPara *Paragraph, startIdx, endIdx int, branches []render.ElseBranch, prefixRuns []Run, branchTruth bool, data TemplateData, ctx *renderContext) ([]BodyElement, error) {
	var branchElements []BodyElement
	var err error

	if branchTruth {
		branchEnd := endIdx
		if len(branches) > 0 {
			branchEnd = branches[0].Index
		}
		branchElements, err = renderBodyElementRange(body, plan, startIdx+1, branchEnd, data, ctx)
		if err != nil {
			return nil, err
		}
	} else {
		for j, branch := range branches {
			if branch.BranchType == "elsif" || branch.BranchType == "elif" || branch.BranchType == "elseif" {
				branchEntry := bodyRenderBranch{
					index:      branch.Index,
					branchType: branch.BranchType,
					condition:  branch.Condition,
				}
				entry := planEntryAt(plan, startIdx)
				for _, candidate := range entry.branches {
					if candidate.index == branch.Index {
						branchEntry = candidate
						break
					}
				}

				elsifExpr := branchEntry.expr
				if elsifExpr == nil {
					elsifExpr, err = ParseExpression(branch.Condition)
					if err != nil {
						return nil, fmt.Errorf("failed to parse elsif condition: %w", err)
					}
				}

				condValue, err := elsifExpr.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate elsif condition: %w", err)
				}
				if !isTruthy(condValue) {
					continue
				}

				branchEnd := endIdx
				if j+1 < len(branches) {
					branchEnd = branches[j+1].Index
				}
				branchElements, err = renderBodyElementRange(body, plan, branch.Index+1, branchEnd, data, ctx)
				if err != nil {
					return nil, err
				}
				break
			}
			if branch.BranchType == "else" {
				branchElements, err = renderBodyElementRange(body, plan, branch.Index+1, endIdx, data, ctx)
				if err != nil {
					return nil, err
				}
				break
			}
		}
	}

	if len(prefixRuns) == 0 || len(branchElements) == 0 {
		return branchElements, nil
	}

	if firstPara, ok := branchElements[0].(*Paragraph); ok {
		newPara := &Paragraph{
			Properties: firstPara.Properties,
		}
		newPara.Runs = append(newPara.Runs, prefixRuns...)
		newPara.Runs = append(newPara.Runs, firstPara.Runs...)
		branchElements[0] = newPara
		return branchElements, nil
	}

	prefixPara := &Paragraph{
		Properties: openingPara.Properties,
		Runs:       prefixRuns,
	}
	return append([]BodyElement{prefixPara}, branchElements...), nil
}

func renderIncludedFragment(fragmentName string, frag *fragment, data TemplateData, ctx *renderContext) ([]BodyElement, error) {
	if frag == nil {
		return nil, fmt.Errorf("fragment not found: %s", fragmentName)
	}

	if frag.namespaces != nil {
		for prefix, uri := range frag.namespaces {
			if existingURI, exists := ctx.collectedNamespaces[prefix]; exists {
				if existingURI != uri {
					if prefix == "" {
						continue
					}
					return nil, fmt.Errorf(
						"namespace conflict in fragment %q: prefix %q used for both %q and %q",
						fragmentName, prefix, existingURI, uri)
				}
			} else {
				ctx.collectedNamespaces[prefix] = uri
			}
		}
	}

	if frag.parsed == nil || frag.parsed.Body == nil {
		return nil, nil
	}
	if ctx != nil && frag.isDocx {
		ctx.usedDocxFragments[fragmentName] = true
	}

	for _, f := range ctx.fragmentStack {
		if f == fragmentName {
			return nil, fmt.Errorf("circular fragment reference detected: %s", fragmentName)
		}
	}

	ctx.fragmentStack = append(ctx.fragmentStack, fragmentName)
	maxDepth := 10
	if ctx.renderDepth > 0 {
		maxDepth = ctx.renderDepth
	}
	if len(ctx.fragmentStack) > maxDepth {
		ctx.fragmentStack = ctx.fragmentStack[:len(ctx.fragmentStack)-1]
		return nil, fmt.Errorf("maximum render depth exceeded")
	}

	renderedBody, err := func() (*Body, error) {
		defer func() {
			ctx.fragmentStack = ctx.fragmentStack[:len(ctx.fragmentStack)-1]
		}()
		return RenderBodyWithControlStructures(frag.parsed.Body, data, ctx)
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to render fragment %s: %w", fragmentName, err)
	}
	applyFragmentFontOverrides(renderedBody.Elements, fragmentName, ctx)

	if frag.isDocx && len(frag.relationships) > 0 {
		rangeStart, exists := ctx.fragmentIDAllocations[fragmentName]
		if !exists {
			rangeStart = ctx.nextFragmentIDRange
			ctx.fragmentIDAllocations[fragmentName] = rangeStart
			ctx.nextFragmentIDRange += FragmentIDRangeSize
		}

		if !ctx.fragmentResourcesAdded[fragmentName] {
			imageCounter := 1
			for _, rel := range frag.relationships {
				if !isMediaRelationship(rel) {
					continue
				}

				idNum, err := extractRelationshipNumber(rel.ID)
				if err != nil {
					return nil, fmt.Errorf("invalid relationship ID %s in fragment %s: %w", rel.ID, fragmentName, err)
				}
				if idNum >= FragmentIDRangeSize {
					return nil, fmt.Errorf("fragment %s relationship ID %s exceeds range size %d",
						fragmentName, rel.ID, FragmentIDRangeSize)
				}

				newID := fmt.Sprintf("rId%d", rangeStart+idNum)
				newTarget := renameMediaPath(rel.Target, fragmentName, imageCounter)
				if mediaContent, ok := frag.mediaFiles[rel.Target]; ok {
					newFilename := filepath.Base(newTarget)
					ctx.fragmentMedia[newFilename] = mediaContent
				}

				ctx.fragmentRelationships = append(ctx.fragmentRelationships, Relationship{
					ID:     newID,
					Type:   rel.Type,
					Target: newTarget,
				})
				imageCounter++
			}
			ctx.fragmentResourcesAdded[fragmentName] = true
		}

		idMap := make(map[string]string)
		for _, rel := range frag.relationships {
			if !isMediaRelationship(rel) {
				continue
			}
			idNum, _ := extractRelationshipNumber(rel.ID)
			idMap[rel.ID] = fmt.Sprintf("rId%d", rangeStart+idNum)
		}

		tempDoc := &Document{Body: renderedBody}
		updateDocumentRelationshipIDs(tempDoc, idMap)
	}

	if frag.isDocx && ctx.numbering != nil && len(frag.numberingXML) > 0 {
		var numMap map[string]string
		var err error
		if frag.compiled != nil && frag.compiled.numbering != nil {
			numMap, err = ctx.numbering.ensureCompiledFragmentDefinitions(fragmentName, frag.compiled.numbering)
		} else {
			numMap, err = ctx.numbering.ensureFragmentDefinitions(fragmentName, frag.numberingXML, frag.stylesXML)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to merge numbering for fragment %s: %w", fragmentName, err)
		}
		if len(numMap) > 0 {
			tempDoc := &Document{Body: renderedBody}
			updateDocumentNumberingIDs(tempDoc, numMap)
		}
	}

	return renderedBody.Elements, nil
}

// RenderBodyWithControlStructures renders a document body handling control structures
func RenderBodyWithControlStructures(body *Body, data TemplateData, ctx *renderContext) (*Body, error) {
	rendered, err := renderBodyWithElementOrder(body, data, ctx)
	if err != nil {
		return nil, err
	}

	// Apply table merging to fix split tables from for loops outside tables
	rendered.Elements = MergeConsecutiveTables(rendered.Elements)

	return rendered, nil
}

// renderBodyWithElementOrder renders using the new Elements field that preserves order
func renderBodyWithElementOrder(body *Body, data TemplateData, ctx *renderContext) (*Body, error) {
	rendered := &Body{
		SectionProperties: body.SectionProperties,
	}
	elements, err := renderBodyElementRange(body, resolveBodyRenderPlan(body, ctx), 0, len(body.Elements), data, ctx)
	if err != nil {
		return nil, err
	}
	rendered.Elements = elements

	return rendered, nil
}

// extractPrefixRunsBeforeControlMarker returns runs that appear before the first
// control marker in the paragraph text. It supports markers split across runs.
func extractPrefixRunsBeforeControlMarker(runs []Run, marker string) []Run {
	if len(runs) == 0 {
		return nil
	}

	var fullText strings.Builder
	for _, run := range runs {
		if run.Break != nil {
			// Keep line break position aligned with the run walk below.
			fullText.WriteByte('\n')
		}
		if run.Text != nil {
			fullText.WriteString(run.Text.Content)
		}
	}

	markerIdx := strings.Index(fullText.String(), marker)
	if markerIdx <= 0 {
		// markerIdx == 0 means marker starts immediately; no prefix.
		// markerIdx < 0 means marker wasn't found; safest is no prefix.
		return nil
	}

	remaining := markerIdx
	prefixRuns := make([]Run, 0, len(runs))

	for _, run := range runs {
		if remaining <= 0 {
			break
		}

		if run.Break != nil {
			prefixRuns = append(prefixRuns, run)
			remaining--
			if remaining <= 0 {
				break
			}
		}

		if run.Text == nil {
			continue
		}

		textLen := len(run.Text.Content)
		if textLen <= remaining {
			prefixRuns = append(prefixRuns, run)
			remaining -= textLen
			continue
		}

		// Marker starts inside this run - keep only the prefix text segment.
		prefixRun := run
		textCopy := *run.Text
		textCopy.Content = run.Text.Content[:remaining]
		prefixRun.Text = &textCopy
		prefixRuns = append(prefixRuns, prefixRun)
		remaining = 0
	}

	// If we could not map the full prefix offset, avoid injecting partial data.
	if remaining > 0 {
		return nil
	}

	return prefixRuns
}

// renderInlineForLoop handles loops that are entirely within one paragraph
func renderInlineForLoop(para *Paragraph, loopText string, data TemplateData, _ *renderContext) ([]Paragraph, error) {
	// Extract the for syntax and body
	// Format: "{{for item in items}} content {{end}}"
	forStart := strings.Index(loopText, "{{for ")
	forEnd := strings.Index(loopText[forStart:], "}}") + forStart + 2

	if forStart < 0 || forEnd < 0 {
		return nil, fmt.Errorf("invalid inline for loop syntax")
	}

	// Find the matching {{end}} for this {{for}} by counting depth
	endStart := render.FindMatchingEnd(loopText, forEnd)
	if endStart < 0 {
		return nil, fmt.Errorf("no matching {{end}} for {{for}} loop")
	}

	// Extract parts
	prefix := loopText[:forStart]
	forExpr := loopText[forStart+6 : forEnd-2] // Remove {{for and }}
	loopBody := loopText[forEnd:endStart]
	suffix := loopText[endStart+7:] // After {{end}}

	// Parse for syntax
	forNode, err := parseForSyntax(strings.TrimSpace(forExpr))
	if err != nil {
		return nil, fmt.Errorf("invalid for syntax: %w", err)
	}

	// Evaluate collection
	collection, err := forNode.Collection.Evaluate(data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate collection: %w", err)
	}

	// Build result
	var resultText strings.Builder

	// Process prefix (may contain template expressions)
	processedPrefix, err := processTemplateText(prefix, data)
	if err != nil {
		return nil, err
	}
	resultText.WriteString(processedPrefix)

	// Iterate over collection
	items, err := toSlice(collection)
	if err != nil {
		return nil, fmt.Errorf("failed to convert collection to slice: %w", err)
	}
	for idx, item := range items {
		loopData := newChildTemplateData(data, 2)
		loopData[forNode.Variable] = item
		if forNode.IndexVar != "" {
			loopData[forNode.IndexVar] = idx
		}

		// Process loop body with substitutions
		processedBody, err := processTemplateText(loopBody, loopData)
		if err != nil {
			return nil, err
		}
		resultText.WriteString(processedBody)
	}

	// Process suffix (may contain additional template expressions)
	processedSuffix, err := processTemplateText(suffix, data)
	if err != nil {
		return nil, err
	}
	resultText.WriteString(processedSuffix)

	// Create new paragraph with processed text
	resultPara := &Paragraph{
		Properties: para.Properties,
	}

	// Create a new run with the processed text
	if len(para.Runs) > 0 {
		// Copy properties from first run
		run := &Run{
			Properties: para.Runs[0].Properties,
			Text: &Text{
				Content: resultText.String(),
				Space:   "preserve",
			},
		}
		resultPara.Runs = append(resultPara.Runs, *run)
	} else {
		// Create default run
		run := &Run{
			Text: &Text{
				Content: resultText.String(),
				Space:   "preserve",
			},
		}
		resultPara.Runs = append(resultPara.Runs, *run)
	}

	return []Paragraph{*resultPara}, nil
}

// processTemplateText processes template variables and control structures in text
// Only processes control structures that are complete within the text
func processTemplateText(text string, data TemplateData) (string, error) {
	// Tokenize the text
	tokens := Tokenize(text)

	// Check if we have complete control structures (balanced if/end)
	// If not, treat control structure tokens as variables (they'll be handled at table/paragraph level)
	if !hasCompleteControlStructures(tokens) {
		// Fall back to simple variable substitution only
		return processTokensSimple(tokens, data)
	}

	// Process tokens with control structure support
	result, _, err := processTokens(tokens, 0, data)
	return result, err
}

// hasCompleteControlStructures checks if all control structures are balanced
func hasCompleteControlStructures(tokens []Token) bool {
	depth := 0
	for _, token := range tokens {
		switch token.Type {
		case TokenIf, TokenUnless, TokenFor:
			depth++
		case TokenEnd:
			depth--
			if depth < 0 {
				return false // More ends than starts
			}
		}
	}
	return depth == 0 // All control structures are balanced
}

// processTokensSimple processes tokens with variable substitution only (no control structures)
func processTokensSimple(tokens []Token, data TemplateData) (string, error) {
	var result strings.Builder

	for _, token := range tokens {
		switch token.Type {
		case TokenText:
			result.WriteString(token.Value)

		case TokenVariable:
			// Evaluate the variable
			value, err := EvaluateVariable(token.Value, data)
			if err != nil || value == nil {
				// Try to parse as an expression
				expr, parseErr := ParseExpression(token.Value)
				if parseErr != nil {
					// Not an expression either, leave empty
					result.WriteString("")
				} else {
					// Evaluate the expression
					exprValue, evalErr := expr.Evaluate(data)
					if evalErr != nil {
						result.WriteString("")
					} else {
						result.WriteString(FormatValue(exprValue))
					}
				}
			} else {
				result.WriteString(FormatValue(value))
			}

		default:
			// Leave control structure tokens as-is - they'll be handled at table/paragraph level
			result.WriteString("{{")
			if token.Type == TokenIf {
				result.WriteString("if ")
			} else if token.Type == TokenUnless {
				result.WriteString("unless ")
			} else if token.Type == TokenElse {
				result.WriteString("else")
			} else if token.Type == TokenElsif {
				result.WriteString("elsif ")
			} else if token.Type == TokenFor {
				result.WriteString("for ")
			} else if token.Type == TokenEnd {
				// End doesn't need the keyword repeated
				result.WriteString("end")
				result.WriteString("}}")
				continue
			}
			result.WriteString(token.Value)
			result.WriteString("}}")
		}
	}

	return result.String(), nil
}

// processTokens processes a slice of tokens starting at the given index
// Returns: (rendered text, next index to process, error)
func processTokens(tokens []Token, startIdx int, data TemplateData) (string, int, error) {
	var result strings.Builder
	i := startIdx

	for i < len(tokens) {
		token := tokens[i]

		switch token.Type {
		case TokenText:
			result.WriteString(token.Value)
			i++

		case TokenVariable:
			// Evaluate the variable
			value, err := EvaluateVariable(token.Value, data)
			if err != nil || value == nil {
				// Try to parse as an expression
				expr, parseErr := ParseExpression(token.Value)
				if parseErr != nil {
					// Not an expression either, leave empty
					result.WriteString("")
				} else {
					// Evaluate the expression
					exprValue, evalErr := expr.Evaluate(data)
					if evalErr != nil {
						result.WriteString("")
					} else {
						result.WriteString(FormatValue(exprValue))
					}
				}
			} else {
				result.WriteString(FormatValue(value))
			}
			i++

		case TokenIf:
			// Process if statement
			rendered, nextIdx, err := processIfStatement(tokens, i, data)
			if err != nil {
				return "", i, err
			}
			result.WriteString(rendered)
			i = nextIdx

		case TokenUnless:
			// Process unless statement (inverted if)
			rendered, nextIdx, err := processUnlessStatement(tokens, i, data)
			if err != nil {
				return "", i, err
			}
			result.WriteString(rendered)
			i = nextIdx

		case TokenFor:
			// Process for loop statement
			rendered, nextIdx, err := processForStatement(tokens, i, data)
			if err != nil {
				return "", i, err
			}
			result.WriteString(rendered)
			i = nextIdx

		case TokenElse, TokenElsif:
			// These should be handled by their parent if/unless
			// If we encounter them here, we're at the end of a branch
			return result.String(), i, nil

		case TokenEnd:
			// End of a control structure
			return result.String(), i + 1, nil

		default:
			// Unknown token type - skip it
			i++
		}
	}

	return result.String(), i, nil
}

// processIfStatement processes an if statement and its branches
func processIfStatement(tokens []Token, startIdx int, data TemplateData) (string, int, error) {
	if startIdx >= len(tokens) || tokens[startIdx].Type != TokenIf {
		return "", startIdx, fmt.Errorf("expected if token at index %d", startIdx)
	}

	// Evaluate the if condition
	condition := tokens[startIdx].Value
	conditionResult, err := evaluateCondition(condition, data)
	if err != nil {
		return "", startIdx, fmt.Errorf("failed to evaluate if condition: %w", err)
	}

	// Find the branches (else/elsif) and end
	branches, endIdx, err := findIfBranches(tokens, startIdx)
	if err != nil {
		return "", startIdx, err
	}

	// Determine which branch to execute
	if conditionResult {
		// Execute the if branch (from startIdx+1 to first branch or end)
		branchStart := startIdx + 1
		branchEnd := endIdx
		if len(branches) > 0 {
			branchEnd = branches[0].index
		}

		result, _, err := processTokens(tokens[branchStart:branchEnd], 0, data)
		return result, endIdx + 1, err
	}

	// Check elsif branches
	for i, branch := range branches {
		switch branch.branchType {
		case "elsif":
			// Evaluate elsif condition
			elsifResult, err := evaluateCondition(branch.condition, data)
			if err != nil {
				return "", startIdx, fmt.Errorf("failed to evaluate elsif condition: %w", err)
			}

			if elsifResult {
				// Execute this elsif branch
				branchStart := branch.index + 1
				branchEnd := endIdx
				if i+1 < len(branches) {
					branchEnd = branches[i+1].index
				}

				result, _, err := processTokens(tokens[branchStart:branchEnd], 0, data)
				return result, endIdx + 1, err
			}
		case "else":
			// Execute else branch
			branchStart := branch.index + 1
			result, _, err := processTokens(tokens[branchStart:endIdx], 0, data)
			return result, endIdx + 1, err
		}
	}

	// No branch matched, return empty
	return "", endIdx + 1, nil
}

// processUnlessStatement processes an unless statement (inverted if)
func processUnlessStatement(tokens []Token, startIdx int, data TemplateData) (string, int, error) {
	if startIdx >= len(tokens) || tokens[startIdx].Type != TokenUnless {
		return "", startIdx, fmt.Errorf("expected unless token at index %d", startIdx)
	}

	// Evaluate the unless condition (inverted)
	condition := tokens[startIdx].Value
	conditionResult, err := evaluateCondition(condition, data)
	if err != nil {
		return "", startIdx, fmt.Errorf("failed to evaluate unless condition: %w", err)
	}

	// Find the else branch and end
	elseIdx := -1
	endIdx := -1
	depth := 1

	for i := startIdx + 1; i < len(tokens); i++ {
		switch tokens[i].Type {
		case TokenIf, TokenUnless, TokenFor:
			depth++
		case TokenElse:
			if depth == 1 && elseIdx == -1 {
				elseIdx = i
			}
		case TokenEnd:
			depth--
			if depth == 0 {
				endIdx = i
			}
		}
		if endIdx != -1 {
			break
		}
	}

	if endIdx == -1 {
		return "", startIdx, fmt.Errorf("no matching end for unless statement")
	}

	// Unless is inverted: execute if condition is false
	if !conditionResult {
		// Execute the unless branch
		branchStart := startIdx + 1
		branchEnd := endIdx
		if elseIdx != -1 {
			branchEnd = elseIdx
		}

		result, _, err := processTokens(tokens[branchStart:branchEnd], 0, data)
		return result, endIdx + 1, err
	} else if elseIdx != -1 {
		// Execute else branch
		result, _, err := processTokens(tokens[elseIdx+1:endIdx], 0, data)
		return result, endIdx + 1, err
	}

	// Condition was true, skip unless block
	return "", endIdx + 1, nil
}

// findIfBranches finds all elsif/else branches for an if statement
func findIfBranches(tokens []Token, startIdx int) ([]ifBranch, int, error) {
	var branches []ifBranch
	endIdx := -1
	depth := 1

	for i := startIdx + 1; i < len(tokens); i++ {
		if depth == 1 {
			switch tokens[i].Type {
			case TokenElsif:
				branches = append(branches, ifBranch{
					index:      i,
					branchType: "elsif",
					condition:  tokens[i].Value,
				})
			case TokenElse:
				branches = append(branches, ifBranch{
					index:      i,
					branchType: "else",
					condition:  "",
				})
			}
		}

		switch tokens[i].Type {
		case TokenIf, TokenUnless, TokenFor:
			depth++
		case TokenEnd:
			depth--
			if depth == 0 {
				endIdx = i
			}
		}

		if endIdx != -1 {
			break
		}
	}

	if endIdx == -1 {
		return nil, -1, fmt.Errorf("no matching end for if statement")
	}

	return branches, endIdx, nil
}

// ifBranch represents an elsif or else branch
type ifBranch struct {
	index      int
	branchType string
	condition  string
}

// processForStatement processes a for loop and returns the rendered result
func processForStatement(tokens []Token, startIdx int, data TemplateData) (string, int, error) {
	if startIdx >= len(tokens) || tokens[startIdx].Type != TokenFor {
		return "", startIdx, fmt.Errorf("expected for token at index %d", startIdx)
	}

	// Parse the for expression (e.g. "item in items" or "idx, item in items")
	forNode, err := parseForSyntax(tokens[startIdx].Value)
	if err != nil {
		return "", startIdx, fmt.Errorf("invalid for syntax: %w", err)
	}

	// Find the matching {{end}} by tracking nesting depth
	endIdx := -1
	depth := 1
	for i := startIdx + 1; i < len(tokens); i++ {
		switch tokens[i].Type {
		case TokenIf, TokenUnless, TokenFor:
			depth++
		case TokenEnd:
			depth--
			if depth == 0 {
				endIdx = i
			}
		}
		if endIdx != -1 {
			break
		}
	}

	if endIdx == -1 {
		return "", startIdx, fmt.Errorf("no matching end for for loop")
	}

	// Evaluate the collection
	collectionVal, err := forNode.Collection.Evaluate(data)
	if err != nil {
		return "", endIdx + 1, fmt.Errorf("failed to evaluate for collection: %w", err)
	}

	items, err := toSlice(collectionVal)
	if err != nil {
		return "", endIdx + 1, fmt.Errorf("failed to convert collection to slice: %w", err)
	}

	// Extract body tokens (between for and end)
	bodyTokens := tokens[startIdx+1 : endIdx]

	// Iterate and render
	var result strings.Builder
	for idx, item := range items {
		loopData := newChildTemplateData(data, 2)
		loopData[forNode.Variable] = item
		if forNode.IndexVar != "" {
			loopData[forNode.IndexVar] = idx
		}

		rendered, _, err := processTokens(bodyTokens, 0, loopData)
		if err != nil {
			return "", startIdx, err
		}
		result.WriteString(rendered)
	}

	return result.String(), endIdx + 1, nil
}

// evaluateCondition evaluates a condition expression
func evaluateCondition(condition string, data TemplateData) (bool, error) {
	// Parse and evaluate the condition
	expr, err := ParseExpression(condition)
	if err != nil {
		return false, fmt.Errorf("failed to parse condition: %w", err)
	}

	result, err := expr.Evaluate(data)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	// Convert result to boolean
	return isTruthy(result), nil
}

// RenderTableWithControlStructures renders a table with support for loops and conditionals
func RenderTableWithControlStructures(table *Table, data TemplateData, ctx *renderContext) (*Table, error) {
	rendered := &Table{
		Properties: table.Properties,
		Grid:       table.Grid,
	}

	// Process each row
	i := 0
	for i < len(table.Rows) {
		row := &table.Rows[i]

		// Check if this row contains control structures in its first cell
		controlType, controlContent := render.DetectTableRowControlStructure(row)

		switch controlType {
		case "for":
			// Find matching end
			endIdx, err := render.FindMatchingTableEnd(table.Rows, i)
			if err != nil {
				return nil, fmt.Errorf("no matching end for table for loop: %w", err)
			}

			// Render for loop
			renderedRows, err := renderTableForLoop(table.Rows[i:endIdx+1], controlContent, data, ctx)
			if err != nil {
				return nil, err
			}
			rendered.Rows = append(rendered.Rows, renderedRows...)
			i = endIdx + 1

		case "if":
			// Find matching else/elsif/end
			endIdx, branches, err := render.FindMatchingTableIfEnd(table.Rows, i)
			if err != nil {
				return nil, fmt.Errorf("no matching end for table if: %w", err)
			}

			// Adjust branch indices to be relative to the slice
			adjustedBranches := make([]render.ElseBranch, len(branches))
			for idx, branch := range branches {
				adjustedBranches[idx] = render.ElseBranch{
					Index:      branch.Index - i,
					BranchType: branch.BranchType,
					Condition:  branch.Condition,
				}
			}

			// Render if/elsif/else
			renderedRows, err := renderTableIfElse(table.Rows[i:endIdx+1], controlContent, adjustedBranches, data, ctx)
			if err != nil {
				return nil, err
			}
			rendered.Rows = append(rendered.Rows, renderedRows...)
			i = endIdx + 1

		case "unless":
			// Find matching else/elsif/end
			endIdx, branches, err := render.FindMatchingTableIfEnd(table.Rows, i)
			if err != nil {
				return nil, fmt.Errorf("no matching end for table unless: %w", err)
			}

			// Adjust branch indices to be relative to the slice
			adjustedBranches := make([]render.ElseBranch, len(branches))
			for idx, branch := range branches {
				adjustedBranches[idx] = render.ElseBranch{
					Index:      branch.Index - i,
					BranchType: branch.BranchType,
					Condition:  branch.Condition,
				}
			}

			// Render unless/elsif/else (unless is inverted if)
			renderedRows, err := renderTableUnlessElse(table.Rows[i:endIdx+1], controlContent, adjustedBranches, data, ctx)
			if err != nil {
				return nil, err
			}
			rendered.Rows = append(rendered.Rows, renderedRows...)
			i = endIdx + 1

		case "else", "elsif", "elseif", "elif", "end":
			// Skip control structure rows - they shouldn't be in output
			i++

		default:
			// Regular row, render normally
			renderedRow, err := RenderTableRow(row, data, ctx)
			if err != nil {
				return nil, err
			}
			rendered.Rows = append(rendered.Rows, *renderedRow)
			i++
		}
	}

	return rendered, nil
}

// RenderTableRow renders a single table row
func RenderTableRow(row *TableRow, data TemplateData, ctx *renderContext) (*TableRow, error) {
	rendered := &TableRow{
		Properties: row.Properties,
	}

	// Render each cell
	for _, cell := range row.Cells {
		renderedCell, err := RenderTableCell(&cell, data, ctx)
		if err != nil {
			return nil, err
		}
		// Ensure cell has at least one paragraph (Word requirement)
		if len(renderedCell.Paragraphs) == 0 {
			renderedCell.Paragraphs = append(renderedCell.Paragraphs, Paragraph{})
		}
		rendered.Cells = append(rendered.Cells, *renderedCell)
	}

	return rendered, nil
}

// RenderTableCell renders a table cell
func RenderTableCell(cell *TableCell, data TemplateData, ctx *renderContext) (*TableCell, error) {
	rendered := &TableCell{
		Properties: cell.Properties,
	}

	// Convert paragraphs to BodyElements so we can handle multi-paragraph control structures
	elements := make([]BodyElement, len(cell.Paragraphs))
	for i := range cell.Paragraphs {
		elements[i] = &cell.Paragraphs[i]
	}

	// Use renderElementsWithContext to handle control structures that span multiple paragraphs
	renderedElements, err := renderElementsWithContext(elements, data, ctx)
	if err != nil {
		return nil, err
	}

	// Convert back to paragraphs
	for _, elem := range renderedElements {
		if para, ok := elem.(*Paragraph); ok {
			rendered.Paragraphs = append(rendered.Paragraphs, *para)
		}
	}

	return rendered, nil
}

// renderTableForLoop renders a for loop in a table
func renderTableForLoop(rows []TableRow, forExpr string, data TemplateData, ctx *renderContext) ([]TableRow, error) {
	// Parse for syntax
	forNode, err := parseForSyntax(strings.TrimSpace(forExpr))
	if err != nil {
		return nil, fmt.Errorf("invalid for syntax: %w", err)
	}

	// Evaluate collection
	collection, err := forNode.Collection.Evaluate(data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate collection: %w", err)
	}

	// Convert to slice
	items, err := toSlice(collection)
	if err != nil {
		return nil, fmt.Errorf("failed to convert collection to slice: %w", err)
	}

	// Collect body rows (skip first and last row which contain for/end)
	bodyRows := rows[1 : len(rows)-1]

	var result []TableRow

	// Iterate over collection
	for idx, item := range items {
		loopData := newChildTemplateData(data, 2)
		loopData[forNode.Variable] = item
		if forNode.IndexVar != "" {
			loopData[forNode.IndexVar] = idx
		}

		// Process body rows with loop data
		i := 0
		for i < len(bodyRows) {
			row := &bodyRows[i]
			controlType, controlContent := render.DetectTableRowControlStructure(row)

			switch controlType {
			case "for":
				// Find matching end for nested for loop
				endIdx, err := render.FindMatchingTableEndInSlice(bodyRows, i)
				if err != nil {
					return nil, fmt.Errorf("failed to find matching end for nested for: %w", err)
				}

				// Render nested for loop block
				renderedRows, err := renderTableForLoop(bodyRows[i:endIdx+1], controlContent, loopData, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, renderedRows...)
				i = endIdx + 1

			case "if":
				// Find matching else/elsif/end
				endIdx, branches, err := render.FindMatchingTableIfEndInSlice(bodyRows, i)
				if err != nil {
					return nil, fmt.Errorf("failed to find matching end for nested if: %w", err)
				}

				// Adjust branch indices to be relative to the slice
				adjustedBranches := make([]render.ElseBranch, len(branches))
				for idx, branch := range branches {
					adjustedBranches[idx] = render.ElseBranch{
						Index:      branch.Index - i,
						BranchType: branch.BranchType,
						Condition:  branch.Condition,
					}
				}

				// Render if/elsif/else block
				renderedRows, err := renderTableIfElse(bodyRows[i:endIdx+1], controlContent, adjustedBranches, loopData, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, renderedRows...)
				i = endIdx + 1

			case "unless":
				// Find matching else/elsif/end
				endIdx, branches, err := render.FindMatchingTableIfEndInSlice(bodyRows, i)
				if err != nil {
					return nil, fmt.Errorf("failed to find matching end for nested unless: %w", err)
				}

				// Adjust branch indices to be relative to the slice
				adjustedBranches := make([]render.ElseBranch, len(branches))
				for idx, branch := range branches {
					adjustedBranches[idx] = render.ElseBranch{
						Index:      branch.Index - i,
						BranchType: branch.BranchType,
						Condition:  branch.Condition,
					}
				}

				// Render unless/elsif/else block
				renderedRows, err := renderTableUnlessElse(bodyRows[i:endIdx+1], controlContent, adjustedBranches, loopData, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, renderedRows...)
				i = endIdx + 1

			default:
				// Regular row, render with loop data
				renderedRow, err := RenderTableRow(row, loopData, ctx)
				if err != nil {
					return nil, err
				}
				result = append(result, *renderedRow)
				i++
			}
		}
	}

	return result, nil
}

// renderTableIfElse renders an if/elsif/else in a table
func renderTableIfElse(rows []TableRow, ifExpr string, branches []render.ElseBranch, data TemplateData, ctx *renderContext) ([]TableRow, error) {
	// Parse condition
	cond, err := ParseExpression(ifExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse if condition: %w", err)
	}

	// Evaluate condition
	condResult, err := cond.Evaluate(data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate if condition: %w", err)
	}

	var bodyRows []TableRow

	if isTruthy(condResult) {
		// Use if branch
		if len(branches) > 0 {
			bodyRows = rows[1:branches[0].Index]
		} else {
			bodyRows = rows[1 : len(rows)-1]
		}
	} else {
		// Check elsif branches
		branchFound := false
		for i, branch := range branches {
			if branch.BranchType == "elsif" {
				// Evaluate elsif condition
				elsifCond, err := ParseExpression(branch.Condition)
				if err != nil {
					return nil, fmt.Errorf("failed to parse elsif condition: %w", err)
				}

				elsifResult, err := elsifCond.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate elsif condition: %w", err)
				}

				if isTruthy(elsifResult) {
					// Use this elsif branch
					var branchEnd int
					if i+1 < len(branches) {
						branchEnd = branches[i+1].Index
					} else {
						branchEnd = len(rows) - 1
					}
					bodyRows = rows[branch.Index+1 : branchEnd]
					branchFound = true
					break
				}
			} else if branch.BranchType == "else" && !branchFound {
				// Use else branch
				bodyRows = rows[branch.Index+1 : len(rows)-1]
				branchFound = true
				break
			}
		}
	}

	// Render selected rows, handling nested control structures
	var result []TableRow
	i := 0
	for i < len(bodyRows) {
		row := &bodyRows[i]
		controlType, controlContent := render.DetectTableRowControlStructure(row)

		switch controlType {
		case "for":
			// Find matching end for nested for loop
			endIdx, err := render.FindMatchingTableEndInSlice(bodyRows, i)
			if err != nil {
				return nil, fmt.Errorf("failed to find matching end for nested for: %w", err)
			}

			// Render nested for loop block
			renderedRows, err := renderTableForLoop(bodyRows[i:endIdx+1], controlContent, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, renderedRows...)
			i = endIdx + 1

		case "if":
			// Find matching else/elsif/end for nested if
			endIdx, nestedBranches, err := render.FindMatchingTableIfEndInSlice(bodyRows, i)
			if err != nil {
				return nil, fmt.Errorf("failed to find matching end for nested if: %w", err)
			}

			// Adjust branch indices to be relative to the slice
			adjustedBranches := make([]render.ElseBranch, len(nestedBranches))
			for idx, branch := range nestedBranches {
				adjustedBranches[idx] = render.ElseBranch{
					Index:      branch.Index - i,
					BranchType: branch.BranchType,
					Condition:  branch.Condition,
				}
			}

			// Render nested if/elsif/else block
			renderedRows, err := renderTableIfElse(bodyRows[i:endIdx+1], controlContent, adjustedBranches, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, renderedRows...)
			i = endIdx + 1

		case "unless":
			// Find matching else/elsif/end for nested unless
			endIdx, nestedBranches, err := render.FindMatchingTableIfEndInSlice(bodyRows, i)
			if err != nil {
				return nil, fmt.Errorf("failed to find matching end for nested unless: %w", err)
			}

			// Adjust branch indices to be relative to the slice
			adjustedBranches := make([]render.ElseBranch, len(nestedBranches))
			for idx, branch := range nestedBranches {
				adjustedBranches[idx] = render.ElseBranch{
					Index:      branch.Index - i,
					BranchType: branch.BranchType,
					Condition:  branch.Condition,
				}
			}

			// Render nested unless/elsif/else block
			renderedRows, err := renderTableUnlessElse(bodyRows[i:endIdx+1], controlContent, adjustedBranches, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, renderedRows...)
			i = endIdx + 1

		default:
			// Regular row, render with data
			renderedRow, err := RenderTableRow(row, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, *renderedRow)
			i++
		}
	}

	return result, nil
}

// renderTableUnlessElse renders an unless/elsif/else in a table (inverted if)
func renderTableUnlessElse(rows []TableRow, unlessExpr string, branches []render.ElseBranch, data TemplateData, ctx *renderContext) ([]TableRow, error) {
	// Parse condition
	cond, err := ParseExpression(unlessExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse unless condition: %w", err)
	}

	// Evaluate condition
	condResult, err := cond.Evaluate(data)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate unless condition: %w", err)
	}

	var bodyRows []TableRow

	// Unless is inverted: render unless branch if condition is falsy
	if !isTruthy(condResult) {
		// Use unless branch
		if len(branches) > 0 {
			bodyRows = rows[1:branches[0].Index]
		} else {
			bodyRows = rows[1 : len(rows)-1]
		}
	} else {
		// Check elsif branches (evaluated when unless condition is true)
		branchFound := false
		for i, branch := range branches {
			if branch.BranchType == "elsif" {
				// Evaluate elsif condition
				elsifCond, err := ParseExpression(branch.Condition)
				if err != nil {
					return nil, fmt.Errorf("failed to parse elsif condition: %w", err)
				}

				elsifResult, err := elsifCond.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate elsif condition: %w", err)
				}

				if isTruthy(elsifResult) {
					// Use this elsif branch
					var branchEnd int
					if i+1 < len(branches) {
						branchEnd = branches[i+1].Index
					} else {
						branchEnd = len(rows) - 1
					}
					bodyRows = rows[branch.Index+1 : branchEnd]
					branchFound = true
					break
				}
			} else if branch.BranchType == "else" && !branchFound {
				// Use else branch
				bodyRows = rows[branch.Index+1 : len(rows)-1]
				branchFound = true
				break
			}
		}
	}

	// Render selected rows, handling nested control structures
	var result []TableRow
	i := 0
	for i < len(bodyRows) {
		row := &bodyRows[i]
		controlType, controlContent := render.DetectTableRowControlStructure(row)

		switch controlType {
		case "for":
			// Find matching end for nested for loop
			endIdx, err := render.FindMatchingTableEndInSlice(bodyRows, i)
			if err != nil {
				return nil, fmt.Errorf("failed to find matching end for nested for: %w", err)
			}

			// Render nested for loop block
			renderedRows, err := renderTableForLoop(bodyRows[i:endIdx+1], controlContent, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, renderedRows...)
			i = endIdx + 1

		case "if":
			// Find matching else/elsif/end for nested if
			endIdx, nestedBranches, err := render.FindMatchingTableIfEndInSlice(bodyRows, i)
			if err != nil {
				return nil, fmt.Errorf("failed to find matching end for nested if: %w", err)
			}

			// Adjust branch indices to be relative to the slice
			adjustedBranches := make([]render.ElseBranch, len(nestedBranches))
			for idx, branch := range nestedBranches {
				adjustedBranches[idx] = render.ElseBranch{
					Index:      branch.Index - i,
					BranchType: branch.BranchType,
					Condition:  branch.Condition,
				}
			}

			// Render nested if/elsif/else block
			renderedRows, err := renderTableIfElse(bodyRows[i:endIdx+1], controlContent, adjustedBranches, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, renderedRows...)
			i = endIdx + 1

		case "unless":
			// Find matching else/elsif/end for nested unless
			endIdx, nestedBranches, err := render.FindMatchingTableIfEndInSlice(bodyRows, i)
			if err != nil {
				return nil, fmt.Errorf("failed to find matching end for nested unless: %w", err)
			}

			// Adjust branch indices to be relative to the slice
			adjustedBranches := make([]render.ElseBranch, len(nestedBranches))
			for idx, branch := range nestedBranches {
				adjustedBranches[idx] = render.ElseBranch{
					Index:      branch.Index - i,
					BranchType: branch.BranchType,
					Condition:  branch.Condition,
				}
			}

			// Render nested unless/elsif/else block
			renderedRows, err := renderTableUnlessElse(bodyRows[i:endIdx+1], controlContent, adjustedBranches, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, renderedRows...)
			i = endIdx + 1

		default:
			// Regular row, render with data
			renderedRow, err := RenderTableRow(row, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, *renderedRow)
			i++
		}
	}

	return result, nil
}
