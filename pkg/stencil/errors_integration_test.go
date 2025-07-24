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
		if _, ok := err.(*TemplateError); !ok {
			t.Error("Expected TemplateError type")
		}
		if !strings.Contains(err.Error(), "line 10, column 5") {
			t.Errorf("Expected error to contain position info, got: %s", err.Error())
		}
	})

	t.Run("Create parse error", func(t *testing.T) {
		err := NewParseError("unexpected token", "{{for", 42)
		if _, ok := err.(*ParseError); !ok {
			t.Error("Expected ParseError type")
		}
		if !strings.Contains(err.Error(), "position 42") {
			t.Errorf("Expected error to contain position info, got: %s", err.Error())
		}
	})

	t.Run("Create evaluation error", func(t *testing.T) {
		err := NewEvaluationError("user.name", errors.New("undefined variable"))
		if _, ok := err.(*EvaluationError); !ok {
			t.Error("Expected EvaluationError type")
		}
		if !strings.Contains(err.Error(), "user.name") {
			t.Errorf("Expected error to contain expression, got: %s", err.Error())
		}
	})

	t.Run("Create function error", func(t *testing.T) {
		err := NewFunctionError("uppercase", []interface{}{}, "requires 1 argument")
		if _, ok := err.(*FunctionError); !ok {
			t.Error("Expected FunctionError type")
		}
		if !strings.Contains(err.Error(), "uppercase") {
			t.Errorf("Expected error to contain function name, got: %s", err.Error())
		}
	})

	t.Run("Create document error", func(t *testing.T) {
		err := NewDocumentError("read", "template.docx", errors.New("file not found"))
		if _, ok := err.(*DocumentError); !ok {
			t.Error("Expected DocumentError type")
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

