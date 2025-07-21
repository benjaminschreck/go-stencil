package stencil

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// XMLFragment represents a collection of XML elements from the xml() function
type XMLFragment struct {
	Elements []XMLElement
}

// XMLElement represents a parsed XML element
type XMLElement struct {
	Type       string            // "element", "text", or "chardata"
	Name       xml.Name          // element name (for type="element")
	Attrs      []xml.Attr        // attributes (for type="element")
	Content    []XMLElement      // child elements (for type="element")
	Text       string            // text content (for type="text" or "chardata")
}

// parseXML parses XML content using the wrapper technique from original Stencil
func parseXML(content string) (*XMLFragment, error) {
	if content == "" {
		return &XMLFragment{Elements: []XMLElement{}}, nil
	}
	
	// Wrap content in a root element for parsing (same as original Stencil)
	wrappedContent := "<a>" + content + "</a>"
	
	decoder := xml.NewDecoder(strings.NewReader(wrappedContent))
	
	// Parse the XML
	root, err := parseXMLElement(decoder, nil)
	if err != nil {
		return nil, NewParseError("XML syntax error", content, 0)
	}
	
	// Handle case where root is nil
	if root == nil {
		return &XMLFragment{Elements: []XMLElement{}}, nil
	}
	
	// Return the children of the root "a" element (unwrap)
	return &XMLFragment{Elements: root.Content}, nil
}

// parseXMLElement recursively parses XML elements into XMLElement structure
func parseXMLElement(decoder *xml.Decoder, parent *XMLElement) (*XMLElement, error) {
	var currentElement *XMLElement
	
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
			newElement := &XMLElement{
				Type:    "element",
				Name:    t.Name,
				Attrs:   t.Attr,
				Content: []XMLElement{},
			}
			
			// Recursively parse children
			_, err := parseXMLElement(decoder, newElement)
			if err != nil {
				return nil, err
			}
			
			if parent != nil {
				parent.Content = append(parent.Content, *newElement)
			} else {
				currentElement = newElement
			}
			
		case xml.EndElement:
			// If we're inside a parent and this EndElement matches the parent name,
			// then we're done with this level
			if parent != nil && parent.Name == t.Name {
				return parent, nil
			}
			
			// If we're at the root level and this is an EndElement,
			// return the current element
			if parent == nil && currentElement != nil {
				return currentElement, nil
			}
			
		case xml.CharData:
			textContent := strings.TrimSpace(string(t))
			// Only create text nodes for non-empty content
			if textContent != "" {
				textElement := XMLElement{
					Type: "text",
					Text: textContent,
				}
				if parent != nil {
					parent.Content = append(parent.Content, textElement)
				} else {
					// If we have character data at root level, create a text element
					if currentElement == nil {
						currentElement = &XMLElement{
							Type:    "container",
							Content: []XMLElement{},
						}
					}
					if currentElement.Type == "container" {
						currentElement.Content = append(currentElement.Content, textElement)
					}
				}
			}
		}
	}
	
	return currentElement, nil
}

// xmlToOOXMLFragment converts XML content to an OOXML fragment
func xmlToOOXMLFragment(content string) (*XMLFragment, error) {
	if content == "" {
		return &XMLFragment{Elements: []XMLElement{}}, nil
	}
	
	// Parse XML
	xmlFragment, err := parseXML(content)
	if err != nil {
		return nil, err
	}
	
	return xmlFragment, nil
}

// registerXMLFunction registers the xml() function
func registerXMLFunction(registry *DefaultFunctionRegistry) {
	xmlFn := NewSimpleFunction("xml", 1, 1, func(args ...interface{}) (interface{}, error) {
		// Handle nil input - assert it's a string (same as original Stencil)
		if args[0] == nil {
			return nil, fmt.Errorf("xml() function requires a string argument, got nil")
		}
		
		// Convert to string - assert it's a string (same as original Stencil)
		content, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("xml() function requires a string argument, got %T", args[0])
		}
		
		// Parse XML and convert to OOXML fragment
		xmlFragment, err := xmlToOOXMLFragment(content)
		if err != nil {
			return nil, fmt.Errorf("xml() function error: %w", err)
		}
		
		// Return as OOXML fragment
		return &OOXMLFragment{Content: xmlFragment}, nil
	})
	
	registry.RegisterFunction(xmlFn)
}