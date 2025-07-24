package stencil

import (
	"fmt"
	"strings"
)

// RenderWithContext implementations for all control structures

func (n *IfNode) RenderWithContext(data TemplateData, ctx *renderContext) (string, error) {
	// Evaluate the condition
	condValue, err := n.Condition.Evaluate(data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate if condition: %w", err)
	}
	
	// Check if condition is truthy
	if isTruthy(condValue) {
		return renderControlBodyWithContext(n.ThenBody, data, ctx)
	}
	
	// Check elsif conditions
	for _, elsif := range n.ElsIfs {
		elsifValue, err := elsif.Condition.Evaluate(data)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate elsif condition: %w", err)
		}
		
		if isTruthy(elsifValue) {
			return renderControlBodyWithContext(elsif.Body, data, ctx)
		}
	}
	
	// Fall back to else body
	if len(n.ElseBody) > 0 {
		return renderControlBodyWithContext(n.ElseBody, data, ctx)
	}
	
	return "", nil
}

func (n *UnlessNode) RenderWithContext(data TemplateData, ctx *renderContext) (string, error) {
	// Evaluate the condition
	condValue, err := n.Condition.Evaluate(data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate unless condition: %w", err)
	}
	
	// Unless is the opposite of if - render body if condition is falsy
	if !isTruthy(condValue) {
		return renderControlBodyWithContext(n.ThenBody, data, ctx)
	}
	
	// Fall back to else body
	if len(n.ElseBody) > 0 {
		return renderControlBodyWithContext(n.ElseBody, data, ctx)
	}
	
	return "", nil
}

func (n *ForNode) RenderWithContext(data TemplateData, ctx *renderContext) (string, error) {
	// Evaluate the collection expression
	collectionValue, err := n.Collection.Evaluate(data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate collection: %w", err)
	}
	
	// Convert to slice
	slice, err := toSlice(collectionValue)
	if err != nil {
		return "", fmt.Errorf("failed to iterate over collection: %w", err)
	}
	
	var result strings.Builder
	
	for idx, item := range slice {
		// Create a new data context with the loop variable(s)
		loopData := make(TemplateData)
		for k, v := range data {
			loopData[k] = v
		}
		
		// Set the loop variable(s)
		loopData[n.Variable] = item
		if n.IndexVar != "" {
			loopData[n.IndexVar] = idx
		}
		
		// Render the body
		bodyResult, err := renderControlBodyWithContext(n.Body, loopData, ctx)
		if err != nil {
			return "", err
		}
		result.WriteString(bodyResult)
	}
	
	return result.String(), nil
}

func (n *TextNode) RenderWithContext(data TemplateData, ctx *renderContext) (string, error) {
	return n.Content, nil
}

func (n *ExpressionContentNode) RenderWithContext(data TemplateData, ctx *renderContext) (string, error) {
	value, err := n.Expression.Evaluate(data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate expression: %w", err)
	}
	
	// Check for special marker types that need context
	if marker, ok := value.(*TableRowMarker); ok {
		// Handle table row markers
		return fmt.Sprintf("{{TABLE_ROW_MARKER:%s}}", marker.Action), nil
	} else if marker, ok := value.(*TableColumnMarker); ok {
		// Handle table column markers
		return marker.String(), nil
	} else if marker, ok := value.(LinkReplacementMarker); ok && ctx != nil {
		// Handle link replacement markers
		markerKey := fmt.Sprintf("link_%d", len(ctx.linkMarkers))
		linkMarker := &marker
		ctx.linkMarkers[markerKey] = linkMarker
		return fmt.Sprintf("{{LINK_REPLACEMENT:%s}}", markerKey), nil
	}
	
	return FormatValue(value), nil
}

func (n *IncludeNode) RenderWithContext(data TemplateData, ctx *renderContext) (string, error) {
	if ctx == nil || ctx.fragments == nil {
		return "", fmt.Errorf("fragments not available in render context")
	}
	
	// Check render depth
	config := GetGlobalConfig()
	if ctx.renderDepth >= config.MaxRenderDepth {
		return "", fmt.Errorf("maximum render depth exceeded: %d", config.MaxRenderDepth)
	}
	
	// Evaluate the fragment name expression
	nameValue, err := n.FragmentName.Evaluate(data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate fragment name: %w", err)
	}
	
	// Fragment name must be a string
	fragmentName, ok := nameValue.(string)
	if !ok {
		return "", fmt.Errorf("fragment name must be a string, got %T", nameValue)
	}
	
	// Check for circular references
	for _, stackName := range ctx.fragmentStack {
		if stackName == fragmentName {
			return "", fmt.Errorf("circular fragment reference detected: %s", fragmentName)
		}
	}
	
	// Find the fragment
	fragment, exists := ctx.fragments[fragmentName]
	if !exists {
		return "", fmt.Errorf("fragment not found: %s", fragmentName)
	}
	
	// Push fragment onto stack and increment depth
	ctx.fragmentStack = append(ctx.fragmentStack, fragmentName)
	ctx.renderDepth++
	defer func() {
		// Pop fragment from stack and decrement depth
		ctx.fragmentStack = ctx.fragmentStack[:len(ctx.fragmentStack)-1]
		ctx.renderDepth--
	}()
	
	// Parse the fragment content as control structures
	structures, err := ParseControlStructures(fragment.content)
	if err != nil {
		return "", fmt.Errorf("failed to parse fragment %s: %w", fragmentName, err)
	}
	
	// Render the fragment structures with context
	return renderControlBodyWithContext(structures, data, ctx)
}

// renderControlBodyWithContext renders a slice of control structures with context
func renderControlBodyWithContext(body []ControlStructure, data TemplateData, ctx *renderContext) (string, error) {
	var result strings.Builder
	
	for _, structure := range body {
		rendered, err := structure.RenderWithContext(data, ctx)
		if err != nil {
			return "", err
		}
		result.WriteString(rendered)
	}
	
	return result.String(), nil
}

// ProcessTemplateWithFragments processes a template string with control structures and fragments
func ProcessTemplateWithFragments(content string, data TemplateData, fragments map[string]*fragment) (string, error) {
	// Parse control structures
	structures, err := ParseControlStructures(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	// Create render context with fragments
	ctx := &renderContext{
		fragments:      fragments,
		fragmentStack:  make([]string, 0),
		renderDepth:    0,
		ooxmlFragments: make(map[string]interface{}),
	}
	
	// Render with context
	return renderControlBodyWithContext(structures, data, ctx)
}