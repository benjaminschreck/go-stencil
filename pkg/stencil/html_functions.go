package stencil

import (
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"strings"
)

// HTMLRuns represents a collection of OOXML runs generated from HTML
type HTMLRuns struct {
	Runs []HTMLRun
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
	"sup":    true,
	"sub":    true,
	"span":   true,
	"br":     true,
	"strong": true,
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
		case "b", "em", "strong":
			props.Bold = &Empty{}
		case "i":
			props.Italic = &Empty{}
		case "u":
			props.Underline = &UnderlineStyle{Val: "single"}
		case "s":
			props.Strike = &Empty{}
		case "sup":
			props.VerticalAlign = &VerticalAlign{Val: "superscript"}
		case "sub":
			props.VerticalAlign = &VerticalAlign{Val: "subscript"}
		case "span":
			// Span doesn't add any formatting
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