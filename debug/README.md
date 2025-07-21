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