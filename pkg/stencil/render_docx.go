package stencil

import (
	"fmt"
	"strings"
)


// elseBranch represents an else/elsif branch in an if statement
type elseBranch struct {
	index      int    // Index of the branch paragraph
	branchType string // "else", "elsif", "elif", "elseif"
	condition  string // Condition for elsif branches
}


// findMatchingEndInElements finds the matching {{end}} for a control structure in elements
func findMatchingEndInElements(elements []BodyElement, startIdx int) (int, error) {
	depth := 1
	for i := startIdx + 1; i < len(elements); i++ {
		if para, ok := elements[i].(*Paragraph); ok {
			controlType, _ := detectControlStructure(para)
			switch controlType {
			case "for", "if", "unless":
				depth++
			case "end":
				depth--
				if depth == 0 {
					return i, nil
				}
			}
		}
	}
	return -1, fmt.Errorf("no matching end found")
}

// findIfStructureInElements finds the structure of an if statement including elsif/else branches
func findIfStructureInElements(elements []BodyElement, startIdx int) (endIdx int, branches []elseBranch, err error) {
	depth := 1
	branches = []elseBranch{}

	for i := startIdx + 1; i < len(elements); i++ {
		if para, ok := elements[i].(*Paragraph); ok {
			controlType, condition := detectControlStructure(para)

			if depth == 1 {
				switch controlType {
				case "elsif", "elseif", "elif":
					branches = append(branches, elseBranch{
						index:      i,
						branchType: "elsif",
						condition:  condition,
					})
				case "else":
					branches = append(branches, elseBranch{
						index:      i,
						branchType: "else",
						condition:  "",
					})
				}
			}

			switch controlType {
			case "for", "if", "unless":
				depth++
			case "end":
				depth--
				if depth == 0 {
					return i, branches, nil
				}
			}
		}
	}

	return -1, nil, fmt.Errorf("no matching end found")
}

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

// mergeConsecutiveRuns merges consecutive runs in a paragraph to handle split template variables
func mergeConsecutiveRuns(para *Paragraph) {
	// Handle paragraphs with Content (which can contain hyperlinks)
	if len(para.Content) > 0 {
		mergeConsecutiveRunsWithContent(para)
		return
	}
	
	// Handle legacy paragraphs with only Runs array
	if len(para.Runs) <= 1 {
		return
	}

	var mergedRuns []Run
	var currentRun *Run

	for i, run := range para.Runs {
		if i == 0 {
			// Start with first run
			newRun := run
			currentRun = &newRun
			continue
		}

		// Check if this run can be merged with the previous one
		// Only merge if the current run has text AND no break
		if run.Text != nil && run.Break == nil && currentRun != nil && currentRun.Text != nil {
			// Merge text content
			currentRun.Text.Content += run.Text.Content
		} else {
			// Save current run and start new one
			if currentRun != nil {
				mergedRuns = append(mergedRuns, *currentRun)
			}
			
			// If this run has both break and text, we need to split it
			if run.Break != nil && run.Text != nil {
				// First add the break
				breakRun := Run{
					Properties: run.Properties,
					Break:      run.Break,
				}
				mergedRuns = append(mergedRuns, breakRun)
				
				// Then create a new run with just the text
				textRun := Run{
					Properties: run.Properties,
					Text:       run.Text,
				}
				currentRun = &textRun
			} else {
				newRun := run
				currentRun = &newRun
			}
		}
	}

	// Don't forget the last run
	if currentRun != nil {
		mergedRuns = append(mergedRuns, *currentRun)
	}

	para.Runs = mergedRuns
}

// mergeConsecutiveRunsWithContent merges runs while preserving hyperlink boundaries
func mergeConsecutiveRunsWithContent(para *Paragraph) {
	if len(para.Content) == 0 {
		return
	}
	
	var mergedContent []interface{}
	var pendingRuns []Run
	
	// Helper function to merge pending runs
	mergePendingRuns := func() {
		if len(pendingRuns) == 0 {
			return
		}
		
		merged := mergeRunSlice(pendingRuns)
		for _, run := range merged {
			r := run // Create a new variable to avoid aliasing
			mergedContent = append(mergedContent, &r)
		}
		pendingRuns = nil
	}
	
	// Process content elements
	for _, content := range para.Content {
		switch c := content.(type) {
		case *Run:
			// Accumulate runs outside hyperlinks
			pendingRuns = append(pendingRuns, *c)
			
		case *Hyperlink:
			// First, merge any pending runs
			mergePendingRuns()
			
			// Process hyperlink runs separately
			if len(c.Runs) > 1 {
				mergedHyperlinkRuns := mergeRunSlice(c.Runs)
				h := *c // Copy hyperlink
				h.Runs = mergedHyperlinkRuns
				mergedContent = append(mergedContent, &h)
			} else {
				mergedContent = append(mergedContent, c)
			}
		}
	}
	
	// Merge any remaining runs
	mergePendingRuns()
	
	// Update paragraph content
	para.Content = make([]ParagraphContent, len(mergedContent))
	for i, content := range mergedContent {
		para.Content[i] = content.(ParagraphContent)
	}
	
	// Also update the legacy Runs array for compatibility
	para.Runs = nil
	for _, content := range para.Content {
		if run, ok := content.(*Run); ok {
			para.Runs = append(para.Runs, *run)
		}
	}
}

// mergeRunSlice merges a slice of runs
func mergeRunSlice(runs []Run) []Run {
	if len(runs) <= 1 {
		return runs
	}
	
	var merged []Run
	var current *Run
	
	for i, run := range runs {
		if i == 0 {
			newRun := run
			current = &newRun
			continue
		}
		
		// Only merge text runs without breaks
		if run.Text != nil && run.Break == nil && current != nil && current.Text != nil {
			current.Text.Content += run.Text.Content
		} else {
			if current != nil {
				merged = append(merged, *current)
			}
			
			// Handle runs with both break and text
			if run.Break != nil && run.Text != nil {
				// First add the break
				breakRun := Run{
					Properties: run.Properties,
					Break:      run.Break,
				}
				merged = append(merged, breakRun)
				
				// Then create a new run with just the text
				textRun := Run{
					Properties: run.Properties,
					Text:       run.Text,
				}
				current = &textRun
			} else {
				newRun := run
				current = &newRun
			}
		}
	}
	
	if current != nil {
		merged = append(merged, *current)
	}
	
	return merged
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
			mergeConsecutiveRuns(&p)
			body.Elements[i] = &p // Update the element with merged runs
		case *Table:
			// Also merge runs in table cells
			t := *el // Create a copy
			for rowIdx, row := range t.Rows {
				for cellIdx, cell := range row.Cells {
					for paraIdx, para := range cell.Paragraphs {
						p := para // Create a copy
						mergeConsecutiveRuns(&p)
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
			controlType, controlContent := detectControlStructure(&para)

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
				endIdx, err := findMatchingEndInElements(body.Elements, i)
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
				endIdx, elseBranches, err := findIfStructureInElements(body.Elements, i)
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
						branchEnd = elseBranches[0].index
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
						if branch.branchType == "elsif" || branch.branchType == "elif" || branch.branchType == "elseif" {
							expr, err := ParseExpression(branch.condition)
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
									branchEnd = elseBranches[j+1].index
								} else {
									branchEnd = endIdx
								}

								branchBody := body.Elements[branch.index+1 : branchEnd]
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
						} else if branch.branchType == "else" && !branchRendered {
							// Render else branch
							branchBody := body.Elements[branch.index+1 : endIdx]
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
				endIdx, elseBranches, err := findIfStructureInElements(body.Elements, i)
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
					if len(elseBranches) > 0 && elseBranches[0].branchType == "else" {
						branchEnd = elseBranches[0].index
					} else {
						branchEnd = endIdx
					}

					branchBody := body.Elements[i+1 : branchEnd]
					branchElements, err := renderElementsWithContext(branchBody, data, ctx)
					if err != nil {
						return nil, err
					}
					rendered.Elements = append(rendered.Elements, branchElements...)
				} else if len(elseBranches) > 0 && elseBranches[0].branchType == "else" {
					// Render else branch
					branchBody := body.Elements[elseBranches[0].index+1 : endIdx]
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

					// Render the fragment body with the current data context
					renderedBody, err := RenderBodyWithControlStructures(frag.parsed.Body, data, ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to render fragment %s: %w", fragmentName, err)
					}

					// Append the rendered fragment elements
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

// detectControlStructure checks if a paragraph contains a control structure
func detectControlStructure(para *Paragraph) (string, string) {
	// Get the text content of the paragraph
	text := getParagraphText(para)
	text = strings.TrimSpace(text)

	// Check for control structures
	if strings.Contains(text, "{{for ") && strings.Contains(text, "{{end}}") {
		// Handle inline for loop (e.g., "{{for item in items}} content {{end}}")
		return "inline-for", text
	}

	// Check for inline if statements
	if strings.Contains(text, "{{if ") && strings.Contains(text, "{{end}}") {
		// This paragraph contains a complete if statement
		// Let RenderParagraphWithContext handle it
		return "", ""
	}

	// Check for inline unless statements
	if strings.Contains(text, "{{unless ") && strings.Contains(text, "{{end}}") {
		// This paragraph contains a complete unless statement
		// Let RenderParagraphWithContext handle it
		return "", ""
	}

	// Check if text starts with a control structure (even if other content follows)
	if strings.HasPrefix(text, "{{for ") {
		// Extract just the for part
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[5:endIdx] // Remove {{for
			return "for", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{if ") {
		// Extract just the if part
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[4:endIdx] // Remove {{if
			return "if", strings.TrimSpace(content)
		}
	}

	// Check if text contains a for structure (not necessarily at the start)
	if idx := strings.Index(text, "{{for "); idx >= 0 && !strings.Contains(text, "{{end}}") {
		// Extract the for content
		startIdx := idx + 6 // Skip "{{for "
		endIdx := strings.Index(text[startIdx:], "}}")
		if endIdx > 0 {
			content := text[startIdx:startIdx+endIdx]
			return "for", strings.TrimSpace(content)
		}
	}

	// Check if text contains an if structure (not necessarily at the start)
	if idx := strings.Index(text, "{{if "); idx >= 0 && !strings.Contains(text, "{{end}}") {
		// Extract the if condition
		startIdx := idx + 5 // Skip "{{if "
		endIdx := strings.Index(text[startIdx:], "}}")
		if endIdx > 0 {
			content := text[startIdx:startIdx+endIdx]
			return "if", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{unless ") {
		// Extract just the unless part
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[8:endIdx] // Remove {{unless
			return "unless", strings.TrimSpace(content)
		}
	}

	// Check if text contains an unless structure (not necessarily at the start)
	if idx := strings.Index(text, "{{unless "); idx >= 0 && !strings.Contains(text, "{{end}}") {
		// Extract the unless condition
		startIdx := idx + 9 // Skip "{{unless "
		endIdx := strings.Index(text[startIdx:], "}}")
		if endIdx > 0 {
			content := text[startIdx:startIdx+endIdx]
			return "unless", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{end}}") {
		return "end", ""
	}

	if strings.HasPrefix(text, "{{else}}") {
		return "else", ""
	}

	// Check for elsif/elseif/elif variants
	if strings.HasPrefix(text, "{{elsif ") {
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[8:endIdx] // Remove {{elsif
			return "elsif", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{elseif ") {
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[9:endIdx] // Remove {{elseif
			return "elseif", strings.TrimSpace(content)
		}
	}

	if strings.HasPrefix(text, "{{elif ") {
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[7:endIdx] // Remove {{elif
			return "elif", strings.TrimSpace(content)
		}
	}

	// Check for include directive
	if strings.HasPrefix(text, "{{include ") {
		endIdx := strings.Index(text, "}}")
		if endIdx > 0 {
			content := text[10:endIdx] // Remove {{include
			return "include", strings.TrimSpace(content)
		}
	}

	return "", ""
}

// getParagraphText extracts all text from a paragraph
func getParagraphText(para *Paragraph) string {
	var result strings.Builder
	for _, run := range para.Runs {
		if run.Text != nil {
			result.WriteString(run.Text.Content)
		}
	}
	return result.String()
}


// Removed - using existing toSlice from control.go

// renderInlineForLoop handles loops that are entirely within one paragraph
func renderInlineForLoop(para *Paragraph, loopText string, data TemplateData, ctx *renderContext) ([]Paragraph, error) {
	// Extract the for syntax and body
	// Format: "{{for item in items}} content {{end}}"
	forStart := strings.Index(loopText, "{{for ")
	forEnd := strings.Index(loopText[forStart:], "}}") + forStart + 2
	endStart := strings.Index(loopText, "{{end}}")

	if forStart < 0 || forEnd < 0 || endStart < 0 {
		return nil, fmt.Errorf("invalid inline for loop syntax")
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
	resultText.WriteString(prefix)

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

	resultText.WriteString(suffix)

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

// processTemplateText processes template variables in text
func processTemplateText(text string, data TemplateData) (string, error) {
	// Tokenize the text
	tokens := Tokenize(text)

	var result strings.Builder
	for _, token := range tokens {
		switch token.Type {
		case TokenText:
			result.WriteString(token.Value)
		case TokenVariable:
			// Evaluate the variable
			value, err := EvaluateVariable(token.Value, data)
			if err != nil {
				// If variable not found, leave empty
				result.WriteString("")
			} else {
				result.WriteString(FormatValue(value))
			}
		default:
			// Leave other tokens as-is for now
			result.WriteString("{{")
			result.WriteString(token.Value)
			result.WriteString("}}")
		}
	}

	return result.String(), nil
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
		controlType, controlContent := detectTableRowControlStructure(row)

		switch controlType {
		case "for":
			// Find matching end
			endIdx, err := findMatchingTableEnd(table.Rows, i)
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
			// Find matching else/end
			elseIdx, endIdx, err := findMatchingTableIfEnd(table.Rows, i)
			if err != nil {
				return nil, fmt.Errorf("no matching end for table if: %w", err)
			}

			// Render if/else
			renderedRows, err := renderTableIfElse(table.Rows[i:endIdx+1], controlContent, elseIdx-i, data, ctx)
			if err != nil {
				return nil, err
			}
			rendered.Rows = append(rendered.Rows, renderedRows...)
			i = endIdx + 1

		case "else", "end":
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

// detectTableRowControlStructure checks if a table row contains control structures
func detectTableRowControlStructure(row *TableRow) (string, string) {
	if len(row.Cells) == 0 || len(row.Cells[0].Paragraphs) == 0 {
		return "", ""
	}

	// Check first paragraph of first cell
	return detectControlStructure(&row.Cells[0].Paragraphs[0])
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
		mergeConsecutiveRuns(&p)

		// Check if this paragraph contains an inline for loop
		controlType, controlContent := detectControlStructure(&p)

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

// findMatchingTableEnd finds the matching end for a table control structure
func findMatchingTableEnd(rows []TableRow, startIdx int) (int, error) {
	depth := 1
	for i := startIdx + 1; i < len(rows); i++ {
		controlType, _ := detectTableRowControlStructure(&rows[i])

		switch controlType {
		case "for", "if":
			depth++
		case "end":
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("no matching end found")
}

// findMatchingTableIfEnd finds the matching else/end for a table if
func findMatchingTableIfEnd(rows []TableRow, startIdx int) (int, int, error) {
	depth := 1
	elseIdx := -1

	for i := startIdx + 1; i < len(rows); i++ {
		controlType, _ := detectTableRowControlStructure(&rows[i])

		switch controlType {
		case "for", "if":
			depth++
		case "else":
			if depth == 1 && elseIdx == -1 {
				elseIdx = i
			}
		case "end":
			depth--
			if depth == 0 {
				return elseIdx, i, nil
			}
		}
	}
	return -1, -1, fmt.Errorf("no matching end found")
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
			controlType, controlContent := detectTableRowControlStructure(row)

			switch controlType {
			case "for":
				// Find matching end for nested for loop
				endIdx, err := findMatchingTableEndInSlice(bodyRows, i)
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
				// Find matching else/end
				elseIdx, endIdx, err := findMatchingTableIfEndInSlice(bodyRows, i)
				if err != nil {
					return nil, fmt.Errorf("failed to find matching end for nested if: %w", err)
				}

				// Render if/else block
				renderedRows, err := renderTableIfElse(bodyRows[i:endIdx+1], controlContent, elseIdx-i, loopData, ctx)
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

// findMatchingTableIfEndInSlice finds matching else/end in a slice of rows
// findMatchingTableEndInSlice finds the matching end for a table control structure in a slice
func findMatchingTableEndInSlice(rows []TableRow, startIdx int) (int, error) {
	depth := 1
	for i := startIdx + 1; i < len(rows); i++ {
		controlType, _ := detectTableRowControlStructure(&rows[i])

		switch controlType {
		case "for", "if":
			depth++
		case "end":
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("no matching end found")
}

func findMatchingTableIfEndInSlice(rows []TableRow, startIdx int) (int, int, error) {
	depth := 1
	elseIdx := -1

	for i := startIdx + 1; i < len(rows); i++ {
		controlType, _ := detectTableRowControlStructure(&rows[i])

		switch controlType {
		case "for", "if":
			depth++
		case "else":
			if depth == 1 && elseIdx == -1 {
				elseIdx = i
			}
		case "end":
			depth--
			if depth == 0 {
				return elseIdx, i, nil
			}
		}
	}
	return -1, -1, fmt.Errorf("no matching end found")
}

// renderTableIfElse renders an if/else in a table
func renderTableIfElse(rows []TableRow, ifExpr string, elseIdx int, data TemplateData, ctx *renderContext) ([]TableRow, error) {
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
		if elseIdx > 0 {
			bodyRows = rows[1:elseIdx]
		} else {
			bodyRows = rows[1 : len(rows)-1]
		}
	} else if elseIdx > 0 {
		// Use else branch
		bodyRows = rows[elseIdx+1 : len(rows)-1]
	}

	// Render selected rows
	var result []TableRow
	for _, row := range bodyRows {
		renderedRow, err := RenderTableRow(&row, data, ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, *renderedRow)
	}

	return result, nil
}
