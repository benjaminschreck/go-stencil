package stencil

import (
	"encoding/xml"
	"fmt"
	"path"
	"strconv"
	"strings"
)

// extractRelationshipNumber extracts the numeric ID from a relationship ID like "rId6"
func extractRelationshipNumber(rId string) (int, error) {
	if !strings.HasPrefix(rId, "rId") {
		return 0, fmt.Errorf("invalid relationship ID format: %s", rId)
	}

	numStr := strings.TrimPrefix(rId, "rId")
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid relationship ID number: %s", rId)
	}

	return num, nil
}

// renameMediaPath renames a media path to avoid conflicts
// Example: "media/image1.png" + "header" + 1 -> "media/image_header_1.png"
// Note: Always use forward slashes for DOCX internal paths (ZIP format requirement)
func renameMediaPath(originalPath, fragmentName string, counter int) string {
	dir := path.Dir(originalPath)
	ext := path.Ext(originalPath)
	newName := fmt.Sprintf("image_%s_%d%s", fragmentName, counter, ext)
	return path.Join(dir, newName)
}

// isMediaRelationship checks if a relationship is for media (images, video, etc.)
func isMediaRelationship(rel Relationship) bool {
	mediaTypes := []string{
		"image",
		"video",
		"audio",
	}

	for _, mediaType := range mediaTypes {
		if strings.Contains(strings.ToLower(rel.Type), mediaType) {
			return true
		}
	}

	return false
}

// cloneDocument creates a deep copy of a Document
func cloneDocument(doc *Document) *Document {
	if doc == nil {
		return nil
	}

	cloned := &Document{
		XMLName: doc.XMLName,
	}

	// Deep copy Attrs slice
	if doc.Attrs != nil {
		cloned.Attrs = make([]xml.Attr, len(doc.Attrs))
		copy(cloned.Attrs, doc.Attrs)
	}

	if doc.Body != nil {
		cloned.Body = cloneBody(doc.Body)
	}

	return cloned
}

// cloneBody creates a deep copy of a Body
func cloneBody(body *Body) *Body {
	if body == nil {
		return nil
	}

	cloned := &Body{}

	if body.Elements != nil {
		cloned.Elements = make([]BodyElement, len(body.Elements))
		for i, elem := range body.Elements {
			switch e := elem.(type) {
			case *Paragraph:
				cloned.Elements[i] = cloneParagraph(e)
			case *Table:
				cloned.Elements[i] = cloneTable(e)
			default:
				// For unknown types, just copy the reference
				cloned.Elements[i] = elem
			}
		}
	}

	// Clone SectionProperties if present
	if body.SectionProperties != nil {
		clonedSP := cloneRawXMLElement(body.SectionProperties)
		cloned.SectionProperties = &clonedSP
	}

	return cloned
}

// cloneParagraph creates a deep copy of a Paragraph
func cloneParagraph(para *Paragraph) *Paragraph {
	if para == nil {
		return nil
	}

	cloned := &Paragraph{
		Properties: para.Properties, // Properties can be shallow copied
	}

	if para.Attrs != nil {
		cloned.Attrs = make([]xml.Attr, len(para.Attrs))
		copy(cloned.Attrs, para.Attrs)
	}

	// Clone Content (ordered list)
	if para.Content != nil {
		cloned.Content = make([]ParagraphContent, len(para.Content))
		for i, item := range para.Content {
			switch c := item.(type) {
			case *Run:
				cloned.Content[i] = cloneRun(c)
			case *Hyperlink:
				cloned.Content[i] = cloneHyperlink(c)
			default:
				cloned.Content[i] = item
			}
		}
	}

	// Clone legacy Runs field
	if para.Runs != nil {
		cloned.Runs = make([]Run, len(para.Runs))
		for i, run := range para.Runs {
			cloned.Runs[i] = *cloneRun(&run)
		}
	}

	// Clone legacy Hyperlinks field
	if para.Hyperlinks != nil {
		cloned.Hyperlinks = make([]Hyperlink, len(para.Hyperlinks))
		for i, link := range para.Hyperlinks {
			cloned.Hyperlinks[i] = *cloneHyperlink(&link)
		}
	}

	return cloned
}

// cloneRun creates a deep copy of a Run
func cloneRun(run *Run) *Run {
	if run == nil {
		return nil
	}

	cloned := &Run{
		Properties: run.Properties, // Shallow copy is fine
		Text:       run.Text,       // Shallow copy is fine
		Break:      run.Break,      // Shallow copy is fine
	}

	if run.Attrs != nil {
		cloned.Attrs = make([]xml.Attr, len(run.Attrs))
		copy(cloned.Attrs, run.Attrs)
	}

	// Clone RawXML elements (contains images!)
	if run.RawXML != nil {
		cloned.RawXML = make([]RawXMLElement, len(run.RawXML))
		for i, raw := range run.RawXML {
			cloned.RawXML[i] = cloneRawXMLElement(&raw)
		}
	}

	return cloned
}

// cloneRawXMLElement creates a deep copy of a RawXMLElement
func cloneRawXMLElement(raw *RawXMLElement) RawXMLElement {
	if raw == nil {
		return RawXMLElement{}
	}

	cloned := RawXMLElement{
		XMLName: raw.XMLName,
	}

	// Deep copy Content bytes
	if raw.Content != nil {
		cloned.Content = make([]byte, len(raw.Content))
		copy(cloned.Content, raw.Content)
	}

	// Deep copy Attrs slice
	if raw.Attrs != nil {
		cloned.Attrs = make([]xml.Attr, len(raw.Attrs))
		copy(cloned.Attrs, raw.Attrs)
	}

	return cloned
}

// cloneHyperlink creates a deep copy of a Hyperlink
func cloneHyperlink(link *Hyperlink) *Hyperlink {
	if link == nil {
		return nil
	}

	cloned := &Hyperlink{
		ID:      link.ID,
		History: link.History,
	}

	// Clone Runs
	if link.Runs != nil {
		cloned.Runs = make([]Run, len(link.Runs))
		for i, run := range link.Runs {
			cloned.Runs[i] = *cloneRun(&run)
		}
	}

	return cloned
}

// cloneTable creates a deep copy of a Table
func cloneTable(table *Table) *Table {
	if table == nil {
		return nil
	}

	cloned := &Table{
		Properties: table.Properties, // Shallow copy is fine
	}

	// Clone Grid
	if table.Grid != nil {
		cloned.Grid = table.Grid // Shallow copy is fine
	}

	// Clone Rows
	if table.Rows != nil {
		cloned.Rows = make([]TableRow, len(table.Rows))
		for i, row := range table.Rows {
			cloned.Rows[i] = *cloneTableRow(&row)
		}
	}

	return cloned
}

// cloneTableRow creates a deep copy of a TableRow
func cloneTableRow(row *TableRow) *TableRow {
	if row == nil {
		return nil
	}

	cloned := &TableRow{
		Properties: row.Properties, // Shallow copy is fine
	}

	// Clone Cells
	if row.Cells != nil {
		cloned.Cells = make([]TableCell, len(row.Cells))
		for i, cell := range row.Cells {
			cloned.Cells[i] = *cloneTableCell(&cell)
		}
	}

	return cloned
}

// cloneTableCell creates a deep copy of a TableCell
func cloneTableCell(cell *TableCell) *TableCell {
	if cell == nil {
		return nil
	}

	cloned := &TableCell{
		Properties: cell.Properties, // Shallow copy is fine
	}

	// Clone Paragraphs
	if cell.Paragraphs != nil {
		cloned.Paragraphs = make([]Paragraph, len(cell.Paragraphs))
		for i, para := range cell.Paragraphs {
			cloned.Paragraphs[i] = *cloneParagraph(&para)
		}
	}

	return cloned
}

// updateDocumentRelationshipIDs updates all relationship IDs in a document
func updateDocumentRelationshipIDs(doc *Document, idMap map[string]string) {
	if doc == nil || doc.Body == nil {
		return
	}

	updateBodyRelationshipIDs(doc.Body, idMap)
}

// updateBodyRelationshipIDs updates relationship IDs in a body
func updateBodyRelationshipIDs(body *Body, idMap map[string]string) {
	if body == nil {
		return
	}

	for _, elem := range body.Elements {
		switch e := elem.(type) {
		case *Paragraph:
			updateParagraphRelationshipIDs(e, idMap)
		case *Table:
			updateTableRelationshipIDs(e, idMap)
		}
	}
}

// updateParagraphRelationshipIDs updates relationship IDs in a paragraph
func updateParagraphRelationshipIDs(para *Paragraph, idMap map[string]string) {
	if para == nil {
		return
	}

	// Update in Content (ordered list)
	for _, item := range para.Content {
		switch c := item.(type) {
		case *Run:
			updateRunRelationshipIDs(c, idMap)
		case *Hyperlink:
			updateHyperlinkRelationshipIDs(c, idMap)
		}
	}

	// Update in legacy Runs
	for i := range para.Runs {
		updateRunRelationshipIDs(&para.Runs[i], idMap)
	}

	// Update in legacy Hyperlinks
	for i := range para.Hyperlinks {
		updateHyperlinkRelationshipIDs(&para.Hyperlinks[i], idMap)
	}
}

// updateRunRelationshipIDs updates relationship IDs in a run
func updateRunRelationshipIDs(run *Run, idMap map[string]string) {
	if run == nil {
		return
	}

	// Update RawXML elements (contains images!)
	for i := range run.RawXML {
		updateRawXMLRelationshipIDs(&run.RawXML[i], idMap)
	}
}

// updateRawXMLRelationshipIDs updates relationship IDs in raw XML content
func updateRawXMLRelationshipIDs(raw *RawXMLElement, idMap map[string]string) {
	if raw == nil || len(raw.Content) == 0 {
		return
	}

	content := string(raw.Content)

	// Replace relationship ID references
	// Note: RawXML stores namespaces as full URIs, not as prefixes
	// Example: "http://schemas.openxmlformats.org/officeDocument/2006/relationships:embed="rId6""
	// not "r:embed="rId6""
	relationshipsNS := "http://schemas.openxmlformats.org/officeDocument/2006/relationships"

	for oldID, newID := range idMap {
		// Replace with full namespace URI (as stored in RawXML)
		content = strings.ReplaceAll(content,
			fmt.Sprintf(`%s:embed="%s"`, relationshipsNS, oldID),
			fmt.Sprintf(`%s:embed="%s"`, relationshipsNS, newID))

		// Also try r:embed (prefix form, in case it's used)
		content = strings.ReplaceAll(content,
			fmt.Sprintf(`r:embed="%s"`, oldID),
			fmt.Sprintf(`r:embed="%s"`, newID))

		// Replace r:id references (hyperlinks)
		content = strings.ReplaceAll(content,
			fmt.Sprintf(`%s:id="%s"`, relationshipsNS, oldID),
			fmt.Sprintf(`%s:id="%s"`, relationshipsNS, newID))

		content = strings.ReplaceAll(content,
			fmt.Sprintf(`r:id="%s"`, oldID),
			fmt.Sprintf(`r:id="%s"`, newID))
	}

	raw.Content = []byte(content)
}

// updateHyperlinkRelationshipIDs updates relationship IDs in a hyperlink
func updateHyperlinkRelationshipIDs(link *Hyperlink, idMap map[string]string) {
	if link == nil {
		return
	}

	// Update hyperlink's own relationship ID
	if newID, ok := idMap[link.ID]; ok {
		link.ID = newID
	}

	// Update runs within the hyperlink
	for i := range link.Runs {
		updateRunRelationshipIDs(&link.Runs[i], idMap)
	}
}

// updateTableRelationshipIDs updates relationship IDs in a table
func updateTableRelationshipIDs(table *Table, idMap map[string]string) {
	if table == nil {
		return
	}

	for i := range table.Rows {
		updateTableRowRelationshipIDs(&table.Rows[i], idMap)
	}
}

// updateTableRowRelationshipIDs updates relationship IDs in a table row
func updateTableRowRelationshipIDs(row *TableRow, idMap map[string]string) {
	if row == nil {
		return
	}

	for i := range row.Cells {
		updateTableCellRelationshipIDs(&row.Cells[i], idMap)
	}
}

// updateTableCellRelationshipIDs updates relationship IDs in a table cell
func updateTableCellRelationshipIDs(cell *TableCell, idMap map[string]string) {
	if cell == nil {
		return
	}

	// Update paragraphs in cell
	for i := range cell.Paragraphs {
		updateParagraphRelationshipIDs(&cell.Paragraphs[i], idMap)
	}
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
