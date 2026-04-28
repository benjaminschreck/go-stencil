package stencil

import (
	"bytes"
	"strings"
	"testing"
)

func TestTemplateSchemaFlattening(t *testing.T) {
	schema := validationSchemaFromTemplateSchema(TemplateSchema{
		"user": Object(TemplateSchema{
			"name": String,
			"age":  Nullable(Number),
		}),
		"items": List(Object(TemplateSchema{
			"title": String,
			"price": Number,
		})),
		"tags": List(String),
	})

	fields := make(map[string]FieldDefinition)
	for _, field := range schema.Fields {
		fields[field.Path] = field
	}

	assertField := func(path, kind string, collection, nullable bool) {
		t.Helper()
		field, ok := fields[path]
		if !ok {
			t.Fatalf("missing field %q in %#v", path, fields)
		}
		if field.Type != kind || field.Collection != collection || field.Nullable != nullable {
			t.Fatalf("field %s = %#v, want type=%s collection=%v nullable=%v", path, field, kind, collection, nullable)
		}
	}

	assertField("user", semanticKindObject, false, false)
	assertField("user.name", semanticKindString, false, false)
	assertField("user.age", semanticKindNumber, false, true)
	assertField("items", semanticKindObject, true, false)
	assertField("items.title", semanticKindString, false, false)
	assertField("items.price", semanticKindNumber, false, false)
	assertField("tags", semanticKindString, true, false)
}

func TestPreparedTemplateValidateValidatesMainTemplateAndNestedFragments(t *testing.T) {
	tmpl := prepareValidationTemplate(t, `{{include "outer"}}`)
	if err := tmpl.AddFragment("outer", `Hello {{user.name}} {{include "inner"}}`); err != nil {
		t.Fatalf("AddFragment outer failed: %v", err)
	}
	if err := tmpl.AddFragmentFromBytes("inner", buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{for item in items}}{{item.title}}{{end}}</w:t></w:r></w:p>`),
	})); err != nil {
		t.Fatalf("AddFragmentFromBytes inner failed: %v", err)
	}

	result, err := tmpl.Validate(TemplateSchema{
		"user":  Object(TemplateSchema{"name": String}),
		"items": List(Object(TemplateSchema{"title": String})),
	})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid template, got issues: %+v", result.Issues)
	}
	if result.Summary.CheckedTokens != 6 {
		t.Fatalf("checked tokens = %d, want 6", result.Summary.CheckedTokens)
	}
}

func TestPreparedTemplateValidateReportsFragmentUnknownField(t *testing.T) {
	tmpl := prepareValidationTemplate(t, `{{include "frag"}}`)
	if err := tmpl.AddFragment("frag", `{{missing.value}}`); err != nil {
		t.Fatalf("AddFragment failed: %v", err)
	}

	result, err := tmpl.Validate(TemplateSchema{})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid template")
	}
	if !hasValidationIssue(result, IssueCodeUnknownField, "fragment:frag/word/document.xml", "missing.value") {
		t.Fatalf("expected fragment unknown-field issue, got %+v", result.Issues)
	}
}

func TestPreparedTemplateValidateIgnoresDOCXFragmentHeaderFooterTokens(t *testing.T) {
	tmpl := prepareValidationTemplate(t, `{{include "frag"}}`)
	if err := tmpl.AddFragmentFromBytes("frag", buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{user.name}}</w:t></w:r></w:p>`),
		"word/header1.xml":  validationHeaderXML(`<w:p><w:r><w:t>{{missing.header}}</w:t></w:r></w:p>`),
		"word/footer1.xml":  validationFooterXML(`<w:p><w:r><w:t>{{missing.footer}}</w:t></w:r></w:p>`),
	})); err != nil {
		t.Fatalf("AddFragmentFromBytes failed: %v", err)
	}

	result, err := tmpl.Validate(TemplateSchema{
		"user": Object(TemplateSchema{"name": String}),
	})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid template, got issues: %+v", result.Issues)
	}
	if result.Summary.CheckedTokens != 2 {
		t.Fatalf("checked tokens = %d, want 2", result.Summary.CheckedTokens)
	}
}

func TestPreparedTemplateValidateFragmentCanUseCallerLoopScope(t *testing.T) {
	tmpl := prepareValidationTemplate(t, `{{for item in items}}{{include "row"}}{{end}}`)
	if err := tmpl.AddFragment("row", `{{item.title}}`); err != nil {
		t.Fatalf("AddFragment failed: %v", err)
	}

	result, err := tmpl.Validate(TemplateSchema{
		"items": List(Object(TemplateSchema{"title": String})),
	})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid template, got issues: %+v", result.Issues)
	}
}

func TestPreparedTemplateValidateReportsMissingAndCircularFragments(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		tmpl := prepareValidationTemplate(t, `{{include "missing"}}`)
		result, err := tmpl.Validate(nil)
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
		if result.Valid {
			t.Fatalf("expected invalid template")
		}
		if !hasValidationMessage(result, "fragment not found: missing") {
			t.Fatalf("expected missing fragment issue, got %+v", result.Issues)
		}
		if !hasValidationSuggestion(result, "AddFragment") {
			t.Fatalf("expected missing fragment suggestion, got %+v", result.Issues)
		}
	})

	t.Run("circular", func(t *testing.T) {
		tmpl := prepareValidationTemplate(t, `{{include "a"}}`)
		if err := tmpl.AddFragment("a", `{{include "b"}}`); err != nil {
			t.Fatalf("AddFragment a failed: %v", err)
		}
		if err := tmpl.AddFragment("b", `{{include "a"}}`); err != nil {
			t.Fatalf("AddFragment b failed: %v", err)
		}
		result, err := tmpl.Validate(nil)
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
		if result.Valid {
			t.Fatalf("expected invalid template")
		}
		if !hasValidationMessage(result, "circular fragment reference detected: a") {
			t.Fatalf("expected circular fragment issue, got %+v", result.Issues)
		}
		if !hasValidationSuggestion(result, "Remove the include") {
			t.Fatalf("expected circular fragment suggestion, got %+v", result.Issues)
		}
	})
}

func TestPreparedTemplateValidateDynamicIncludeDoesNotTraverseFragments(t *testing.T) {
	tmpl := prepareValidationTemplate(t, `{{include fragmentName}}`)
	if err := tmpl.AddFragment("unused", `{{missing.value}}`); err != nil {
		t.Fatalf("AddFragment failed: %v", err)
	}

	result, err := tmpl.Validate(TemplateSchema{"fragmentName": String})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid dynamic include without fragment traversal, got %+v", result.Issues)
	}
}

func TestPreparedTemplateValidateFunctionRegistry(t *testing.T) {
	tmpl := prepareValidationTemplate(t, `{{custom(user.name)}} {{custom(user.name, user.name)}}`)
	registry := NewFunctionRegistry()
	if err := registry.RegisterFunction(NewSimpleFunction("custom", 1, 1, func(args ...interface{}) (interface{}, error) {
		return "", nil
	})); err != nil {
		t.Fatalf("RegisterFunction failed: %v", err)
	}
	tmpl.registry = registry

	result, err := tmpl.Validate(TemplateSchema{
		"user": Object(TemplateSchema{"name": String}),
	})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid template")
	}
	if !hasValidationIssue(result, IssueCodeFunctionArgError, "word/document.xml", "custom") {
		t.Fatalf("expected function arg issue, got %+v", result.Issues)
	}
}

func TestPreparedTemplateValidateOrdersIssuesByTraversalOrdinal(t *testing.T) {
	tmpl := prepareValidationTemplate(t, `{{missing.main}} {{include "frag"}}`)
	if err := tmpl.AddFragment("frag", `{{missing.fragment}}`); err != nil {
		t.Fatalf("AddFragment failed: %v", err)
	}

	result, err := tmpl.Validate(TemplateSchema{})
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if len(result.Issues) < 2 {
		t.Fatalf("expected at least two issues, got %+v", result.Issues)
	}

	if result.Issues[0].Token.Expression != "missing.main" {
		t.Fatalf("first issue expression = %q, want missing.main; issues: %+v", result.Issues[0].Token.Expression, result.Issues)
	}
	if result.Issues[1].Token.Expression != "missing.fragment" {
		t.Fatalf("second issue expression = %q, want missing.fragment; issues: %+v", result.Issues[1].Token.Expression, result.Issues)
	}
	if result.Issues[0].Location.TokenOrdinal >= result.Issues[1].Location.TokenOrdinal {
		t.Fatalf("issue ordinals are not increasing: %+v", result.Issues)
	}
}

func prepareValidationTemplate(t *testing.T, content string) *PreparedTemplate {
	t.Helper()
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>` + content + `</w:t></w:r></w:p>`),
	})
	tmpl, err := Prepare(bytes.NewReader(docx))
	if err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	t.Cleanup(func() {
		_ = tmpl.Close()
	})
	return tmpl
}

func hasValidationIssue(result ValidateTemplateResult, code StencilIssueCode, partContains, expression string) bool {
	for _, issue := range result.Issues {
		if issue.Code == code && strings.Contains(issue.Location.Part, partContains) && issue.Token.Expression == expression {
			return true
		}
	}
	return false
}

func hasValidationMessage(result ValidateTemplateResult, message string) bool {
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, message) {
			return true
		}
	}
	return false
}

func hasValidationSuggestion(result ValidateTemplateResult, suggestion string) bool {
	for _, issue := range result.Issues {
		for _, candidate := range issue.Suggestions {
			if strings.Contains(candidate, suggestion) {
				return true
			}
		}
	}
	return false
}
