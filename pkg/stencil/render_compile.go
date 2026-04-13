package stencil

import (
	"strings"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

type paragraphRenderPlanMode int

const (
	paragraphRenderPlanStatic paragraphRenderPlanMode = iota
	paragraphRenderPlanVariables
	paragraphRenderPlanControl
)

type paragraphRenderPlan struct {
	mode                  paragraphRenderPlanMode
	fullText              string
	useLegacyRunRendering bool
	hasActualControlNodes bool
	mergedParagraph       *Paragraph
	legacyRuns            []Run
	controlStructures     []ControlStructure
}

func buildTemplateParagraphPlans(doc *Document, fragments map[string]*fragment) map[*Paragraph]*paragraphRenderPlan {
	plans := make(map[*Paragraph]*paragraphRenderPlan)

	if doc != nil && doc.Body != nil {
		collectParagraphPlansFromBody(doc.Body, plans)
	}

	for _, frag := range fragments {
		if frag == nil || frag.parsed == nil || frag.parsed.Body == nil {
			continue
		}
		collectParagraphPlansFromBody(frag.parsed.Body, plans)
	}

	return plans
}

func collectParagraphPlansFromBody(body *Body, plans map[*Paragraph]*paragraphRenderPlan) {
	if body == nil {
		return
	}

	for _, elem := range body.Elements {
		switch e := elem.(type) {
		case *Paragraph:
			plans[e] = compileParagraphRenderPlan(e)
		case *Table:
			collectParagraphPlansFromTable(e, plans)
		}
	}
}

func collectParagraphPlansFromTable(table *Table, plans map[*Paragraph]*paragraphRenderPlan) {
	if table == nil {
		return
	}

	for rowIdx := range table.Rows {
		for cellIdx := range table.Rows[rowIdx].Cells {
			cell := &table.Rows[rowIdx].Cells[cellIdx]
			for paraIdx := range cell.Paragraphs {
				para := &cell.Paragraphs[paraIdx]
				plans[para] = compileParagraphRenderPlan(para)
			}
		}
	}
}

func resolveParagraphRenderPlan(para *Paragraph, ctx *renderContext) *paragraphRenderPlan {
	if para == nil {
		return nil
	}
	if ctx != nil && ctx.paragraphPlans != nil {
		if plan, ok := ctx.paragraphPlans[para]; ok {
			return plan
		}
	}
	return compileParagraphRenderPlan(para)
}

func compileParagraphRenderPlan(para *Paragraph) *paragraphRenderPlan {
	if para == nil {
		return nil
	}

	fullText := buildParagraphRenderText(para)
	if !strings.Contains(fullText, "{{") {
		return &paragraphRenderPlan{
			mode:     paragraphRenderPlanStatic,
			fullText: fullText,
		}
	}

	plan := &paragraphRenderPlan{
		mode:     paragraphRenderPlanVariables,
		fullText: fullText,
	}

	if len(para.Content) > 0 {
		hasProofErr := false
		hasHyperlink := false
		for _, content := range para.Content {
			switch content.(type) {
			case *ProofErr:
				hasProofErr = true
			case *Hyperlink:
				hasHyperlink = true
			}
		}

		plan.useLegacyRunRendering = hasProofErr && !hasHyperlink
		if plan.useLegacyRunRendering {
			baseRuns := para.Runs
			if len(baseRuns) == 0 {
				baseRuns = make([]Run, 0, len(para.Content))
				for _, content := range para.Content {
					if run, ok := content.(*Run); ok {
						baseRuns = append(baseRuns, *run)
					}
				}
			}

			plan.legacyRuns = cloneRunsForLegacyRendering(baseRuns)
			legacyPara := &Paragraph{Runs: plan.legacyRuns}
			render.MergeConsecutiveRuns(legacyPara)
			plan.legacyRuns = legacyPara.Runs
		}
	}

	plan.mergedParagraph = cloneParagraph(para)
	render.MergeConsecutiveRuns(plan.mergedParagraph)

	tokens := Tokenize(fullText)
	hasControlStructures := false
	for _, token := range tokens {
		switch token.Type {
		case TokenIf, TokenFor, TokenUnless, TokenElse, TokenElsif, TokenEnd, TokenInclude:
			hasControlStructures = true
		}
	}

	if !hasControlStructures {
		return plan
	}

	structures, err := ParseControlStructures(fullText)
	if err != nil || len(structures) == 0 {
		return plan
	}

	for _, structure := range structures {
		switch structure.(type) {
		case *IfNode, *ForNode, *UnlessNode, *IncludeNode:
			plan.hasActualControlNodes = true
		}
	}

	if plan.hasActualControlNodes {
		plan.mode = paragraphRenderPlanControl
		plan.controlStructures = structures
	}

	return plan
}
