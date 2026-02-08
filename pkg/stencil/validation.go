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
	headerPartPattern = regexp.MustCompile(`^word/header(\d+)\.xml$`)
	footerPartPattern = regexp.MustCompile(`^word/footer(\d+)\.xml$`)
)

// IssueSeverity indicates parser issue severity.
type IssueSeverity string

const (
	IssueSeverityError   IssueSeverity = "error"
	IssueSeverityWarning IssueSeverity = "warning"
)

// StencilIssueCode contains syntax-level issue codes emitted by go-stencil.
type StencilIssueCode string

const (
	IssueCodeSyntaxError          StencilIssueCode = "SYNTAX_ERROR"
	IssueCodeControlBlockMismatch StencilIssueCode = "CONTROL_BLOCK_MISMATCH"
	IssueCodeUnsupportedExpr      StencilIssueCode = "UNSUPPORTED_EXPRESSION"
)

// TokenKind identifies extracted token/reference categories.
type TokenKind string

const (
	TokenKindVariable TokenKind = "variable"
	TokenKindControl  TokenKind = "control"
	TokenKindFunction TokenKind = "function"
)

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

// StencilValidationIssue is a syntax issue emitted by go-stencil.
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

// ExtractReferencesResult contains parsed references extracted from template tokens.
type ExtractReferencesResult struct {
	References []TemplateTokenRef `json:"references"`
	Metadata   StencilMetadata    `json:"metadata"`
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

func sortValidationIssues(issues []StencilValidationIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		left := issues[i]
		right := issues[j]

		if left.Location.TokenOrdinal != right.Location.TokenOrdinal {
			return left.Location.TokenOrdinal < right.Location.TokenOrdinal
		}
		if left.Location.Part != right.Location.Part {
			return left.Location.Part < right.Location.Part
		}
		if left.Location.CharStartUTF16 != right.Location.CharStartUTF16 {
			return left.Location.CharStartUTF16 < right.Location.CharStartUTF16
		}
		if left.Code != right.Code {
			return left.Code < right.Code
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
