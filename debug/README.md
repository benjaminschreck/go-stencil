# Debug Tools

This directory contains debugging tools for the go-stencil project.

## Tools

### examine_table_position
Examines the structure of DOCX files to debug table positioning issues.

**Usage:**
```bash
cd examine_table_position
go run main.go <docx-file>
```

**Purpose:** 
- Shows the exact order of paragraphs and tables in a DOCX file
- Helps identify when tables are being moved from their original positions
- Useful for debugging template rendering issues

### extract_docx_text
Extracts text content from DOCX files to inspect template expressions.

**Usage:**
```bash
cd extract_docx_text
go run main.go <docx-file>
```

**Purpose:**
- Extracts all text content from a DOCX file
- Shows template expressions found in the document
- Helps identify when template expressions are split across multiple runs

### test_table_html
Tests HTML rendering in table cells with a simple example.

**Usage:**
```bash
cd test_table_html
go run main.go
```

**Purpose:**
- Tests HTML function rendering within table cells
- Demonstrates proper table structure for template loops
- Creates a test output file to verify HTML formatting

## Known Issues

### HTML in Table Cells

When using HTML functions inside table cells within for loops, ensure that:

1. **Template expressions are not split across multiple runs or paragraphs**
   - Word often splits text when you edit it or apply formatting
   - This breaks template processing

2. **The table structure follows this pattern:**
   ```
   Row 1: {{for item in items}}
   Row 2: {{html(item.field)}} ... (actual content row)
   Row 3: {{end}}
   ```

3. **Each template expression must be in a single text run**

### Tips for Creating Templates

To avoid Word splitting template expressions:
- Type template expressions in one go without editing
- Don't apply formatting to parts of template expressions
- Use a plain text editor to create template content first, then paste into Word
- Avoid using spell check on template expressions
- Save immediately after typing template expressions to prevent Word from auto-formatting