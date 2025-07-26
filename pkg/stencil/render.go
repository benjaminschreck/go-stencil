package stencil

import (
	"fmt"
	"regexp"
	"strings"
)

// RenderDocument renders a document with the given data
func RenderDocument(doc *Document, data TemplateData) (*Document, error) {
	return RenderDocumentWithContext(doc, data, nil)
}

// RenderDocumentWithContext renders a document with the given data and context
func RenderDocumentWithContext(doc *Document, data TemplateData, ctx *renderContext) (*Document, error) {
	// Create a deep copy of the document
	rendered := &Document{
		XMLName: doc.XMLName,
	}
	
	if doc.Body != nil {
		body, err := RenderBodyWithContext(doc.Body, data, ctx)
		if err != nil {
			return nil, err
		}
		rendered.Body = body
	}
	
	return rendered, nil
}

// RenderBody renders a document body
func RenderBody(body *Body, data TemplateData) (*Body, error) {
	return RenderBodyWithContext(body, data, nil)
}

// RenderBodyWithContext renders a document body with context
func RenderBodyWithContext(body *Body, data TemplateData, ctx *renderContext) (*Body, error) {
	// Use the new control structure aware rendering
	return RenderBodyWithControlStructures(body, data, ctx)
}

// RenderParagraph renders a paragraph
func RenderParagraph(para *Paragraph, data TemplateData) (*Paragraph, error) {
	return RenderParagraphWithContext(para, data, nil)
}

// RenderParagraphWithContext renders a paragraph with context
func RenderParagraphWithContext(para *Paragraph, data TemplateData, ctx *renderContext) (*Paragraph, error) {
	// First check if the paragraph contains control structures
	// by getting the full text content including line breaks
	fullText := ""
	
	// Use Content if available, otherwise fall back to Runs
	if len(para.Content) > 0 {
		for _, content := range para.Content {
			switch c := content.(type) {
			case *Run:
				if c.Text != nil {
					fullText += c.Text.Content
				} else if c.Break != nil {
					fullText += "\n"
				}
			case *Hyperlink:
				// Get text from hyperlink runs
				for _, run := range c.Runs {
					if run.Text != nil {
						fullText += run.Text.Content
					} else if run.Break != nil {
						fullText += "\n"
					}
				}
			}
		}
	} else {
		// Fall back to legacy fields
		for _, run := range para.Runs {
			if run.Text != nil {
				fullText += run.Text.Content
			} else if run.Break != nil {
				// Include line breaks in the full text
				fullText += "\n"
			}
		}
		// Also check hyperlinks
		for _, hyperlink := range para.Hyperlinks {
			for _, run := range hyperlink.Runs {
				if run.Text != nil {
					fullText += run.Text.Content
				} else if run.Break != nil {
					fullText += "\n"
				}
			}
		}
	}
	
	// Check if we have control structures
	// Note: We only check for actual control structure tokens, not variable tokens
	tokens := Tokenize(fullText)
	hasControlStructures := false
	for _, token := range tokens {
		switch token.Type {
		case TokenIf, TokenFor, TokenUnless, TokenElse, TokenElsif, TokenEnd:
			hasControlStructures = true
		}
	}
	
	// If we have control structures, parse and render them
	// Only use control structure processing when actual control structures are present
	if hasControlStructures {
		// DEBUG: This should not happen for simple variable substitution
		// fmt.Printf("DEBUG: Control structure processing for text: %q\n", fullText)
		// Parse the control structures
		structures, err := ParseControlStructures(fullText)
		// Only proceed if we successfully parsed control structures
		if err == nil && len(structures) > 0 {
			// Check if we actually have control structure nodes (not just text/expression nodes)
			hasActualControlStructures := false
			for _, structure := range structures {
				switch structure.(type) {
				case *IfNode, *ForNode, *UnlessNode:
					hasActualControlStructures = true
				}
			}
			
			// Only use control structure rendering if we have actual control structures
			if hasActualControlStructures {
				// This should not be reached for simple variable substitution
				// Render the control structures
				renderedText, err := renderControlBodyWithContext(structures, data, ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to render control structures in paragraph: %w", err)
				}
			
				// Create a new paragraph with the rendered text
				rendered := &Paragraph{
					Properties: para.Properties,
				}
				
				// Convert the rendered text into runs, preserving line breaks
				if len(para.Runs) > 0 {
					// Split the rendered text by newlines to preserve line breaks
					lines := strings.Split(renderedText, "\n")
					
					for i, line := range lines {
						if line != "" {
							// Create a text run for this line
							textRun := Run{
								Properties: para.Runs[0].Properties,
								Text: &Text{
									Content: line,
								},
							}
							
							// Check if the run contains OOXML fragments that need to be expanded
							if ooxmlFragmentRegex.MatchString(line) {
								// Process the run to handle OOXML fragments
								expandedRuns, err := expandOOXMLFragments(&textRun, data, ctx)
								if err != nil {
									return nil, err
								}
								rendered.Runs = append(rendered.Runs, expandedRuns...)
							} else {
								rendered.Runs = append(rendered.Runs, textRun)
							}
						}
						
						// Add a line break run between lines (but not after the last line)
						if i < len(lines)-1 {
							breakRun := Run{
								Properties: para.Runs[0].Properties,
								Break:      &Break{},
							}
							rendered.Runs = append(rendered.Runs, breakRun)
						}
					}
				}
				
				return rendered, nil
			}
		}
	}
	
	// Otherwise, render normally
	rendered := &Paragraph{
		Properties: para.Properties,
	}
	
	// Use Content if available to preserve order of runs and hyperlinks
	if len(para.Content) > 0 {
		for _, content := range para.Content {
			switch c := content.(type) {
			case *Run:
				renderedRun, err := RenderRunWithContext(c, data, ctx)
				if err != nil {
					return nil, err
				}
				
				// Check if the run contains OOXML fragments that need to be expanded
				if renderedRun.Text != nil && ooxmlFragmentRegex.MatchString(renderedRun.Text.Content) {
					// Process the run to handle OOXML fragments
					expandedRuns, err := expandOOXMLFragments(renderedRun, data, ctx)
					if err != nil {
						return nil, err
					}
					for _, expRun := range expandedRuns {
						rendered.Content = append(rendered.Content, &expRun)
						rendered.Runs = append(rendered.Runs, expRun)
					}
				} else {
					rendered.Content = append(rendered.Content, renderedRun)
					rendered.Runs = append(rendered.Runs, *renderedRun)
				}
			case *Hyperlink:
				renderedHyperlink, err := RenderHyperlinkWithContext(c, data, ctx)
				if err != nil {
					return nil, err
				}
				rendered.Content = append(rendered.Content, renderedHyperlink)
				rendered.Hyperlinks = append(rendered.Hyperlinks, *renderedHyperlink)
			}
		}
	} else {
		// Fall back to legacy fields
		// Render runs
		for _, run := range para.Runs {
			renderedRun, err := RenderRunWithContext(&run, data, ctx)
			if err != nil {
				return nil, err
			}
			
			// Check if the run contains OOXML fragments that need to be expanded
			if renderedRun.Text != nil && ooxmlFragmentRegex.MatchString(renderedRun.Text.Content) {
				// Process the run to handle OOXML fragments
				expandedRuns, err := expandOOXMLFragments(renderedRun, data, ctx)
				if err != nil {
					return nil, err
				}
				rendered.Runs = append(rendered.Runs, expandedRuns...)
			} else {
				rendered.Runs = append(rendered.Runs, *renderedRun)
			}
		}
		
		// Render hyperlinks
		for _, hyperlink := range para.Hyperlinks {
			renderedHyperlink, err := RenderHyperlinkWithContext(&hyperlink, data, ctx)
			if err != nil {
				return nil, err
			}
			rendered.Hyperlinks = append(rendered.Hyperlinks, *renderedHyperlink)
		}
	}
	
	return rendered, nil
}

// ooxmlFragmentRegex matches OOXML fragment placeholders
var ooxmlFragmentRegex = regexp.MustCompile(`\{\{OOXML_FRAGMENT:([^}]+)\}\}`)

// RenderRun renders a run of text
func RenderRun(run *Run, data TemplateData) (*Run, error) {
	return RenderRunWithContext(run, data, nil)
}

// RenderRunWithContext renders a run of text with context
func RenderRunWithContext(run *Run, data TemplateData, ctx *renderContext) (*Run, error) {
	rendered := &Run{
		Properties: run.Properties,
		Break:      run.Break,
	}
	
	if run.Text != nil {
		renderedText, err := RenderTextWithContext(run.Text, data, ctx)
		if err != nil {
			return nil, err
		}
		rendered.Text = renderedText
	}
	
	return rendered, nil
}

// RenderHyperlinkWithContext renders a hyperlink with context
func RenderHyperlinkWithContext(hyperlink *Hyperlink, data TemplateData, ctx *renderContext) (*Hyperlink, error) {
	rendered := &Hyperlink{
		ID:      hyperlink.ID,
		History: hyperlink.History,
	}
	
	// Render runs within the hyperlink
	for _, run := range hyperlink.Runs {
		renderedRun, err := RenderRunWithContext(&run, data, ctx)
		if err != nil {
			return nil, err
		}
		rendered.Runs = append(rendered.Runs, *renderedRun)
	}
	
	return rendered, nil
}

// convertXMLElementToRuns converts an XML element to DOCX runs
func convertXMLElementToRuns(elem XMLElement, templateRun *Run) []Run {
	var runs []Run
	
	switch elem.Type {
	case "text":
		// Simple text element
		if elem.Text != "" {
			textRun := Run{
				Properties: templateRun.Properties,
				Text: &Text{
					XMLName: templateRun.Text.XMLName,
					Space:   templateRun.Text.Space,
					Content: elem.Text,
				},
			}
			runs = append(runs, textRun)
		}
		
	case "element":
		// Check if it's a known OOXML element
		switch elem.Name.Local {
		case "br":
			// Line break or page break
			breakType := ""
			for _, attr := range elem.Attrs {
				if attr.Name.Local == "type" {
					breakType = attr.Value
				}
			}
			breakRun := Run{
				Properties: templateRun.Properties,
				Break:      &Break{Type: breakType},
			}
			runs = append(runs, breakRun)
			
		case "t":
			// Text element - extract text content
			textContent := extractTextFromXMLElement(elem)
			if textContent != "" {
				textRun := Run{
					Properties: templateRun.Properties,
					Text: &Text{
						XMLName: templateRun.Text.XMLName,
						Space:   templateRun.Text.Space,
						Content: textContent,
					},
				}
				runs = append(runs, textRun)
			}
			
		case "r":
			// Run element - process its children
			for _, child := range elem.Content {
				childRuns := convertXMLElementToRuns(child, templateRun)
				runs = append(runs, childRuns...)
			}
			
		default:
			// For other elements, process children
			for _, child := range elem.Content {
				childRuns := convertXMLElementToRuns(child, templateRun)
				runs = append(runs, childRuns...)
			}
		}
	}
	
	return runs
}

// extractTextFromXMLElement extracts all text content from an XML element and its children
func extractTextFromXMLElement(elem XMLElement) string {
	var text string
	
	if elem.Type == "text" {
		return elem.Text
	}
	
	for _, child := range elem.Content {
		text += extractTextFromXMLElement(child)
	}
	
	return text
}

// expandOOXMLFragments processes OOXML fragments in a run and returns multiple runs if needed
func expandOOXMLFragments(run *Run, data TemplateData, ctx *renderContext) ([]Run, error) {
	content := run.Text.Content
	
	// Find all OOXML fragment placeholders
	matches := ooxmlFragmentRegex.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return []Run{*run}, nil
	}
	
	var runs []Run
	lastEnd := 0
	
	for _, match := range matches {
		// match[0] and match[1] are the start and end of the full match
		// match[2] and match[3] are the start and end of the submatch (fragment type)
		fragmentStart := match[0]
		fragmentEnd := match[1]
		fragmentType := content[match[2]:match[3]]
		
		// Add any text before the fragment as a regular run
		if fragmentStart > lastEnd {
			beforeText := content[lastEnd:fragmentStart]
			if strings.TrimSpace(beforeText) != "" {
				textRun := Run{
					Properties: run.Properties,
					Text: &Text{
						XMLName: run.Text.XMLName,
						Space:   run.Text.Space,
						Content: beforeText,
					},
				}
				runs = append(runs, textRun)
			}
		}
		
		// Handle the fragment based on its key
		// Try to retrieve the actual fragment from context
		var fragmentContent interface{}
		if ctx != nil && ctx.ooxmlFragments != nil {
			fragmentContent = ctx.ooxmlFragments[fragmentType]
		}
		
		if fragmentContent != nil {
			switch content := fragmentContent.(type) {
			case *Break:
				// Page break
				breakRun := Run{
					Properties: run.Properties,
					Break:      content,
				}
				runs = append(runs, breakRun)
				
			case *HTMLRuns:
				// HTML runs - expand into multiple runs
				for _, htmlRun := range content.Runs {
					newRun := Run{
						Properties: htmlRun.Properties,
					}
					
					// Convert HTML run elements to text/breaks
					for _, elem := range htmlRun.Content {
						switch elem.Type {
						case "text":
							if newRun.Text == nil {
								newRun.Text = &Text{
									XMLName: run.Text.XMLName,
									Space:   run.Text.Space,
									Content: elem.Text,
								}
							} else {
								newRun.Text.Content += elem.Text
							}
						case "break":
							// If there's already text, create a new run for the break
							if newRun.Text != nil {
								runs = append(runs, newRun)
								newRun = Run{
									Properties: htmlRun.Properties,
									Break:      &Break{}, // Empty break for line break
								}
							} else {
								newRun.Break = &Break{} // Empty break for line break
							}
						}
					}
					
					// Add the final run if it has content
					if newRun.Text != nil || newRun.Break != nil {
						runs = append(runs, newRun)
					}
				}
				
			case *XMLFragment:
				// XML fragment - convert XML elements to runs
				for _, elem := range content.Elements {
					expandedRuns := convertXMLElementToRuns(elem, run)
					runs = append(runs, expandedRuns...)
				}
				
			default:
				// Unknown fragment type, preserve as text
				fragmentRun := Run{
					Properties: run.Properties,
					Text: &Text{
						XMLName: run.Text.XMLName,
						Space:   run.Text.Space,
						Content: fmt.Sprintf("{{OOXML_FRAGMENT:%s}}", fragmentType),
					},
				}
				runs = append(runs, fragmentRun)
			}
		} else {
			// Fragment not found in context - this might be a legacy placeholder
			// Check if it's one of the known types
			if fragmentType == "*stencil.Break" {
				// Legacy page break placeholder
				breakRun := Run{
					Properties: run.Properties,
					Break:      &Break{Type: "page"},
				}
				runs = append(runs, breakRun)
			} else {
				// Unknown fragment, preserve as text
				fragmentRun := Run{
					Properties: run.Properties,
					Text: &Text{
						XMLName: run.Text.XMLName,
						Space:   run.Text.Space,
						Content: fmt.Sprintf("{{OOXML_FRAGMENT:%s}}", fragmentType),
					},
				}
				runs = append(runs, fragmentRun)
			}
		}
		
		lastEnd = fragmentEnd
	}
	
	// Add any remaining text after the last fragment
	if lastEnd < len(content) {
		afterText := content[lastEnd:]
		if strings.TrimSpace(afterText) != "" {
			textRun := Run{
				Properties: run.Properties,
				Text: &Text{
					XMLName: run.Text.XMLName,
					Space:   run.Text.Space,
					Content: afterText,
				},
			}
			runs = append(runs, textRun)
		}
	}
	
	// If no runs were created, return the original run
	if len(runs) == 0 {
		return []Run{*run}, nil
	}
	
	return runs, nil
}

// RenderText renders text content with variable substitution
func RenderText(text *Text, data TemplateData) (*Text, error) {
	return RenderTextWithContext(text, data, nil)
}

// RenderTextWithContext renders text content with variable substitution and context
func RenderTextWithContext(text *Text, data TemplateData, ctx *renderContext) (*Text, error) {
	content := text.Content
	
	// Find and replace all template variables
	tokens := Tokenize(content)
	var result strings.Builder
	
	for _, token := range tokens {
		switch token.Type {
		case TokenText:
			result.WriteString(token.Value)
		case TokenVariable:
			// Try to parse as an expression first, fall back to simple variable evaluation
			if expr, err := ParseExpression(token.Value); err == nil {
				value, err := expr.Evaluate(data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate expression %s: %w", token.Value, err)
				}
				// Check if the value is an OOXML fragment that needs special handling
				if fragment, ok := value.(*OOXMLFragment); ok {
					// Store the fragment in context and create a placeholder
					var fragmentKey string
					if ctx != nil && ctx.ooxmlFragments != nil {
						fragmentKey = fmt.Sprintf("fragment_%d", len(ctx.ooxmlFragments))
						ctx.ooxmlFragments[fragmentKey] = fragment.Content
					} else {
						// Fallback to type-based placeholder when no context
						fragmentKey = fmt.Sprintf("%T", fragment.Content)
					}
					result.WriteString(fmt.Sprintf("{{OOXML_FRAGMENT:%s}}", fragmentKey))
				} else if marker, ok := value.(*TableRowMarker); ok {
					// Handle table row markers
					result.WriteString(fmt.Sprintf("{{TABLE_ROW_MARKER:%s}}", marker.Action))
				} else if marker, ok := value.(*TableColumnMarker); ok {
					// Handle table column markers
					result.WriteString(marker.String())
				} else if marker, ok := value.(LinkReplacementMarker); ok {
					// Handle link replacement markers
					if ctx != nil {
						markerKey := fmt.Sprintf("link_%d", len(ctx.linkMarkers))
						linkMarker := &marker
						ctx.linkMarkers[markerKey] = linkMarker
						result.WriteString(fmt.Sprintf("{{LINK_REPLACEMENT:%s}}", markerKey))
					} else {
						// If no context, we can't store the marker, so just write empty string
						result.WriteString("")
					}
				} else {
					result.WriteString(FormatValue(value))
				}
			} else {
				// Fall back to simple variable evaluation for backward compatibility
				value, err := EvaluateVariable(token.Value, data)
				if err != nil {
					return nil, fmt.Errorf("failed to evaluate variable %s: %w", token.Value, err)
				}
				// Check if the value is an OOXML fragment that needs special handling
				if fragment, ok := value.(*OOXMLFragment); ok {
					// Store the fragment in context and create a placeholder
					var fragmentKey string
					if ctx != nil && ctx.ooxmlFragments != nil {
						fragmentKey = fmt.Sprintf("fragment_%d", len(ctx.ooxmlFragments))
						ctx.ooxmlFragments[fragmentKey] = fragment.Content
					} else {
						// Fallback to type-based placeholder when no context
						fragmentKey = fmt.Sprintf("%T", fragment.Content)
					}
					result.WriteString(fmt.Sprintf("{{OOXML_FRAGMENT:%s}}", fragmentKey))
				} else if marker, ok := value.(*TableRowMarker); ok {
					// Handle table row markers
					result.WriteString(fmt.Sprintf("{{TABLE_ROW_MARKER:%s}}", marker.Action))
				} else if marker, ok := value.(*TableColumnMarker); ok {
					// Handle table column markers
					result.WriteString(marker.String())
				} else if marker, ok := value.(LinkReplacementMarker); ok {
					// Handle link replacement markers
					if ctx != nil {
						markerKey := fmt.Sprintf("link_%d", len(ctx.linkMarkers))
						linkMarker := &marker
						ctx.linkMarkers[markerKey] = linkMarker
						result.WriteString(fmt.Sprintf("{{LINK_REPLACEMENT:%s}}", markerKey))
					} else {
						// If no context, we can't store the marker, so just write empty string
						result.WriteString("")
					}
				} else {
					result.WriteString(FormatValue(value))
				}
			}
		case TokenPageBreak:
			// Handle pageBreak token - call the pageBreak function
			registry := GetDefaultFunctionRegistry()
			if fn, exists := registry.GetFunction("pageBreak"); exists {
				value, err := fn.Call()
				if err != nil {
					return nil, fmt.Errorf("failed to call pageBreak function: %w", err)
				}
				// Check if the value is an OOXML fragment that needs special handling
				if fragment, ok := value.(*OOXMLFragment); ok {
					// Store the fragment in context and create a placeholder
					var fragmentKey string
					if ctx != nil && ctx.ooxmlFragments != nil {
						fragmentKey = fmt.Sprintf("fragment_%d", len(ctx.ooxmlFragments))
						ctx.ooxmlFragments[fragmentKey] = fragment.Content
					} else {
						// Fallback to type-based placeholder when no context
						fragmentKey = fmt.Sprintf("%T", fragment.Content)
					}
					result.WriteString(fmt.Sprintf("{{OOXML_FRAGMENT:%s}}", fragmentKey))
				} else {
					result.WriteString(FormatValue(value))
				}
			} else {
				// If pageBreak function is not registered, just add empty
				result.WriteString("")
			}
		default:
			// For now, other token types are preserved as-is
			result.WriteString("{{")
			if token.Type == TokenIf {
				result.WriteString("if ")
			} else if token.Type == TokenFor {
				result.WriteString("for ")
			} else if token.Type == TokenEnd {
				result.WriteString("end")
			} else if token.Type == TokenElse {
				result.WriteString("else")
			}
			result.WriteString(token.Value)
			result.WriteString("}}")
		}
	}
	
	return &Text{
		XMLName: text.XMLName,
		Space:   text.Space,
		Content: result.String(),
	}, nil
}