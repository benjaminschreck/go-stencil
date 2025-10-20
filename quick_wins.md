# Quick Wins Refactoring Progress

## Overview
Following Option 2 from REFACTORING_PLAN.md - the recommended quick wins approach.

**Goal**: 70% of benefit with 25% of effort

## Commits Plan (4 commits, 5-7 hours estimated)

### Commit 1: Split xml.go → xml/ package ✅ COMPLETED
**Status**: Completed
**Files created**:
- [x] pkg/stencil/xml/types.go (Core types and interfaces - 42 lines)
- [x] pkg/stencil/xml/document.go (Document and Body structures - 179 lines)
- [x] pkg/stencil/xml/paragraph.go (Paragraph structures - 586 lines)
- [x] pkg/stencil/xml/run.go (Run and text structures - 341 lines)
- [x] pkg/stencil/xml/table.go (Table structures - 562 lines)

**Files updated**:
- [x] pkg/stencil/xml.go (Now 88 lines of re-exports, was 1748 lines)
- [x] pkg/stencil/render.go (Fixed fragment type check for xml.Break)

**Outcome**:
- xml.go reduced from 1,748 lines to 88 lines (95% reduction)
- Code split into 5 logical files totaling 1,710 lines
- All tests passing
- Full backward compatibility maintained

---

### Commit 2: Split render_docx.go → render/ package ✅ COMPLETED
**Status**: Completed (split into sub-commits 2a-2e)
**Achievement**: Extracted all pure helper functions (607 lines) to render/ package
**Outcome**:
- render/ package now has 4 files: helpers.go (187), control.go (208), body.go (78), table.go (134)
- render_docx.go reduced from ~2,270 lines to 1,663 lines (27% reduction)
- All tests passing ✅
- Clean architecture without circular dependencies ✅

---

#### Commit 2a: Extract helper functions (run merging) ✅ COMPLETED
**Status**: Completed
**Files created**:
- [x] pkg/stencil/render/helpers.go (187 lines)
  - MergeConsecutiveRuns() (exported)
  - mergeConsecutiveRunsWithContent()
  - mergeRunSlice()

**Files updated**:
- [x] pkg/stencil/render_docx.go (Removed extracted functions, added import)
- [x] All test files updated to use render.MergeConsecutiveRuns()

**Outcome**:
- render_docx.go reduced from ~2136 to 1956 lines (180 lines extracted)
- 187 lines moved to render/helpers.go
- All tests passing
- No import cycles (uses xml package types directly)

---

#### Commit 2b: Extract control structure functions ✅ COMPLETED
**Status**: Completed
**Files created**:
- [x] pkg/stencil/render/control.go (208 lines)
  - DetectControlStructure() (exported)
  - GetParagraphText() (exported)
  - FindMatchingEnd() (exported)

**Files updated**:
- [x] pkg/stencil/render_docx.go (Replaced implementations with wrappers)

**Outcome**:
- render_docx.go reduced from ~1956 to 1767 lines (189 lines extracted)
- 208 lines in render/control.go
- All tests passing
- Note: Token-dependent functions (renderInlineForLoop, processTemplateText, etc.) remain in render_docx.go to avoid circular dependencies. These will be extracted in a future commit after proper abstraction.

---

#### Commit 2c: Extract body rendering helper functions ✅ DONE
**Status**: Completed
**Files created**:
- [x] pkg/stencil/render/body.go
  - ElseBranch type (exported for use in render_docx.go)
  - FindMatchingEndInElements() - finds matching {{end}} for control structures
  - FindIfStructureInElements() - finds if/elsif/else branch structure

**Note**: Only pure helper functions were extracted to avoid circular dependencies.
The main rendering orchestration functions (renderElementsWithContext, RenderBodyWithControlStructures,
renderBodyWithElementOrder) remain in render_docx.go as they call back into the stencil package.

**Files updated**:
- [x] pkg/stencil/render_docx.go (Added wrappers for extracted functions)

**Expected outcome**: Helper functions extracted, tests passing ✅

---

#### Commit 2d: Extract table rendering helper functions ✅ DONE
**Status**: Completed
**Files created**:
- [x] pkg/stencil/render/table.go
  - DetectTableRowControlStructure() - detects control structures in table rows
  - FindMatchingTableEnd() - finds matching {{end}} for table control structures
  - FindMatchingTableIfEnd() - finds if/elsif/else branches in tables
  - FindMatchingTableEndInSlice() - finds matching {{end}} in row slices
  - FindMatchingTableIfEndInSlice() - finds if/elsif/else branches in row slices

**Note**: Only pure helper functions were extracted to avoid circular dependencies.
The main table rendering functions (RenderTableWithControlStructures, renderTableForLoop,
renderTableIfElse, renderTableUnlessElse) remain in render_docx.go as they call back
into the stencil package.

**Files updated**:
- [x] pkg/stencil/render_docx.go (Added wrappers for extracted functions)

**Expected outcome**: Table helper functions extracted, tests passing ✅

---

#### Commit 2e: Final state - Refactoring complete ✅ DONE
**Status**: Refactoring complete to the extent architecturally sound
**What was accomplished**:
- [x] Extracted all pure helper functions to render/ package
- [x] body.go: Body element control structure matching
- [x] table.go: Table row control structure matching
- [x] control.go: Control structure detection (from 2b)
- [x] helpers.go: Run merging utilities (from 2a)
- [x] All tests passing

**What remains in render_docx.go** (~1300 lines):
- Main orchestration functions that call back into stencil package
- Token processing (renderInlineForLoop, processTemplateText, etc.)
- Rendering functions (RenderBodyWithControlStructures, RenderTableWithControlStructures, etc.)
- These MUST remain to avoid circular import: render → stencil ❌

**Final architecture**:
```
pkg/stencil/
├── xml/                    # Pure XML structures
├── render/                # Pure rendering helpers
│   ├── control.go         # Control structure detection
│   ├── helpers.go         # Run merging
│   ├── body.go           # Body element helpers
│   └── table.go          # Table row helpers
├── render_docx.go         # Rendering orchestration (calls stencil types/functions)
└── ...                    # Expression, tokenizer, etc.
```

**Note**: Cannot reduce render_docx.go further without introducing circular dependencies.
The current architecture is clean and maintainable.

---

#### Commit 2f: Remove wrapper functions and type duplication ✅ COMPLETED
**Status**: Completed - cleanup commit to remove unnecessary abstraction
**Changes made**:
1. **Deleted duplicate type**:
   - Removed `type elseBranch` from render_docx.go
   - Updated all references to use `render.ElseBranch` instead

2. **Removed type conversion boilerplate**:
   - `findIfStructureInElements()`: now returns `render.ElseBranch` directly
   - `findMatchingTableIfEnd()`: now returns `render.ElseBranch` directly
   - `findMatchingTableIfEndInSlice()`: now returns `render.ElseBranch` directly
   - Updated all callers to use `render.ElseBranch` (changed `.index` → `.Index`, etc.)

3. **Removed wrapper functions**:
   - Deleted `detectControlStructure()` wrapper
   - Deleted `getParagraphText()` wrapper
   - Deleted `findMatchingEnd()` wrapper
   - Deleted `detectTableRowControlStructure()` wrapper
   - Deleted `findMatchingEndInElements()` wrapper
   - Deleted `findIfStructureInElements()` wrapper
   - Deleted `findMatchingTableEnd()` wrapper
   - Deleted `findMatchingTableIfEnd()` wrapper
   - Deleted `findMatchingTableEndInSlice()` wrapper
   - Deleted `findMatchingTableIfEndInSlice()` wrapper
   - Updated all call sites to use `render.*` functions directly

4. **Updated test files**:
   - Added `render` import to test files
   - Updated test function calls to use `render.*` functions

**Files updated**:
- [x] pkg/stencil/render_docx.go
  - Removed `elseBranch` type definition
  - Updated all `elseBranch` references to `render.ElseBranch`
  - Removed all wrapper functions
  - Updated all wrapper call sites to direct `render.*` calls
- [x] pkg/stencil/render_docx_test.go (added render import, updated calls)
- [x] pkg/stencil/render_inline_for_typo_test.go (updated calls)
- [x] pkg/stencil/render_inline_for_with_if_test.go (updated calls)
- [x] pkg/stencil/render_table_debug_test.go (added render import, updated calls)

**Outcome**:
- 105 lines removed from render_docx.go (1,663 → 1,558 lines)
- Cleaner, more direct code
- No runtime conversion overhead
- Easier to understand (fewer indirection layers)
- All tests passing ✅

**Note**: Preserved the internal `ifBranch` type for token-level processing as it serves a different purpose than `render.ElseBranch`

---

### Commit 2.1: Alternative Bold Approach (PROPOSAL - NOT IMPLEMENTED)

This section describes an alternative, more aggressive refactoring approach that could have been taken, achieving results similar to Commit 1's 95% reduction. This is documented for future reference but is **NOT** part of the current quick wins plan.

#### Philosophy: Proper Separation Like xml/ Package

The xml/ package extraction (Commit 1) achieved a 95% reduction by moving ALL logic to sub-packages and using re-exports. Commit 2 only achieved 27% because it was overly conservative about circular dependencies.

**Key insight**: render/ CAN import stencil for TYPES without creating circular imports, as long as stencil only imports render for FUNCTIONS. This is standard Go practice.

---

#### Commit 2.1a: Extract body rendering to render/body/ ⏳ PROPOSAL

**Files to create**:
- [ ] pkg/stencil/render/body/render.go (~500 lines)
  - Move `renderBodyWithElementOrder()` - main body rendering logic
  - Move `RenderBodyWithControlStructures()` - entry point
  - Move `renderElementsWithContext()` - recursive rendering
  - Import `stencil` package for `TemplateData`, `renderContext` types
  - Keep all existing helper functions from body.go

**Files to create/update**:
- [ ] pkg/stencil/render_body.go (new, ~10 lines re-exports)
```go
package stencil

import "github.com/benjaminschreck/go-stencil/pkg/stencil/render/body"

// Re-export body rendering functions
var RenderBodyWithControlStructures = body.RenderBodyWithControlStructures
```

- [ ] pkg/stencil/render_docx.go
  - Delete body rendering functions
  - Keep only paragraph/run level rendering

**Expected outcome**: ~500 lines moved to render/body/

---

#### Commit 2.1b: Extract table rendering to render/table/ ⏳ PROPOSAL

**Files to create**:
- [ ] pkg/stencil/render/table/render.go (~450 lines)
  - Move `RenderTableWithControlStructures()` - main entry point
  - Move `renderTableForLoop()` - table loop rendering
  - Move `renderTableIfElse()` - table conditional rendering
  - Move `renderTableUnlessElse()` - table unless rendering
  - Move `RenderTableRow()` - row rendering
  - Move `RenderTableCell()` - cell rendering
  - Keep all existing helper functions from table.go

**Files to create/update**:
- [ ] pkg/stencil/render_table.go (new, ~10 lines re-exports)
```go
package stencil

import "github.com/benjaminschreck/go-stencil/pkg/stencil/render/table"

var RenderTableWithControlStructures = table.RenderTableWithControlStructures
var RenderTableRow = table.RenderTableRow
var RenderTableCell = table.RenderTableCell
```

**Expected outcome**: ~450 lines moved to render/table/

---

#### Commit 2.1c: Extract inline/token processing to render/inline/ ⏳ PROPOSAL

**Files to create**:
- [ ] pkg/stencil/render/inline/process.go (~300 lines)
  - Move `renderInlineForLoop()` - inline loop processing
  - Move `processTemplateText()` - template text processing
  - Move `processTokens()` - token processing
  - Move `processIfStatement()` - inline if processing
  - Move `processUnlessStatement()` - inline unless processing
  - Move `processTokensSimple()` - simple token processing
  - Move `hasCompleteControlStructures()` - structure checking
  - Move `findIfBranches()` - branch finding
  - Move `evaluateCondition()` - condition evaluation
  - Import `stencil` package for `Token`, `TemplateData`, `ParseExpression`, etc.

**Files to create/update**:
- [ ] pkg/stencil/render_inline.go (new, ~5 lines re-exports)
```go
package stencil

import "github.com/benjaminschreck/go-stencil/pkg/stencil/render/inline"

// Internal use only - not re-exported as these are implementation details
```

**Expected outcome**: ~300 lines moved to render/inline/

---

#### Commit 2.1d: Extract paragraph/run rendering to render/paragraph/ ⏳ PROPOSAL

**Files to create**:
- [ ] pkg/stencil/render/paragraph/render.go (~200 lines)
  - Move `RenderParagraphWithContext()` - main paragraph rendering
  - Move any paragraph-specific utilities

**Files to create/update**:
- [ ] pkg/stencil/render_paragraph.go (new, ~5 lines re-exports)
```go
package stencil

import "github.com/benjaminschreck/go-stencil/pkg/stencil/render/paragraph"

var RenderParagraphWithContext = paragraph.RenderParagraphWithContext
```

**Expected outcome**: ~200 lines moved to render/paragraph/

---

#### Commit 2.1e: Convert render_docx.go to pure re-exports ⏳ PROPOSAL

**Files to update**:
- [ ] pkg/stencil/render_docx.go (reduce to ~100 lines)
  - Keep only re-exports and type aliases
  - Remove all implementation code
  - Should look similar to xml.go after Commit 1

**Example final state**:
```go
package stencil

import (
    "github.com/benjaminschreck/go-stencil/pkg/stencil/render/body"
    "github.com/benjaminschreck/go-stencil/pkg/stencil/render/table"
    "github.com/benjaminschreck/go-stencil/pkg/stencil/render/paragraph"
)

// Re-export body rendering
var RenderBodyWithControlStructures = body.RenderBodyWithControlStructures

// Re-export table rendering
var RenderTableWithControlStructures = table.RenderTableWithControlStructures
var RenderTableRow = table.RenderTableRow
var RenderTableCell = table.RenderTableCell

// Re-export paragraph rendering
var RenderParagraphWithContext = paragraph.RenderParagraphWithContext

// ... other re-exports
```

**Expected outcome**: render_docx.go ~100 lines (94% reduction, matching Commit 1)

---

#### Commit 2.1f: Documentation and cleanup ⏳ PROPOSAL

**Final architecture**:
```
pkg/stencil/
├── xml/                      # Pure XML structures (Commit 1)
│   ├── types.go
│   ├── document.go
│   ├── paragraph.go
│   ├── run.go
│   └── table.go
├── render/                   # Rendering logic
│   ├── helpers.go           # Run merging (Commit 2a)
│   ├── control.go           # Control detection (Commit 2b)
│   ├── body/
│   │   └── render.go        # Body rendering (Commit 2.1a)
│   ├── table/
│   │   └── render.go        # Table rendering (Commit 2.1b)
│   ├── inline/
│   │   └── process.go       # Token processing (Commit 2.1c)
│   └── paragraph/
│       └── render.go        # Paragraph rendering (Commit 2.1d)
├── xml.go                    # Re-exports (88 lines)
├── render_docx.go           # Re-exports (100 lines) ← 94% reduction!
├── render_body.go           # Re-exports (10 lines)
├── render_table.go          # Re-exports (10 lines)
├── render_paragraph.go      # Re-exports (5 lines)
└── ... (other files)
```

**Expected outcome**:
- render_docx.go: ~2,270 → ~100 lines (94% reduction)
- Matches Commit 1's approach and success
- Clear, organized structure
- No circular dependencies (render imports stencil for types)
- All tests passing ✅

**Why this works**:
```
render/body/ → imports stencil (for types: TemplateData, renderContext)
stencil      → imports render/body (for functions: RenderBodyWithControlStructures)

This is NOT circular because:
- render imports stencil for TYPE definitions
- stencil imports render for FUNCTION definitions
- Go's import rules allow this!
```

**Comparison**:

| Approach | Lines Moved | render_docx.go Final | Reduction | Like Commit 1? |
|----------|-------------|---------------------|-----------|----------------|
| Current (2a-e) | 607 | 1,663 lines | 27% | ❌ No |
| Proposal (2.1a-f) | ~1,450 | ~100 lines | 94% | ✅ Yes |
| Commit 1 (xml) | ~1,660 | 88 lines | 95% | ✅ Reference |

**Status**: This is a PROPOSAL only. Not implemented. Documented for future consideration.

---

### Commit 3: Move functions to functions/ package ⏳ PENDING
**Status**: Not started
**Files to move/reorganize**:
- [ ] pkg/stencil/functions/*.go → functions/ subpackages
- [ ] Create functions/string/, functions/math/, functions/date/, etc.

**Files to update**:
- [ ] pkg/stencil/functions.go (Add re-exports)
- [ ] Update imports across codebase

**Expected outcome**: Better organization of 30+ function files

---

### Commit 4: Add package documentation ⏳ PENDING
**Status**: Not started
**Files to update**:
- [ ] pkg/stencil/xml/doc.go
- [ ] pkg/stencil/render/doc.go
- [ ] pkg/stencil/functions/doc.go
- [ ] Update pkg/stencil/doc.go

**Expected outcome**: Comprehensive package-level documentation

---

### Commit 5: Complete render_docx.go modularization (Commit 2.1) ⏳ PENDING
**Status**: Not started - scheduled as Phase 2, Commit 5 in REFACTORING_PLAN.md
**Goal**: Complete the render/ package refactoring to match the xml/ package pattern (95% file size reduction)

**Target Architecture:**
```
pkg/stencil/render/
├── body/
│   ├── body.go           # RenderBodyWithControlStructures
│   ├── paragraph.go      # RenderParagraphsWithControlStructures
│   ├── inline.go         # processInlineTokens (renamed from processTokensInRuns)
│   └── loop.go           # loop control structure logic
├── table/
│   ├── table.go          # RenderTableWithControlStructures
│   └── rows.go           # RenderTableRowsWithControlStructures
└── paragraph/
    └── paragraph.go      # RenderParagraph, RenderRun
```

**Final state of render_docx.go (~100 lines):**
- Re-exports all rendering functions
- Maintains backward compatibility
- Matches xml.go pattern (88 lines)

**Rationale:**
- Achieves 94% reduction (1,558 → ~100 lines) matching Commit 1
- Creates domain-specific packages for rendering logic
- Improves long-term maintainability
- **NOT circular**: render/body/ imports stencil for TYPES, stencil imports render/body for FUNCTIONS

**Steps:**
1. Create directory structure: `render/body/`, `render/table/`, `render/paragraph/`
2. Extract body rendering to `render/body/body.go`:
   - RenderBodyWithControlStructures
   - Related types: bodyControlBlock, blockType
3. Extract paragraph processing to `render/body/paragraph.go`:
   - RenderParagraphsWithControlStructures
   - Related types: paragraphControlBlock
4. Extract inline processing to `render/body/inline.go`:
   - processInlineTokens (rename from processTokensInRuns)
   - Helper functions for token processing
5. Extract loop logic to `render/body/loop.go`:
   - Loop control structure rendering
   - Loop context management
6. Extract table rendering to `render/table/table.go`:
   - RenderTableWithControlStructures
   - Related types: tableControlBlock
7. Extract table row rendering to `render/table/rows.go`:
   - RenderTableRowsWithControlStructures
   - Related types: tableRowControlBlock
8. Extract paragraph/run rendering to `render/paragraph/paragraph.go`:
   - RenderParagraph
   - RenderRun
9. Update render_docx.go to re-export all functions (keep ~100 lines like xml.go)
10. Update imports in all dependent files
11. Run tests: `go test ./...`
12. Verify file size: render_docx.go should be ~100 lines (was 1,558 after Commit 2f)
13. Validate no circular dependencies: `go build ./...`

**Expected Impact:**
- render_docx.go: 1,558 → ~100 lines (94% reduction)
- Total lines extracted: ~1,450 lines
- Matches xml/ package pattern (Commit 1: 95% reduction)
- Architectural consistency across codebase

**Time Estimate:** 3-5 hours

---

## Current Status
- **Commits completed**: 7 / 10 (Commit 1 ✅, Commit 2a ✅, Commit 2b ✅, Commit 2c ✅, Commit 2d ✅, Commit 2e ✅, Commit 2f ✅)
- **Estimated time remaining**: 5-8 hours (Commits 3, 4, & 5)
- **Next action**: Continue with Commit 3 - Move functions to functions/ package (optional), or Commit 5 for architectural consistency

**Commit 2 Summary**:
- render_docx.go reduced from ~2,270 lines to 1,558 lines (31% reduction, 712 lines removed)
- Created 4 files in render/ package: helpers.go (187), control.go (208), body.go (78), table.go (134)
- All pure helper functions extracted
- All wrapper functions removed
- All type duplication eliminated
- Clean architecture without circular dependencies
- All tests passing ✅

## Notes
- Keeping re-exports permanently (Option A from REFACTORING_PLAN.md)
- All changes maintain backward compatibility
- Each commit should pass all tests
