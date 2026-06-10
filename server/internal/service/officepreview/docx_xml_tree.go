package officepreview

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// docxXMLNode is a lightweight WordprocessingML tree.
type docxXMLNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr    `xml:",any,attr"`
	Children []docxXMLNode `xml:",any"`
	Content  string        `xml:",chardata"`
}

func (n *docxXMLNode) attr(local string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == local && a.Value != "" {
			return a.Value
		}
	}
	return ""
}

func (n *docxXMLNode) relAttr(local string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == local && a.Value != "" && strings.Contains(a.Name.Space, "relationships") {
			return a.Value
		}
	}
	return ""
}

func (n *docxXMLNode) child(local string) *docxXMLNode {
	for i := range n.Children {
		if n.Children[i].XMLName.Local == local {
			return &n.Children[i]
		}
	}
	return nil
}

func (n *docxXMLNode) childAttr(childLocal, attrLocal string) string {
	if ch := n.child(childLocal); ch != nil {
		return ch.attr(attrLocal)
	}
	return ""
}

func (n *docxXMLNode) findDeep(local string) *docxXMLNode {
	for i := range n.Children {
		if n.Children[i].XMLName.Local == local {
			return &n.Children[i]
		}
		if found := n.Children[i].findDeep(local); found != nil {
			return found
		}
	}
	return nil
}

func (n *docxXMLNode) findAllDeep(local string) []*docxXMLNode {
	var out []*docxXMLNode
	for i := range n.Children {
		if n.Children[i].XMLName.Local == local {
			out = append(out, &n.Children[i])
		}
		out = append(out, n.Children[i].findAllDeep(local)...)
	}
	return out
}

func parseDocxXML(data []byte) (*docxXMLNode, error) {
	var root docxXMLNode
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	return &root, nil
}

func parseDocxInt(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// twipsToPx converts twentieths of a point (dxa/twips) to CSS pixels at 96 dpi.
func twipsToPx(twips int64) float64 {
	return float64(twips) * 96.0 / 1440.0
}

// halfPtToPx converts Word font size (half-points) to CSS pixels.
func halfPtToPx(halfPt int64) float64 {
	return float64(halfPt) * 96.0 / 144.0
}

func docxCollectText(n *docxXMLNode) string {
	if strings.TrimSpace(n.Content) != "" {
		return n.Content
	}
	var parts []string
	for i := range n.Children {
		if t := docxCollectText(&n.Children[i]); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "")
}

// ----- Theme -----

type docxTheme struct {
	colors map[string]string // scheme name → RRGGBB hex (no #)
}

func loadDocxTheme(zr *zip.Reader) *docxTheme {
	theme := &docxTheme{colors: make(map[string]string)}
	for _, name := range []string{"word/theme/theme1.xml", "word/theme/theme11.xml"} {
		data, err := readZipFile(zr, name)
		if err != nil {
			continue
		}
		root, err := parseDocxXML(data)
		if err != nil {
			continue
		}
		clrScheme := root.findDeep("clrScheme")
		if clrScheme == nil {
			continue
		}
		for i := range clrScheme.Children {
			schemeName := clrScheme.Children[i].XMLName.Local
			hex := docxExtractColorHex(&clrScheme.Children[i])
			if hex != "" {
				theme.colors[schemeName] = hex
			}
		}
		break
	}
	return theme
}

func docxExtractColorHex(node *docxXMLNode) string {
	if srgb := node.child("srgbClr"); srgb != nil {
		if val := srgb.attr("val"); val != "" {
			return strings.ToUpper(val)
		}
	}
	if sys := node.child("sysClr"); sys != nil {
		if val := sys.attr("lastClr"); val != "" {
			return strings.ToUpper(val)
		}
	}
	return ""
}

func (t *docxTheme) resolveScheme(name string) string {
	if t == nil || t.colors == nil {
		return ""
	}
	// Documents reference colors by their logical slot (tx2, bg1, ...) or the
	// long form (text2, background1), but the theme stores them under the
	// clrScheme element names (dk2, lt1, ...). Normalize before lookup.
	switch name {
	case "tx1", "text1":
		name = "dk1"
	case "tx2", "text2":
		name = "dk2"
	case "bg1", "background1":
		name = "lt1"
	case "bg2", "background2":
		name = "lt2"
	}
	return t.colors[name]
}

// resolveDocxColor returns a CSS color from w:color or w:themeColor nodes.
func resolveDocxColor(node *docxXMLNode, theme *docxTheme) string {
	if node == nil {
		return ""
	}
	if color := node.child("color"); color != nil {
		if val := strings.ToUpper(color.attr("val")); val != "" && val != "AUTO" {
			return "#" + val
		}
	}
	if tc := node.child("themeColor"); tc != nil {
		hex := theme.resolveScheme(tc.attr("val"))
		if hex == "" {
			return ""
		}
		r, g, b := docxHexToRGB(hex)
		if tint := tc.child("themeTint"); tint != nil {
			f := float64(parseDocxInt(tint.attr("val"))) / 255000.0
			r = uint8(float64(r) + float64(255-r)*f)
			g = uint8(float64(g) + float64(255-g)*f)
			b = uint8(float64(b) + float64(255-b)*f)
		}
		if shade := tc.child("themeShade"); shade != nil {
			f := float64(parseDocxInt(shade.attr("val"))) / 255000.0
			r = uint8(float64(r) * f)
			g = uint8(float64(g) * f)
			b = uint8(float64(b) * f)
		}
		return fmt.Sprintf("#%02X%02X%02X", r, g, b)
	}
	return ""
}

func docxHexToRGB(hex string) (uint8, uint8, uint8) {
	if len(hex) != 6 {
		return 0, 0, 0
	}
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	return uint8(r), uint8(g), uint8(b)
}

func docxShdCSS(shd *docxXMLNode, theme *docxTheme) string {
	if shd == nil || shd.attr("fill") == "" {
		return ""
	}
	fill := strings.ToUpper(shd.attr("fill"))
	if fill == "AUTO" || fill == "FFFFFF" && shd.attr("val") == "clear" {
		return ""
	}
	return "background-color:#" + fill + ";"
}

func docxTcMarCSS(tcMar *docxXMLNode) string {
	if tcMar == nil {
		return ""
	}
	var parts []string
	for _, side := range []string{"top", "left", "bottom", "right"} {
		if m := tcMar.child(side); m != nil && m.attr("type") == "dxa" {
			if w := parseDocxInt(m.attr("w")); w > 0 {
				parts = append(parts, fmt.Sprintf("padding-%s:%.2fpx", side, twipsToPx(w)))
			}
		}
	}
	return strings.Join(parts, ";")
}

func docxCnfStylePrType(cnf *docxXMLNode) string {
	if cnf == nil {
		return ""
	}
	val := cnf.attr("val")
	if len(val) != 12 {
		return ""
	}
	types := []string{
		"firstRow", "lastRow", "firstCol", "lastCol",
		"oddVBand", "evenVBand", "band1Horz", "band2Horz",
		"firstRowFirstCol", "firstRowLastCol", "lastRowFirstCol", "lastRowLastCol",
	}
	for i, typ := range types {
		if val[i] == '1' {
			return typ
		}
	}
	return ""
}

func docxPBdrCSS(pBdr *docxXMLNode) string {
	if pBdr == nil {
		return ""
	}
	var parts []string
	for _, side := range []string{"top", "left", "bottom", "right"} {
		if b := pBdr.child(side); b != nil {
			if css := docxParaBorderSideCSS(b, side); css != "" {
				parts = append(parts, css)
			}
		}
	}
	return strings.Join(parts, "")
}

func docxParaBorderSideCSS(border *docxXMLNode, side string) string {
	val := border.attr("val")
	if val == "" || val == "nil" || val == "none" {
		return ""
	}
	sz := parseDocxInt(border.attr("sz"))
	if sz <= 0 {
		sz = 4
	}
	widthPx := float64(sz) / 8.0 * 96.0 / 72.0
	if widthPx < 0.5 {
		widthPx = 0.5
	}
	clr := "#000000"
	if c := border.attr("color"); c != "" && strings.ToUpper(c) != "AUTO" {
		clr = "#" + strings.ToUpper(c)
	}
	style := "solid"
	switch val {
	case "dashed":
		style = "dashed"
	case "dotted":
		style = "dotted"
	case "double":
		style = "double"
	}
	css := fmt.Sprintf("border-%s:%.2fpx %s %s", side, widthPx, style, clr)
	if space := parseDocxInt(border.attr("space")); space > 0 {
		css += fmt.Sprintf(";padding-%s:%.2fpx", side, float64(space)*96.0/72.0)
	}
	return css
}

func docxBorderCSS(border *docxXMLNode, theme *docxTheme) string {
	if border == nil {
		return ""
	}
	val := border.attr("val")
	if val == "" || val == "nil" || val == "none" {
		return ""
	}
	width := parseDocxInt(border.attr("sz"))
	if width <= 0 {
		width = 4
	}
	// Round table/cell border widths to whole pixels. Under
	// border-collapse, fractional widths (e.g. a 0.5px table outline meeting
	// a 2.2px cell border) fail to merge cleanly in browsers and leave a
	// doubled hairline alongside the thicker border.
	widthPx := int(float64(width)/8.0 + 0.5)
	if widthPx < 1 {
		widthPx = 1
	}
	clr := "#000000"
	if c := border.child("color"); c != nil {
		if v := strings.ToUpper(c.attr("val")); v != "" && v != "AUTO" {
			clr = "#" + v
		}
	}
	style := "solid"
	switch val {
	case "dashed", "dashSmallGap":
		style = "dashed"
	case "dotted":
		style = "dotted"
	case "double":
		style = "double"
	}
	return fmt.Sprintf("border:%dpx %s %s;", widthPx, style, clr)
}

func docxBordersCSS(borders *docxXMLNode, theme *docxTheme) string {
	if borders == nil {
		return ""
	}
	var parts []string
	for _, side := range []string{"top", "left", "bottom", "right"} {
		if b := borders.child(side); b != nil {
			if v := b.attr("val"); v == "none" || v == "nil" {
				// An explicit "none" must override a border inherited from the
				// table/cell style, so emit it rather than nothing.
				parts = append(parts, "border-"+side+":none;")
			} else if css := docxBorderCSS(b, theme); css != "" {
				parts = append(parts, strings.Replace(css, "border:", "border-"+side+":", 1))
			}
		}
	}
	return strings.Join(parts, "")
}

func resolveDocxSolidFillColor(fill *docxXMLNode, theme *docxTheme) string {
	if fill == nil {
		return ""
	}
	if srgb := fill.findDeep("srgbClr"); srgb != nil {
		if val := strings.ToUpper(strings.TrimSpace(srgb.attr("val"))); len(val) == 6 {
			return "#" + val
		}
	}
	if scheme := fill.findDeep("schemeClr"); scheme != nil {
		if hex := theme.resolveScheme(scheme.attr("val")); hex != "" {
			return "#" + docxApplyColorTransforms(hex, scheme)
		}
	}
	return ""
}

// docxApplyColorTransforms applies OOXML luminance/tint/shade transforms (as
// found on schemeClr nodes) to a base hex color, mirroring the DrawingML logic
// used on the presentation side. Returns the transformed hex (no leading '#').
func docxApplyColorTransforms(hex string, node *docxXMLNode) string {
	if len(hex) != 6 || node == nil {
		return hex
	}
	r := hexByteU8(hex, 0)
	g := hexByteU8(hex, 2)
	b := hexByteU8(hex, 4)
	for i := range node.Children {
		child := &node.Children[i]
		val := float64(parseDocxInt(child.attr("val")))
		switch child.XMLName.Local {
		case "tint":
			f := val / 100000
			r = uint8(float64(r) + float64(255-r)*f)
			g = uint8(float64(g) + float64(255-g)*f)
			b = uint8(float64(b) + float64(255-b)*f)
		case "shade":
			f := val / 100000
			r = uint8(float64(r) * f)
			g = uint8(float64(g) * f)
			b = uint8(float64(b) * f)
		case "lumMod":
			h, l, s := rgbToHLS(r, g, b)
			l = clampF(l * val / 100000)
			r, g, b = hlsToRGB(h, l, s)
		case "lumOff":
			h, l, s := rgbToHLS(r, g, b)
			l = clampF(l + val/100000)
			r, g, b = hlsToRGB(h, l, s)
		}
	}
	return fmt.Sprintf("%02X%02X%02X", r, g, b)
}

func docxDrawingLnWidthPx(ln *docxXMLNode) float64 {
	if ln == nil {
		return 1.5
	}
	w := parseDocxInt(ln.attr("w"))
	if w <= 0 {
		return 1.5
	}
	return float64(w) / 12700.0 * 96.0 / 72.0
}

func docxAnchorOffsetPx(anchor *docxXMLNode, posLocal string) float64 {
	if anchor == nil {
		return 0
	}
	pos := anchor.child(posLocal)
	if pos == nil {
		return 0
	}
	if off := pos.child("posOffset"); off != nil {
		if v := parseDocxInt(strings.TrimSpace(docxCollectText(off))); v > 0 {
			return emuToPx(v)
		}
	}
	return 0
}

func docxAnchorExtentPx(anchor *docxXMLNode) (float64, float64) {
	if anchor == nil {
		return 0, 0
	}
	ext := anchor.child("extent")
	if ext == nil {
		return 0, 0
	}
	return emuToPx(parseDocxInt(ext.attr("cx"))), emuToPx(parseDocxInt(ext.attr("cy")))
}

func docxFontFamily(rFonts *docxXMLNode) string {
	if rFonts == nil {
		return ""
	}
	for _, attr := range []string{"ascii", "hAnsi", "cs", "eastAsia"} {
		if tf := strings.TrimSpace(rFonts.attr(attr)); tf != "" {
			return safeFontFamily(tf)
		}
	}
	return ""
}
