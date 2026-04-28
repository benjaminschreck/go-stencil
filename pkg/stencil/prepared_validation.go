package stencil

import (
	"bytes"
	"fmt"
	"sort"
)

const fragmentPartPrefix = "fragment:"

type templateValidationWalker struct {
	tmpl          *template
	resources     *templateRenderResources
	schema        ValidationSchema
	mainSpans     []tokenSpan
	spans         []tokenSpan
	spanGroups    [][]tokenSpan
	issues        []StencilValidationIssue
	visited       map[string]bool
	fragmentSpans map[string][]tokenSpan
	stack         []string
	nextOrdinal   int
}

// Validate validates a prepared template and all statically included fragments
// against the provided type schema without rendering a DOCX.
func (pt *PreparedTemplate) Validate(schema TemplateSchema) (ValidateTemplateResult, error) {
	if pt == nil {
		return ValidateTemplateResult{}, fmt.Errorf("invalid template")
	}
	pt.mu.RLock()
	if pt.closed || pt.template == nil {
		pt.mu.RUnlock()
		return ValidateTemplateResult{}, fmt.Errorf("template is closed")
	}
	defer pt.mu.RUnlock()
	tmpl := pt.template
	registry := pt.registry

	resources, err := tmpl.ensureRenderResources()
	if err != nil {
		return ValidateTemplateResult{}, fmt.Errorf("failed to prepare template validation resources: %w", err)
	}

	validationSchema := validationSchemaFromTemplateSchema(schema)
	validationSchema.Functions = functionDefinitionsFromRegistry(registry)

	walker := &templateValidationWalker{
		tmpl:          tmpl,
		resources:     resources,
		schema:        validationSchema,
		visited:       make(map[string]bool),
		fragmentSpans: make(map[string][]tokenSpan),
		stack:         make([]string, 0),
	}

	if err := walker.scanMainTemplate(); err != nil {
		return ValidateTemplateResult{}, err
	}
	walker.traverseStaticIncludes(walker.spans)

	syntaxIssues := make([]StencilValidationIssue, 0)
	for _, group := range walker.spanGroups {
		syntaxIssues = append(syntaxIssues, validateTokenSpans(group)...)
	}
	semanticIssues := make([]StencilValidationIssue, 0)
	semanticIssues = append(semanticIssues, walker.validateSemanticSpansWithIncludes()...)
	allIssues := make([]StencilValidationIssue, 0, len(syntaxIssues)+len(semanticIssues)+len(walker.issues))
	allIssues = append(allIssues, syntaxIssues...)
	allIssues = append(allIssues, semanticIssues...)
	allIssues = append(allIssues, walker.issues...)
	sortPreparedValidationIssues(allIssues)
	for i := range allIssues {
		allIssues[i].ID = fmt.Sprintf("iss_%03d", i+1)
	}

	errorCount, warningCount := summarizeIssueSeverities(allIssues)
	return ValidateTemplateResult{
		Valid: errorCount == 0,
		Summary: StencilValidationSummary{
			CheckedTokens:      len(walker.spans),
			ErrorCount:         errorCount,
			WarningCount:       warningCount,
			ReturnedIssueCount: len(allIssues),
		},
		Issues:          allIssues,
		IssuesTruncated: false,
		Metadata:        newValidationMetadata(tmpl.source, ""),
	}, nil
}

func (w *templateValidationWalker) scanMainTemplate() error {
	spans, err := scanDOCXTokenSpans(w.tmpl.source)
	if err != nil {
		return err
	}
	w.addSpans(spans, "")
	w.mainSpans = spans
	return nil
}

func (w *templateValidationWalker) traverseStaticIncludes(spans []tokenSpan) {
	for _, span := range spans {
		name, ok := staticIncludeName(span)
		if !ok {
			continue
		}
		w.traverseFragmentInclude(name, span)
	}
}

func (w *templateValidationWalker) traverseFragmentInclude(name string, includeSpan tokenSpan) {
	for _, stackName := range w.stack {
		if stackName == name {
			w.appendIncludeIssue(
				includeSpan,
				fmt.Sprintf("circular fragment reference detected: %s. This include chain references a fragment that is already being validated.", name),
				[]string{
					fmt.Sprintf("Remove the include of %q from one fragment in the cycle.", name),
					"Move shared content into a separate fragment that does not include its callers.",
				},
			)
			return
		}
	}
	if w.visited[name] {
		return
	}

	frag, err := w.tmpl.resolveFragment(name)
	if err != nil {
		w.appendIncludeIssue(
			includeSpan,
			fmt.Sprintf("failed to resolve fragment %s: %v. Check the configured fragment resolver and ensure it can load this fragment during validation.", name, err),
			[]string{
				fmt.Sprintf("Verify the fragment resolver can resolve %q.", name),
				"Run validation with the same fragment resolver configuration used at render time.",
			},
		)
		return
	}
	if frag == nil {
		w.appendIncludeIssue(
			includeSpan,
			fmt.Sprintf("fragment not found: %s. Add this fragment before validation or update the include expression to reference an existing fragment.", name),
			[]string{
				fmt.Sprintf("Call AddFragment or AddFragmentFromBytes with the name %q before Validate.", name),
				"Check the include name for casing or spelling differences.",
			},
		)
		return
	}

	w.visited[name] = true
	w.stack = append(w.stack, name)
	defer func() {
		w.stack = w.stack[:len(w.stack)-1]
	}()

	fragmentSpans, err := w.scanFragment(name, frag)
	if err != nil {
		w.appendIncludeIssue(
			includeSpan,
			fmt.Sprintf("failed to validate fragment %s: %v. The fragment could not be parsed as validation input.", name, err),
			[]string{
				fmt.Sprintf("Inspect fragment %q for invalid DOCX XML or malformed text content.", name),
				"Re-add the fragment from valid DOCX bytes or plain text before validation.",
			},
		)
		return
	}
	added := w.addSpans(fragmentSpans, fragmentPartPrefix+name+"/")
	w.fragmentSpans[name] = added
	w.traverseStaticIncludes(added)
}

func (w *templateValidationWalker) validateSemanticSpansWithIncludes() []StencilValidationIssue {
	issues := make([]StencilValidationIssue, 0)
	fieldIndex := indexFieldDefinitions(w.schema.Fields)
	functionIndex := indexFunctionDefinitions(w.schema.Functions)
	activeFragments := make(map[string]bool)

	var validateSpans func([]tokenSpan, []map[string]semanticScopedVar) []StencilValidationIssue
	validateSpans = func(spans []tokenSpan, scopeStack []map[string]semanticScopedVar) []StencilValidationIssue {
		groupIssues := make([]StencilValidationIssue, 0)
		validateSemanticTokenSpansWithContext(
			spans,
			fieldIndex,
			functionIndex,
			semanticSeverity(true),
			scopeStack,
			func(includeSpan tokenSpan, includeScope []map[string]semanticScopedVar) []StencilValidationIssue {
				name, ok := staticIncludeName(includeSpan)
				if !ok {
					return nil
				}
				if activeFragments[name] {
					return nil
				}
				fragmentSpans := w.fragmentSpans[name]
				if len(fragmentSpans) == 0 {
					return nil
				}

				activeFragments[name] = true
				defer delete(activeFragments, name)
				return validateSpans(fragmentSpans, includeScope)
			},
			&groupIssues,
		)
		return groupIssues
	}

	issues = append(issues, validateSpans(w.mainSpans, []map[string]semanticScopedVar{{}})...)
	return issues
}

func (w *templateValidationWalker) scanFragment(name string, frag *fragment) ([]tokenSpan, error) {
	if frag == nil {
		return nil, nil
	}
	if frag.isDocx {
		if err := frag.ensurePrepared(w.resources.mainStylesXML); err != nil {
			return nil, err
		}
		if frag.parsed == nil || frag.parsed.Body == nil {
			return nil, nil
		}
		return scanDocumentBodyTokenSpans("word/document.xml", frag.parsed.Body, 0), nil
	}
	if frag.parsed != nil && frag.parsed.Body != nil {
		return scanDocumentBodyTokenSpans("word/document.xml", frag.parsed.Body, 0), nil
	}
	parsed, err := ParseDocument(bytes.NewReader([]byte(wrapInDocumentXML(frag.content))))
	if err != nil {
		return nil, fmt.Errorf("failed to parse text fragment %s: %w", name, err)
	}
	return scanDocumentBodyTokenSpans("word/document.xml", parsed.Body, 0), nil
}

func (w *templateValidationWalker) addSpans(spans []tokenSpan, partPrefix string) []tokenSpan {
	added := make([]tokenSpan, 0, len(spans))
	for _, span := range spans {
		if partPrefix != "" {
			span.Part = partPrefix + span.Part
			span.AnchorID = buildAnchorID(span)
		}
		span.TokenOrdinal = w.nextOrdinal
		w.nextOrdinal++
		added = append(added, span)
		w.spans = append(w.spans, span)
	}
	if len(added) > 0 {
		w.spanGroups = append(w.spanGroups, added)
	}
	return added
}

func (w *templateValidationWalker) appendIncludeIssue(span tokenSpan, message string, suggestions []string) {
	location := locationFromSpan(span)
	w.issues = append(w.issues, StencilValidationIssue{
		Severity:    IssueSeverityError,
		Code:        IssueCodeControlBlockMismatch,
		Message:     message,
		Suggestions: suggestions,
		Token: TemplateTokenRef{
			Raw:        span.Raw,
			Kind:       TokenKindControl,
			Expression: span.Token.Value,
			Location:   location,
		},
		Location: location,
	})
}

func sortPreparedValidationIssues(issues []StencilValidationIssue) {
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
		if left.Severity != right.Severity {
			return left.Severity < right.Severity
		}
		if left.Token.Expression != right.Token.Expression {
			return left.Token.Expression < right.Token.Expression
		}
		return left.Message < right.Message
	})
}

func staticIncludeName(span tokenSpan) (string, bool) {
	if span.Malformed || span.Token.Type != TokenInclude {
		return "", false
	}
	node, err := ParseExpressionStrict(span.Token.Value)
	if err != nil {
		return "", false
	}
	lit, ok := node.(*LiteralNode)
	if !ok {
		return "", false
	}
	name, ok := lit.Value.(string)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}

func functionDefinitionsFromRegistry(registry FunctionRegistry) []FunctionDefinition {
	if registry == nil {
		registry = GetDefaultFunctionRegistry()
	}
	names := registry.ListFunctions()
	sort.Strings(names)
	defs := make([]FunctionDefinition, 0, len(names))
	for _, name := range names {
		fn, ok := registry.GetFunction(name)
		if !ok || fn == nil {
			continue
		}
		defs = append(defs, FunctionDefinition{
			Name:       fn.Name(),
			MinArgs:    fn.MinArgs(),
			MaxArgs:    fn.MaxArgs(),
			ReturnKind: semanticKindAny,
		})
	}
	return defs
}

func scanDocumentBodyTokenSpans(partName string, body *Body, startOrdinal int) []tokenSpan {
	if body == nil {
		return nil
	}
	ordinal := startOrdinal
	paragraphIndex := 0
	spans := make([]tokenSpan, 0)
	var scanElements func([]BodyElement)
	scanElements = func(elements []BodyElement) {
		for _, elem := range elements {
			switch e := elem.(type) {
			case *Paragraph:
				paragraphSpans := scanParagraphTokenSpans(partName, paragraphIndex, e, ordinal)
				spans = append(spans, paragraphSpans...)
				ordinal += len(paragraphSpans)
				paragraphIndex++
			case *Table:
				for rowIdx := range e.Rows {
					for cellIdx := range e.Rows[rowIdx].Cells {
						cell := &e.Rows[rowIdx].Cells[cellIdx]
						for paraIdx := range cell.Paragraphs {
							para := &cell.Paragraphs[paraIdx]
							paragraphSpans := scanParagraphTokenSpans(partName, paragraphIndex, para, ordinal)
							spans = append(spans, paragraphSpans...)
							ordinal += len(paragraphSpans)
							paragraphIndex++
						}
					}
				}
			}
		}
	}
	scanElements(body.Elements)
	return spans
}
