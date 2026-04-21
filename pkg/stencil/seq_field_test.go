package stencil

import (
	"strconv"
	"testing"
)

func TestRenumberSEQFieldsInElements_RewritesRenderedResultsAcrossRepeatedFragments(t *testing.T) {
	body := []BodyElement{
		&Paragraph{Runs: seqFieldRuns("Anlage K", "Beweise", "1")},
		&Paragraph{Runs: seqFieldRuns("Anlage K", "Beweise", "1")},
		&Paragraph{Runs: seqFieldRuns("Anlage K", "Beweise", "1")},
	}

	renumberSEQFieldsInElements(body)

	for idx, elem := range body {
		para := elem.(*Paragraph)
		got := fieldResultText(para)
		want := strconv.Itoa(idx + 1)
		if got != want {
			t.Fatalf("paragraph %d field result = %q, want %q", idx, got, want)
		}
		if countRawRunElementMatches(para, "<w:fldChar") != 3 {
			t.Fatalf("paragraph %d lost fldChar elements", idx)
		}
		if countRawRunElementMatches(para, "<w:instrText") != 3 {
			t.Fatalf("paragraph %d lost instrText elements", idx)
		}
	}
}

func TestRenumberSEQFieldsInElements_KeepsIndependentCountersPerIdentifier(t *testing.T) {
	body := []BodyElement{
		&Paragraph{Runs: seqFieldRuns("Anlage K", "Beweise", "1")},
		&Paragraph{Runs: seqFieldRuns("Anlage K", "Anlage", "7")},
		&Paragraph{Runs: seqFieldRuns("Anlage K", "Beweise", "1")},
	}

	renumberSEQFieldsInElements(body)

	if got := fieldResultText(body[0].(*Paragraph)); got != "1" {
		t.Fatalf("first Beweise result = %q, want %q", got, "1")
	}
	if got := fieldResultText(body[1].(*Paragraph)); got != "1" {
		t.Fatalf("Anlage result = %q, want %q", got, "1")
	}
	if got := fieldResultText(body[2].(*Paragraph)); got != "2" {
		t.Fatalf("second Beweise result = %q, want %q", got, "2")
	}
}

func seqFieldRuns(prefix, identifier, visible string) []Run {
	return []Run{
		textRun(prefix),
		rawRun(`<w:fldChar w:fldCharType="begin"/>`),
		rawRun(`<w:instrText xml:space="preserve"> SEQ </w:instrText>`),
		rawRun(`<w:instrText>` + identifier + `</w:instrText>`),
		rawRun(`<w:instrText xml:space="preserve"> \* ARABIC </w:instrText>`),
		rawRun(`<w:fldChar w:fldCharType="separate"/>`),
		textRun(visible),
		rawRun(`<w:fldChar w:fldCharType="end"/>`),
	}
}

func fieldResultText(para *Paragraph) string {
	afterSeparate := false
	for idx := range para.Runs {
		run := para.Runs[idx]
		if countRawRunElementMatches(&Paragraph{Runs: []Run{run}}, `w:fldCharType="separate"`) > 0 {
			afterSeparate = true
			continue
		}
		if countRawRunElementMatches(&Paragraph{Runs: []Run{run}}, `w:fldCharType="end"`) > 0 {
			break
		}
		if afterSeparate && run.Text != nil && run.Text.Content != "" {
			return run.Text.Content
		}
	}
	return ""
}
