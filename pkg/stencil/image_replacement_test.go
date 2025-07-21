package stencil

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestProcessImageReplacements(t *testing.T) {
	// Create test image markers
	pngData, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
	jpegData, _ := base64.StdEncoding.DecodeString("/9j/4AAQSkZJRgABAQAAAAAAAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k=")
	
	// Test XML with image elements - note the exact marker format must match
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<w:body>
		<w:p>
			<w:r>
				<w:drawing>
					<a:blip r:embed="rId2"/>
				</w:drawing>
				<w:t>{{IMAGE_REPLACEMENT:image/png:` + fmt.Sprintf("%d", len(pngData)) + `}}</w:t>
			</w:r>
		</w:p>
		<w:p>
			<w:r>
				<w:pict>
					<v:imagedata r:id="rId3"/>
				</w:pict>
				<w:t>{{IMAGE_REPLACEMENT:image/jpeg:` + fmt.Sprintf("%d", len(jpegData)) + `}}</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`
	
	imageMarkers := map[string]*imageReplacementMarker{
		"img_0": {
			mimeType: "image/png",
			data:     pngData,
		},
		"img_1": {
			mimeType: "image/jpeg",
			data:     jpegData,
		},
	}

	// Process image replacements
	processedXML, replacements, err := ProcessImageReplacements([]byte(xmlContent), imageMarkers)
	if err != nil {
		t.Fatalf("ProcessImageReplacements failed: %v", err)
	}

	// Check that replacements were created
	if len(replacements) != 2 {
		t.Errorf("Expected 2 replacements, got %d", len(replacements))
	}

	// Verify that the replacement markers were removed from the XML
	processedStr := string(processedXML)
	if strings.Contains(processedStr, "{{IMAGE_REPLACEMENT:") {
		t.Error("Image replacement markers were not removed from XML")
	}

	// Check that replacements have correct relationship IDs
	foundRId2 := false
	foundRId3 := false
	for oldID, replacement := range replacements {
		if oldID == "rId2" {
			foundRId2 = true
			if replacement.MIMEType != "image/png" {
				t.Errorf("rId2 replacement has wrong MIME type: %s", replacement.MIMEType)
			}
			if !bytesEqual(replacement.Data, pngData) {
				t.Error("rId2 replacement has wrong data")
			}
		}
		if oldID == "rId3" {
			foundRId3 = true
			if replacement.MIMEType != "image/jpeg" {
				t.Errorf("rId3 replacement has wrong MIME type: %s", replacement.MIMEType)
			}
			if !bytesEqual(replacement.Data, jpegData) {
				t.Error("rId3 replacement has wrong data")
			}
		}
	}

	if !foundRId2 {
		t.Error("No replacement found for rId2")
	}
	if !foundRId3 {
		t.Error("No replacement found for rId3")
	}
}

func TestImageReplacementInTemplate(t *testing.T) {
	// Test data URI for a small red PNG image
	redPNG := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAGAQIABllJAQAAAABJRU5ErkJggg=="
	
	// Create test data with replaceImage function call
	data := TemplateData{
		"newImage": redPNG,
	}

	// Test expression evaluation
	expr, err := ParseExpression(`replaceImage(newImage)`)
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}

	result, err := expr.Evaluate(data)
	if err != nil {
		t.Fatalf("Failed to evaluate expression: %v", err)
	}

	// Verify result is an imageReplacementMarker
	marker, ok := result.(*imageReplacementMarker)
	if !ok {
		t.Fatalf("Expected *imageReplacementMarker, got %T", result)
	}

	if marker.mimeType != "image/png" {
		t.Errorf("Wrong MIME type: %s", marker.mimeType)
	}

	// Verify the data was decoded properly
	expectedData, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAGAQIABllJAQAAAABJRU5ErkJggg==")
	if !bytesEqual(marker.data, expectedData) {
		t.Error("Image data was not decoded correctly")
	}
}

func TestImageFilenameGeneration(t *testing.T) {
	tests := []struct {
		mimeType string
		index    int
		wantExt  string
	}{
		{"image/png", 1, ".png"},
		{"image/jpeg", 2, ".jpg"},
		{"image/bmp", 3, ".bmp"},
		{"image/gif", 4, ".gif"},
	}

	for _, tt := range tests {
		filename := generateImageFilename(tt.mimeType, tt.index)
		if !strings.HasSuffix(filename, tt.wantExt) {
			t.Errorf("generateImageFilename(%s, %d) = %s, want suffix %s", tt.mimeType, tt.index, filename, tt.wantExt)
		}
		if !strings.Contains(filename, "image") {
			t.Errorf("generateImageFilename(%s, %d) = %s, want 'image' in filename", tt.mimeType, tt.index, filename)
		}
	}
}