package officepreview

import (
	"os"
	"strings"
	"testing"
)

func TestDocxSDTRendersInlineRunContent(t *testing.T) {
	t.Parallel()
	p, err := parseDocxXML([]byte(`<w:p xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:sdt>
    <w:sdtPr><w:showingPlcHdr/></w:sdtPr>
    <w:sdtContent><w:r><w:t>Score #</w:t></w:r></w:sdtContent>
  </w:sdt>
</w:p>`))
	if err != nil {
		t.Fatal(err)
	}
	sdt := p.child("sdt")
	html := renderDocxSDT(sdt, &docxRenderCtx{styles: &docxStyleSheet{styles: map[string]*docxStyleDef{}}, theme: &docxTheme{}}, nil)
	if !strings.Contains(html, "Score #") {
		t.Fatalf("html = %q", html)
	}
}

func TestDocxPPrCSSParagraphBorder(t *testing.T) {
	t.Parallel()
	pPr, err := parseDocxXML([]byte(`<w:pPr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="2E75B6"/></w:pBdr>
</w:pPr>`))
	if err != nil {
		t.Fatal(err)
	}
	css := docxPPrCSS(pPr, &docxTheme{})
	if !strings.Contains(css, "border-bottom") {
		t.Fatalf("css = %q", css)
	}
	if !strings.Contains(css, "#2E75B6") {
		t.Fatalf("css = %q", css)
	}
}

func TestDocxPPrCSSAlignment(t *testing.T) {
	t.Parallel()
	pPr, err := parseDocxXML([]byte(`<w:pPr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:jc w:val="center"/>
  <w:spacing w:after="100"/>
</w:pPr>`))
	if err != nil {
		t.Fatal(err)
	}
	css := docxPPrCSS(pPr, &docxTheme{})
	if !strings.Contains(css, "text-align:center") {
		t.Fatalf("css = %q", css)
	}
	if !strings.Contains(css, "margin-bottom") {
		t.Fatalf("css = %q", css)
	}
}

func TestDocxRPrCSSBoldAndColor(t *testing.T) {
	t.Parallel()
	rPr, err := parseDocxXML([]byte(`<w:rPr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:b/>
  <w:color w:val="666666"/>
  <w:sz w:val="28"/>
</w:rPr>`))
	if err != nil {
		t.Fatal(err)
	}
	css := docxRPrCSS(rPr, &docxTheme{})
	if !strings.Contains(css, "font-weight:700") {
		t.Fatalf("css = %q", css)
	}
	if !strings.Contains(css, "color:#666666") {
		t.Fatalf("css = %q", css)
	}
	if !strings.Contains(css, "font-size:") {
		t.Fatalf("css = %q", css)
	}
}

func TestDocxHeadingStyleResolution(t *testing.T) {
	t.Parallel()
	sheet := &docxStyleSheet{
		styles: map[string]*docxStyleDef{
			"Heading1": {styleID: "Heading1", name: "heading 1", outlineLvl: 0, rPr: mustDocxNode(`<w:rPr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:color w:val="2E74B5"/></w:rPr>`)},
		},
	}
	p, err := parseDocxXML([]byte(`<w:p xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
  <w:r><w:t>Title</w:t></w:r>
</w:p>`))
	if err != nil {
		t.Fatal(err)
	}
	resolved := sheet.resolveParagraph(p, &docxTheme{}, &docxNumbering{}, map[string][]int64{})
	if resolved.tag != "h1" {
		t.Fatalf("tag = %q, want h1", resolved.tag)
	}
}

func TestDocxWorksheetRendersVisualHTML(t *testing.T) {
	path := "../../../../data/course-files/managed-files/C-VIDVCN/7c3257e0-ccd9-45b2-8517-bd33b75fc02a.docx"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("sample docx not available:", err)
	}
	html, err := ConvertToHTML(data, "worksheet.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Project Familiarity Worksheet",
		"Project Discovery",
		"docx-page",
		"font-weight:700",
		"color:#666666",
		"border-bottom",
		"#2E75B6",
		"#CCCCCC",
		"docx-footer",
		"Part 1",
		"Part 3",
		"D5E8F0",
		"Arial",
		"min-height",
		" of 2 ",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in rendered html", want)
		}
	}
	if strings.Count(html, `class="docx-page"`) != 2 {
		t.Fatalf("expected 2 pages, got %d", strings.Count(html, `class="docx-page"`))
	}
	if strings.Count(html, `<div class="docx-footer">`) != 2 {
		t.Fatalf("expected footer on each page, got %d", strings.Count(html, `<div class="docx-footer">`))
	}
	if strings.Contains(html, "This document has no previewable text") {
		t.Fatal("fell back to empty markdown path")
	}
}

func TestDocxComplexTableRenders(t *testing.T) {
	path := "../../../../data/course-files/managed-files/C-VIDVCN/17af3c91-1d6a-4f83-af96-5876e8bcf8d5.docx"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("sample docx not available:", err)
	}
	html, err := ConvertToHTML(data, "rubric.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"docx-table",
		"Student Academic",
		"Self-Assessment",
		"Score #",
		"Content details",
		"Behavior",
		"Subject:",
		"#E4EDEB",
		"#355D7E",
		"#7BA79D",
		`<img class="docx-image"`,
		"color:#FFFFFF",
		"Teacher",
		"Use the Self-Assessment",
		"vertical-align:super",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in rendered html", want)
		}
	}
	if strings.Contains(html, "This document has no previewable text") {
		t.Fatal("fell back to empty markdown path")
	}
}

func mustDocxNode(xmlStr string) *docxXMLNode {
	n, err := parseDocxXML([]byte(xmlStr))
	if err != nil {
		panic(err)
	}
	return n
}
