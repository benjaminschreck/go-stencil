# Go-Stencil Development Plan

## Project Overview

Go-Stencil is a Go implementation of a template engine for DOCX files. It provides a powerful and flexible way to generate dynamic documents using a simple template syntax with test-driven development approach.

## Template Syntax

This implementation uses `{{}}` syntax for all template features:

- Variable substitution: `{{variable}}`, `{{customer.name}}`, `{{price * 1.2}}`
- Control structures: `{{if condition}}...{{end}}`, `{{for item in items}}...{{end}}`
- Functions: `{{uppercase(name)}}`, `{{format("%.2f", price)}}`
- Special operations: `{{include fragmentName}}`, `{{pageBreak()}}`

## Development Philosophy

- Small, incremental commits
- Write tests before implementing features
- Each commit should pass all tests
- Follow Go best practices and idioms
- Maintain API compatibility where sensible
- Never ask for permission for actions that have already been approved (e.g., if the user says to commit, just commit without asking for confirmation)

## Continuing Development

When starting a new Claude Code session, use the prompt in `CONTINUE_PROMPT.md` to quickly get Claude up to speed with the project context and continue where you left off.

## Development Setup

### Claude Code Hooks

Claude Code hooks are configured in `.claude/settings.json` to automate development workflows.

#### PostToolUse Hooks

**Automatic Testing**

- Triggers after Edit, MultiEdit, or Write operations
- Only runs when `.go` files are modified
- Executes `go test ./... -v`
- Reports test results immediately

#### UserPromptSubmit Hook - Commit Checklist

Provides a pre-commit checklist when commit-related prompts are detected:

- Reminds to run tests
- Suggests documentation updates
- Prompts for godoc comment updates
- Reminds about example updates

#### Stop Hook - Documentation Review

At the end of each Claude Code session:

- Lists recently modified files
- Reminds to review documentation
- Suggests specific files that may need updates

These hooks ensure:

- Tests are run frequently during development
- Documentation stays synchronized with code
- Quality checks happen automatically
- No manual hook setup is required

To manually run tests:

```bash
go test ./... -v
```

### Debugging Guidelines

**Important**: Always use Go programs for debugging and development tasks. Do not use bash scripts or Python scripts.

All debug programs should be placed in the `debug/` folder of the repository. This ensures:

- Consistency with the project's Go-based architecture
- Type safety and better error handling
- Easier maintenance and testing
- Better integration with the existing codebase

Example debug program structure:

```
debug/
├── parse_docx/       # Debug DOCX parsing
│   └── main.go
├── test_tokenizer/   # Test template tokenization
│   └── main.go
├── dump_xml/         # Dump XML structure
│   └── main.go
└── README.md         # Documentation for debug tools
```

When creating a debug program:

1. Create a new folder under `debug/` with a descriptive name
2. Write a `main.go` file with the debug functionality
3. Use the project's packages (`github.com/benjaminschreck/go-stencil/pkg/...`)
4. Add comments explaining what the debug tool does
5. Include example usage in the code
6. Add the debug pragram to `debug/README.md`

Example debug program:

```go
// debug/parse_docx/main.go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/benjaminschreck/go-stencil/pkg/stencil"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Usage: go run main.go <docx-file>")
    }

    // Debug code here...
}
```

## Implementation Roadmap

### Phase 1: Foundation (Commits 1-5)

**Commit 1: Project Setup**

- Initialize Go module for benjaminschreck/go-stencil
- Setup basic project structure
- Create README with project description

**Commit 2: DOCX File Handling**

- Implement basic DOCX zip file reading
- Create structures for document parts
- Test reading document.xml content
- Test reading relationships

**Commit 3: XML Parsing Foundation**

- Setup XML parsing utilities
- Create document model structures
- Test parsing basic paragraph elements
- Test parsing run elements

**Commit 4: Template Token Recognition**

- Implement regex patterns for template syntax
- Create tokenizer for finding template markers
- Test identifying {{variable}} substitution tokens
- Test identifying {{if}}, {{for}}, {{end}} control tokens

**Commit 5: Basic Template Structure**

- Create Template and PreparedTemplate types
- Implement basic Prepare() function
- Implement basic Render() function skeleton
- Test template preparation lifecycle

### Phase 2: Basic Substitution (Commits 6-10)

**Commit 6: Simple Variable Substitution**

- Implement {{variable}} parsing
- Create data context handling
- Test single variable substitution
- Test missing variable behavior

**Commit 7: Nested Field Access**

- Implement dot notation ({{customer.name}})
- Implement bracket notation ({{customer['name']}})
- Test nested object access
- Test array index access

**Commit 8: Expression Parser Foundation**

- Create expression AST structures
- Implement basic expression tokenizer
- Test tokenizing simple expressions
- Test tokenizing complex expressions

**Commit 9: Mathematical Operations**

- Implement +, -, \*, /, % operators
- Implement operator precedence
- Implement parentheses support
- Test arithmetic expressions

**Commit 10: Expression Evaluation**

- Connect parser to evaluator
- Implement expression context
- Test mathematical expression evaluation
- Test combined variable and math expressions

### Phase 3: Control Structures (Commits 11-20)

**Commit 11: AST for Control Structures**

- Create control flow AST nodes
- Parse {{if condition}} and {{end}} markers
- Test AST generation for conditionals
- Test nested structure parsing

**Commit 12: Basic If Statement**

- Implement {{if condition}} evaluation
- Implement truthiness rules
- Test conditional content inclusion
- Test false condition behavior

**Commit 13: If-Else Statement**

- Implement {{else}} clause
- Test if-else branching
- Test multiple else scenarios
- Test nested if-else

**Commit 14: Elsif/Elseif Support**

- Implement {{elsif condition}} variants (also: {{elseif}}, {{elif}})
- Test elsif chain evaluation
- Test complex conditional trees
- Test all syntax variants

**Commit 15: Unless Statement**

- Implement {{unless condition}} as negated if
- Implement unless-else
- Test unless behavior
- Test unless with expressions

**Commit 16: Logical Operators**

- Implement & (AND) operator
- Implement | (OR) operator
- Implement ! (NOT) operator
- Test logical expression evaluation

**Commit 17: For Loop Structure**

- Parse {{for x in list}} syntax
- Create loop context management
- Test basic loop parsing
- Test loop variable scoping

**Commit 18: Basic For Loop**

- Implement collection iteration
- Implement loop variable binding
- Test array iteration
- Test empty collection behavior

**Commit 19: Indexed For Loop**

- Implement {{for idx, x in list}}
- Add index variable support
- Test indexed iteration
- Test nested loops

**Commit 20: Loop Content Generation**

- Implement content duplication for loops
- Handle paragraph/row repetition
- Test content multiplication
- Test nested loop content

### Phase 4: Built-in Functions (Commits 21-35)

**Commit 21: Function Framework**

- Create Function interface
- Implement function registry
- Create function call parser
- Test function call parsing

**Commit 22: Basic Functions**

- Implement empty()
- Implement coalesce()
- Implement list()
- Test function calls in expressions

**Commit 23: Data Access Functions**

- Implement data() function
- Implement map() function
- Test whole data access
- Test collection mapping

**Commit 24: Type Conversion Functions**

- Implement str()
- Implement integer()
- Implement decimal()
- Test type conversions

**Commit 25: String Functions Part 1**

- Implement lowercase()
- Implement uppercase()
- Implement titlecase()
- Test case transformations

**Commit 26: String Functions Part 2**

- Implement join()
- Implement joinAnd()
- Implement replace()
- Test string operations

**Commit 27: String Functions Part 3**

- Implement length()
- Test length on strings
- Test length on arrays
- Test length on maps

**Commit 28: Date Functions**

- Implement date() formatting
- Implement locale support
- Test date parsing
- Test format patterns

**Commit 29: Number Formatting**

- Implement format()
- Implement formatWithLocale()
- Implement currency()
- Implement percent()

**Commit 30: Math Functions**

- Implement round()
- Implement floor()
- Implement ceil()
- Test rounding operations

**Commit 31: Aggregate Functions**

- Implement sum()
- Implement contains()
- Test list operations
- Test edge cases

**Commit 32: Document Functions**

- Implement pageBreak()
- Create OOXML page break
- Test page break insertion
- Test multiple page breaks

**Commit 33: Range Function**

- Implement range() function
- Test numeric ranges
- Test range in loops
- Test range edge cases

**Commit 34: Switch Function**

- Implement switch() function
- Test case matching
- Test default cases
- Test nested switches

**Commit 35: Custom Function Support**

- Implement function registration API
- Create FunctionProvider interface
- Test custom function calls
- Test function override

### Phase 5: Advanced Features (Commits 36-45)

**Commit 36: HTML Function Foundation**

- Create HTML parser
- Map HTML to OOXML
- Test basic tags (b, i, u)
- Test nested HTML

**Commit 37: HTML Advanced Tags**

- Implement span with styles
- Implement sub/sup
- Implement br tags
- Test complex HTML

**Commit 38: XML Function**

- Implement xml() raw insertion
- Create XML validation
- Test XML injection
- Test invalid XML handling

**Commit 39: Table Detection**

- Identify table contexts
- Create table model
- Test table structure parsing
- Test cell identification

**Commit 40: Table Row Operations**

- Implement hideRow()
- Handle row removal
- Test row hiding in loops
- Test conditional rows

**Commit 41: Table Column Operations**

- Implement hideColumn()
- Implement resize strategies
- Test column hiding
- Test different strategies

**Commit 42: Image Handling**

- Parse image relationships
- Create image model
- Test image detection
- Test image properties

**Commit 43: Link Handling**

- Parse hyperlink relationships
- Implement replaceLink()
- Test link replacement
- Test multiple links

**Commit 44: Fragment Support**

- Implement {{include fragmentName}}
- Create fragment loading
- Test fragment inclusion
- Test nested fragments

### Phase 6: Performance & Polish (Commits 45-54)

**Commit 45: Template Caching**

- Implement prepared template cache
- Add cache configuration
- Test cache behavior
- Test memory management

**Commit 46: Resource Management**

- Implement proper cleanup
- Add Close() methods
- Test resource leaks
- Test concurrent access

**Commit 47: Error Handling**

- Create error types
- Improve error messages
- Test error scenarios
- Test error recovery

**Commit 48: Logging Framework**

- Add structured logging
- Create debug mode
- Test log output
- Test performance impact

**Commit 49: Configuration**

- Add environment variables
- Create config structure
- Test configuration loading
- Test defaults

**Commit 50: Benchmarks**

- Create benchmark suite
- Test template preparation
- Test rendering performance
- Profile hot paths

**Commit 51: API Polish**

- Finalize public API
- Add documentation
- Create examples
- Test API usability

**Commit 52: Integration Tests**

- Test real-world templates
- Test large documents
- Test complex scenarios
- Test edge cases

**Commit 53: Documentation**

- Complete API docs
- Add usage examples
- Create getting started guide
- Update README

<!-- **Commit dont implement yet: Standalone Binary**

- Create CLI interface
- Add file watchers
- Test CLI operations
- Test standalone mode -->

## Testing Strategy

- Unit tests for each component
- Integration tests for features
- Benchmark tests for performance
- Example templates for validation

## Dependencies

**IMPORTANT: Minimize external dependencies. Only add new Go packages when absolutely necessary.**

- Use Go standard library for all functionality whenever possible
- Before adding any external dependency, consider:
  - Can this be implemented using the standard library?
  - Is the dependency well-maintained and stable?
  - Does it significantly simplify the implementation?
  - Will it improve performance or security?
- Current approach:
  - ZIP file handling: Use standard `archive/zip`
  - XML parsing: Use standard `encoding/xml`
  - Image processing: Use standard library where possible

## Success Criteria

- Complete feature set for template-based DOCX generation
- High performance
- Comprehensive test coverage (>80%)
- Clear documentation
- Easy-to-use API

## Comprehensive Feature List

### Core Syntax

All template expressions use double curly braces `{{}}` with descriptive keywords:

#### Variable Substitution

- Basic: `{{variable}}`
- Nested: `{{customer.name}}`, `{{items[0].price}}`
- Expressions: `{{price * 1.2}}`, `{{(basePrice + tax) * quantity}}`

#### Control Structures

- Conditionals: `{{if condition}}...{{end}}`, `{{if x > 5}}...{{else}}...{{end}}`
- Extended conditionals: `{{elsif condition}}`, `{{elseif condition}}`, `{{elif condition}}`
- Negated conditionals: `{{unless condition}}...{{end}}`
- Loops: `{{for item in items}}...{{end}}`, `{{for i, item in items}}...{{end}}`

#### Built-in Functions

**Important**: All functions require parentheses `()`, even when called with no arguments.

- String functions: `{{uppercase(name)}}`, `{{lowercase(text)}}`, `{{join(items, ", ")}}`
- Math functions: `{{round(price)}}`, `{{sum(numbers)}}`, `{{floor(value)}}`
- Formatting: `{{format("%.2f", price)}}`, `{{date("YYYY-MM-DD", dateValue)}}`
- Type conversion: `{{str(number)}}`, `{{integer(text)}}`, `{{decimal(value)}}`

#### Document Operations

- Page breaks: `{{pageBreak()}}` (note: requires parentheses)
- HTML content: `{{html("<b>Bold text</b>")}}`
- Raw XML: `{{xml("<w:br/>")}}`
- Table operations: `{{hideRow()}}`, `{{hideColumn()}}`, `{{hideColumn(1, "redistribute")}}`
- Link replacement: `{{replaceLink("https://example.com")}}`
- Fragment inclusion: `{{include "Header Template"}}`

#### Special Functions

- Data access: `{{data()}}` (entire context), `{{map("price", items)}}`
- Conditionals: `{{empty(value)}}`, `{{contains(item, list)}}`
- Utilities: `{{coalesce(value1, value2, default)}}`, `{{range(1, 10)}}`
