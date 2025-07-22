# Getting Started with go-stencil

This guide will help you get up and running with go-stencil, a powerful template engine for Microsoft Office documents.

## Installation

Install go-stencil using Go modules:

```bash
go get github.com/benjaminschreck/go-stencil
```

## Your First Template

### Step 1: Create a Template Document

Create a Word document (`template.docx`) with the following content:

```
Dear {{customer.name}},

Thank you for your order placed on {{date("January 2, 2006", orderDate)}}.

Your order details:
{{for item in items}}
- {{item.product}}: {{item.quantity}} x ${{format("%.2f", item.price)}}
{{end}}

Total: ${{format("%.2f", total)}}

{{if isPremium}}
As a premium customer, you've earned {{loyaltyPoints}} points!
{{else}}
Join our premium program to earn loyalty points!
{{end}}

Best regards,
{{companyName}}
```

### Step 2: Write Your Go Program

Create a `main.go` file:

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
    // Prepare the template
    tmpl, err := stencil.PrepareFile("template.docx")
    if err != nil {
        log.Fatal("Failed to prepare template:", err)
    }
    defer tmpl.Close()
    
    // Prepare the data
    data := stencil.TemplateData{
        "customer": map[string]interface{}{
            "name": "John Smith",
        },
        "orderDate": time.Now(),
        "items": []map[string]interface{}{
            {
                "product":  "Laptop",
                "quantity": 1,
                "price":    999.99,
            },
            {
                "product":  "Mouse",
                "quantity": 2,
                "price":    29.99,
            },
        },
        "total":        1059.97,
        "isPremium":    true,
        "loyaltyPoints": 106,
        "companyName":  "Tech Store Inc.",
    }
    
    // Render the template
    output, err := tmpl.Render(data)
    if err != nil {
        log.Fatal("Failed to render template:", err)
    }
    
    // Save the output
    outputFile, err := os.Create("output.docx")
    if err != nil {
        log.Fatal("Failed to create output file:", err)
    }
    defer outputFile.Close()
    
    _, err = io.Copy(outputFile, output)
    if err != nil {
        log.Fatal("Failed to save output:", err)
    }
    
    log.Println("Document generated successfully!")
}
```

### Step 3: Run Your Program

```bash
go run main.go
```

You'll now have an `output.docx` file with all the template variables replaced with your data!

## Understanding Template Syntax

### Variables

Access simple variables:
```
Hello {{name}}!
```

Access nested properties:
```
Customer: {{customer.name}}
Address: {{customer.address.street}}
```

Access array elements:
```
First item: {{items[0].name}}
```

### Expressions

Perform calculations:
```
Subtotal: ${{price * quantity}}
With tax: ${{price * quantity * 1.08}}
```

### Conditionals

Basic if statement:
```
{{if hasDiscount}}
    Discount applied: -${{discount}}
{{end}}
```

If-else statement:
```
{{if isPremium}}
    Premium shipping: FREE
{{else}}
    Standard shipping: $9.99
{{end}}
```

Multiple conditions:
```
{{if orderTotal > 100}}
    You qualify for free shipping!
{{elsif orderTotal > 50}}
    You qualify for discounted shipping!
{{else}}
    Standard shipping rates apply.
{{end}}
```

### Loops

Iterate over arrays:
```
{{for product in products}}
    - {{product.name}}: ${{product.price}}
{{end}}
```

Access index in loops:
```
{{for idx, product in products}}
    {{idx + 1}}. {{product.name}}
{{end}}
```

### Functions

go-stencil includes many built-in functions:

```
{{uppercase(name)}}                    // Convert to uppercase
{{lowercase(email)}}                   // Convert to lowercase
{{format("%.2f", price)}}             // Format numbers
{{date("Jan 2, 2006", orderDate)}}    // Format dates
{{join(items, ", ")}}                 // Join array elements
{{currency(amount)}}                  // Format as currency
{{if empty(notes)}}No notes{{end}}    // Check if empty
```

## Working with Tables

go-stencil can dynamically generate table rows:

```
| Product | Quantity | Price |
{{for item in items}}
| {{item.product}} | {{item.quantity}} | ${{format("%.2f", item.price)}} |
{{end}}
```

Hide rows conditionally:
```
{{for item in items}}
{{if item.quantity > 0}}
| {{item.product}} | {{item.quantity}} |
{{else}}
{{hideRow()}}
{{end}}
{{end}}
```

## Advanced Features

### Custom Functions

Add your own template functions:

```go
// Define a custom function
type TaxCalculator struct {
    rate float64
}

func (t TaxCalculator) Name() string {
    return "calculateTax"
}

func (t TaxCalculator) Call(args ...interface{}) (interface{}, error) {
    if len(args) < 1 {
        return 0, fmt.Errorf("calculateTax requires an amount")
    }
    
    amount, ok := args[0].(float64)
    if !ok {
        return 0, fmt.Errorf("amount must be a number")
    }
    
    return amount * t.rate, nil
}

// Register the function
engine := stencil.NewWithOptions(
    stencil.WithFunction("calculateTax", TaxCalculator{rate: 0.08}),
)

// Use in template
tmpl, err := engine.PrepareFile("template.docx")
```

Then in your template:
```
Tax: ${{format("%.2f", calculateTax(subtotal))}}
```

### Template Fragments

Reuse common content across templates:

```go
// Add a text fragment
err := tmpl.AddFragment("disclaimer", 
    "This document is confidential and proprietary.")

// Add a formatted fragment from another DOCX
headerBytes, _ := os.ReadFile("header.docx")
err := tmpl.AddFragmentFromBytes("header", headerBytes)
```

Use fragments in your template:
```
{{include "header"}}

Your content here...

{{include "disclaimer"}}
```

### Caching for Performance

Enable caching when rendering the same template multiple times:

```go
// Enable global cache
stencil.SetCacheConfig(100, 10*time.Minute)

// Templates are automatically cached
tmpl1, _ := stencil.PrepareFile("invoice.docx") // Reads from disk
tmpl2, _ := stencil.PrepareFile("invoice.docx") // Returns from cache

// Clear cache when needed
stencil.ClearCache()
```

### Using the Engine API

For more control, use the Engine API:

```go
// Create a custom engine
engine := stencil.NewWithOptions(
    stencil.WithCache(50),
    stencil.WithLogLevel("debug"),
    stencil.WithStrictMode(true), // Fail on undefined variables
)

// Prepare templates using the engine
tmpl, err := engine.PrepareFile("template.docx")
```

## Error Handling

go-stencil provides detailed error information:

```go
output, err := tmpl.Render(data)
if err != nil {
    var templateErr *stencil.TemplateError
    if errors.As(err, &templateErr) {
        log.Printf("Error type: %s", templateErr.Type)
        log.Printf("Error message: %s", templateErr.Message)
        
        // Access error details
        if lineNum, ok := templateErr.Details["line"].(int); ok {
            log.Printf("Error at line: %d", lineNum)
        }
    }
}
```

## Next Steps

- Explore the [API Reference](API.md) for detailed documentation
- Check out the [Functions Reference](FUNCTIONS.md) for all built-in functions
- See [Examples](EXAMPLES.md) for more complex use cases
- Read about [Template Best Practices](BEST_PRACTICES.md)

## Getting Help

If you encounter issues:

1. Check the error messages - they often contain helpful details
2. Enable debug logging to see what's happening:
   ```go
   engine := stencil.NewWithOptions(
       stencil.WithLogLevel("debug"),
   )
   ```
3. Review the examples in the `examples/` directory
4. File an issue on GitHub with a minimal reproducible example