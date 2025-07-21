package stencil

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// DocxReader handles reading and parsing DOCX files
type DocxReader struct {
	reader *zip.Reader
	Parts  map[string]*zip.File
}

// DocumentPart represents a part of the DOCX package
type DocumentPart struct {
	Name    string
	Content []byte
}

// Relationship represents a relationship in the DOCX package
type Relationship struct {
	ID         string `xml:"Id,attr"`
	Type       string `xml:"Type,attr"`
	Target     string `xml:"Target,attr"`
	TargetMode string `xml:"TargetMode,attr,omitempty"`
}

// Relationships represents the collection of relationships
type Relationships struct {
	XMLName      xml.Name       `xml:"Relationships"`
	Namespace    string         `xml:"xmlns,attr"`
	Relationship []Relationship `xml:"Relationship"`
}

// NewDocxReader creates a new DOCX reader
func NewDocxReader(r io.ReaderAt, size int64) (*DocxReader, error) {
	zipReader, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip file: %w", err)
	}

	dr := &DocxReader{
		reader: zipReader,
		Parts:  make(map[string]*zip.File),
	}

	// Index all parts by name
	for _, file := range zipReader.File {
		dr.Parts[file.Name] = file
	}

	// Check if this is a valid DOCX file by looking for required parts
	if _, ok := dr.Parts["word/document.xml"]; !ok {
		return nil, fmt.Errorf("not a valid DOCX file: missing word/document.xml")
	}

	return dr, nil
}

// GetDocumentXML retrieves the content of word/document.xml
func (dr *DocxReader) GetDocumentXML() (string, error) {
	file, ok := dr.Parts["word/document.xml"]
	if !ok {
		return "", fmt.Errorf("document.xml not found")
	}

	rc, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open document.xml: %w", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read document.xml: %w", err)
	}

	return string(content), nil
}

// GetRelationshipsXML retrieves the content of word/_rels/document.xml.rels
func (dr *DocxReader) GetRelationshipsXML() (string, error) {
	file, ok := dr.Parts["word/_rels/document.xml.rels"]
	if !ok {
		return "", fmt.Errorf("document.xml.rels not found")
	}

	rc, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open document.xml.rels: %w", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("failed to read document.xml.rels: %w", err)
	}

	return string(content), nil
}

// GetRelationships retrieves relationships for a given part
func (dr *DocxReader) GetRelationships(partName string) ([]Relationship, error) {
	// Convert part name to its relationships file name
	// e.g., "word/document.xml" -> "word/_rels/document.xml.rels"
	dir := ""
	base := partName
	if idx := strings.LastIndex(partName, "/"); idx != -1 {
		dir = partName[:idx]
		base = partName[idx+1:]
	}

	relPath := fmt.Sprintf("%s/_rels/%s.rels", dir, base)
	if dir == "" {
		relPath = fmt.Sprintf("_rels/%s.rels", base)
	}

	file, ok := dr.Parts[relPath]
	if !ok {
		// Missing relationships file is not an error, just return empty
		return []Relationship{}, nil
	}

	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open relationships file: %w", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read relationships file: %w", err)
	}

	var rels Relationships
	if err := xml.Unmarshal(content, &rels); err != nil {
		return nil, fmt.Errorf("failed to parse relationships: %w", err)
	}

	return rels.Relationship, nil
}

// GetPart retrieves the content of a specific part
func (dr *DocxReader) GetPart(partName string) ([]byte, error) {
	file, ok := dr.Parts[partName]
	if !ok {
		return nil, fmt.Errorf("part %s not found", partName)
	}

	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open part %s: %w", partName, err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read part %s: %w", partName, err)
	}

	return content, nil
}

// ListParts returns a list of all part names in the DOCX
func (dr *DocxReader) ListParts() []string {
	parts := make([]string, 0, len(dr.Parts))
	for name := range dr.Parts {
		parts = append(parts, name)
	}
	return parts
}

// DocxReaderFromFile creates a DocxReader from a file path
func DocxReaderFromFile(path string) (*DocxReader, error) {
	// Read the entire file into memory
	// In a production system, we might want to use os.Open and os.Stat
	// for better memory efficiency with large files
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	reader := bytes.NewReader(content)
	return NewDocxReader(reader, int64(len(content)))
}