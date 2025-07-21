package stencil

import (
	"testing"
)

func TestLengthFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// String length tests
		{
			name: "length() of simple string",
			args: []interface{}{"hello"},
			want: 5,
		},
		{
			name: "length() of empty string",
			args: []interface{}{""},
			want: 0,
		},
		{
			name: "length() of string with spaces",
			args: []interface{}{"hello world"},
			want: 11,
		},
		{
			name: "length() of string with unicode",
			args: []interface{}{"café"},
			want: 4,
		},
		{
			name: "length() of string with emojis",
			args: []interface{}{"hello 😊 world"},
			want: 13,
		},
		// Array length tests
		{
			name: "length() of array",
			args: []interface{}{[]interface{}{"a", "b", "c"}},
			want: 3,
		},
		{
			name: "length() of empty array",
			args: []interface{}{[]interface{}{}},
			want: 0,
		},
		{
			name: "length() of array with nil values",
			args: []interface{}{[]interface{}{"a", nil, "b", nil, "c"}},
			want: 5,
		},
		{
			name: "length() of string slice",
			args: []interface{}{[]string{"one", "two", "three"}},
			want: 3,
		},
		{
			name: "length() of int slice",
			args: []interface{}{[]int{1, 2, 3, 4, 5}},
			want: 5,
		},
		// Map length tests
		{
			name: "length() of map",
			args: []interface{}{map[string]interface{}{"a": 1, "b": 2, "c": 3}},
			want: 3,
		},
		{
			name: "length() of empty map",
			args: []interface{}{map[string]interface{}{}},
			want: 0,
		},
		// Special cases
		{
			name: "length() of nil",
			args: []interface{}{nil},
			want: 0,
		},
		{
			name: "length() of number",
			args: []interface{}{12345},
			want: 5, // length of string representation
		},
		{
			name: "length() of float",
			args: []interface{}{123.45},
			want: 6, // length of string representation
		},
		{
			name: "length() of boolean true",
			args: []interface{}{true},
			want: 4, // length of "true"
		},
		{
			name: "length() of boolean false",
			args: []interface{}{false},
			want: 5, // length of "false"
		},
		// Error cases
		{
			name:    "length() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "length() with too many arguments",
			args:    []interface{}{"hello", "world"},
			wantErr: true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("length")
			if !exists {
				t.Skipf("Function length not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("length() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("length() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLengthFunctionInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		// length() with variables
		{
			name: "length() of string variable",
			expr: "length(name)",
			data: TemplateData{"name": "John Doe"},
			want: 8,
		},
		{
			name: "length() of array variable",
			expr: "length(items)",
			data: TemplateData{
				"items": []interface{}{"apple", "banana", "cherry"},
			},
			want: 3,
		},
		{
			name: "length() of map variable",
			expr: "length(user)",
			data: TemplateData{
				"user": map[string]interface{}{
					"name": "Alice",
					"age":  30,
					"city": "NYC",
				},
			},
			want: 3,
		},
		// length() with nested fields
		{
			name: "length() of nested field",
			expr: "length(user.name)",
			data: TemplateData{
				"user": map[string]interface{}{
					"name": "Alice Smith",
				},
			},
			want: 11,
		},
		{
			name: "length() of nested array",
			expr: "length(order.items)",
			data: TemplateData{
				"order": map[string]interface{}{
					"items": []interface{}{"a", "b", "c", "d"},
				},
			},
			want: 4,
		},
		// length() with function results
		{
			name: "length() of join result",
			expr: "length(join(words, \" \"))",
			data: TemplateData{
				"words": []interface{}{"hello", "world"},
			},
			want: 11,
		},
		{
			name: "length() of uppercase result",
			expr: "length(uppercase(\"test\"))",
			data: TemplateData{},
			want: 4,
		},
		{
			name: "length() of coalesce result",
			expr: "length(coalesce(missing, \"default\"))",
			data: TemplateData{"missing": nil},
			want: 7,
		},
		// length() in conditions
		{
			name: "length() comparison",
			expr: "length(name) > 5",
			data: TemplateData{"name": "Alice"},
			want: false,
		},
		{
			name: "length() equality check",
			expr: "length(items) == 3",
			data: TemplateData{
				"items": []interface{}{"a", "b", "c"},
			},
			want: true,
		},
		// Complex expressions
		{
			name: "length() with arithmetic",
			expr: "length(text) * 2",
			data: TemplateData{"text": "hello"},
			want: 10,
		},
		{
			name: "length() of literal string",
			expr: "length(\"test string\")",
			data: TemplateData{},
			want: 11,
		},
		{
			name: "length() of empty literal",
			expr: "length(\"\")",
			data: TemplateData{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if function not implemented
			registry := GetDefaultFunctionRegistry()
			if _, exists := registry.GetFunction("length"); !exists {
				t.Skip("Function length not yet implemented")
				return
			}

			expr, err := ParseExpression(tt.expr)
			if err != nil {
				t.Errorf("ParseExpression() error = %v", err)
				return
			}

			got, err := expr.Evaluate(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Expression.Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLengthFunctionEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
		want interface{}
	}{
		// Multi-byte character handling
		{
			name: "length() of Japanese characters",
			args: []interface{}{"こんにちは"},
			want: 5,
		},
		{
			name: "length() of mixed ASCII and Unicode",
			args: []interface{}{"Hello世界"},
			want: 7,
		},
		// Different collection types
		{
			name: "length() of interface{} containing string",
			args: []interface{}{interface{}("test")},
			want: 4,
		},
		{
			name: "length() of nested arrays",
			args: []interface{}{[]interface{}{
				[]interface{}{1, 2},
				[]interface{}{3, 4, 5},
			}},
			want: 2, // outer array length
		},
		// Numeric edge cases
		{
			name: "length() of negative number",
			args: []interface{}{-123},
			want: 4, // length of "-123"
		},
		{
			name: "length() of zero",
			args: []interface{}{0},
			want: 1, // length of "0"
		},
		{
			name: "length() of large number",
			args: []interface{}{1234567890},
			want: 10,
		},
		// Special string cases
		{
			name: "length() of string with newlines",
			args: []interface{}{"line1\nline2\nline3"},
			want: 17,
		},
		{
			name: "length() of string with tabs",
			args: []interface{}{"a\tb\tc"},
			want: 5,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("length")
			if !exists {
				t.Skipf("Function length not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if err != nil {
				t.Errorf("length() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("length() = %v, want %v", got, tt.want)
			}
		})
	}
}