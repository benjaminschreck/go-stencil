package stencil

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType string
		wantMsg  string
	}{
		{
			name:     "TemplateError",
			err:      &TemplateError{Message: "invalid syntax", Line: 10, Column: 5},
			wantType: "TemplateError",
			wantMsg:  "template error at line 10, column 5: invalid syntax",
		},
		{
			name:     "ParseError",
			err:      &ParseError{Message: "unexpected token", Token: "{{for", Position: 42},
			wantType: "ParseError",
			wantMsg:  "parse error at position 42 near '{{for': unexpected token",
		},
		{
			name:     "EvaluationError",
			err:      &EvaluationError{Expression: "user.name", Cause: errors.New("undefined variable")},
			wantType: "EvaluationError",
			wantMsg:  "evaluation error for expression 'user.name': undefined variable",
		},
		{
			name:     "FunctionError",
			err:      &FunctionError{Function: "uppercase", Args: []interface{}{"test", 123}, Message: "invalid argument type"},
			wantType: "FunctionError",
			wantMsg:  "function error in 'uppercase(test, 123)': invalid argument type",
		},
		{
			name:     "DocumentError",
			err:      &DocumentError{Operation: "save", Path: "output.docx", Cause: errors.New("permission denied")},
			wantType: "DocumentError",
			wantMsg:  "document error during save of 'output.docx': permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Error() method
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}

			// Test type assertions
			switch tt.wantType {
			case "TemplateError":
				if _, ok := tt.err.(*TemplateError); !ok {
					t.Errorf("Expected *TemplateError, got %T", tt.err)
				}
			case "ParseError":
				if _, ok := tt.err.(*ParseError); !ok {
					t.Errorf("Expected *ParseError, got %T", tt.err)
				}
			case "EvaluationError":
				if _, ok := tt.err.(*EvaluationError); !ok {
					t.Errorf("Expected *EvaluationError, got %T", tt.err)
				}
			case "FunctionError":
				if _, ok := tt.err.(*FunctionError); !ok {
					t.Errorf("Expected *FunctionError, got %T", tt.err)
				}
			case "DocumentError":
				if _, ok := tt.err.(*DocumentError); !ok {
					t.Errorf("Expected *DocumentError, got %T", tt.err)
				}
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test wrapping errors
	baseErr := errors.New("base error")
	
	evalErr := &EvaluationError{
		Expression: "x + y",
		Cause:     baseErr,
	}
	
	// Test Unwrap
	if unwrapped := errors.Unwrap(evalErr); unwrapped != baseErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
	}
	
	// Test Is
	if !errors.Is(evalErr, baseErr) {
		t.Error("errors.Is() should return true for wrapped error")
	}
}

func TestNewTemplateError(t *testing.T) {
	err := NewTemplateError("invalid syntax", 10, 5)
	
	templateErr, ok := err.(*TemplateError)
	if !ok {
		t.Fatalf("NewTemplateError should return *TemplateError, got %T", err)
	}
	
	if templateErr.Line != 10 || templateErr.Column != 5 {
		t.Errorf("NewTemplateError position = (%d, %d), want (10, 5)", 
			templateErr.Line, templateErr.Column)
	}
}

func TestNewParseError(t *testing.T) {
	err := NewParseError("unexpected token", "{{for", 42)
	
	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("NewParseError should return *ParseError, got %T", err)
	}
	
	if parseErr.Token != "{{for" || parseErr.Position != 42 {
		t.Errorf("NewParseError token/position = (%s, %d), want ({{for, 42)", 
			parseErr.Token, parseErr.Position)
	}
}

func TestErrorRecovery(t *testing.T) {
	// Test that we can recover from panics and convert them to errors
	defer func() {
		if r := recover(); r != nil {
			err := RecoverError(r)
			if err == nil {
				t.Error("RecoverError should return an error for panic")
			}
			if !strings.Contains(err.Error(), "panic recovered") {
				t.Errorf("RecoverError message should contain 'panic recovered', got: %s", err.Error())
			}
		}
	}()
	
	// This should panic
	panic("test panic")
}

func TestErrorContext(t *testing.T) {
	// Test adding context to errors
	baseErr := errors.New("file not found")
	
	contextErr := WithContext(baseErr, "preparing template", map[string]interface{}{
		"file": "template.docx",
		"size": 1024,
	})
	
	if !strings.Contains(contextErr.Error(), "file not found") {
		t.Error("WithContext should preserve original error message")
	}
	
	if !strings.Contains(contextErr.Error(), "preparing template") {
		t.Error("WithContext should include operation context")
	}
}

func TestMultiError(t *testing.T) {
	// Test collecting multiple errors
	multi := NewMultiError()
	
	multi.Add(errors.New("error 1"))
	multi.Add(errors.New("error 2"))
	multi.Add(nil) // Should be ignored
	multi.Add(errors.New("error 3"))
	
	if multi.Len() != 3 {
		t.Errorf("MultiError.Len() = %d, want 3", multi.Len())
	}
	
	err := multi.Err()
	if err == nil {
		t.Error("MultiError.Err() should return non-nil for non-empty errors")
	}
	
	// Test empty MultiError
	emptyMulti := NewMultiError()
	if emptyMulti.Err() != nil {
		t.Error("MultiError.Err() should return nil for empty errors")
	}
}

func TestValidationError(t *testing.T) {
	// Test validation errors with multiple issues
	validationErr := &ValidationError{
		Issues: []ValidationIssue{
			{Field: "name", Message: "required field"},
			{Field: "age", Message: "must be positive"},
			{Field: "email", Message: "invalid format"},
		},
	}
	
	errMsg := validationErr.Error()
	if !strings.Contains(errMsg, "3 validation issues") {
		t.Errorf("ValidationError should mention issue count, got: %s", errMsg)
	}
	
	// Test individual issue access
	if len(validationErr.Issues) != 3 {
		t.Errorf("ValidationError.Issues length = %d, want 3", len(validationErr.Issues))
	}
}