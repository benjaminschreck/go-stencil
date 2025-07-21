package stencil

import (
	"testing"
)

func TestEvaluateVariable(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		data       TemplateData
		want       interface{}
		wantErr    bool
	}{
		{
			name:       "simple variable",
			expression: "name",
			data: TemplateData{
				"name": "John Doe",
			},
			want:    "John Doe",
			wantErr: false,
		},
		{
			name:       "numeric variable",
			expression: "age",
			data: TemplateData{
				"age": 30,
			},
			want:    30,
			wantErr: false,
		},
		{
			name:       "boolean variable",
			expression: "active",
			data: TemplateData{
				"active": true,
			},
			want:    true,
			wantErr: false,
		},
		{
			name:       "missing variable",
			expression: "missing",
			data:       TemplateData{},
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "variable with spaces",
			expression: " name ",
			data: TemplateData{
				"name": "John",
			},
			want:    "John",
			wantErr: false,
		},
		{
			name:       "empty data map",
			expression: "anything",
			data:       nil,
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "float variable",
			expression: "price",
			data: TemplateData{
				"price": 19.99,
			},
			want:    19.99,
			wantErr: false,
		},
		{
			name:       "nil value",
			expression: "nullValue",
			data: TemplateData{
				"nullValue": nil,
			},
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateVariable(tt.expression, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateVariable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateNestedField(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		data       TemplateData
		want       interface{}
		wantErr    bool
	}{
		// Dot notation tests
		{
			name:       "simple dot notation",
			expression: "customer.name",
			data: TemplateData{
				"customer": map[string]interface{}{
					"name": "John Doe",
				},
			},
			want:    "John Doe",
			wantErr: false,
		},
		{
			name:       "nested dot notation",
			expression: "customer.address.city",
			data: TemplateData{
				"customer": map[string]interface{}{
					"address": map[string]interface{}{
						"city": "New York",
					},
				},
			},
			want:    "New York",
			wantErr: false,
		},
		{
			name:       "missing nested field",
			expression: "customer.missing",
			data: TemplateData{
				"customer": map[string]interface{}{
					"name": "John",
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:       "missing parent field",
			expression: "missing.field",
			data:       TemplateData{},
			want:       nil,
			wantErr:    false,
		},
		{
			name:       "dot notation with spaces",
			expression: " customer.name ",
			data: TemplateData{
				"customer": map[string]interface{}{
					"name": "Jane",
				},
			},
			want:    "Jane",
			wantErr: false,
		},
		// Bracket notation tests
		{
			name:       "simple bracket notation",
			expression: "customer['name']",
			data: TemplateData{
				"customer": map[string]interface{}{
					"name": "John Doe",
				},
			},
			want:    "John Doe",
			wantErr: false,
		},
		{
			name:       "bracket notation with double quotes",
			expression: `customer["name"]`,
			data: TemplateData{
				"customer": map[string]interface{}{
					"name": "John Doe",
				},
			},
			want:    "John Doe",
			wantErr: false,
		},
		{
			name:       "nested bracket notation",
			expression: "customer['address']['city']",
			data: TemplateData{
				"customer": map[string]interface{}{
					"address": map[string]interface{}{
						"city": "Boston",
					},
				},
			},
			want:    "Boston",
			wantErr: false,
		},
		{
			name:       "mixed dot and bracket notation",
			expression: "customer.address['city']",
			data: TemplateData{
				"customer": map[string]interface{}{
					"address": map[string]interface{}{
						"city": "Chicago",
					},
				},
			},
			want:    "Chicago",
			wantErr: false,
		},
		{
			name:       "bracket notation with spaces in key",
			expression: "data['first name']",
			data: TemplateData{
				"data": map[string]interface{}{
					"first name": "Alice",
				},
			},
			want:    "Alice",
			wantErr: false,
		},
		// Array access tests
		{
			name:       "array index access",
			expression: "items[0]",
			data: TemplateData{
				"items": []interface{}{"first", "second", "third"},
			},
			want:    "first",
			wantErr: false,
		},
		{
			name:       "array index with dot notation",
			expression: "items[1].name",
			data: TemplateData{
				"items": []interface{}{
					map[string]interface{}{"name": "Item 1"},
					map[string]interface{}{"name": "Item 2"},
				},
			},
			want:    "Item 2",
			wantErr: false,
		},
		{
			name:       "out of bounds array access",
			expression: "items[10]",
			data: TemplateData{
				"items": []interface{}{"a", "b"},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:       "negative array index",
			expression: "items[-1]",
			data: TemplateData{
				"items": []interface{}{"a", "b", "c"},
			},
			want:    "c",
			wantErr: false,
		},
		// Type coercion tests
		{
			name:       "accessing non-map with dot notation",
			expression: "name.field",
			data: TemplateData{
				"name": "string value",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name:       "accessing non-array with bracket index",
			expression: "name[0]",
			data: TemplateData{
				"name": "string",
			},
			want:    nil,
			wantErr: false,
		},
		// Complex nested structures
		{
			name:       "deeply nested structure",
			expression: "data.users[0].profile.settings['theme']",
			data: TemplateData{
				"data": map[string]interface{}{
					"users": []interface{}{
						map[string]interface{}{
							"profile": map[string]interface{}{
								"settings": map[string]interface{}{
									"theme": "dark",
								},
							},
						},
					},
				},
			},
			want:    "dark",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateVariable(tt.expression, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateVariable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EvaluateVariable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{
			name:  "string value",
			value: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "integer value",
			value: 42,
			want:  "42",
		},
		{
			name:  "float value",
			value: 3.14,
			want:  "3.14",
		},
		{
			name:  "boolean true",
			value: true,
			want:  "true",
		},
		{
			name:  "boolean false",
			value: false,
			want:  "false",
		},
		{
			name:  "nil value",
			value: nil,
			want:  "",
		},
		{
			name:  "slice value",
			value: []string{"a", "b", "c"},
			want:  "[a b c]",
		},
		{
			name:  "map value",
			value: map[string]int{"a": 1},
			want:  "map[a:1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatValue(tt.value)
			if got != tt.want {
				t.Errorf("FormatValue() = %v, want %v", got, tt.want)
			}
		})
	}
}