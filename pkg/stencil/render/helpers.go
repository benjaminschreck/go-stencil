package render

import (
	"reflect"

	"github.com/benjaminschreck/go-stencil/pkg/stencil/xml"
)

// runPropertiesEquivalent checks if two run properties are equivalent for merging purposes
// This is important to preserve formatting like bold, italic, etc.
func runPropertiesEquivalent(p1, p2 *xml.RunProperties) bool {
	// If both are nil, they're equivalent
	if p1 == nil && p2 == nil {
		return true
	}

	// If one is nil and the other isn't, they're not equivalent
	if (p1 == nil) != (p2 == nil) {
		return false
	}

	// Use reflect.DeepEqual to compare the properties
	// This will check all fields including Bold, Italic, Underline, etc.
	return reflect.DeepEqual(p1, p2)
}

// MergeConsecutiveRuns merges consecutive runs in a paragraph to handle split template variables
func MergeConsecutiveRuns(para *xml.Paragraph) {
	// Handle paragraphs with Content (which can contain hyperlinks)
	if len(para.Content) > 0 {
		mergeConsecutiveRunsWithContent(para)
		return
	}

	// Handle legacy paragraphs with only Runs array
	if len(para.Runs) <= 1 {
		return
	}

	var mergedRuns []xml.Run
	var currentRun *xml.Run

	for i, run := range para.Runs {
		if i == 0 {
			// Start with first run
			newRun := run
			currentRun = &newRun
			continue
		}

		// Check if this run can be merged with the previous one
		// Only merge if:
		// 1. Both runs have text and no break
		// 2. Both runs have equivalent formatting properties
		if run.Text != nil && run.Break == nil && currentRun != nil && currentRun.Text != nil && runPropertiesEquivalent(run.Properties, currentRun.Properties) {
			// Merge text content
			currentRun.Text.Content += run.Text.Content
			// Preserve xml:space="preserve" if either run has it
			// This is important for preserving leading/trailing spaces
			if run.Text.Space == "preserve" || currentRun.Text.Space == "preserve" {
				currentRun.Text.Space = "preserve"
			}
		} else {
			// Save current run and start new one
			if currentRun != nil {
				mergedRuns = append(mergedRuns, *currentRun)
			}

			// If this run has both break and text, we need to split it
			if run.Break != nil && run.Text != nil {
				// First add the break
				breakRun := xml.Run{
					Properties: run.Properties,
					Break:      run.Break,
				}
				mergedRuns = append(mergedRuns, breakRun)

				// Then create a new run with just the text
				textRun := xml.Run{
					Properties: run.Properties,
					Text:       run.Text,
				}
				currentRun = &textRun
			} else {
				newRun := run
				currentRun = &newRun
			}
		}
	}

	// Don't forget the last run
	if currentRun != nil {
		mergedRuns = append(mergedRuns, *currentRun)
	}

	para.Runs = mergedRuns
}

// mergeConsecutiveRunsWithContent merges runs while preserving hyperlink boundaries
func mergeConsecutiveRunsWithContent(para *xml.Paragraph) {
	if len(para.Content) == 0 {
		return
	}

	var mergedContent []interface{}
	var pendingRuns []xml.Run

	// Helper function to merge pending runs
	mergePendingRuns := func() {
		if len(pendingRuns) == 0 {
			return
		}

		merged := mergeRunSlice(pendingRuns)
		for _, run := range merged {
			r := run // Create a new variable to avoid aliasing
			mergedContent = append(mergedContent, &r)
		}
		pendingRuns = nil
	}

	// Process content elements
	for _, content := range para.Content {
		switch c := content.(type) {
		case *xml.Run:
			// Accumulate runs outside hyperlinks
			pendingRuns = append(pendingRuns, *c)

		case *xml.Hyperlink:
			// First, merge any pending runs
			mergePendingRuns()

			// Process hyperlink runs separately
			if len(c.Runs) > 1 {
				mergedHyperlinkRuns := mergeRunSlice(c.Runs)
				h := *c // Copy hyperlink
				h.Runs = mergedHyperlinkRuns
				mergedContent = append(mergedContent, &h)
			} else {
				mergedContent = append(mergedContent, c)
			}
		}
	}

	// Merge any remaining runs
	mergePendingRuns()

	// Update paragraph content
	para.Content = make([]xml.ParagraphContent, len(mergedContent))
	for i, content := range mergedContent {
		para.Content[i] = content.(xml.ParagraphContent)
	}

	// Also update the legacy Runs array for compatibility
	para.Runs = nil
	for _, content := range para.Content {
		if run, ok := content.(*xml.Run); ok {
			para.Runs = append(para.Runs, *run)
		}
	}
}

// mergeRunSlice merges a slice of runs
func mergeRunSlice(runs []xml.Run) []xml.Run {
	if len(runs) <= 1 {
		return runs
	}

	var merged []xml.Run
	var current *xml.Run

	for i, run := range runs {
		if i == 0 {
			newRun := run
			current = &newRun
			continue
		}

		// Only merge text runs without breaks that have equivalent properties
		if run.Text != nil && run.Break == nil && current != nil && current.Text != nil && runPropertiesEquivalent(run.Properties, current.Properties) {
			current.Text.Content += run.Text.Content
			// Preserve xml:space="preserve" if either run has it
			if run.Text.Space == "preserve" || current.Text.Space == "preserve" {
				current.Text.Space = "preserve"
			}
		} else {
			if current != nil {
				merged = append(merged, *current)
			}

			// Handle runs with both break and text
			if run.Break != nil && run.Text != nil {
				// First add the break
				breakRun := xml.Run{
					Properties: run.Properties,
					Break:      run.Break,
				}
				merged = append(merged, breakRun)

				// Then create a new run with just the text
				textRun := xml.Run{
					Properties: run.Properties,
					Text:       run.Text,
				}
				current = &textRun
			} else {
				newRun := run
				current = &newRun
			}
		}
	}

	if current != nil {
		merged = append(merged, *current)
	}

	return merged
}
