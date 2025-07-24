package stencil

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestDocxReader_Read(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *bytes.Buffer
		wantErr bool
		check   func(t *testing.T, dr *DocxReader)
	}{
		{
			name: "read valid docx with document.xml",
			setup: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				
				// Add document.xml
				f, _ := w.Create("word/document.xml")
				f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><document>content</document>`))
				
				// Add _rels/.rels
				f, _ = w.Create("_rels/.rels")
				f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Relationships></Relationships>`))
				
				w.Close()
				return buf
			},
			wantErr: false,
			check: func(t *testing.T, dr *DocxReader) {
				if dr == nil {
					t.Fatal("expected non-nil DocxReader")
				}
				if len(dr.Parts) == 0 {
					t.Error("expected parts to be loaded")
				}
			},
		},
		{
			name: "read empty zip file",
			setup: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				w.Close()
				return buf
			},
			wantErr: true,
		},
		{
			name: "read non-zip file",
			setup: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				buf.WriteString("not a zip file")
				return buf
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tt.setup()
			reader := bytes.NewReader(buf.Bytes())
			
			dr, err := NewDocxReader(reader, int64(buf.Len()))
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDocxReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(t, dr)
			}
		})
	}
}

func TestDocxReader_GetDocumentXML(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *bytes.Buffer
		want    string
		wantErr bool
	}{
		{
			name: "get document.xml content",
			setup: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				
				f, _ := w.Create("word/document.xml")
				content := `<?xml version="1.0" encoding="UTF-8"?><document>Hello {{name}}</document>`
				f.Write([]byte(content))
				
				w.Close()
				return buf
			},
			want:    `<?xml version="1.0" encoding="UTF-8"?><document>Hello {{name}}</document>`,
			wantErr: false,
		},
		{
			name: "missing document.xml",
			setup: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				
				// Add other files but not document.xml
				f, _ := w.Create("word/styles.xml")
				f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><styles/>`))
				
				w.Close()
				return buf
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tt.setup()
			reader := bytes.NewReader(buf.Bytes())
			
			dr, err := NewDocxReader(reader, int64(buf.Len()))
			if err != nil {
				// If NewDocxReader fails, we expect wantErr to be true
				if !tt.wantErr {
					t.Fatalf("NewDocxReader() error = %v", err)
				}
				return
			}
			
			got, err := dr.GetDocumentXML()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDocumentXML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if got != tt.want {
				t.Errorf("GetDocumentXML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDocxReader_GetRelationships(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *bytes.Buffer
		want    []Relationship
		wantErr bool
	}{
		{
			name: "read document relationships",
			setup: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				
				// Need to add document.xml first
				f, _ := w.Create("word/document.xml")
				f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><document/>`))
				
				f, _ = w.Create("word/_rels/document.xml.rels")
				content := `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
	<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com" TargetMode="External"/>
</Relationships>`
				f.Write([]byte(content))
				
				w.Close()
				return buf
			},
			want: []Relationship{
				{
					ID:     "rId1",
					Type:   "http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles",
					Target: "styles.xml",
				},
				{
					ID:         "rId2",
					Type:       "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink",
					Target:     "https://example.com",
					TargetMode: "External",
				},
			},
			wantErr: false,
		},
		{
			name: "missing relationships file",
			setup: func() *bytes.Buffer {
				buf := new(bytes.Buffer)
				w := zip.NewWriter(buf)
				
				f, _ := w.Create("word/document.xml")
				f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><document/>`))
				
				w.Close()
				return buf
			},
			want:    []Relationship{},
			wantErr: false, // Missing relationships is not an error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tt.setup()
			reader := bytes.NewReader(buf.Bytes())
			
			dr, err := NewDocxReader(reader, int64(buf.Len()))
			if err != nil {
				t.Fatalf("NewDocxReader() error = %v", err)
			}
			
			got, err := dr.GetRelationships("word/document.xml")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRelationships() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if len(got) != len(tt.want) {
				t.Errorf("GetRelationships() returned %d relationships, want %d", len(got), len(tt.want))
				return
			}
			
			for i, rel := range got {
				if rel.ID != tt.want[i].ID ||
					rel.Type != tt.want[i].Type ||
					rel.Target != tt.want[i].Target ||
					rel.TargetMode != tt.want[i].TargetMode {
					t.Errorf("GetRelationships()[%d] = %+v, want %+v", i, rel, tt.want[i])
				}
			}
		})
	}
}