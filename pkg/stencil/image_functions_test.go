package stencil

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"strings"
	"testing"
)

func TestReplaceImageFunction(t *testing.T) {
	// Test data URI for a 1x1 transparent PNG
	testDataURI := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
	
	tests := []struct {
		name    string
		args    []interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid data URI",
			args:    []interface{}{testDataURI},
			wantErr: false,
		},
		{
			name:    "no arguments",
			args:    []interface{}{},
			wantErr: true,
			errMsg:  "requires exactly 1 argument",
		},
		{
			name:    "too many arguments",
			args:    []interface{}{testDataURI, "extra"},
			wantErr: true,
			errMsg:  "requires exactly 1 argument",
		},
		{
			name:    "non-string argument",
			args:    []interface{}{123},
			wantErr: true,
			errMsg:  "argument must be a string",
		},
		{
			name:    "invalid data URI",
			args:    []interface{}{"not a data uri"},
			wantErr: true,
			errMsg:  "invalid data URI format",
		},
		{
			name:    "unsupported image type",
			args:    []interface{}{"data:image/webp;base64,UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA"},
			wantErr: true,
			errMsg:  "unsupported image type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := replaceImageFunc(tt.args...)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("replaceImage() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("replaceImage() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("replaceImage() unexpected error = %v", err)
				}
				if result == nil {
					t.Error("replaceImage() returned nil result")
				}
				// Check that result is an imageReplacementMarker
				marker, ok := result.(*imageReplacementMarker)
				if !ok {
					t.Errorf("replaceImage() returned %T, want *imageReplacementMarker", result)
				} else {
					// Verify the marker contains valid data
					if marker.mimeType == "" {
						t.Error("replaceImage() returned marker with empty mimeType")
					}
					if len(marker.data) == 0 {
						t.Error("replaceImage() returned marker with empty data")
					}
				}
			}
		})
	}
}

func TestReplaceImageMarkerInExpression(t *testing.T) {
	// Test that the replaceImage function works in expressions
	testDataURI := "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAAAAAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k="
	
	// Create expression parser
	expr, err := ParseExpression(`replaceImage("` + testDataURI + `")`)
	if err != nil {
		t.Fatalf("Failed to parse expression: %v", err)
	}
	
	// Evaluate the expression
	result, err := expr.Evaluate(TemplateData{})
	if err != nil {
		t.Fatalf("Failed to evaluate expression: %v", err)
	}
	
	// Verify result is an imageReplacementMarker
	marker, ok := result.(*imageReplacementMarker)
	if !ok {
		t.Fatalf("Expression returned %T, want *imageReplacementMarker", result)
	}
	
	if marker.mimeType != "image/jpeg" {
		t.Errorf("Marker mimeType = %s, want image/jpeg", marker.mimeType)
	}
	
	// Verify the data was decoded properly
	expectedData, _ := base64.StdEncoding.DecodeString("/9j/4AAQSkZJRgABAQAAAAAAAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k=")
	if !bytes.Equal(marker.data, expectedData) {
		t.Errorf("Marker data length = %d, want %d", len(marker.data), len(expectedData))
	}
}

func TestFindImageNodes(t *testing.T) {
	tests := []struct {
		name     string
		xmlInput string
		wantIDs  []string
	}{
		{
			name: "blip with embed attribute",
			xmlInput: `<w:drawing>
				<wp:inline>
					<a:graphic>
						<a:graphicData>
							<pic:pic>
								<pic:blipFill>
									<a:blip r:embed="rId2"/>
								</pic:blipFill>
							</pic:pic>
						</a:graphicData>
					</a:graphic>
				</wp:inline>
			</w:drawing>`,
			wantIDs: []string{"rId2"},
		},
		{
			name: "imagedata with id attribute",
			xmlInput: `<v:shape>
				<v:imagedata r:id="rId3"/>
			</v:shape>`,
			wantIDs: []string{"rId3"},
		},
		{
			name: "multiple images",
			xmlInput: `<w:document>
				<w:drawing>
					<a:blip r:embed="rId2"/>
				</w:drawing>
				<w:pict>
					<v:shape>
						<v:imagedata r:id="rId3"/>
					</v:shape>
				</w:pict>
				<w:drawing>
					<a:blip r:embed="rId4"/>
				</w:drawing>
			</w:document>`,
			wantIDs: []string{"rId2", "rId3", "rId4"},
		},
		{
			name: "no images",
			xmlInput: `<w:document>
				<w:p>
					<w:r>
						<w:t>Just text, no images</w:t>
					</w:r>
				</w:p>
			</w:document>`,
			wantIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := xml.NewDecoder(strings.NewReader(tt.xmlInput))
			var foundIDs []string
			
			for {
				tok, err := decoder.Token()
				if err != nil {
					break
				}
				
				if startElem, ok := tok.(xml.StartElement); ok {
					// Check for blip elements
					if startElem.Name.Local == "blip" {
						for _, attr := range startElem.Attr {
							if attr.Name.Local == "embed" {
								foundIDs = append(foundIDs, attr.Value)
							}
						}
					}
					// Check for imagedata elements
					if startElem.Name.Local == "imagedata" {
						for _, attr := range startElem.Attr {
							if attr.Name.Local == "id" {
								foundIDs = append(foundIDs, attr.Value)
							}
						}
					}
				}
			}
			
			if len(foundIDs) != len(tt.wantIDs) {
				t.Errorf("Found %d image IDs, want %d", len(foundIDs), len(tt.wantIDs))
				return
			}
			
			for i, id := range foundIDs {
				if id != tt.wantIDs[i] {
					t.Errorf("Found ID[%d] = %s, want %s", i, id, tt.wantIDs[i])
				}
			}
		})
	}
}

func TestImageReplacementMarkerString(t *testing.T) {
	marker := &imageReplacementMarker{
		mimeType: "image/png",
		data:     make([]byte, 1024),
	}
	
	want := "[Image: image/png, 1024 bytes]"
	got := marker.String()
	
	if got != want {
		t.Errorf("imageReplacementMarker.String() = %s, want %s", got, want)
	}
}

func TestIsImageNode(t *testing.T) {
	tests := []struct {
		name    string
		element xml.StartElement
		want    bool
		wantID  string
	}{
		{
			name: "blip element with embed",
			element: xml.StartElement{
				Name: xml.Name{Local: "blip"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "embed"}, Value: "rId2"},
				},
			},
			want:   true,
			wantID: "rId2",
		},
		{
			name: "imagedata element with id",
			element: xml.StartElement{
				Name: xml.Name{Local: "imagedata"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "id"}, Value: "rId3"},
				},
			},
			want:   true,
			wantID: "rId3",
		},
		{
			name: "blip element without embed",
			element: xml.StartElement{
				Name: xml.Name{Local: "blip"},
				Attr: []xml.Attr{},
			},
			want:   false,
			wantID: "",
		},
		{
			name: "non-image element",
			element: xml.StartElement{
				Name: xml.Name{Local: "paragraph"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "id"}, Value: "p1"},
				},
			},
			want:   false,
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isImage, id := isImageNode(tt.element)
			if isImage != tt.want {
				t.Errorf("isImageNode() isImage = %v, want %v", isImage, tt.want)
			}
			if id != tt.wantID {
				t.Errorf("isImageNode() id = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestUpdateImageRelationshipID(t *testing.T) {
	tests := []struct {
		name     string
		element  xml.StartElement
		newID    string
		wantAttr string
	}{
		{
			name: "update blip embed",
			element: xml.StartElement{
				Name: xml.Name{Local: "blip"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "embed"}, Value: "rId2"},
					{Name: xml.Name{Local: "cstate"}, Value: "print"},
				},
			},
			newID:    "rId10",
			wantAttr: "embed",
		},
		{
			name: "update imagedata id",
			element: xml.StartElement{
				Name: xml.Name{Local: "imagedata"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "id"}, Value: "rId3"},
					{Name: xml.Name{Local: "title"}, Value: "Picture 1"},
				},
			},
			newID:    "rId11",
			wantAttr: "id",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the element to modify
			elem := tt.element
			
			// Update the ID
			updateImageRelationshipID(&elem, tt.newID)
			
			// Find and verify the updated attribute
			var found bool
			for _, attr := range elem.Attr {
				if attr.Name.Local == tt.wantAttr {
					found = true
					if attr.Value != tt.newID {
						t.Errorf("Attribute %s = %s, want %s", tt.wantAttr, attr.Value, tt.newID)
					}
					break
				}
			}
			
			if !found {
				t.Errorf("Attribute %s not found after update", tt.wantAttr)
			}
		})
	}
}