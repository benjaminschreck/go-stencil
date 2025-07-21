package stencil

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseDataURI(t *testing.T) {
	tests := []struct {
		name        string
		dataURI     string
		wantMIME    string
		wantData    []byte
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid PNG data URI",
			dataURI:  "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
			wantMIME: "image/png",
			wantData: func() []byte {
				data, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")
				return data
			}(),
			wantErr: false,
		},
		{
			name:     "valid JPEG data URI",
			dataURI:  "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAAAAAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k=",
			wantMIME: "image/jpeg",
			wantData: func() []byte {
				data, _ := base64.StdEncoding.DecodeString("/9j/4AAQSkZJRgABAQAAAAAAAAD/2wBDAAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/2wBDAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQH/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAv/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k=")
				return data
			}(),
			wantErr: false,
		},
		{
			name:        "invalid data URI format - missing data: prefix",
			dataURI:     "image/png;base64,iVBORw0KGgo=",
			wantErr:     true,
			errContains: "invalid data URI format",
		},
		{
			name:        "invalid data URI format - missing base64 marker",
			dataURI:     "data:image/png,iVBORw0KGgo=",
			wantErr:     true,
			errContains: "missing base64 marker",
		},
		{
			name:        "unsupported mime type",
			dataURI:     "data:image/webp;base64,UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA",
			wantErr:     true,
			errContains: "unsupported image type",
		},
		{
			name:        "invalid base64 data",
			dataURI:     "data:image/png;base64,!!!invalid!!!",
			wantErr:     true,
			errContains: "invalid base64 data",
		},
		{
			name:        "empty data URI",
			dataURI:     "",
			wantErr:     true,
			errContains: "empty data URI",
		},
		{
			name:        "data URI with no image data",
			dataURI:     "data:image/png;base64,",
			wantErr:     true,
			errContains: "no image data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mimeType, data, err := parseDataURI(tt.dataURI)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseDataURI() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseDataURI() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			
			if err != nil {
				t.Errorf("parseDataURI() unexpected error = %v", err)
				return
			}
			
			if mimeType != tt.wantMIME {
				t.Errorf("parseDataURI() mimeType = %v, want %v", mimeType, tt.wantMIME)
			}
			
			if !bytesEqual(data, tt.wantData) {
				t.Errorf("parseDataURI() data length = %v, want %v", len(data), len(tt.wantData))
			}
		})
	}
}

func TestGetImageExtension(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		want     string
	}{
		{"PNG mime type", "image/png", ".png"},
		{"JPEG mime type", "image/jpeg", ".jpg"},
		{"BMP mime type", "image/bmp", ".bmp"},
		{"GIF mime type", "image/gif", ".gif"},
		{"unknown mime type", "image/webp", ".png"}, // default
		{"invalid mime type", "text/plain", ".png"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getImageExtension(tt.mimeType); got != tt.want {
				t.Errorf("getImageExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateImageFilename(t *testing.T) {
	tests := []struct {
		name      string
		mimeType  string
		index     int
		wantMatch string // Use contains match since we might have random components
	}{
		{"PNG file", "image/png", 1, "image1_"},
		{"JPEG file", "image/jpeg", 2, "image2_"},
		{"BMP file", "image/bmp", 3, "image3_"},
		{"GIF file", "image/gif", 4, "image4_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateImageFilename(tt.mimeType, tt.index)
			if !strings.Contains(got, tt.wantMatch) {
				t.Errorf("generateImageFilename() = %v, want to contain %v", got, tt.wantMatch)
			}
			ext := getImageExtension(tt.mimeType)
			if !strings.HasSuffix(got, ext) {
				t.Errorf("generateImageFilename() = %v, want to end with %v", got, ext)
			}
		})
	}
}

func TestImageRelationship(t *testing.T) {
	tests := []struct {
		name         string
		relationship Relationship
		isImage      bool
	}{
		{
			name: "image relationship",
			relationship: Relationship{
				ID:     "rId2",
				Type:   "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image",
				Target: "media/image1.png",
			},
			isImage: true,
		},
		{
			name: "non-image relationship",
			relationship: Relationship{
				ID:     "rId1",
				Type:   "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles",
				Target: "styles.xml",
			},
			isImage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isImageRelationship(tt.relationship); got != tt.isImage {
				t.Errorf("isImageRelationship() = %v, want %v", got, tt.isImage)
			}
		})
	}
}

func TestAddImageRelationship(t *testing.T) {
	rels := &Relationships{
		Relationship: []Relationship{
			{
				ID:     "rId1",
				Type:   "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles",
				Target: "styles.xml",
			},
			{
				ID:     "rId2",
				Type:   "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image",
				Target: "media/image1.png",
			},
		},
	}

	newID := addImageRelationship(rels, "media/newimage.jpg")
	
	// Check that new ID was generated
	if newID == "" {
		t.Error("addImageRelationship() returned empty ID")
	}
	
	// Check that new relationship was added
	if len(rels.Relationship) != 3 {
		t.Errorf("Expected 3 relationships after adding, got %d", len(rels.Relationship))
	}
	
	// Find and verify the new relationship
	var found bool
	for _, rel := range rels.Relationship {
		if rel.ID == newID {
			found = true
			if rel.Type != imageRelationshipType {
				t.Errorf("New relationship has wrong type: %s", rel.Type)
			}
			if rel.Target != "media/newimage.jpg" {
				t.Errorf("New relationship has wrong target: %s", rel.Target)
			}
			break
		}
	}
	
	if !found {
		t.Error("New relationship not found in relationships")
	}
}

func TestGetNextRelationshipID(t *testing.T) {
	tests := []struct {
		name string
		rels *Relationships
		want string
	}{
		{
			name: "empty relationships",
			rels: &Relationships{},
			want: "rId1",
		},
		{
			name: "existing relationships",
			rels: &Relationships{
				Relationship: []Relationship{
					{ID: "rId1"},
					{ID: "rId2"},
					{ID: "rId3"},
				},
			},
			want: "rId4",
		},
		{
			name: "non-sequential IDs",
			rels: &Relationships{
				Relationship: []Relationship{
					{ID: "rId1"},
					{ID: "rId5"},
					{ID: "rId3"},
				},
			},
			want: "rId6",
		},
		{
			name: "with non-numeric IDs",
			rels: &Relationships{
				Relationship: []Relationship{
					{ID: "rId1"},
					{ID: "rIdInvalid"},
					{ID: "rId3"},
				},
			},
			want: "rId4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNextRelationshipID(tt.rels); got != tt.want {
				t.Errorf("getNextRelationshipID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to compare byte slices
func bytesEqual(a, b []byte) bool {
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