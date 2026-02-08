package stencil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ExpressionNode represents a node in the expression AST
type ExpressionNode interface {
	String() string
	Evaluate(data TemplateData) (interface{}, error)
}

// LiteralNode represents a literal value (string, number, boolean)
type LiteralNode struct {
	Value interface{}
}

func (n *LiteralNode) String() string {
	// Add quotes around string values for proper formatting
	if str, ok := n.Value.(string); ok {
		return fmt.Sprintf("Literal(%q)", str)
	}
	return fmt.Sprintf("Literal(%v)", n.Value)
}

func (n *LiteralNode) Evaluate(data TemplateData) (interface{}, error) {
	return n.Value, nil
}

// VariableNode represents a variable reference
type VariableNode struct {
	Name string
}

func (n *VariableNode) String() string {
	return fmt.Sprintf("Variable(%s)", n.Name)
}

func (n *VariableNode) Evaluate(data TemplateData) (interface{}, error) {
	return EvaluateVariable(n.Name, data)
}

// BinaryOpNode represents a binary operation
type BinaryOpNode struct {
	Left     ExpressionNode
	Operator string
	Right    ExpressionNode
}

func (n *BinaryOpNode) String() string {
	return fmt.Sprintf("BinaryOp(%s %s %s)", n.Left.String(), n.Operator, n.Right.String())
}

func (n *BinaryOpNode) Evaluate(data TemplateData) (interface{}, error) {
	leftVal, err := n.Left.Evaluate(data)
	if err != nil {
		return nil, err
	}

	rightVal, err := n.Right.Evaluate(data)
	if err != nil {
		return nil, err
	}

	return EvaluateBinaryOperation(leftVal, n.Operator, rightVal)
}

// UnaryOpNode represents a unary operation
type UnaryOpNode struct {
	Operator string
	Operand  ExpressionNode
}

func (n *UnaryOpNode) String() string {
	return fmt.Sprintf("UnaryOp(%s %s)", n.Operator, n.Operand.String())
}

func (n *UnaryOpNode) Evaluate(data TemplateData) (interface{}, error) {
	operandVal, err := n.Operand.Evaluate(data)
	if err != nil {
		return nil, err
	}

	switch n.Operator {
	case "!":
		return !isTruthy(operandVal), nil
	case "-":
		return evaluateUnaryMinus(operandVal)
	case "+":
		return evaluateUnaryPlus(operandVal)
	default:
		return nil, fmt.Errorf("unknown unary operator: %s", n.Operator)
	}
}

// FunctionCallNode represents a function call
type FunctionCallNode struct {
	Name string
	Args []ExpressionNode
}

// FieldAccessNode represents field access (obj.field)
type FieldAccessNode struct {
	Object ExpressionNode
	Field  string
}

func (n *FieldAccessNode) String() string {
	return fmt.Sprintf("FieldAccess(%s.%s)", n.Object.String(), n.Field)
}

func (n *FieldAccessNode) Evaluate(data TemplateData) (interface{}, error) {
	obj, err := n.Object.Evaluate(data)
	if err != nil {
		return nil, err
	}
	return accessMapField(obj, n.Field), nil
}

// IndexAccessNode represents index access (obj[index])
type IndexAccessNode struct {
	Object ExpressionNode
	Index  ExpressionNode
}

func (n *IndexAccessNode) String() string {
	return fmt.Sprintf("IndexAccess(%s[%s])", n.Object.String(), n.Index.String())
}

func (n *IndexAccessNode) Evaluate(data TemplateData) (interface{}, error) {
	obj, err := n.Object.Evaluate(data)
	if err != nil {
		return nil, err
	}

	indexVal, err := n.Index.Evaluate(data)
	if err != nil {
		return nil, err
	}

	// Handle string keys and integer indices
	switch idx := indexVal.(type) {
	case int:
		return accessArrayIndex(obj, idx), nil
	case string:
		return accessMapField(obj, idx), nil
	case float64:
		// Convert float to int for array access
		return accessArrayIndex(obj, int(idx)), nil
	default:
		return nil, fmt.Errorf("invalid index type: %T", indexVal)
	}
}

func (n *FunctionCallNode) String() string {
	args := make([]string, len(n.Args))
	for i, arg := range n.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("FunctionCall(%s, [%s])", n.Name, strings.Join(args, ", "))
}

func (n *FunctionCallNode) Evaluate(data TemplateData) (interface{}, error) {
	// Special handling for data() function
	if n.Name == "data" && len(n.Args) == 0 {
		return data, nil
	}

	// Get the function registry from data context if available
	var registry FunctionRegistry
	if reg, ok := data["__functions__"]; ok {
		if funcReg, ok := reg.(FunctionRegistry); ok {
			registry = funcReg
		}
	}

	// If no registry, use default registry
	if registry == nil {
		registry = GetDefaultFunctionRegistry()
	}

	// Look up the function
	fn, exists := registry.GetFunction(n.Name)
	if !exists {
		return nil, fmt.Errorf("unknown function: %s", n.Name)
	}

	// Evaluate arguments
	args := make([]interface{}, len(n.Args))
	for i, arg := range n.Args {
		val, err := arg.Evaluate(data)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate argument %d for function %s: %w", i, n.Name, err)
		}
		args[i] = val
	}

	// Call the function
	return fn.Call(args...)
}

// ExpressionToken represents a token in an expression
type ExpressionToken struct {
	Type  ExpressionTokenType
	Value string
	Pos   int
}

type ExpressionTokenType int

const (
	ExprTokenIdentifier ExpressionTokenType = iota
	ExprTokenNumber
	ExprTokenString
	ExprTokenOperator
	ExprTokenLeftParen
	ExprTokenRightParen
	ExprTokenComma
	ExprTokenEOF
	ExprTokenInvalid
)

var (
	// Regular expressions for tokenizing expressions
	identifierRegex  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`)
	numberRegex      = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?`)
	stringRegex      = regexp.MustCompile(`^"([^"\\]|\\.)*"`)
	singleQuoteRegex = regexp.MustCompile(`^'([^'\\]|\\.)*'`)
	// German typographic quotes: „..." (U+201E opening)
	// Matches „text" with closing quotes: " (U+201C), " (U+201D), or " (U+0022 regular ASCII)
	// Using UTF-8 bytes: \xe2\x80\x9e = „, \xe2\x80\x9c = ", \xe2\x80\x9d = ", \x22 = "
	// The regex needs to exclude the opening quote and any of the closing quotes in the middle
	germanQuoteRegex = regexp.MustCompile("^\xe2\x80\x9e([^\xe2\x80\x9c\xe2\x80\x9d\"\\\\]|\\\\.)*[\xe2\x80\x9c\xe2\x80\x9d\"]")
	// French/Swiss quotes: »...« (U+00BB and U+00AB)
	frenchQuoteRegex = regexp.MustCompile(`^»([^«\\]|\\.)*«`)
	operatorRegex    = regexp.MustCompile(`^(==|!=|<=|>=|\+|\-|\*|\/|\%|\&|\||\!|<|>|=)`)
)

// TokenizeExpression tokenizes an expression string
func TokenizeExpression(expr string) ([]ExpressionToken, error) {
	var tokens []ExpressionToken
	pos := 0

	for pos < len(expr) {
		// Skip whitespace
		if expr[pos] == ' ' || expr[pos] == '\t' || expr[pos] == '\n' {
			pos++
			continue
		}

		remaining := expr[pos:]

		// Try to match identifiers (variables, function names, keywords)
		if match := identifierRegex.FindString(remaining); match != "" {
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenIdentifier,
				Value: match,
				Pos:   pos,
			})
			pos += len(match)
			continue
		}

		// Try to match numbers
		if match := numberRegex.FindString(remaining); match != "" {
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenNumber,
				Value: match,
				Pos:   pos,
			})
			pos += len(match)
			continue
		}

		// Try to match double-quoted strings
		if match := stringRegex.FindString(remaining); match != "" {
			// Remove quotes from the value
			value := match[1 : len(match)-1]
			// Unescape common escape sequences
			value = strings.ReplaceAll(value, `\"`, `"`)
			value = strings.ReplaceAll(value, `\\`, `\`)
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenString,
				Value: value,
				Pos:   pos,
			})
			pos += len(match)
			continue
		}

		// Try to match single-quoted strings
		if match := singleQuoteRegex.FindString(remaining); match != "" {
			// Remove quotes from the value
			value := match[1 : len(match)-1]
			// Unescape common escape sequences
			value = strings.ReplaceAll(value, `\'`, `'`)
			value = strings.ReplaceAll(value, `\\`, `\`)
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenString,
				Value: value,
				Pos:   pos,
			})
			pos += len(match)
			continue
		}

		// Try to match German typographic quotes: „..."
		if match := germanQuoteRegex.FindString(remaining); match != "" {
			// Remove German quotes from the value („ is 3 bytes, " is 3 bytes in UTF-8)
			value := string([]rune(match)[1 : len([]rune(match))-1])
			// Unescape common escape sequences
			value = strings.ReplaceAll(value, `\"`, `"`)
			value = strings.ReplaceAll(value, `\\`, `\`)
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenString,
				Value: value,
				Pos:   pos,
			})
			pos += len(match)
			continue
		}

		// Try to match French/Swiss quotes: »...«
		if match := frenchQuoteRegex.FindString(remaining); match != "" {
			// Remove French quotes from the value (» and « are each 2 bytes in UTF-8)
			value := string([]rune(match)[1 : len([]rune(match))-1])
			// Unescape common escape sequences
			value = strings.ReplaceAll(value, `\"`, `"`)
			value = strings.ReplaceAll(value, `\\`, `\`)
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenString,
				Value: value,
				Pos:   pos,
			})
			pos += len(match)
			continue
		}

		// Try to match operators
		if match := operatorRegex.FindString(remaining); match != "" {
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenOperator,
				Value: match,
				Pos:   pos,
			})
			pos += len(match)
			continue
		}

		// Handle parentheses
		if expr[pos] == '(' {
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenLeftParen,
				Value: "(",
				Pos:   pos,
			})
			pos++
			continue
		}

		if expr[pos] == ')' {
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenRightParen,
				Value: ")",
				Pos:   pos,
			})
			pos++
			continue
		}

		// Handle commas
		if expr[pos] == ',' {
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenComma,
				Value: ",",
				Pos:   pos,
			})
			pos++
			continue
		}

		// Handle dots for field access
		if expr[pos] == '.' {
			// Check if this is part of a number (e.g., .5)
			if pos+1 < len(expr) && expr[pos+1] >= '0' && expr[pos+1] <= '9' {
				// This is a decimal number starting with a dot
				if match := regexp.MustCompile(`^\.[0-9]+`).FindString(remaining); match != "" {
					tokens = append(tokens, ExpressionToken{
						Type:  ExprTokenNumber,
						Value: "0" + match, // Convert .5 to 0.5
						Pos:   pos,
					})
					pos += len(match)
					continue
				}
			}
			// Otherwise, treat as operator for field access
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenOperator,
				Value: ".",
				Pos:   pos,
			})
			pos++
			continue
		}

		// Handle brackets for array/map access
		if expr[pos] == '[' {
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenOperator,
				Value: "[",
				Pos:   pos,
			})
			pos++
			continue
		}

		if expr[pos] == ']' {
			tokens = append(tokens, ExpressionToken{
				Type:  ExprTokenOperator,
				Value: "]",
				Pos:   pos,
			})
			pos++
			continue
		}

		// If we get here, we have an unrecognized character
		return nil, fmt.Errorf("unexpected character '%c' at position %d", expr[pos], pos)
	}

	// Add EOF token
	tokens = append(tokens, ExpressionToken{
		Type: ExprTokenEOF,
		Pos:  pos,
	})

	return tokens, nil
}

// ParseExpression parses an expression string into an AST
func ParseExpression(expr string) (ExpressionNode, error) {
	return parseExpressionWithMode(expr, false)
}

// ParseExpressionStrict parses an expression string into an AST and requires full token consumption.
// This is used by validation flows to reject trailing tokens such as "name name2".
func ParseExpressionStrict(expr string) (ExpressionNode, error) {
	return parseExpressionWithMode(expr, true)
}

func parseExpressionWithMode(expr string, requireEOF bool) (ExpressionNode, error) {
	tokens, err := TokenizeExpression(expr)
	if err != nil {
		return nil, err
	}

	parser := &ExpressionParser{
		tokens: tokens,
		pos:    0,
	}

	node, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}

	if requireEOF && parser.current().Type != ExprTokenEOF {
		token := parser.current()
		return nil, fmt.Errorf("unexpected trailing token %q at position %d", token.Value, token.Pos)
	}

	return node, nil
}

// ExpressionParser parses expressions into AST nodes
type ExpressionParser struct {
	tokens []ExpressionToken
	pos    int
}

func (p *ExpressionParser) current() ExpressionToken {
	if p.pos >= len(p.tokens) {
		return ExpressionToken{Type: ExprTokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *ExpressionParser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// parseExpression parses a complete expression
func (p *ExpressionParser) parseExpression() (ExpressionNode, error) {
	return p.parseLogicalOr()
}

// parseLogicalOr parses logical OR expressions (lowest precedence)
func (p *ExpressionParser) parseLogicalOr() (ExpressionNode, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}

	for p.current().Type == ExprTokenOperator && p.current().Value == "|" {
		op := p.current().Value
		p.advance()
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Left: left, Operator: op, Right: right}
	}

	return left, nil
}

// parseLogicalAnd parses logical AND expressions
func (p *ExpressionParser) parseLogicalAnd() (ExpressionNode, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}

	for p.current().Type == ExprTokenOperator && p.current().Value == "&" {
		op := p.current().Value
		p.advance()
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Left: left, Operator: op, Right: right}
	}

	return left, nil
}

// parseEquality parses equality expressions (==, !=)
func (p *ExpressionParser) parseEquality() (ExpressionNode, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}

	for p.current().Type == ExprTokenOperator && (p.current().Value == "==" || p.current().Value == "!=") {
		op := p.current().Value
		p.advance()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Left: left, Operator: op, Right: right}
	}

	return left, nil
}

// parseComparison parses comparison expressions (<, >, <=, >=)
func (p *ExpressionParser) parseComparison() (ExpressionNode, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}

	for p.current().Type == ExprTokenOperator &&
		(p.current().Value == "<" || p.current().Value == ">" ||
			p.current().Value == "<=" || p.current().Value == ">=") {
		op := p.current().Value
		p.advance()
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Left: left, Operator: op, Right: right}
	}

	return left, nil
}

// parseTerm parses addition and subtraction (lower precedence)
func (p *ExpressionParser) parseTerm() (ExpressionNode, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	for p.current().Type == ExprTokenOperator && (p.current().Value == "+" || p.current().Value == "-") {
		op := p.current().Value
		p.advance()
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Left: left, Operator: op, Right: right}
	}

	return left, nil
}

// parseFactor parses multiplication, division, and modulo (higher precedence)
func (p *ExpressionParser) parseFactor() (ExpressionNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.current().Type == ExprTokenOperator &&
		(p.current().Value == "*" || p.current().Value == "/" || p.current().Value == "%") {
		op := p.current().Value
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryOpNode{Left: left, Operator: op, Right: right}
	}

	return left, nil
}

// parseUnary parses unary expressions (!, -, +)
func (p *ExpressionParser) parseUnary() (ExpressionNode, error) {
	if p.current().Type == ExprTokenOperator &&
		(p.current().Value == "!" || p.current().Value == "-" || p.current().Value == "+") {
		op := p.current().Value
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryOpNode{Operator: op, Operand: operand}, nil
	}

	return p.parseFieldAccess()
}

// parseFieldAccess parses field access expressions (obj.field, obj[key])
func (p *ExpressionParser) parseFieldAccess() (ExpressionNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	for {
		if p.current().Type == ExprTokenOperator && p.current().Value == "." {
			p.advance() // consume '.'
			if p.current().Type != ExprTokenIdentifier {
				return nil, fmt.Errorf("expected identifier after '.'")
			}
			field := p.current().Value
			p.advance()
			left = &FieldAccessNode{Object: left, Field: field}
		} else if p.current().Type == ExprTokenOperator && p.current().Value == "[" {
			p.advance() // consume '['
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if p.current().Type != ExprTokenOperator || p.current().Value != "]" {
				return nil, fmt.Errorf("expected ']' after array index")
			}
			p.advance() // consume ']'
			left = &IndexAccessNode{Object: left, Index: index}
		} else {
			break
		}
	}

	return left, nil
}

// parsePrimary parses primary expressions (literals, variables, parenthesized expressions)
func (p *ExpressionParser) parsePrimary() (ExpressionNode, error) {
	token := p.current()

	switch token.Type {
	case ExprTokenNumber:
		p.advance()
		// Try to parse as integer first
		if intVal, err := strconv.Atoi(token.Value); err == nil {
			return &LiteralNode{Value: intVal}, nil
		}
		// Otherwise parse as float
		if floatVal, err := strconv.ParseFloat(token.Value, 64); err == nil {
			return &LiteralNode{Value: floatVal}, nil
		}
		return nil, fmt.Errorf("invalid number: %s", token.Value)

	case ExprTokenString:
		p.advance()
		return &LiteralNode{Value: token.Value}, nil

	case ExprTokenIdentifier:
		p.advance()
		// Check if this is a boolean literal
		if token.Value == "true" {
			return &LiteralNode{Value: true}, nil
		}
		if token.Value == "false" {
			return &LiteralNode{Value: false}, nil
		}
		if token.Value == "null" || token.Value == "nil" {
			return &LiteralNode{Value: nil}, nil
		}

		// Check for function call
		if p.current().Type == ExprTokenLeftParen {
			return p.parseFunctionCall(token.Value)
		}

		// Otherwise it's a variable
		return &VariableNode{Name: token.Value}, nil

	case ExprTokenLeftParen:
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.current().Type != ExprTokenRightParen {
			return nil, fmt.Errorf("expected ')' after expression")
		}
		p.advance()
		return expr, nil

	default:
		return nil, fmt.Errorf("unexpected token: %s", token.Value)
	}
}

// parseFunctionCall parses a function call
func (p *ExpressionParser) parseFunctionCall(name string) (ExpressionNode, error) {
	if p.current().Type != ExprTokenLeftParen {
		return nil, fmt.Errorf("expected '(' after function name")
	}
	p.advance() // consume '('

	var args []ExpressionNode

	// Handle empty argument list
	if p.current().Type == ExprTokenRightParen {
		p.advance()
		return &FunctionCallNode{Name: name, Args: args}, nil
	}

	// Parse arguments
	for {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		if p.current().Type == ExprTokenComma {
			p.advance()
			continue
		}

		if p.current().Type == ExprTokenRightParen {
			p.advance()
			break
		}

		return nil, fmt.Errorf("expected ',' or ')' in function arguments")
	}

	return &FunctionCallNode{Name: name, Args: args}, nil
}

// EvaluateBinaryOperation evaluates a binary operation between two values
func EvaluateBinaryOperation(left interface{}, operator string, right interface{}) (interface{}, error) {
	switch operator {
	case "+":
		return evaluateAddition(left, right)
	case "-":
		return evaluateSubtraction(left, right)
	case "*":
		return evaluateMultiplication(left, right)
	case "/":
		return evaluateDivision(left, right)
	case "%":
		return evaluateModulo(left, right)
	case "==":
		return evaluateEquals(left, right), nil
	case "!=":
		return !evaluateEquals(left, right), nil
	case "<":
		return evaluateLessThan(left, right)
	case ">":
		return evaluateGreaterThan(left, right)
	case "<=":
		return evaluateLessEqual(left, right)
	case ">=":
		return evaluateGreaterEqual(left, right)
	case "&":
		return evaluateLogicalAnd(left, right), nil
	case "|":
		return evaluateLogicalOr(left, right), nil
	default:
		return nil, fmt.Errorf("unknown binary operator: %s", operator)
	}
}

// Helper functions for arithmetic operations
func evaluateAddition(left, right interface{}) (interface{}, error) {
	// Handle string concatenation
	if leftStr, ok := left.(string); ok {
		rightStr := FormatValue(right)
		return leftStr + rightStr, nil
	}
	if rightStr, ok := right.(string); ok {
		leftStr := FormatValue(left)
		return leftStr + rightStr, nil
	}

	// Handle numeric addition
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("cannot add %T and %T", left, right)
	}

	// Return int if both operands were integers
	if isInteger(left) && isInteger(right) {
		return int(leftNum + rightNum), nil
	}
	return leftNum + rightNum, nil
}

func evaluateSubtraction(left, right interface{}) (interface{}, error) {
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("cannot subtract %T and %T", left, right)
	}

	if isInteger(left) && isInteger(right) {
		return int(leftNum - rightNum), nil
	}
	return leftNum - rightNum, nil
}

func evaluateMultiplication(left, right interface{}) (interface{}, error) {
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("cannot multiply %T and %T", left, right)
	}

	if isInteger(left) && isInteger(right) {
		return int(leftNum * rightNum), nil
	}
	return leftNum * rightNum, nil
}

func evaluateDivision(left, right interface{}) (interface{}, error) {
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("cannot divide %T and %T", left, right)
	}

	if rightNum == 0 {
		return nil, fmt.Errorf("division by zero")
	}

	result := leftNum / rightNum
	// Return int if the result is a whole number and both operands were integers
	if isInteger(left) && isInteger(right) && result == float64(int(result)) {
		return int(result), nil
	}
	return result, nil
}

func evaluateModulo(left, right interface{}) (interface{}, error) {
	leftInt, leftOk := toInt(left)
	rightInt, rightOk := toInt(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("modulo operation requires integers, got %T and %T", left, right)
	}

	if rightInt == 0 {
		return nil, fmt.Errorf("modulo by zero")
	}

	return leftInt % rightInt, nil
}

// Helper functions for comparison operations
func evaluateEquals(left, right interface{}) bool {
	// Handle nil cases
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}

	// Try numeric comparison
	if leftNum, leftOk := toFloat64(left); leftOk {
		if rightNum, rightOk := toFloat64(right); rightOk {
			return leftNum == rightNum
		}
	}

	// Direct comparison
	return left == right
}

func evaluateLessThan(left, right interface{}) (interface{}, error) {
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("cannot compare %T and %T", left, right)
	}

	return leftNum < rightNum, nil
}

func evaluateGreaterThan(left, right interface{}) (interface{}, error) {
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("cannot compare %T and %T", left, right)
	}

	return leftNum > rightNum, nil
}

func evaluateLessEqual(left, right interface{}) (interface{}, error) {
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("cannot compare %T and %T", left, right)
	}

	return leftNum <= rightNum, nil
}

func evaluateGreaterEqual(left, right interface{}) (interface{}, error) {
	leftNum, leftOk := toFloat64(left)
	rightNum, rightOk := toFloat64(right)

	if !leftOk || !rightOk {
		return nil, fmt.Errorf("cannot compare %T and %T", left, right)
	}

	return leftNum >= rightNum, nil
}

// Helper functions for logical operations
func evaluateLogicalAnd(left, right interface{}) bool {
	return isTruthy(left) && isTruthy(right)
}

func evaluateLogicalOr(left, right interface{}) bool {
	return isTruthy(left) || isTruthy(right)
}

// Utility functions for type conversion and checks
func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func toInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		if v == float32(int(v)) {
			return int(v), true
		}
	case float64:
		if v == float64(int(v)) {
			return int(v), true
		}
	}
	return 0, false
}

func isInteger(val interface{}) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case int, int8, int16, int32, int64:
		return v != 0
	case uint, uint8, uint16, uint32, uint64:
		return v != 0
	case float32, float64:
		return v != 0.0
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		return true // Non-nil objects are truthy
	}
}

// Helper functions for unary operations
func evaluateUnaryMinus(operand interface{}) (interface{}, error) {
	num, ok := toFloat64(operand)
	if !ok {
		return nil, fmt.Errorf("cannot apply unary minus to %T", operand)
	}

	if isInteger(operand) {
		return -int(num), nil
	}
	return -num, nil
}

func evaluateUnaryPlus(operand interface{}) (interface{}, error) {
	num, ok := toFloat64(operand)
	if !ok {
		return nil, fmt.Errorf("cannot apply unary plus to %T", operand)
	}

	// Unary plus just returns the numeric value unchanged
	if isInteger(operand) {
		return int(num), nil
	}
	return num, nil
}
