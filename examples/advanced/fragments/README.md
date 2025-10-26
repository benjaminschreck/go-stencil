# Fragment Templates for Go-Stencil

This directory contains comprehensive fragment templates that demonstrate the full capabilities of go-stencil's fragment system, including nested fragments, complex tables, styled content, and advanced template features.

## Fragment Files

### 📄 fragment1.md - Company Header Fragment
**Purpose:** Reusable company header with contact information and document control
**Features Demonstrated:**
- Conditional content display ({{if}} statements)
- Dynamic timestamps
- Table formatting
- Variable substitution
- Can include fragment2 for nested demonstration

**Key Variables:**
- `showSales`, `showSupport` - Control which departments to display
- `documentId`, `version`, `classification` - Document metadata
- `isConfidential` - Toggle confidentiality warning
- `includeSubFragment` - Enable/disable nested fragment inclusion

**TODO Items:**
- Style heading with blue color (#003366) and 18pt font
- Add company logo image (logo.png, 200x100px)
- Style contact table with light gray background (#F5F5F5)
- Add horizontal line with blue color and 3pt thickness

---

### 📄 fragment2.md - Product Catalog Fragment
**Purpose:** Dynamic product catalog with pricing, images, and recommendations
**Features Demonstrated:**
- For loops with indexed iteration
- Nested tables (tables within tables)
- Complex expressions and calculations
- HTML formatting hints
- Conditional formatting
- Advanced functions (format, currency, sum, map, etc.)
- Can include fragment3 for deepest nesting

**Key Variables:**
- `quarter`, `year` - Time period for catalog
- `products[]` - Array of product objects with code, name, category, price, stock, features, discounts
- `electronicsCategories[]` - Nested category data
- `pricingTiers[]` - Tiered pricing information
- `recommendations[]` - Product recommendations
- `includeFooter` - Enable/disable fragment3 inclusion

**TODO Items:**
- Style heading with green color (#006633)
- Add product images for each item (300x200px)
- Style tables with alternating row colors
- Apply conditional formatting for prices
- Add decorative footer image (800x100px)

---

### 📄 fragment3.md - Legal & Compliance Footer Fragment
**Purpose:** Comprehensive legal footer with GDPR compliance, terms, and metadata
**Features Demonstrated:**
- Deeply nested conditionals
- Complex table structures
- Date formatting with multiple formats
- String functions (uppercase, join, joinAnd, replace)
- Switch statements for status display
- Unless statements for negative conditions
- Multi-level data structures
- Revision history tracking

**Key Variables:**
- `requiresFullTerms` - Show comprehensive vs. standard terms
- `companyName`, `companyWebsite` - Company information
- `isConfidential`, `acceptsLiability` - Legal flags
- `includeGDPR` - Toggle GDPR section
- `dataController` - GDPR contact information
- `multiJurisdiction` - Enable multi-country legal info
- `jurisdictions[]` - Array of legal jurisdictions
- `certifications[]` - Compliance certifications
- `legalAddress`, `legalContact` - Contact details
- `revisionHistory[]` - Document version history

**TODO Items:**
- Style with gray background (#F5F5F5) and italic text
- Add legal scales icon (50x50px)
- Highlight liability section with yellow background and red border
- Add EU flag icon (30x20px) for GDPR section
- Style metadata section with small font (9pt)
- Add certification logos (100x100px each)
- Style contact box with red left border (5pt)
- Add gradient footer line

---

## 📄 comprehensive_features_with_fragments.md - Main Template

**Purpose:** Master template that demonstrates all fragment capabilities including 3-level nesting
**Nesting Structure:**
```
Main Template
├── {{include "fragment1"}} (Company Header)
│   └── {{include "fragment2"}} (Product Catalog)
│       └── {{include "fragment3"}} (Legal Footer)
```

**Features Demonstrated:**
- All basic template features (variables, expressions, control structures)
- Fragment inclusion at multiple levels
- Data sharing across nested fragments
- Complex data structures passed through all levels

---

## 🔄 Converting to DOCX

The `.md` files are provided as templates that need to be converted to `.docx` format for use with go-stencil.

**To use these fragments:**

1. **Convert Markdown to DOCX:**
   - Open each `.md` file in Microsoft Word, Google Docs, or LibreOffice
   - Apply the styling indicated by TODO comments
   - Add images at the specified locations
   - Save as `.docx` format in the same directory

2. **Apply Styling:**
   - Follow the TODO comments in each file for:
     - Font colors and sizes
     - Background colors
     - Table formatting
     - Borders and lines
     - Image placement and sizing

3. **Add Images:**
   - Prepare images according to the specifications in TODO comments
   - Insert at marked locations
   - Apply alignment and sizing as specified

4. **Test the Fragments:**
   ```bash
   cd examples/advanced
   go run main.go
   ```
   - This will run Example 9: Nested Fragments Showcase
   - Output will be saved to `output/nested_fragments_output.docx`

---

## 📊 Styling Guide

### Color Palette
- **Primary Blue:** #003366 (headings, borders)
- **Secondary Green:** #006633 (product section)
- **Light Gray:** #F5F5F5, #E8E8E8 (backgrounds)
- **Light Blue:** #E6F3FF, #F0F8FF (callouts)
- **Light Yellow:** #FFFACD (highlights)
- **Light Pink:** #FFE6E6 (warnings)

### Typography
- **Headings:** 18pt (main), 14pt (sub), 12pt (minor)
- **Body Text:** 11pt
- **Small Text:** 9pt (metadata, footers)
- **Fonts:** Professional sans-serif (Arial, Calibri) or serif (Times New Roman)

### Tables
- **Header Row:** Bold text, colored background
- **Borders:** 1pt solid, varying colors
- **Padding:** 5-10pt for cells
- **Alternating Rows:** Light colored backgrounds for readability

### Images
- **Logos:** 200x100px (header)
- **Icons:** 30x30px to 50x50px
- **Product Images:** 300x200px
- **Decorative:** 800x100px (full width)
- **Certifications:** 100x100px

---

## 🎯 Example Usage

### Simple Fragment Usage
```go
tmpl, _ := engine.PrepareFile("template.docx")
tmpl.AddFragment("header", "Simple text fragment")
// Or load from DOCX
headerBytes, _ := os.ReadFile("fragments/fragment1.docx")
tmpl.AddFragmentFromBytes("header", headerBytes)
```

### Nested Fragments
```go
// Main template contains: {{include "fragment1"}}
// fragment1 contains: {{include "fragment2"}}
// fragment2 contains: {{include "fragment3"}}
// All fragments share the same data context
```

### With Full Data
See `nestedFragmentsExample()` in `main.go` for a complete example with all required data structures.

---

## 🧪 Features Tested

Each fragment tests different go-stencil capabilities:

✅ **fragment1:**
- Conditional display
- Table structures
- Dynamic timestamps
- Variable substitution
- Single-level nesting

✅ **fragment2:**
- For loops with indexes
- Nested tables
- Complex expressions
- HTML formatting
- Advanced functions
- Two-level nesting

✅ **fragment3:**
- Deep conditionals
- Multi-level data structures
- Date formatting
- String manipulation
- Switch statements
- Three-level nesting (deepest)

✅ **All Fragments:**
- Style preservation (via TODO markers)
- Image placeholders
- Complex data types (arrays, maps, nested objects)
- Error handling
- Edge cases

---

## 📝 Notes

- All template syntax uses `{{}}` brackets
- Functions require parentheses: `{{uppercase(name)}}` not `{{uppercase name}}`
- Fragment inclusion is paragraph-level (cannot be inline)
- Data context is shared across all nested fragments
- Maximum nesting depth is configurable (default: 10 levels)
- Circular references are automatically detected and prevented
- Styles from fragments are merged into the main document

---

## 🐛 Troubleshooting

**Fragment not found error:**
- Ensure the fragment name matches exactly (case-sensitive)
- Check that `AddFragment()` or `AddFragmentFromBytes()` was called
- Verify the fragment file exists if loading from DOCX

**Circular reference error:**
- Check your fragment inclusion chain
- Ensure fragment1 doesn't include itself (directly or indirectly)

**Styling not preserved:**
- Verify you converted .md to .docx properly
- Check that styles.xml is present in the fragment DOCX
- Ensure styling is applied in Word/LibreOffice before saving

**Images not showing:**
- Confirm images are embedded in the DOCX, not linked
- Check that media files were included when saving the DOCX
- Verify image formats are supported (PNG, JPG, etc.)

---

## 📚 Additional Resources

- [Go-Stencil Documentation](../../README.md)
- [Template Syntax Guide](../../docs/SYNTAX.md)
- [Functions Reference](../../docs/FUNCTIONS.md)
- [Examples](../../docs/EXAMPLES.md)

---

**Generated for go-stencil v1.0.0**
