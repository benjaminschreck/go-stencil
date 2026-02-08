# Validation Migration Notes (v6)

This document describes how backend and frontend consumers should migrate to the new go-stencil validation APIs.

## New Public APIs

Use these package-level functions:

- `ValidateTemplate(input ValidateTemplateInput) (ValidateTemplateResult, error)` (preferred integration path)
- `ValidateTemplateSyntax(input ValidateTemplateSyntaxInput) (ValidateTemplateSyntaxResult, error)`
- `ExtractReferences(input ExtractReferencesInput) (ExtractReferencesResult, error)`

All APIs operate on raw DOCX bytes and return deterministic metadata.

## Preferred Integration Path

Backend integrations should call `ValidateTemplate` for end-to-end validation:

- syntax validation
- reference extraction
- semantic schema validation
- strict/warning severity behavior
- warning filtering (`includeWarnings`)
- issue truncation (`maxIssues`)

`ValidateTemplateSyntax` and `ExtractReferences` remain available as low-level building blocks.

## Metadata Contract

- `metadata.documentHash` is `sha256:<hex>` over raw DOCX bytes.
- `metadata.templateRevisionId` is a passthrough value from input.
- `metadata.parserVersion` identifies the scanner/parser generation (`v6`).

Use `documentHash` and `templateRevisionId` together to reject stale validation results.

## Location and Ordering Contract

- Token scanning order is deterministic for identical DOCX bytes.
- Part traversal order is:
  - `word/document.xml`
  - `word/header*.xml` (numeric ascending)
  - `word/footer*.xml` (numeric ascending)
- `location.tokenOrdinal` is stable for identical input bytes.
- `location.anchorId` is deterministic from token identity and location.

Consumers should treat `tokenOrdinal` as the primary stable ordering key.

## Validation Behavior

- `maxIssues=0` means unbounded issue return.
- `issuesTruncated=true` only when post-filter issues exceed returned issues due to `maxIssues`.
- `summary.returnedIssueCount == len(issues)` is always maintained.
- `summary.errorCount` and `summary.warningCount` are pre-filter and pre-truncation counts.
- Unmatched opening control blocks are reported with location anchored to the opening token.
- Every returned issue includes both `token` and `location`.

go-stencil emits these issue codes:

- `SYNTAX_ERROR`
- `CONTROL_BLOCK_MISMATCH`
- `UNSUPPORTED_EXPRESSION`
- `UNKNOWN_FIELD`
- `UNKNOWN_FUNCTION`
- `FUNCTION_ARGUMENT_ERROR`
- `TYPE_MISMATCH`

Semantic validation details:

- Unknown field detection is schema/reference based (not render-time nil behavior).
- `strict=true` classifies semantic issues as `error`.
- `strict=false` classifies semantic issues as `warning`.
- `includeWarnings=false` filters warnings from returned `issues` without changing summary pre-counts.

## Backend Mapping Guidance

Backend should keep only:

- request validation and HTTP status mapping
- context existence/schemaVersion checks
- one call to `ValidateTemplate`
- direct DTO mapping from stencil output to API response

## Frontend Highlight Guidance

- Index highlights by `location.anchorId` and `location.tokenOrdinal`.
- Compare response metadata (`documentHash`, optional `templateRevisionId`) before rendering highlights.
- If `issuesTruncated=true`, display a partial-results indicator and allow re-run with higher `maxIssues`.

## Strict Expression Parsing

Validation now uses strict expression parsing (`ParseExpressionStrict`):

- Full expression token consumption is required.
- Trailing tokens such as `{{name other}}` are reported as `UNSUPPORTED_EXPRESSION`.
- Existing rendering behavior remains unchanged because non-strict `ParseExpression` is still used by render paths.
