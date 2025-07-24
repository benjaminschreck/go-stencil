# go-stencil Examples

This document provides practical examples of using go-stencil for various document generation scenarios.

## Basic Examples

### Simple Letter Template

**Template (letter.docx):**
```
{{date("January 2, 2006", date)}}

Dear {{recipient}},

{{content}}

Sincerely,
{{sender}}
```

**Code:**
```go
tmpl, err := stencil.PrepareFile("letter.docx")
if err != nil {
    log.Fatal(err)
}
defer tmpl.Close()

data := stencil.TemplateData{
    "date":      time.Now(),
    "recipient": "Ms. Johnson",
    "content":   "Thank you for your interest in our products...",
    "sender":    "John Smith",
}

output, err := tmpl.Render(data)
```

### Invoice Template

**Template (invoice.docx):**
```
INVOICE #{{invoiceNumber}}
Date: {{date("01/02/2006", invoiceDate)}}

Bill To:
{{customer.name}}
{{customer.address}}

Items:
{{for item in items}}
{{item.description}} | {{item.quantity}} | ${{format("%.2f", item.unitPrice)}} | ${{format("%.2f", item.quantity * item.unitPrice)}}
{{end}}

Subtotal: ${{format("%.2f", subtotal)}}
Tax ({{format("%.1f", taxRate * 100)}}%): ${{format("%.2f", subtotal * taxRate)}}
Total: ${{format("%.2f", subtotal * (1 + taxRate))}}

{{if notes}}
Notes: {{notes}}
{{end}}
```

**Code:**
```go
data := stencil.TemplateData{
    "invoiceNumber": "INV-2024-001",
    "invoiceDate":   time.Now(),
    "customer": map[string]interface{}{
        "name":    "ACME Corporation",
        "address": "123 Business St, Suite 100",
    },
    "items": []map[string]interface{}{
        {
            "description": "Consulting Services",
            "quantity":    10,
            "unitPrice":   150.00,
        },
        {
            "description": "Software License",
            "quantity":    1,
            "unitPrice":   500.00,
        },
    },
    "subtotal": 2000.00,
    "taxRate":  0.08,
    "notes":    "Payment due within 30 days",
}
```

## Advanced Examples

### Dynamic Report with Charts

**Template (report.docx):**
```
{{uppercase(reportTitle)}}
Generated on {{date("Monday, January 2, 2006", generatedDate)}}

EXECUTIVE SUMMARY
{{summary}}

{{if showDetails}}
DETAILED ANALYSIS

{{for section in sections}}
{{section.title}}
{{section.content}}

Key Metrics:
{{for metric in section.metrics}}
- {{metric.name}}: {{metric.value}}{{if metric.unit}} {{metric.unit}}{{end}}
{{end}}

{{if section.showChart}}
[Chart placeholder - would contain actual chart]
{{end}}
{{end}}
{{end}}

{{pageBreak}}

APPENDIX
{{for appendix in appendices}}
{{include appendix}}
{{end}}
```

**Code:**
```go
// Add fragments for appendices
tmpl.AddFragment("methodology", "Our analysis methodology...")
tmpl.AddFragment("glossary", "Terms and definitions...")

data := stencil.TemplateData{
    "reportTitle":    "quarterly performance report",
    "generatedDate":  time.Now(),
    "summary":        "Overall performance exceeded expectations...",
    "showDetails":    true,
    "sections": []map[string]interface{}{
        {
            "title":   "Sales Performance",
            "content": "Q4 sales increased by 23%...",
            "metrics": []map[string]interface{}{
                {"name": "Total Revenue", "value": 1500000, "unit": "USD"},
                {"name": "Growth Rate", "value": 23, "unit": "%"},
                {"name": "New Customers", "value": 157},
            },
            "showChart": true,
        },
        {
            "title":   "Market Analysis",
            "content": "Market share expanded in key regions...",
            "metrics": []map[string]interface{}{
                {"name": "Market Share", "value": 18.5, "unit": "%"},
                {"name": "Brand Recognition", "value": 72, "unit": "%"},
            },
            "showChart": false,
        },
    },
    "appendices": []string{"methodology", "glossary"},
}
```

### Dynamic Table Generation

**Template with complex table logic:**
```
INVENTORY REPORT

{{for category in categories}}
{{uppercase(category.name)}}

| Product | Stock | Price | Status |
|---------|-------|-------|---------|
{{for product in category.products}}
{{if product.stock > 0}}
| {{product.name}} | {{product.stock}} | ${{format("%.2f", product.price)}} | {{if product.stock < 10}}Low Stock{{else}}Available{{end}} |
{{else}}
{{hideRow()}}
{{end}}
{{end}}

Total {{category.name}} Value: ${{format("%.2f", sum(map("value", category.products)))}}

{{end}}
```

**Code:**
```go
// Custom function to calculate product value
type ProductValueFunction struct{}

func (f ProductValueFunction) Name() string {
    return "productValue"
}

func (f ProductValueFunction) Call(args ...interface{}) (interface{}, error) {
    // Implementation to calculate stock * price
}

engine := stencil.NewWithOptions(
    stencil.WithFunction("productValue", ProductValueFunction{}),
)

data := stencil.TemplateData{
    "categories": []map[string]interface{}{
        {
            "name": "Electronics",
            "products": []map[string]interface{}{
                {"name": "Laptop", "stock": 15, "price": 999.99},
                {"name": "Mouse", "stock": 0, "price": 29.99}, // Will be hidden
                {"name": "Keyboard", "stock": 8, "price": 79.99},
            },
        },
    },
}
```

### Multi-language Document

**Template with conditional language support:**
```
{{if language == "es"}}
    Estimado {{customer.name}},
    
    Gracias por su pedido #{{orderNumber}}.
{{elsif language == "fr"}}
    Cher {{customer.name}},
    
    Merci pour votre commande #{{orderNumber}}.
{{else}}
    Dear {{customer.name}},
    
    Thank you for your order #{{orderNumber}}.
{{end}}

{{for item in items}}
- {{item.name}}: {{if language == "es"}}{{currency(item.price)}} EUR{{else}}{{currency(item.price)}}{{end}}
{{end}}
```

### Contract Template with Conditional Clauses

**Template (contract.docx):**
```
SERVICE AGREEMENT

This agreement is entered into on {{date("January 2, 2006", agreementDate)}} between:

{{company.name}} ("Service Provider")
{{client.name}} ("Client")

SERVICES
The Service Provider agrees to provide the following services:
{{for service in services}}
- {{service.description}}{{if service.timeline}} (Timeline: {{service.timeline}}){{end}}
{{end}}

PAYMENT TERMS
Total Contract Value: {{currency(totalValue)}}
Payment Schedule: {{paymentTerms}}

{{if includeNDA}}
NON-DISCLOSURE AGREEMENT
Both parties agree to maintain confidentiality...
{{end}}

{{if includeIPClause}}
INTELLECTUAL PROPERTY
{{if ipOwnership == "client"}}
All intellectual property created under this agreement shall belong to the Client.
{{elsif ipOwnership == "shared"}}
Intellectual property shall be jointly owned by both parties.
{{else}}
The Service Provider retains all intellectual property rights.
{{end}}
{{end}}

{{if customClauses}}
ADDITIONAL TERMS
{{for clause in customClauses}}
{{clause.title}}
{{clause.content}}
{{end}}
{{end}}

Signatures:

_____________________               _____________________
{{company.representative}}          {{client.representative}}
{{company.name}}                    {{client.name}}
```

### Form Letter with Complex Logic

**Template using multiple functions and conditions:**
```
{{if customer.preferredName}}
    {{set "salutation" customer.preferredName}}
{{else}}
    {{set "salutation" (join(list(customer.title, customer.lastName), " "))}}
{{end}}

Dear {{salutation}},

{{if customer.accountStatus == "premium"}}
    {{if customer.yearsWithUs >= 5}}
        As a valued premium member for {{customer.yearsWithUs}} years, you're eligible for our exclusive loyalty rewards!
    {{else}}
        Welcome to our premium membership program!
    {{end}}
{{elsif customer.accountStatus == "trial"}}
    We hope you're enjoying your trial. {{daysRemaining}} days remaining.
{{else}}
    Thank you for being our customer.
{{end}}

{{if not(empty(customer.recentPurchases))}}
Based on your recent purchases:
{{for purchase in customer.recentPurchases | limit(3)}}
- {{purchase.item}} ({{date("Jan 2", purchase.date)}})
{{end}}

We recommend:
{{for item in recommendations}}
- {{item.name}}: {{item.description}}
{{end}}
{{end}}

{{if customer.birthday | monthEquals(currentMonth)}}
🎉 Happy Birthday! Enjoy {{customer.birthdayDiscount}}% off your next purchase!
{{end}}

Best regards,
{{agent.name}}
{{agent.title}}
```

## Working with Images and Links

### Dynamic Image Replacement

**Template with image placeholders:**
```
PRODUCT CATALOG

{{for product in products}}
{{product.name}}
Price: {{currency(product.price)}}
{{product.description}}

Learn more: {{replaceLink(product.productUrl)}}
{{end}}
```

**Code:**
```go
// Load image and convert to base64
imageData, err := os.ReadFile("product1.png")
if err != nil {
    log.Fatal(err)
}
base64Image := "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData)

data := stencil.TemplateData{
    "products": []map[string]interface{}{
        {
            "name":        "Premium Widget",
            "imageData":   base64Image,
            "price":       99.99,
            "description": "Our top-of-the-line widget...",
            "productUrl":  "https://example.com/widget",
        },
    },
}
```

### HTML Content Rendering

**Template with formatted content:**
```
NEWSLETTER

{{html(headerHtml)}}

Dear {{subscriber.name}},

{{html(mainContent)}}

{{if subscriber.showPromotions}}
SPECIAL OFFERS:
{{for promo in promotions}}
{{html(promo.formattedContent)}}
{{end}}
{{end}}

{{html(footerHtml)}}
```

**Code:**
```go
data := stencil.TemplateData{
    "headerHtml": "<h1>Monthly Newsletter</h1><hr/>",
    "mainContent": `
        <p>Welcome to our <b>June 2024</b> newsletter!</p>
        <p>This month's highlights:</p>
        <ul>
            <li><i>New product launches</i></li>
            <li><u>Upcoming events</u></li>
            <li>Customer <span style="color: red;">success stories</span></li>
        </ul>
    `,
    "footerHtml": "<hr/><p><small>© 2024 Company. All rights reserved.</small></p>",
}
```

## Performance Optimization Examples

### Batch Processing with Caching

```go
// Enable caching for batch processing
stencil.SetCacheConfig(10, 30*time.Minute)

// Process multiple documents with the same template
tmpl, err := stencil.PrepareFile("invoice-template.docx")
if err != nil {
    log.Fatal(err)
}
defer tmpl.Close()

// Process invoices in batches
for _, batch := range invoiceBatches {
    for _, invoice := range batch {
        output, err := tmpl.Render(invoice)
        if err != nil {
            log.Printf("Failed to render invoice %s: %v", invoice["number"], err)
            continue
        }
        
        // Save output
        filename := fmt.Sprintf("invoice-%s.docx", invoice["number"])
        saveOutput(output, filename)
    }
}
```

### Concurrent Template Rendering

```go
// Create a worker pool for concurrent rendering
type RenderJob struct {
    Template string
    Data     stencil.TemplateData
    Output   string
}

func processJobs(jobs <-chan RenderJob, results chan<- error) {
    for job := range jobs {
        tmpl, err := stencil.PrepareFile(job.Template)
        if err != nil {
            results <- err
            continue
        }
        
        output, err := tmpl.Render(job.Data)
        if err != nil {
            tmpl.Close()
            results <- err
            continue
        }
        
        err = saveOutput(output, job.Output)
        tmpl.Close()
        results <- err
    }
}

// Use the worker pool
numWorkers := 4
jobs := make(chan RenderJob, 100)
results := make(chan error, 100)

// Start workers
for w := 1; w <= numWorkers; w++ {
    go processJobs(jobs, results)
}

// Queue jobs
for _, doc := range documents {
    jobs <- RenderJob{
        Template: doc.Template,
        Data:     doc.Data,
        Output:   doc.OutputPath,
    }
}
close(jobs)

// Collect results
for range documents {
    if err := <-results; err != nil {
        log.Printf("Render error: %v", err)
    }
}
```

## Error Handling Examples

### Comprehensive Error Handling

```go
func generateDocument(templatePath string, data stencil.TemplateData) error {
    tmpl, err := stencil.PrepareFile(templatePath)
    if err != nil {
        var templateErr *stencil.TemplateError
        if errors.As(err, &templateErr) {
            switch templateErr.Type {
            case "ParseError":
                return fmt.Errorf("template syntax error: %s", templateErr.Message)
            case "ValidationError":
                return fmt.Errorf("template validation failed: %s", templateErr.Message)
            default:
                return fmt.Errorf("template error (%s): %s", templateErr.Type, templateErr.Message)
            }
        }
        return fmt.Errorf("failed to prepare template: %w", err)
    }
    defer tmpl.Close()
    
    output, err := tmpl.Render(data)
    if err != nil {
        var templateErr *stencil.TemplateError
        if errors.As(err, &templateErr) {
            // Log detailed error information
            log.Printf("Render error details: %+v", templateErr.Details)
            
            if varName, ok := templateErr.Details["variable"].(string); ok {
                return fmt.Errorf("undefined variable '%s' in template", varName)
            }
            
            return fmt.Errorf("render error: %s", templateErr.Message)
        }
        return fmt.Errorf("failed to render template: %w", err)
    }
    
    // Save output...
    return nil
}
```

## Next Steps

- Review the [API Reference](API.md) for detailed method signatures
- Check the [Functions Reference](FUNCTIONS.md) for all available functions
- See [Best Practices](BEST_PRACTICES.md) for tips on template design