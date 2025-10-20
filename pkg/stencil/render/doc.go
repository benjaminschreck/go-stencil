// Package render provides helper functions for DOCX template rendering.
//
// This package contains pure helper functions extracted from the main rendering
// pipeline to improve code organization and maintainability. These functions handle
// specific aspects of template rendering without depending on the main stencil package,
// avoiding circular dependencies.
//
// # Structure Organization
//
// The package is organized into logical files based on functionality:
//
//   - helpers.go: Run merging utilities for consecutive text runs
//   - control.go: Control structure detection and text extraction from paragraphs
//   - body.go: Body element processing (finding control structures in element lists)
//   - table.go: Table-specific processing (finding control structures in table rows)
//
// # Key Functions
//
// MergeConsecutiveRuns: Combines consecutive text runs to simplify processing and
// improve performance. This is essential for handling template tokens that may be
// split across multiple runs due to Word's internal formatting.
//
// DetectControlStructure: Identifies control structure markers ({{for}}, {{if}}, etc.)
// in paragraph text, enabling the template engine to process loops and conditionals.
//
// FindMatchingEnd: Locates the closing {{end}} marker for control structures,
// supporting nested structures and proper scope handling.
//
// # Design Principles
//
// Pure Functions: All functions in this package are pure helpers that:
//   - Do not maintain state
//   - Do not call back into the stencil package
//   - Work with xml package types directly
//   - Can be tested independently
//
// No Circular Dependencies: This package imports xml types but is imported by
// the main stencil package. It does NOT import stencil, ensuring clean architecture.
//
// # Usage
//
// These functions are used internally by the stencil rendering pipeline and are also
// exported for testing and potential third-party use. Most users will interact with
// these through the main stencil package API.
//
// Example of using run merging:
//
//	runs := []*xml.Run{
//	    {Content: []interface{}{&xml.Text{Value: "Hello "}}},
//	    {Content: []interface{}{&xml.Text{Value: "world"}}},
//	}
//	merged := render.MergeConsecutiveRuns(runs)
//	// Result: single run containing "Hello world"
//
// Example of detecting control structures:
//
//	para := &xml.Paragraph{
//	    Content: []xml.ParagraphContent{
//	        &xml.Run{Content: []interface{}{&xml.Text{Value: "{{for item in items}}"}}},
//	    },
//	}
//	ctrl := render.DetectControlStructure(para)
//	// Result: ctrl.Type == "for", ctrl.Variable == "item", ctrl.Expression == "items"
package render
