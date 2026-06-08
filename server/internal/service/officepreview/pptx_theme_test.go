package officepreview

import "testing"

func TestResolveTypefaceFromTheme(t *testing.T) {
	theme := &pptxTheme{
		fonts: map[string]string{
			"+mj-lt": "Calibri Light",
			"+mn-lt": "Calibri",
		},
	}
	if got := theme.resolveTypeface("+mn-lt"); got != "Calibri" {
		t.Fatalf("resolveTypeface(+mn-lt) = %q", got)
	}
	if got := theme.resolveTypeface("Arial"); got != "Arial" {
		t.Fatalf("resolveTypeface(Arial) = %q", got)
	}
}

func TestResolveShapeLstStyleUsesMasterTitleStyle(t *testing.T) {
	titleStyle, err := parsePptxXML([]byte(`<p:titleStyle xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <a:lvl1pPr><a:defRPr sz="4400"/></a:lvl1pPr>
</p:titleStyle>`))
	if err != nil {
		t.Fatal(err)
	}
	sp, err := parsePptxXML([]byte(`<p:sp xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:nvSpPr><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr>
  <p:txBody><a:p><a:r><a:t>Hello</a:t></a:r></a:p></p:txBody>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	txBody := sp.child("txBody")
	ph := sp.findDeep("ph")
	phk := &phKey{typ: ph.attr("type"), idx: ph.attr("idx")}
	lst := resolveShapeLstStyle(sp, txBody, phk, pptxTextStyles{title: titleStyle})
	if lst == nil || lst.XMLName.Local != "titleStyle" {
		t.Fatalf("lst style = %#v", lst)
	}
}
