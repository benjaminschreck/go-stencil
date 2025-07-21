package stencil

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestFunctionRegistry(t *testing.T) {
	registry := NewFunctionRegistry()

	// Test RegisterFunction
	testFunc := NewSimpleFunction("test", 1, 2, func(args ...interface{}) (interface{}, error) {
		return "test result", nil
	})

	err := registry.RegisterFunction(testFunc)
	if err != nil {
		t.Errorf("RegisterFunction() error = %v", err)
	}

	// Test GetFunction
	fn, exists := registry.GetFunction("test")
	if !exists {
		t.Errorf("GetFunction() function not found")
	}
	if fn.Name() != "test" {
		t.Errorf("GetFunction() name = %v, want test", fn.Name())
	}

	// Test function doesn't exist
	_, exists = registry.GetFunction("nonexistent")
	if exists {
		t.Errorf("GetFunction() found nonexistent function")
	}

	// Test ListFunctions
	functions := registry.ListFunctions()
	if len(functions) != 1 || functions[0] != "test" {
		t.Errorf("ListFunctions() = %v, want [test]", functions)
	}

	// Test registering function with empty name
	emptyNameFunc := NewSimpleFunction("", 0, 0, func(args ...interface{}) (interface{}, error) {
		return nil, nil
	})
	err = registry.RegisterFunction(emptyNameFunc)
	if err == nil {
		t.Errorf("RegisterFunction() should fail with empty name")
	}
}

func TestSimpleFunctionImpl(t *testing.T) {
	tests := []struct {
		name     string
		function Function
		args     []interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name: "function with correct args",
			function: NewSimpleFunction("add", 2, 2, func(args ...interface{}) (interface{}, error) {
				a := args[0].(int)
				b := args[1].(int)
				return a + b, nil
			}),
			args: []interface{}{5, 3},
			want: 8,
		},
		{
			name: "function with too few args",
			function: NewSimpleFunction("add", 2, 2, func(args ...interface{}) (interface{}, error) {
				return nil, nil
			}),
			args:    []interface{}{5},
			wantErr: true,
		},
		{
			name: "function with too many args",
			function: NewSimpleFunction("add", 2, 2, func(args ...interface{}) (interface{}, error) {
				return nil, nil
			}),
			args:    []interface{}{5, 3, 1},
			wantErr: true,
		},
		{
			name: "function with unlimited args",
			function: NewSimpleFunction("concat", 1, -1, func(args ...interface{}) (interface{}, error) {
				result := ""
				for _, arg := range args {
					result += fmt.Sprintf("%v", arg)
				}
				return result, nil
			}),
			args: []interface{}{"hello", " ", "world"},
			want: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.function.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Function.Call() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Function.Call() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuiltinFunctions(t *testing.T) {
	registry := GetDefaultFunctionRegistry()

	tests := []struct {
		name     string
		funcName string
		args     []interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "empty() with empty string",
			funcName: "empty",
			args:     []interface{}{""},
			want:     true,
		},
		{
			name:     "empty() with non-empty string",
			funcName: "empty",
			args:     []interface{}{"hello"},
			want:     false,
		},
		{
			name:     "empty() with nil",
			funcName: "empty",
			args:     []interface{}{nil},
			want:     true,
		},
		{
			name:     "empty() with zero",
			funcName: "empty",
			args:     []interface{}{0},
			want:     true,
		},
		{
			name:     "empty() with false",
			funcName: "empty",
			args:     []interface{}{false},
			want:     true,
		},
		{
			name:     "empty() with empty slice",
			funcName: "empty",
			args:     []interface{}{[]interface{}{}},
			want:     true,
		},
		{
			name:     "coalesce() with first non-empty",
			funcName: "coalesce",
			args:     []interface{}{"hello", "world"},
			want:     "hello",
		},
		{
			name:     "coalesce() with first empty",
			funcName: "coalesce",
			args:     []interface{}{"", "world"},
			want:     "world",
		},
		{
			name:     "coalesce() with all empty",
			funcName: "coalesce",
			args:     []interface{}{"", nil, 0},
			want:     nil,
		},
		{
			name:     "coalesce() mixed types",
			funcName: "coalesce",
			args:     []interface{}{nil, "", 42, "backup"},
			want:     42,
		},
		{
			name:     "list() with no args",
			funcName: "list",
			args:     []interface{}{},
			want:     []interface{}{},
		},
		{
			name:     "list() with multiple args",
			funcName: "list",
			args:     []interface{}{"a", "b", "c"},
			want:     []interface{}{"a", "b", "c"},
		},
		{
			name:     "list() mixed types",
			funcName: "list",
			args:     []interface{}{1, "hello", true},
			want:     []interface{}{1, "hello", true},
		},
		{
			name:     "pageBreak() with no args",
			funcName: "pageBreak",
			args:     []interface{}{},
			want:     &OOXMLFragment{Content: &Break{Type: "page"}},
		},
		{
			name:     "range() with single arg",
			funcName: "range",
			args:     []interface{}{5},
			want:     []interface{}{0, 1, 2, 3, 4},
		},
		{
			name:     "range() with start and end",
			funcName: "range",
			args:     []interface{}{1, 5},
			want:     []interface{}{1, 2, 3, 4},
		},
		{
			name:     "range() with start, end, and step",
			funcName: "range",
			args:     []interface{}{1, 6, 2},
			want:     []interface{}{1, 3, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				t.Errorf("Function %s not found in registry", tt.funcName)
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Function %s error = %v, wantErr %v", tt.funcName, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Special handling for OOXML fragment comparison
				if wantFragment, ok := tt.want.(*OOXMLFragment); ok {
					if gotFragment, ok := got.(*OOXMLFragment); ok {
						if wantBreak, ok := wantFragment.Content.(*Break); ok {
							if gotBreak, ok := gotFragment.Content.(*Break); ok {
								if wantBreak.Type != gotBreak.Type {
									t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
								}
								return
							}
						}
						t.Errorf("Function %s OOXML fragment content type mismatch = %v, want %v", tt.funcName, got, tt.want)
						return
					}
					t.Errorf("Function %s return type mismatch = %v, want %v", tt.funcName, got, tt.want)
					return
				}
				
				// Special handling for slice comparison
				if ttWantSlice, ok := tt.want.([]interface{}); ok {
					if gotSlice, ok := got.([]interface{}); ok {
						if len(ttWantSlice) != len(gotSlice) {
							t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
							return
						}
						for i := range ttWantSlice {
							if ttWantSlice[i] != gotSlice[i] {
								t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
								return
							}
						}
						return
					}
				}
				
				if got != tt.want {
					t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
				}
			}
		})
	}
}

func TestFunctionCallInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		{
			name: "simple function call",
			expr: "empty(name)",
			data: TemplateData{"name": ""},
			want: true,
		},
		{
			name: "function call with literal",
			expr: "empty(\"hello\")",
			data: TemplateData{},
			want: false,
		},
		{
			name: "coalesce function call",
			expr: "coalesce(name, \"default\")",
			data: TemplateData{"name": ""},
			want: "default",
		},
		{
			name: "list function call",
			expr: "list(1, 2, 3)",
			data: TemplateData{},
			want: []interface{}{1, 2, 3},
		},
		{
			name: "nested function calls",
			expr: "empty(coalesce(name, \"\"))",
			data: TemplateData{"name": nil},
			want: true,
		},
		{
			name: "function call with arithmetic",
			expr: "list(1 + 2, 3 * 4)",
			data: TemplateData{},
			want: []interface{}{3, 12},
		},
		{
			name:    "unknown function",
			expr:    "unknown()",
			data:    TemplateData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if !tt.wantErr {
				// Special handling for slice comparison
				if ttWantSlice, ok := tt.want.([]interface{}); ok {
					if gotSlice, ok := got.([]interface{}); ok {
						if len(ttWantSlice) != len(gotSlice) {
							t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
							return
						}
						for i := range ttWantSlice {
							if ttWantSlice[i] != gotSlice[i] {
								t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
								return
							}
						}
						return
					}
				}
				
				if got != tt.want {
					t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCallFunction(t *testing.T) {
	tests := []struct {
		name    string
		funcName string
		data    TemplateData
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name:     "call empty function",
			funcName: "empty",
			data:     TemplateData{},
			args:     []interface{}{"test"},
			want:     false,
		},
		{
			name:     "call with custom registry",
			funcName: "custom",
			data: TemplateData{
				"__functions__": func() FunctionRegistry {
					registry := NewFunctionRegistry()
					customFunc := NewSimpleFunction("custom", 0, 0, func(args ...interface{}) (interface{}, error) {
						return "custom result", nil
					})
					registry.RegisterFunction(customFunc)
					return registry
				}(),
			},
			args: []interface{}{},
			want: "custom result",
		},
		{
			name:     "unknown function",
			funcName: "unknown",
			data:     TemplateData{},
			args:     []interface{}{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CallFunction(tt.funcName, tt.data, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CallFunction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("CallFunction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomFunctionRegistration(t *testing.T) {
	// Test registering a custom function globally
	customFunc := NewSimpleFunction("multiply", 2, 2, func(args ...interface{}) (interface{}, error) {
		a, ok1 := args[0].(int)
		b, ok2 := args[1].(int)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("multiply requires integer arguments")
		}
		return a * b, nil
	})

	err := RegisterGlobalFunction("multiply", customFunc)
	if err != nil {
		t.Errorf("RegisterGlobalFunction() error = %v", err)
	}

	// Test that the function is available in the global registry
	registry := GetDefaultFunctionRegistry()
	fn, exists := registry.GetFunction("multiply")
	if !exists {
		t.Errorf("Custom function not found in global registry")
	}

	result, err := fn.Call(5, 3)
	if err != nil {
		t.Errorf("Custom function call error = %v", err)
	}
	if result != 15 {
		t.Errorf("Custom function result = %v, want 15", result)
	}

	// Test using the function in an expression
	expr, err := ParseExpression("multiply(6, 7)")
	if err != nil {
		t.Errorf("ParseExpression() error = %v", err)
		return
	}

	got, err := expr.Evaluate(TemplateData{})
	if err != nil {
		t.Errorf("Expression.Evaluate() error = %v", err)
		return
	}
	if got != 42 {
		t.Errorf("Expression.Evaluate() = %v, want 42", got)
	}
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"nil", nil, true},
		{"true", true, false},
		{"false", false, true},
		{"zero int", 0, true},
		{"positive int", 1, false},
		{"negative int", -1, false},
		{"zero float", 0.0, true},
		{"positive float", 1.5, false},
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"empty slice", []interface{}{}, true},
		{"non-empty slice", []interface{}{1}, false},
		{"empty string slice", []string{}, true},
		{"non-empty string slice", []string{"a"}, false},
		{"empty int slice", []int{}, true},
		{"non-empty int slice", []int{1}, false},
		{"empty map", map[string]interface{}{}, true},
		{"non-empty map", map[string]interface{}{"key": "value"}, false},
		{"non-nil object", struct{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEmpty(tt.value)
			if got != tt.want {
				t.Errorf("isEmpty(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestDataFunction(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		{
			name: "data() returns entire context",
			expr: "data()",
			data: TemplateData{
				"name": "John",
				"age":  30,
				"tags": []interface{}{"developer", "go"},
			},
			want: TemplateData{
				"name": "John",
				"age":  30,
				"tags": []interface{}{"developer", "go"},
			},
		},
		{
			name: "data() in expressions",
			expr: "data()['name']",
			data: TemplateData{"name": "Alice"},
			want: "Alice",
		},
		{
			name: "data() with field access",
			expr: "data().user.email",
			data: TemplateData{
				"user": map[string]interface{}{
					"email": "test@example.com",
				},
			},
			want: "test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if !tt.wantErr {
				// For TemplateData comparison
				if wantMap, ok := tt.want.(TemplateData); ok {
					if gotMap, ok := got.(TemplateData); ok {
						for key, wantVal := range wantMap {
							gotVal, exists := gotMap[key]
							if !exists {
								t.Errorf("Missing key %s in result", key)
								continue
							}
							// Deep comparison for slices
							if !compareValues(gotVal, wantVal) {
								t.Errorf("For key %s: got %v, want %v", key, gotVal, wantVal)
							}
						}
						return
					}
				}
				
				if got != tt.want {
					t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMapFunction(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		data    interface{}
		want    interface{}
		wantErr bool
	}{
		{
			name: "map simple field from array of objects",
			path: "name",
			data: []interface{}{
				map[string]interface{}{"name": "Alice", "age": 30},
				map[string]interface{}{"name": "Bob", "age": 25},
				map[string]interface{}{"name": "Charlie", "age": 35},
			},
			want: []interface{}{"Alice", "Bob", "Charlie"},
		},
		{
			name: "map nested field",
			path: "user.email",
			data: []interface{}{
				map[string]interface{}{
					"user": map[string]interface{}{
						"email": "alice@example.com",
					},
				},
				map[string]interface{}{
					"user": map[string]interface{}{
						"email": "bob@example.com",
					},
				},
			},
			want: []interface{}{"alice@example.com", "bob@example.com"},
		},
		{
			name: "map from single object",
			path: "title",
			data: map[string]interface{}{"title": "Manager", "dept": "IT"},
			want: []interface{}{"Manager"},
		},
		{
			name: "map with missing fields",
			path: "missing",
			data: []interface{}{
				map[string]interface{}{"name": "Alice"},
				map[string]interface{}{"missing": "Found", "name": "Bob"},
			},
			want: []interface{}{"Found"},
		},
		{
			name: "map from nil",
			path: "field",
			data: nil,
			want: []interface{}{},
		},
		{
			name: "map with arrays in result",
			path: "tags",
			data: []interface{}{
				map[string]interface{}{"tags": []interface{}{"go", "rust"}},
				map[string]interface{}{"tags": []interface{}{"python"}},
			},
			want: []interface{}{"go", "rust", "python"},
		},
		{
			name: "empty path returns original data",
			path: "",
			data: []interface{}{1, 2, 3},
			want: []interface{}{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapExtract(tt.path, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapExtract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !compareSlices(got.([]interface{}), tt.want.([]interface{})) {
					t.Errorf("mapExtract() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMapFunctionInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		{
			name: "map function in template",
			expr: "map(\"price\", items)",
			data: TemplateData{
				"items": []interface{}{
					map[string]interface{}{"name": "Item1", "price": 10.5},
					map[string]interface{}{"name": "Item2", "price": 20.0},
				},
			},
			want: []interface{}{10.5, 20.0},
		},
		{
			name: "map with nested path",
			expr: "map(\"product.price\", orders)",
			data: TemplateData{
				"orders": []interface{}{
					map[string]interface{}{
						"id": 1,
						"product": map[string]interface{}{
							"name": "Widget",
							"price": 99.99,
						},
					},
					map[string]interface{}{
						"id": 2,
						"product": map[string]interface{}{
							"name": "Gadget",
							"price": 149.99,
						},
					},
				},
			},
			want: []interface{}{99.99, 149.99},
		},
		{
			name: "map in for loop context",
			expr: "list(map(\"name\", users))",
			data: TemplateData{
				"users": []interface{}{
					map[string]interface{}{"name": "Alice"},
					map[string]interface{}{"name": "Bob"},
				},
			},
			want: []interface{}{[]interface{}{"Alice", "Bob"}},
		},
		{
			name:    "map with non-string first argument",
			expr:    "map(123, items)",
			data:    TemplateData{"items": []interface{}{}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if !tt.wantErr {
				if !compareValues(got, tt.want) {
					t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Helper function to compare values including slices
func compareValues(a, b interface{}) bool {
	if aSlice, ok := a.([]interface{}); ok {
		if bSlice, ok := b.([]interface{}); ok {
			return compareSlices(aSlice, bSlice)
		}
		return false
	}
	return a == b
}

// Helper function to compare slices
func compareSlices(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !compareValues(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestTypeConversionFunctions(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		input    interface{}
		want     interface{}
		wantErr  bool
	}{
		// str() function tests
		{
			name:     "str() with string",
			funcName: "str",
			input:    "hello",
			want:     "hello",
		},
		{
			name:     "str() with integer",
			funcName: "str",
			input:    42,
			want:     "42",
		},
		{
			name:     "str() with float",
			funcName: "str",
			input:    3.14,
			want:     "3.14",
		},
		{
			name:     "str() with boolean true",
			funcName: "str",
			input:    true,
			want:     "true",
		},
		{
			name:     "str() with boolean false",
			funcName: "str",
			input:    false,
			want:     "false",
		},
		{
			name:     "str() with nil",
			funcName: "str",
			input:    nil,
			want:     "",
		},
		{
			name:     "str() with slice",
			funcName: "str",
			input:    []interface{}{1, 2, 3},
			want:     "[1 2 3]",
		},
		{
			name:     "str() with map",
			funcName: "str",
			input:    map[string]interface{}{"key": "value"},
			want:     "map[key:value]",
		},
		// integer() function tests
		{
			name:     "integer() with integer",
			funcName: "integer",
			input:    42,
			want:     42,
		},
		{
			name:     "integer() with float",
			funcName: "integer",
			input:    3.14,
			want:     3,
		},
		{
			name:     "integer() with float64",
			funcName: "integer",
			input:    3.99,
			want:     3,
		},
		{
			name:     "integer() with string integer",
			funcName: "integer",
			input:    "42",
			want:     42,
		},
		{
			name:     "integer() with string float",
			funcName: "integer",
			input:    "3.14",
			want:     3,
		},
		{
			name:     "integer() with boolean true",
			funcName: "integer",
			input:    true,
			want:     1,
		},
		{
			name:     "integer() with boolean false",
			funcName: "integer",
			input:    false,
			want:     0,
		},
		{
			name:     "integer() with nil",
			funcName: "integer",
			input:    nil,
			want:     nil,
		},
		{
			name:     "integer() with invalid string",
			funcName: "integer",
			input:    "abc",
			wantErr:  true,
		},
		// decimal() function tests
		{
			name:     "decimal() with integer",
			funcName: "decimal",
			input:    42,
			want:     42.0,
		},
		{
			name:     "decimal() with float32",
			funcName: "decimal",
			input:    float32(3.14),
			want:     float64(float32(3.14)), // Account for float32 precision
		},
		{
			name:     "decimal() with float64",
			funcName: "decimal",
			input:    3.14159,
			want:     3.14159,
		},
		{
			name:     "decimal() with string float",
			funcName: "decimal",
			input:    "3.14159",
			want:     3.14159,
		},
		{
			name:     "decimal() with string integer",
			funcName: "decimal",
			input:    "42",
			want:     42.0,
		},
		{
			name:     "decimal() with boolean true",
			funcName: "decimal",
			input:    true,
			want:     1.0,
		},
		{
			name:     "decimal() with boolean false",
			funcName: "decimal",
			input:    false,
			want:     0.0,
		},
		{
			name:     "decimal() with nil",
			funcName: "decimal",
			input:    nil,
			want:     nil,
		},
		{
			name:     "decimal() with invalid string",
			funcName: "decimal",
			input:    "not a number",
			wantErr:  true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				t.Errorf("Function %s not found in registry", tt.funcName)
				return
			}

			got, err := fn.Call(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Function %s error = %v, wantErr %v", tt.funcName, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.funcName == "decimal" && got != nil && tt.want != nil {
					// Special handling for float comparison
					gotFloat, _ := got.(float64)
					wantFloat, _ := tt.want.(float64)
					if gotFloat != wantFloat {
						t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
					}
				} else if got != tt.want {
					t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
				}
			}
		})
	}
}

func TestTypeConversionInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		{
			name: "str() in concatenation",
			expr: "str(count) + \" items\"",
			data: TemplateData{"count": 42},
			want: "42 items",
		},
		{
			name: "integer() in arithmetic",
			expr: "integer(\"10\") + integer(\"20\")",
			data: TemplateData{},
			want: 30,
		},
		{
			name: "decimal() in arithmetic",
			expr: "decimal(\"3.14\") + decimal(\"2.86\")",
			data: TemplateData{},
			want: 6.0,
		},
		{
			name: "type conversion chain",
			expr: "str(integer(decimal(\"3.7\")))",
			data: TemplateData{},
			want: "3",
		},
		{
			name: "integer() with variable",
			expr: "integer(price)",
			data: TemplateData{"price": "99.99"},
			want: 99,
		},
		{
			name: "decimal() with calculation",
			expr: "decimal(total) / decimal(count)",
			data: TemplateData{"total": "100", "count": "3"},
			want: 100.0 / 3.0,
		},
		{
			name: "str() with nil handling",
			expr: "coalesce(str(missing), \"default\")",
			data: TemplateData{},
			want: "default",
		},
		{
			name: "type conversion in conditionals",
			expr: "integer(enabled)",
			data: TemplateData{"enabled": true},
			want: 1,
		},
		{
			name:    "integer() with non-numeric string",
			expr:    "integer(\"abc\")",
			data:    TemplateData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if !tt.wantErr {
				// Special handling for float comparison
				if gotFloat, ok := got.(float64); ok {
					if wantFloat, ok := tt.want.(float64); ok {
						if gotFloat != wantFloat {
							t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
						}
						return
					}
				}
				
				if got != tt.want {
					t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMathFunctions(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		input    interface{}
		want     interface{}
		wantErr  bool
	}{
		// round() function tests
		{
			name:     "round() positive float",
			funcName: "round",
			input:    3.14,
			want:     3,
		},
		{
			name:     "round() positive float round up",
			funcName: "round",
			input:    3.7,
			want:     4,
		},
		{
			name:     "round() negative float",
			funcName: "round",
			input:    -2.3,
			want:     -2,
		},
		{
			name:     "round() negative float round down",
			funcName: "round",
			input:    -2.7,
			want:     -3,
		},
		{
			name:     "round() exactly half positive",
			funcName: "round",
			input:    2.5,
			want:     3,
		},
		{
			name:     "round() exactly half negative",
			funcName: "round",
			input:    -2.5,
			want:     -3,
		},
		{
			name:     "round() integer",
			funcName: "round",
			input:    5,
			want:     5,
		},
		{
			name:     "round() zero",
			funcName: "round",
			input:    0.0,
			want:     0,
		},
		{
			name:     "round() nil",
			funcName: "round",
			input:    nil,
			want:     nil,
		},
		{
			name:     "round() string number",
			funcName: "round",
			input:    "3.7",
			want:     4,
		},
		{
			name:     "round() invalid string",
			funcName: "round",
			input:    "abc",
			wantErr:  true,
		},
		{
			name:     "round() non-number type",
			funcName: "round",
			input:    []interface{}{1, 2, 3},
			wantErr:  true,
		},
		// floor() function tests
		{
			name:     "floor() positive float",
			funcName: "floor",
			input:    3.99,
			want:     3,
		},
		{
			name:     "floor() negative float",
			funcName: "floor",
			input:    -2.1,
			want:     -3,
		},
		{
			name:     "floor() exactly integer",
			funcName: "floor",
			input:    5.0,
			want:     5,
		},
		{
			name:     "floor() small positive",
			funcName: "floor",
			input:    0.1,
			want:     0,
		},
		{
			name:     "floor() small negative",
			funcName: "floor",
			input:    -0.1,
			want:     -1,
		},
		{
			name:     "floor() integer",
			funcName: "floor",
			input:    42,
			want:     42,
		},
		{
			name:     "floor() nil",
			funcName: "floor",
			input:    nil,
			want:     nil,
		},
		{
			name:     "floor() string number",
			funcName: "floor",
			input:    "3.9",
			want:     3,
		},
		{
			name:     "floor() invalid input",
			funcName: "floor",
			input:    "not a number",
			wantErr:  true,
		},
		// ceil() function tests
		{
			name:     "ceil() positive float",
			funcName: "ceil",
			input:    3.01,
			want:     4,
		},
		{
			name:     "ceil() negative float",
			funcName: "ceil",
			input:    -2.9,
			want:     -2,
		},
		{
			name:     "ceil() exactly integer",
			funcName: "ceil",
			input:    5.0,
			want:     5,
		},
		{
			name:     "ceil() small positive",
			funcName: "ceil",
			input:    0.1,
			want:     1,
		},
		{
			name:     "ceil() small negative",
			funcName: "ceil",
			input:    -0.1,
			want:     0,
		},
		{
			name:     "ceil() integer",
			funcName: "ceil",
			input:    -5,
			want:     -5,
		},
		{
			name:     "ceil() nil",
			funcName: "ceil",
			input:    nil,
			want:     nil,
		},
		{
			name:     "ceil() string number",
			funcName: "ceil",
			input:    "2.1",
			want:     3,
		},
		{
			name:     "ceil() invalid input",
			funcName: "ceil",
			input:    true,
			wantErr:  true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				t.Errorf("Function %s not found in registry", tt.funcName)
				return
			}

			got, err := fn.Call(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Function %s error = %v, wantErr %v", tt.funcName, err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
			}
		})
	}
}

func TestMathFunctionsInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		{
			name: "round() in arithmetic",
			expr: "round(price) + 10",
			data: TemplateData{"price": 39.99},
			want: 50, // round(39.99) = 40, 40 + 10 = 50
		},
		{
			name: "floor() with variable",
			expr: "floor(discount)",
			data: TemplateData{"discount": 15.85},
			want: 15,
		},
		{
			name: "ceil() with calculation",
			expr: "ceil(total / count)",
			data: TemplateData{"total": 100.0, "count": 3.0},
			want: 34, // ceil(100/3) = ceil(33.33) = 34
		},
		{
			name: "math functions combined",
			expr: "round(price) - floor(discount)",
			data: TemplateData{"price": 29.8, "discount": 3.9},
			want: 27, // round(29.8) - floor(3.9) = 30 - 3 = 27
		},
		{
			name: "nested math functions",
			expr: "ceil(floor(value))",
			data: TemplateData{"value": 5.7},
			want: 5, // ceil(floor(5.7)) = ceil(5) = 5
		},
		{
			name: "math function with string conversion",
			expr: "str(round(pi))",
			data: TemplateData{"pi": 3.14159},
			want: "3",
		},
		{
			name: "math function in conditional",
			expr: "round(score)",
			data: TemplateData{"score": 87.6},
			want: 88,
		},
		{
			name: "floor() with negative result",
			expr: "floor(balance)",
			data: TemplateData{"balance": -2.5},
			want: -3,
		},
		{
			name: "ceil() with negative result",
			expr: "ceil(debt)",
			data: TemplateData{"debt": -1.1},
			want: -1,
		},
		{
			name: "math functions with zero",
			expr: "round(zero) + floor(zero) + ceil(zero)",
			data: TemplateData{"zero": 0.0},
			want: 0,
		},
		{
			name:    "round() with non-numeric",
			expr:    "round(name)",
			data:    TemplateData{"name": "Alice"},
			wantErr: true,
		},
		{
			name:    "floor() with missing variable",
			expr:    "floor(missing)",
			data:    TemplateData{},
			want:    nil, // floor(nil) = nil
		},
		{
			name:    "ceil() with boolean",
			expr:    "ceil(enabled)",
			data:    TemplateData{"enabled": true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestAggregateFunctions(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		args     []interface{}
		want     interface{}
		wantErr  bool
	}{
		// sum() function tests
		{
			name:     "sum() with integer list",
			funcName: "sum",
			args:     []interface{}{[]interface{}{1, 2, 3, 4, 5}},
			want:     15,
		},
		{
			name:     "sum() with float list",
			funcName: "sum",
			args:     []interface{}{[]interface{}{1.5, 2.5, 3.0}},
			want:     7.0,
		},
		{
			name:     "sum() with mixed number types",
			funcName: "sum",
			args:     []interface{}{[]interface{}{100, 20, 3, 0.45}},
			want:     123.45,
		},
		{
			name:     "sum() with single element",
			funcName: "sum",
			args:     []interface{}{[]interface{}{17}},
			want:     17,
		},
		{
			name:     "sum() with empty list",
			funcName: "sum",
			args:     []interface{}{[]interface{}{}},
			want:     0,
		},
		{
			name:     "sum() with nil",
			funcName: "sum",
			args:     []interface{}{nil},
			want:     0,
		},
		{
			name:     "sum() with string numbers",
			funcName: "sum",
			args:     []interface{}{[]interface{}{"1", "2.5", "3"}},
			want:     6.5,
		},
		{
			name:     "sum() with mixed types",
			funcName: "sum",
			args:     []interface{}{[]interface{}{1, "2", 3.5}},
			want:     6.5,
		},
		{
			name:     "sum() with negative numbers",
			funcName: "sum",
			args:     []interface{}{[]interface{}{-1, 2, -3, 4}},
			want:     2,
		},
		{
			name:     "sum() with invalid number string",
			funcName: "sum",
			args:     []interface{}{[]interface{}{"abc", 1, 2}},
			wantErr:  true,
		},
		{
			name:     "sum() with non-slice argument",
			funcName: "sum",
			args:     []interface{}{"not a list"},
			wantErr:  true,
		},
		// contains() function tests
		{
			name:     "contains() with string in string list",
			funcName: "contains",
			args:     []interface{}{"apple", []interface{}{"apple", "banana", "cherry"}},
			want:     true,
		},
		{
			name:     "contains() with string not in list",
			funcName: "contains",
			args:     []interface{}{"grape", []interface{}{"apple", "banana", "cherry"}},
			want:     false,
		},
		{
			name:     "contains() with number in number list",
			funcName: "contains",
			args:     []interface{}{2, []interface{}{1, 2, 3}},
			want:     true,
		},
		{
			name:     "contains() with number not in list",
			funcName: "contains",
			args:     []interface{}{5, []interface{}{1, 2, 3}},
			want:     false,
		},
		{
			name:     "contains() with string representation match",
			funcName: "contains",
			args:     []interface{}{"2", []interface{}{1, 2, 3}},
			want:     true, // "2" should match 2 when converted to string
		},
		{
			name:     "contains() with number representation match",
			funcName: "contains",
			args:     []interface{}{2, []interface{}{"1", "2", "3"}},
			want:     true, // 2 should match "2" when converted to string
		},
		{
			name:     "contains() with empty list",
			funcName: "contains",
			args:     []interface{}{"anything", []interface{}{}},
			want:     false,
		},
		{
			name:     "contains() with nil item",
			funcName: "contains",
			args:     []interface{}{nil, []interface{}{"a", "b", nil}},
			want:     true,
		},
		{
			name:     "contains() with nil list",
			funcName: "contains",
			args:     []interface{}{"item", nil},
			want:     false,
		},
		{
			name:     "contains() with boolean values",
			funcName: "contains",
			args:     []interface{}{true, []interface{}{true, false}},
			want:     true,
		},
		{
			name:     "contains() with mixed types",
			funcName: "contains",
			args:     []interface{}{"true", []interface{}{true, false, "maybe"}},
			want:     true, // "true" should match true when converted to string
		},
		{
			name:     "contains() case sensitive",
			funcName: "contains",
			args:     []interface{}{"Apple", []interface{}{"apple", "banana"}},
			want:     false, // Should be case sensitive
		},
		{
			name:     "contains() with non-slice second argument",
			funcName: "contains",
			args:     []interface{}{"item", "not a list"},
			wantErr:  true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				t.Errorf("Function %s not found in registry", tt.funcName)
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Function %s error = %v, wantErr %v", tt.funcName, err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
			}
		})
	}
}

func TestAggregateFunctionsInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		{
			name: "sum() with variable list",
			expr: "sum(prices)",
			data: TemplateData{"prices": []interface{}{10, 20, 30}},
			want: 60,
		},
		{
			name: "sum() with map extraction",
			expr: "sum(map(\"price\", items))",
			data: TemplateData{
				"items": []interface{}{
					map[string]interface{}{"name": "Item1", "price": 10},
					map[string]interface{}{"name": "Item2", "price": 20},
					map[string]interface{}{"name": "Item3", "price": 30},
				},
			},
			want: 60,
		},
		{
			name: "sum() in arithmetic",
			expr: "sum(values) + 10",
			data: TemplateData{"values": []interface{}{1, 2, 3}},
			want: 16, // sum(1,2,3) + 10 = 6 + 10 = 16
		},
		{
			name: "sum() with empty list",
			expr: "sum(empty)",
			data: TemplateData{"empty": []interface{}{}},
			want: 0,
		},
		{
			name: "contains() with string search",
			expr: "contains(\"apple\", fruits)",
			data: TemplateData{"fruits": []interface{}{"apple", "banana", "cherry"}},
			want: true,
		},
		{
			name: "contains() with variable search",
			expr: "contains(search, items)",
			data: TemplateData{
				"search": "target",
				"items":  []interface{}{"item1", "target", "item3"},
			},
			want: true,
		},
		{
			name: "contains() in conditional",
			expr: "contains(name, allowed)",
			data: TemplateData{
				"name":    "admin",
				"allowed": []interface{}{"admin", "user", "guest"},
			},
			want: true,
		},
		{
			name: "contains() with number",
			expr: "contains(id, validIds)",
			data: TemplateData{
				"id":       42,
				"validIds": []interface{}{10, 42, 99},
			},
			want: true,
		},
		{
			name: "contains() returns false",
			expr: "contains(missing, list)",
			data: TemplateData{
				"missing": "not here",
				"list":    []interface{}{"a", "b", "c"},
			},
			want: false,
		},
		{
			name: "combined functions",
			expr: "contains(str(sum(values)), results)",
			data: TemplateData{
				"values":  []interface{}{1, 2, 3}, // sum = 6
				"results": []interface{}{"5", "6", "7"},
			},
			want: true, // str(6) = "6", and "6" is in results
		},
		{
			name: "sum() with decimal result",
			expr: "sum(prices)",
			data: TemplateData{"prices": []interface{}{1.1, 2.2, 3.3}},
			want: 6.6,
		},
		{
			name: "contains() with boolean",
			expr: "contains(flag, flags)",
			data: TemplateData{
				"flag":  true,
				"flags": []interface{}{true, false},
			},
			want: true,
		},
		{
			name:    "sum() with non-list",
			expr:    "sum(notAList)",
			data:    TemplateData{"notAList": "string"},
			wantErr: true,
		},
		{
			name:    "contains() with non-list",
			expr:    "contains(\"item\", notAList)",
			data:    TemplateData{"notAList": "string"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if !tt.wantErr {
				// Special handling for float comparison
				if gotFloat, ok := got.(float64); ok {
					if wantFloat, ok := tt.want.(float64); ok {
						if gotFloat != wantFloat {
							t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
						}
						return
					}
				}
				
				if got != tt.want {
					t.Errorf("Expression.Evaluate() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestPageBreakFunction(t *testing.T) {
	registry := GetDefaultFunctionRegistry()
	
	// Test basic function registration and call
	fn, exists := registry.GetFunction("pageBreak")
	if !exists {
		t.Errorf("pageBreak function not found in registry")
		return
	}
	
	// Test function call with no arguments
	result, err := fn.Call()
	if err != nil {
		t.Errorf("pageBreak() call error = %v", err)
		return
	}
	
	// Check that it returns an OOXML fragment
	fragment, ok := result.(*OOXMLFragment)
	if !ok {
		t.Errorf("pageBreak() should return *OOXMLFragment, got %T", result)
		return
	}
	
	// Check that the fragment contains a Break with type="page"
	breakElement, ok := fragment.Content.(*Break)
	if !ok {
		t.Errorf("pageBreak() fragment should contain *Break, got %T", fragment.Content)
		return
	}
	
	if breakElement.Type != "page" {
		t.Errorf("pageBreak() break type = %s, want 'page'", breakElement.Type)
	}
	
	// Test function call with wrong number of arguments
	_, err = fn.Call("unexpected argument")
	if err == nil {
		t.Errorf("pageBreak() should reject arguments")
	}
}

func TestPageBreakInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		wantErr bool
	}{
		{
			name: "pageBreak() in expression",
			expr: "pageBreak()",
			data: TemplateData{},
		},
		{
			name:    "pageBreak() with arguments should fail",
			expr:    "pageBreak(\"invalid\")",
			data:    TemplateData{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			
			if !tt.wantErr {
				// Check that we got an OOXML fragment
				fragment, ok := got.(*OOXMLFragment)
				if !ok {
					t.Errorf("Expression should return *OOXMLFragment, got %T", got)
					return
				}
				
				// Check that it's a page break
				if breakElement, ok := fragment.Content.(*Break); ok {
					if breakElement.Type != "page" {
						t.Errorf("Break type = %s, want 'page'", breakElement.Type)
					}
				} else {
					t.Errorf("Fragment should contain *Break, got %T", fragment.Content)
				}
			}
		})
	}
}

func TestRangeFunction(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		want     []interface{}
		wantErr  bool
	}{
		// Basic range tests matching original Stencil behavior
		{
			name: "range(5)",
			args: []interface{}{5},
			want: []interface{}{0, 1, 2, 3, 4},
		},
		{
			name: "range(1, 5)",
			args: []interface{}{1, 5},
			want: []interface{}{1, 2, 3, 4},
		},
		{
			name: "range(1, 6, 2)",
			args: []interface{}{1, 6, 2},
			want: []interface{}{1, 3, 5},
		},
		
		// Edge cases
		{
			name: "range(0) - empty range",
			args: []interface{}{0},
			want: []interface{}{},
		},
		{
			name: "range(1, 1) - empty range",
			args: []interface{}{1, 1},
			want: []interface{}{},
		},
		{
			name: "range(5, 1) - empty range (start > end)",
			args: []interface{}{5, 1},
			want: []interface{}{},
		},
		{
			name: "range(10, 0, -1) - descending",
			args: []interface{}{10, 0, -1},
			want: []interface{}{10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		},
		{
			name: "range(10, 0, -2) - descending with step",
			args: []interface{}{10, 0, -2},
			want: []interface{}{10, 8, 6, 4, 2},
		},
		{
			name: "range with negative start",
			args: []interface{}{-3, 3},
			want: []interface{}{-3, -2, -1, 0, 1, 2},
		},
		{
			name: "range with large step",
			args: []interface{}{0, 10, 5},
			want: []interface{}{0, 5},
		},
		
		// Type conversions
		{
			name: "range with float arguments",
			args: []interface{}{1.0, 5.0, 2.0},
			want: []interface{}{1, 3},
		},
		{
			name: "range with string numbers",
			args: []interface{}{"1", "5"},
			want: []interface{}{1, 2, 3, 4},
		},
		
		// Error cases
		{
			name:    "range() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "range() with too many arguments",
			args:    []interface{}{1, 2, 3, 4},
			wantErr: true,
		},
		{
			name:    "range() with zero step",
			args:    []interface{}{1, 5, 0},
			wantErr: true,
		},
		{
			name:    "range() with non-numeric argument",
			args:    []interface{}{"not a number"},
			wantErr: true,
		},
		{
			name:    "range() with nil argument",
			args:    []interface{}{nil},
			want:    []interface{}{},
		},
	}

	registry := GetDefaultFunctionRegistry()
	fn, exists := registry.GetFunction("range")
	if !exists {
		t.Fatalf("range function not found in registry")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("range(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotSlice, ok := got.([]interface{})
				if !ok {
					t.Errorf("range(%v) should return []interface{}, got %T", tt.args, got)
					return
				}

				if !compareSlices(gotSlice, tt.want) {
					t.Errorf("range(%v) = %v, want %v", tt.args, gotSlice, tt.want)
				}
			}
		})
	}
}

func TestRangeInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    []interface{}
		wantErr bool
	}{
		{
			name: "range() in expression",
			expr: "range(5)",
			data: TemplateData{},
			want: []interface{}{0, 1, 2, 3, 4},
		},
		{
			name: "range() with variables",
			expr: "range(start, end)",
			data: TemplateData{"start": 2, "end": 6},
			want: []interface{}{2, 3, 4, 5},
		},
		{
			name: "range() with calculation",
			expr: "range(count * 2)",
			data: TemplateData{"count": 3},
			want: []interface{}{0, 1, 2, 3, 4, 5},
		},
		{
			name: "range() with step from variable",
			expr: "range(0, 10, step)",
			data: TemplateData{"step": 3},
			want: []interface{}{0, 3, 6, 9},
		},
		{
			name:    "range() with invalid arguments",
			expr:    "range(\"invalid\")",
			data:    TemplateData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			if !tt.wantErr {
				gotSlice, ok := got.([]interface{})
				if !ok {
					t.Errorf("Expression should return []interface{}, got %T", got)
					return
				}

				if !compareSlices(gotSlice, tt.want) {
					t.Errorf("Expression.Evaluate() = %v, want %v", gotSlice, tt.want)
				}
			}
		})
	}
}

func TestTypeConversionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		input    interface{}
		checkFn  func(interface{}, error) bool
	}{
		{
			name:     "integer() with large uint64",
			funcName: "integer",
			input:    uint64(12345678901234567890),
			checkFn: func(got interface{}, err error) bool {
				// Should convert but may overflow
				return err == nil && got != nil
			},
		},
		{
			name:     "decimal() with very small float",
			funcName: "decimal",
			input:    0.0000000001,
			checkFn: func(got interface{}, err error) bool {
				return err == nil && got.(float64) == 0.0000000001
			},
		},
		{
			name:     "str() with custom type",
			funcName: "str",
			input:    struct{ Name string }{Name: "test"},
			checkFn: func(got interface{}, err error) bool {
				// Should produce some string representation
				return err == nil && got != nil && len(got.(string)) > 0
			},
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				t.Errorf("Function %s not found in registry", tt.funcName)
				return
			}

			got, err := fn.Call(tt.input)
			if !tt.checkFn(got, err) {
				t.Errorf("Function %s check failed for input %v: got %v, err %v", 
					tt.funcName, tt.input, got, err)
			}
		})
	}
}

func TestSwitchFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// Basic case matching tests
		{
			name: "basic string case match",
			args: []interface{}{"a", "a", 1, "b", 2, "c", 3},
			want: 1,
		},
		{
			name: "basic number case match",
			args: []interface{}{2, 1, "one", 2, "two", 3, "three"},
			want: "two",
		},
		{
			name: "second case match",
			args: []interface{}{"b", "a", 1, "b", 2, "c", 3},
			want: 2,
		},
		{
			name: "third case match",
			args: []interface{}{"c", "a", 1, "b", 2, "c", 3},
			want: 3,
		},
		{
			name: "no match without default",
			args: []interface{}{"x", "a", 1, "b", 2, "c", 3},
			want: nil,
		},
		{
			name: "no match with default (even number of args)",
			args: []interface{}{"x", "a", 1, "b", 2, "default"},
			want: "default",
		},
		{
			name: "match found with default available",
			args: []interface{}{"a", "a", 1, "b", 2, "default"},
			want: 1,
		},
		
		// Null handling tests
		{
			name: "null expression matches null case",
			args: []interface{}{nil, "a", 1, nil, 2, "c", 3},
			want: 2,
		},
		{
			name: "null expression no match",
			args: []interface{}{nil, "a", 1, "b", 2, "c", 3},
			want: nil,
		},
		{
			name: "null expression with default",
			args: []interface{}{nil, "a", 1, "b", 2, "default"},
			want: "default",
		},
		{
			name: "non-null expression matches null case (should not match)",
			args: []interface{}{"a", nil, 1, "b", 2, "c", 3},
			want: nil,
		},
		
		// Type matching tests
		{
			name: "integer matches integer",
			args: []interface{}{1, 1, "one", 2, "two"},
			want: "one",
		},
		{
			name: "float matches float",
			args: []interface{}{1.5, 1.5, "one-half", 2.5, "two-half"},
			want: "one-half",
		},
		{
			name: "boolean matches boolean",
			args: []interface{}{true, true, "yes", false, "no"},
			want: "yes",
		},
		{
			name: "string matches string exactly",
			args: []interface{}{"hello", "hello", "greeting", "goodbye", "farewell"},
			want: "greeting",
		},
		
		// Edge cases
		{
			name: "empty string expression",
			args: []interface{}{"", "", "empty", "full", "not empty"},
			want: "empty",
		},
		{
			name: "zero value expression",
			args: []interface{}{0, 0, "zero", 1, "one"},
			want: "zero",
		},
		{
			name: "false expression",
			args: []interface{}{false, false, "false case", true, "true case"},
			want: "false case",
		},
		
		// Complex values
		{
			name: "slice case matching",
			args: []interface{}{[]interface{}{1, 2}, []interface{}{1, 2}, "match", "other", "no match"},
			want: nil, // Slices don't match with equals, so no match
		},
		{
			name: "map case matching", 
			args: []interface{}{map[string]interface{}{"a": 1}, map[string]interface{}{"a": 1}, "match", "other", "no match"},
			want: nil, // Maps don't match with equals, so no match
		},
		
		// First match wins
		{
			name: "first match wins (duplicate cases)",
			args: []interface{}{"a", "a", "first", "a", "second", "b", "third"},
			want: "first",
		},
		
		// Minimum arguments validation
		{
			name:    "insufficient arguments (0 args)",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "insufficient arguments (1 arg)",
			args:    []interface{}{"expr"},
			wantErr: true,
		},
		{
			name:    "insufficient arguments (2 args)",
			args:    []interface{}{"expr", "case1"},
			wantErr: true,
		},
		
		// Minimum valid case
		{
			name: "minimum valid arguments (3 args)",
			args: []interface{}{"a", "a", "result"},
			want: "result",
		},
		
		// Mixed types in cases
		{
			name: "mixed types with string match",
			args: []interface{}{"test", 1, "one", "test", "string", false, "boolean"},
			want: "string",
		},
		{
			name: "mixed types with number match",
			args: []interface{}{42, "string", "text", 42, "number", true, "bool"},
			want: "number",
		},
		
		// Case sensitivity
		{
			name: "case sensitive string matching",
			args: []interface{}{"Hello", "hello", "lowercase", "Hello", "correct case"},
			want: "correct case",
		},
		
		// Large number of cases
		{
			name: "many cases",
			args: []interface{}{"f", "a", 1, "b", 2, "c", 3, "d", 4, "e", 5, "f", 6, "g", 7},
			want: 6,
		},
		
		// Return various types
		{
			name: "return integer",
			args: []interface{}{"match", "match", 42},
			want: 42,
		},
		{
			name: "return float",
			args: []interface{}{"match", "match", 3.14},
			want: 3.14,
		},
		{
			name: "return boolean",
			args: []interface{}{"match", "match", true},
			want: true,
		},
		{
			name: "return slice",
			args: []interface{}{"match", "match", []interface{}{1, 2, 3}},
			want: []interface{}{1, 2, 3},
		},
		{
			name: "return map",
			args: []interface{}{"match", "match", map[string]interface{}{"key": "value"}},
			want: map[string]interface{}{"key": "value"},
		},
		{
			name: "return nil",
			args: []interface{}{"match", "match", nil},
			want: nil,
		},
	}

	registry := GetDefaultFunctionRegistry()
	fn, exists := registry.GetFunction("switch")
	if !exists {
		t.Fatalf("switch function not found in registry")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("switch(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Special handling for slice comparison
				if wantSlice, ok := tt.want.([]interface{}); ok {
					if gotSlice, ok := got.([]interface{}); ok {
						if !compareSlices(gotSlice, wantSlice) {
							t.Errorf("switch(%v) = %v, want %v", tt.args, got, tt.want)
						}
						return
					}
				}
				
				// Special handling for map comparison
				if wantMap, ok := tt.want.(map[string]interface{}); ok {
					if gotMap, ok := got.(map[string]interface{}); ok {
						if len(wantMap) != len(gotMap) {
							t.Errorf("switch(%v) = %v, want %v", tt.args, got, tt.want)
							return
						}
						for k, v := range wantMap {
							if gotV, exists := gotMap[k]; !exists || gotV != v {
								t.Errorf("switch(%v) = %v, want %v", tt.args, got, tt.want)
								return
							}
						}
						return
					}
				}
				
				// Regular comparison
				if got != tt.want {
					t.Errorf("switch(%v) = %v, want %v", tt.args, got, tt.want)
				}
			}
		})
	}
}

// Mock FunctionProvider for testing
type MockFunctionProvider struct {
	functions map[string]Function
}

func NewMockFunctionProvider() *MockFunctionProvider {
	return &MockFunctionProvider{
		functions: make(map[string]Function),
	}
}

func (p *MockFunctionProvider) AddFunction(name string, fn Function) {
	p.functions[name] = fn
}

func (p *MockFunctionProvider) ProvideFunctions() map[string]Function {
	return p.functions
}

func TestFunctionProvider(t *testing.T) {
	// Create a mock function provider
	provider := NewMockFunctionProvider()
	
	// Add custom functions
	doubleFunc := NewSimpleFunction("double", 1, 1, func(args ...interface{}) (interface{}, error) {
		val, ok := args[0].(int)
		if !ok {
			return nil, fmt.Errorf("double() requires integer argument")
		}
		return val * 2, nil
	})
	provider.AddFunction("double", doubleFunc)
	
	greetFunc := NewSimpleFunction("greet", 1, 1, func(args ...interface{}) (interface{}, error) {
		name, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("greet() requires string argument")
		}
		return "Hello, " + name + "!", nil
	})
	provider.AddFunction("greet", greetFunc)
	
	// Test ProvideFunctions
	functions := provider.ProvideFunctions()
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}
	
	if _, exists := functions["double"]; !exists {
		t.Errorf("double function not found in provider")
	}
	
	if _, exists := functions["greet"]; !exists {
		t.Errorf("greet function not found in provider")
	}
	
	// Test function calls
	result, err := functions["double"].Call(5)
	if err != nil {
		t.Errorf("double function call error: %v", err)
	}
	if result != 10 {
		t.Errorf("double(5) = %v, want 10", result)
	}
	
	result, err = functions["greet"].Call("World")
	if err != nil {
		t.Errorf("greet function call error: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("greet(\"World\") = %v, want \"Hello, World!\"", result)
	}
}

func TestRegisterFunctionsFromProvider(t *testing.T) {
	// Create a mock function provider
	provider := NewMockFunctionProvider()
	
	// Add custom functions
	addFunc := NewSimpleFunction("add", 2, 2, func(args ...interface{}) (interface{}, error) {
		a, ok1 := args[0].(int)
		b, ok2 := args[1].(int)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("add() requires integer arguments")
		}
		return a + b, nil
	})
	provider.AddFunction("add", addFunc)
	
	maxFunc := NewSimpleFunction("max", 2, 2, func(args ...interface{}) (interface{}, error) {
		a, ok1 := args[0].(int)
		b, ok2 := args[1].(int)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("max() requires integer arguments")
		}
		if a > b {
			return a, nil
		}
		return b, nil
	})
	provider.AddFunction("max", maxFunc)
	
	// Register functions from provider to global registry
	err := RegisterFunctionsFromProvider(provider)
	if err != nil {
		t.Errorf("RegisterFunctionsFromProvider() error = %v", err)
	}
	
	// Test that functions are available in global registry
	registry := GetDefaultFunctionRegistry()
	
	addFn, exists := registry.GetFunction("add")
	if !exists {
		t.Errorf("add function not found in global registry")
	}
	
	result, err := addFn.Call(3, 7)
	if err != nil {
		t.Errorf("add function call error: %v", err)
	}
	if result != 10 {
		t.Errorf("add(3, 7) = %v, want 10", result)
	}
	
	maxFn, exists := registry.GetFunction("max")
	if !exists {
		t.Errorf("max function not found in global registry")
	}
	
	result, err = maxFn.Call(5, 3)
	if err != nil {
		t.Errorf("max function call error: %v", err)
	}
	if result != 5 {
		t.Errorf("max(5, 3) = %v, want 5", result)
	}
}

func TestCreateRegistryWithProvider(t *testing.T) {
	// Create a mock function provider
	provider := NewMockFunctionProvider()
	
	// Add custom functions that might override built-ins
	customUppercaseFunc := NewSimpleFunction("uppercase", 1, 1, func(args ...interface{}) (interface{}, error) {
		str, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("custom uppercase() requires string argument")
		}
		return "CUSTOM_" + strings.ToUpper(str), nil
	})
	provider.AddFunction("uppercase", customUppercaseFunc)
	
	newFunc := NewSimpleFunction("newFunction", 1, 1, func(args ...interface{}) (interface{}, error) {
		return "custom result: " + fmt.Sprintf("%v", args[0]), nil
	})
	provider.AddFunction("newFunction", newFunc)
	
	// Create registry with provider
	registry, err := CreateRegistryWithProvider(provider)
	if err != nil {
		t.Errorf("CreateRegistryWithProvider() error = %v", err)
	}
	
	// Test that built-in functions are still available
	emptyFn, exists := registry.GetFunction("empty")
	if !exists {
		t.Errorf("empty function not found in registry with provider")
	}
	
	result, err := emptyFn.Call("")
	if err != nil {
		t.Errorf("empty function call error: %v", err)
	}
	if result != true {
		t.Errorf("empty(\"\") = %v, want true", result)
	}
	
	// Test that custom function is available
	newFn, exists := registry.GetFunction("newFunction")
	if !exists {
		t.Errorf("newFunction not found in registry with provider")
	}
	
	result, err = newFn.Call("test")
	if err != nil {
		t.Errorf("newFunction call error: %v", err)
	}
	if result != "custom result: test" {
		t.Errorf("newFunction(\"test\") = %v, want \"custom result: test\"", result)
	}
	
	// Test that function override works (custom uppercase should override built-in)
	uppercaseFn, exists := registry.GetFunction("uppercase")
	if !exists {
		t.Errorf("uppercase function not found in registry with provider")
	}
	
	result, err = uppercaseFn.Call("hello")
	if err != nil {
		t.Errorf("uppercase function call error: %v", err)
	}
	if result != "CUSTOM_HELLO" {
		t.Errorf("uppercase(\"hello\") = %v, want \"CUSTOM_HELLO\"", result)
	}
}

func TestCustomFunctionInExpressions(t *testing.T) {
	// Register a custom function in global registry
	concatFunc := NewSimpleFunction("concat", 2, -1, func(args ...interface{}) (interface{}, error) {
		var result string
		for _, arg := range args {
			if arg != nil {
				result += FormatValue(arg)
			}
		}
		return result, nil
	})
	
	err := RegisterGlobalFunction("concat", concatFunc)
	if err != nil {
		t.Errorf("RegisterGlobalFunction() error = %v", err)
	}
	
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		{
			name: "custom concat function",
			expr: "concat(\"Hello\", \" \", \"World\")",
			data: TemplateData{},
			want: "Hello World",
		},
		{
			name: "custom concat with variables",
			expr: "concat(greeting, \" \", name, \"!\")",
			data: TemplateData{"greeting": "Hi", "name": "Alice"},
			want: "Hi Alice!",
		},
		{
			name: "custom concat with numbers",
			expr: "concat(\"Count: \", count)",
			data: TemplateData{"count": 42},
			want: "Count: 42",
		},
		{
			name: "custom concat in conditional",
			expr: "concat(\"Status: \", status)",
			data: TemplateData{"status": "active"},
			want: "Status: active",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestFunctionOverride(t *testing.T) {
	// Create a new registry to avoid affecting global state
	registry := NewFunctionRegistry()
	
	// Register basic functions
	registerBasicFunctions(registry)
	
	// Test original function behavior
	originalEmpty, exists := registry.GetFunction("empty")
	if !exists {
		t.Errorf("original empty function not found")
	}
	
	result, err := originalEmpty.Call("")
	if err != nil {
		t.Errorf("original empty function call error: %v", err)
	}
	if result != true {
		t.Errorf("original empty(\"\") = %v, want true", result)
	}
	
	// Override the empty function
	customEmptyFunc := NewSimpleFunction("empty", 1, 1, func(args ...interface{}) (interface{}, error) {
		// Custom empty: only nil is considered empty
		return args[0] == nil, nil
	})
	
	err = registry.RegisterFunction(customEmptyFunc)
	if err != nil {
		t.Errorf("RegisterFunction() override error = %v", err)
	}
	
	// Test overridden function behavior
	overriddenEmpty, exists := registry.GetFunction("empty")
	if !exists {
		t.Errorf("overridden empty function not found")
	}
	
	// Empty string should now return false (not empty in custom implementation)
	result, err = overriddenEmpty.Call("")
	if err != nil {
		t.Errorf("overridden empty function call error: %v", err)
	}
	if result != false {
		t.Errorf("overridden empty(\"\") = %v, want false", result)
	}
	
	// nil should still return true
	result, err = overriddenEmpty.Call(nil)
	if err != nil {
		t.Errorf("overridden empty function call error: %v", err)
	}
	if result != true {
		t.Errorf("overridden empty(nil) = %v, want true", result)
	}
}

func TestFunctionProviderErrorHandling(t *testing.T) {
	// Test registering function with empty name
	provider := NewMockFunctionProvider()
	emptyNameFunc := NewSimpleFunction("", 0, 0, func(args ...interface{}) (interface{}, error) {
		return nil, nil
	})
	provider.AddFunction("emptyName", emptyNameFunc)
	
	err := RegisterFunctionsFromProvider(provider)
	if err == nil {
		t.Errorf("RegisterFunctionsFromProvider() should fail with empty function name")
	}
	
	// Test CreateRegistryWithProvider with invalid function
	registry, err := CreateRegistryWithProvider(provider)
	if err == nil {
		t.Errorf("CreateRegistryWithProvider() should fail with empty function name")
	}
	if registry != nil {
		t.Errorf("CreateRegistryWithProvider() should return nil registry on error")
	}
}

func TestCustomFunctionComplexScenarios(t *testing.T) {
	// Create a provider with multiple complex functions
	provider := NewMockFunctionProvider()
	
	// Math function that works with both numbers and strings
	powerFunc := NewSimpleFunction("power", 2, 2, func(args ...interface{}) (interface{}, error) {
		base, err := toNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("power() first argument must be a number")
		}
		
		exp, err := toNumber(args[1])
		if err != nil {
			return nil, fmt.Errorf("power() second argument must be a number")
		}
		
		result := math.Pow(base, exp)
		
		// Return int if result is a whole number, float otherwise
		if result == float64(int(result)) {
			return int(result), nil
		}
		return result, nil
	})
	provider.AddFunction("power", powerFunc)
	
	// Conditional function
	ifElseFunc := NewSimpleFunction("ifElse", 3, 3, func(args ...interface{}) (interface{}, error) {
		condition := args[0]
		trueValue := args[1]
		falseValue := args[2]
		
		// Evaluate condition using same logic as isEmpty (but inverted)
		isTrue := !isEmpty(condition)
		
		if isTrue {
			return trueValue, nil
		}
		return falseValue, nil
	})
	provider.AddFunction("ifElse", ifElseFunc)
	
	// Create registry with provider
	registry, err := CreateRegistryWithProvider(provider)
	if err != nil {
		t.Errorf("CreateRegistryWithProvider() error = %v", err)
	}
	
	// Test complex function interactions
	tests := []struct {
		name     string
		funcName string
		args     []interface{}
		want     interface{}
		wantErr  bool
	}{
		{
			name:     "power function with integers",
			funcName: "power",
			args:     []interface{}{2, 3},
			want:     8,
		},
		{
			name:     "power function with floats",
			funcName: "power",
			args:     []interface{}{2.0, 0.5},
			want:     math.Sqrt(2.0),
		},
		{
			name:     "power function with string numbers",
			funcName: "power",
			args:     []interface{}{"3", "2"},
			want:     9,
		},
		{
			name:     "ifElse with true condition",
			funcName: "ifElse",
			args:     []interface{}{true, "yes", "no"},
			want:     "yes",
		},
		{
			name:     "ifElse with false condition",
			funcName: "ifElse",
			args:     []interface{}{false, "yes", "no"},
			want:     "no",
		},
		{
			name:     "ifElse with empty string condition",
			funcName: "ifElse",
			args:     []interface{}{"", "yes", "no"},
			want:     "no",
		},
		{
			name:     "ifElse with non-empty string condition",
			funcName: "ifElse",
			args:     []interface{}{"hello", "yes", "no"},
			want:     "yes",
		},
		{
			name:     "power function with invalid argument",
			funcName: "power",
			args:     []interface{}{"not a number", 2},
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				t.Errorf("Function %s not found in registry", tt.funcName)
				return
			}
			
			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Function %s error = %v, wantErr %v", tt.funcName, err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				// Special handling for float comparison
				if gotFloat, ok := got.(float64); ok {
					if wantFloat, ok := tt.want.(float64); ok {
						if gotFloat != wantFloat {
							t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
						}
						return
					}
				}
				
				if got != tt.want {
					t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
				}
			}
		})
	}
}

func TestSwitchInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		{
			name: "switch with string variable",
			expr: "switch(status, \"active\", \"✓\", \"inactive\", \"✗\", \"Unknown\")",
			data: TemplateData{"status": "active"},
			want: "✓",
		},
		{
			name: "switch with number variable",
			expr: "switch(level, 1, \"Basic\", 2, \"Premium\", 3, \"Enterprise\", \"Unknown\")",
			data: TemplateData{"level": 2},
			want: "Premium",
		},
		{
			name: "switch with default case",
			expr: "switch(status, \"active\", \"✓\", \"inactive\", \"✗\", \"Unknown\")",
			data: TemplateData{"status": "pending"},
			want: "Unknown",
		},
		{
			name: "switch without default case",
			expr: "switch(color, \"red\", \"#FF0000\", \"green\", \"#00FF00\", \"blue\", \"#0000FF\")",
			data: TemplateData{"color": "yellow"},
			want: nil,
		},
		{
			name: "switch with expression as input",
			expr: "switch(score / 10, 10, \"Perfect\", 9, \"Excellent\", 8, \"Good\", \"Needs improvement\")",
			data: TemplateData{"score": 90},
			want: "Excellent",
		},
		{
			name: "switch with boolean expression",
			expr: "switch(count > 0, true, \"Has items\", false, \"Empty\")",
			data: TemplateData{"count": 5},
			want: "Has items",
		},
		{
			name: "switch with null value",
			expr: "switch(name, null, \"No name\", \"admin\", \"Administrator\", \"Regular user\")",
			data: TemplateData{},
			want: "No name",
		},
		{
			name: "nested switch calls",
			expr: "switch(type, \"user\", switch(role, \"admin\", \"Admin User\", \"guest\", \"Guest User\", \"Regular User\"), \"system\", \"System Account\", \"Unknown\")",
			data: TemplateData{"type": "user", "role": "admin"},
			want: "Admin User",
		},
		{
			name: "switch with function calls in cases",
			expr: "switch(status, \"active\", uppercase(\"active\"), \"inactive\", uppercase(\"inactive\"), \"unknown\")",
			data: TemplateData{"status": "active"},
			want: "ACTIVE",
		},
		{
			name: "switch in arithmetic",
			expr: "switch(grade, \"A\", 4, \"B\", 3, \"C\", 2, \"D\", 1, 0) * 10",
			data: TemplateData{"grade": "B"},
			want: 30,
		},
		{
			name: "switch with string concatenation",
			expr: "\"Result: \" + switch(success, true, \"Pass\", false, \"Fail\", \"Unknown\")",
			data: TemplateData{"success": true},
			want: "Result: Pass",
		},
		{
			name: "switch with coalesce",
			expr: "switch(coalesce(priority, \"normal\"), \"high\", \"🔴\", \"normal\", \"⚪\", \"low\", \"🟢\", \"❓\")",
			data: TemplateData{},
			want: "⚪",
		},
		{
			name: "switch result in other functions",
			expr: "uppercase(switch(lang, \"en\", \"english\", \"es\", \"spanish\", \"fr\", \"french\", \"unknown\"))",
			data: TemplateData{"lang": "es"},
			want: "SPANISH",
		},
		{
			name: "switch with variable cases and values",
			expr: "switch(input, case1, value1, case2, value2, defaultValue)",
			data: TemplateData{"input": "test", "case1": "test", "value1": "matched", "case2": "other", "value2": "not matched", "defaultValue": "default"},
			want: "matched",
		},
		{
			name: "switch with integer zero",
			expr: "switch(count, 0, \"None\", 1, \"One\", \"Many\")",
			data: TemplateData{"count": 0},
			want: "None",
		},
		{
			name: "switch with boolean false",
			expr: "switch(enabled, false, \"Disabled\", true, \"Enabled\")",
			data: TemplateData{"enabled": false},
			want: "Disabled",
		},
		{
			name: "switch with empty string",
			expr: "switch(name, \"\", \"No name provided\", \"Anonymous\", \"Unknown user\", name)",
			data: TemplateData{"name": ""},
			want: "No name provided",
		},
		{
			name:    "switch with insufficient arguments",
			expr:    "switch(status, \"active\")",
			data:    TemplateData{"status": "active"},
			wantErr: true,
		},
		{
			name:    "switch with non-existent variable",
			expr:    "switch(missing_var, \"test\", \"result\")",
			data:    TemplateData{},
			want:    nil, // missing_var is nil, doesn't match "test"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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