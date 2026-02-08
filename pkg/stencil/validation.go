package stencil

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	validationParserVersion = "v6"
	wordMLNamespace         = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
)

var (
	headerPartPattern   = regexp.MustCompile(`^word/header(\d+)\.xml$`)
	footerPartPattern   = regexp.MustCompile(`^word/footer(\d+)\.xml$`)
	literalIndexPattern = regexp.MustCompile(`\[[^\]]+\]`)
)

// IssueSeverity indicates parser issue severity.
type IssueSeverity string

const (
	IssueSeverityError   IssueSeverity = "error"
	IssueSeverityWarning IssueSeverity = "warning"
)

// StencilIssueCode contains validation issue codes emitted by go-stencil.
type StencilIssueCode string

const (
	IssueCodeSyntaxError          StencilIssueCode = "SYNTAX_ERROR"
	IssueCodeControlBlockMismatch StencilIssueCode = "CONTROL_BLOCK_MISMATCH"
	IssueCodeUnsupportedExpr      StencilIssueCode = "UNSUPPORTED_EXPRESSION"
	IssueCodeUnknownField         StencilIssueCode = "UNKNOWN_FIELD"
	IssueCodeUnknownFunction      StencilIssueCode = "UNKNOWN_FUNCTION"
	IssueCodeFunctionArgError     StencilIssueCode = "FUNCTION_ARGUMENT_ERROR"
	IssueCodeTypeMismatch         StencilIssueCode = "TYPE_MISMATCH"
)

// TokenKind identifies extracted token/reference categories.
type TokenKind string

const (
	TokenKindVariable TokenKind = "variable"
	TokenKindControl  TokenKind = "control"
	TokenKindFunction TokenKind = "function"
)

// ValidateTemplateInput controls full template validation behavior.
type ValidateTemplateInput struct {
	DocxBytes          []byte           `json:"-"`
	TemplateRevisionID string           `json:"templateRevisionId,omitempty"`
	Strict             bool             `json:"strict,omitempty"`
	IncludeWarnings    bool             `json:"includeWarnings,omitempty"`
	MaxIssues          int              `json:"maxIssues,omitempty"` // 0 = unlimited
	Schema             ValidationSchema `json:"schema"`
}

// ValidationSchema contains field/function schema definitions used for semantic validation.
type ValidationSchema struct {
	Fields    []FieldDefinition    `json:"fields"`
	Functions []FunctionDefinition `json:"functions"`
}

// FieldDefinition defines one field path and type.
type FieldDefinition struct {
	Path       string `json:"path"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable,omitempty"`
	Collection bool   `json:"collection,omitempty"`
}

// FunctionDefinition defines one function signature.
type FunctionDefinition struct {
	Name       string     `json:"name"`
	MinArgs    int        `json:"minArgs,omitempty"`
	MaxArgs    int        `json:"maxArgs,omitempty"`
	ArgKinds   [][]string `json:"argKinds,omitempty"`
	ReturnKind string     `json:"returnKind,omitempty"`
}

// ValidateTemplateSyntaxInput controls syntax validation behavior.
type ValidateTemplateSyntaxInput struct {
	DocxBytes          []byte `json:"-"`
	TemplateRevisionID string `json:"templateRevisionId,omitempty"`
	MaxIssues          int    `json:"maxIssues,omitempty"` // 0 = unlimited
}

// ExtractReferencesInput controls reference extraction behavior.
type ExtractReferencesInput struct {
	DocxBytes          []byte `json:"-"`
	TemplateRevisionID string `json:"templateRevisionId,omitempty"`
}

// TemplateLocation identifies a token location in a DOCX part.
type TemplateLocation struct {
	Part           string `json:"part"`
	ParagraphIndex int    `json:"paragraphIndex"`
	RunIndex       int    `json:"runIndex"`
	CharStartUTF16 int    `json:"charStartUtf16"`
	CharEndUTF16   int    `json:"charEndUtf16"`
	TokenOrdinal   int    `json:"tokenOrdinal"`
	AnchorID       string `json:"anchorId,omitempty"`
}

// TemplateTokenRef references one token-derived item.
type TemplateTokenRef struct {
	Raw        string           `json:"raw"`
	Kind       TokenKind        `json:"kind"`
	Expression string           `json:"expression,omitempty"`
	Location   TemplateLocation `json:"location"`
}

// StencilValidationIssue is a validation issue emitted by go-stencil.
type StencilValidationIssue struct {
	ID          string           `json:"id"`
	Severity    IssueSeverity    `json:"severity"`
	Code        StencilIssueCode `json:"code"`
	Message     string           `json:"message"`
	Token       TemplateTokenRef `json:"token"`
	Location    TemplateLocation `json:"location"`
	Suggestions []string         `json:"suggestions,omitempty"`
}

// StencilValidationSummary contains validation counters.
type StencilValidationSummary struct {
	CheckedTokens      int `json:"checkedTokens"`
	ErrorCount         int `json:"errorCount"`
	WarningCount       int `json:"warningCount"`
	ReturnedIssueCount int `json:"returnedIssueCount"`
}

// StencilMetadata identifies parser metadata and request passthrough fields.
type StencilMetadata struct {
	DocumentHash       string `json:"documentHash"`
	TemplateRevisionID string `json:"templateRevisionId,omitempty"`
	ParserVersion      string `json:"parserVersion"`
}

// ValidateTemplateSyntaxResult contains syntax validation output.
type ValidateTemplateSyntaxResult struct {
	Valid           bool                     `json:"valid"`
	Summary         StencilValidationSummary `json:"summary"`
	Issues          []StencilValidationIssue `json:"issues"`
	IssuesTruncated bool                     `json:"issuesTruncated"`
	Metadata        StencilMetadata          `json:"metadata"`
}

// ValidateTemplateResult contains full validation output.
type ValidateTemplateResult struct {
	Valid           bool                     `json:"valid"`
	Summary         StencilValidationSummary `json:"summary"`
	Issues          []StencilValidationIssue `json:"issues"`
	IssuesTruncated bool                     `json:"issuesTruncated"`
	Metadata        StencilMetadata          `json:"metadata"`
}

// ExtractReferencesResult contains parsed references extracted from template tokens.
type ExtractReferencesResult struct {
	References []TemplateTokenRef `json:"references"`
	Metadata   StencilMetadata    `json:"metadata"`
}

// ValidateTemplate validates DOCX template syntax and semantics in a single call.
func ValidateTemplate(input ValidateTemplateInput) (ValidateTemplateResult, error) {
	if len(input.DocxBytes) == 0 {
		return ValidateTemplateResult{}, fmt.Errorf("docx bytes are required")
	}
	if input.MaxIssues < 0 {
		return ValidateTemplateResult{}, fmt.Errorf("maxIssues must be >= 0")
	}

	spans, err := scanDOCXTokenSpans(input.DocxBytes)
	if err != nil {
		return ValidateTemplateResult{}, err
	}

	syntaxIssues := validateTokenSpans(spans)
	semanticIssues := validateSemanticTokenSpans(spans, input.Schema, input.Strict)
	allIssues := make([]StencilValidationIssue, 0, len(syntaxIssues)+len(semanticIssues))
	allIssues = append(allIssues, syntaxIssues...)
	allIssues = append(allIssues, semanticIssues...)

	sortValidationIssues(allIssues)
	for i := range allIssues {
		allIssues[i].ID = fmt.Sprintf("iss_%03d", i+1)
	}

	errorCount, warningCount := summarizeIssueSeverities(allIssues)
	filteredIssues := filterIssues(allIssues, input.IncludeWarnings)
	returnedIssues, issuesTruncated := truncateIssues(filteredIssues, input.MaxIssues)

	return ValidateTemplateResult{
		Valid: errorCount == 0,
		Summary: StencilValidationSummary{
			CheckedTokens:      len(spans),
			ErrorCount:         errorCount,
			WarningCount:       warningCount,
			ReturnedIssueCount: len(returnedIssues),
		},
		Issues:          returnedIssues,
		IssuesTruncated: issuesTruncated,
		Metadata:        newValidationMetadata(input.DocxBytes, input.TemplateRevisionID),
	}, nil
}

// ValidateTemplateSyntax validates DOCX template syntax and control balance.
func ValidateTemplateSyntax(input ValidateTemplateSyntaxInput) (ValidateTemplateSyntaxResult, error) {
	if len(input.DocxBytes) == 0 {
		return ValidateTemplateSyntaxResult{}, fmt.Errorf("docx bytes are required")
	}
	if input.MaxIssues < 0 {
		return ValidateTemplateSyntaxResult{}, fmt.Errorf("maxIssues must be >= 0")
	}

	spans, err := scanDOCXTokenSpans(input.DocxBytes)
	if err != nil {
		return ValidateTemplateSyntaxResult{}, err
	}

	issues := validateTokenSpans(spans)
	sortValidationIssues(issues)
	for i := range issues {
		issues[i].ID = fmt.Sprintf("iss_%03d", i+1)
	}

	returnedIssues := issues
	issuesTruncated := false
	if input.MaxIssues > 0 && len(issues) > input.MaxIssues {
		returnedIssues = issues[:input.MaxIssues]
		issuesTruncated = true
	}

	result := ValidateTemplateSyntaxResult{
		Valid: len(issues) == 0,
		Summary: StencilValidationSummary{
			CheckedTokens:      len(spans),
			ErrorCount:         len(issues),
			WarningCount:       0,
			ReturnedIssueCount: len(returnedIssues),
		},
		Issues:          returnedIssues,
		IssuesTruncated: issuesTruncated,
		Metadata:        newValidationMetadata(input.DocxBytes, input.TemplateRevisionID),
	}

	return result, nil
}

// ExtractReferences extracts variable/function/control references from parsed token ASTs.
func ExtractReferences(input ExtractReferencesInput) (ExtractReferencesResult, error) {
	if len(input.DocxBytes) == 0 {
		return ExtractReferencesResult{}, fmt.Errorf("docx bytes are required")
	}

	spans, err := scanDOCXTokenSpans(input.DocxBytes)
	if err != nil {
		return ExtractReferencesResult{}, err
	}

	references := extractReferencesFromSpans(spans)
	sortTemplateReferences(references)

	return ExtractReferencesResult{
		References: references,
		Metadata:   newValidationMetadata(input.DocxBytes, input.TemplateRevisionID),
	}, nil
}

type tokenSpan struct {
	Part             string
	ParagraphIndex   int
	RunIndex         int
	CharStartUTF16   int
	CharEndUTF16     int
	Raw              string
	Inner            string
	Token            Token
	TokenOrdinal     int
	AnchorID         string
	Malformed        bool
	MalformedMessage string
}

type scanRun struct {
	RunIndex int
	Text     string
}

type scanChar struct {
	Rune           rune
	RunIndex       int
	ParagraphUTF16 int
}

type validationControlFrame struct {
	span    tokenSpan
	sawElse bool
}

type orderedPart struct {
	Name  string
	Index int
}

const (
	semanticKindAny     = "any"
	semanticKindUnknown = "unknown"
	semanticKindNull    = "null"
	semanticKindString  = "string"
	semanticKindNumber  = "number"
	semanticKindBool    = "bool"
	semanticKindObject  = "object"
	semanticKindArray   = "array"
)

type semanticTypeInfo struct {
	Kind        string
	Known       bool
	Nullable    bool
	ElementKind string
}

type semanticScopedVar struct {
	TypeInfo     semanticTypeInfo
	SchemaPrefix string
}

type semanticControlFrame struct {
	TokenType TokenType
	HasScope  bool
}

func scanDOCXTokenSpans(docxBytes []byte) ([]tokenSpan, error) {
	reader := bytes.NewReader(docxBytes)
	docxReader, err := NewDocxReader(reader, int64(len(docxBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX: %w", err)
	}

	parts := validationPartOrder(docxReader.ListParts())
	spans := make([]tokenSpan, 0, 32)
	ordinal := 0

	for _, part := range parts {
		content, err := docxReader.GetPart(part)
		if err != nil {
			return nil, fmt.Errorf("failed to read DOCX part %s: %w", part, err)
		}

		partSpans, err := scanPartTokenSpans(part, content, ordinal)
		if err != nil {
			return nil, fmt.Errorf("failed to scan part %s: %w", part, err)
		}

		spans = append(spans, partSpans...)
		ordinal += len(partSpans)
	}

	return spans, nil
}

func validationPartOrder(partNames []string) []string {
	headers := make([]orderedPart, 0)
	footers := make([]orderedPart, 0)

	for _, partName := range partNames {
		if matches := headerPartPattern.FindStringSubmatch(partName); len(matches) == 2 {
			idx, err := strconv.Atoi(matches[1])
			if err == nil {
				headers = append(headers, orderedPart{Name: partName, Index: idx})
			}
			continue
		}
		if matches := footerPartPattern.FindStringSubmatch(partName); len(matches) == 2 {
			idx, err := strconv.Atoi(matches[1])
			if err == nil {
				footers = append(footers, orderedPart{Name: partName, Index: idx})
			}
		}
	}

	sort.Slice(headers, func(i, j int) bool {
		if headers[i].Index == headers[j].Index {
			return headers[i].Name < headers[j].Name
		}
		return headers[i].Index < headers[j].Index
	})

	sort.Slice(footers, func(i, j int) bool {
		if footers[i].Index == footers[j].Index {
			return footers[i].Name < footers[j].Name
		}
		return footers[i].Index < footers[j].Index
	})

	ordered := []string{"word/document.xml"}
	for _, p := range headers {
		ordered = append(ordered, p.Name)
	}
	for _, p := range footers {
		ordered = append(ordered, p.Name)
	}

	return ordered
}

func scanPartTokenSpans(partName string, content []byte, startOrdinal int) ([]tokenSpan, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	paragraphIndex := 0
	ordinal := startOrdinal
	spans := make([]tokenSpan, 0)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		startElement, ok := token.(xml.StartElement)
		if !ok || !isWordParagraphElement(startElement) {
			continue
		}

		var paragraph Paragraph
		if err := decoder.DecodeElement(&paragraph, &startElement); err != nil {
			return nil, err
		}

		paragraphSpans := scanParagraphTokenSpans(partName, paragraphIndex, &paragraph, ordinal)
		spans = append(spans, paragraphSpans...)
		ordinal += len(paragraphSpans)
		paragraphIndex++
	}

	return spans, nil
}

func isWordParagraphElement(start xml.StartElement) bool {
	if start.Name.Local != "p" {
		return false
	}
	if start.Name.Space == "" {
		return true
	}
	return start.Name.Space == wordMLNamespace
}

func scanParagraphTokenSpans(partName string, paragraphIndex int, paragraph *Paragraph, startOrdinal int) []tokenSpan {
	runs := collectScanRuns(paragraph)
	chars := flattenScanChars(runs)
	if len(chars) == 0 {
		return nil
	}

	spans := make([]tokenSpan, 0)
	inToken := false
	tokenStart := 0
	ordinal := startOrdinal

	for i := 0; i < len(chars); {
		if !inToken {
			if hasPair(chars, i, '{', '{') {
				inToken = true
				tokenStart = i
				i += 2
				continue
			}
			i++
			continue
		}

		if hasPair(chars, i, '}', '}') {
			span := newTokenSpanFromRange(partName, paragraphIndex, chars, tokenStart, i+1, ordinal, false, "")
			spans = append(spans, span)
			ordinal++
			inToken = false
			i += 2
			continue
		}

		i++
	}

	if inToken {
		span := newTokenSpanFromRange(partName, paragraphIndex, chars, tokenStart, len(chars)-1, ordinal, true, "unclosed template token")
		spans = append(spans, span)
	}

	return spans
}

func hasPair(chars []scanChar, idx int, first, second rune) bool {
	return idx+1 < len(chars) && chars[idx].Rune == first && chars[idx+1].Rune == second
}

func collectScanRuns(paragraph *Paragraph) []scanRun {
	runs := make([]scanRun, 0, len(paragraph.Runs)+len(paragraph.Hyperlinks))
	runIndex := 0

	appendRun := func(run *Run) {
		text := ""
		if run != nil && run.Text != nil {
			text = run.Text.Content
		}
		runs = append(runs, scanRun{RunIndex: runIndex, Text: text})
		runIndex++
	}

	if len(paragraph.Content) > 0 {
		for _, content := range paragraph.Content {
			switch c := content.(type) {
			case *Run:
				appendRun(c)
			case *Hyperlink:
				for i := range c.Runs {
					run := c.Runs[i]
					appendRun(&run)
				}
			}
		}
		return runs
	}

	for i := range paragraph.Runs {
		run := paragraph.Runs[i]
		appendRun(&run)
	}

	for i := range paragraph.Hyperlinks {
		hyperlink := paragraph.Hyperlinks[i]
		for j := range hyperlink.Runs {
			run := hyperlink.Runs[j]
			appendRun(&run)
		}
	}

	return runs
}

func flattenScanChars(runs []scanRun) []scanChar {
	chars := make([]scanChar, 0)
	paragraphUTF16 := 0

	for _, run := range runs {
		for _, r := range run.Text {
			chars = append(chars, scanChar{
				Rune:           r,
				RunIndex:       run.RunIndex,
				ParagraphUTF16: paragraphUTF16,
			})

			units := utf16RuneLength(r)
			paragraphUTF16 += units
		}
	}

	return chars
}

func utf16RuneLength(r rune) int {
	if r > 0xFFFF {
		return 2
	}
	return 1
}

func newTokenSpanFromRange(
	partName string,
	paragraphIndex int,
	chars []scanChar,
	startIndex int,
	endIndex int,
	ordinal int,
	malformed bool,
	malformedMessage string,
) tokenSpan {
	start := chars[startIndex]
	end := chars[endIndex]

	runes := make([]rune, 0, endIndex-startIndex+1)
	for i := startIndex; i <= endIndex; i++ {
		runes = append(runes, chars[i].Rune)
	}
	raw := string(runes)

	inner := ""
	token := Token{Type: TokenText}
	if strings.HasPrefix(raw, "{{") && strings.HasSuffix(raw, "}}") && len(raw) >= 4 {
		inner = strings.TrimSpace(raw[2 : len(raw)-2])
		token = parseToken(inner)
	}

	span := tokenSpan{
		Part:             partName,
		ParagraphIndex:   paragraphIndex,
		RunIndex:         start.RunIndex,
		CharStartUTF16:   start.ParagraphUTF16,
		CharEndUTF16:     end.ParagraphUTF16 + utf16RuneLength(end.Rune),
		Raw:              raw,
		Inner:            inner,
		Token:            token,
		TokenOrdinal:     ordinal,
		Malformed:        malformed,
		MalformedMessage: malformedMessage,
	}
	span.AnchorID = buildAnchorID(span)

	return span
}

func buildAnchorID(span tokenSpan) string {
	seed := strings.Join([]string{
		span.Part,
		strconv.Itoa(span.ParagraphIndex),
		strconv.Itoa(span.CharStartUTF16),
		strconv.Itoa(span.CharEndUTF16),
		span.Raw,
	}, "|")

	sum := sha256.Sum256([]byte(seed))
	return "anchor_" + hex.EncodeToString(sum[:8])
}

func validateTokenSpans(spans []tokenSpan) []StencilValidationIssue {
	issues := make([]StencilValidationIssue, 0)
	controlStack := make([]validationControlFrame, 0)

	appendIssue := func(code StencilIssueCode, message string, span tokenSpan, kind TokenKind, expression string) {
		issue := StencilValidationIssue{
			Severity: IssueSeverityError,
			Code:     code,
			Message:  message,
			Token: TemplateTokenRef{
				Raw:        span.Raw,
				Kind:       kind,
				Expression: expression,
				Location:   locationFromSpan(span),
			},
			Location: locationFromSpan(span),
		}
		issues = append(issues, issue)
	}

	for _, span := range spans {
		if span.Malformed {
			appendIssue(IssueCodeSyntaxError, span.MalformedMessage, span, TokenKindControl, "")
			continue
		}

		switch span.Token.Type {
		case TokenText:
			if span.Inner == "" {
				appendIssue(IssueCodeSyntaxError, "empty template token", span, TokenKindControl, "")
			}
		case TokenVariable:
			if _, err := ParseExpressionStrict(span.Token.Value); err != nil {
				appendIssue(IssueCodeUnsupportedExpr, fmt.Sprintf("unsupported expression: %v", err), span, TokenKindVariable, span.Token.Value)
			}
		case TokenIf:
			if _, err := ParseExpressionStrict(span.Token.Value); err != nil {
				appendIssue(IssueCodeUnsupportedExpr, fmt.Sprintf("unsupported if expression: %v", err), span, TokenKindControl, span.Token.Value)
			}
			controlStack = append(controlStack, validationControlFrame{span: span})
		case TokenUnless:
			if _, err := ParseExpressionStrict(span.Token.Value); err != nil {
				appendIssue(IssueCodeUnsupportedExpr, fmt.Sprintf("unsupported unless expression: %v", err), span, TokenKindControl, span.Token.Value)
			}
			controlStack = append(controlStack, validationControlFrame{span: span})
		case TokenFor:
			if _, err := parseForSyntaxWithExpressionParser(span.Token.Value, ParseExpressionStrict); err != nil {
				code := IssueCodeSyntaxError
				if strings.Contains(err.Error(), "collection expression") {
					code = IssueCodeUnsupportedExpr
				}
				appendIssue(code, fmt.Sprintf("invalid for expression: %v", err), span, TokenKindControl, span.Token.Value)
			}
			controlStack = append(controlStack, validationControlFrame{span: span})
		case TokenInclude:
			if _, err := ParseExpressionStrict(span.Token.Value); err != nil {
				appendIssue(IssueCodeUnsupportedExpr, fmt.Sprintf("unsupported include expression: %v", err), span, TokenKindControl, span.Token.Value)
			}
		case TokenElsif:
			if len(controlStack) == 0 || controlStack[len(controlStack)-1].span.Token.Type != TokenIf {
				appendIssue(IssueCodeControlBlockMismatch, "{{elsif}} must be inside an {{if}} block", span, TokenKindControl, span.Token.Value)
			} else if controlStack[len(controlStack)-1].sawElse {
				appendIssue(IssueCodeControlBlockMismatch, "{{elsif}} cannot appear after {{else}} in an {{if}} block", span, TokenKindControl, span.Token.Value)
			}
			if _, err := ParseExpressionStrict(span.Token.Value); err != nil {
				appendIssue(IssueCodeUnsupportedExpr, fmt.Sprintf("unsupported elsif expression: %v", err), span, TokenKindControl, span.Token.Value)
			}
		case TokenElse:
			if len(controlStack) == 0 {
				appendIssue(IssueCodeControlBlockMismatch, "{{else}} has no matching opening control block", span, TokenKindControl, "")
				continue
			}

			top := &controlStack[len(controlStack)-1]
			if top.span.Token.Type != TokenIf && top.span.Token.Type != TokenUnless {
				appendIssue(IssueCodeControlBlockMismatch, "{{else}} only matches {{if}} or {{unless}}", span, TokenKindControl, "")
			} else if top.sawElse {
				appendIssue(IssueCodeControlBlockMismatch, "{{else}} can only appear once in an {{if}} or {{unless}} block", span, TokenKindControl, "")
			} else {
				top.sawElse = true
			}
		case TokenEnd:
			if len(controlStack) == 0 {
				appendIssue(IssueCodeControlBlockMismatch, "{{end}} has no matching opening control block", span, TokenKindControl, "")
				continue
			}
			controlStack = controlStack[:len(controlStack)-1]
		}
	}

	for _, opening := range controlStack {
		appendIssue(
			IssueCodeControlBlockMismatch,
			fmt.Sprintf("missing {{end}} for opening control block %q", opening.span.Raw),
			opening.span,
			TokenKindControl,
			opening.span.Token.Value,
		)
	}

	return issues
}

func validateSemanticTokenSpans(
	spans []tokenSpan,
	schema ValidationSchema,
	strict bool,
) []StencilValidationIssue {
	issues := make([]StencilValidationIssue, 0)
	fieldIndex := indexFieldDefinitions(schema.Fields)
	functionIndex := indexFunctionDefinitions(schema.Functions)
	severity := semanticSeverity(strict)

	scopeStack := []map[string]semanticScopedVar{{}}
	controlStack := make([]semanticControlFrame, 0)

	for _, span := range spans {
		if span.Malformed {
			continue
		}

		switch span.Token.Type {
		case TokenVariable:
			node, err := ParseExpressionStrict(span.Token.Value)
			if err != nil {
				continue
			}
			_ = inferExpressionType(node, span, scopeStack, fieldIndex, functionIndex, severity, &issues)
		case TokenIf:
			node, err := ParseExpressionStrict(span.Token.Value)
			if err == nil {
				_ = inferExpressionType(node, span, scopeStack, fieldIndex, functionIndex, severity, &issues)
			}
			controlStack = append(controlStack, semanticControlFrame{TokenType: TokenIf})
		case TokenUnless:
			node, err := ParseExpressionStrict(span.Token.Value)
			if err == nil {
				_ = inferExpressionType(node, span, scopeStack, fieldIndex, functionIndex, severity, &issues)
			}
			controlStack = append(controlStack, semanticControlFrame{TokenType: TokenUnless})
		case TokenElsif:
			node, err := ParseExpressionStrict(span.Token.Value)
			if err != nil {
				continue
			}
			_ = inferExpressionType(node, span, scopeStack, fieldIndex, functionIndex, severity, &issues)
		case TokenFor:
			pushedScope := false
			forNode, err := parseForSyntaxWithExpressionParser(span.Token.Value, ParseExpressionStrict)
			if err == nil {
				collectionType := inferExpressionType(
					forNode.Collection,
					span,
					scopeStack,
					fieldIndex,
					functionIndex,
					severity,
					&issues,
				)

				localScope := make(map[string]semanticScopedVar)
				localScope[forNode.Variable] = semanticScopedVar{
					TypeInfo:     forLoopVariableType(collectionType),
					SchemaPrefix: forLoopSchemaPrefix(forNode.Collection, scopeStack, fieldIndex),
				}
				if forNode.IndexVar != "" {
					localScope[forNode.IndexVar] = semanticScopedVar{
						TypeInfo: semanticKnownType(semanticKindNumber),
					}
				}
				scopeStack = append(scopeStack, localScope)
				pushedScope = true
			}

			controlStack = append(controlStack, semanticControlFrame{
				TokenType: TokenFor,
				HasScope:  pushedScope,
			})
		case TokenInclude:
			node, err := ParseExpressionStrict(span.Token.Value)
			if err != nil {
				continue
			}
			_ = inferExpressionType(node, span, scopeStack, fieldIndex, functionIndex, severity, &issues)
		case TokenEnd:
			if len(controlStack) == 0 {
				continue
			}

			top := controlStack[len(controlStack)-1]
			controlStack = controlStack[:len(controlStack)-1]

			if top.HasScope && len(scopeStack) > 1 {
				scopeStack = scopeStack[:len(scopeStack)-1]
			}
		}
	}

	return issues
}

func inferExpressionType(
	node ExpressionNode,
	span tokenSpan,
	scopeStack []map[string]semanticScopedVar,
	fieldIndex map[string]FieldDefinition,
	functionIndex map[string]FunctionDefinition,
	severity IssueSeverity,
	issues *[]StencilValidationIssue,
) semanticTypeInfo {
	if node == nil {
		return semanticUnknownType()
	}

	if path, ok := referencePathFromNode(node); ok {
		fieldType, _, found := resolveFieldReference(path, scopeStack, fieldIndex)
		if !found {
			appendValidationIssue(
				issues,
				severity,
				IssueCodeUnknownField,
				fmt.Sprintf("unknown field: %s", path),
				span,
				TokenKindVariable,
				path,
			)
			return semanticUnknownType()
		}
		return fieldType
	}

	switch n := node.(type) {
	case *LiteralNode:
		return semanticTypeFromLiteral(n.Value)
	case *FunctionCallNode:
		argTypes := make([]semanticTypeInfo, 0, len(n.Args))
		for _, arg := range n.Args {
			argTypes = append(argTypes, inferExpressionType(
				arg,
				span,
				scopeStack,
				fieldIndex,
				functionIndex,
				severity,
				issues,
			))
		}

		functionName := normalizeFunctionName(n.Name)
		functionDef, exists := functionIndex[functionName]
		if !exists {
			appendValidationIssue(
				issues,
				severity,
				IssueCodeUnknownFunction,
				fmt.Sprintf("unknown function: %s", n.Name),
				span,
				TokenKindFunction,
				n.Name,
			)
			return semanticUnknownType()
		}

		if len(n.Args) < functionDef.MinArgs || (functionDef.MaxArgs >= 0 && len(n.Args) > functionDef.MaxArgs) {
			appendValidationIssue(
				issues,
				severity,
				IssueCodeFunctionArgError,
				fmt.Sprintf(
					"function %s expects %s arguments, got %d",
					n.Name,
					formatArgumentRange(functionDef.MinArgs, functionDef.MaxArgs),
					len(n.Args),
				),
				span,
				TokenKindFunction,
				n.Name,
			)
		}

		for i, argType := range argTypes {
			if i >= len(functionDef.ArgKinds) {
				continue
			}
			allowedKinds := normalizeAllowedKinds(functionDef.ArgKinds[i])
			if len(allowedKinds) == 0 {
				continue
			}
			if isAllowedArgumentType(argType, allowedKinds) {
				continue
			}

			appendValidationIssue(
				issues,
				severity,
				IssueCodeTypeMismatch,
				fmt.Sprintf(
					"function %s argument %d expects %s, got %s",
					n.Name,
					i+1,
					strings.Join(allowedKinds, "/"),
					semanticTypeName(argType),
				),
				span,
				TokenKindFunction,
				n.Name,
			)
		}

		return semanticTypeFromKind(functionDef.ReturnKind)
	case *BinaryOpNode:
		left := inferExpressionType(n.Left, span, scopeStack, fieldIndex, functionIndex, severity, issues)
		right := inferExpressionType(n.Right, span, scopeStack, fieldIndex, functionIndex, severity, issues)
		return inferBinaryResultType(n.Operator, left, right)
	case *UnaryOpNode:
		_ = inferExpressionType(n.Operand, span, scopeStack, fieldIndex, functionIndex, severity, issues)
		switch n.Operator {
		case "!":
			return semanticKnownType(semanticKindBool)
		case "+", "-":
			return semanticKnownType(semanticKindNumber)
		default:
			return semanticUnknownType()
		}
	case *FieldAccessNode:
		return inferExpressionType(n.Object, span, scopeStack, fieldIndex, functionIndex, severity, issues)
	case *IndexAccessNode:
		objectType := inferExpressionType(n.Object, span, scopeStack, fieldIndex, functionIndex, severity, issues)
		_ = inferExpressionType(n.Index, span, scopeStack, fieldIndex, functionIndex, severity, issues)
		if objectType.Kind == semanticKindArray && objectType.ElementKind != "" {
			return semanticKnownType(objectType.ElementKind)
		}
		if objectType.Kind == semanticKindString {
			return semanticKnownType(semanticKindString)
		}
		return semanticUnknownType()
	default:
		return semanticUnknownType()
	}
}

func appendValidationIssue(
	issues *[]StencilValidationIssue,
	severity IssueSeverity,
	code StencilIssueCode,
	message string,
	span tokenSpan,
	kind TokenKind,
	expression string,
) {
	location := locationFromSpan(span)
	*issues = append(*issues, StencilValidationIssue{
		Severity: severity,
		Code:     code,
		Message:  message,
		Token: TemplateTokenRef{
			Raw:        span.Raw,
			Kind:       kind,
			Expression: expression,
			Location:   location,
		},
		Location: location,
	})
}

func semanticSeverity(strict bool) IssueSeverity {
	if strict {
		return IssueSeverityError
	}
	return IssueSeverityWarning
}

func indexFieldDefinitions(fields []FieldDefinition) map[string]FieldDefinition {
	index := make(map[string]FieldDefinition, len(fields))
	for _, field := range fields {
		path := normalizeFieldPath(field.Path)
		if path == "" {
			continue
		}
		index[path] = field
	}
	return index
}

func indexFunctionDefinitions(functions []FunctionDefinition) map[string]FunctionDefinition {
	index := make(map[string]FunctionDefinition, len(functions))
	for _, function := range functions {
		name := normalizeFunctionName(function.Name)
		if name == "" {
			continue
		}
		index[name] = function
	}
	return index
}

func resolveFieldReference(
	path string,
	scopeStack []map[string]semanticScopedVar,
	fieldIndex map[string]FieldDefinition,
) (semanticTypeInfo, string, bool) {
	normalizedPath := normalizeFieldPath(path)
	if normalizedPath == "" {
		return semanticUnknownType(), "", false
	}

	root, remainder := splitReferencePath(normalizedPath)
	if scopedVar, ok := resolveScopedVariable(root, scopeStack); ok {
		if remainder == "" {
			return scopedVar.TypeInfo, scopedVar.SchemaPrefix, true
		}
		if scopedVar.SchemaPrefix == "" {
			return semanticUnknownType(), "", false
		}

		scopedPath := joinReferencePath(scopedVar.SchemaPrefix, remainder)
		fieldType, resolvedPath, found := lookupFieldType(scopedPath, fieldIndex)
		return fieldType, resolvedPath, found
	}

	return lookupFieldType(normalizedPath, fieldIndex)
}

func lookupFieldType(path string, fieldIndex map[string]FieldDefinition) (semanticTypeInfo, string, bool) {
	candidates := fieldPathCandidates(path)
	for _, candidate := range candidates {
		fieldDef, exists := fieldIndex[candidate]
		if !exists {
			continue
		}
		return semanticTypeFromField(fieldDef), candidate, true
	}
	return semanticUnknownType(), "", false
}

func fieldPathCandidates(path string) []string {
	normalizedPath := normalizeFieldPath(path)
	if normalizedPath == "" {
		return nil
	}

	strippedPath := stripLiteralIndices(normalizedPath)
	if strippedPath == normalizedPath || strippedPath == "" {
		return []string{normalizedPath}
	}

	return []string{normalizedPath, strippedPath}
}

func normalizeFieldPath(path string) string {
	return strings.TrimSpace(path)
}

func normalizeFunctionName(name string) string {
	return strings.TrimSpace(name)
}

func stripLiteralIndices(path string) string {
	cleaned := literalIndexPattern.ReplaceAllString(path, "")
	cleaned = strings.TrimSpace(cleaned)
	for strings.Contains(cleaned, "..") {
		cleaned = strings.ReplaceAll(cleaned, "..", ".")
	}
	cleaned = strings.Trim(cleaned, ".")
	return cleaned
}

func splitReferencePath(path string) (string, string) {
	if path == "" {
		return "", ""
	}

	splitAt := strings.IndexAny(path, ".[")
	if splitAt == -1 {
		return path, ""
	}

	root := path[:splitAt]
	remainder := path[splitAt:]
	remainder = strings.TrimPrefix(remainder, ".")
	return root, remainder
}

func resolveScopedVariable(name string, scopeStack []map[string]semanticScopedVar) (semanticScopedVar, bool) {
	for i := len(scopeStack) - 1; i >= 0; i-- {
		if scopedVar, ok := scopeStack[i][name]; ok {
			return scopedVar, true
		}
	}
	return semanticScopedVar{}, false
}

func joinReferencePath(prefix, remainder string) string {
	normalizedPrefix := normalizeFieldPath(prefix)
	normalizedRemainder := strings.TrimSpace(strings.TrimPrefix(remainder, "."))

	if normalizedPrefix == "" {
		return normalizedRemainder
	}
	if normalizedRemainder == "" {
		return normalizedPrefix
	}
	if strings.HasPrefix(normalizedRemainder, "[") {
		return normalizedPrefix + normalizedRemainder
	}
	return normalizedPrefix + "." + normalizedRemainder
}

func forLoopSchemaPrefix(
	collection ExpressionNode,
	scopeStack []map[string]semanticScopedVar,
	fieldIndex map[string]FieldDefinition,
) string {
	collectionPath, ok := referencePathFromNode(collection)
	if !ok {
		return ""
	}

	_, resolvedPath, found := resolveFieldReference(collectionPath, scopeStack, fieldIndex)
	if !found {
		return stripLiteralIndices(collectionPath)
	}
	return stripLiteralIndices(resolvedPath)
}

func forLoopVariableType(collectionType semanticTypeInfo) semanticTypeInfo {
	switch collectionType.Kind {
	case semanticKindArray:
		if collectionType.ElementKind != "" {
			return semanticKnownType(collectionType.ElementKind)
		}
		return semanticUnknownType()
	case semanticKindString:
		return semanticKnownType(semanticKindString)
	case semanticKindObject:
		return semanticKnownType(semanticKindObject)
	default:
		return semanticUnknownType()
	}
}

func semanticTypeFromField(field FieldDefinition) semanticTypeInfo {
	kind := normalizeSchemaKind(field.Type)
	if kind == "" {
		kind = semanticKindAny
	}

	if field.Collection {
		elementKind := kind
		if elementKind == "" {
			elementKind = semanticKindAny
		}
		return semanticTypeInfo{
			Kind:        semanticKindArray,
			Known:       true,
			Nullable:    field.Nullable,
			ElementKind: elementKind,
		}
	}

	return semanticTypeInfo{
		Kind:     kind,
		Known:    true,
		Nullable: field.Nullable,
	}
}

func semanticTypeFromKind(kind string) semanticTypeInfo {
	normalized := normalizeSchemaKind(kind)
	if normalized == "" {
		normalized = semanticKindAny
	}
	return semanticKnownType(normalized)
}

func semanticTypeFromLiteral(value interface{}) semanticTypeInfo {
	switch value.(type) {
	case nil:
		return semanticKnownType(semanticKindNull)
	case string:
		return semanticKnownType(semanticKindString)
	case bool:
		return semanticKnownType(semanticKindBool)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return semanticKnownType(semanticKindNumber)
	default:
		return semanticUnknownType()
	}
}

func semanticKnownType(kind string) semanticTypeInfo {
	return semanticTypeInfo{
		Kind:  normalizeSchemaKind(kind),
		Known: true,
	}
}

func semanticUnknownType() semanticTypeInfo {
	return semanticTypeInfo{
		Kind:  semanticKindUnknown,
		Known: false,
	}
}

func inferBinaryResultType(operator string, left semanticTypeInfo, right semanticTypeInfo) semanticTypeInfo {
	switch operator {
	case "==", "!=", "<", ">", "<=", ">=", "&", "|":
		return semanticKnownType(semanticKindBool)
	case "+", "-", "*", "/", "%":
		if operator == "+" && (left.Kind == semanticKindString || right.Kind == semanticKindString) {
			return semanticKnownType(semanticKindString)
		}
		return semanticKnownType(semanticKindNumber)
	default:
		return semanticUnknownType()
	}
}

func normalizeSchemaKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", semanticKindAny:
		return semanticKindAny
	case "string", "str", "text":
		return semanticKindString
	case "number", "int", "integer", "float", "decimal":
		return semanticKindNumber
	case "bool", "boolean":
		return semanticKindBool
	case "object", "map":
		return semanticKindObject
	case "array", "list", "slice":
		return semanticKindArray
	case "null", "nil":
		return semanticKindNull
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

func normalizeAllowedKinds(kinds []string) []string {
	normalized := make([]string, 0, len(kinds))
	seen := make(map[string]struct{}, len(kinds))

	for _, kind := range kinds {
		normalizedKind := normalizeSchemaKind(kind)
		if normalizedKind == "" {
			continue
		}
		if _, exists := seen[normalizedKind]; exists {
			continue
		}
		seen[normalizedKind] = struct{}{}
		normalized = append(normalized, normalizedKind)
	}

	return normalized
}

func isAllowedArgumentType(argType semanticTypeInfo, allowedKinds []string) bool {
	if len(allowedKinds) == 0 {
		return true
	}
	if !argType.Known || argType.Kind == semanticKindUnknown || argType.Kind == semanticKindAny {
		return true
	}

	for _, allowedKind := range allowedKinds {
		if allowedKind == semanticKindAny || allowedKind == argType.Kind {
			return true
		}
	}

	return false
}

func semanticTypeName(argType semanticTypeInfo) string {
	if !argType.Known || argType.Kind == "" {
		return semanticKindUnknown
	}
	return argType.Kind
}

func formatArgumentRange(minArgs, maxArgs int) string {
	switch {
	case maxArgs < 0:
		return fmt.Sprintf("at least %d", minArgs)
	case minArgs == maxArgs:
		return fmt.Sprintf("%d", minArgs)
	default:
		return fmt.Sprintf("%d to %d", minArgs, maxArgs)
	}
}

func extractReferencesFromSpans(spans []tokenSpan) []TemplateTokenRef {
	references := make([]TemplateTokenRef, 0)

	appendRef := func(span tokenSpan, kind TokenKind, expression string) {
		ref := TemplateTokenRef{
			Raw:      span.Raw,
			Kind:     kind,
			Location: locationFromSpan(span),
		}
		if expression != "" {
			ref.Expression = expression
		}
		references = append(references, ref)
	}

	for _, span := range spans {
		if span.Malformed {
			continue
		}

		switch span.Token.Type {
		case TokenVariable:
			node, err := ParseExpressionStrict(span.Token.Value)
			if err != nil {
				continue
			}
			collectExpressionReferences(node, func(kind TokenKind, expression string) {
				appendRef(span, kind, expression)
			})
		case TokenIf, TokenUnless, TokenElsif:
			appendRef(span, TokenKindControl, span.Token.Value)
			node, err := ParseExpressionStrict(span.Token.Value)
			if err != nil {
				continue
			}
			collectExpressionReferences(node, func(kind TokenKind, expression string) {
				appendRef(span, kind, expression)
			})
		case TokenFor:
			appendRef(span, TokenKindControl, span.Token.Value)
			forNode, err := parseForSyntaxWithExpressionParser(span.Token.Value, ParseExpressionStrict)
			if err != nil {
				continue
			}
			collectExpressionReferences(forNode.Collection, func(kind TokenKind, expression string) {
				appendRef(span, kind, expression)
			})
		case TokenInclude:
			appendRef(span, TokenKindControl, span.Token.Value)
			node, err := ParseExpressionStrict(span.Token.Value)
			if err != nil {
				continue
			}
			collectExpressionReferences(node, func(kind TokenKind, expression string) {
				appendRef(span, kind, expression)
			})
		}
	}

	return references
}

func collectExpressionReferences(node ExpressionNode, emit func(kind TokenKind, expression string)) {
	if node == nil {
		return
	}

	if path, ok := referencePathFromNode(node); ok {
		emit(TokenKindVariable, path)
		if indexNode, ok := node.(*IndexAccessNode); ok {
			if _, literal := indexNode.Index.(*LiteralNode); !literal {
				collectExpressionReferences(indexNode.Index, emit)
			}
		}
		return
	}

	switch n := node.(type) {
	case *FunctionCallNode:
		emit(TokenKindFunction, n.Name)
		for _, arg := range n.Args {
			collectExpressionReferences(arg, emit)
		}
	case *BinaryOpNode:
		collectExpressionReferences(n.Left, emit)
		collectExpressionReferences(n.Right, emit)
	case *UnaryOpNode:
		collectExpressionReferences(n.Operand, emit)
	case *FieldAccessNode:
		collectExpressionReferences(n.Object, emit)
	case *IndexAccessNode:
		collectExpressionReferences(n.Object, emit)
		collectExpressionReferences(n.Index, emit)
	}
}

func referencePathFromNode(node ExpressionNode) (string, bool) {
	switch n := node.(type) {
	case *VariableNode:
		return n.Name, true
	case *FieldAccessNode:
		base, ok := referencePathFromNode(n.Object)
		if !ok {
			return "", false
		}
		return base + "." + n.Field, true
	case *IndexAccessNode:
		base, ok := referencePathFromNode(n.Object)
		if !ok {
			return "", false
		}

		literal, ok := n.Index.(*LiteralNode)
		if !ok {
			return "", false
		}

		switch v := literal.Value.(type) {
		case int:
			return fmt.Sprintf("%s[%d]", base, v), true
		case float64:
			if v == float64(int(v)) {
				return fmt.Sprintf("%s[%d]", base, int(v)), true
			}
			return fmt.Sprintf("%s[%g]", base, v), true
		case string:
			return fmt.Sprintf("%s[%q]", base, v), true
		default:
			return "", false
		}
	default:
		return "", false
	}
}

func locationFromSpan(span tokenSpan) TemplateLocation {
	return TemplateLocation{
		Part:           span.Part,
		ParagraphIndex: span.ParagraphIndex,
		RunIndex:       span.RunIndex,
		CharStartUTF16: span.CharStartUTF16,
		CharEndUTF16:   span.CharEndUTF16,
		TokenOrdinal:   span.TokenOrdinal,
		AnchorID:       span.AnchorID,
	}
}

func newValidationMetadata(docxBytes []byte, templateRevisionID string) StencilMetadata {
	sum := sha256.Sum256(docxBytes)
	return StencilMetadata{
		DocumentHash:       "sha256:" + hex.EncodeToString(sum[:]),
		TemplateRevisionID: templateRevisionID,
		ParserVersion:      validationParserVersion,
	}
}

func summarizeIssueSeverities(issues []StencilValidationIssue) (int, int) {
	errorCount := 0
	warningCount := 0

	for _, issue := range issues {
		switch issue.Severity {
		case IssueSeverityWarning:
			warningCount++
		default:
			errorCount++
		}
	}

	return errorCount, warningCount
}

func filterIssues(issues []StencilValidationIssue, includeWarnings bool) []StencilValidationIssue {
	if includeWarnings {
		return issues
	}

	filtered := make([]StencilValidationIssue, 0, len(issues))
	for _, issue := range issues {
		if issue.Severity == IssueSeverityWarning {
			continue
		}
		filtered = append(filtered, issue)
	}

	return filtered
}

func truncateIssues(issues []StencilValidationIssue, maxIssues int) ([]StencilValidationIssue, bool) {
	if maxIssues == 0 || len(issues) <= maxIssues {
		return issues, false
	}
	return issues[:maxIssues], true
}

func sortValidationIssues(issues []StencilValidationIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		left := issues[i]
		right := issues[j]

		if left.Location.Part != right.Location.Part {
			return left.Location.Part < right.Location.Part
		}
		if left.Location.TokenOrdinal != right.Location.TokenOrdinal {
			return left.Location.TokenOrdinal < right.Location.TokenOrdinal
		}
		if left.Location.CharStartUTF16 != right.Location.CharStartUTF16 {
			return left.Location.CharStartUTF16 < right.Location.CharStartUTF16
		}
		if left.Code != right.Code {
			return left.Code < right.Code
		}
		if left.Severity != right.Severity {
			return left.Severity < right.Severity
		}
		if left.Token.Expression != right.Token.Expression {
			return left.Token.Expression < right.Token.Expression
		}
		return left.Message < right.Message
	})
}

func sortTemplateReferences(references []TemplateTokenRef) {
	sort.SliceStable(references, func(i, j int) bool {
		left := references[i]
		right := references[j]

		if left.Location.TokenOrdinal != right.Location.TokenOrdinal {
			return left.Location.TokenOrdinal < right.Location.TokenOrdinal
		}
		if left.Location.Part != right.Location.Part {
			return left.Location.Part < right.Location.Part
		}
		if left.Location.CharStartUTF16 != right.Location.CharStartUTF16 {
			return left.Location.CharStartUTF16 < right.Location.CharStartUTF16
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		if left.Expression != right.Expression {
			return left.Expression < right.Expression
		}
		return left.Raw < right.Raw
	})
}
