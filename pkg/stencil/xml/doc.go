// Package xml provides XML structure definitions and types for DOCX documents.
//
// This package contains the core XML structures used by go-stencil to parse and manipulate
// DOCX files. DOCX files are essentially ZIP archives containing XML files that define
// the document structure, content, and formatting.
//
// # Structure Organization
//
// The package is organized into logical files based on XML element types:
//
//   - types.go: Core interfaces (BodyElement, ParagraphContent, RawXMLElement) and common types
//   - document.go: Top-level Document and Body structures
//   - paragraph.go: Paragraph elements and their properties (alignment, spacing, etc.)
//   - run.go: Run elements (text runs with formatting), Text, and Break elements
//   - table.go: Table structures (Table, TableRow, TableCell) and their properties
//
// # Key Concepts
//
// BodyElement: Top-level elements that can appear in a document body (paragraphs, tables).
//
// ParagraphContent: Elements that can appear within a paragraph (runs, hyperlinks, breaks).
//
// Run: A contiguous sequence of text with consistent formatting. Runs are the atomic
// units of text formatting in DOCX files.
//
// # Usage
//
// This package is primarily used internally by the stencil package for DOCX parsing
// and rendering. Most users will interact with these types through the main stencil
// package API, which re-exports the common types.
//
// Example of working with document structure:
//
//	doc := &xml.Document{
//	    Body: xml.Body{
//	        Elements: []xml.BodyElement{
//	            &xml.Paragraph{
//	                Content: []xml.ParagraphContent{
//	                    &xml.Run{
//	                        Content: []interface{}{
//	                            &xml.Text{Value: "Hello, world!"},
//	                        },
//	                    },
//	                },
//	            },
//	        },
//	    },
//	}
//
// # XML Namespaces
//
// DOCX XML uses several namespaces:
//   - w: (word processing) - Main WordProcessingML namespace
//   - r: (relationships) - Relationships namespace
//   - a: (drawing) - DrawingML namespace
//
// These are defined in the XML tags throughout the structures.
package xml
