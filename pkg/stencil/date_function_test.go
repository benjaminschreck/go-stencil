package stencil

import (
	"testing"
	"time"
)

func TestDateFunction(t *testing.T) {
	// Create test dates
	testDate := time.Date(2024, time.January, 15, 14, 30, 45, 0, time.UTC)
	
	tests := []struct {
		name    string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// Basic date formatting tests
		{
			name: "date() with simple format",
			args: []interface{}{"2006-01-02", testDate},
			want: "2024-01-15",
		},
		{
			name: "date() with full datetime format",
			args: []interface{}{"2006-01-02 15:04:05", testDate},
			want: "2024-01-15 14:30:45",
		},
		{
			name: "date() with month name",
			args: []interface{}{"January 2, 2006", testDate},
			want: "January 15, 2024",
		},
		{
			name: "date() with short month name",
			args: []interface{}{"Jan 2, 2006", testDate},
			want: "Jan 15, 2024",
		},
		{
			name: "date() with day of week",
			args: []interface{}{"Mon, 02 Jan 2006", testDate},
			want: "Mon, 15 Jan 2024",
		},
		{
			name: "date() with 12-hour format",
			args: []interface{}{"3:04 PM", testDate},
			want: "2:30 PM",
		},
		{
			name: "date() with timezone",
			args: []interface{}{"2006-01-02T15:04:05Z07:00", testDate},
			want: "2024-01-15T14:30:45Z",
		},
		// String parsing tests
		{
			name: "date() parsing ISO8601 string",
			args: []interface{}{"2006-01-02", "2024-01-15T14:30:45Z"},
			want: "2024-01-15",
		},
		{
			name: "date() parsing date-only string",
			args: []interface{}{"Jan 2, 2006", "2024-01-15"},
			want: "Jan 15, 2024",
		},
		{
			name: "date() parsing RFC3339 string",
			args: []interface{}{"2006-01-02", "2024-01-15T14:30:45Z"},
			want: "2024-01-15",
		},
		// Different format patterns
		{
			name: "date() with year only",
			args: []interface{}{"2006", testDate},
			want: "2024",
		},
		{
			name: "date() with month/day only",
			args: []interface{}{"01/02", testDate},
			want: "01/15",
		},
		{
			name: "date() with custom separator",
			args: []interface{}{"2006.01.02", testDate},
			want: "2024.01.15",
		},
		// Locale support (3 arguments)
		{
			name: "date() with locale - English",
			args: []interface{}{"en", "January 2, 2006", testDate},
			want: "January 15, 2024",
		},
		{
			name: "date() with locale - German month",
			args: []interface{}{"de", "2. January 2006", testDate},
			want: "15. Januar 2024",
		},
		{
			name: "date() with locale - French month",
			args: []interface{}{"fr", "2 January 2006", testDate},
			want: "15 janvier 2024",
		},
		{
			name: "date() with locale - Spanish weekday",
			args: []interface{}{"es", "Monday, 2 January 2006", testDate},
			want: "lunes, 15 enero 2024",
		},
		// Nil and empty handling
		{
			name: "date() with nil format",
			args: []interface{}{nil, testDate},
			want: nil,
		},
		{
			name: "date() with nil date",
			args: []interface{}{"2006-01-02", nil},
			want: nil,
		},
		{
			name: "date() with empty string date",
			args: []interface{}{"2006-01-02", ""},
			want: nil,
		},
		{
			name: "date() with nil locale but valid args",
			args: []interface{}{nil, "2006-01-02", testDate},
			want: nil,
		},
		// Unix timestamp support
		{
			name: "date() with unix timestamp",
			args: []interface{}{"2006-01-02", int64(1705329045)}, // 2024-01-15 14:30:45 UTC
			want: "2024-01-15",
		},
		// Error cases
		{
			name:    "date() with no arguments",
			args:    []interface{}{},
			wantErr: true,
		},
		{
			name:    "date() with one argument",
			args:    []interface{}{"2006-01-02"},
			wantErr: true,
		},
		{
			name:    "date() with too many arguments",
			args:    []interface{}{"en", "2006-01-02", testDate, "extra"},
			wantErr: true,
		},
		{
			name:    "date() with unparseable date string",
			args:    []interface{}{"2006-01-02", "not a date"},
			wantErr: true,
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("date")
			if !exists {
				t.Skipf("Function date not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("date() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("date() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDateFunctionInExpressions(t *testing.T) {
	// Create test date
	testDate := time.Date(2024, time.March, 10, 9, 15, 0, 0, time.UTC)
	
	tests := []struct {
		name    string
		expr    string
		data    TemplateData
		want    interface{}
		wantErr bool
	}{
		// date() with variables
		{
			name: "date() with date variable",
			expr: "date(\"2006-01-02\", birthDate)",
			data: TemplateData{
				"birthDate": testDate,
			},
			want: "2024-03-10",
		},
		{
			name: "date() with format from variable",
			expr: "date(format, myDate)",
			data: TemplateData{
				"format": "Jan 2, 2006",
				"myDate": testDate,
			},
			want: "Mar 10, 2024",
		},
		{
			name: "date() with nested field",
			expr: "date(\"2006-01-02\", user.createdAt)",
			data: TemplateData{
				"user": map[string]interface{}{
					"createdAt": testDate,
				},
			},
			want: "2024-03-10",
		},
		// date() with locale expressions
		{
			name: "date() with locale variable",
			expr: "date(lang, \"January 2, 2006\", eventDate)",
			data: TemplateData{
				"lang":      "fr",
				"eventDate": testDate,
			},
			want: "mars 10, 2024",
		},
		{
			name: "date() with all variables",
			expr: "date(locale, dateFormat, timestamp)",
			data: TemplateData{
				"locale":     "de",
				"dateFormat": "2. January 2006",
				"timestamp":  testDate,
			},
			want: "10. März 2024",
		},
		// Combined with other functions
		{
			name: "date() with uppercase",
			expr: "uppercase(date(\"Jan\", myDate))",
			data: TemplateData{
				"myDate": testDate,
			},
			want: "MAR",
		},
		{
			name: "date() with string concatenation",
			expr: "\"Today is \" + date(\"Monday\", now)",
			data: TemplateData{
				"now": testDate,
			},
			want: "Today is Sunday",
		},
		{
			name: "date() in conditional",
			expr: "date(\"2006\", created) == \"2024\"",
			data: TemplateData{
				"created": testDate,
			},
			want: true,
		},
		// Edge cases
		{
			name: "date() with literal strings",
			expr: "date(\"2006-01-02\", \"2024-03-10\")",
			data: TemplateData{},
			want: "2024-03-10",
		},
		{
			name: "date() with nil check",
			expr: "coalesce(date(\"2006-01-02\", missing), \"No date\")",
			data: TemplateData{
				"missing": nil,
			},
			want: "No date",
		},
		// Error cases
		{
			name: "date() with invalid date",
			expr: "date(\"2006-01-02\", \"invalid\")",
			data: TemplateData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if function not implemented
			registry := GetDefaultFunctionRegistry()
			if _, exists := registry.GetFunction("date"); !exists {
				t.Skip("Function date not yet implemented")
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

func TestDateFunctionWithDifferentTimeTypes(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
		want string
	}{
		// Different Go time representations
		{
			name: "date() with time.Time",
			args: []interface{}{"2006-01-02", time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC)},
			want: "2024-05-20",
		},
		{
			name: "date() with pointer to time.Time",
			args: []interface{}{"2006-01-02", func() *time.Time { t := time.Date(2024, 5, 20, 0, 0, 0, 0, time.UTC); return &t }()},
			want: "2024-05-20",
		},
		{
			name: "date() with millisecond unix timestamp",
			args: []interface{}{"2006-01-02", int64(1716163200000)}, // 2024-05-20 in milliseconds
			want: "2024-05-20",
		},
		// Different string formats to parse
		{
			name: "date() parsing RFC3339",
			args: []interface{}{"Jan 2", "2024-05-20T10:30:00Z"},
			want: "May 20",
		},
		{
			name: "date() parsing date with slashes",
			args: []interface{}{"January", "05/20/2024"},
			want: "May",
		},
		{
			name: "date() parsing date with dots",
			args: []interface{}{"2006", "20.05.2024"},
			want: "2024",
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("date")
			if !exists {
				t.Skipf("Function date not yet implemented")
				return
			}

			got, err := fn.Call(tt.args...)
			if err != nil {
				t.Errorf("date() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("date() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDateFunctionLocaleFormats(t *testing.T) {
	testDate := time.Date(2024, time.July, 4, 0, 0, 0, 0, time.UTC)
	
	tests := []struct {
		name   string
		locale string
		format string
		want   string
	}{
		// Different locale formats
		{
			name:   "English weekday",
			locale: "en",
			format: "Monday",
			want:   "Thursday",
		},
		{
			name:   "German weekday",
			locale: "de",
			format: "Monday",
			want:   "Donnerstag",
		},
		{
			name:   "French weekday",
			locale: "fr",
			format: "Monday",
			want:   "jeudi",
		},
		{
			name:   "Spanish month",
			locale: "es",
			format: "January",
			want:   "julio",
		},
		{
			name:   "Italian month",
			locale: "it",
			format: "January 2006",
			want:   "luglio 2024",
		},
		// Complex locale formats
		{
			name:   "German full date",
			locale: "de-DE",
			format: "Monday, 2. January 2006",
			want:   "Donnerstag, 4. Juli 2024",
		},
		{
			name:   "French Canadian",
			locale: "fr-CA", 
			format: "2 January 2006",
			want:   "4 juillet 2024",
		},
	}

	registry := GetDefaultFunctionRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, exists := registry.GetFunction("date")
			if !exists {
				t.Skipf("Function date not yet implemented")
				return
			}

			got, err := fn.Call(tt.locale, tt.format, testDate)
			if err != nil {
				t.Errorf("date() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("date() = %v, want %v", got, tt.want)
			}
		})
	}
}