package stencil

import "github.com/benjaminschreck/go-stencil/pkg/stencil/render"

type bodyRenderPlan struct {
	body    *Body
	entries []bodyRenderPlanEntry
}

type bodyRenderPlanEntry struct {
	controlType    string
	controlContent string
	endIdx         int
	branches       []bodyRenderBranch
	forNode        *ForNode
	conditionExpr  ExpressionNode
	includeExpr    ExpressionNode
}

type bodyRenderBranch struct {
	index      int
	branchType string
	condition  string
	expr       ExpressionNode
}

type openBodyControl struct {
	index       int
	controlType string
}

func buildTemplateBodyPlans(doc *Document, fragments map[string]*fragment) map[*Body]*bodyRenderPlan {
	plans := make(map[*Body]*bodyRenderPlan)

	if doc != nil && doc.Body != nil {
		plans[doc.Body] = compileBodyRenderPlan(doc.Body)
	}

	for _, frag := range fragments {
		if frag == nil || frag.parsed == nil || frag.parsed.Body == nil {
			continue
		}
		plans[frag.parsed.Body] = compileBodyRenderPlan(frag.parsed.Body)
	}

	return plans
}

func resolveBodyRenderPlan(body *Body, ctx *renderContext) *bodyRenderPlan {
	if body == nil {
		return nil
	}
	if ctx != nil && ctx.bodyPlans != nil {
		if plan, ok := ctx.bodyPlans[body]; ok {
			return plan
		}
	}
	return compileBodyRenderPlan(body)
}

func compileBodyRenderPlan(body *Body) *bodyRenderPlan {
	if body == nil {
		return nil
	}

	plan := &bodyRenderPlan{
		body:    body,
		entries: make([]bodyRenderPlanEntry, len(body.Elements)),
	}
	for i := range plan.entries {
		plan.entries[i].endIdx = -1
	}

	stack := make([]openBodyControl, 0, 8)

	for i, elem := range body.Elements {
		para, ok := elem.(*Paragraph)
		if !ok {
			continue
		}

		controlType, controlContent := render.DetectControlStructure(para)
		entry := &plan.entries[i]
		entry.controlType = controlType
		entry.controlContent = controlContent

		switch controlType {
		case "for":
			if forNode, err := parseForSyntax(controlContent); err == nil {
				entry.forNode = forNode
			}
			stack = append(stack, openBodyControl{index: i, controlType: controlType})
		case "if", "unless":
			if expr, err := ParseExpression(controlContent); err == nil {
				entry.conditionExpr = expr
			}
			stack = append(stack, openBodyControl{index: i, controlType: controlType})
		case "include":
			if expr, err := ParseExpression(controlContent); err == nil {
				entry.includeExpr = expr
			}
		case "elsif", "elseif", "elif":
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			if open.controlType != "if" && open.controlType != "unless" {
				continue
			}
			branch := bodyRenderBranch{
				index:      i,
				branchType: "elsif",
				condition:  controlContent,
			}
			if expr, err := ParseExpression(controlContent); err == nil {
				branch.expr = expr
			}
			plan.entries[open.index].branches = append(plan.entries[open.index].branches, branch)
		case "else":
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			if open.controlType != "if" && open.controlType != "unless" {
				continue
			}
			plan.entries[open.index].branches = append(plan.entries[open.index].branches, bodyRenderBranch{
				index:      i,
				branchType: "else",
			})
		case "end":
			if len(stack) == 0 {
				continue
			}
			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			plan.entries[open.index].endIdx = i
		}
	}

	return plan
}

