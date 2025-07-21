package stencil

import (
	"errors"
	"strings"
	"testing"
)

func TestTemplateErrorIntegration(t *testing.T) {
	// Test basic error creation and messages
	t.Run("Create template error", func(t *testing.T) {
		err := NewTemplateError("invalid syntax", 10, 5)
		if !IsTemplateError(err) {
			t.Error("Expected IsTemplateError to return true")
		}
		if !strings.Contains(err.Error(), "line 10, column 5") {
			t.Errorf("Expected error to contain position info, got: %s", err.Error())
		}
	})

	t.Run("Create parse error", func(t *testing.T) {
		err := NewParseError("unexpected token", "{{for", 42)
		if !IsParseError(err) {
			t.Error("Expected IsParseError to return true")
		}
		if !strings.Contains(err.Error(), "position 42") {
			t.Errorf("Expected error to contain position info, got: %s", err.Error())
		}
	})

	t.Run("Create evaluation error", func(t *testing.T) {
		err := NewEvaluationError("user.name", errors.New("undefined variable"))
		if !IsEvaluationError(err) {
			t.Error("Expected IsEvaluationError to return true")
		}
		if !strings.Contains(err.Error(), "user.name") {
			t.Errorf("Expected error to contain expression, got: %s", err.Error())
		}
	})

	t.Run("Create function error", func(t *testing.T) {
		err := NewFunctionError("uppercase", []interface{}{}, "requires 1 argument")
		if !IsFunctionError(err) {
			t.Error("Expected IsFunctionError to return true")
		}
		if !strings.Contains(err.Error(), "uppercase") {
			t.Errorf("Expected error to contain function name, got: %s", err.Error())
		}
	})

	t.Run("Create document error", func(t *testing.T) {
		err := NewDocumentError("read", "template.docx", errors.New("file not found"))
		if !IsDocumentError(err) {
			t.Error("Expected IsDocumentError to return true")
		}
		if !strings.Contains(err.Error(), "template.docx") {
			t.Errorf("Expected error to contain file path, got: %s", err.Error())
		}
	})
}


func TestMultipleErrors(t *testing.T) {
	// Test collecting multiple validation errors
	multi := NewMultiError()
	
	// Simulate multiple errors during template processing
	multi.Add(NewTemplateError("missing closing tag", 10, 5))
	multi.Add(NewParseError("invalid expression", "{{x+", 15))
	multi.Add(NewEvaluationError("user.id", nil))
	
	err := multi.Err()
	if err == nil {
		t.Fatal("expected multi-error")
	}
	
	errMsg := err.Error()
	if !strings.Contains(errMsg, "3 errors occurred") {
		t.Errorf("expected '3 errors occurred' in message, got: %s", errMsg)
	}
}

func TestErrorContextIntegration(t *testing.T) {
	// Test error with context
	baseErr := NewTemplateError("invalid syntax", 5, 10)
	
	contextErr := WithContext(baseErr, "processing template", map[string]interface{}{
		"file":     "template.docx",
		"fragment": "header",
	})
	
	errMsg := contextErr.Error()
	if !strings.Contains(errMsg, "processing template") {
		t.Errorf("expected operation in error message, got: %s", errMsg)
	}
	
	// Should still be able to get the original error type
	var templateErr *TemplateError
	if !errors.As(contextErr, &templateErr) {
		t.Error("should be able to unwrap to TemplateError")
	}
}

func TestErrorTypeChecking(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		checker func(error) bool
		want    bool
	}{
		{
			name:    "IsTemplateError - true",
			err:     NewTemplateError("test", 1, 1),
			checker: IsTemplateError,
			want:    true,
		},
		{
			name:    "IsTemplateError - false",
			err:     NewParseError("test", "", 0),
			checker: IsTemplateError,
			want:    false,
		},
		{
			name:    "IsParseError - true",
			err:     NewParseError("test", "", 0),
			checker: IsParseError,
			want:    true,
		},
		{
			name:    "IsEvaluationError - true",
			err:     NewEvaluationError("expr", nil),
			checker: IsEvaluationError,
			want:    true,
		},
		{
			name:    "IsFunctionError - true",
			err:     NewFunctionError("test", nil, "error"),
			checker: IsFunctionError,
			want:    true,
		},
		{
			name:    "IsDocumentError - true",
			err:     NewDocumentError("read", "file.docx", nil),
			checker: IsDocumentError,
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.checker(tt.err); got != tt.want {
				t.Errorf("checker() = %v, want %v", got, tt.want)
			}
		})
	}
}