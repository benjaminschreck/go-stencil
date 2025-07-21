package stencil

import (
	"reflect"
	"testing"
)

func TestTokenizeExpression(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		want     []ExpressionToken
		wantErr  bool
	}{
		{
			name: "simple variable",
			expr: "name",
			want: []ExpressionToken{
				{Type: ExprTokenIdentifier, Value: "name", Pos: 0},
				{Type: ExprTokenEOF, Pos: 4},
			},
		},
		{
			name: "integer literal",
			expr: "42",
			want: []ExpressionToken{
				{Type: ExprTokenNumber, Value: "42", Pos: 0},
				{Type: ExprTokenEOF, Pos: 2},
			},
		},
		{
			name: "float literal",
			expr: "3.14",
			want: []ExpressionToken{
				{Type: ExprTokenNumber, Value: "3.14", Pos: 0},
				{Type: ExprTokenEOF, Pos: 4},
			},
		},
		{
			name: "decimal starting with dot",
			expr: ".5",
			want: []ExpressionToken{
				{Type: ExprTokenNumber, Value: "0.5", Pos: 0},
				{Type: ExprTokenEOF, Pos: 2},
			},
		},
		{
			name: "string literal double quotes",
			expr: `"hello world"`,
			want: []ExpressionToken{
				{Type: ExprTokenString, Value: "hello world", Pos: 0},
				{Type: ExprTokenEOF, Pos: 13},
			},
		},
		{
			name: "string literal single quotes",
			expr: `'hello world'`,
			want: []ExpressionToken{
				{Type: ExprTokenString, Value: "hello world", Pos: 0},
				{Type: ExprTokenEOF, Pos: 13},
			},
		},
		{
			name: "string with escaped quotes",
			expr: `"hello \"world\""`,
			want: []ExpressionToken{
				{Type: ExprTokenString, Value: `hello "world"`, Pos: 0},
				{Type: ExprTokenEOF, Pos: 17},
			},
		},
		{
			name: "boolean literals",
			expr: "true false",
			want: []ExpressionToken{
				{Type: ExprTokenIdentifier, Value: "true", Pos: 0},
				{Type: ExprTokenIdentifier, Value: "false", Pos: 5},
				{Type: ExprTokenEOF, Pos: 10},
			},
		},
		{
			name: "null literal",
			expr: "null nil",
			want: []ExpressionToken{
				{Type: ExprTokenIdentifier, Value: "null", Pos: 0},
				{Type: ExprTokenIdentifier, Value: "nil", Pos: 5},
				{Type: ExprTokenEOF, Pos: 8},
			},
		},
		{
			name: "arithmetic operators",
			expr: "+ - * / %",
			want: []ExpressionToken{
				{Type: ExprTokenOperator, Value: "+", Pos: 0},
				{Type: ExprTokenOperator, Value: "-", Pos: 2},
				{Type: ExprTokenOperator, Value: "*", Pos: 4},
				{Type: ExprTokenOperator, Value: "/", Pos: 6},
				{Type: ExprTokenOperator, Value: "%", Pos: 8},
				{Type: ExprTokenEOF, Pos: 9},
			},
		},
		{
			name: "comparison operators",
			expr: "== != < > <= >=",
			want: []ExpressionToken{
				{Type: ExprTokenOperator, Value: "==", Pos: 0},
				{Type: ExprTokenOperator, Value: "!=", Pos: 3},
				{Type: ExprTokenOperator, Value: "<", Pos: 6},
				{Type: ExprTokenOperator, Value: ">", Pos: 8},
				{Type: ExprTokenOperator, Value: "<=", Pos: 10},
				{Type: ExprTokenOperator, Value: ">=", Pos: 13},
				{Type: ExprTokenEOF, Pos: 15},
			},
		},
		{
			name: "logical operators",
			expr: "& | !",
			want: []ExpressionToken{
				{Type: ExprTokenOperator, Value: "&", Pos: 0},
				{Type: ExprTokenOperator, Value: "|", Pos: 2},
				{Type: ExprTokenOperator, Value: "!", Pos: 4},
				{Type: ExprTokenEOF, Pos: 5},
			},
		},
		{
			name: "parentheses",
			expr: "( )",
			want: []ExpressionToken{
				{Type: ExprTokenLeftParen, Value: "(", Pos: 0},
				{Type: ExprTokenRightParen, Value: ")", Pos: 2},
				{Type: ExprTokenEOF, Pos: 3},
			},
		},
		{
			name: "brackets",
			expr: "[ ]",
			want: []ExpressionToken{
				{Type: ExprTokenOperator, Value: "[", Pos: 0},
				{Type: ExprTokenOperator, Value: "]", Pos: 2},
				{Type: ExprTokenEOF, Pos: 3},
			},
		},
		{
			name: "comma",
			expr: ",",
			want: []ExpressionToken{
				{Type: ExprTokenComma, Value: ",", Pos: 0},
				{Type: ExprTokenEOF, Pos: 1},
			},
		},
		{
			name: "dot operator",
			expr: ".",
			want: []ExpressionToken{
				{Type: ExprTokenOperator, Value: ".", Pos: 0},
				{Type: ExprTokenEOF, Pos: 1},
			},
		},
		{
			name: "simple arithmetic expression",
			expr: "a + b",
			want: []ExpressionToken{
				{Type: ExprTokenIdentifier, Value: "a", Pos: 0},
				{Type: ExprTokenOperator, Value: "+", Pos: 2},
				{Type: ExprTokenIdentifier, Value: "b", Pos: 4},
				{Type: ExprTokenEOF, Pos: 5},
			},
		},
		{
			name: "complex expression",
			expr: "(price * 1.2) + tax",
			want: []ExpressionToken{
				{Type: ExprTokenLeftParen, Value: "(", Pos: 0},
				{Type: ExprTokenIdentifier, Value: "price", Pos: 1},
				{Type: ExprTokenOperator, Value: "*", Pos: 7},
				{Type: ExprTokenNumber, Value: "1.2", Pos: 9},
				{Type: ExprTokenRightParen, Value: ")", Pos: 12},
				{Type: ExprTokenOperator, Value: "+", Pos: 14},
				{Type: ExprTokenIdentifier, Value: "tax", Pos: 16},
				{Type: ExprTokenEOF, Pos: 19},
			},
		},
		{
			name: "function call",
			expr: "max(a, b)",
			want: []ExpressionToken{
				{Type: ExprTokenIdentifier, Value: "max", Pos: 0},
				{Type: ExprTokenLeftParen, Value: "(", Pos: 3},
				{Type: ExprTokenIdentifier, Value: "a", Pos: 4},
				{Type: ExprTokenComma, Value: ",", Pos: 5},
				{Type: ExprTokenIdentifier, Value: "b", Pos: 7},
				{Type: ExprTokenRightParen, Value: ")", Pos: 8},
				{Type: ExprTokenEOF, Pos: 9},
			},
		},
		{
			name: "field access",
			expr: "customer.name",
			want: []ExpressionToken{
				{Type: ExprTokenIdentifier, Value: "customer", Pos: 0},
				{Type: ExprTokenOperator, Value: ".", Pos: 8},
				{Type: ExprTokenIdentifier, Value: "name", Pos: 9},
				{Type: ExprTokenEOF, Pos: 13},
			},
		},
		{
			name: "array access",
			expr: "items[0]",
			want: []ExpressionToken{
				{Type: ExprTokenIdentifier, Value: "items", Pos: 0},
				{Type: ExprTokenOperator, Value: "[", Pos: 5},
				{Type: ExprTokenNumber, Value: "0", Pos: 6},
				{Type: ExprTokenOperator, Value: "]", Pos: 7},
				{Type: ExprTokenEOF, Pos: 8},
			},
		},
		{
			name: "whitespace handling",
			expr: "  a  +  b  ",
			want: []ExpressionToken{
				{Type: ExprTokenIdentifier, Value: "a", Pos: 2},
				{Type: ExprTokenOperator, Value: "+", Pos: 5},
				{Type: ExprTokenIdentifier, Value: "b", Pos: 8},
				{Type: ExprTokenEOF, Pos: 11},
			},
		},
		{
			name: "invalid character",
			expr: "a @ b",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TokenizeExpression(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenizeExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TokenizeExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseExpression(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		want    string // String representation of the AST
		wantErr bool
	}{
		{
			name: "integer literal",
			expr: "42",
			want: "Literal(42)",
		},
		{
			name: "float literal",
			expr: "3.14",
			want: "Literal(3.14)",
		},
		{
			name: "string literal",
			expr: `"hello"`,
			want: "Literal(hello)",
		},
		{
			name: "boolean true",
			expr: "true",
			want: "Literal(true)",
		},
		{
			name: "boolean false",
			expr: "false",
			want: "Literal(false)",
		},
		{
			name: "null literal",
			expr: "null",
			want: "Literal(<nil>)",
		},
		{
			name: "nil literal",
			expr: "nil",
			want: "Literal(<nil>)",
		},
		{
			name: "variable",
			expr: "name",
			want: "Variable(name)",
		},
		{
			name: "parenthesized expression",
			expr: "(42)",
			want: "Literal(42)",
		},
		{
			name: "function call no args",
			expr: "foo()",
			want: "FunctionCall(foo, [])",
		},
		{
			name: "function call one arg",
			expr: "max(42)",
			want: "FunctionCall(max, [Literal(42)])",
		},
		{
			name: "function call multiple args",
			expr: `max(42, "hello", true)`,
			want: "FunctionCall(max, [Literal(42), Literal(hello), Literal(true)])",
		},
		{
			name: "nested function call",
			expr: "outer(inner(42))",
			want: "FunctionCall(outer, [FunctionCall(inner, [Literal(42)])])",
		},
		{
			name: "decimal starting with dot",
			expr: ".5",
			want: "Literal(0.5)",
		},
		{
			name: "empty expression",
			expr: "",
			wantErr: true,
		},
		{
			name: "unclosed parentheses",
			expr: "(42",
			wantErr: true,
		},
		{
			name: "unclosed function call",
			expr: "max(42",
			wantErr: true,
		},
		{
			name: "invalid function arguments",
			expr: "max(42,)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExpression(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.want {
				t.Errorf("ParseExpression() = %v, want %v", got.String(), tt.want)
			}
		})
	}
}

func TestExpressionNodeEvaluation(t *testing.T) {
	data := TemplateData{
		"name": "John",
		"age":  30,
		"active": true,
		"customer": map[string]interface{}{
			"name": "Jane",
		},
	}

	tests := []struct {
		name    string
		node    ExpressionNode
		want    interface{}
		wantErr bool
	}{
		{
			name: "literal integer",
			node: &LiteralNode{Value: 42},
			want: 42,
		},
		{
			name: "literal string",
			node: &LiteralNode{Value: "hello"},
			want: "hello",
		},
		{
			name: "literal boolean",
			node: &LiteralNode{Value: true},
			want: true,
		},
		{
			name: "literal null",
			node: &LiteralNode{Value: nil},
			want: nil,
		},
		{
			name: "variable exists",
			node: &VariableNode{Name: "name"},
			want: "John",
		},
		{
			name: "variable missing",
			node: &VariableNode{Name: "missing"},
			want: nil,
		},
		{
			name: "nested variable",
			node: &VariableNode{Name: "customer.name"},
			want: "Jane",
		},
		{
			name: "binary operation addition",
			node: &BinaryOpNode{
				Left:     &LiteralNode{Value: 1},
				Operator: "+",
				Right:    &LiteralNode{Value: 2},
			},
			want: 3,
		},
		{
			name: "logical NOT true",
			node: &UnaryOpNode{
				Operator: "!",
				Operand:  &LiteralNode{Value: true},
			},
			want: false,
		},
		{
			name: "logical NOT false",
			node: &UnaryOpNode{
				Operator: "!",
				Operand:  &LiteralNode{Value: false},
			},
			want: true,
		},
		{
			name: "unary minus",
			node: &UnaryOpNode{
				Operator: "-",
				Operand:  &LiteralNode{Value: 5},
			},
			want: -5,
		},
		{
			name: "unary plus",
			node: &UnaryOpNode{
				Operator: "+",
				Operand:  &LiteralNode{Value: 5},
			},
			want: 5,
		},
		{
			name: "function call not implemented",
			node: &FunctionCallNode{
				Name: "max",
				Args: []ExpressionNode{
					&LiteralNode{Value: 1},
					&LiteralNode{Value: 2},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.node.Evaluate(data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpressionNode.Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExpressionNode.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMathematicalOperations(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		want    interface{}
		wantErr bool
	}{
		// Addition
		{
			name: "integer addition",
			expr: "2 + 3",
			want: 5,
		},
		{
			name: "float addition",
			expr: "2.5 + 1.5",
			want: 4.0,
		},
		{
			name: "mixed number addition",
			expr: "2 + 1.5",
			want: 3.5,
		},
		{
			name: "string concatenation",
			expr: `"hello" + " world"`,
			want: "hello world",
		},
		{
			name: "string and number concatenation",
			expr: `"value: " + 42`,
			want: "value: 42",
		},
		// Subtraction
		{
			name: "integer subtraction",
			expr: "5 - 3",
			want: 2,
		},
		{
			name: "float subtraction",
			expr: "5.5 - 2.2",
			want: 3.3,
		},
		// Multiplication
		{
			name: "integer multiplication",
			expr: "4 * 3",
			want: 12,
		},
		{
			name: "float multiplication",
			expr: "2.5 * 2.0",
			want: 5.0,
		},
		// Division
		{
			name: "integer division (whole result)",
			expr: "6 / 2",
			want: 3,
		},
		{
			name: "integer division (float result)",
			expr: "5 / 2",
			want: 2.5,
		},
		{
			name: "float division",
			expr: "7.5 / 2.5",
			want: 3.0,
		},
		{
			name: "division by zero",
			expr: "5 / 0",
			wantErr: true,
		},
		// Modulo
		{
			name: "modulo operation",
			expr: "10 % 3",
			want: 1,
		},
		{
			name: "modulo by zero",
			expr: "5 % 0",
			wantErr: true,
		},
		// Operator precedence
		{
			name: "multiplication before addition",
			expr: "2 + 3 * 4",
			want: 14,
		},
		{
			name: "parentheses override precedence",
			expr: "(2 + 3) * 4",
			want: 20,
		},
		{
			name: "complex precedence",
			expr: "2 + 3 * 4 - 1",
			want: 13,
		},
		{
			name: "division and multiplication",
			expr: "8 / 2 * 3",
			want: 12,
		},
		// Comparison operators
		{
			name: "less than true",
			expr: "2 < 3",
			want: true,
		},
		{
			name: "less than false",
			expr: "3 < 2",
			want: false,
		},
		{
			name: "greater than true",
			expr: "3 > 2",
			want: true,
		},
		{
			name: "less than or equal",
			expr: "2 <= 2",
			want: true,
		},
		{
			name: "greater than or equal",
			expr: "3 >= 2",
			want: true,
		},
		{
			name: "equality true",
			expr: "2 == 2",
			want: true,
		},
		{
			name: "equality false",
			expr: "2 == 3",
			want: false,
		},
		{
			name: "inequality true",
			expr: "2 != 3",
			want: true,
		},
		{
			name: "inequality false",
			expr: "2 != 2",
			want: false,
		},
		// Logical operators
		{
			name: "logical and true",
			expr: "true & true",
			want: true,
		},
		{
			name: "logical and false",
			expr: "true & false",
			want: false,
		},
		{
			name: "logical or true",
			expr: "false | true",
			want: true,
		},
		{
			name: "logical or false",
			expr: "false | false",
			want: false,
		},
		// Complex expressions
		{
			name: "complex arithmetic",
			expr: "(2 + 3) * (4 - 1)",
			want: 15,
		},
		{
			name: "comparison with arithmetic",
			expr: "2 + 3 > 4",
			want: true,
		},
		{
			name: "logical with comparison",
			expr: "2 > 1 & 3 < 4",
			want: true,
		},
		// Unary operators
		{
			name: "logical NOT true",
			expr: "!true",
			want: false,
		},
		{
			name: "logical NOT false",
			expr: "!false",
			want: true,
		},
		{
			name: "logical NOT with variable",
			expr: "!enabled",
			want: true,
		},
		{
			name: "unary minus integer",
			expr: "-5",
			want: -5,
		},
		{
			name: "unary minus float",
			expr: "-3.14",
			want: -3.14,
		},
		{
			name: "unary plus",
			expr: "+5",
			want: 5,
		},
		{
			name: "double negative",
			expr: "--5",
			want: 5,
		},
		{
			name: "NOT with comparison",
			expr: "!(2 > 3)",
			want: true,
		},
		{
			name: "complex unary expression",
			expr: "!(false | (2 > 3))",
			want: true,
		},
		{
			name: "unary in arithmetic",
			expr: "-5 + 10",
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseExpression(tt.expr)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseExpression() error = %v", err)
				}
				return
			}

			// Use test data that includes 'enabled' for unary tests
			testData := TemplateData{
				"enabled": false,
			}
			got, err := node.Evaluate(testData)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFieldAndIndexAccess(t *testing.T) {
	data := TemplateData{
		"customer": map[string]interface{}{
			"name": "John",
			"address": map[string]interface{}{
				"city": "New York",
			},
		},
		"items": []interface{}{
			map[string]interface{}{"name": "Item 1", "price": 10.5},
			map[string]interface{}{"name": "Item 2", "price": 20.0},
		},
		"numbers": []interface{}{1, 2, 3, 4, 5},
	}

	tests := []struct {
		name    string
		expr    string
		want    interface{}
		wantErr bool
	}{
		{
			name: "simple field access",
			expr: "customer.name",
			want: "John",
		},
		{
			name: "nested field access",
			expr: "customer.address.city",
			want: "New York",
		},
		{
			name: "array index access",
			expr: "numbers[0]",
			want: 1,
		},
		{
			name: "array index with field access",
			expr: "items[0].name",
			want: "Item 1",
		},
		{
			name: "mixed access with arithmetic",
			expr: "items[0].price + items[1].price",
			want: 30.5,
		},
		{
			name: "array index out of bounds",
			expr: "numbers[10]",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseExpression(tt.expr)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseExpression() error = %v", err)
				}
				return
			}

			got, err := node.Evaluate(data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpressionIntegration(t *testing.T) {
	data := TemplateData{
		"name": "John",
		"age":  30,
		"items": []interface{}{
			map[string]interface{}{"name": "Item 1", "price": 10.0},
			map[string]interface{}{"name": "Item 2", "price": 20.0},
		},
	}

	tests := []struct {
		name    string
		expr    string
		want    interface{}
		wantErr bool
	}{
		{
			name: "simple literal",
			expr: "42",
			want: 42,
		},
		{
			name: "simple variable",
			expr: "name",
			want: "John",
		},
		{
			name: "array access (returns whole array for now)",
			expr: "items",
			want: []interface{}{
				map[string]interface{}{"name": "Item 1", "price": 10.0},
				map[string]interface{}{"name": "Item 2", "price": 20.0},
			},
		},
		{
			name: "boolean literal",
			expr: "true",
			want: true,
		},
		{
			name: "function call (not implemented yet)",
			expr: "max(1, 2)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := ParseExpression(tt.expr)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseExpression() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			got, err := node.Evaluate(data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}