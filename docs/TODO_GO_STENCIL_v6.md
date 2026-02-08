# TODO: go-stencil Validation Support (v6)

## Status
This v6 plan supersedes v1-v5 stencil planning docs.
Legacy file `docs/TODO_GO_STENCIL.md` remains historical.

## Why v6 (Research Findings)
The current package behavior introduces gaps that v5 did not explicitly close:
- `Tokenize` is text-only and has no positional metadata (`pkg/stencil/tokenizer.go`).
- Rendering merges runs before control detection (`pkg/stencil/render/helpers.go`, `pkg/stencil/render.go`, `pkg/stencil/render_docx.go`), which can destroy original run boundaries needed for stable `runIndex`/UTF-16 locations.
- `ParseExpression` returns parsed AST without a required EOF check (`pkg/stencil/expression.go`), so trailing garbage can be accepted unless guarded.
- Field existence/function registry semantics are intentionally not owned here; variable evaluation returns `nil` for missing paths (`pkg/stencil/eval.go`).

## Goal
Provide reusable, deterministic DOCX syntax/reference primitives for backend/frontend validation flows without regressing existing render behavior.

## Ownership Boundary (unchanged, clarified)
- go-stencil owns DOCX token scanning, syntax parsing, control-structure balance checks, and reference extraction.
- go-stencil owns stable token location metadata and deterministic ordering.
- go-stencil does not own context/schema semantics (`UNKNOWN_FIELD`, `TYPE_MISMATCH`, etc.).
- go-stencil does not own API transport DTOs.

## Release Gates
- G1: Location-aware scanner exists and is deterministic.
- G2: Expression parser is strict (full token consumption required).
- G3: Public validation APIs are available and documented.
- G4: Golden-fixture tests cover split runs, headers/footers, nested control structures, and truncation.

## Determinism and Location Rules (Normative)
- `documentHash` is `sha256:<hex>` over raw input DOCX bytes.
- Part traversal order is deterministic: `word/document.xml`, then `word/header*.xml` (numeric asc), then `word/footer*.xml` (numeric asc).
- `tokenOrdinal` increments in traversal order and is stable for identical DOCX bytes.
- UTF-16 ranges must map to original (pre-merged) text runs.
- `anchorId` must be deterministic from token identity + location (`part`, `paragraphIndex`, UTF-16 range, token raw text).
- Validation scanner must not mutate parsed document trees.

## API Direction
Public APIs remain:
- `ValidateTemplateSyntax(input ValidateTemplateSyntaxInput) (ValidateTemplateSyntaxResult, error)`
- `ExtractReferences(input ExtractReferencesInput) (ExtractReferencesResult, error)`

Behavioral clarifications:
- `maxIssues=0` means unbounded.
- `issuesTruncated=true` only when discovered issues exceed capped returned issues.
- Returned issues always contain `token` and `location`; unmatched-control errors must anchor to the opening token location.

## TODO (Ordered)
- [x] Add internal DOCX scanner that emits token spans from unmerged runs across document/header/footer parts.
- [x] Introduce internal `TokenSpan` model (`part`, `paragraphIndex`, `runIndex`, UTF-16 offsets, raw text, ordinal).
- [x] Implement strict expression parse wrapper that rejects non-EOF remainder.
- [x] Unify validation parser path so control parsing logic is shared (avoid drift between `control.go` and `render_docx.go` flows).
- [x] Implement `ValidateTemplateSyntax` using scanner + strict parser + control-balance validator.
- [x] Implement `ExtractReferences` from parsed AST (variables/functions/control expressions).
- [x] Emit only stencil-owned syntax codes: `SYNTAX_ERROR`, `CONTROL_BLOCK_MISMATCH`, `UNSUPPORTED_EXPRESSION`.
- [x] Return metadata: `documentHash`, `templateRevisionId` passthrough, `parserVersion`.
- [x] Add deterministic ordering tests for token ordinals and references.
- [x] Add split-run tests (`{{na` + `me}}`, split across hyperlinks/runs).
- [x] Add header/footer fixture tests (token extraction + locations).
- [x] Add invalid-expression tests for trailing-token rejection.
- [x] Add truncation tests (`maxIssues`) and `summary.returnedIssueCount` consistency checks.
- [x] Add migration notes for backend/frontend consumers.

## Non-goals
- Backend context/schema validation.
- HTTP status mapping.
- Frontend highlight logic.
