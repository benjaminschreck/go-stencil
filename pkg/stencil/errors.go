// Package stencil provides custom error types for better error handling and reporting.
package stencil

import (
	"fmt"
	"strings"
)

// TemplateError represents an error in the template structure or syntax
type TemplateError struct {
	Message string
	Line    int
	Column  int
}

func (e *TemplateError) Error() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("template error at line %d, column %d: %s", e.Line, e.Column, e.Message)
	} else if e.Line > 0 {
		return fmt.Sprintf("template error at line %d: %s", e.Line, e.Message)
	}
	return fmt.Sprintf("template error: %s", e.Message)
}

// NewTemplateError creates a new template error with position information
func NewTemplateError(message string, line, column int) error {
	return &TemplateError{
		Message: message,
		Line:    line,
		Column:  column,
	}
}

// ParseError represents an error during template parsing
type ParseError struct {
	Message  string
	Token    string
	Position int
}

func (e *ParseError) Error() string {
	if e.Token != "" {
		return fmt.Sprintf("parse error at position %d near '%s': %s", e.Position, e.Token, e.Message)
	}
	return fmt.Sprintf("parse error at position %d: %s", e.Position, e.Message)
}

// NewParseError creates a new parse error
func NewParseError(message, token string, position int) error {
	return &ParseError{
		Message:  message,
		Token:    token,
		Position: position,
	}
}

// EvaluationError represents an error during expression evaluation
type EvaluationError struct {
	Expression string
	Cause      error
}

func (e *EvaluationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("evaluation error for expression '%s': %v", e.Expression, e.Cause)
	}
	return fmt.Sprintf("evaluation error for expression '%s'", e.Expression)
}

func (e *EvaluationError) Unwrap() error {
	return e.Cause
}

// NewEvaluationError creates a new evaluation error
func NewEvaluationError(expression string, cause error) error {
	return &EvaluationError{
		Expression: expression,
		Cause:      cause,
	}
}

// FunctionError represents an error in a template function call
type FunctionError struct {
	Function string
	Args     []interface{}
	Message  string
}

func (e *FunctionError) Error() string {
	argsStr := make([]string, len(e.Args))
	for i, arg := range e.Args {
		argsStr[i] = fmt.Sprintf("%v", arg)
	}
	return fmt.Sprintf("function error in '%s(%s)': %s", e.Function, strings.Join(argsStr, ", "), e.Message)
}

// NewFunctionError creates a new function error
func NewFunctionError(function string, args []interface{}, message string) error {
	return &FunctionError{
		Function: function,
		Args:     args,
		Message:  message,
	}
}

// DocumentError represents an error during document operations
type DocumentError struct {
	Operation string
	Path      string
	Cause     error
}

func (e *DocumentError) Error() string {
	if e.Path != "" && e.Cause != nil {
		return fmt.Sprintf("document error during %s of '%s': %v", e.Operation, e.Path, e.Cause)
	} else if e.Path != "" {
		return fmt.Sprintf("document error during %s of '%s'", e.Operation, e.Path)
	} else if e.Cause != nil {
		return fmt.Sprintf("document error during %s: %v", e.Operation, e.Cause)
	}
	return fmt.Sprintf("document error during %s", e.Operation)
}

func (e *DocumentError) Unwrap() error {
	return e.Cause
}

// NewDocumentError creates a new document error
func NewDocumentError(operation, path string, cause error) error {
	return &DocumentError{
		Operation: operation,
		Path:      path,
		Cause:     cause,
	}
}

// ValidationIssue represents a single validation problem
type ValidationIssue struct {
	Field   string
	Message string
}

// ValidationError represents multiple validation issues
type ValidationError struct {
	Issues []ValidationIssue
}

func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "validation error"
	}
	
	if len(e.Issues) == 1 {
		return fmt.Sprintf("validation error: %s - %s", e.Issues[0].Field, e.Issues[0].Message)
	}
	
	var parts []string
	parts = append(parts, fmt.Sprintf("%d validation issues:", len(e.Issues)))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("  %s: %s", issue.Field, issue.Message))
	}
	return strings.Join(parts, "\n")
}

// MultiError collects multiple errors
type MultiError struct {
	errors []error
}

// NewMultiError creates a new multi-error collector
func NewMultiError() *MultiError {
	return &MultiError{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collection (ignores nil errors)
func (m *MultiError) Add(err error) {
	if err != nil {
		m.errors = append(m.errors, err)
	}
}

// Len returns the number of errors
func (m *MultiError) Len() int {
	return len(m.errors)
}

// Err returns the multi-error or nil if empty
func (m *MultiError) Err() error {
	if len(m.errors) == 0 {
		return nil
	}
	if len(m.errors) == 1 {
		return m.errors[0]
	}
	return m
}

func (m *MultiError) Error() string {
	if len(m.errors) == 0 {
		return "no errors"
	}
	
	if len(m.errors) == 1 {
		return m.errors[0].Error()
	}
	
	var parts []string
	parts = append(parts, fmt.Sprintf("%d errors occurred:", len(m.errors)))
	for i, err := range m.errors {
		parts = append(parts, fmt.Sprintf("  [%d] %v", i+1, err))
	}
	return strings.Join(parts, "\n")
}

// ContextError adds context to an existing error
type ContextError struct {
	Operation string
	Context   map[string]interface{}
	Cause     error
}

func (e *ContextError) Error() string {
	var contextParts []string
	for k, v := range e.Context {
		contextParts = append(contextParts, fmt.Sprintf("%s=%v", k, v))
	}
	
	if len(contextParts) > 0 {
		return fmt.Sprintf("%s [%s]: %v", e.Operation, strings.Join(contextParts, ", "), e.Cause)
	}
	return fmt.Sprintf("%s: %v", e.Operation, e.Cause)
}

func (e *ContextError) Unwrap() error {
	return e.Cause
}

// WithContext wraps an error with additional context
func WithContext(err error, operation string, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return &ContextError{
		Operation: operation,
		Context:   context,
		Cause:     err,
	}
}

// RecoverError converts a panic recovery value to an error
func RecoverError(r interface{}) error {
	switch v := r.(type) {
	case error:
		return fmt.Errorf("panic recovered: %w", v)
	case string:
		return fmt.Errorf("panic recovered: %s", v)
	default:
		return fmt.Errorf("panic recovered: %v", v)
	}
}

// IsTemplateError checks if an error is a template error
func IsTemplateError(err error) bool {
	_, ok := err.(*TemplateError)
	return ok
}

// IsParseError checks if an error is a parse error
func IsParseError(err error) bool {
	_, ok := err.(*ParseError)
	return ok
}

// IsEvaluationError checks if an error is an evaluation error
func IsEvaluationError(err error) bool {
	_, ok := err.(*EvaluationError)
	return ok
}

// IsFunctionError checks if an error is a function error
func IsFunctionError(err error) bool {
	_, ok := err.(*FunctionError)
	return ok
}

// IsDocumentError checks if an error is a document error
func IsDocumentError(err error) bool {
	_, ok := err.(*DocumentError)
	return ok
}