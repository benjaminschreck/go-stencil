package stencil

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFragmentResolver_ExplicitFragmentTakesPrecedence(t *testing.T) {
	reader := createTestDocx(t, `{{include "frag"}}`)

	prepared, err := prepare(reader)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	if err := prepared.AddFragment("frag", "Explicit fragment"); err != nil {
		t.Fatalf("AddFragment failed: %v", err)
	}

	resolverCalls := 0
	prepared.SetFragmentResolver(FragmentResolverFunc(func(name string) ([]byte, error) {
		resolverCalls++
		return []byte("Resolved fragment"), nil
	}))

	rendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output, err := io.ReadAll(rendered)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	content := extractTextFromDOCX(t, output)
	if !strings.Contains(content, "Explicit fragment") {
		t.Fatalf("expected explicit fragment content, got %q", content)
	}
	if resolverCalls != 0 {
		t.Fatalf("expected resolver not to be called, got %d calls", resolverCalls)
	}
}

func TestFragmentResolver_CachesResolvedDocxFragments(t *testing.T) {
	reader := createTestDocx(t, `{{include "frag"}}`)

	prepared, err := prepare(reader)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	resolverCalls := 0
	prepared.SetFragmentResolver(FragmentResolverFunc(func(name string) ([]byte, error) {
		resolverCalls++
		return createSimpleDOCX(t, "Resolved DOCX Fragment"), nil
	}))

	for i := 0; i < 2; i++ {
		rendered, err := prepared.Render(TemplateData{})
		if err != nil {
			t.Fatalf("Render run %d failed: %v", i, err)
		}
		output, err := io.ReadAll(rendered)
		if err != nil {
			t.Fatalf("ReadAll run %d failed: %v", i, err)
		}
		if !strings.Contains(extractTextFromDOCX(t, output), "Resolved DOCX Fragment") {
			t.Fatalf("render %d missing resolved fragment content", i)
		}
	}

	if resolverCalls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolverCalls)
	}
}

func TestFragmentResolver_RendersNestedResolvedFragments(t *testing.T) {
	reader := createTestDocx(t, `{{include "outer"}}`)

	prepared, err := prepare(reader)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	resolved := map[string][]byte{
		"outer": []byte(`before {{include "inner"}} after`),
		"inner": []byte("nested"),
	}
	resolverCalls := map[string]int{}
	prepared.SetFragmentResolver(FragmentResolverFunc(func(name string) ([]byte, error) {
		resolverCalls[name]++
		return resolved[name], nil
	}))

	rendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output, err := io.ReadAll(rendered)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	content := extractTextFromDOCX(t, output)
	if !strings.Contains(content, "before nested after") {
		t.Fatalf("expected nested resolved content, got %q", content)
	}
	if resolverCalls["outer"] != 1 || resolverCalls["inner"] != 1 {
		t.Fatalf("expected one resolver call per fragment, got outer=%d inner=%d", resolverCalls["outer"], resolverCalls["inner"])
	}
}

func TestRenderResources_StaticHeaderFooterCached(t *testing.T) {
	docx := createDOCXWithHeaderIncludeAndStyle(t, "Static header", "MainStyle", `<w:color w:val="0000FF"/>`)

	prepared, err := prepare(bytes.NewReader(docx))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	resources, err := prepared.template.ensureRenderResources()
	if err != nil {
		t.Fatalf("ensureRenderResources failed: %v", err)
	}

	if !bytes.Contains(resources.staticParts["word/header1.xml"], []byte("Static header")) {
		t.Fatal("expected static header to be cached")
	}
	if resources.dynamicParts["word/header1.xml"] {
		t.Fatal("expected header1.xml to be treated as static")
	}
	if resources.dynamicParts["word/document.xml"] {
		t.Fatal("expected document.xml without markers to be treated as static")
	}
}

func TestRenderResources_SplitRunMarkersStayDynamic(t *testing.T) {
	docx := createDOCXWithSplitRunMarkers([]string{"{", "{name}}"}, []string{"{{hea", "der}}"})

	prepared, err := prepare(bytes.NewReader(docx))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	resources, err := prepared.template.ensureRenderResources()
	if err != nil {
		t.Fatalf("ensureRenderResources failed: %v", err)
	}
	if !resources.dynamicParts["word/document.xml"] {
		t.Fatal("expected document.xml with split-run marker to stay dynamic")
	}
	if !resources.dynamicParts["word/header1.xml"] {
		t.Fatal("expected header1.xml with split-run marker to stay dynamic")
	}

	rendered, err := prepared.Render(TemplateData{
		"name":   "Alice",
		"header": "Header Value",
	})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output, err := io.ReadAll(rendered)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if content := extractTextFromDOCX(t, output); !strings.Contains(content, "Alice") {
		t.Fatalf("expected rendered body content, got %q", content)
	}
	headerXML := extractPartFromDOCX(t, output, "word/header1.xml")
	if !strings.Contains(headerXML, "Header Value") {
		t.Fatalf("expected rendered header content, got %q", headerXML)
	}
	if strings.Contains(headerXML, "{{") || strings.Contains(headerXML, "}}") {
		t.Fatalf("expected header markers to be rendered, got %q", headerXML)
	}
}

func TestRenderIncludedFragment_RespectsMaxRenderDepthForDOCXFragments(t *testing.T) {
	originalConfig := GetGlobalConfig()
	defer SetGlobalConfig(originalConfig)
	SetGlobalConfig(&Config{
		CacheMaxSize:   originalConfig.CacheMaxSize,
		CacheTTL:       originalConfig.CacheTTL,
		LogLevel:       originalConfig.LogLevel,
		MaxRenderDepth: 2,
		StrictMode:     originalConfig.StrictMode,
	})

	prepared, err := prepare(createTestDocx(t, `{{include "outer"}}`))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	if err := prepared.AddFragmentFromBytes("outer", createSimpleDOCX(t, `outer {{include "middle"}}`)); err != nil {
		t.Fatalf("AddFragmentFromBytes outer failed: %v", err)
	}
	if err := prepared.AddFragmentFromBytes("middle", createSimpleDOCX(t, `middle {{include "inner"}}`)); err != nil {
		t.Fatalf("AddFragmentFromBytes middle failed: %v", err)
	}
	if err := prepared.AddFragmentFromBytes("inner", createSimpleDOCX(t, `inner`)); err != nil {
		t.Fatalf("AddFragmentFromBytes inner failed: %v", err)
	}

	_, err = prepared.Render(TemplateData{})
	if err == nil {
		t.Fatal("expected render to fail for excessive DOCX fragment nesting")
	}
	if !strings.Contains(err.Error(), "maximum render depth exceeded") {
		t.Fatalf("expected maximum render depth error, got %v", err)
	}
}

func TestRenderIncludedFragment_RespectsMaxRenderDepthAcrossMixedTextAndDOCXChain(t *testing.T) {
	originalConfig := GetGlobalConfig()
	defer SetGlobalConfig(originalConfig)
	SetGlobalConfig(&Config{
		CacheMaxSize:   originalConfig.CacheMaxSize,
		CacheTTL:       originalConfig.CacheTTL,
		LogLevel:       originalConfig.LogLevel,
		MaxRenderDepth: 2,
		StrictMode:     originalConfig.StrictMode,
	})

	prepared, err := prepare(createTestDocx(t, `{{include "a"}}`))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	if err := prepared.AddFragment("a", `A {{include "b"}}`); err != nil {
		t.Fatalf("AddFragment a failed: %v", err)
	}
	if err := prepared.AddFragmentFromBytes("b", createSimpleDOCX(t, `B {{include "c"}}`)); err != nil {
		t.Fatalf("AddFragmentFromBytes b failed: %v", err)
	}
	if err := prepared.AddFragment("c", `C`); err != nil {
		t.Fatalf("AddFragment c failed: %v", err)
	}

	_, err = prepared.Render(TemplateData{})
	if err == nil {
		t.Fatal("expected render to fail for mixed text/DOCX fragment nesting")
	}
	if !strings.Contains(err.Error(), "maximum render depth exceeded") {
		t.Fatalf("expected maximum render depth error, got %v", err)
	}
}

func TestAddFragmentFromBytes_RejectsInvalidDOCXAtAddTime(t *testing.T) {
	prepared, err := prepare(createTestDocx(t, "Hello"))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	if err := prepared.AddFragmentFromBytes("bad", []byte("not a docx")); err == nil {
		t.Fatal("expected AddFragmentFromBytes to reject invalid DOCX bytes")
	}
}

func TestAddFragmentFromBytes_RejectsMalformedRelationshipsAtAddTime(t *testing.T) {
	prepared, err := prepare(createTestDocx(t, "Hello"))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	if err := prepared.AddFragmentFromBytes("bad", createDOCXWithMalformedRelationships(t, "Fragment")); err == nil {
		t.Fatal("expected AddFragmentFromBytes to reject malformed relationships")
	}
}

func TestAddFragmentFromBytes_ReplacingFragmentRefreshesMergedStyles(t *testing.T) {
	mainDoc := createDOCXWithStyle(t, `Main {{include "fragment"}}`, "MainStyle", `<w:color w:val="0000FF"/>`)

	prepared, err := prepare(bytes.NewReader(mainDoc))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	if err := prepared.AddFragmentFromBytes("fragment", createDOCXWithStyle(t, "Old", "FragStyleOld", `<w:color w:val="FF0000"/>`)); err != nil {
		t.Fatalf("AddFragmentFromBytes old failed: %v", err)
	}

	firstRendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("first Render failed: %v", err)
	}
	firstBytes, err := io.ReadAll(firstRendered)
	if err != nil {
		t.Fatalf("first ReadAll failed: %v", err)
	}
	if !containsStyle(extractStylesFromDOCX(t, firstBytes), "FragStyleOld") {
		t.Fatal("expected old fragment style in first render")
	}

	if err := prepared.AddFragmentFromBytes("fragment", createDOCXWithStyle(t, "New", "FragStyleNew", `<w:color w:val="00FF00"/>`)); err != nil {
		t.Fatalf("AddFragmentFromBytes new failed: %v", err)
	}

	secondRendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("second Render failed: %v", err)
	}
	secondBytes, err := io.ReadAll(secondRendered)
	if err != nil {
		t.Fatalf("second ReadAll failed: %v", err)
	}

	if content := extractTextFromDOCX(t, secondBytes); !strings.Contains(content, "New") {
		t.Fatalf("expected replacement fragment content, got %q", content)
	}
	styles := extractStylesFromDOCX(t, secondBytes)
	if containsStyle(styles, "FragStyleOld") {
		t.Fatalf("did not expect stale old style after fragment replacement, got %v", styles)
	}
	if !containsStyle(styles, "FragStyleNew") {
		t.Fatalf("expected new fragment style after replacement, got %v", styles)
	}
}

func TestSetFragmentResolver_EvictsPreviouslyResolvedFragments(t *testing.T) {
	prepared, err := prepare(createTestDocx(t, `{{include "frag"}}`))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	resolverACalls := 0
	prepared.SetFragmentResolver(FragmentResolverFunc(func(name string) ([]byte, error) {
		resolverACalls++
		return []byte("Resolver A"), nil
	}))

	firstRendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("first Render failed: %v", err)
	}
	firstBytes, err := io.ReadAll(firstRendered)
	if err != nil {
		t.Fatalf("first ReadAll failed: %v", err)
	}
	if content := extractTextFromDOCX(t, firstBytes); !strings.Contains(content, "Resolver A") {
		t.Fatalf("expected first resolver content, got %q", content)
	}
	if resolverACalls != 1 {
		t.Fatalf("expected first resolver to be called once, got %d", resolverACalls)
	}

	resolverBCalls := 0
	prepared.SetFragmentResolver(FragmentResolverFunc(func(name string) ([]byte, error) {
		resolverBCalls++
		return []byte("Resolver B"), nil
	}))

	secondRendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("second Render failed: %v", err)
	}
	secondBytes, err := io.ReadAll(secondRendered)
	if err != nil {
		t.Fatalf("second ReadAll failed: %v", err)
	}
	if content := extractTextFromDOCX(t, secondBytes); !strings.Contains(content, "Resolver B") {
		t.Fatalf("expected second resolver content, got %q", content)
	}
	if resolverBCalls != 1 {
		t.Fatalf("expected second resolver to be called once, got %d", resolverBCalls)
	}
}

func TestDirectoryFragmentResolver_RejectsPathTraversal(t *testing.T) {
	rootDir := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.docx")
	if err := os.WriteFile(outsideFile, createSimpleDOCX(t, "secret"), 0o644); err != nil {
		t.Fatalf("write outside fragment: %v", err)
	}

	resolver := DirectoryFragmentResolver(rootDir)

	if _, err := resolver.ResolveFragment("../" + strings.TrimSuffix(filepath.Base(outsideFile), ".docx")); err == nil {
		t.Fatal("expected traversal fragment name to be rejected")
	}
	if _, err := resolver.ResolveFragment(outsideFile); err == nil {
		t.Fatal("expected absolute fragment name to be rejected")
	}
}

func TestDirectoryFragmentResolver_RejectsSymlinkEscape(t *testing.T) {
	rootDir := t.TempDir()
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "secret.docx")
	if err := os.WriteFile(outsideFile, createSimpleDOCX(t, "secret"), 0o644); err != nil {
		t.Fatalf("write outside fragment: %v", err)
	}

	linkPath := filepath.Join(rootDir, "frag.docx")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	resolver := DirectoryFragmentResolver(rootDir)
	if _, err := resolver.ResolveFragment("frag"); err == nil {
		t.Fatal("expected symlinked fragment outside resolver root to be rejected")
	}
}

func TestFragmentResolver_RechecksMissesOnLaterRenders(t *testing.T) {
	rootDir := t.TempDir()
	prepared, err := prepare(createTestDocx(t, `{{include "frag"}}`))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	prepared.SetFragmentResolver(DirectoryFragmentResolver(rootDir))

	if _, err := prepared.Render(TemplateData{}); err == nil || !strings.Contains(err.Error(), "fragment not found") {
		t.Fatalf("expected missing fragment error on first render, got %v", err)
	}

	fragmentPath := filepath.Join(rootDir, "frag.docx")
	if err := os.WriteFile(fragmentPath, createSimpleDOCX(t, "appeared later"), 0o644); err != nil {
		t.Fatalf("write fragment: %v", err)
	}

	rendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("second Render failed: %v", err)
	}
	output, err := io.ReadAll(rendered)
	if err != nil {
		t.Fatalf("second ReadAll failed: %v", err)
	}
	if content := extractTextFromDOCX(t, output); !strings.Contains(content, "appeared later") {
		t.Fatalf("expected fragment to be resolved on later render, got %q", content)
	}
}

func TestDirectoryFragmentResolver_PicksUpRootCreatedLater(t *testing.T) {
	parentDir := t.TempDir()
	rootDir := filepath.Join(parentDir, "fragments")

	prepared, err := prepare(createTestDocx(t, `{{include "frag"}}`))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	prepared.SetFragmentResolver(DirectoryFragmentResolver(rootDir))

	if _, err := prepared.Render(TemplateData{}); err == nil || !strings.Contains(err.Error(), "fragment not found") {
		t.Fatalf("expected missing fragment error before root exists, got %v", err)
	}

	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("create fragment root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "frag.docx"), createSimpleDOCX(t, "created later"), 0o644); err != nil {
		t.Fatalf("write fragment: %v", err)
	}

	rendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("Render failed after root creation: %v", err)
	}
	output, err := io.ReadAll(rendered)
	if err != nil {
		t.Fatalf("ReadAll failed after root creation: %v", err)
	}
	if content := extractTextFromDOCX(t, output); !strings.Contains(content, "created later") {
		t.Fatalf("expected fragment from late-created root, got %q", content)
	}
}

func TestPreparedTemplate_ConcurrentFragmentAccessAcrossClonedHandles(t *testing.T) {
	prepared, err := prepare(createTestDocx(t, `{{include "frag"}}`))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	clone, ok := prepared.cloneHandle()
	if !ok {
		t.Fatal("expected cloneHandle to succeed")
	}
	defer clone.Close()

	if err := prepared.AddFragment("frag", "initial"); err != nil {
		t.Fatalf("AddFragment initial failed: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			rendered, err := clone.Render(TemplateData{})
			if err != nil {
				errCh <- err
				return
			}
			if _, err := io.ReadAll(rendered); err != nil {
				errCh <- err
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			if err := prepared.AddFragment("frag", "updated"); err != nil {
				errCh <- err
				return
			}
		}
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent fragment access failed: %v", err)
		}
	}
}

func TestRenderHeaderOrFooter_RendersSplitDelimitersWithoutContiguousMarkers(t *testing.T) {
	docx := createDOCXWithSplitRunMarkers([]string{"Body"}, []string{"{", "{header}", "}"})

	prepared, err := prepare(bytes.NewReader(docx))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	resources, err := prepared.template.ensureRenderResources()
	if err != nil {
		t.Fatalf("ensureRenderResources failed: %v", err)
	}
	if !resources.dynamicParts["word/header1.xml"] {
		t.Fatal("expected split-delimiter header to stay dynamic")
	}

	rendered, err := prepared.Render(TemplateData{"header": "Header Value"})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output, err := io.ReadAll(rendered)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	headerXML := extractPartFromDOCX(t, output, "word/header1.xml")
	if !strings.Contains(headerXML, "Header Value") {
		t.Fatalf("expected rendered header value, got %q", headerXML)
	}
	if strings.Contains(headerXML, "{{") || strings.Contains(headerXML, "}}") {
		t.Fatalf("expected header markers to be rendered, got %q", headerXML)
	}
}

func TestRenderResources_LiteralBracesStayStatic(t *testing.T) {
	docx := createDOCXWithSplitRunMarkers([]string{`{"body":1}`}, []string{`{"header":1}`})

	prepared, err := prepare(bytes.NewReader(docx))
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	defer prepared.Close()

	resources, err := prepared.template.ensureRenderResources()
	if err != nil {
		t.Fatalf("ensureRenderResources failed: %v", err)
	}
	if resources.dynamicParts["word/document.xml"] {
		t.Fatal("expected document.xml with literal braces to stay static")
	}
	if resources.dynamicParts["word/header1.xml"] {
		t.Fatal("expected header1.xml with literal braces to stay static")
	}

	rendered, err := prepared.Render(TemplateData{})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	output, err := io.ReadAll(rendered)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if headerXML := extractPartFromDOCX(t, output, "word/header1.xml"); !strings.Contains(headerXML, `{"header":1}`) {
		t.Fatalf("expected literal header braces to survive, got %q", headerXML)
	}
	if bodyText := extractTextFromDOCX(t, output); !strings.Contains(bodyText, `{"body":1}`) {
		t.Fatalf("expected literal body braces to survive, got %q", bodyText)
	}
}

func createDOCXWithSplitRunMarkers(bodyParts, headerParts []string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	rels, _ := w.Create("_rels/.rels")
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	wordRels, _ := w.Create("word/_rels/document.xml.rels")
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>
</Relationships>`)

	doc, _ := w.Create("word/document.xml")
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>
    <w:p>`+joinRuns(bodyParts)+`</w:p>
    <w:sectPr>
      <w:headerReference w:type="default" r:id="rId1"/>
    </w:sectPr>
  </w:body>
</w:document>`)

	header, _ := w.Create("word/header1.xml")
	io.WriteString(header, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:p>`+joinRuns(headerParts)+`</w:p>
</w:hdr>`)

	ct, _ := w.Create("[Content_Types].xml")
	io.WriteString(ct, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>
</Types>`)

	w.Close()
	return buf.Bytes()
}

func createDOCXWithMalformedRelationships(t *testing.T, content string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	rels, err := w.Create("_rels/.rels")
	if err != nil {
		t.Fatalf("create _rels/.rels: %v", err)
	}
	io.WriteString(rels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`)

	doc, err := w.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create word/document.xml: %v", err)
	}
	io.WriteString(doc, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>`+content+`</w:t></w:r></w:p>
  </w:body>
</w:document>`)

	wordRels, err := w.Create("word/_rels/document.xml.rels")
	if err != nil {
		t.Fatalf("create word/_rels/document.xml.rels: %v", err)
	}
	io.WriteString(wordRels, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1"`)

	ct, err := w.Create("[Content_Types].xml")
	if err != nil {
		t.Fatalf("create [Content_Types].xml: %v", err)
	}
	io.WriteString(ct, `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`)

	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func joinRuns(parts []string) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(`<w:r><w:t>`)
		b.WriteString(part)
		b.WriteString(`</w:t></w:r>`)
	}
	return b.String()
}
