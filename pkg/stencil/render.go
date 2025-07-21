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
		rendered.Runs = append(rendered.Runs, *renderedRun)
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
		
		// Check if the rendered text contains OOXML fragment placeholders
		if ooxmlFragmentRegex.MatchString(renderedText.Content) {
			// Extract fragments and create separate runs/elements
			return processOOXMLFragments(rendered, renderedText, data)
		}
		
		rendered.Text = renderedText
	}
	
	return rendered, nil
}

// processOOXMLFragments processes OOXML fragments in text and creates appropriate elements
func processOOXMLFragments(run *Run, text *Text, data TemplateData) (*Run, error) {
	content := text.Content
	
	// For simplicity, if there's a page break fragment, create a run with just the break
	if strings.Contains(content, "{{OOXML_FRAGMENT:*stencil.Break}}") {
		// Remove the placeholder text and set the break
		cleanContent := ooxmlFragmentRegex.ReplaceAllString(content, "")
		
		// If there's remaining text, keep it
		if strings.TrimSpace(cleanContent) != "" {
			run.Text = &Text{
				XMLName: text.XMLName,
				Space:   text.Space,
				Content: cleanContent,
			}
		} else {
			run.Text = nil
		}
		
		// Add the page break
		run.Break = &Break{Type: "page"}
	} else {
		// No OOXML fragments, keep the original text
		run.Text = text
	}
	
	return run, nil
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
					// For now, we'll just store a placeholder - the actual OOXML injection
					// will be handled at a higher level in the rendering pipeline
					result.WriteString(fmt.Sprintf("{{OOXML_FRAGMENT:%T}}", fragment.Content))
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
					// For now, we'll just store a placeholder - the actual OOXML injection
					// will be handled at a higher level in the rendering pipeline
					result.WriteString(fmt.Sprintf("{{OOXML_FRAGMENT:%T}}", fragment.Content))
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