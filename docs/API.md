# API Reference

## Package stencil

The `stencil` package provides a powerful template engine for Microsoft Word documents (DOCX). It enables dynamic document generation by processing templates with placeholders, control structures, and built-in functions.

## Core Types

### PreparedTemplate
Represents a prepared template ready for rendering.

```go
type PreparedTemplate struct {
    // Contains prepared template data
}
```

Common methods:

```go
func (pt *PreparedTemplate) Render(data TemplateData) (io.Reader, error)
func (pt *PreparedTemplate) Validate(schema TemplateSchema) (ValidateTemplateResult, error)
func (pt *PreparedTemplate) Close() error
func (pt *PreparedTemplate) AddFragment(name, content string) error
func (pt *PreparedTemplate) AddFragmentFromBytes(name string, docxBytes []byte) error
```

### TemplateData
Type alias for template data context.

```go
type TemplateData = map[string]interface{}
```

### TemplateSchema
Render-shaped type schema used by `PreparedTemplate.Validate`.

```go
type TemplateSchema map[string]TemplateType

var (
    String TemplateType
    Number TemplateType
    Bool   TemplateType
    Any    TemplateType
)

func Object(fields TemplateSchema) TemplateType
func List(element TemplateType) TemplateType
func Nullable(t TemplateType) TemplateType
```

Example:

```go
result, err := tmpl.Validate(stencil.TemplateSchema{
    "user": stencil.Object(stencil.TemplateSchema{
        "name": stencil.String,
        "age":  stencil.Number,
    }),
    "items": stencil.List(stencil.Object(stencil.TemplateSchema{
        "title": stencil.String,
        "price": stencil.Number,
    })),
})
```

`Validate` checks the prepared template body, main-template header/footer parts, and the bodies of all statically reachable fragment includes such as `{{include "header"}}`. Dynamic includes such as `{{include fragmentName}}` are syntax/type-checked, but not traversed because the concrete fragment cannot be known without render data. DOCX fragment validation currently scans the fragment document body, not fragment header/footer parts.

### Engine
The main template engine that manages template preparation and rendering.

```go
type Engine struct {
    config   *Config
    cache    *TemplateCache
    registry FunctionRegistry
}
```

## Functions

### Validation APIs

#### ValidateTemplate (Preferred)
Validates template syntax + semantic schema compatibility from raw DOCX bytes in one call.

```go
func ValidateTemplate(input ValidateTemplateInput) (ValidateTemplateResult, error)
```

```go
type ValidateTemplateInput struct {
    DocxBytes          []byte
    TemplateRevisionID string
    Strict             bool
    IncludeWarnings    bool
    MaxIssues          int // 0 = unlimited
    Schema             ValidationSchema
}

type ValidationSchema struct {
    Fields    []FieldDefinition
    Functions []FunctionDefinition
}

type FieldDefinition struct {
    Path       string
    Type       string
    Nullable   bool
    Collection bool
}

type FunctionDefinition struct {
    Name       string
    MinArgs    int
    MaxArgs    int
    ArgKinds   [][]string
    ReturnKind string
}

type ValidateTemplateResult struct {
    Valid           bool
    Summary         StencilValidationSummary
    Issues          []StencilValidationIssue
    IssuesTruncated bool
    Metadata        StencilMetadata
}
```

Key behavior:
- Performs syntax and semantic checks in one pass.
- Supports syntax and semantic issue codes:
  - `SYNTAX_ERROR`
  - `CONTROL_BLOCK_MISMATCH`
  - `UNSUPPORTED_EXPRESSION`
  - `UNKNOWN_FIELD`
  - `UNKNOWN_FUNCTION`
  - `FUNCTION_ARGUMENT_ERROR`
  - `TYPE_MISMATCH`
- `strict=true` emits semantic issues as `error`; `strict=false` emits semantic issues as `warning`.
- `includeWarnings=false` filters warnings from returned `issues` (summary counts remain pre-filter).
- `maxIssues=0` means unbounded issue return.
- `issuesTruncated=true` only when post-filter issues exceed `maxIssues`.
- `summary.errorCount` and `summary.warningCount` are pre-filter and pre-truncation.
- `summary.returnedIssueCount == len(issues)` is always maintained.
- Returned metadata includes `documentHash`, optional `templateRevisionId`, and `parserVersion`.
- Issues are emitted with deterministic `location` data (`part`, `tokenOrdinal`, UTF-16 offsets, `anchorId`).
- Returned issues always include both `token` and `location`.

Example:
```go
result, err := stencil.ValidateTemplate(stencil.ValidateTemplateInput{
    DocxBytes:       docxBytes,
    Strict:          true,
    IncludeWarnings: true,
    MaxIssues:       0,
    Schema:          schema,
})
```

#### ValidateTemplateSyntax (Low-Level)
Validates syntax/control structure only.

```go
func ValidateTemplateSyntax(input ValidateTemplateSyntaxInput) (ValidateTemplateSyntaxResult, error)
```

Use this when semantic schema validation is handled elsewhere.

#### ExtractReferences (Low-Level)
Extracts variable/function/control references from parsed template token ASTs.

```go
func ExtractReferences(input ExtractReferencesInput) (ExtractReferencesResult, error)
```

Key behavior:
- Traverses `word/document.xml`, then headers, then footers in deterministic order.
- Reference ordering is deterministic for identical DOCX bytes.

### Template Preparation

#### PrepareFile
Prepares a template from a file path.

```go
func PrepareFile(filePath string) (*PreparedTemplate, error)
```

**Parameters:**
- `filePath`: Path to the template file (.docx)

**Returns:**
- `*PreparedTemplate`: Prepared template ready for rendering
- `error`: Error if preparation fails

**Example:**
```go
tmpl, err := stencil.PrepareFile("invoice-template.docx")
if err != nil {
    log.Fatal(err)
}
defer tmpl.Close()
```

#### Prepare
Prepares a template from an io.Reader.

```go
func Prepare(reader io.Reader) (*PreparedTemplate, error)
```

**Parameters:**
- `reader`: Reader containing the template data

**Returns:**
- `*PreparedTemplate`: Prepared template ready for rendering
- `error`: Error if preparation fails

**Example:**
```go
file, err := os.Open("template.docx")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

tmpl, err := stencil.Prepare(file)
```

### Engine Creation

#### New
Creates a new Engine with default configuration.

```go
func New() *Engine
```

#### NewWithOptions
Creates a new Engine with functional options.

```go
func NewWithOptions(opts ...Option) *Engine
```

**Available Options:**
- `WithConfig(config *Config)`: Set a complete engine configuration
- `WithCache(maxSize int)`: Enable caching with maximum templates
- `WithFunction(name string, fn Function)`: Register a custom function
- `WithFunctionProvider(provider FunctionProvider)`: Register multiple functions

**Example:**
```go
engine := stencil.NewWithOptions(
    stencil.WithConfig(&stencil.Config{
        CacheMaxSize:   100,
        CacheTTL:       10 * time.Minute,
        LogLevel:       "debug",
        MaxRenderDepth: 100,
        StrictMode:     true,
    }),
    stencil.WithFunction("myUpper", myUpperFunc),
)
```

#### NewWithConfig
Creates a new Engine with a complete configuration.

```go
func NewWithConfig(config *Config) *Engine
```

### Template Rendering

#### (*PreparedTemplate) Render
Renders the template with provided data.

```go
func (pt *PreparedTemplate) Render(data TemplateData) (io.Reader, error)
```

**Parameters:**
- `data`: Map containing template variables

**Returns:**
- `io.Reader`: Reader containing the rendered document
- `error`: Error if rendering fails

**Example:**
```go
data := stencil.TemplateData{
    "customer": map[string]interface{}{
        "name": "ACME Corp",
        "address": "123 Main St",
    },
    "items": []map[string]interface{}{
        {"product": "Widget", "quantity": 10, "price": 19.99},
        {"product": "Gadget", "quantity": 5, "price": 29.99},
    },
    "date": time.Now(),
}

output, err := tmpl.Render(data)
if err != nil {
    log.Fatal(err)
}
```

### Fragment Management

#### (*PreparedTemplate) AddFragment
Adds a named text fragment to the template.

```go
func (pt *PreparedTemplate) AddFragment(name, content string) error
```

**Parameters:**
- `name`: Fragment identifier
- `content`: Text content of the fragment

**Example:**
```go
err := tmpl.AddFragment("copyright", "© 2024 My Company. All rights reserved.")
```

#### (*PreparedTemplate) AddFragmentFromBytes
Adds a pre-formatted DOCX fragment.

```go
func (pt *PreparedTemplate) AddFragmentFromBytes(name string, docxBytes []byte) error
```

**Parameters:**
- `name`: Fragment identifier
- `docxBytes`: Bytes of a DOCX file to use as fragment

**Example:**
```go
headerBytes, err := os.ReadFile("header-fragment.docx")
if err != nil {
    log.Fatal(err)
}
err = tmpl.AddFragmentFromBytes("header", headerBytes)
```

### Custom Functions

#### Function Interface
Interface for custom template functions.

```go
type Function interface {
    // Call executes the function with arguments
    Call(args ...interface{}) (interface{}, error)

    // Name returns the function name
    Name() string

    // MinArgs returns the minimum number of arguments required
    MinArgs() int

    // MaxArgs returns the maximum number of arguments allowed (-1 for unlimited)
    MaxArgs() int
}
```

#### RegisterGlobalFunction
Registers a function globally for all templates.

```go
func RegisterGlobalFunction(name string, fn Function) error
```

**Example:**
```go
myUpperFunc := stencil.NewSimpleFunction("myUpper", 1, 1, func(args ...interface{}) (interface{}, error) {
    if len(args) < 1 {
        return "", fmt.Errorf("uppercase requires at least one argument")
    }
    return strings.ToUpper(fmt.Sprint(args[0])), nil
})

stencil.RegisterGlobalFunction("myUpper", myUpperFunc)
```

#### FunctionProvider Interface
Interface for providing multiple functions.

```go
type FunctionProvider interface {
    ProvideFunctions() map[string]Function
}
```

#### RegisterFunctionsFromProvider
Registers functions from a provider.

```go
func RegisterFunctionsFromProvider(provider FunctionProvider) error
```

### Cache Management

#### SetCacheConfig
Configures the global template cache.

```go
func SetCacheConfig(maxSize int, ttl time.Duration)
```

**Parameters:**
- `maxSize`: Maximum number of templates to cache
- `ttl`: Time-to-live for cached templates

#### ClearCache
Clears all cached templates.

```go
func ClearCache()
```

## Configuration

### Config Structure
Complete configuration for the template engine.

```go
type Config struct {
    // CacheMaxSize is the maximum number of templates to cache (0 = disabled)
    CacheMaxSize int
    
    // CacheTTL is the time-to-live for cached templates
    CacheTTL time.Duration
    
    // LogLevel controls logging verbosity (debug, info, warn, error)
    LogLevel string

    // MaxRenderDepth prevents infinite recursion in templates
    MaxRenderDepth int

    // StrictMode enables strict variable checking
    StrictMode bool
}
```

### DefaultConfig
Returns the default configuration.

```go
func DefaultConfig() *Config
```

## Error Types

### TemplateError
Base error type for template-related errors.

```go
type TemplateError struct {
    Message string
    Line    int
    Column  int
}
```

### Common Error Types
- `ParseError`: Template parsing failed
- `RenderError`: Template rendering failed
- `FunctionError`: Function execution failed
- `ValidationError`: Validation failed
- `ResourceError`: Resource access failed

## Built-in Functions Reference

See the [Functions Documentation](FUNCTIONS.md) for a complete list of built-in functions.

## Thread Safety

- `Engine` instances are thread-safe and can be shared across goroutines
- `PreparedTemplate` instances are NOT thread-safe; use one per goroutine or synchronize access
- The global template cache is thread-safe
- Custom functions should be thread-safe if used concurrently

## Best Practices

1. **Always close templates** when done to release resources:
   ```go
   tmpl, err := stencil.PrepareFile("template.docx")
   if err != nil {
       log.Fatal(err)
   }
   defer tmpl.Close()
   ```

2. **Use caching** for frequently used templates:
   ```go
   stencil.SetCacheConfig(100, 10*time.Minute)
   ```

3. **Handle errors appropriately**:
   ```go
   output, err := tmpl.Render(data)
   if err != nil {
       var templateErr *stencil.TemplateError
       if errors.As(err, &templateErr) {
           log.Printf("Template error at line %d, column %d: %s",
               templateErr.Line,
               templateErr.Column,
               templateErr.Message,
           )
       }
   }
   ```

4. **Use strict mode** during development to catch undefined variables:
   ```go
   engine := stencil.NewWithOptions(
       stencil.WithConfig(&stencil.Config{
           CacheMaxSize:   100,
           LogLevel:       "info",
           MaxRenderDepth: 100,
           StrictMode:     true,
       }),
   )
   ```

5. **Prepare templates once** and render multiple times for better performance.
