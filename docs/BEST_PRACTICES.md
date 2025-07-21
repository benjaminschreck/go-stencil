# Best Practices for go-stencil Templates

This guide provides recommendations for creating efficient, maintainable, and robust document templates with go-stencil.

## Template Design

### 1. Keep Templates Simple

**Do:** Break complex logic into smaller, manageable pieces
```
{{if hasDiscount}}
  Discount: {{format("%.0f", discountPercent)}}%
{{end}}
```

**Avoid:** Deeply nested conditionals
```
{{if customer}}{{if customer.status}}{{if customer.status == "premium"}}{{if discounts}}...{{end}}{{end}}{{end}}{{end}}
```

### 2. Use Meaningful Variable Names

**Do:** Use descriptive names that indicate the data type
```
{{customerName}}
{{orderItems}}
{{isActive}}
{{hasPermission}}
```

**Avoid:** Single letters or ambiguous names
```
{{n}}
{{data}}
{{flag}}
```

### 3. Prepare Data in Your Application

**Do:** Pre-calculate complex values in Go
```go
data := stencil.TemplateData{
    "subtotal": 100.00,
    "tax": 8.00,
    "total": 108.00,  // Pre-calculated
}
```

**Avoid:** Complex calculations in templates
```
{{(price * quantity) * (1 + taxRate) - discount + shipping}}
```

## Performance Optimization

### 1. Enable Template Caching

For applications that render the same templates multiple times:

```go
// Enable global caching
stencil.SetCacheConfig(50, 30*time.Minute)

// Or use a custom engine with caching
engine := stencil.NewWithOptions(
    stencil.WithCache(100),
    stencil.WithCacheTTL(1*time.Hour),
)
```

### 2. Minimize Function Calls in Loops

**Do:** Pre-process data when possible
```go
// In Go:
for i, item := range items {
    items[i]["formattedPrice"] = fmt.Sprintf("$%.2f", item["price"])
}

// In template:
{{for item in items}}
  {{item.formattedPrice}}
{{end}}
```

**Avoid:** Repeated function calls
```
{{for item in items}}
  {{format("$%.2f", item.price)}}  // Called for each item
{{end}}
```

### 3. Use Fragments for Repeated Content

Instead of duplicating content across templates:

```go
// Define once
tmpl.AddFragment("companyHeader", "ACME Corp - Professional Services")

// Use many times
{{include "companyHeader"}}
```

## Error Handling

### 1. Use Strict Mode During Development

```go
engine := stencil.NewWithOptions(
    stencil.WithStrictMode(true),
)
```

This helps catch:
- Undefined variables
- Type mismatches
- Invalid function calls

### 2. Provide Default Values

**Do:** Use coalesce for fallbacks
```
{{coalesce(user.displayName, user.email, "Guest")}}
```

**Do:** Check for empty values
```
{{if empty(notes)}}
  No additional notes
{{else}}
  {{notes}}
{{end}}
```

### 3. Validate Data Before Rendering

```go
func validateInvoiceData(data stencil.TemplateData) error {
    if _, ok := data["invoiceNumber"]; !ok {
        return errors.New("invoiceNumber is required")
    }
    
    if items, ok := data["items"].([]interface{}); ok {
        if len(items) == 0 {
            return errors.New("at least one item is required")
        }
    }
    
    return nil
}

// Use before rendering
if err := validateInvoiceData(data); err != nil {
    return fmt.Errorf("invalid invoice data: %w", err)
}
```

## Document Structure

### 1. Preserve Document Formatting

- Test templates with various data to ensure formatting remains intact
- Use styles in the original document rather than inline formatting
- Keep table structures consistent

### 2. Handle Dynamic Tables Properly

**For hiding rows:**
```
{{for item in items}}
{{if item.isVisible}}
| {{item.name}} | {{item.value}} |
{{else}}
{{hideRow}}
{{end}}
{{end}}
```

**For conditional columns:**
```
| Name | {{if showPrices}}Price{{else}}{{hideColumn}}{{end}} |
```

### 3. Use Page Breaks Wisely

```
{{for chapter in chapters}}
{{if not(chapter.isFirst)}}
{{pageBreak}}
{{end}}
{{chapter.content}}
{{end}}
```

## Data Organization

### 1. Structure Data Hierarchically

```go
data := stencil.TemplateData{
    "invoice": map[string]interface{}{
        "number": "INV-001",
        "date": time.Now(),
        "customer": map[string]interface{}{
            "name": "ACME Corp",
            "address": map[string]interface{}{
                "street": "123 Main St",
                "city": "Springfield",
            },
        },
    },
}
```

Template usage:
```
Invoice #{{invoice.number}}
Customer: {{invoice.customer.name}}
Address: {{invoice.customer.address.street}}, {{invoice.customer.address.city}}
```

### 2. Use Arrays for Repeated Data

```go
data := stencil.TemplateData{
    "items": []map[string]interface{}{
        {"name": "Widget", "quantity": 10, "price": 9.99},
        {"name": "Gadget", "quantity": 5, "price": 19.99},
    },
}
```

### 3. Include Metadata

```go
data := stencil.TemplateData{
    // Actual data
    "content": content,
    
    // Metadata for template logic
    "_meta": map[string]interface{}{
        "generatedAt": time.Now(),
        "version": "1.2.3",
        "environment": "production",
    },
}
```

## Testing Templates

### 1. Create Test Documents

Maintain a set of test templates with edge cases:
- Empty data sets
- Very long text
- Special characters
- Missing optional fields

### 2. Use Debug Logging

```go
engine := stencil.NewWithOptions(
    stencil.WithLogLevel("debug"),
)
```

### 3. Test with Realistic Data

```go
func TestInvoiceTemplate(t *testing.T) {
    testCases := []struct {
        name string
        data stencil.TemplateData
    }{
        {
            name: "single item",
            data: createInvoiceData(1),
        },
        {
            name: "multiple items",
            data: createInvoiceData(50),
        },
        {
            name: "with discounts",
            data: createInvoiceDataWithDiscounts(),
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test template rendering
        })
    }
}
```

## Security Considerations

### 1. Sanitize User Input

When including user-provided content:

```go
// Sanitize HTML content before passing to template
sanitized := sanitizeHTML(userContent)
data["content"] = sanitized
```

### 2. Be Careful with XML Function

The `xml()` function inserts raw XML. Only use with trusted content:

```
{{xml(trustedXmlContent)}}  // OK if content is validated
{{xml(userInput)}}  // DANGER - could break document
```

### 3. Validate File Paths

When using fragments from files:

```go
// Validate path is within allowed directory
if !strings.HasPrefix(filepath.Clean(fragmentPath), allowedDir) {
    return errors.New("invalid fragment path")
}
```

## Maintenance

### 1. Version Your Templates

Keep track of template versions:
- Use version control (git)
- Document changes
- Test backward compatibility

### 2. Document Template Variables

Create documentation for each template:

```markdown
# Invoice Template Variables

Required:
- invoice.number (string): Invoice number
- invoice.date (time.Time): Invoice date
- customer.name (string): Customer name
- items ([]Item): Line items

Optional:
- discount (float64): Discount percentage
- notes (string): Additional notes
```

### 3. Use Consistent Naming Conventions

- camelCase for variables: `customerName`
- Use prefixes for booleans: `isActive`, `hasDiscount`
- Use plural for arrays: `items`, `products`

## Common Pitfalls to Avoid

### 1. Don't Modify Original Templates

Always work on copies:
```go
// Create a working copy
templateCopy := copyFile(originalTemplate)
tmpl, err := stencil.PrepareFile(templateCopy)
```

### 2. Handle Missing Relationships

In DOCX files, be aware of:
- Images without proper relationships
- Hyperlinks that may not exist
- Referenced styles that might be missing

### 3. Test Across Different Office Versions

Templates may render differently in:
- Different versions of Microsoft Office
- LibreOffice/OpenOffice
- Google Docs (after import)

### 4. Avoid Excessive Template Logic

If you find yourself writing complex logic in templates, consider:
- Moving logic to your application code
- Creating custom functions
- Simplifying the data structure

## Summary

Following these best practices will help you create maintainable, efficient, and robust document templates. Remember:

1. Keep templates simple and focused
2. Prepare data in your application
3. Use caching for performance
4. Handle errors gracefully
5. Test thoroughly with realistic data
6. Document your templates
7. Consider security implications

For more examples and patterns, see the [Examples](EXAMPLES.md) documentation.