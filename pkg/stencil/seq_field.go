package stencil

import (
	"bytes"
	"encoding/xml"
	"strconv"
	"strings"
)

type seqFieldRenumberState struct {
	counters map[string]int
	stack    []*seqFieldFrame
}

type seqFieldFrame struct {
	instruction strings.Builder
	inResult    bool
	resultRuns  []*Run
}

type seqFieldFormat int

const (
	seqFieldFormatArabic seqFieldFormat = iota
	seqFieldFormatRomanUpper
	seqFieldFormatRomanLower
)

type seqFieldSpec struct {
	identifier string
	format     seqFieldFormat
}

func renumberSEQFieldsInElements(elements []BodyElement) {
	if len(elements) == 0 {
		return
	}

	state := &seqFieldRenumberState{
		counters: make(map[string]int),
	}
	for _, elem := range elements {
		renumberSEQFieldsInBodyElement(elem, state)
	}
}

func renumberSEQFieldsInBodyElement(elem BodyElement, state *seqFieldRenumberState) {
	switch e := elem.(type) {
	case *Paragraph:
		renumberSEQFieldsInParagraph(e, state)
	case *Table:
		for rowIdx := range e.Rows {
			for cellIdx := range e.Rows[rowIdx].Cells {
				for paraIdx := range e.Rows[rowIdx].Cells[cellIdx].Paragraphs {
					renumberSEQFieldsInParagraph(&e.Rows[rowIdx].Cells[cellIdx].Paragraphs[paraIdx], state)
				}
			}
		}
	}
}

func renumberSEQFieldsInParagraph(para *Paragraph, state *seqFieldRenumberState) {
	if para == nil || state == nil {
		return
	}

	for _, run := range paragraphRunsInOrder(para) {
		renumberSEQFieldsInRun(run, state)
	}
}

func paragraphRunsInOrder(para *Paragraph) []*Run {
	if para == nil {
		return nil
	}

	var runs []*Run
	if len(para.Content) > 0 {
		for _, content := range para.Content {
			switch c := content.(type) {
			case *Run:
				runs = append(runs, c)
			case *Hyperlink:
				for runIdx := range c.Runs {
					runs = append(runs, &c.Runs[runIdx])
				}
			}
		}
		return runs
	}

	for runIdx := range para.Runs {
		runs = append(runs, &para.Runs[runIdx])
	}
	for linkIdx := range para.Hyperlinks {
		for runIdx := range para.Hyperlinks[linkIdx].Runs {
			runs = append(runs, &para.Hyperlinks[linkIdx].Runs[runIdx])
		}
	}
	return runs
}

func renumberSEQFieldsInRun(run *Run, state *seqFieldRenumberState) {
	if run == nil || state == nil {
		return
	}

	if current := currentSEQFieldFrame(state); current != nil && current.inResult && run.Text != nil {
		current.resultRuns = append(current.resultRuns, run)
	}

	for _, raw := range run.RawXML {
		kind, value, ok := parseFieldInstructionRawXML(raw.Content)
		if !ok {
			continue
		}

		switch kind {
		case "begin":
			state.stack = append(state.stack, &seqFieldFrame{})
		case "instrText":
			if current := currentSEQFieldFrame(state); current != nil && !current.inResult {
				current.instruction.WriteString(value)
			}
		case "separate":
			if current := currentSEQFieldFrame(state); current != nil {
				current.inResult = true
			}
		case "end":
			state.finishCurrentSEQField()
		}
	}
}

func currentSEQFieldFrame(state *seqFieldRenumberState) *seqFieldFrame {
	if state == nil || len(state.stack) == 0 {
		return nil
	}
	return state.stack[len(state.stack)-1]
}

func (state *seqFieldRenumberState) finishCurrentSEQField() {
	if state == nil || len(state.stack) == 0 {
		return
	}

	idx := len(state.stack) - 1
	frame := state.stack[idx]
	state.stack = state.stack[:idx]

	spec, ok := parseSEQFieldInstruction(frame.instruction.String())
	if !ok || len(frame.resultRuns) == 0 {
		return
	}

	state.counters[spec.identifier]++
	setSEQFieldResult(frame.resultRuns, formatSEQFieldValue(state.counters[spec.identifier], spec.format))
}

func parseFieldInstructionRawXML(raw []byte) (kind, value string, ok bool) {
	if len(raw) == 0 {
		return "", "", false
	}

	decoder := xml.NewDecoder(bytes.NewReader(raw))
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", "", false
		}

		start, isStart := token.(xml.StartElement)
		if !isStart {
			continue
		}

		switch start.Name.Local {
		case "fldChar":
			for _, attr := range start.Attr {
				if attr.Name.Local == "fldCharType" {
					return attr.Value, "", true
				}
			}
			return "", "", false
		case "instrText":
			var text strings.Builder
			for {
				token, err := decoder.Token()
				if err != nil {
					return "", "", false
				}
				switch t := token.(type) {
				case xml.CharData:
					text.Write([]byte(t))
				case xml.EndElement:
					if t.Name.Local == start.Name.Local {
						return "instrText", text.String(), true
					}
				}
			}
		default:
			return "", "", false
		}
	}
}

func parseSEQFieldInstruction(instruction string) (seqFieldSpec, bool) {
	parts := strings.Fields(instruction)
	if len(parts) < 2 || !strings.EqualFold(parts[0], "SEQ") {
		return seqFieldSpec{}, false
	}

	spec := seqFieldSpec{
		identifier: parts[1],
		format:     seqFieldFormatArabic,
	}

	for idx := 2; idx < len(parts)-1; idx++ {
		if parts[idx] != `\*` {
			continue
		}
		switch parts[idx+1] {
		case "ROMAN":
			spec.format = seqFieldFormatRomanUpper
		case "roman":
			spec.format = seqFieldFormatRomanLower
		case "ARABIC", "Arabic", "arabic":
			spec.format = seqFieldFormatArabic
		}
	}

	return spec, true
}

func formatSEQFieldValue(value int, format seqFieldFormat) string {
	switch format {
	case seqFieldFormatRomanUpper:
		return intToRoman(value)
	case seqFieldFormatRomanLower:
		return strings.ToLower(intToRoman(value))
	default:
		return strconv.Itoa(value)
	}
}

func intToRoman(value int) string {
	if value <= 0 {
		return strconv.Itoa(value)
	}

	var result strings.Builder
	numerals := []struct {
		value  int
		symbol string
	}{
		{1000, "M"},
		{900, "CM"},
		{500, "D"},
		{400, "CD"},
		{100, "C"},
		{90, "XC"},
		{50, "L"},
		{40, "XL"},
		{10, "X"},
		{9, "IX"},
		{5, "V"},
		{4, "IV"},
		{1, "I"},
	}

	for _, numeral := range numerals {
		for value >= numeral.value {
			result.WriteString(numeral.symbol)
			value -= numeral.value
		}
	}

	return result.String()
}

func setSEQFieldResult(runs []*Run, value string) {
	if len(runs) == 0 {
		return
	}

	wroteValue := false
	for _, run := range runs {
		if run == nil {
			continue
		}
		if run.Text == nil {
			run.Text = &Text{}
		}
		if !wroteValue {
			run.Text.Content = value
			run.Text.Space = ""
			wroteValue = true
			continue
		}
		run.Text.Content = ""
		run.Text.Space = ""
	}
}
