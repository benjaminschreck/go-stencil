package stencil

import (
	"fmt"
	"strings"
)

// ControlStructure represents a control flow structure in templates
type ControlStructure interface {
	Render(data TemplateData) (string, error)
	RenderWithContext(data TemplateData, ctx *renderContext) (string, error)
	String() string
}

// IfNode represents an if statement
type IfNode struct {
	Condition ExpressionNode
	ThenBody  []ControlStructure
	ElseBody  []ControlStructure
	ElsIfs    []*ElsIfNode
}

func (n *IfNode) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("If(%s)", n.Condition.String()))
	if len(n.ElsIfs) > 0 {
		for _, elsif := range n.ElsIfs {
			parts = append(parts, elsif.String())
		}
	}
	if len(n.ElseBody) > 0 {
		parts = append(parts, "Else")
	}
	return strings.Join(parts, " ")
}

func (n *IfNode) Render(data TemplateData) (string, error) {
	// Evaluate the condition
	condValue, err := n.Condition.Evaluate(data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate if condition: %w", err)
	}
	
	// Check if condition is truthy
	if isTruthy(condValue) {
		return renderControlBody(n.ThenBody, data)
	}
	
	// Check elsif conditions
	for _, elsif := range n.ElsIfs {
		elsifValue, err := elsif.Condition.Evaluate(data)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate elsif condition: %w", err)
		}
		
		if isTruthy(elsifValue) {
			return renderControlBody(elsif.Body, data)
		}
	}
	
	// Fall back to else body
	if len(n.ElseBody) > 0 {
		return renderControlBody(n.ElseBody, data)
	}
	
	return "", nil
}

// ElsIfNode represents an elsif/elseif clause
type ElsIfNode struct {
	Condition ExpressionNode
	Body      []ControlStructure
}

func (n *ElsIfNode) String() string {
	return fmt.Sprintf("ElsIf(%s)", n.Condition.String())
}

// UnlessNode represents an unless statement (negated if)
type UnlessNode struct {
	Condition ExpressionNode
	ThenBody  []ControlStructure
	ElseBody  []ControlStructure
}

func (n *UnlessNode) String() string {
	return fmt.Sprintf("Unless(%s)", n.Condition.String())
}

func (n *UnlessNode) Render(data TemplateData) (string, error) {
	// Evaluate the condition
	condValue, err := n.Condition.Evaluate(data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate unless condition: %w", err)
	}
	
	// Unless is the opposite of if - render body if condition is falsy
	if !isTruthy(condValue) {
		return renderControlBody(n.ThenBody, data)
	}
	
	// Fall back to else body
	if len(n.ElseBody) > 0 {
		return renderControlBody(n.ElseBody, data)
	}
	
	return "", nil
}

// ForNode represents a for loop
type ForNode struct {
	Variable    string
	IndexVar    string // Optional index variable for indexed loops
	Collection  ExpressionNode
	Body        []ControlStructure
}

func (n *ForNode) String() string {
	if n.IndexVar != "" {
		return fmt.Sprintf("For(%s, %s in %s)", n.IndexVar, n.Variable, n.Collection.String())
	}
	return fmt.Sprintf("For(%s in %s)", n.Variable, n.Collection.String())
}

func (n *ForNode) Render(data TemplateData) (string, error) {
	// Evaluate the collection
	collectionVal, err := n.Collection.Evaluate(data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate collection: %w", err)
	}
	
	// Convert to slice
	items, err := toSlice(collectionVal)
	if err != nil {
		return "", fmt.Errorf("collection is not iterable: %w", err)
	}
	
	var result strings.Builder
	
	// Iterate over items
	for i, item := range items {
		// Create new data context for this iteration
		loopData := make(TemplateData)
		
		// Copy existing data
		for k, v := range data {
			loopData[k] = v
		}
		
		// Add loop variables
		loopData[n.Variable] = item
		if n.IndexVar != "" {
			loopData[n.IndexVar] = i
		}
		
		// Render the body with loop context
		bodyResult, err := renderControlBody(n.Body, loopData)
		if err != nil {
			return "", err
		}
		
		result.WriteString(bodyResult)
	}
	
	return result.String(), nil
}

// TextNode represents plain text content
type TextNode struct {
	Content string
}

func (n *TextNode) String() string {
	return fmt.Sprintf("Text(%q)", n.Content)
}

func (n *TextNode) Render(data TemplateData) (string, error) {
	return n.Content, nil
}

// IncludeNode represents an include statement
type IncludeNode struct {
	FragmentName ExpressionNode
}

func (n *IncludeNode) String() string {
	return fmt.Sprintf("Include(%s)", n.FragmentName.String())
}

func (n *IncludeNode) Render(data TemplateData) (string, error) {
	// This will be implemented in RenderWithContext
	return "", fmt.Errorf("include rendering requires context")
}

// ExpressionContentNode represents an expression that should be evaluated and output
type ExpressionContentNode struct {
	Expression ExpressionNode
}

func (n *ExpressionContentNode) String() string {
	return fmt.Sprintf("Expression(%s)", n.Expression.String())
}

func (n *ExpressionContentNode) Render(data TemplateData) (string, error) {
	value, err := n.Expression.Evaluate(data)
	if err != nil {
		return "", err
	}
	return FormatValue(value), nil
}

// renderControlBody renders a list of control structures
func renderControlBody(body []ControlStructure, data TemplateData) (string, error) {
	var result strings.Builder
	for _, item := range body {
		rendered, err := item.Render(data)
		if err != nil {
			return "", err
		}
		result.WriteString(rendered)
	}
	return result.String(), nil
}

// ControlParser parses control structures from template tokens
type ControlParser struct {
	tokens []Token
	pos    int
}

// ParseControlStructures parses tokens into control structures
func ParseControlStructures(content string) ([]ControlStructure, error) {
	tokens := Tokenize(content)
	parser := &ControlParser{
		tokens: tokens,
		pos:    0,
	}
	return parser.parseStructures()
}

func (p *ControlParser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenText, Value: ""}
	}
	return p.tokens[p.pos]
}

func (p *ControlParser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *ControlParser) parseStructures() ([]ControlStructure, error) {
	var structures []ControlStructure
	
	for p.pos < len(p.tokens) {
		token := p.current()
		
		switch token.Type {
		case TokenText:
			if token.Value != "" {
				structures = append(structures, &TextNode{Content: token.Value})
			}
			p.advance()
			
		case TokenVariable:
			// Parse as expression
			expr, err := ParseExpression(token.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse expression %s: %w", token.Value, err)
			}
			structures = append(structures, &ExpressionContentNode{Expression: expr})
			p.advance()
			
		case TokenIf:
			ifNode, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			structures = append(structures, ifNode)
			
		case TokenUnless:
			unlessNode, err := p.parseUnless()
			if err != nil {
				return nil, err
			}
			structures = append(structures, unlessNode)
			
		case TokenFor:
			forNode, err := p.parseFor()
			if err != nil {
				return nil, err
			}
			structures = append(structures, forNode)
			
		case TokenInclude:
			includeNode, err := p.parseInclude()
			if err != nil {
				return nil, err
			}
			structures = append(structures, includeNode)
			
		default:
			return nil, fmt.Errorf("unexpected token type: %v", token.Type)
		}
	}
	
	return structures, nil
}

func (p *ControlParser) parseIf() (*IfNode, error) {
	if p.current().Type != TokenIf {
		return nil, fmt.Errorf("expected if token")
	}
	
	// Parse condition
	conditionStr := p.current().Value
	condition, err := ParseExpression(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse if condition: %w", err)
	}
	p.advance()
	
	ifNode := &IfNode{
		Condition: condition,
	}
	
	// Parse then body until we hit else, elsif, or end
	thenBody, err := p.parseBodyUntil(TokenElse, TokenElsif, TokenEnd)
	if err != nil {
		return nil, err
	}
	ifNode.ThenBody = thenBody
	
	// Handle elsif clauses
	for p.current().Type == TokenElsif {
		elsif, err := p.parseElsIf()
		if err != nil {
			return nil, err
		}
		ifNode.ElsIfs = append(ifNode.ElsIfs, elsif)
	}
	
	// Handle else clause
	if p.current().Type == TokenElse {
		p.advance() // consume else token
		elseBody, err := p.parseBodyUntil(TokenEnd)
		if err != nil {
			return nil, err
		}
		ifNode.ElseBody = elseBody
	}
	
	// Consume end token
	if p.current().Type != TokenEnd {
		return nil, fmt.Errorf("expected end token to close if statement")
	}
	p.advance()
	
	return ifNode, nil
}

func (p *ControlParser) parseElsIf() (*ElsIfNode, error) {
	if p.current().Type != TokenElsif {
		return nil, fmt.Errorf("expected elsif token")
	}
	
	// Parse condition
	conditionStr := p.current().Value
	condition, err := ParseExpression(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse elsif condition: %w", err)
	}
	p.advance()
	
	// Parse body until else, elsif, or end
	body, err := p.parseBodyUntil(TokenElse, TokenElsif, TokenEnd)
	if err != nil {
		return nil, err
	}
	
	return &ElsIfNode{
		Condition: condition,
		Body:      body,
	}, nil
}

func (p *ControlParser) parseUnless() (*UnlessNode, error) {
	if p.current().Type != TokenUnless {
		return nil, fmt.Errorf("expected unless token")
	}
	
	// Parse condition
	conditionStr := p.current().Value
	condition, err := ParseExpression(conditionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse unless condition: %w", err)
	}
	p.advance()
	
	unlessNode := &UnlessNode{
		Condition: condition,
	}
	
	// Parse then body until we hit else or end
	thenBody, err := p.parseBodyUntil(TokenElse, TokenEnd)
	if err != nil {
		return nil, err
	}
	unlessNode.ThenBody = thenBody
	
	// Handle else clause
	if p.current().Type == TokenElse {
		p.advance() // consume else token
		elseBody, err := p.parseBodyUntil(TokenEnd)
		if err != nil {
			return nil, err
		}
		unlessNode.ElseBody = elseBody
	}
	
	// Consume end token
	if p.current().Type != TokenEnd {
		return nil, fmt.Errorf("expected end token to close unless statement")
	}
	p.advance()
	
	return unlessNode, nil
}

func (p *ControlParser) parseFor() (*ForNode, error) {
	if p.current().Type != TokenFor {
		return nil, fmt.Errorf("expected for token")
	}
	
	// Parse for loop syntax: "var in collection" or "idx, var in collection"
	forStr := p.current().Value
	forNode, err := parseForSyntax(forStr)
	if err != nil {
		return nil, err
	}
	p.advance()
	
	// Parse body until end
	body, err := p.parseBodyUntil(TokenEnd)
	if err != nil {
		return nil, err
	}
	forNode.Body = body
	
	// Consume end token
	if p.current().Type != TokenEnd {
		return nil, fmt.Errorf("expected end token to close for loop")
	}
	p.advance()
	
	return forNode, nil
}

func (p *ControlParser) parseInclude() (*IncludeNode, error) {
	if p.current().Type != TokenInclude {
		return nil, fmt.Errorf("expected include token")
	}
	
	// Parse fragment name expression
	fragmentNameStr := p.current().Value
	fragmentName, err := ParseExpression(fragmentNameStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse include fragment name: %w", err)
	}
	p.advance()
	
	return &IncludeNode{
		FragmentName: fragmentName,
	}, nil
}

func (p *ControlParser) parseBodyUntil(stopTokens ...TokenType) ([]ControlStructure, error) {
	var body []ControlStructure
	
	for p.pos < len(p.tokens) {
		current := p.current()
		
		// Check if we hit a stop token
		for _, stopType := range stopTokens {
			if current.Type == stopType {
				return body, nil
			}
		}
		
		// Parse the current structure
		switch current.Type {
		case TokenText:
			if current.Value != "" {
				body = append(body, &TextNode{Content: current.Value})
			}
			p.advance()
			
		case TokenVariable:
			expr, err := ParseExpression(current.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse expression %s: %w", current.Value, err)
			}
			body = append(body, &ExpressionContentNode{Expression: expr})
			p.advance()
			
		case TokenIf:
			ifNode, err := p.parseIf()
			if err != nil {
				return nil, err
			}
			body = append(body, ifNode)
			
		case TokenUnless:
			unlessNode, err := p.parseUnless()
			if err != nil {
				return nil, err
			}
			body = append(body, unlessNode)
			
		case TokenFor:
			forNode, err := p.parseFor()
			if err != nil {
				return nil, err
			}
			body = append(body, forNode)
			
		case TokenInclude:
			includeNode, err := p.parseInclude()
			if err != nil {
				return nil, err
			}
			body = append(body, includeNode)
			
		default:
			return nil, fmt.Errorf("unexpected token in body: %v", current.Type)
		}
	}
	
	return body, nil
}

// parseForSyntax parses the for loop syntax
func parseForSyntax(forStr string) (*ForNode, error) {
	// Remove extra whitespace
	forStr = strings.TrimSpace(forStr)
	
	// Look for " in " to split variable(s) from collection
	inIndex := strings.Index(forStr, " in ")
	if inIndex == -1 {
		return nil, fmt.Errorf("invalid for loop syntax: missing 'in' keyword")
	}
	
	varsStr := strings.TrimSpace(forStr[:inIndex])
	collectionStr := strings.TrimSpace(forStr[inIndex+4:])
	
	// Parse collection expression
	collection, err := ParseExpression(collectionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse collection expression: %w", err)
	}
	
	// Parse variables - check if there's a comma for indexed loop
	if strings.Contains(varsStr, ",") {
		// Indexed loop: "idx, var in collection"
		parts := strings.Split(varsStr, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid indexed for loop syntax")
		}
		
		indexVar := strings.TrimSpace(parts[0])
		variable := strings.TrimSpace(parts[1])
		
		return &ForNode{
			IndexVar:   indexVar,
			Variable:   variable,
			Collection: collection,
		}, nil
	} else {
		// Simple loop: "var in collection"
		return &ForNode{
			Variable:   varsStr,
			Collection: collection,
		}, nil
	}
}

// toSlice converts various types to []interface{} for iteration
func toSlice(val interface{}) ([]interface{}, error) {
	if val == nil {
		return []interface{}{}, nil
	}
	
	switch v := val.(type) {
	case []interface{}:
		return v, nil
	case []string:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case []int:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case []float64:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case []bool:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case []map[string]interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = item
		}
		return result, nil
	case map[string]interface{}:
		// Convert map to slice of key-value pairs
		result := make([]interface{}, 0, len(v))
		for key, value := range v {
			result = append(result, map[string]interface{}{
				"key":   key,
				"value": value,
			})
		}
		return result, nil
	case TemplateData:
		// Convert TemplateData to slice of key-value pairs
		result := make([]interface{}, 0, len(v))
		for key, value := range v {
			result = append(result, map[string]interface{}{
				"key":   key,
				"value": value,
			})
		}
		return result, nil
	case string:
		// Iterate over characters
		result := make([]interface{}, len(v))
		for i, char := range v {
			result[i] = string(char)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("type %T is not iterable", val)
	}
}