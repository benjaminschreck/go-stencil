package stencil

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/render"
)

// ValidationResult contains the results of template validation
type ValidationResult struct {
	Valid    bool              // Whether the template is valid
	Errors   []ValidationError // List of validation errors found
	Warnings []ValidationError // List of warnings (non-fatal issues)

	// Introspection data
	Variables      []string          // Variables referenced in the template
	Functions      []string          // Functions called in the template
	FragmentRefs   []string          // Fragment names referenced via {{include}}
	ControlStructs []ControlStructInfo // Control structures found
}

// ValidationError represents a single validation error or warning
type ValidationError struct {
	Type     ValidationErrorType
	Message  string
	Location string // Optional: paragraph or element location
	Token    string // The problematic token/expression
}

func (e ValidationError) Error() string {
	if e.Location != "" {
		return fmt.Sprintf("%s: %s (at %s)", e.Type, e.Message, e.Location)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ValidationErrorType categorizes validation errors
type ValidationErrorType string

const (
	ValidationErrorSyntax            ValidationErrorType = "syntax"
	ValidationErrorUnbalanced        ValidationErrorType = "unbalanced"
	ValidationErrorExpression        ValidationErrorType = "expression"
	ValidationErrorUnknownFunction   ValidationErrorType = "unknown_function"
	ValidationErrorMissingFragment   ValidationErrorType = "missing_fragment"
	ValidationErrorInvalidFor        ValidationErrorType = "invalid_for"
	ValidationErrorInvalidCondition  ValidationErrorType = "invalid_condition"
	ValidationErrorUnclosedDelimiter ValidationErrorType = "unclosed_delimiter"
)

// ControlStructInfo contains information about a control structure
type ControlStructInfo struct {
	Type       string // "if", "for", "unless", etc.
	Expression string // The condition or loop expression
	Location   string // Location in document
}

// ValidationOptions configures validation behavior
type ValidationOptions struct {
	// CheckFunctions validates that all function calls reference known functions
	CheckFunctions bool
	// CheckFragments validates that all {{include}} references have matching fragments
	CheckFragments bool
	// StrictMode reports warnings as errors
	StrictMode bool
	// Registry is the function registry to check against (uses default if nil)
	Registry FunctionRegistry
}

// DefaultValidationOptions returns sensible default validation options
func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{
		CheckFunctions: true,
		CheckFragments: true,
		StrictMode:     false,
		Registry:       nil,
	}
}

// Validate checks the template for syntax errors without rendering.
// Returns a ValidationResult containing any errors found and introspection data.
func (pt *PreparedTemplate) Validate() *ValidationResult {
	return pt.ValidateWithOptions(DefaultValidationOptions())
}

// ValidateWithOptions checks the template with custom validation options
func (pt *PreparedTemplate) ValidateWithOptions(opts ValidationOptions) *ValidationResult {
	result := &ValidationResult{
		Valid:          true,
		Errors:         make([]ValidationError, 0),
		Warnings:       make([]ValidationError, 0),
		Variables:      make([]string, 0),
		Functions:      make([]string, 0),
		FragmentRefs:   make([]string, 0),
		ControlStructs: make([]ControlStructInfo, 0),
	}

	if pt == nil || pt.template == nil || pt.template.document == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorSyntax,
			Message: "template is nil or not properly prepared",
		})
		return result
	}

	// Get the function registry
	registry := opts.Registry
	if registry == nil {
		registry = GetDefaultFunctionRegistry()
	}

	// Create a validator context
	ctx := &validatorContext{
		opts:         opts,
		registry:     registry,
		fragments:    pt.template.fragments,
		result:       result,
		seenVars:     make(map[string]bool),
		seenFuncs:    make(map[string]bool),
		seenFrags:    make(map[string]bool),
	}

	// Validate document body
	if pt.template.document.Body != nil {
		ctx.validateBody(pt.template.document.Body)
	}

	// Convert maps to slices
	for v := range ctx.seenVars {
		result.Variables = append(result.Variables, v)
	}
	for f := range ctx.seenFuncs {
		result.Functions = append(result.Functions, f)
	}
	for fr := range ctx.seenFrags {
		result.FragmentRefs = append(result.FragmentRefs, fr)
	}

	// In strict mode, treat warnings as errors
	if opts.StrictMode && len(result.Warnings) > 0 {
		result.Errors = append(result.Errors, result.Warnings...)
		result.Warnings = nil
	}

	// Set valid flag based on errors
	result.Valid = len(result.Errors) == 0

	return result
}

// validatorContext holds state during validation
type validatorContext struct {
	opts      ValidationOptions
	registry  FunctionRegistry
	fragments map[string]*fragment
	result    *ValidationResult
	seenVars  map[string]bool
	seenFuncs map[string]bool
	seenFrags map[string]bool

	// Control structure stack for balancing
	controlStack []string
}

func (ctx *validatorContext) addError(errType ValidationErrorType, message, location, token string) {
	ctx.result.Errors = append(ctx.result.Errors, ValidationError{
		Type:     errType,
		Message:  message,
		Location: location,
		Token:    token,
	})
}

func (ctx *validatorContext) addWarning(errType ValidationErrorType, message, location, token string) {
	ctx.result.Warnings = append(ctx.result.Warnings, ValidationError{
		Type:     errType,
		Message:  message,
		Location: location,
		Token:    token,
	})
}

func (ctx *validatorContext) validateBody(body *Body) {
	// First, check for balanced control structures across elements
	ctx.validateControlStructureBalance(body.Elements)

	// Then validate each element
	for i, elem := range body.Elements {
		location := fmt.Sprintf("element %d", i)
		switch el := elem.(type) {
		case *Paragraph:
			ctx.validateParagraph(el, location)
		case *Table:
			ctx.validateTable(el, location)
		}
	}
}

func (ctx *validatorContext) validateControlStructureBalance(elements []BodyElement) {
	// Track control structure depth
	type stackEntry struct {
		ctrlType string
		location string
	}
	var stack []stackEntry

	for i, elem := range elements {
		var text string
		location := fmt.Sprintf("element %d", i)

		switch el := elem.(type) {
		case *Paragraph:
			text = render.GetParagraphText(el)
		case *Table:
			// Check tables recursively
			ctx.validateTableControlBalance(el, location)
			continue
		default:
			continue
		}

		// Detect control structures in text
		controlType, controlContent := detectControlType(text)

		switch controlType {
		case "for", "if", "unless":
			stack = append(stack, stackEntry{ctrlType: controlType, location: location})
			ctx.result.ControlStructs = append(ctx.result.ControlStructs, ControlStructInfo{
				Type:       controlType,
				Expression: controlContent,
				Location:   location,
			})
		case "elsif", "elseif", "elif", "else":
			// These need to be inside an if/unless
			if len(stack) == 0 || (stack[len(stack)-1].ctrlType != "if" && stack[len(stack)-1].ctrlType != "unless") {
				ctx.addError(ValidationErrorUnbalanced,
					fmt.Sprintf("{{%s}} without matching {{if}} or {{unless}}", controlType),
					location, controlType)
			}
		case "end":
			if len(stack) == 0 {
				ctx.addError(ValidationErrorUnbalanced,
					"{{end}} without matching opening control structure",
					location, "end")
			} else {
				stack = stack[:len(stack)-1]
			}
		case "inline-for", "inline-if", "inline-unless":
			// These are self-contained, no balance needed
			ctx.result.ControlStructs = append(ctx.result.ControlStructs, ControlStructInfo{
				Type:       strings.TrimPrefix(controlType, "inline-"),
				Expression: controlContent,
				Location:   location + " (inline)",
			})
		}
	}

	// Check for unclosed control structures
	for _, entry := range stack {
		ctx.addError(ValidationErrorUnbalanced,
			fmt.Sprintf("unclosed {{%s}} - missing {{end}}", entry.ctrlType),
			entry.location, entry.ctrlType)
	}
}

func (ctx *validatorContext) validateTableControlBalance(table *Table, tableLocation string) {
	for rowIdx, row := range table.Rows {
		for cellIdx, cell := range row.Cells {
			cellLocation := fmt.Sprintf("%s, row %d, cell %d", tableLocation, rowIdx, cellIdx)

			var cellElements []BodyElement
			for i := range cell.Paragraphs {
				cellElements = append(cellElements, &cell.Paragraphs[i])
			}

			// Validate balance within cell
			ctx.validateCellControlBalance(cellElements, cellLocation)
		}
	}
}

func (ctx *validatorContext) validateCellControlBalance(elements []BodyElement, location string) {
	var stack []string

	for _, elem := range elements {
		if para, ok := elem.(*Paragraph); ok {
			text := render.GetParagraphText(para)
			controlType, _ := detectControlType(text)

			switch controlType {
			case "for", "if", "unless":
				stack = append(stack, controlType)
			case "end":
				if len(stack) == 0 {
					ctx.addError(ValidationErrorUnbalanced,
						"{{end}} without matching opening control structure",
						location, "end")
				} else {
					stack = stack[:len(stack)-1]
				}
			}
		}
	}

	for _, ctrlType := range stack {
		ctx.addError(ValidationErrorUnbalanced,
			fmt.Sprintf("unclosed {{%s}} in table cell - missing {{end}}", ctrlType),
			location, ctrlType)
	}
}

func (ctx *validatorContext) validateParagraph(para *Paragraph, location string) {
	text := render.GetParagraphText(para)

	// Find all template tokens
	tokens := Tokenize(text)

	for _, token := range tokens {
		switch token.Type {
		case TokenVariable:
			ctx.validateExpression(token.Value, location)

		case TokenIf, TokenUnless, TokenElsif:
			ctx.validateCondition(token.Value, location)

		case TokenFor:
			ctx.validateForLoop(token.Value, location)

		case TokenInclude:
			ctx.validateInclude(token.Value, location)
		}
	}

	// Check for unclosed delimiters
	ctx.checkUnclosedDelimiters(text, location)
}

func (ctx *validatorContext) validateTable(table *Table, location string) {
	for rowIdx, row := range table.Rows {
		for cellIdx, cell := range row.Cells {
			cellLocation := fmt.Sprintf("%s, row %d, cell %d", location, rowIdx, cellIdx)
			for paraIdx := range cell.Paragraphs {
				ctx.validateParagraph(&cell.Paragraphs[paraIdx], cellLocation)
			}
		}
	}
}

func (ctx *validatorContext) validateExpression(expr string, location string) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return
	}

	// Try to parse the expression
	node, err := ParseExpression(expr)
	if err != nil {
		ctx.addError(ValidationErrorExpression,
			fmt.Sprintf("invalid expression: %v", err),
			location, expr)
		return
	}

	// Extract variables and functions from the AST
	ctx.extractFromNode(node, location)
}

func (ctx *validatorContext) validateCondition(condition string, location string) {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		ctx.addError(ValidationErrorInvalidCondition,
			"empty condition",
			location, "")
		return
	}

	// Parse as expression
	node, err := ParseExpression(condition)
	if err != nil {
		ctx.addError(ValidationErrorInvalidCondition,
			fmt.Sprintf("invalid condition expression: %v", err),
			location, condition)
		return
	}

	ctx.extractFromNode(node, location)
}

func (ctx *validatorContext) validateForLoop(forExpr string, location string) {
	forExpr = strings.TrimSpace(forExpr)
	if forExpr == "" {
		ctx.addError(ValidationErrorInvalidFor,
			"empty for loop expression",
			location, "")
		return
	}

	// Try to parse for syntax
	_, err := parseForSyntax(forExpr)
	if err != nil {
		ctx.addError(ValidationErrorInvalidFor,
			fmt.Sprintf("invalid for loop: %v", err),
			location, forExpr)
		return
	}

	// Extract the collection expression for validation
	inIndex := strings.Index(forExpr, " in ")
	if inIndex > 0 {
		collectionExpr := strings.TrimSpace(forExpr[inIndex+4:])
		ctx.validateExpression(collectionExpr, location)
	}
}

func (ctx *validatorContext) validateInclude(includeName string, location string) {
	// Remove quotes if present
	includeName = strings.TrimSpace(includeName)
	includeName = strings.Trim(includeName, `"'`)

	ctx.seenFrags[includeName] = true

	// Check if fragment exists
	if ctx.opts.CheckFragments && ctx.fragments != nil {
		if _, exists := ctx.fragments[includeName]; !exists {
			ctx.addError(ValidationErrorMissingFragment,
				fmt.Sprintf("fragment '%s' not found", includeName),
				location, includeName)
		}
	}
}

func (ctx *validatorContext) extractFromNode(node ExpressionNode, location string) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *VariableNode:
		ctx.seenVars[n.Name] = true

	case *FunctionCallNode:
		ctx.seenFuncs[n.Name] = true

		// Check if function exists
		if ctx.opts.CheckFunctions {
			if _, exists := ctx.registry.GetFunction(n.Name); !exists {
				// Special case: data() is always valid
				if n.Name != "data" {
					ctx.addError(ValidationErrorUnknownFunction,
						fmt.Sprintf("unknown function: %s", n.Name),
						location, n.Name)
				}
			}
		}

		// Recursively check arguments
		for _, arg := range n.Args {
			ctx.extractFromNode(arg, location)
		}

	case *BinaryOpNode:
		ctx.extractFromNode(n.Left, location)
		ctx.extractFromNode(n.Right, location)

	case *UnaryOpNode:
		ctx.extractFromNode(n.Operand, location)

	case *FieldAccessNode:
		ctx.extractFromNode(n.Object, location)

	case *IndexAccessNode:
		ctx.extractFromNode(n.Object, location)
		ctx.extractFromNode(n.Index, location)
	}
}

func (ctx *validatorContext) checkUnclosedDelimiters(text string, location string) {
	// Count opening and closing braces
	openCount := strings.Count(text, "{{")
	closeCount := strings.Count(text, "}}")

	if openCount > closeCount {
		ctx.addError(ValidationErrorUnclosedDelimiter,
			fmt.Sprintf("unclosed template delimiter: %d opening '{{' but only %d closing '}}'",
				openCount, closeCount),
			location, "")
	} else if closeCount > openCount {
		ctx.addWarning(ValidationErrorUnclosedDelimiter,
			fmt.Sprintf("extra closing delimiters: %d closing '}}' but only %d opening '{{'",
				closeCount, openCount),
			location, "")
	}
}

// detectControlType determines the control structure type from text
func detectControlType(text string) (string, string) {
	text = strings.TrimSpace(text)

	// Check for inline structures (both open and close in same text)
	if strings.Contains(text, "{{for ") && strings.Contains(text, "{{end}}") {
		return "inline-for", extractContent(text, "{{for ", "}}")
	}
	if strings.Contains(text, "{{if ") && strings.Contains(text, "{{end}}") {
		return "inline-if", extractContent(text, "{{if ", "}}")
	}
	if strings.Contains(text, "{{unless ") && strings.Contains(text, "{{end}}") {
		return "inline-unless", extractContent(text, "{{unless ", "}}")
	}

	// Check for control structure start
	if strings.Contains(text, "{{for ") {
		return "for", extractContent(text, "{{for ", "}}")
	}
	if strings.Contains(text, "{{if ") {
		return "if", extractContent(text, "{{if ", "}}")
	}
	if strings.Contains(text, "{{unless ") {
		return "unless", extractContent(text, "{{unless ", "}}")
	}

	// Check for control structure modifiers
	if strings.Contains(text, "{{elsif ") {
		return "elsif", extractContent(text, "{{elsif ", "}}")
	}
	if strings.Contains(text, "{{elseif ") {
		return "elseif", extractContent(text, "{{elseif ", "}}")
	}
	if strings.Contains(text, "{{elif ") {
		return "elif", extractContent(text, "{{elif ", "}}")
	}
	if strings.Contains(text, "{{else}}") {
		return "else", ""
	}

	// Check for end
	if strings.Contains(text, "{{end}}") {
		return "end", ""
	}

	return "", ""
}

// extractContent extracts the content between a prefix and suffix
func extractContent(text, prefix, suffix string) string {
	startIdx := strings.Index(text, prefix)
	if startIdx < 0 {
		return ""
	}
	startIdx += len(prefix)

	remaining := text[startIdx:]
	endIdx := strings.Index(remaining, suffix)
	if endIdx < 0 {
		return remaining
	}

	return strings.TrimSpace(remaining[:endIdx])
}

// HasErrors returns true if validation found any errors
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if validation found any warnings
func (r *ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// Error returns a combined error message if there are errors, nil otherwise
func (r *ValidationResult) Error() error {
	if len(r.Errors) == 0 {
		return nil
	}

	var messages []string
	for _, e := range r.Errors {
		messages = append(messages, e.Error())
	}

	return fmt.Errorf("template validation failed with %d error(s):\n  - %s",
		len(r.Errors), strings.Join(messages, "\n  - "))
}

// String returns a human-readable summary of the validation result
func (r *ValidationResult) String() string {
	var sb strings.Builder

	if r.Valid {
		sb.WriteString("Template is valid\n")
	} else {
		sb.WriteString("Template is INVALID\n")
	}

	if len(r.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("\nErrors (%d):\n", len(r.Errors)))
		for _, e := range r.Errors {
			sb.WriteString(fmt.Sprintf("  - %s\n", e.Error()))
		}
	}

	if len(r.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("\nWarnings (%d):\n", len(r.Warnings)))
		for _, e := range r.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", e.Error()))
		}
	}

	if len(r.Variables) > 0 {
		sb.WriteString(fmt.Sprintf("\nVariables referenced: %s\n", strings.Join(r.Variables, ", ")))
	}

	if len(r.Functions) > 0 {
		sb.WriteString(fmt.Sprintf("Functions called: %s\n", strings.Join(r.Functions, ", ")))
	}

	if len(r.FragmentRefs) > 0 {
		sb.WriteString(fmt.Sprintf("Fragments referenced: %s\n", strings.Join(r.FragmentRefs, ", ")))
	}

	if len(r.ControlStructs) > 0 {
		sb.WriteString(fmt.Sprintf("\nControl structures (%d):\n", len(r.ControlStructs)))
		for _, cs := range r.ControlStructs {
			sb.WriteString(fmt.Sprintf("  - %s: %s (at %s)\n", cs.Type, cs.Expression, cs.Location))
		}
	}

	return sb.String()
}

// ValidateExpression validates a single expression string
// This can be used to check expressions outside of template context
func ValidateExpression(expr string) error {
	_, err := ParseExpression(expr)
	return err
}

// ValidateForSyntax validates a for loop expression
func ValidateForSyntax(forExpr string) error {
	_, err := parseForSyntax(forExpr)
	return err
}

// templateTokenRegex matches template tokens for validation
var templateTokenRegex = regexp.MustCompile(`\{\{([^}]*)\}\}`)

// ExtractTemplateTokens extracts all template tokens from text
// Useful for pre-validation of template content
func ExtractTemplateTokens(text string) []string {
	matches := templateTokenRegex.FindAllStringSubmatch(text, -1)
	tokens := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			tokens = append(tokens, strings.TrimSpace(m[1]))
		}
	}
	return tokens
}
