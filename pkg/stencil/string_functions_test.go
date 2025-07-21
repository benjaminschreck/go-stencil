package stencil

import (
	"testing"
)

func TestStringCaseFunctions(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		input    interface{}
		want     interface{}
		wantErr  bool
	}{
		// lowercase() function tests
		{
			name:     "lowercase() with uppercase string",
			funcName: "lowercase",
			input:    "HELLO WORLD",
			want:     "hello world",
		},
		{
			name:     "lowercase() with mixed case",
			funcName: "lowercase",
			input:    "HeLLo WoRLd",
			want:     "hello world",
		},
		{
			name:     "lowercase() with already lowercase",
			funcName: "lowercase",
			input:    "hello world",
			want:     "hello world",
		},
		{
			name:     "lowercase() with numbers and symbols",
			funcName: "lowercase",
			input:    "Test123!@#",
			want:     "test123!@#",
		},
		{
			name:     "lowercase() with empty string",
			funcName: "lowercase",
			input:    "",
			want:     "",
		},
		{
			name:     "lowercase() with nil",
			funcName: "lowercase",
			input:    nil,
			want:     nil,
		},
		{
			name:     "lowercase() with integer",
			funcName: "lowercase",
			input:    123,
			want:     "123",
		},
		{
			name:     "lowercase() with boolean",
			funcName: "lowercase",
			input:    true,
			want:     "true",
		},
		// uppercase() function tests
		{
			name:     "uppercase() with lowercase string",
			funcName: "uppercase",
			input:    "hello world",
			want:     "HELLO WORLD",
		},
		{
			name:     "uppercase() with mixed case",
			funcName: "uppercase",
			input:    "HeLLo WoRLd",
			want:     "HELLO WORLD",
		},
		{
			name:     "uppercase() with already uppercase",
			funcName: "uppercase",
			input:    "HELLO WORLD",
			want:     "HELLO WORLD",
		},
		{
			name:     "uppercase() with numbers and symbols",
			funcName: "uppercase",
			input:    "test123!@#",
			want:     "TEST123!@#",
		},
		{
			name:     "uppercase() with empty string",
			funcName: "uppercase",
			input:    "",
			want:     "",
		},
		{
			name:     "uppercase() with nil",
			funcName: "uppercase",
			input:    nil,
			want:     nil,
		},
		{
			name:     "uppercase() with integer",
			funcName: "uppercase",
			input:    456,
			want:     "456",
		},
		{
			name:     "uppercase() with boolean",
			funcName: "uppercase",
			input:    false,
			want:     "FALSE",
		},
		// titlecase() function tests
		{
			name:     "titlecase() with lowercase string",
			funcName: "titlecase",
			input:    "hello world",
			want:     "Hello World",
		},
		{
			name:     "titlecase() with uppercase string",
			funcName: "titlecase",
			input:    "HELLO WORLD",
			want:     "Hello World",
		},
		{
			name:     "titlecase() with mixed case",
			funcName: "titlecase",
			input:    "hELLo wORLd",
			want:     "Hello World",
		},
		{
			name:     "titlecase() with single word",
			funcName: "titlecase",
			input:    "hello",
			want:     "Hello",
		},
		{
			name:     "titlecase() with multiple spaces",
			funcName: "titlecase",
			input:    "hello   world   test",
			want:     "Hello   World   Test",
		},
		{
			name:     "titlecase() with numbers",
			funcName: "titlecase",
			input:    "test 123 case",
			want:     "Test 123 Case",
		},
		{
			name:     "titlecase() with empty string",
			funcName: "titlecase",
			input:    "",
			want:     "",
		},
		{
			name:     "titlecase() with nil",
			funcName: "titlecase",
			input:    nil,
			want:     nil,
		},
		{
			name:     "titlecase() with single letter words",
			funcName: "titlecase",
			input:    "a b c",
			want:     "A B C",
		},
		{
			name:     "titlecase() with apostrophes",
			funcName: "titlecase",
			input:    "it's john's book",
			want:     "It's John's Book",
		},
		{
			name:     "titlecase() with hyphens",
			funcName: "titlecase",
			input:    "well-known fact",
			want:     "Well-known Fact",
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				// Function not yet implemented, skip this test for now
				t.Skipf("Function %s not yet implemented", tt.funcName)
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

func TestStringCaseFunctionsInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		// lowercase() in expressions
		{
			name: "lowercase() with variable",
			expr: "lowercase(name)",
			data: TemplateData{"name": "JOHN DOE"},
			want: "john doe",
		},
		{
			name: "lowercase() with literal",
			expr: "lowercase(\"HELLO\")",
			data: TemplateData{},
			want: "hello",
		},
		{
			name: "lowercase() with concatenation",
			expr: "lowercase(first + \" \" + last)",
			data: TemplateData{"first": "JOHN", "last": "DOE"},
			want: "john doe",
		},
		{
			name: "lowercase() with nested function",
			expr: "lowercase(coalesce(name, \"DEFAULT\"))",
			data: TemplateData{"name": nil},
			want: "default",
		},
		// uppercase() in expressions
		{
			name: "uppercase() with variable",
			expr: "uppercase(name)",
			data: TemplateData{"name": "john doe"},
			want: "JOHN DOE",
		},
		{
			name: "uppercase() with literal",
			expr: "uppercase(\"hello\")",
			data: TemplateData{},
			want: "HELLO",
		},
		{
			name: "uppercase() with concatenation",
			expr: "uppercase(first + \" \" + last)",
			data: TemplateData{"first": "john", "last": "doe"},
			want: "JOHN DOE",
		},
		// titlecase() in expressions
		{
			name: "titlecase() with variable",
			expr: "titlecase(name)",
			data: TemplateData{"name": "john doe"},
			want: "John Doe",
		},
		{
			name: "titlecase() with literal",
			expr: "titlecase(\"hello world\")",
			data: TemplateData{},
			want: "Hello World",
		},
		{
			name: "titlecase() with nested field",
			expr: "titlecase(user.fullName)",
			data: TemplateData{
				"user": map[string]interface{}{
					"fullName": "jane smith",
				},
			},
			want: "Jane Smith",
		},
		// Combined case functions
		{
			name: "uppercase() of lowercase()",
			expr: "uppercase(lowercase(\"HeLLo\"))",
			data: TemplateData{},
			want: "HELLO",
		},
		{
			name: "titlecase() in conditional",
			expr: "titlecase(name)",
			data: TemplateData{"name": "mary jane watson"},
			want: "Mary Jane Watson",
		},
		{
			name: "case function with number conversion",
			expr: "uppercase(str(count))",
			data: TemplateData{"count": 42},
			want: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests for functions not yet implemented
			if tt.expr == "lowercase(name)" || tt.expr == "uppercase(name)" || tt.expr == "titlecase(name)" {
				registry := GetDefaultFunctionRegistry()
				funcName := ""
				if tt.expr == "lowercase(name)" {
					funcName = "lowercase"
				} else if tt.expr == "uppercase(name)" {
					funcName = "uppercase"
				} else if tt.expr == "titlecase(name)" {
					funcName = "titlecase"
				}
				
				if funcName != "" {
					if _, exists := registry.GetFunction(funcName); !exists {
						t.Skipf("Function %s not yet implemented", funcName)
						return
					}
				}
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

func TestStringCaseFunctionsEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		args     []interface{}
		want     interface{}
		wantErr  bool
	}{
		// Wrong number of arguments
		{
			name:     "lowercase() with no arguments",
			funcName: "lowercase",
			args:     []interface{}{},
			wantErr:  true,
		},
		{
			name:     "lowercase() with too many arguments",
			funcName: "lowercase",
			args:     []interface{}{"hello", "world"},
			wantErr:  true,
		},
		{
			name:     "uppercase() with no arguments",
			funcName: "uppercase",
			args:     []interface{}{},
			wantErr:  true,
		},
		{
			name:     "uppercase() with too many arguments",
			funcName: "uppercase",
			args:     []interface{}{"hello", "world"},
			wantErr:  true,
		},
		{
			name:     "titlecase() with no arguments",
			funcName: "titlecase",
			args:     []interface{}{},
			wantErr:  true,
		},
		{
			name:     "titlecase() with too many arguments",
			funcName: "titlecase",
			args:     []interface{}{"hello", "world"},
			wantErr:  true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				// Function not yet implemented, skip this test for now
				t.Skipf("Function %s not yet implemented", tt.funcName)
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

func TestStringCaseFunctionsWithSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		input    string
		want     string
	}{
		// Unicode characters
		{
			name:     "lowercase() with accented characters",
			funcName: "lowercase",
			input:    "CAFÉ RÉSUMÉ",
			want:     "café résumé",
		},
		{
			name:     "uppercase() with accented characters",
			funcName: "uppercase",
			input:    "café résumé",
			want:     "CAFÉ RÉSUMÉ",
		},
		{
			name:     "titlecase() with accented characters",
			funcName: "titlecase",
			input:    "café résumé",
			want:     "Café Résumé",
		},
		// Special symbols
		{
			name:     "lowercase() with emoji",
			funcName: "lowercase",
			input:    "HELLO 😊 WORLD",
			want:     "hello 😊 world",
		},
		{
			name:     "uppercase() with emoji",
			funcName: "uppercase",
			input:    "hello 😊 world",
			want:     "HELLO 😊 WORLD",
		},
		{
			name:     "titlecase() with emoji",
			funcName: "titlecase",
			input:    "hello 😊 world",
			want:     "Hello 😊 World",
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction(tt.funcName)
			if !exists {
				// Function not yet implemented, skip this test for now
				t.Skipf("Function %s not yet implemented", tt.funcName)
				return
			}

			got, err := fn.Call(tt.input)
			if err != nil {
				t.Errorf("Function %s error = %v", tt.funcName, err)
				return
			}

			if got != tt.want {
				t.Errorf("Function %s = %v, want %v", tt.funcName, got, tt.want)
			}
		})
	}
}

func TestJoinFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// join() with no separator
		{
			name: "join() with no separator concatenates",
			args: []interface{}{[]interface{}{"hello", "world"}},
			want: "helloworld",
		},
		{
			name: "join() with empty collection",
			args: []interface{}{[]interface{}{}},
			want: "",
		},
		{
			name: "join() with nil collection",
			args: []interface{}{nil},
			want: "",
		},
		{
			name: "join() with one element",
			args: []interface{}{[]interface{}{"single"}},
			want: "single",
		},
		{
			name: "join() with numbers",
			args: []interface{}{[]interface{}{1, 2, 3}},
			want: "123",
		},
		{
			name: "join() with mixed types",
			args: []interface{}{[]interface{}{"test", 123, true}},
			want: "test123true",
		},
		{
			name: "join() filters nil values",
			args: []interface{}{[]interface{}{"a", nil, "b"}},
			want: "ab",
		},
		// join() with separator
		{
			name: "join() with comma separator",
			args: []interface{}{[]interface{}{"apple", "banana", "cherry"}, ","},
			want: "apple,banana,cherry",
		},
		{
			name: "join() with space separator",
			args: []interface{}{[]interface{}{"hello", "world"}, " "},
			want: "hello world",
		},
		{
			name: "join() with multi-char separator",
			args: []interface{}{[]interface{}{"one", "two", "three"}, " - "},
			want: "one - two - three",
		},
		{
			name: "join() with nil separator",
			args: []interface{}{[]interface{}{"a", "b"}, nil},
			want: "ab",
		},
		{
			name: "join() with empty string separator",
			args: []interface{}{[]interface{}{"a", "b", "c"}, ""},
			want: "abc",
		},
		// error cases
		{
			name:    "join() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "join() with non-collection first argument",
			args:    []interface{}{"not a collection"},
			want:    "not a collection", // strings are treated as character arrays but joined back
		},
		{
			name:    "join() with non-string separator",
			args:    []interface{}{[]interface{}{"a", "b"}, 123},
			wantErr: true,
		},
		{
			name:    "join() with too many arguments",
			args:    []interface{}{[]interface{}{"a"}, ",", "extra"},
			wantErr: true,
		},
		// string slice support
		{
			name: "join() with string slice",
			args: []interface{}{[]string{"one", "two", "three"}, "-"},
			want: "one-two-three",
		},
		// int slice support
		{
			name: "join() with int slice",
			args: []interface{}{[]int{1, 2, 3}, ","},
			want: "1,2,3",
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("join")
			if !exists {
				t.Skipf("Function join not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("join() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("join() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinAndFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// Basic joinAnd cases
		{
			name: "joinAnd() with empty collection",
			args: []interface{}{[]interface{}{}, ", ", " and "},
			want: "",
		},
		{
			name: "joinAnd() with one element",
			args: []interface{}{[]interface{}{"single"}, ", ", " and "},
			want: "single",
		},
		{
			name: "joinAnd() with two elements",
			args: []interface{}{[]interface{}{"first", "second"}, ", ", " and "},
			want: "first and second",
		},
		{
			name: "joinAnd() with three elements",
			args: []interface{}{[]interface{}{"one", "two", "three"}, ", ", " and "},
			want: "one, two and three",
		},
		{
			name: "joinAnd() with many elements",
			args: []interface{}{[]interface{}{"a", "b", "c", "d", "e"}, ", ", " and "},
			want: "a, b, c, d and e",
		},
		{
			name: "joinAnd() with numbers",
			args: []interface{}{[]interface{}{1, 2, 3, 4}, ", ", " & "},
			want: "1, 2, 3 & 4",
		},
		{
			name: "joinAnd() with nil values filtered",
			args: []interface{}{[]interface{}{"a", nil, "b", nil, "c"}, ", ", " and "},
			want: "a, b and c",
		},
		{
			name: "joinAnd() with different separators",
			args: []interface{}{[]interface{}{"apple", "banana", "cherry"}, "; ", " or "},
			want: "apple; banana or cherry",
		},
		// Error cases
		{
			name:    "joinAnd() with wrong number of arguments",
			args:    []interface{}{[]interface{}{"a", "b"}},
			wantErr: true,
		},
		{
			name:    "joinAnd() with non-collection",
			args:    []interface{}{"not a collection", ", ", " and "},
			want:    "n, o, t,  , a,  , c, o, l, l, e, c, t, i, o and n", // strings are treated as character arrays
		},
		{
			name:    "joinAnd() with non-string separator1",
			args:    []interface{}{[]interface{}{"a", "b"}, 123, " and "},
			wantErr: true,
		},
		{
			name:    "joinAnd() with non-string separator2",
			args:    []interface{}{[]interface{}{"a", "b"}, ", ", 456},
			wantErr: true,
		},
		// Edge cases from documentation
		{
			name: "joinAnd() example from docs",
			args: []interface{}{[]interface{}{1, 2, 3, 4}, ", ", " and "},
			want: "1, 2, 3 and 4",
		},
		// String slice support
		{
			name: "joinAnd() with string slice",
			args: []interface{}{[]string{"red", "green", "blue"}, ", ", " and "},
			want: "red, green and blue",
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("joinAnd")
			if !exists {
				t.Skipf("Function joinAnd not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("joinAnd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("joinAnd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplaceFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// Basic replace cases
		{
			name: "replace() simple replacement",
			args: []interface{}{"hello world", "world", "universe"},
			want: "hello universe",
		},
		{
			name: "replace() multiple occurrences",
			args: []interface{}{"the quick brown fox jumps over the lazy dog", "the", "a"},
			want: "a quick brown fox jumps over a lazy dog",
		},
		{
			name: "replace() no match",
			args: []interface{}{"hello world", "xyz", "abc"},
			want: "hello world",
		},
		{
			name: "replace() empty pattern",
			args: []interface{}{"hello", "", "-"},
			want: "-h-e-l-l-o-",
		},
		{
			name: "replace() empty replacement",
			args: []interface{}{"hello world", " world", ""},
			want: "hello",
		},
		{
			name: "replace() with numbers",
			args: []interface{}{123456, "3", "X"},
			want: "12X456",
		},
		{
			name: "replace() with boolean",
			args: []interface{}{true, "true", "false"},
			want: "false",
		},
		{
			name: "replace() case sensitive",
			args: []interface{}{"Hello World", "world", "Earth"},
			want: "Hello World",
		},
		{
			name: "replace() special characters",
			args: []interface{}{"price: $100.00", "$", "€"},
			want: "price: €100.00",
		},
		{
			name: "replace() with nil text returns empty",
			args: []interface{}{nil, "pattern", "replacement"},
			want: "",
		},
		{
			name: "replace() with nil pattern",
			args: []interface{}{"hello", nil, "x"},
			want: "hello",
		},
		{
			name: "replace() with nil replacement",
			args: []interface{}{"hello", "l", nil},
			want: "heo",
		},
		// Error cases
		{
			name:    "replace() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "replace() with one argument",
			args:    []interface{}{"text"},
			wantErr: true,
		},
		{
			name:    "replace() with two arguments",
			args:    []interface{}{"text", "pattern"},
			wantErr: true,
		},
		{
			name:    "replace() with too many arguments",
			args:    []interface{}{"text", "pattern", "replacement", "extra"},
			wantErr: true,
		},
		// Complex patterns
		{
			name: "replace() with repeated pattern",
			args: []interface{}{"aaabbbccc", "b", "X"},
			want: "aaaXXXccc",
		},
		{
			name: "replace() overlapping patterns",
			args: []interface{}{"aaa", "aa", "b"},
			want: "ba",
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("replace")
			if !exists {
				t.Skipf("Function replace not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("replace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("replace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringFunctionsInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		// join() in expressions
		{
			name: "join() with variable",
			expr: "join(items, \", \")",
			data: TemplateData{
				"items": []interface{}{"apple", "banana", "cherry"},
			},
			want: "apple, banana, cherry",
		},
		{
			name: "join() with map result",
			expr: "join(map(\"name\", users), \"; \")",
			data: TemplateData{
				"users": []interface{}{
					map[string]interface{}{"name": "Alice", "age": 30},
					map[string]interface{}{"name": "Bob", "age": 25},
				},
			},
			want: "Alice; Bob",
		},
		{
			name: "join() with list function",
			expr: "join(list(\"a\", \"b\", \"c\"), \"-\")",
			data: TemplateData{},
			want: "a-b-c",
		},
		// joinAnd() in expressions
		{
			name: "joinAnd() with variable",
			expr: "joinAnd(colors, \", \", \" and \")",
			data: TemplateData{
				"colors": []interface{}{"red", "green", "blue"},
			},
			want: "red, green and blue",
		},
		{
			name: "joinAnd() in sentence",
			expr: "\"We have \" + joinAnd(items, \", \", \" and \") + \" in stock.\"",
			data: TemplateData{
				"items": []interface{}{"apples", "oranges", "bananas"},
			},
			want: "We have apples, oranges and bananas in stock.",
		},
		// replace() in expressions
		{
			name: "replace() with variables",
			expr: "replace(template, placeholder, value)",
			data: TemplateData{
				"template":    "Hello, {name}!",
				"placeholder": "{name}",
				"value":       "World",
			},
			want: "Hello, World!",
		},
		{
			name: "replace() chained",
			expr: "replace(replace(text, \"foo\", \"bar\"), \"baz\", \"qux\")",
			data: TemplateData{
				"text": "foo baz foo",
			},
			want: "bar qux bar",
		},
		{
			name: "replace() with uppercase",
			expr: "replace(uppercase(message), \"WORLD\", \"UNIVERSE\")",
			data: TemplateData{
				"message": "Hello World",
			},
			want: "HELLO UNIVERSE",
		},
		// Combined string functions
		{
			name: "join and lowercase",
			expr: "lowercase(join(words, \" \"))",
			data: TemplateData{
				"words": []interface{}{"HELLO", "WORLD"},
			},
			want: "hello world",
		},
		{
			name: "titlecase joined string",
			expr: "titlecase(join(list(first, last), \" \"))",
			data: TemplateData{
				"first": "john",
				"last":  "doe",
			},
			want: "John Doe",
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