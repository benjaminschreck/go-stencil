# Validation Migration Notes (v6)

This document describes how backend and frontend consumers should migrate to the new go-stencil validation APIs.

## New Public APIs

Use these package-level functions:

- `ValidateTemplateSyntax(input ValidateTemplateSyntaxInput) (ValidateTemplateSyntaxResult, error)`
- `ExtractReferences(input ExtractReferencesInput) (ExtractReferencesResult, error)`

Both APIs operate on raw DOCX bytes and return deterministic metadata.

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
- `issuesTruncated=true` only when discovered issues exceed returned issues due to `maxIssues`.
- `summary.returnedIssueCount == len(issues)` is always maintained.
- Unmatched opening control blocks are reported with location anchored to the opening token.

go-stencil only emits syntax-layer codes:

- `SYNTAX_ERROR`
- `CONTROL_BLOCK_MISMATCH`
- `UNSUPPORTED_EXPRESSION`

## Backend Mapping Guidance

Backend semantic validation should remain separate from go-stencil syntax validation.

- Keep schema/context checks in backend (`UNKNOWN_FIELD`, `TYPE_MISMATCH`, `UNKNOWN_FUNCTION`, etc.).
- Use `ExtractReferences` outputs for deterministic field/function discovery inputs.
- Preserve stencil `location` and `token` payloads when mapping syntax issues to API DTOs.

## Frontend Highlight Guidance

- Index highlights by `location.anchorId` and `location.tokenOrdinal`.
- Compare response metadata (`documentHash`, optional `templateRevisionId`) before rendering highlights.
- If `issuesTruncated=true`, display a partial-results indicator and allow re-run with higher `maxIssues`.

## Strict Expression Parsing

Validation now uses strict expression parsing (`ParseExpressionStrict`):

- Full expression token consumption is required.
- Trailing tokens such as `{{name other}}` are reported as `UNSUPPORTED_EXPRESSION`.
- Existing rendering behavior remains unchanged because non-strict `ParseExpression` is still used by render paths.
