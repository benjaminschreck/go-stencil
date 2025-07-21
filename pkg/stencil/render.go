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
	rendered := &Paragraph{
		Properties: para.Properties,
	}
	
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
				} else if marker, ok := value.(*imageReplacementMarker); ok {
					// Handle image replacement markers
					markerKey := fmt.Sprintf("img_%d", len(ctx.imageMarkers))
					if ctx != nil {
						ctx.imageMarkers[markerKey] = marker
					}
					result.WriteString(fmt.Sprintf("{{IMAGE_REPLACEMENT:%s:%d}}", marker.mimeType, len(marker.data)))
				} else if marker, ok := value.(LinkReplacementMarker); ok {
					// Handle link replacement markers
					markerKey := fmt.Sprintf("link_%d", len(ctx.linkMarkers))
					if ctx != nil {
						linkMarker := &marker
						ctx.linkMarkers[markerKey] = linkMarker
					}
					result.WriteString(fmt.Sprintf("{{LINK_REPLACEMENT:%s}}", markerKey))
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
				} else if marker, ok := value.(*imageReplacementMarker); ok {
					// Handle image replacement markers
					markerKey := fmt.Sprintf("img_%d", len(ctx.imageMarkers))
					if ctx != nil {
						ctx.imageMarkers[markerKey] = marker
					}
					result.WriteString(fmt.Sprintf("{{IMAGE_REPLACEMENT:%s:%d}}", marker.mimeType, len(marker.data)))
				} else if marker, ok := value.(LinkReplacementMarker); ok {
					// Handle link replacement markers
					markerKey := fmt.Sprintf("link_%d", len(ctx.linkMarkers))
					if ctx != nil {
						linkMarker := &marker
						ctx.linkMarkers[markerKey] = linkMarker
					}
					result.WriteString(fmt.Sprintf("{{LINK_REPLACEMENT:%s}}", markerKey))
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