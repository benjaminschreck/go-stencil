package stencil

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"math"
	"strconv"
	"strings"
)

const (
	htmlDefaultTableColumnWidth = 2400
	htmlDefaultTableWidth       = 0
)

// HTMLRuns represents a collection of OOXML runs generated from HTML
type HTMLRuns struct {
	Runs []HTMLRun
}

// HTMLTable represents a DOCX table generated from HTML table markup.
type HTMLTable struct {
	Table *Table
}

// HTMLBody represents DOCX body elements generated from block-level HTML.
type HTMLBody struct {
	Elements []BodyElement
}

// HTMLRun represents a single OOXML run with specific formatting
type HTMLRun struct {
	Properties *RunProperties
	Content    []HTMLRunElement
}

// HTMLRunElement represents an element within a run (text or break)
type HTMLRunElement struct {
	Type string // "text" or "break"
	Text string // for text elements
}

// HTMLNode represents a node in the HTML parse tree
type HTMLNode struct {
	Type     string
	Content  string
	Children []*HTMLNode
	Attrs    map[string]string
}

// legalTags defines the set of supported HTML tags
var legalTags = map[string]bool{
	"b":      true,
	"em":     true,
	"i":      true,
	"u":      true,
	"s":      true,
	"strike": true,
	"sup":    true,
	"sub":    true,
	"span":   true,
	"br":     true,
	"strong": true,
	"p":      true,
	"div":    true,
	"table":  true,
	"thead":  true,
	"tbody":  true,
	"tfoot":  true,
	"tr":     true,
	"td":     true,
	"th":     true,
}

// parseHTML parses HTML content into a tree structure
func parseHTML(content string) (*HTMLNode, error) {
	if content == "" {
		return &HTMLNode{Type: "text", Content: ""}, nil
	}

	// Preprocess <br> tags to <br/> for XML compatibility
	content = strings.ReplaceAll(content, "<br>", "<br/>")

	// Wrap content in a root element for parsing
	wrappedContent := "<span>" + content + "</span>"

	decoder := xml.NewDecoder(strings.NewReader(wrappedContent))

	// Parse the XML
	root, err := parseHTMLNode(decoder, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid HTML: %w", err)
	}

	// Handle case where root is nil
	if root == nil {
		return &HTMLNode{Type: "text", Content: ""}, nil
	}

	// Return the children of the root span (unwrap)
	if len(root.Children) == 0 {
		return &HTMLNode{Type: "text", Content: ""}, nil
	}

	// If there's only one child and it's a text node, return it directly
	if len(root.Children) == 1 && root.Children[0].Type == "text" {
		return root.Children[0], nil
	}

	// Otherwise, return a container with all children
	return &HTMLNode{Type: "container", Children: root.Children}, nil
}

// parseHTMLNode recursively parses XML nodes into HTMLNode structure
func parseHTMLNode(decoder *xml.Decoder, parent *HTMLNode) (*HTMLNode, error) {
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			tagName := strings.ToLower(t.Name.Local)

			// Check if tag is legal
			if !legalTags[tagName] {
				return nil, fmt.Errorf("unsupported HTML tag: %s", tagName)
			}

			newNode := &HTMLNode{
				Type:     tagName,
				Children: []*HTMLNode{},
				Attrs:    make(map[string]string),
			}

			// Parse attributes (though we don't use them for basic formatting)
			for _, attr := range t.Attr {
				newNode.Attrs[attr.Name.Local] = attr.Value
			}

			// For self-closing br tags, don't recurse
			if tagName == "br" {
				if parent != nil {
					parent.Children = append(parent.Children, newNode)
				} else {
					return newNode, nil
				}
				continue
			}

			// For non-self-closing tags, recursively parse children
			_, err := parseHTMLNode(decoder, newNode)
			if err != nil {
				return nil, err
			}

			if parent != nil {
				parent.Children = append(parent.Children, newNode)
			} else {
				return newNode, nil
			}

		case xml.EndElement:
			// Check if this EndElement matches the current parent
			tagName := strings.ToLower(t.Name.Local)

			// If we're inside a parent and this EndElement matches the parent type,
			// then we're done with this level
			if parent != nil && parent.Type == tagName {
				return parent, nil
			}

			// If this is an EndElement for a self-closing tag like br,
			// just ignore it and continue
			if tagName == "br" {
				continue
			}

			// For other cases, this might be an error, but let's continue for now
			continue

		case xml.CharData:
			textContent := string(t)
			// Always create text nodes for any character data, even if empty/whitespace
			textNode := &HTMLNode{
				Type:    "text",
				Content: textContent,
			}
			if parent != nil {
				parent.Children = append(parent.Children, textNode)
			} else {
				return textNode, nil
			}
		}
	}

	return parent, nil
}

// htmlToOOXMLRuns converts HTML content to OOXML runs
func htmlToOOXMLRuns(content string) (*HTMLRuns, error) {
	if content == "" {
		return &HTMLRuns{Runs: []HTMLRun{}}, nil
	}

	// Parse HTML
	htmlTree, err := parseHTML(content)
	if err != nil {
		return nil, err
	}

	// Convert to runs
	runs := convertNodeToRuns(htmlTree, []string{})

	return &HTMLRuns{Runs: runs}, nil
}

func htmlToOOXMLBody(content string) (*HTMLBody, error) {
	htmlTree, err := parseHTML(content)
	if err != nil {
		return nil, err
	}

	elements := htmlNodeChildrenToBodyElements(htmlTree)
	return &HTMLBody{Elements: elements}, nil
}

func htmlNodeChildrenToBodyElements(node *HTMLNode) []BodyElement {
	if node == nil {
		return nil
	}

	children := node.Children
	if node.Type != "container" && len(children) == 0 {
		children = []*HTMLNode{node}
	}

	var elements []BodyElement
	var inlineNodes []*HTMLNode

	flushInline := func() {
		if len(inlineNodes) == 0 {
			return
		}
		inlineContainer := &HTMLNode{Type: "container", Children: inlineNodes}
		runs := convertNodeToRuns(inlineContainer, []string{})
		if htmlRunsHaveVisibleContent(runs) {
			para := paragraphFromHTMLRuns(runs)
			elements = append(elements, &para)
		}
		inlineNodes = nil
	}

	for _, child := range children {
		switch child.Type {
		case "table":
			flushInline()
			if table := htmlNodeToOOXMLTable(child); table != nil {
				elements = append(elements, table)
			}
		case "p", "div":
			flushInline()
			runs := convertNodeChildrenToRuns(child, []string{})
			if htmlRunsHaveVisibleContent(runs) {
				para := paragraphFromHTMLRuns(runs)
				elements = append(elements, &para)
			}
		default:
			inlineNodes = append(inlineNodes, child)
		}
	}
	flushInline()

	return elements
}

func htmlToOOXMLTable(content string) (*HTMLTable, error) {
	htmlTree, err := parseHTML(content)
	if err != nil {
		return nil, err
	}

	tableNode := findFirstHTMLNode(htmlTree, "table")
	if tableNode == nil {
		return nil, fmt.Errorf("no table element found")
	}

	return &HTMLTable{Table: htmlNodeToOOXMLTable(tableNode)}, nil
}

func htmlNodeToOOXMLTable(tableNode *HTMLNode) *Table {
	rows := htmlTableRows(tableNode)
	if len(rows) == 0 {
		return newHTMLTable(nil, htmlNodeStyle(tableNode))
	}

	tableStyle := htmlNodeStyle(tableNode)
	maxCols := 0
	tableRows := make([]TableRow, 0, len(rows))
	columnWidths := []int{}
	for _, rowNode := range rows {
		cells := htmlTableCells(rowNode)
		if len(cells) == 0 {
			continue
		}

		row := TableRow{Cells: make([]TableCell, 0, len(cells))}
		colCount := 0
		for _, cellNode := range cells {
			cellStyle := htmlNodeStyle(cellNode)
			colspan := htmlNodeIntAttr(cellNode, "colspan", 1)
			if colspan < 1 {
				colspan = 1
			}
			cellColumnStart := colCount
			colCount += colspan

			cellRuns := convertNodeChildrenToRuns(cellNode, []string{})
			cell := TableCell{
				Properties: &TableCellProperties{
					Width: &Width{Type: "auto", Val: 0},
				},
				Paragraphs: []Paragraph{paragraphFromHTMLRunsWithStyle(cellRuns, cellStyle, cellNode.Type == "th")},
			}
			applyHTMLCellStyle(&cell, cellStyle, cellNode.Type == "th")
			if colspan > 1 {
				cell.Properties.GridSpan = &GridSpan{Val: colspan}
			}
			if width, ok := parseHTMLWidth(cellStyle["width"]); ok && colspan == 1 {
				columnWidths = ensureIntSliceLength(columnWidths, cellColumnStart+1)
				columnWidths[cellColumnStart] = width.Val
			}
			row.Cells = append(row.Cells, cell)
		}

		if colCount > maxCols {
			maxCols = colCount
		}
		tableRows = append(tableRows, row)
	}

	return newHTMLTableWithColumnCount(tableRows, maxCols, tableStyle, columnWidths)
}

func htmlNeedsBodyRendering(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "<table") ||
		strings.Contains(lower, "<p") ||
		strings.Contains(lower, "<div")
}

func findFirstHTMLNode(node *HTMLNode, nodeType string) *HTMLNode {
	if node == nil {
		return nil
	}
	if node.Type == nodeType {
		return node
	}
	for _, child := range node.Children {
		if found := findFirstHTMLNode(child, nodeType); found != nil {
			return found
		}
	}
	return nil
}

func htmlTableRows(node *HTMLNode) []*HTMLNode {
	if node == nil {
		return nil
	}
	var rows []*HTMLNode
	for _, child := range node.Children {
		switch child.Type {
		case "tr":
			rows = append(rows, child)
		case "thead", "tbody", "tfoot":
			rows = append(rows, htmlTableRows(child)...)
		}
	}
	return rows
}

func htmlTableCells(row *HTMLNode) []*HTMLNode {
	if row == nil {
		return nil
	}
	var cells []*HTMLNode
	for _, child := range row.Children {
		if child.Type == "td" || child.Type == "th" {
			cells = append(cells, child)
		}
	}
	return cells
}

func htmlNodeIntAttr(node *HTMLNode, name string, fallback int) int {
	if node == nil || node.Attrs == nil {
		return fallback
	}
	value, ok := node.Attrs[name]
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func convertNodeChildrenToRuns(node *HTMLNode, formatPath []string) []HTMLRun {
	if node == nil {
		return []HTMLRun{}
	}
	container := &HTMLNode{Type: "container", Children: node.Children}
	return convertNodeToRuns(container, formatPath)
}

func paragraphFromHTMLRuns(htmlRuns []HTMLRun) Paragraph {
	para := Paragraph{}
	for _, htmlRun := range htmlRuns {
		for _, elem := range htmlRun.Content {
			run := Run{Properties: htmlRun.Properties}
			switch elem.Type {
			case "text":
				run.Text = &Text{Space: "preserve", Content: elem.Text}
			case "break":
				run.Break = &Break{}
			default:
				continue
			}
			para.Runs = append(para.Runs, run)
		}
	}
	if len(para.Runs) == 0 {
		para.Runs = []Run{{Text: &Text{Space: "preserve", Content: ""}}}
	}
	return para
}

func htmlRunsHaveVisibleContent(htmlRuns []HTMLRun) bool {
	for _, htmlRun := range htmlRuns {
		for _, elem := range htmlRun.Content {
			if elem.Type == "break" {
				return true
			}
			if strings.TrimSpace(elem.Text) != "" {
				return true
			}
		}
	}
	return false
}

func parseHTMLStyle(style string) map[string]string {
	result := map[string]string{}
	for _, declaration := range strings.Split(style, ";") {
		name, value, ok := strings.Cut(declaration, ":")
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		if name == "" || value == "" {
			continue
		}
		result[name] = value
	}
	return result
}

func htmlNodeStyle(node *HTMLNode) map[string]string {
	if node == nil || node.Attrs == nil {
		return map[string]string{}
	}
	return parseHTMLStyle(node.Attrs["style"])
}

func parseHTMLWidth(value string) (*Width, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return nil, false
	}
	if strings.HasSuffix(value, "%") {
		n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "%")), 64)
		if err != nil || n < 0 {
			return nil, false
		}
		return &Width{Type: "pct", Val: int(math.Round(n * 50))}, true
	}
	if strings.HasSuffix(value, "px") {
		n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "px")), 64)
		if err != nil || n < 0 {
			return nil, false
		}
		return &Width{Type: "dxa", Val: int(math.Round(n * 15))}, true
	}
	if strings.HasSuffix(value, "pt") {
		n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "pt")), 64)
		if err != nil || n < 0 {
			return nil, false
		}
		return &Width{Type: "dxa", Val: int(math.Round(n * 20))}, true
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return nil, false
	}
	return &Width{Type: "dxa", Val: n}, true
}

func normalizeHTMLColor(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(value, "#") {
		value = strings.TrimPrefix(value, "#")
	}
	value = strings.ToUpper(value)
	switch value {
	case "BLACK":
		return "000000", true
	case "WHITE":
		return "FFFFFF", true
	case "RED":
		return "FF0000", true
	case "GREEN":
		return "008000", true
	case "BLUE":
		return "0000FF", true
	case "YELLOW":
		return "FFFF00", true
	case "GRAY", "GREY":
		return "808080", true
	case "TRANSPARENT", "NONE":
		return "", false
	}
	if len(value) == 3 && isHexColor(value) {
		return strings.Repeat(value[:1], 2) + strings.Repeat(value[1:2], 2) + strings.Repeat(value[2:], 2), true
	}
	if len(value) == 6 && isHexColor(value) {
		return value, true
	}
	return "", false
}

func isHexColor(value string) bool {
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func parseHTMLBorder(style map[string]string) (*BorderProperties, bool) {
	borderValue := strings.ToLower(strings.TrimSpace(style["border"]))
	if borderValue == "" {
		borderValue = strings.ToLower(strings.TrimSpace(style["border-width"]))
	}
	if borderValue == "" {
		return nil, false
	}
	if borderValue == "none" || borderValue == "0" || borderValue == "0px" {
		return &BorderProperties{Val: "nil"}, true
	}

	border := &BorderProperties{Val: "single", Sz: "4", Space: "0", Color: "auto"}
	fields := strings.Fields(borderValue)
	for _, field := range fields {
		switch field {
		case "none", "hidden":
			border.Val = "nil"
		case "solid":
			border.Val = "single"
		case "dashed":
			border.Val = "dashed"
		case "dotted":
			border.Val = "dotted"
		default:
			if size, ok := parseHTMLBorderSize(field); ok {
				border.Sz = size
				continue
			}
			if color, ok := normalizeHTMLColor(field); ok {
				border.Color = color
			}
		}
	}
	if color, ok := normalizeHTMLColor(style["border-color"]); ok {
		border.Color = color
	}
	if size, ok := parseHTMLBorderSize(style["border-width"]); ok {
		border.Sz = size
	}
	return border, true
}

func parseHTMLBorderSize(value string) (string, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", false
	}
	var points float64
	var err error
	switch {
	case strings.HasSuffix(value, "px"):
		points, err = strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "px")), 64)
	case strings.HasSuffix(value, "pt"):
		points, err = strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "pt")), 64)
	default:
		points, err = strconv.ParseFloat(value, 64)
	}
	if err != nil || points < 0 {
		return "", false
	}
	return strconv.Itoa(max(1, int(math.Round(points*8)))), true
}

func allTableBorders(border *BorderProperties) *TableBorders {
	if border == nil {
		return nil
	}
	return &TableBorders{
		Top:     cloneBorderProperties(border),
		Left:    cloneBorderProperties(border),
		Bottom:  cloneBorderProperties(border),
		Right:   cloneBorderProperties(border),
		InsideH: cloneBorderProperties(border),
		InsideV: cloneBorderProperties(border),
	}
}

func allCellBorders(border *BorderProperties) *TableCellBorders {
	if border == nil {
		return nil
	}
	return &TableCellBorders{
		Top:    cloneBorderProperties(border),
		Bottom: cloneBorderProperties(border),
		Left:   cloneBorderProperties(border),
		Right:  cloneBorderProperties(border),
	}
}

func cloneBorderProperties(border *BorderProperties) *BorderProperties {
	if border == nil {
		return nil
	}
	clone := *border
	return &clone
}

func applyHTMLCellStyle(cell *TableCell, style map[string]string, header bool) {
	if cell.Properties == nil {
		cell.Properties = &TableCellProperties{}
	}
	if width, ok := parseHTMLWidth(style["width"]); ok {
		cell.Properties.Width = width
	}
	background := style["background-color"]
	if background == "" {
		background = style["background"]
	}
	if color, ok := normalizeHTMLColor(background); ok {
		cell.Properties.Shading = &Shading{Val: "clear", Color: "auto", Fill: color}
	} else if header {
		cell.Properties.Shading = &Shading{Val: "clear", Color: "auto", Fill: "EDEDED"}
	}
	if border, ok := parseHTMLBorder(style); ok {
		cell.Properties.TcBorders = allCellBorders(border)
	}
	switch strings.ToLower(strings.TrimSpace(style["vertical-align"])) {
	case "top":
		cell.Properties.VAlign = &VerticalAlign{Val: "top"}
	case "middle", "center":
		cell.Properties.VAlign = &VerticalAlign{Val: "center"}
	case "bottom":
		cell.Properties.VAlign = &VerticalAlign{Val: "bottom"}
	}
}

func paragraphFromHTMLRunsWithStyle(htmlRuns []HTMLRun, style map[string]string, header bool) Paragraph {
	para := paragraphFromHTMLRuns(htmlRuns)
	if header {
		for i := range para.Runs {
			if para.Runs[i].Properties == nil {
				para.Runs[i].Properties = &RunProperties{}
			}
			para.Runs[i].Properties.Bold = &Empty{}
		}
	}
	if alignment := htmlTextAlignment(style["text-align"]); alignment != "" {
		para.Properties = &ParagraphProperties{
			Alignment: &Alignment{Val: alignment},
		}
	}
	return para
}

func htmlTextAlignment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "left", "center", "right", "both":
		return strings.ToLower(strings.TrimSpace(value))
	case "justify":
		return "both"
	default:
		return ""
	}
}

func ensureIntSliceLength(values []int, length int) []int {
	for len(values) < length {
		values = append(values, 0)
	}
	return values
}

func newHTMLTable(rows []TableRow, style map[string]string) *Table {
	return newHTMLTableWithColumnCount(rows, 0, style, nil)
}

func newHTMLTableWithColumnCount(rows []TableRow, columnCount int, style map[string]string, columnWidths []int) *Table {
	if columnCount < 1 {
		for _, row := range rows {
			count := 0
			for _, cell := range row.Cells {
				count += getCellSpan(&cell)
			}
			if count > columnCount {
				columnCount = count
			}
		}
	}
	if columnCount < 1 {
		columnCount = 1
	}

	columns := make([]GridColumn, columnCount)
	for i := range columns {
		width := htmlDefaultTableColumnWidth
		if i < len(columnWidths) && columnWidths[i] > 0 {
			width = columnWidths[i]
		}
		columns[i] = GridColumn{Width: width}
	}

	border := &BorderProperties{Val: "single", Sz: "4", Space: "0", Color: "auto"}
	if parsedBorder, ok := parseHTMLBorder(style); ok {
		border = parsedBorder
	}
	width := &Width{Type: "auto", Val: htmlDefaultTableWidth}
	if parsedWidth, ok := parseHTMLWidth(style["width"]); ok {
		width = parsedWidth
	}
	return &Table{
		Properties: &TableProperties{
			Width:   width,
			Borders: allTableBorders(border),
			CellMargins: &TableCellMargins{
				Top:    &CellMargin{Width: 80, Type: "dxa"},
				Left:   &CellMargin{Width: 80, Type: "dxa"},
				Bottom: &CellMargin{Width: 80, Type: "dxa"},
				Right:  &CellMargin{Width: 80, Type: "dxa"},
			},
		},
		Grid: &TableGrid{Columns: columns},
		Rows: rows,
	}
}

// convertNodeToRuns recursively converts HTML nodes to OOXML runs
func convertNodeToRuns(node *HTMLNode, formatPath []string) []HTMLRun {
	if node == nil {
		return []HTMLRun{}
	}

	// First, collect all elements with their format paths
	elements := collectElements(node, formatPath)

	// Then group them by formatting
	return groupElementsByFormatting(elements)
}

// ElementWithPath represents an element with its formatting path
type ElementWithPath struct {
	Type    string   // "text" or "break"
	Content string   // text content (for text elements)
	Path    []string // formatting path
}

// collectElements recursively collects all elements (text and breaks) with their formatting paths
func collectElements(node *HTMLNode, formatPath []string) []ElementWithPath {
	if node == nil {
		return []ElementWithPath{}
	}

	switch node.Type {
	case "text":
		if node.Content == "" {
			return []ElementWithPath{}
		}
		return []ElementWithPath{{
			Type:    "text",
			Content: html.UnescapeString(node.Content),
			Path:    formatPath,
		}}

	case "br":
		return []ElementWithPath{{
			Type:    "break",
			Content: "",
			Path:    formatPath,
		}}

	case "container":
		// Process all children with current format path
		var allElements []ElementWithPath
		for _, child := range node.Children {
			elements := collectElements(child, formatPath)
			allElements = append(allElements, elements...)
		}
		return allElements

	default:
		// For formatting tags, add to format path and process children
		newPath := append(formatPath, node.Type)
		var allElements []ElementWithPath
		for _, child := range node.Children {
			elements := collectElements(child, newPath)
			allElements = append(allElements, elements...)
		}
		return allElements
	}
}

// groupElementsByFormatting groups consecutive elements with the same formatting into runs
// For the test expectations, breaks are inline within runs (not separate runs)
func groupElementsByFormatting(elements []ElementWithPath) []HTMLRun {
	if len(elements) == 0 {
		return []HTMLRun{}
	}

	var runs []HTMLRun
	var currentContent []HTMLRunElement
	var currentPath []string

	for _, element := range elements {
		// Start a new run if:
		// 1. No current run started yet
		// 2. Formatting path changed (and we're not dealing with a break)
		if element.Type != "break" && (len(currentPath) == 0 || !pathsEqual(currentPath, element.Path)) {

			// Finish the current run if it has content
			if len(currentContent) > 0 {
				runs = append(runs, HTMLRun{
					Properties: pathToRunProperties(currentPath),
					Content:    currentContent,
				})
			}

			// Start a new run
			currentPath = element.Path
			currentContent = []HTMLRunElement{}
		}

		// Add elements to current run
		switch element.Type {
		case "text":
			currentContent = append(currentContent, HTMLRunElement{
				Type: element.Type,
				Text: element.Content,
			})
		case "break":
			// Add break inline to current content
			currentContent = append(currentContent, HTMLRunElement{
				Type: "break",
				Text: "",
			})
		}
	}

	// Finish the last run
	if len(currentContent) > 0 {
		runs = append(runs, HTMLRun{
			Properties: pathToRunProperties(currentPath),
			Content:    currentContent,
		})
	}

	return runs
}

// pathsEqual compares two formatting paths for equality
func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// pathToRunProperties converts a formatting path to RunProperties
func pathToRunProperties(path []string) *RunProperties {
	props := &RunProperties{}

	for _, tag := range path {
		switch tag {
		case "b", "strong":
			props.Bold = &Empty{}
		case "i", "em":
			props.Italic = &Empty{}
		case "u":
			props.Underline = &UnderlineStyle{Val: "single"}
		case "s", "strike":
			props.Strike = &Empty{}
		case "sup":
			props.VerticalAlign = &VerticalAlign{Val: "superscript"}
		case "sub":
			props.VerticalAlign = &VerticalAlign{Val: "subscript"}
		case "span", "p", "div":
			// These tags group content but don't add direct run formatting.
		}
	}

	return props
}

// registerHTMLFunction registers the html() function
func registerHTMLFunction(registry *DefaultFunctionRegistry) {
	htmlFn := NewSimpleFunction("html", 1, 1, func(args ...interface{}) (interface{}, error) {
		// Handle nil input
		if args[0] == nil {
			return nil, nil
		}

		// Convert to string
		content := FormatValue(args[0])

		if htmlNeedsBodyRendering(content) {
			htmlBody, err := htmlToOOXMLBody(content)
			if err != nil {
				return nil, fmt.Errorf("html() function error: %w", err)
			}
			return &OOXMLFragment{Content: htmlBody}, nil
		}

		// Parse HTML and convert to OOXML runs
		htmlRuns, err := htmlToOOXMLRuns(content)
		if err != nil {
			return nil, fmt.Errorf("html() function error: %w", err)
		}

		// Return as OOXML fragment
		return &OOXMLFragment{Content: htmlRuns}, nil
	})

	registry.RegisterFunction(htmlFn)
}
