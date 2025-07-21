package stencil

import (
	"testing"
)

func TestFormatFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// Basic formatting tests
		{
			name: "format() integer with %d",
			args: []interface{}{"%d", 42},
			want: "42",
		},
		{
			name: "format() float with %f",
			args: []interface{}{"%f", 3.14159},
			want: "3.141590",
		},
		{
			name: "format() float with %.2f",
			args: []interface{}{"%.2f", 3.14159},
			want: "3.14",
		},
		{
			name: "format() float with %.2f large number",
			args: []interface{}{"%.2f", 1000000.0},
			want: "1000000.00",
		},
		{
			name: "format() string with %s",
			args: []interface{}{"%s", "hello"},
			want: "hello",
		},
		{
			name: "format() multiple arguments",
			args: []interface{}{"Hello %s, you have %d items worth $%.2f", "John", 5, 123.456},
			want: "Hello John, you have 5 items worth $123.46",
		},
		// Type conversions
		{
			name: "format() string to integer",
			args: []interface{}{"%d", "123"},
			want: "123",
		},
		{
			name: "format() float to integer",
			args: []interface{}{"%d", 123.789},
			want: "123",
		},
		{
			name: "format() integer to float",
			args: []interface{}{"%.2f", 100},
			want: "100.00",
		},
		// Special formatting
		{
			name: "format() with padding",
			args: []interface{}{"%5d", 42},
			want: "   42",
		},
		{
			name: "format() with zero padding",
			args: []interface{}{"%05d", 42},
			want: "00042",
		},
		{
			name: "format() with left alignment",
			args: []interface{}{"%-5d", 42},
			want: "42   ",
		},
		{
			name: "format() percentage literal",
			args: []interface{}{"100%%"},
			want: "100%",
		},
		{
			name: "format() hexadecimal",
			args: []interface{}{"%x", 255},
			want: "ff",
		},
		{
			name: "format() uppercase hexadecimal",
			args: []interface{}{"%X", 255},
			want: "FF",
		},
		// Nil handling
		{
			name: "format() with nil pattern",
			args: []interface{}{nil, 42},
			want: nil,
		},
		{
			name: "format() with nil value for %s",
			args: []interface{}{"%s", nil},
			want: "null",
		},
		{
			name: "format() with nil value for %d",
			args: []interface{}{"%d", nil},
			want: "0",
		},
		// Error cases
		{
			name:    "format() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "format() with only pattern",
			args:    []interface{}{"%d"},
			wantErr: true,
		},
		{
			name: "format() with %q pattern",
			args: []interface{}{"%q", "hello"},
			want: `"hello"`,
		},
		{
			name:    "format() type mismatch",
			args:    []interface{}{"%d", "not a number"},
			wantErr: true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("format")
			if !exists {
				t.Skipf("Function format not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("format() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatWithLocaleFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// English locale (default)
		{
			name: "formatWithLocale() English decimal",
			args: []interface{}{"en", "%.2f", 1000000.0},
			want: "1000000.00",
		},
		{
			name: "formatWithLocale() English decimal",
			args: []interface{}{"en-US", "%.2f", 1234.56},
			want: "1234.56",
		},
		// German locale
		{
			name: "formatWithLocale() German decimal",
			args: []interface{}{"de-DE", "%.2f", 1234.56},
			want: "1234.56",
		},
		// Complex formatting
		{
			name: "formatWithLocale() with multiple values",
			args: []interface{}{"en", "Total: %.2f EUR", 1234567.89},
			want: "Total: 1234567.89 EUR",
		},
		// Nil handling
		{
			name: "formatWithLocale() with nil locale",
			args: []interface{}{nil, "%.2f", 123.45},
			want: nil,
		},
		{
			name: "formatWithLocale() with nil pattern",
			args: []interface{}{"en", nil, 123.45},
			want: nil,
		},
		// Error cases
		{
			name:    "formatWithLocale() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "formatWithLocale() with one argument",
			args:    []interface{}{"en"},
			wantErr: true,
		},
		{
			name:    "formatWithLocale() with two arguments",
			args:    []interface{}{"en", "%.2f"},
			wantErr: true,
		},
		{
			name:    "formatWithLocale() with invalid locale",
			args:    []interface{}{"invalid-locale", "%.2f", 123.45},
			want:    "123.45", // Should fall back to default formatting
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("formatWithLocale")
			if !exists {
				t.Skipf("Function formatWithLocale not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatWithLocale() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("formatWithLocale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCurrencyFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// Default locale (assuming en-US for tests)
		{
			name: "currency() simple integer",
			args: []interface{}{100},
			want: "$100.00",
		},
		{
			name: "currency() simple float",
			args: []interface{}{123.45},
			want: "$123.45",
		},
		{
			name: "currency() thousands",
			args: []interface{}{1234567.89},
			want: "$1,234,567.89",
		},
		{
			name: "currency() negative",
			args: []interface{}{-123.45},
			want: "-$123.45",
		},
		// With locale
		{
			name: "currency() Euro",
			args: []interface{}{123.45, "de-DE"},
			want: "123,45 €",
		},
		{
			name: "currency() British Pound",
			args: []interface{}{123.45, "en-GB"},
			want: "£123.45",
		},
		{
			name: "currency() Japanese Yen",
			args: []interface{}{1234, "ja-JP"},
			want: "¥1,234",
		},
		{
			name: "currency() Hungarian Forint",
			args: []interface{}{123.45, "hu-HU"},
			want: "123,45 Ft",
		},
		{
			name: "currency() French Euro",
			args: []interface{}{1234.56, "fr-FR"},
			want: "1 234,56 €",
		},
		// Type conversion
		{
			name: "currency() string number",
			args: []interface{}{"123.45"},
			want: "$123.45",
		},
		{
			name: "currency() integer",
			args: []interface{}{123},
			want: "$123.00",
		},
		// Nil handling
		{
			name: "currency() with nil value",
			args: []interface{}{nil},
			want: nil,
		},
		{
			name: "currency() with nil locale",
			args: []interface{}{123.45, nil},
			want: "$123.45", // Should use default locale
		},
		// Error cases
		{
			name:    "currency() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "currency() with too many arguments",
			args:    []interface{}{123, "en-US", "extra"},
			wantErr: true,
		},
		{
			name:    "currency() with non-numeric value",
			args:    []interface{}{"not a number"},
			wantErr: true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("currency")
			if !exists {
				t.Skipf("Function currency not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("currency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("currency() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPercentFunction(t *testing.T) {
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// Default locale (assuming en-US)
		{
			name: "percent() decimal as percentage",
			args: []interface{}{0.95},
			want: "95%",
		},
		{
			name: "percent() whole number",
			args: []interface{}{1.0},
			want: "100%",
		},
		{
			name: "percent() small decimal",
			args: []interface{}{0.1234},
			want: "12.34%",
		},
		{
			name: "percent() zero",
			args: []interface{}{0.0},
			want: "0%",
		},
		{
			name: "percent() greater than 100%",
			args: []interface{}{1.5},
			want: "150%",
		},
		{
			name: "percent() very large percentage",
			args: []interface{}{123.0},
			want: "12,300%",
		},
		// With locale
		{
			name: "percent() German locale",
			args: []interface{}{0.95, "de-DE"},
			want: "95 %",
		},
		{
			name: "percent() French locale",
			args: []interface{}{0.1234, "fr-FR"},
			want: "12,34 %",
		},
		{
			name: "percent() Hungarian locale",
			args: []interface{}{0.95, "hu-HU"},
			want: "95%",
		},
		{
			name: "percent() Hungarian large number",
			args: []interface{}{123.0, "hu-HU"},
			want: "12 300%",
		},
		// Type conversion
		{
			name: "percent() integer",
			args: []interface{}{1},
			want: "100%",
		},
		{
			name: "percent() string number",
			args: []interface{}{"0.75"},
			want: "75%",
		},
		// Nil handling
		{
			name: "percent() with nil value",
			args: []interface{}{nil},
			want: nil,
		},
		{
			name: "percent() with nil locale",
			args: []interface{}{0.95, nil},
			want: "95%", // Should use default locale
		},
		// Error cases
		{
			name:    "percent() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "percent() with too many arguments",
			args:    []interface{}{0.95, "en-US", "extra"},
			wantErr: true,
		},
		{
			name:    "percent() with non-numeric value",
			args:    []interface{}{"not a number"},
			wantErr: true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("percent")
			if !exists {
				t.Skipf("Function percent not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("percent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("percent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNumberFormattingInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		// format() in expressions
		{
			name: "format() with variable",
			expr: `format("Total: $%.2f", total)`,
			data: TemplateData{"total": 123.456},
			want: "Total: $123.46",
		},
		{
			name: "format() with calculation",
			expr: `format("%.2f", price * quantity)`,
			data: TemplateData{"price": 10.5, "quantity": 3},
			want: "31.50",
		},
		{
			name: "format() with nested field",
			expr: `format("%d items", order.count)`,
			data: TemplateData{
				"order": map[string]interface{}{
					"count": 42,
				},
			},
			want: "42 items",
		},
		// formatWithLocale() in expressions
		{
			name: "formatWithLocale() with variables",
			expr: `formatWithLocale(locale, "%.2f", amount)`,
			data: TemplateData{
				"locale": "en",
				"amount": 1234.56,
			},
			want: "1234.56",
		},
		// currency() in expressions
		{
			name: "currency() with variable",
			expr: `currency(price)`,
			data: TemplateData{"price": 99.99},
			want: "$99.99",
		},
		{
			name: "currency() with locale variable",
			expr: `currency(amount, userLocale)`,
			data: TemplateData{
				"amount":     123.45,
				"userLocale": "en-GB",
			},
			want: "£123.45",
		},
		// percent() in expressions
		{
			name: "percent() with calculation",
			expr: `percent(completed / total)`,
			data: TemplateData{
				"completed": 85.0,
				"total":     100.0,
			},
			want: "85%",
		},
		{
			name: "percent() in concatenation",
			expr: `"Progress: " + percent(0.75)`,
			data: TemplateData{},
			want: "Progress: 75%",
		},
		// Combined functions
		{
			name: "format with currency",
			expr: `format("Price: %s", currency(99.99))`,
			data: TemplateData{},
			want: "Price: $99.99",
		},
		{
			name: "conditional formatting",
			expr: `format("Status: %s", percent(score))`,
			data: TemplateData{"score": 0.92},
			want: "Status: 92%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if functions not implemented
			registry := GetDefaultFunctionRegistry()
			funcs := []string{"format", "formatWithLocale", "currency", "percent"}
			for _, fname := range funcs {
				if _, exists := registry.GetFunction(fname); !exists {
					t.Skipf("Function %s not yet implemented", fname)
					return
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