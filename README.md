# go-stencil

A Go implementation of the Stencil template engine for DOCX files.

## Overview

go-stencil is a powerful template engine that allows you to create dynamic Microsoft Word (.docx) documents using a simple template syntax. It's a Go port of the original [Stencil](https://github.com/erdos/stencil) project, with the implementation primarily developed by Claude Code (Anthropic's AI assistant) through a systematic, test-driven approach.

## Features

- **Simple template syntax** using `{{placeholders}}`
- **Control structures** for conditionals and loops
- **Built-in functions** for formatting and data manipulation
- **Support for tables**, images, and complex document structures
- **High performance** with template caching
- **Thread-safe** rendering with concurrent template support
- **Extensible** with custom functions and providers
- **Minimal dependencies** - uses only the Go standard library

## Installation

```bash
go get github.com/benjaminschreck/go-stencil
```

## Quick Start

```go
package main

import (
    "io"
    "log"
    "os"
    "time"

    "github.com/benjaminschreck/go-stencil/pkg/stencil"
)

func main() {
    // Prepare a template
    tmpl, err := stencil.PrepareFile("template.docx")
    if err != nil {
        log.Fatal(err)
    }
    defer tmpl.Close()

    // Create data for rendering
    data := stencil.TemplateData{
        "name": "John Doe",
		"date": time.Now().Format("January 2, 2006"),
        "items": []map[string]interface{}{
            {"name": "Item 1", "price": 10.00},
            {"name": "Item 2", "price": 20.00},
        },
    }

    // Render the template
    output, err := tmpl.Render(data)
    if err != nil {
        log.Fatal(err)
    }

    // Save the output
    file, err := os.Create("output.docx")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    _, err = io.Copy(file, output)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Template Syntax

### Variables

```
Hello {{name}}!
Your age is {{age}}.
```

### Conditionals

```
{{if isPremium}}
  Welcome, premium user!
{{else}}
  Consider upgrading to premium.
{{end}}
```

### Loops

```
{{for item in items}}
  - {{item.name}}: ${{item.price}}
{{end}}
```

### Functions

```
Total: {{format("%.2f", total)}}
Date: {{date("2006-01-02", orderDate)}}
```

#### Understanding Template Functions

**`format` Function**

- **Purpose**: Formats values using printf-style formatting (like Go's `fmt.Sprintf`)
- **Syntax**: `{{format("pattern", value)}}`
- **Example**: `{{format("%.2f", 19.999)}}` outputs `"19.99"`
- **Common patterns**:
  - `"%.2f"` - Two decimal places (19.99)
  - `"%d"` - Integer (42)
  - `"%05d"` - Zero-padded integer (00042)
  - `"%s: %d"` - String formatting ("Count: 5")

**`date` Function**

- **Purpose**: Formats date/time values into readable strings
- **Syntax**: `{{date("pattern", dateValue)}}`
- **Example**: `{{date("2006-01-02", orderDate)}}` outputs `"2024-03-15"`
- **Uses Go's time format** based on the reference date: Mon Jan 2 15:04:05 MST 2006
- **Common patterns**:
  - `"2006-01-02"` - Date only (2024-03-15)
  - `"January 2, 2006"` - Full date (March 15, 2024)
  - `"02/01/2006"` - European format (15/03/2024)
  - `"01/02/2006"` - US format (03/15/2024)
  - `"Monday, Jan 2, 2006 3:04 PM"` - Full datetime
  - `"15:04:05"` - Time only (10:30:00)

## API Documentation

### Basic Usage

The simplest way to use go-stencil is through the package-level functions:

```go
// Prepare a template from a file
tmpl, err := stencil.PrepareFile("template.docx")

// Or from an io.Reader
tmpl, err := stencil.Prepare(reader)

// Render with data
output, err := tmpl.Render(data)

// Don't forget to close when done
tmpl.Close()
```

### Advanced Usage with Engine

For more control, use the Engine API:

```go
// Create an engine with custom configuration
engine := stencil.NewWithOptions(
    stencil.WithCache(100),  // Enable caching with max 100 templates
    stencil.WithFunction("customFunc", myFunc),
)

// Or with a complete custom configuration
config := &stencil.Config{
    CacheMaxSize:   100,
    CacheTTL:       10 * time.Minute,
    LogLevel:       "info",
    MaxRenderDepth: 50,
    StrictMode:     true,
}
engine := stencil.NewWithConfig(config)

// Use the engine
tmpl, err := engine.PrepareFile("template.docx")
```

### Custom Functions

You can extend go-stencil with custom functions:

```go
// Define a custom function
type GreetingFunction struct{}

func (f GreetingFunction) Name() string {
    return "greeting"
}

func (f GreetingFunction) Call(args ...interface{}) (interface{}, error) {
    if len(args) < 1 {
        return "Hello!", nil
    }
    return fmt.Sprintf("Hello, %v!", args[0]), nil
}

// Register the function
stencil.RegisterGlobalFunction("greeting", GreetingFunction{})

// Or use a function provider for multiple functions
type MyFunctionProvider struct{}

func (p MyFunctionProvider) ProvideFunctions() map[string]stencil.Function {
    return map[string]stencil.Function{
        "greeting": GreetingFunction{},
        "farewell": FarewellFunction{},
    }
}

stencil.RegisterFunctionsFromProvider(MyFunctionProvider{})
```

### Template Fragments

Fragments allow you to reuse content across templates:

```go
// Add a text fragment
err := tmpl.AddFragment("copyright", "© 2024 My Company. All rights reserved.")

// Add a pre-formatted DOCX fragment
fragmentBytes, _ := os.ReadFile("header.docx")
err := tmpl.AddFragmentFromBytes("header", fragmentBytes)

// Use in template: {{include "copyright"}}
```

### Template Caching

Caching improves performance when rendering the same template multiple times:

```go
// Enable caching globally
stencil.SetCacheConfig(100, 10*time.Minute)

// Templates prepared with PrepareFile are automatically cached
tmpl1, _ := stencil.PrepareFile("template.docx") // Reads from disk
tmpl2, _ := stencil.PrepareFile("template.docx") // Returns from cache

// Clear the cache when needed
stencil.ClearCache()
```

## Built-in Functions

go-stencil includes a comprehensive set of built-in functions:

### Data Functions

- `empty(value)` - Check if a value is empty
- `coalesce(value1, value2, ...)` - Return the first non-empty value
- `list(items...)` - Create a list from arguments
- `data()` - Access the entire template data context
- `map(key, collection)` - Extract a specific field from each item in a collection

### String Functions

- `str(value)` - Convert to string
- `lowercase(text)` - Convert to lowercase
- `uppercase(text)` - Convert to uppercase
- `titlecase(text)` - Convert to title case
- `join(items, separator)` - Join items with separator
- `joinAnd(items)` - Join items with commas and "and"
- `replace(text, old, new)` - Replace text
- `length(value)` - Get length of string, array, or map

### Number Functions

- `integer(value)` - Convert to integer
- `decimal(value)` - Convert to decimal
- `round(number)` - Round to nearest integer
- `floor(number)` - Round down
- `ceil(number)` - Round up
- `sum(numbers...)` - Sum of numbers

### Formatting Functions

- `format(pattern, value)` - Format using printf-style pattern
- `formatWithLocale(locale, pattern, value)` - Format with specific locale
- `date(pattern, date)` - Format date/time
- `currency(amount)` - Format as currency
- `percent(value)` - Format as percentage

### Control Functions

- `switch(value, case1, result1, case2, result2, ..., default)` - Switch expression
- `contains(item, collection)` - Check if collection contains item
- `range(start, end)` - Generate a range of numbers

### Document Functions

- `pageBreak()` - Insert a page break
- `hideRow()` - Hide the current table row
- `hideColumn()` - Hide the current table column
- `html(content)` - Insert HTML-formatted content
- `xml(content)` - Insert raw XML content
- `replaceImage(base64Data)` - Replace an image
- `replaceLink(url)` - Replace a hyperlink
- `include(fragmentName)` - Include a named fragment

## Examples

See the [examples](examples/) directory for complete working examples:

- [Simple Example](examples/simple/) - Basic template rendering
- [Advanced Example](examples/advanced/) - Custom functions, fragments, and more

## Documentation

- [Getting Started Guide](docs/GETTING_STARTED.md) - Quick introduction and first steps
- [API Reference](docs/API.md) - Complete API documentation
- [Functions Reference](docs/FUNCTIONS.md) - All built-in template functions
- [Examples](docs/EXAMPLES.md) - Real-world usage examples
- [Best Practices](docs/BEST_PRACTICES.md) - Tips for effective template design

## License

This project is licensed under the Eclipse Public License 2.0 (EPL-2.0), the same license as the original Stencil project.

## Attribution

go-stencil is a Go implementation inspired by the original [Stencil](https://github.com/erdos/stencil) project by Janos Erdos.
