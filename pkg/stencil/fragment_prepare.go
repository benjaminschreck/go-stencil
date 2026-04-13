package stencil

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

func newTextFragment(name string, content string) (*fragment, error) {
	parsed, err := ParseDocument(strings.NewReader(wrapInDocumentXML(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse fragment content: %w", err)
	}

	frag := &fragment{
		name:     name,
		content:  content,
		parsed:   parsed,
		isDocx:   false,
		bodyPlan: compileBodyRenderPlan(parsed.Body),
	}
	frag.paragraphPlans = buildDocumentParagraphPlans(parsed)
	return frag, nil
}

func validateDocxFragmentBytes(docxBytes []byte) error {
	reader := bytes.NewReader(docxBytes)
	docxReader, err := NewDocxReader(reader, int64(len(docxBytes)))
	if err != nil {
		return fmt.Errorf("failed to parse fragment DOCX: %w", err)
	}

	docXML, err := docxReader.GetDocumentXML()
	if err != nil {
		return fmt.Errorf("failed to get fragment document.xml: %w", err)
	}

	if _, err := ParseDocument(bytes.NewReader([]byte(docXML))); err != nil {
		return fmt.Errorf("failed to parse fragment document: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		return fmt.Errorf("failed to read fragment as ZIP: %w", err)
	}

	for _, file := range zipReader.File {
		switch file.Name {
		case "word/_rels/document.xml.rels":
			if err := validateRelationshipsPart(file); err != nil {
				return err
			}
		case "word/styles.xml":
			if err := validateStylesPart(file); err != nil {
				return err
			}
		case "word/numbering.xml":
			if err := validateNumberingPart(file); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateRelationshipsPart(file *zip.File) error {
	data, err := readZipFile(file)
	if err != nil {
		return fmt.Errorf("failed to read relationships: %w", err)
	}
	var rels Relationships
	if err := xml.Unmarshal(data, &rels); err != nil {
		return fmt.Errorf("failed to parse relationships: %w", err)
	}
	return nil
}

func validateStylesPart(file *zip.File) error {
	data, err := readZipFile(file)
	if err != nil {
		return fmt.Errorf("failed to read styles.xml: %w", err)
	}
	if _, err := parseStyles(data); err != nil {
		return err
	}
	return nil
}

func validateNumberingPart(file *zip.File) error {
	data, err := readZipFile(file)
	if err != nil {
		return fmt.Errorf("failed to read numbering.xml: %w", err)
	}
	var numbering struct {
		XMLName xml.Name
	}
	if err := xml.Unmarshal(data, &numbering); err != nil {
		return fmt.Errorf("failed to parse numbering.xml: %w", err)
	}
	return nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func newLazyDocxFragment(name string, docxBytes []byte) *fragment {
	cloned := append([]byte(nil), docxBytes...)
	return &fragment{
		name:     name,
		isDocx:   true,
		docxData: cloned,
	}
}

func newResolvedFragment(name string, raw []byte) (*fragment, error) {
	if _, err := NewDocxReader(bytes.NewReader(raw), int64(len(raw))); err == nil {
		frag := newLazyDocxFragment(name, raw)
		frag.resolvedByResolver = true
		return frag, nil
	}
	frag, err := newTextFragment(name, string(raw))
	if err != nil {
		return nil, err
	}
	frag.resolvedByResolver = true
	return frag, nil
}

func (frag *fragment) ensurePrepared(mainStylesXML []byte) error {
	if frag == nil || !frag.isDocx {
		return nil
	}

	frag.prepareOnce.Do(func() {
		reader := bytes.NewReader(frag.docxData)
		docxReader, err := NewDocxReader(reader, int64(len(frag.docxData)))
		if err != nil {
			frag.prepareErr = fmt.Errorf("failed to parse fragment DOCX: %w", err)
			return
		}

		docXML, err := docxReader.GetDocumentXML()
		if err != nil {
			frag.prepareErr = fmt.Errorf("failed to get fragment document.xml: %w", err)
			return
		}

		doc, err := ParseDocument(bytes.NewReader([]byte(docXML)))
		if err != nil {
			frag.prepareErr = fmt.Errorf("failed to parse fragment document: %w", err)
			return
		}

		frag.parsed = doc
		frag.namespaces = doc.ExtractNamespaces()
		frag.bodyPlan = compileBodyRenderPlan(doc.Body)
		frag.paragraphPlans = buildDocumentParagraphPlans(doc)

		zipReader, err := zip.NewReader(bytes.NewReader(frag.docxData), int64(len(frag.docxData)))
		if err != nil {
			frag.prepareErr = fmt.Errorf("failed to read fragment as ZIP: %w", err)
			return
		}

		mediaFiles := make(map[string][]byte)
		for _, file := range zipReader.File {
			if strings.HasPrefix(file.Name, "word/media/") {
				rc, err := file.Open()
				if err != nil {
					frag.prepareErr = fmt.Errorf("failed to open media file %s: %w", file.Name, err)
					return
				}

				content, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					frag.prepareErr = fmt.Errorf("failed to read media file %s: %w", file.Name, err)
					return
				}

				mediaFiles[strings.TrimPrefix(file.Name, "word/")] = content
			}
		}
		frag.mediaFiles = mediaFiles

		var relationships []Relationship
		for _, file := range zipReader.File {
			switch file.Name {
			case "word/_rels/document.xml.rels":
				rc, err := file.Open()
				if err != nil {
					frag.prepareErr = fmt.Errorf("failed to open relationships: %w", err)
					return
				}

				relsData, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					frag.prepareErr = fmt.Errorf("failed to read relationships: %w", err)
					return
				}

				var rels Relationships
				if err := xml.Unmarshal(relsData, &rels); err != nil {
					frag.prepareErr = fmt.Errorf("failed to parse relationships: %w", err)
					return
				}
				relationships = rels.Relationship
			case "word/styles.xml":
				rc, err := file.Open()
				if err != nil {
					continue
				}
				frag.stylesXML, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					frag.stylesXML = nil
				}
			case "word/numbering.xml":
				rc, err := file.Open()
				if err != nil {
					continue
				}
				frag.numberingXML, err = io.ReadAll(rc)
				rc.Close()
				if err != nil {
					frag.numberingXML = nil
				}
			}
		}
		frag.relationships = relationships

		if err := compileFragmentMetadata(frag); err != nil {
			frag.prepareErr = fmt.Errorf("failed to compile fragment metadata: %w", err)
			return
		}

		override, ok := buildFragmentFontOverride(mainStylesXML, frag.stylesXML)
		if ok {
			frag.fontOverride = override
			frag.hasFontOverride = true
		}
	})

	return frag.prepareErr
}

func installFragmentRenderPlans(ctx *renderContext, frag *fragment) {
	if ctx == nil || frag == nil || frag.parsed == nil || frag.parsed.Body == nil {
		return
	}

	if ctx.bodyPlans != nil && frag.bodyPlan != nil {
		ctx.bodyPlans[frag.parsed.Body] = frag.bodyPlan
	}
	if ctx.paragraphPlans != nil && len(frag.paragraphPlans) > 0 {
		for para, plan := range frag.paragraphPlans {
			ctx.paragraphPlans[para] = plan
		}
	}
}
