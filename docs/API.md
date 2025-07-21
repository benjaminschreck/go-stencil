# API Reference

## Package stencil

The `stencil` package provides a powerful template engine for Microsoft Word documents (DOCX). It enables dynamic document generation by processing templates with placeholders, control structures, and built-in functions.

## Core Types

### Template
Represents a prepared template ready for rendering.

```go
type Template interface {
    // Render executes the template with the provided data
    Render(data TemplateData) (io.Reader, error)
    
    // Close releases resources associated with the template
    Close() error
    
    // AddFragment adds a named text fragment
    AddFragment(name, content string) error
    
    // AddFragmentFromBytes adds a named fragment from DOCX bytes
    AddFragmentFromBytes(name string, docxBytes []byte) error
}
```

### PreparedTemplate
The concrete implementation of the Template interface.

```go
type PreparedTemplate struct {
    // Contains prepared template data
}
```

### TemplateData
Type alias for template data context.

```go
type TemplateData = map[string]interface{}
```

### Engine
The main template engine that manages template preparation and rendering.

```go
type Engine struct {
    config    *Config
    functions map[string]Function
    cache     *TemplateCache
}
```

## Functions

### Template Preparation

#### PrepareFile
Prepares a template from a file path.

```go
func PrepareFile(filePath string) (Template, error)
```

**Parameters:**
- `filePath`: Path to the template file (.docx)

**Returns:**
- `Template`: Prepared template ready for rendering
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
func Prepare(reader io.Reader) (Template, error)
```

**Parameters:**
- `reader`: Reader containing the template data

**Returns:**
- `Template`: Prepared template ready for rendering
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
- `WithCache(maxSize int)`: Enable caching with maximum templates
- `WithCacheTTL(ttl time.Duration)`: Set cache TTL
- `WithFunction(name string, fn Function)`: Register a custom function
- `WithFunctionsProvider(provider FunctionProvider)`: Register multiple functions
- `WithLogger(logger *log.Logger)`: Set custom logger
- `WithLogLevel(level string)`: Set log level (debug, info, warn, error)
- `WithStrictMode(strict bool)`: Enable strict mode for undefined variables

**Example:**
```go
engine := stencil.NewWithOptions(
    stencil.WithCache(100),
    stencil.WithCacheTTL(10*time.Minute),
    stencil.WithFunction("custom", myCustomFunc),
    stencil.WithLogLevel("debug"),
)
```

#### NewWithConfig
Creates a new Engine with a complete configuration.

```go
func NewWithConfig(config *Config) *Engine
```

### Template Rendering

#### (Template) Render
Renders the template with provided data.

```go
func (t *Template) Render(data TemplateData) (io.Reader, error)
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

#### (Template) AddFragment
Adds a named text fragment to the template.

```go
func (t *Template) AddFragment(name, content string) error
```

**Parameters:**
- `name`: Fragment identifier
- `content`: Text content of the fragment

**Example:**
```go
err := tmpl.AddFragment("copyright", "© 2024 My Company. All rights reserved.")
```

#### (Template) AddFragmentFromBytes
Adds a pre-formatted DOCX fragment.

```go
func (t *Template) AddFragmentFromBytes(name string, docxBytes []byte) error
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
    // Name returns the function name
    Name() string
    
    // Call executes the function with arguments
    Call(args ...interface{}) (interface{}, error)
}
```

#### RegisterGlobalFunction
Registers a function globally for all templates.

```go
func RegisterGlobalFunction(name string, fn Function) error
```

**Example:**
```go
type UppercaseFunction struct{}

func (f UppercaseFunction) Name() string {
    return "myUpper"
}

func (f UppercaseFunction) Call(args ...interface{}) (interface{}, error) {
    if len(args) < 1 {
        return "", fmt.Errorf("uppercase requires at least one argument")
    }
    return strings.ToUpper(fmt.Sprint(args[0])), nil
}

stencil.RegisterGlobalFunction("myUpper", UppercaseFunction{})
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
    
    // Logger is a custom logger instance
    Logger *log.Logger
    
    // MaxRenderDepth prevents infinite recursion in templates
    MaxRenderDepth int
    
    // StrictMode enables strict variable checking
    StrictMode bool
    
    // CustomFunctions is a map of custom template functions
    CustomFunctions map[string]Function
    
    // FunctionProviders is a list of function providers
    FunctionProviders []FunctionProvider
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
    Type    string
    Message string
    Details map[string]interface{}
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
- `Template` instances are NOT thread-safe; use one per goroutine or synchronize access
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
           log.Printf("Template error: %s - %s", templateErr.Type, templateErr.Message)
       }
   }
   ```

4. **Use strict mode** during development to catch undefined variables:
   ```go
   engine := stencil.NewWithOptions(
       stencil.WithStrictMode(true),
   )
   ```

5. **Prepare templates once** and render multiple times for better performance.