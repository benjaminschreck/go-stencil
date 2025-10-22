package stencil

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

// renderElementsWithContext renders a slice of elements with the given context
func renderElementsWithContext(elements []BodyElement, data TemplateData, ctx *renderContext) ([]BodyElement, error) {
	// Render elements without processing control structures again
	// This is called from within control structure processing, so we just render individual elements
	result := make([]BodyElement, 0, len(elements))
	
	for _, elem := range elements {
		switch el := elem.(type) {
		case *Paragraph:
			para := *el
			rendered, err := RenderParagraphWithContext(&para, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, rendered)
			
		case *Table:
			table := *el
			rendered, err := RenderTableWithControlStructures(&table, data, ctx)
			if err != nil {
				return nil, err
			}
			result = append(result, rendered)
			
		default:
			// For unknown elements, keep as-is
			result = append(result, elem)
		}
	}
	
	return result, nil
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
	// First, merge runs in all paragraphs to handle split template variables
	for i, elem := range body.Elements {
		switch el := elem.(type) {
		case *Paragraph:
			p := *el // Create a copy
			render.MergeConsecutiveRuns(&p)
			body.Elements[i] = &p // Update the element with merged runs
		case *Table:
			// Also merge runs in table cells
			t := *el // Create a copy
			for rowIdx, row := range t.Rows {
				for cellIdx, cell := range row.Cells {
					for paraIdx, para := range cell.Paragraphs {
						p := para // Create a copy
						render.MergeConsecutiveRuns(&p)
						t.Rows[rowIdx].Cells[cellIdx].Paragraphs[paraIdx] = p
					}
				}
			}
			body.Elements[i] = &t // Update the element with merged runs
		}
	}

	rendered := &Body{
		Elements:          make([]BodyElement, 0),
		SectionProperties: body.SectionProperties, // Preserve section properties
	}

	// Process elements in order
	i := 0
	for i < len(body.Elements) {
		
		elem := body.Elements[i]

		switch el := elem.(type) {
		case *Paragraph:
			para := *el

			// Check if this paragraph contains a control structure
			controlType, controlContent := render.DetectControlStructure(&para)

			switch controlType {
			case "inline-for":
				// Handle inline for loop (entire loop in one paragraph)
				renderedParas, err := renderInlineForLoop(&para, controlContent, data, ctx)
				if err != nil {
					return nil, err
				}
				for _, p := range renderedParas {
					rendered.Elements = append(rendered.Elements, &p)
				}
				i++

			case "for":
				// Handle for loop
				endIdx, err := render.FindMatchingEndInElements(body.Elements, i)
				if err != nil {
					return nil, fmt.Errorf("no matching {{end}} for {{for}} at element %d", i)
				}

				// Parse for loop syntax
				forNode, err := parseForSyntax(controlContent)
				if err != nil {
					return nil, fmt.Errorf("invalid for syntax: %w", err)
				}

				// Get the loop body (elements between for and end)
				loopBody := body.Elements[i+1 : endIdx]

				// Evaluate the collection
				collection, err := forNode.Collection.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate collection: %w", err)
				}

				// Iterate over collection
				items, err := toSlice(collection)
				if err != nil {
					return nil, fmt.Errorf("failed to convert collection to slice: %w", err)
				}

				for idx, item := range items {
					// Create new data context for loop iteration
					loopData := make(TemplateData)
					for k, v := range data {
						loopData[k] = v
					}
					loopData[forNode.Variable] = item
					if forNode.IndexVar != "" {
						loopData[forNode.IndexVar] = idx
					}

					// Render loop body
					loopRendered, err := renderElementsWithContext(loopBody, loopData, ctx)
					if err != nil {
						return nil, err
					}
					rendered.Elements = append(rendered.Elements, loopRendered...)
				}

				// Skip to after the end marker
				if endIdx >= 0 && endIdx < len(body.Elements) {
					i = endIdx + 1
				} else {
					// This should not happen if findMatchingEndInElements worked correctly
					return nil, fmt.Errorf("invalid endIdx %d for for loop at element %d", endIdx, i)
				}

			case "if":
				// Find runs before the {{if}} statement
				var prefixRuns []Run
				ifFound := false
				for _, run := range para.Runs {
					if run.Text != nil && strings.Contains(run.Text.Content, "{{if ") {
						// Check if there's text before {{if}} in this run
						ifIndex := strings.Index(run.Text.Content, "{{if ")
						if ifIndex > 0 {
							// Split this run - keep the prefix
							prefixRun := Run{
								Properties: run.Properties,
								Text: &Text{Content: run.Text.Content[:ifIndex]},
							}
							prefixRuns = append(prefixRuns, prefixRun)
						}
						ifFound = true
						break
					} else if !ifFound {
						// This run comes before the {{if}}, keep it entirely
						prefixRuns = append(prefixRuns, run)
					}
				}

				// Handle if statement
				endIdx, elseBranches, err := render.FindIfStructureInElements(body.Elements, i)
				if err != nil {
					return nil, fmt.Errorf("no matching {{end}} for {{if}} at element %d: %w", i, err)
				}

				// Parse if condition
				expr, err := ParseExpression(controlContent)
				if err != nil {
					return nil, fmt.Errorf("failed to parse if condition: %w", err)
				}

				// Evaluate condition
				condValue, err := expr.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate if condition: %w", err)
				}

				branchRendered := false

				if isTruthy(condValue) {
					// Render the if branch
					var branchEnd int
					if len(elseBranches) > 0 {
						branchEnd = elseBranches[0].Index
					} else {
						branchEnd = endIdx
					}

					branchBody := body.Elements[i+1 : branchEnd]
					branchElements, err := renderElementsWithContext(branchBody, data, ctx)
					if err != nil {
						return nil, err
					}
					
					// If there were runs before the {{if}}, prepend them to the first element
					if len(prefixRuns) > 0 && len(branchElements) > 0 {
						if firstPara, ok := branchElements[0].(*Paragraph); ok {
							// Create a new paragraph with the prefix runs
							newPara := &Paragraph{
								Properties: firstPara.Properties,
							}
							
							// Add all prefix runs (including any line breaks)
							newPara.Runs = append(newPara.Runs, prefixRuns...)
							
							// Add all runs from the first paragraph
							newPara.Runs = append(newPara.Runs, firstPara.Runs...)
							
							// Replace the first element
							branchElements[0] = newPara
						} else if len(prefixRuns) > 0 {
							// If the first element is not a paragraph, create a new paragraph with the prefix
							prefixPara := &Paragraph{
								Properties: para.Properties,
								Runs:       prefixRuns,
							}
							// Insert the prefix paragraph at the beginning
							branchElements = append([]BodyElement{prefixPara}, branchElements...)
						}
					}
					
					rendered.Elements = append(rendered.Elements, branchElements...)
					branchRendered = true
				} else {
					// Check elsif branches
					for j, branch := range elseBranches {
						if branch.BranchType == "elsif" || branch.BranchType == "elif" || branch.BranchType == "elseif" {
							expr, err := ParseExpression(branch.Condition)
							if err != nil {
								return nil, fmt.Errorf("failed to parse elsif condition: %w", err)
							}

							condValue, err := expr.Evaluate(data)
							if err != nil {
								return nil, fmt.Errorf("failed to evaluate elsif condition: %w", err)
							}

							if isTruthy(condValue) {
								// Render this elsif branch
								var branchEnd int
								if j+1 < len(elseBranches) {
									branchEnd = elseBranches[j+1].Index
								} else {
									branchEnd = endIdx
								}

								branchBody := body.Elements[branch.Index+1 : branchEnd]
								branchElements, err := renderElementsWithContext(branchBody, data, ctx)
								if err != nil {
									return nil, err
								}
								
								// If there were runs before the {{if}}, prepend them to the first element
								if len(prefixRuns) > 0 && len(branchElements) > 0 {
									if firstPara, ok := branchElements[0].(*Paragraph); ok {
										// Create a new paragraph with the prefix runs
										newPara := &Paragraph{
											Properties: firstPara.Properties,
										}
										
										// Add all prefix runs (including any line breaks)
										newPara.Runs = append(newPara.Runs, prefixRuns...)
										
										// Add all runs from the first paragraph
										newPara.Runs = append(newPara.Runs, firstPara.Runs...)
										
										// Replace the first element
										branchElements[0] = newPara
									} else if len(prefixRuns) > 0 {
										// If the first element is not a paragraph, create a new paragraph with the prefix
										prefixPara := &Paragraph{
											Properties: para.Properties,
											Runs:       prefixRuns,
										}
										// Insert the prefix paragraph at the beginning
										branchElements = append([]BodyElement{prefixPara}, branchElements...)
									}
								}
								
								rendered.Elements = append(rendered.Elements, branchElements...)
								branchRendered = true
								break
							}
						} else if branch.BranchType == "else" && !branchRendered {
							// Render else branch
							branchBody := body.Elements[branch.Index+1 : endIdx]
							branchElements, err := renderElementsWithContext(branchBody, data, ctx)
							if err != nil {
								return nil, err
							}
							
							// If there were runs before the {{if}}, prepend them to the first element
							if len(prefixRuns) > 0 && len(branchElements) > 0 {
								if firstPara, ok := branchElements[0].(*Paragraph); ok {
									// Create a new paragraph with the prefix runs
									newPara := &Paragraph{
										Properties: firstPara.Properties,
									}
									
									// Add all prefix runs (including any line breaks)
									newPara.Runs = append(newPara.Runs, prefixRuns...)
									
									// Add all runs from the first paragraph
									newPara.Runs = append(newPara.Runs, firstPara.Runs...)
									
									// Replace the first element
									branchElements[0] = newPara
								} else if len(prefixRuns) > 0 {
									// If the first element is not a paragraph, create a new paragraph with the prefix
									prefixPara := &Paragraph{
										Properties: para.Properties,
										Runs:       prefixRuns,
									}
									// Insert the prefix paragraph at the beginning
									branchElements = append([]BodyElement{prefixPara}, branchElements...)
								}
							}
							
							rendered.Elements = append(rendered.Elements, branchElements...)
							break
						}
					}
				}

				// Skip to after the end marker
				if endIdx >= 0 && endIdx < len(body.Elements) {
					i = endIdx + 1
				} else {
					// This should not happen if findIfStructureInElements worked correctly
					return nil, fmt.Errorf("invalid endIdx %d for if statement at element %d", endIdx, i)
				}

			case "unless":
				// Handle unless statement (similar to if but inverted)
				endIdx, elseBranches, err := render.FindIfStructureInElements(body.Elements, i)
				if err != nil {
					return nil, fmt.Errorf("no matching {{end}} for {{unless}} at element %d", i)
				}

				// Parse unless condition
				expr, err := ParseExpression(controlContent)
				if err != nil {
					return nil, fmt.Errorf("failed to parse unless condition: %w", err)
				}

				// Evaluate condition
				condValue, err := expr.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate unless condition: %w", err)
				}

				// Unless renders if condition is falsy (opposite of if)
				if !isTruthy(condValue) {
					// Render the unless branch
					var branchEnd int
					if len(elseBranches) > 0 && elseBranches[0].BranchType == "else" {
						branchEnd = elseBranches[0].Index
					} else {
						branchEnd = endIdx
					}

					branchBody := body.Elements[i+1 : branchEnd]
					branchElements, err := renderElementsWithContext(branchBody, data, ctx)
					if err != nil {
						return nil, err
					}
					rendered.Elements = append(rendered.Elements, branchElements...)
				} else if len(elseBranches) > 0 && elseBranches[0].BranchType == "else" {
					// Render else branch
					branchBody := body.Elements[elseBranches[0].Index+1 : endIdx]
					branchElements, err := renderElementsWithContext(branchBody, data, ctx)
					if err != nil {
						return nil, err
					}
					rendered.Elements = append(rendered.Elements, branchElements...)
				}

				// Skip to after the end marker
				if endIdx >= 0 && endIdx < len(body.Elements) {
					i = endIdx + 1
				} else {
					// This should not happen if findIfStructureInElements worked correctly
					return nil, fmt.Errorf("invalid endIdx %d for unless statement at element %d", endIdx, i)
				}

			case "include":
				// Handle include directive
				// Parse the fragment name expression
				expr, err := ParseExpression(controlContent)
				if err != nil {
					return nil, fmt.Errorf("failed to parse include expression: %w", err)
				}

				// Evaluate the fragment name
				fragmentNameValue, err := expr.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate fragment name: %w", err)
				}

				fragmentName, ok := fragmentNameValue.(string)
				if !ok {
					return nil, fmt.Errorf("fragment name must be a string, got %T", fragmentNameValue)
				}

				// Get fragments from context
				if ctx.fragments == nil {
					return nil, fmt.Errorf("fragments not available in render context")
				}

				// Find the fragment
				frag, exists := ctx.fragments[fragmentName]
				if !exists {
					return nil, fmt.Errorf("fragment not found: %s", fragmentName)
				}

				// Render the fragment content
				if frag.parsed != nil && frag.parsed.Body != nil {
					// Push fragment to stack for circular reference detection
					ctx.fragmentStack = append(ctx.fragmentStack, fragmentName)
					defer func() {
						ctx.fragmentStack = ctx.fragmentStack[:len(ctx.fragmentStack)-1]
					}()

					// Check for circular references
					for _, f := range ctx.fragmentStack[:len(ctx.fragmentStack)-1] {
						if f == fragmentName {
							return nil, fmt.Errorf("circular fragment reference detected: %s", fragmentName)
						}
					}

					// Check render depth
					maxDepth := 10
					if ctx.renderDepth > 0 {
						maxDepth = ctx.renderDepth
					}
					if len(ctx.fragmentStack) > maxDepth {
						return nil, fmt.Errorf("maximum render depth exceeded")
					}

					// Render the fragment body first
					renderedBody, err := RenderBodyWithControlStructures(frag.parsed.Body, data, ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to render fragment %s: %w", fragmentName, err)
					}

					// Handle fragment resources (media files and relationships) AFTER rendering
					if frag.isDocx && len(frag.relationships) > 0 {
						// Allocate ID range for this fragment (if not already allocated)
						rangeStart, exists := ctx.fragmentIDAllocations[fragmentName]
						if !exists {
							rangeStart = ctx.nextFragmentIDRange
							ctx.fragmentIDAllocations[fragmentName] = rangeStart
							ctx.nextFragmentIDRange += FragmentIDRangeSize
						}

						// Add fragment resources only once
						if !ctx.fragmentResourcesAdded[fragmentName] {
							imageCounter := 1

							for _, rel := range frag.relationships {
								// Only process media relationships (images, videos, etc.)
								// Skip other relationships (headers, footers, styles, etc.) as they're not part of the fragment content
								if !isMediaRelationship(rel) {
									continue
								}

								// Extract ID number from rId6 → 6
								idNum, err := extractRelationshipNumber(rel.ID)
								if err != nil {
									return nil, fmt.Errorf("invalid relationship ID %s in fragment %s: %w", rel.ID, fragmentName, err)
								}

								// Check if ID fits in allocated range
								if idNum >= FragmentIDRangeSize {
									return nil, fmt.Errorf("fragment %s relationship ID %s exceeds range size %d",
										fragmentName, rel.ID, FragmentIDRangeSize)
								}

								// Create new relationship with offset ID
								newID := fmt.Sprintf("rId%d", rangeStart+idNum)
								newTarget := renameMediaPath(rel.Target, fragmentName, imageCounter)

								// Copy media file with new name
								if mediaContent, ok := frag.mediaFiles[rel.Target]; ok {
									newFilename := filepath.Base(newTarget)
									ctx.fragmentMedia[newFilename] = mediaContent
								}

								newRel := Relationship{
									ID:     newID,
									Type:   rel.Type,
									Target: newTarget,
								}

								ctx.fragmentRelationships = append(ctx.fragmentRelationships, newRel)
								imageCounter++
							}

							// Mark this fragment's resources as added
							ctx.fragmentResourcesAdded[fragmentName] = true
						}

						// Build ID mapping for XML updates (always needed, even on second inclusion)
						// Only remap media relationship IDs
						idMap := make(map[string]string)
						for _, rel := range frag.relationships {
							if !isMediaRelationship(rel) {
								continue
							}
							idNum, _ := extractRelationshipNumber(rel.ID)
							newID := fmt.Sprintf("rId%d", rangeStart+idNum)
							idMap[rel.ID] = newID
						}

						// Update relationship IDs in the rendered body
						tempDoc := &Document{Body: renderedBody}
						updateDocumentRelationshipIDs(tempDoc, idMap)
					}

					// Append the rendered (and ID-updated) fragment elements
					rendered.Elements = append(rendered.Elements, renderedBody.Elements...)
				}
				i++

			case "end":
				// Unmatched end marker - this should not happen in well-formed templates
				return nil, fmt.Errorf("unmatched {{end}} at element %d", i)

			default:
				// Regular paragraph, render normally
				renderedPara, err := RenderParagraphWithContext(&para, data, ctx)
				if err != nil {
					return nil, err
				}
				rendered.Elements = append(rendered.Elements, renderedPara)
				i++
			}

		case *Table:
			// Render table with control structures
			renderedTable, err := RenderTableWithControlStructures(el, data, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to render table: %w", err)
			}
			rendered.Elements = append(rendered.Elements, renderedTable)
			i++
		}
	}


	return rendered, nil
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
		// Create loop context
		loopData := make(TemplateData)
		for k, v := range data {
			loopData[k] = v
		}
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
		case TokenIf, TokenUnless:
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
		case TokenIf, TokenUnless:
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

	// Render each paragraph in the cell
	for _, para := range cell.Paragraphs {
		// Create a copy of the paragraph and merge consecutive runs
		// This is necessary because Word often splits template expressions across multiple runs
		p := para
		render.MergeConsecutiveRuns(&p)

		// Check if this paragraph contains an inline for loop
		controlType, controlContent := render.DetectControlStructure(&p)

		if controlType == "inline-for" {
			// Handle inline for loop - this will expand to multiple paragraphs
			renderedParas, err := renderInlineForLoop(&p, controlContent, data, ctx)
			if err != nil {
				return nil, err
			}
			// Add all expanded paragraphs
			for _, rp := range renderedParas {
				rendered.Paragraphs = append(rendered.Paragraphs, rp)
			}
		} else {
			// Normal paragraph rendering
			renderedPara, err := RenderParagraphWithContext(&p, data, ctx)
			if err != nil {
				return nil, err
			}
			rendered.Paragraphs = append(rendered.Paragraphs, *renderedPara)
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
		// Create loop context
		loopData := make(TemplateData)
		for k, v := range data {
			loopData[k] = v
		}
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
