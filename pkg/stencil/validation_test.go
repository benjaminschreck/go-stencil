package stencil

import (
	"archive/zip"
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestParseExpressionStrictRejectsTrailingTokens(t *testing.T) {
	if _, err := ParseExpression("name other"); err != nil {
		t.Fatalf("ParseExpression should remain non-strict, got error: %v", err)
	}

	if _, err := ParseExpressionStrict("name other"); err == nil {
		t.Fatalf("ParseExpressionStrict expected trailing-token error")
	}

	if _, err := ParseExpressionStrict("name"); err != nil {
		t.Fatalf("ParseExpressionStrict valid expression failed: %v", err)
	}
}

func TestValidateTemplateSyntax_SplitAcrossRunsAndHyperlink(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`
			<w:p>
				<w:r><w:t>{{na</w:t></w:r>
				<w:hyperlink r:id="rId9"><w:r><w:t>me}}</w:t></w:r></w:hyperlink>
			</w:p>
		`),
	})

	result, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid template, got issues: %+v", result.Issues)
	}
	if result.Summary.CheckedTokens != 1 {
		t.Fatalf("checked token count = %d, want 1", result.Summary.CheckedTokens)
	}

	refs, err := ExtractReferences(ExtractReferencesInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ExtractReferences failed: %v", err)
	}
	if len(refs.References) != 1 {
		t.Fatalf("reference count = %d, want 1", len(refs.References))
	}

	ref := refs.References[0]
	if ref.Expression != "name" {
		t.Fatalf("reference expression = %q, want %q", ref.Expression, "name")
	}
	if ref.Location.Part != "word/document.xml" {
		t.Fatalf("part = %q, want word/document.xml", ref.Location.Part)
	}
	if ref.Location.RunIndex != 0 {
		t.Fatalf("runIndex = %d, want 0", ref.Location.RunIndex)
	}
	if ref.Location.TokenOrdinal != 0 {
		t.Fatalf("tokenOrdinal = %d, want 0", ref.Location.TokenOrdinal)
	}
	if ref.Location.CharStartUTF16 != 0 || ref.Location.CharEndUTF16 != 8 {
		t.Fatalf("UTF-16 range = [%d,%d), want [0,8)", ref.Location.CharStartUTF16, ref.Location.CharEndUTF16)
	}
}

func TestValidateTemplateSyntax_HeaderFooterTraversalOrder(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{body}}</w:t></w:r></w:p>`),
		"word/header2.xml":  validationHeaderXML(`<w:p><w:r><w:t>{{h2}}</w:t></w:r></w:p>`),
		"word/header1.xml":  validationHeaderXML(`<w:p><w:r><w:t>{{h1}}</w:t></w:r></w:p>`),
		"word/footer2.xml":  validationFooterXML(`<w:p><w:r><w:t>{{f2}}</w:t></w:r></w:p>`),
		"word/footer1.xml":  validationFooterXML(`<w:p><w:r><w:t>{{f1}}</w:t></w:r></w:p>`),
	})

	refs, err := ExtractReferences(ExtractReferencesInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ExtractReferences failed: %v", err)
	}

	if len(refs.References) != 5 {
		t.Fatalf("reference count = %d, want 5", len(refs.References))
	}

	gotParts := make([]string, 0, len(refs.References))
	gotOrdinals := make([]int, 0, len(refs.References))
	for _, ref := range refs.References {
		gotParts = append(gotParts, ref.Location.Part)
		gotOrdinals = append(gotOrdinals, ref.Location.TokenOrdinal)
	}

	wantParts := []string{
		"word/document.xml",
		"word/header1.xml",
		"word/header2.xml",
		"word/footer1.xml",
		"word/footer2.xml",
	}
	if !reflect.DeepEqual(gotParts, wantParts) {
		t.Fatalf("parts order = %v, want %v", gotParts, wantParts)
	}

	wantOrdinals := []int{0, 1, 2, 3, 4}
	if !reflect.DeepEqual(gotOrdinals, wantOrdinals) {
		t.Fatalf("token ordinals = %v, want %v", gotOrdinals, wantOrdinals)
	}
}

func TestExtractReferences_DeterministicOrdering(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`
			<w:p><w:r><w:t>{{format(customer.id, order.total)}}</w:t></w:r></w:p>
			<w:p><w:r><w:t>{{if user.active}}{{end}}</w:t></w:r></w:p>
		`),
	})

	first, err := ExtractReferences(ExtractReferencesInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("first ExtractReferences failed: %v", err)
	}
	second, err := ExtractReferences(ExtractReferencesInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("second ExtractReferences failed: %v", err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("reference extraction is not deterministic\nfirst=%+v\nsecond=%+v", first, second)
	}

	if len(first.References) == 0 {
		t.Fatalf("expected references")
	}

	lastOrdinal := -1
	for _, ref := range first.References {
		if ref.Location.TokenOrdinal < lastOrdinal {
			t.Fatalf("token ordinals are not sorted: %d after %d", ref.Location.TokenOrdinal, lastOrdinal)
		}
		lastOrdinal = ref.Location.TokenOrdinal
	}
}

func TestValidateTemplateSyntax_StrictTrailingTokenRejection(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{name other}}</w:t></w:r></w:p>`),
	})

	result, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax failed: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid template")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("issues len = %d, want 1", len(result.Issues))
	}
	if result.Issues[0].Code != IssueCodeUnsupportedExpr {
		t.Fatalf("issue code = %s, want %s", result.Issues[0].Code, IssueCodeUnsupportedExpr)
	}
	if !strings.Contains(result.Issues[0].Message, "trailing") {
		t.Fatalf("expected trailing-token message, got %q", result.Issues[0].Message)
	}
}

func TestValidateTemplateSyntax_InvalidForLoopVariableRejected(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{for 1 in items}}{{end}}</w:t></w:r></w:p>`),
	})

	result, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax failed: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid template")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("issues len = %d, want 1", len(result.Issues))
	}
	if result.Issues[0].Code != IssueCodeSyntaxError {
		t.Fatalf("issue code = %s, want %s", result.Issues[0].Code, IssueCodeSyntaxError)
	}
	if !strings.Contains(result.Issues[0].Message, "invalid for loop variable") {
		t.Fatalf("expected invalid for variable message, got %q", result.Issues[0].Message)
	}
}

func TestValidateTemplateSyntax_EmptyForIndexVariableRejected(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{for , item in items}}{{end}}</w:t></w:r></w:p>`),
	})

	result, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax failed: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid template")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("issues len = %d, want 1", len(result.Issues))
	}
	if result.Issues[0].Code != IssueCodeSyntaxError {
		t.Fatalf("issue code = %s, want %s", result.Issues[0].Code, IssueCodeSyntaxError)
	}
	if !strings.Contains(result.Issues[0].Message, "invalid for loop index variable") {
		t.Fatalf("expected invalid for index variable message, got %q", result.Issues[0].Message)
	}
}

func TestValidateTemplateSyntax_ElsifAfterElseRejected(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{if a}}{{else}}{{elseif b}}{{end}}</w:t></w:r></w:p>`),
	})

	result, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax failed: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid template")
	}

	var found bool
	for _, issue := range result.Issues {
		if issue.Code == IssueCodeControlBlockMismatch && strings.Contains(issue.Message, "cannot appear after {{else}}") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected control block mismatch for elseif after else, got: %+v", result.Issues)
	}
}

func TestValidateTemplateSyntax_DuplicateElseRejected(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{if a}}{{else}}{{else}}{{end}}</w:t></w:r></w:p>`),
	})

	result, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax failed: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid template")
	}

	var found bool
	for _, issue := range result.Issues {
		if issue.Code == IssueCodeControlBlockMismatch && strings.Contains(issue.Message, "can only appear once") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected control block mismatch for duplicate else, got: %+v", result.Issues)
	}
}

func TestValidateTemplateSyntax_AllowsLiteralClosingBraces(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>literal braces }} are plain text</w:t></w:r></w:p>`),
	})

	result, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid template, got issues: %+v", result.Issues)
	}
	if result.Summary.CheckedTokens != 0 {
		t.Fatalf("checked token count = %d, want 0", result.Summary.CheckedTokens)
	}
}

func TestValidateTemplateSyntax_UnmatchedControlAnchorsOpeningToken(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{if condition}}</w:t></w:r></w:p>`),
	})

	result, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax failed: %v", err)
	}
	if result.Valid {
		t.Fatalf("expected invalid template")
	}

	if len(result.Issues) != 1 {
		t.Fatalf("issues len = %d, want 1", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Code != IssueCodeControlBlockMismatch {
		t.Fatalf("issue code = %s, want %s", issue.Code, IssueCodeControlBlockMismatch)
	}
	if issue.Token.Raw != "{{if condition}}" {
		t.Fatalf("issue token raw = %q, want %q", issue.Token.Raw, "{{if condition}}")
	}
	if issue.Location.TokenOrdinal != 0 {
		t.Fatalf("issue token ordinal = %d, want 0", issue.Location.TokenOrdinal)
	}
}

func TestValidateTemplateSyntax_MaxIssuesTruncationAndSummaryConsistency(t *testing.T) {
	docx := buildValidationDOCX(t, map[string]string{
		"word/document.xml": validationDocumentXML(`<w:p><w:r><w:t>{{name other}} {{age years}} {{end}}</w:t></w:r></w:p>`),
	})

	truncated, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{
		DocxBytes: docx,
		MaxIssues: 2,
	})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax (truncated) failed: %v", err)
	}

	if !truncated.IssuesTruncated {
		t.Fatalf("expected issuesTruncated=true")
	}
	if truncated.Summary.ErrorCount <= len(truncated.Issues) {
		t.Fatalf("expected discovered errors > returned issues, got errors=%d returned=%d", truncated.Summary.ErrorCount, len(truncated.Issues))
	}
	if truncated.Summary.ReturnedIssueCount != len(truncated.Issues) {
		t.Fatalf("returnedIssueCount=%d, len(issues)=%d", truncated.Summary.ReturnedIssueCount, len(truncated.Issues))
	}

	unbounded, err := ValidateTemplateSyntax(ValidateTemplateSyntaxInput{DocxBytes: docx, MaxIssues: 0})
	if err != nil {
		t.Fatalf("ValidateTemplateSyntax (unbounded) failed: %v", err)
	}
	if unbounded.IssuesTruncated {
		t.Fatalf("expected issuesTruncated=false when maxIssues=0")
	}
	if unbounded.Summary.ReturnedIssueCount != len(unbounded.Issues) {
		t.Fatalf("returnedIssueCount=%d, len(issues)=%d", unbounded.Summary.ReturnedIssueCount, len(unbounded.Issues))
	}
}

func buildValidationDOCX(t *testing.T, partXML map[string]string) []byte {
	t.Helper()

	if _, ok := partXML["word/document.xml"]; !ok {
		t.Fatal("word/document.xml is required")
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	writePart := func(name, content string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	writePart("_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	relEntries := make([]string, 0)
	relID := 1
	for partName := range partXML {
		if strings.HasPrefix(partName, "word/header") && strings.HasSuffix(partName, ".xml") {
			relEntries = append(relEntries, fmt.Sprintf(
				`  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="%s"/>`,
				relID,
				strings.TrimPrefix(partName, "word/"),
			))
			relID++
		}
		if strings.HasPrefix(partName, "word/footer") && strings.HasSuffix(partName, ".xml") {
			relEntries = append(relEntries, fmt.Sprintf(
				`  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="%s"/>`,
				relID,
				strings.TrimPrefix(partName, "word/"),
			))
			relID++
		}
	}
	sort.Strings(relEntries)

	writePart("word/_rels/document.xml.rels", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
%s
</Relationships>`, strings.Join(relEntries, "\n")))

	partNames := make([]string, 0, len(partXML))
	for partName := range partXML {
		partNames = append(partNames, partName)
	}
	sort.Strings(partNames)
	for _, partName := range partNames {
		writePart(partName, partXML[partName])
	}

	overrides := []string{
		`  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`,
	}
	for _, partName := range partNames {
		if partName == "word/document.xml" {
			continue
		}
		switch {
		case strings.HasPrefix(partName, "word/header") && strings.HasSuffix(partName, ".xml"):
			overrides = append(overrides, fmt.Sprintf(`  <Override PartName="/%s" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>`, partName))
		case strings.HasPrefix(partName, "word/footer") && strings.HasSuffix(partName, ".xml"):
			overrides = append(overrides, fmt.Sprintf(`  <Override PartName="/%s" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>`, partName))
		default:
			overrides = append(overrides, fmt.Sprintf(`  <Override PartName="/%s" ContentType="application/xml"/>`, partName))
		}
	}

	writePart("[Content_Types].xml", fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
%s
</Types>`, strings.Join(overrides, "\n")))

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}

	return buf.Bytes()
}

func validationDocumentXML(bodyElements string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>
%s
  </w:body>
</w:document>`, bodyElements)
}

func validationHeaderXML(bodyElements string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
%s
</w:hdr>`, bodyElements)
}

func validationFooterXML(bodyElements string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
%s
</w:ftr>`, bodyElements)
}
