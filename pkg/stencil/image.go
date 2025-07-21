package stencil

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	imageRelationshipType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
)

// ImageReplacement represents a pending image replacement operation
type ImageReplacement struct {
	OldRelID string
	NewRelID string
	MIMEType string
	Data     []byte
	Filename string
}

// parseDataURI parses a data URI and returns the MIME type and decoded data
func parseDataURI(dataURI string) (string, []byte, error) {
	if dataURI == "" {
		return "", nil, fmt.Errorf("empty data URI")
	}

	// Data URI format: data:[<mediatype>][;base64],<data>
	if !strings.HasPrefix(dataURI, "data:") {
		return "", nil, fmt.Errorf("invalid data URI format")
	}

	// Remove the "data:" prefix
	dataURI = dataURI[5:]

	// Find the comma that separates the metadata from the data
	commaIndex := strings.Index(dataURI, ",")
	if commaIndex == -1 {
		return "", nil, fmt.Errorf("invalid data URI format")
	}

	// Extract metadata and data parts
	metadata := dataURI[:commaIndex]
	dataStr := dataURI[commaIndex+1:]

	if dataStr == "" {
		return "", nil, fmt.Errorf("no image data")
	}

	// Check for base64 encoding
	if !strings.HasSuffix(metadata, ";base64") {
		return "", nil, fmt.Errorf("missing base64 marker")
	}

	// Extract MIME type
	mimeType := strings.TrimSuffix(metadata, ";base64")

	// Validate supported image types
	switch mimeType {
	case "image/png", "image/jpeg", "image/bmp", "image/gif":
		// Supported types
	default:
		return "", nil, fmt.Errorf("unsupported image type: %s", mimeType)
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return "", nil, fmt.Errorf("invalid base64 data: %w", err)
	}

	return mimeType, data, nil
}

// getImageExtension returns the file extension for a given MIME type
func getImageExtension(mimeType string) string {
	switch mimeType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/bmp":
		return ".bmp"
	case "image/gif":
		return ".gif"
	default:
		return ".png" // default
	}
}

// generateImageFilename generates a unique filename for an image
func generateImageFilename(mimeType string, index int) string {
	ext := getImageExtension(mimeType)
	// Generate a unique suffix to avoid conflicts
	uniqueSuffix := fmt.Sprintf("%d", index)
	return fmt.Sprintf("image%d_%s%s", index, uniqueSuffix, ext)
}

// isImageRelationship checks if a relationship is an image relationship
func isImageRelationship(rel Relationship) bool {
	return rel.Type == imageRelationshipType
}

// addImageRelationship adds a new image relationship and returns its ID
func addImageRelationship(rels *Relationships, target string) string {
	newID := getNextRelationshipID(rels)
	
	newRel := Relationship{
		ID:     newID,
		Type:   imageRelationshipType,
		Target: target,
	}
	
	rels.Relationship = append(rels.Relationship, newRel)
	return newID
}

// getNextRelationshipID generates the next available relationship ID
func getNextRelationshipID(rels *Relationships) string {
	maxID := 0
	
	for _, rel := range rels.Relationship {
		if strings.HasPrefix(rel.ID, "rId") {
			idStr := rel.ID[3:]
			if id, err := strconv.Atoi(idStr); err == nil && id > maxID {
				maxID = id
			}
		}
	}
	
	return fmt.Sprintf("rId%d", maxID+1)
}

// isImageNode checks if an XML element is an image node and returns its relationship ID
func isImageNode(elem xml.StartElement) (bool, string) {
	// Check for blip elements (modern Word format)
	if elem.Name.Local == "blip" {
		for _, attr := range elem.Attr {
			if attr.Name.Local == "embed" {
				return true, attr.Value
			}
		}
	}
	
	// Check for imagedata elements (VML format)
	if elem.Name.Local == "imagedata" {
		for _, attr := range elem.Attr {
			if attr.Name.Local == "id" {
				return true, attr.Value
			}
		}
	}
	
	return false, ""
}

// updateImageRelationshipID updates the relationship ID in an image element
func updateImageRelationshipID(elem *xml.StartElement, newID string) {
	if elem.Name.Local == "blip" {
		for i, attr := range elem.Attr {
			if attr.Name.Local == "embed" {
				elem.Attr[i].Value = newID
				return
			}
		}
	}
	
	if elem.Name.Local == "imagedata" {
		for i, attr := range elem.Attr {
			if attr.Name.Local == "id" {
				elem.Attr[i].Value = newID
				return
			}
		}
	}
}

// ProcessImageReplacements processes image replacement markers in the document
// and returns a map of relationship ID to ImageReplacement
func ProcessImageReplacements(xmlContent []byte, imageMarkers map[string]*imageReplacementMarker) ([]byte, map[string]*ImageReplacement, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(xmlContent)))
	var output strings.Builder
	encoder := xml.NewEncoder(&output)
	
	replacements := make(map[string]*ImageReplacement)
	imageCounter := 0
	currentImageContext := ""
	
	for {
		tok, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("error decoding XML: %w", err)
		}
		
		switch t := tok.(type) {
		case xml.StartElement:
			// Check if this is an image element
			isImg, relID := isImageNode(t)
			if isImg {
				currentImageContext = relID
			}
			
			// Write the start element
			if err := encoder.EncodeToken(t); err != nil {
				return nil, nil, err
			}
			
		case xml.EndElement:
			if err := encoder.EncodeToken(t); err != nil {
				return nil, nil, err
			}
			
		case xml.CharData:
			content := string(t)
			
			// Check for image replacement markers
			if strings.Contains(content, "{{IMAGE_REPLACEMENT:") {
				// Parse the marker format: {{IMAGE_REPLACEMENT:mimeType:dataSize}}
				for _, marker := range imageMarkers {
					markerStr := fmt.Sprintf("{{IMAGE_REPLACEMENT:%s:%d}}", marker.mimeType, len(marker.data))
					if strings.Contains(content, markerStr) && currentImageContext != "" {
						// Create image replacement
						imageCounter++
						newFilename := generateImageFilename(marker.mimeType, imageCounter)
						newRelID := fmt.Sprintf("rIdImg%d", imageCounter)
						
						replacements[currentImageContext] = &ImageReplacement{
							OldRelID: currentImageContext,
							NewRelID: newRelID,
							MIMEType: marker.mimeType,
							Data:     marker.data,
							Filename: newFilename,
						}
						
						// Remove the marker from content
						content = strings.Replace(content, markerStr, "", -1)
						break
					}
				}
			}
			
			// Write the possibly modified content
			if err := encoder.EncodeToken(xml.CharData(content)); err != nil {
				return nil, nil, err
			}
			
		default:
			if err := encoder.EncodeToken(t); err != nil {
				return nil, nil, err
			}
		}
	}
	
	if err := encoder.Flush(); err != nil {
		return nil, nil, err
	}
	
	return []byte(output.String()), replacements, nil
}

