package stencil

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrepare(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() io.Reader
		wantErr bool
		check   func(t *testing.T, pt *PreparedTemplate)
	}{
		{
			name: "prepare simple template",
			setup: func() io.Reader {
				buf := createTestDocx(t, "Hello {{name}}!")
				return bytes.NewReader(buf.Bytes())
			},
			wantErr: false,
			check: func(t *testing.T, pt *PreparedTemplate) {
				if pt == nil {
					t.Fatal("expected non-nil PreparedTemplate")
				}
				if pt.template == nil {
					t.Fatal("expected non-nil template")
				}
			},
		},
		{
			name: "prepare invalid docx",
			setup: func() io.Reader {
				return strings.NewReader("not a docx file")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := tt.setup()
			pt, err := Prepare(reader)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Prepare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(t, pt)
			}
			
			if pt != nil {
				pt.Close()
			}
		})
	}
}

func TestPrepareFile(t *testing.T) {
	// Create a temporary test file
	tmpfile, err := os.CreateTemp("", "test*.docx")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	
	// Write test docx content
	buf := createTestDocx(t, "Test content {{variable}}")
	tmpfile.Write(buf.Bytes())
	tmpfile.Close()
	
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "prepare from valid file",
			path:    tmpfile.Name(),
			wantErr: false,
		},
		{
			name:    "prepare from non-existent file",
			path:    "/non/existent/file.docx",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, err := PrepareFile(tt.path)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("PrepareFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if pt != nil {
				pt.Close()
			}
		})
	}
}

func TestPreparedTemplate_Render(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     TemplateData
		wantErr  bool
		check    func(t *testing.T, output io.Reader)
	}{
		{
			name:     "render simple substitution",
			template: "Hello {{name}}!",
			data: TemplateData{
				"name": "World",
			},
			wantErr: false,
			check: func(t *testing.T, output io.Reader) {
				// For now, just check that we get some output
				if output == nil {
					t.Error("expected non-nil output")
				}
			},
		},
		{
			name:     "render with nil data",
			template: "Hello {{name}}!",
			data:     nil,
			wantErr:  false,
		},
		{
			name:     "render with empty data",
			template: "Hello {{name}}!",
			data:     TemplateData{},
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := createTestDocx(t, tt.template)
			reader := bytes.NewReader(buf.Bytes())
			
			pt, err := Prepare(reader)
			if err != nil {
				t.Fatalf("Prepare() error = %v", err)
			}
			defer pt.Close()
			
			output, err := pt.Render(tt.data)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.check != nil {
				tt.check(t, output)
			}
		})
	}
}

func TestPreparedTemplate_Close(t *testing.T) {
	t.Run("can close nil template", func(t *testing.T) {
		pt := &PreparedTemplate{}
		err := pt.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	
	t.Run("can close valid template", func(t *testing.T) {
		buf := createTestDocx(t, "Test")
		reader := bytes.NewReader(buf.Bytes())
		
		pt, err := Prepare(reader)
		if err != nil {
			t.Fatalf("Prepare() error = %v", err)
		}
		
		err = pt.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}

func TestTemplateData(t *testing.T) {
	data := TemplateData{
		"name":  "John",
		"age":   30,
		"items": []string{"one", "two", "three"},
	}

	if data["name"] != "John" {
		t.Errorf("Expected name to be John")
	}

	if data["age"] != 30 {
		t.Errorf("Expected age to be 30")
	}

	items, ok := data["items"].([]string)
	if !ok || len(items) != 3 {
		t.Errorf("Expected items to be a slice of 3 strings")
	}
}

// Helper function to create a test DOCX with given content
func createTestDocx(t *testing.T, content string) *bytes.Buffer {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	
	// Add document.xml with the content
	f, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
	<w:body>
		<w:p>
			<w:r>
				<w:t>%s</w:t>
			</w:r>
		</w:p>
	</w:body>
</w:document>`, content)
	
	f.Write([]byte(xml))
	
	// Add required relationships
	f, _ = w.Create("_rels/.rels")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
	<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))
	
	w.Close()
	return buf
}