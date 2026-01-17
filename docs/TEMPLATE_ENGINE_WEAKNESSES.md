# Go-Stencil Template Engine: Weakness Analysis Report

This document provides a comprehensive analysis of weaknesses, potential issues, and areas for improvement in the go-stencil template engine.

---

## Table of Contents
1. [Security Concerns](#1-security-concerns)
2. [Error Handling Gaps](#2-error-handling-gaps)
3. [Resource Management Issues](#3-resource-management-issues)
4. [Expression Parser Weaknesses](#4-expression-parser-weaknesses)
5. [Control Structure Limitations](#5-control-structure-limitations)
6. [Function Implementation Issues](#6-function-implementation-issues)
7. [Cache Implementation Concerns](#7-cache-implementation-concerns)
8. [XML/HTML Processing Risks](#8-xmlhtml-processing-risks)
9. [Concurrency Issues](#9-concurrency-issues)
10. [API Design Weaknesses](#10-api-design-weaknesses)
11. [Performance Concerns](#11-performance-concerns)
12. [Testing Gaps](#12-testing-gaps)

---

## 1. Security Concerns

### 1.1 XML Injection via `xml()` Function
**File:** `pkg/stencil/xml_functions.go:142-165`

The `xml()` function allows users to inject raw XML content into documents. While the function validates that the XML is well-formed, it doesn't sanitize or restrict the XML elements that can be injected.

**Risk:** A malicious user could inject OOXML elements that could:
- Create external entity references (XXE attacks if document is processed by other tools)
- Insert macros or scripts (VBA through OOXML extensions)
- Create links to external resources

**Recommendation:** Consider implementing an allowlist of safe OOXML elements or at least document the security implications.

### 1.2 HTML Processing with Limited Sanitization
**File:** `pkg/stencil/html_functions.go:37-48`

The `html()` function has a limited allowlist of tags (`legalTags`), which is good. However:
- Attributes are parsed but only `style` is mentioned as potentially used
- No validation of attribute values which could contain malicious content in edge cases

```go
var legalTags = map[string]bool{
    "b": true, "em": true, "i": true, "u": true,
    "s": true, "sup": true, "sub": true, "span": true,
    "br": true, "strong": true,
}
```

### 1.3 Fragment Path Handling
**File:** `pkg/stencil/stencil.go:879-988`

The `AddFragmentFromBytes` function processes external DOCX files. While it validates the DOCX structure, media files from fragments are extracted without path validation:

```go
// Line 920-934: Media file extraction
if strings.HasPrefix(file.Name, "word/media/") {
    relativePath := strings.TrimPrefix(file.Name, "word/")
    mediaFiles[relativePath] = content
}
```

**Risk:** Path traversal in media filenames could potentially be exploited, though the impact is limited since files are written to `word/media/` prefix.

### 1.4 No Rate Limiting on Range Function
**File:** `pkg/stencil/functions.go:929-975`

The `range()` function can create large arrays without any limits:

```go
func rangeNumbers(start, end, step int) ([]interface{}, error) {
    var result []interface{}
    if step > 0 {
        for i := start; i < end; i += step {
            result = append(result, i)
        }
    }
    return result, nil
}
```

**Risk:** A template with `{{for i in range(0, 1000000000)}}` could cause memory exhaustion.

**Recommendation:** Add a maximum range size limit (e.g., `MaxRangeSize` in Config).

---

## 2. Error Handling Gaps

### 2.1 Silent Failures in Variable Access
**File:** `pkg/stencil/eval.go:52-56`

Missing variables return `nil` without error:

```go
// If we hit nil at any point, return nil
if current == nil {
    return nil, nil  // No error, just nil
}
```

This makes debugging templates difficult as typos in variable names silently produce empty output.

**Recommendation:** Add a strict mode that reports missing variables as errors.

### 2.2 Inconsistent Error Propagation in Control Structures
**File:** `pkg/stencil/render_docx.go:66-75`

When for loop parsing fails, the error context lacks the loop expression:

```go
forNode, err := parseForSyntax(controlContent)
if err != nil {
    return nil, fmt.Errorf("invalid for syntax: %w", err)
}
```

The `controlContent` should be included in the error message.

### 2.3 Panic Recovery in matchesCase
**File:** `pkg/stencil/functions.go:1044-1070`

The `matchesCase` function uses `defer/recover` to handle comparison of uncomparable types:

```go
defer func() {
    if r := recover(); r != nil {
        result = false
    }
}()
return expr == caseValue
```

While functional, this is a code smell. Panics should be avoided through proper type checking rather than caught.

### 2.4 Missing Error Types for Some Failures
**File:** `pkg/stencil/errors.go`

The error types are well-designed but not consistently used. Many errors are created with `fmt.Errorf` instead of the custom error types, making it harder for users to programmatically handle specific error types.

---

## 3. Resource Management Issues

### 3.1 Template Close Doesn't Fully Clean Up
**File:** `pkg/stencil/stencil.go:1037-1055`

```go
func (t *template) Close() error {
    t.mu.Lock()
    defer t.mu.Unlock()

    if t.closed {
        return nil
    }

    t.closed = true
    t.fragments = nil
    // Note: We keep docxReader as it may be needed for rendering
    // Note: We keep source as it may be needed for rendering
    return nil
}
```

**Issue:** The comments indicate that `docxReader` and `source` are kept even after close, which may prevent garbage collection.

### 3.2 No Cleanup of DOCX Zip Readers
**File:** `pkg/stencil/docx.go`

The `DocxReader` struct holds references to `zip.File` objects but doesn't implement a `Close()` method. The zip reader itself is kept in memory.

### 3.3 Fragment Resources Accumulate
**File:** `pkg/stencil/stencil.go:46-62`

Fragment resources (media files, relationships) accumulate in the `renderContext` but are never explicitly cleaned up during rendering. For documents with many fragment inclusions, this could consume significant memory.

---

## 4. Expression Parser Weaknesses

### 4.1 Limited Operator Support
**File:** `pkg/stencil/expression.go:239`

The operator regex only captures single-character operators or specific two-character sequences:

```go
operatorRegex = regexp.MustCompile(`^(==|!=|<=|>=|\+|\-|\*|\/|\%|\&|\||\!|<|>|=)`)
```

**Missing:**
- `&&` and `||` (standard AND/OR) - uses single `&` and `|` instead
- `**` (exponentiation)
- String comparison operators

### 4.2 No Support for String Comparison in Control Structures
**File:** `pkg/stencil/expression.go:900-908`

Comparison operators (`<`, `>`, `<=`, `>=`) only work with numeric types:

```go
func evaluateLessThan(left, right interface{}) (interface{}, error) {
    leftNum, leftOk := toFloat64(left)
    rightNum, rightOk := toFloat64(right)
    if !leftOk || !rightOk {
        return nil, fmt.Errorf("cannot compare %T and %T", left, right)
    }
    return leftNum < rightNum, nil
}
```

**Impact:** Cannot do string sorting comparisons like `{{if name < "M"}}`.

### 4.3 Integer Overflow Not Handled
**File:** `pkg/stencil/expression.go:810-813`

Integer arithmetic doesn't check for overflow:

```go
if isInteger(left) && isInteger(right) {
    return int(leftNum + rightNum), nil  // No overflow check
}
```

### 4.4 Regex Compilation in Tokenizer
**File:** `pkg/stencil/expression.go:386-398`

A new regex is compiled inline during tokenization:

```go
if match := regexp.MustCompile(`^\.[0-9]+`).FindString(remaining); match != "" {
```

This should be a pre-compiled package-level variable for performance.

---

## 5. Control Structure Limitations

### 5.1 No Break or Continue Statements
**File:** `pkg/stencil/control.go`

The for loop implementation doesn't support `break` or `continue` statements. This limits the expressiveness of templates.

### 5.2 Limited Scope Management
**File:** `pkg/stencil/control.go:141-154`

Loop variables shadow parent scope but there's no way to access parent scope variables when shadowed:

```go
loopData := make(TemplateData)
for k, v := range data {
    loopData[k] = v  // Full copy
}
loopData[n.Variable] = item  // Overwrites if same name
```

### 5.3 No Maximum Render Depth Enforcement
**File:** `pkg/stencil/config.go:21`

While `MaxRenderDepth` is defined in Config, it's not actually enforced during rendering:

```go
MaxRenderDepth int  // Defined but not enforced
```

**Risk:** Circular fragment references could cause stack overflow.

### 5.4 No Limit on Loop Iterations
**File:** `pkg/stencil/render_docx.go:88-110`

For loops have no iteration limit:

```go
for idx, item := range items {
    // No check for maximum iterations
    loopRendered, err := renderElementsWithContext(loopBody, loopData, ctx)
}
```

Combined with the range() issue, this could cause DoS.

---

## 6. Function Implementation Issues

### 6.1 Type Coercion Inconsistencies
**File:** `pkg/stencil/functions.go:627-676`

The `toInteger` function silently truncates floats:

```go
case float32:
    return int(v), nil  // Truncates without warning
case float64:
    return int(v), nil  // Truncates without warning
```

### 6.2 contains() Uses String Comparison
**File:** `pkg/stencil/functions.go:891-927`

The `contains()` function converts both values to strings for comparison:

```go
searchStr := FormatValue(searchVal)
for _, item := range items {
    itemStr := FormatValue(item)
    if searchStr == itemStr {
        return true, nil
    }
}
```

**Issue:** This means `contains(1, list(1.0))` returns `true` because both format to "1".

### 6.3 Date Functions Locale Handling
**File:** `pkg/stencil/date_functions.go` (referenced)

Date formatting depends on system locale, which could lead to inconsistent output across deployments.

### 6.4 Missing Common String Functions
Notable missing functions:
- `trim()` / `trimLeft()` / `trimRight()`
- `split()`
- `substring()` / `slice()`
- `startsWith()` / `endsWith()`
- `regex()` / `match()`

---

## 7. Cache Implementation Concerns

### 7.1 Cache Key Not Based on Content
**File:** `pkg/stencil/cache.go:53-129`

The cache uses an external `key` parameter rather than hashing template content:

```go
func (tc *TemplateCache) Prepare(reader io.Reader, key string) (*PreparedTemplate, error) {
    // Key is provided externally
}
```

**Risk:** If the same key is used for different templates, incorrect cached templates could be returned.

**Recommendation:** Consider optionally using content hash as part of the key.

### 7.2 No Cache Metrics
The cache doesn't expose hit/miss metrics, making it hard to tune cache configuration.

### 7.3 TTL Checked Only on Access
**File:** `pkg/stencil/cache.go:67-72`

Expired entries are only removed when accessed:

```go
if tc.config.TTL > 0 && time.Now().After(entry.expiry) {
    tc.Remove(key)
}
```

Expired entries consume memory until accessed.

---

## 8. XML/HTML Processing Risks

### 8.1 No XML Declaration Handling
**File:** `pkg/stencil/xml_functions.go:30-31`

The XML parser wraps content in `<a>` tags without considering if the content already has an XML declaration:

```go
wrappedContent := "<a>" + content + "</a>"
```

### 8.2 Whitespace Handling Inconsistency
**File:** `pkg/stencil/xml_functions.go:98-100`

Text content is trimmed:

```go
textContent := strings.TrimSpace(string(t))
if textContent != "" {
```

This may not be desired for all use cases (e.g., preserving code indentation).

### 8.3 Limited Character Encoding Support
The engine assumes UTF-8 throughout but doesn't handle or convert other encodings that may appear in DOCX files.

---

## 9. Concurrency Issues

### 9.1 Global Registry Modification
**File:** `pkg/stencil/functions.go:88-96`

The global function registry can be modified at runtime:

```go
func GetDefaultFunctionRegistry() FunctionRegistry {
    registryOnce.Do(func() {
        globalRegistry = NewFunctionRegistry()
        registerBasicFunctions(globalRegistry)
    })
    return globalRegistry
}
```

**Risk:** If custom functions are registered on the global registry after templates start rendering, there could be race conditions.

### 9.2 Template Data Mutation
**File:** `pkg/stencil/stencil.go:371-378`

Template data is copied but mutations to nested objects still affect the original:

```go
renderData := make(TemplateData)
for k, v := range data {
    renderData[k] = v  // Shallow copy
}
```

If a template modifies a nested map/slice, the original data is affected.

### 9.3 Context Shared During Fragment Rendering
**File:** `pkg/stencil/render_docx.go:269-271`

Fragment elements are rendered with shared context:

```go
fragmentElements, err := renderElementsWithContext(fragment.parsed.Body.Elements, data, ctx)
```

If fragments modify the context, it could affect other parts of the document.

---

## 10. API Design Weaknesses

### 10.1 Inconsistent Nil Handling
Some functions return `nil, nil` for empty input, others return empty collections:

```go
// Returns nil, nil
func EvaluateVariable(...) (interface{}, error) {
    if data == nil { return nil, nil }
}

// Returns empty slice
func createRange(...) {
    return []interface{}{}, nil
}
```

### 10.2 No Streaming API
**File:** `pkg/stencil/stencil.go:365`

The `Render` method returns an `io.Reader` but the entire document is built in memory first:

```go
func (pt *PreparedTemplate) Render(data TemplateData) (io.Reader, error) {
    // ... builds entire document in memory
    return bytes.NewReader(buf.Bytes()), nil
}
```

For very large documents, this could be problematic.

### 10.3 No Template Validation API
There's no way to validate a template without rendering it. A `Validate()` method that checks syntax without data would be useful.

### 10.4 Missing Template Introspection
No API to:
- List variables used in a template
- List fragments required
- Get template metadata

---

## 11. Performance Concerns

### 11.1 Repeated Regex Compilation
**File:** `pkg/stencil/render.go:267`

Fragment regex is compiled at package level but similar patterns are compiled inline elsewhere.

### 11.2 String Concatenation in Loops
**File:** `pkg/stencil/render.go:60-103`

Text content is built using string concatenation:

```go
fullText := ""
for _, content := range para.Content {
    // ...
    fullText += c.Text.Content
}
```

**Recommendation:** Use `strings.Builder` consistently.

### 11.3 Deep Copy of Data for Each Loop Iteration
**File:** `pkg/stencil/control.go:141-148`

Each loop iteration copies the entire data context:

```go
for i, item := range items {
    loopData := make(TemplateData)
    for k, v := range data {
        loopData[k] = v
    }
}
```

For nested loops with large data contexts, this is expensive.

### 11.4 No Document Structure Caching
Parsed document structures are not cached between renders, even though the structure doesn't change.

---

## 12. Testing Gaps

### 12.1 Missing Fuzz Testing
The expression parser and HTML/XML parsers would benefit from fuzz testing to find edge cases and potential panics.

### 12.2 No Concurrency Tests
No tests verify thread-safety of:
- Template caching
- Global registry access
- Concurrent rendering with shared templates

### 12.3 No Memory Leak Tests
No tests verify that templates are properly garbage collected after `Close()`.

### 12.4 Limited Unicode Testing
The test suite doesn't extensively test Unicode handling:
- RTL languages
- Emoji
- Zero-width characters
- Unicode normalization

---

## Summary of Critical Issues

| Priority | Issue | Impact |
|----------|-------|--------|
| High | No range() size limit | Memory exhaustion DoS |
| High | No loop iteration limit | Memory/CPU DoS |
| High | MaxRenderDepth not enforced | Stack overflow with circular fragments |
| Medium | xml() allows arbitrary XML injection | Document integrity risk |
| Medium | Silent variable failures | Hard to debug templates |
| Medium | Shallow copy of template data | Data corruption risk |
| Low | Missing string comparison operators | Limited template expressiveness |
| Low | No streaming API | Memory issues with large docs |

---

## Recommendations

1. **Immediate:** Add limits to `range()` function and loop iterations
2. **Short-term:** Enforce `MaxRenderDepth` and add fragment cycle detection
3. **Short-term:** Add strict mode for variable access validation
4. **Medium-term:** Implement XXE-safe XML handling in `xml()` function
5. **Long-term:** Add streaming rendering for large documents
6. **Long-term:** Add template validation and introspection APIs

---

*Report generated: 2026-01-17*
*Reviewed: go-stencil template engine codebase*
