package stencil

import (
	"testing"
)

func TestCleanEmptyRuns(t *testing.T) {
	// Create a paragraph with mix of empty and non-empty runs
	para := &Paragraph{
		Runs: []Run{
			{}, // Completely empty
			{Text: &Text{Content: "Hello"}}, // Has text
			{}, // Completely empty
			{Text: &Text{Content: ""}}, // Empty text
			{Break: &Break{}}, // Has break
			{}, // Completely empty
			{Text: &Text{Content: "World"}}, // Has text
		},
	}

	cleanEmptyRuns(para)

	// Should have 3 runs: "Hello", break, "World"
	if len(para.Runs) != 3 {
		t.Errorf("Expected 3 runs after cleaning, got %d", len(para.Runs))
	}

	// Verify the remaining runs
	if para.Runs[0].Text == nil || para.Runs[0].Text.Content != "Hello" {
		t.Error("First run should be 'Hello'")
	}

	if para.Runs[1].Break == nil {
		t.Error("Second run should have a break")
	}

	if para.Runs[2].Text == nil || para.Runs[2].Text.Content != "World" {
		t.Error("Third run should be 'World'")
	}
}
