package officepreview

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// pptxXMLNode is a lightweight OOXML tree for slide parsing.
type pptxXMLNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr    `xml:",any,attr"`
	Children []pptxXMLNode `xml:",any"`
	Content  string        `xml:",chardata"`
}

func (n *pptxXMLNode) attr(local string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == local && a.Value != "" {
			return a.Value
		}
	}
	return ""
}

func (n *pptxXMLNode) relAttr(local string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == local && a.Value != "" && strings.Contains(a.Name.Space, "relationships") {
			return a.Value
		}
	}
	return ""
}

func (n *pptxXMLNode) child(local string) *pptxXMLNode {
	for i := range n.Children {
		if n.Children[i].XMLName.Local == local {
			return &n.Children[i]
		}
	}
	return nil
}

func (n *pptxXMLNode) findDeep(local string) *pptxXMLNode {
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

func (n *pptxXMLNode) findAllDeep(local string) []*pptxXMLNode {
	var out []*pptxXMLNode
	for i := range n.Children {
		if n.Children[i].XMLName.Local == local {
			out = append(out, &n.Children[i])
		}
		out = append(out, n.Children[i].findAllDeep(local)...)
	}
	return out
}

func parsePptxXML(data []byte) (*pptxXMLNode, error) {
	var root pptxXMLNode
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	return &root, nil
}

func parseEMU(s string) int64 {
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

// emuToPx converts EMU to CSS pixels (96 dpi).
func emuToPx(emu int64) float64 {
	return float64(emu) * 96.0 / 914400.0
}

// ptToPx converts point size to CSS pixels.
func ptToPx(pt float64) float64 {
	return pt * 96.0 / 72.0
}

// ----- Shape transform -----

type pptxShapeXfrm struct {
	left, top, cx, cy int64
	rotDeg            float64 // degrees clockwise
	flipH, flipV      bool
}

func readShapeXfrm(node *pptxXMLNode) pptxShapeXfrm {
	spPr := node.child("spPr")
	if spPr == nil {
		spPr = node.child("grpSpPr")
	}
	if spPr == nil {
		return pptxShapeXfrm{}
	}
	xfrm := spPr.child("xfrm")
	if xfrm == nil {
		return pptxShapeXfrm{}
	}
	var r pptxShapeXfrm
	if off := xfrm.child("off"); off != nil {
		r.left = parseEMU(off.attr("x"))
		r.top = parseEMU(off.attr("y"))
	}
	if ext := xfrm.child("ext"); ext != nil {
		r.cx = parseEMU(ext.attr("cx"))
		r.cy = parseEMU(ext.attr("cy"))
	}
	if rot := xfrm.attr("rot"); rot != "" {
		if v := parseEMU(rot); v != 0 {
			r.rotDeg = float64(v) / 60000.0
		}
	}
	r.flipH = xfrm.attr("flipH") == "1" || strings.EqualFold(xfrm.attr("flipH"), "true")
	r.flipV = xfrm.attr("flipV") == "1" || strings.EqualFold(xfrm.attr("flipV"), "true")
	return r
}

// ----- Group transform -----

type pptxGroupTransform struct {
	offX, offY     int64
	scaleX, scaleY float64
}

var identityTransform = pptxGroupTransform{scaleX: 1, scaleY: 1}

func (gt pptxGroupTransform) apply(x, y, cx, cy int64) (int64, int64, int64, int64) {
	return gt.offX + int64(float64(x)*gt.scaleX),
		gt.offY + int64(float64(y)*gt.scaleY),
		int64(float64(cx)*gt.scaleX),
		int64(float64(cy)*gt.scaleY)
}

func parseGroupTransform(grpSpPr *pptxXMLNode) pptxGroupTransform {
	xfrm := grpSpPr.child("xfrm")
	if xfrm == nil {
		return identityTransform
	}
	var offX, offY, extCX, extCY, chOffX, chOffY, chExtCX, chExtCY int64
	if off := xfrm.child("off"); off != nil {
		offX = parseEMU(off.attr("x"))
		offY = parseEMU(off.attr("y"))
	}
	if ext := xfrm.child("ext"); ext != nil {
		extCX = parseEMU(ext.attr("cx"))
		extCY = parseEMU(ext.attr("cy"))
	}
	if chOff := xfrm.child("chOff"); chOff != nil {
		chOffX = parseEMU(chOff.attr("x"))
		chOffY = parseEMU(chOff.attr("y"))
	}
	if chExt := xfrm.child("chExt"); chExt != nil {
		chExtCX = parseEMU(chExt.attr("cx"))
		chExtCY = parseEMU(chExt.attr("cy"))
	}
	scaleX, scaleY := 1.0, 1.0
	if chExtCX > 0 {
		scaleX = float64(extCX) / float64(chExtCX)
	}
	if chExtCY > 0 {
		scaleY = float64(extCY) / float64(chExtCY)
	}
	return pptxGroupTransform{
		offX:   offX - int64(float64(chOffX)*scaleX),
		offY:   offY - int64(float64(chOffY)*scaleY),
		scaleX: scaleX,
		scaleY: scaleY,
	}
}

func composeTransforms(parent, child pptxGroupTransform) pptxGroupTransform {
	return pptxGroupTransform{
		offX:   parent.offX + int64(float64(child.offX)*parent.scaleX),
		offY:   parent.offY + int64(float64(child.offY)*parent.scaleY),
		scaleX: parent.scaleX * child.scaleX,
		scaleY: parent.scaleY * child.scaleY,
	}
}

// walkPptxShapes visits every node, passing the accumulated group transform.
// grpSp containers are NOT passed to fn; their children are visited with the
// composed group transform applied.
func walkPptxShapes(node *pptxXMLNode, gt pptxGroupTransform, fn func(local string, n *pptxXMLNode, gt pptxGroupTransform)) {
	if node.XMLName.Local != "grpSp" {
		fn(node.XMLName.Local, node, gt)
	}
	for i := range node.Children {
		child := &node.Children[i]
		childGT := gt
		if child.XMLName.Local == "grpSp" {
			if grpSpPr := child.child("grpSpPr"); grpSpPr != nil {
				localGT := parseGroupTransform(grpSpPr)
				childGT = composeTransforms(gt, localGT)
			}
		}
		walkPptxShapes(child, childGT, fn)
	}
}

// ----- Placeholder map -----

// phKey identifies a placeholder by type and optional index.
type phKey struct {
	typ string
	idx string
}

// phInfo holds the resolved position and default text style for a placeholder.
type phInfo struct {
	left, top, cx, cy int64
	defaultFontPt     float64
	defaultColor      string
	defaultBold       bool
}

// buildPhMap collects placeholder geometry and default text styles from layout
// and master XML trees. Layout values take precedence over master.
func buildPhMap(masterRoot, layoutRoot *pptxXMLNode, theme *pptxTheme) map[phKey]phInfo {
	m := make(map[phKey]phInfo)
	for _, root := range []*pptxXMLNode{masterRoot, layoutRoot} {
		if root == nil {
			continue
		}
		collectPhInfo(root, theme, m)
	}
	return m
}

func collectPhInfo(root *pptxXMLNode, theme *pptxTheme, out map[phKey]phInfo) {
	walkPptxShapes(root, identityTransform, func(local string, n *pptxXMLNode, gt pptxGroupTransform) {
		if local != "sp" {
			return
		}
		ph := n.findDeep("ph")
		if ph == nil {
			return
		}
		key := phKey{typ: ph.attr("type"), idx: ph.attr("idx")}
		existing := out[key]

		xfrm := readShapeXfrm(n)
		if xfrm.cx > 0 && xfrm.cy > 0 {
			l, t, cx, cy := gt.apply(xfrm.left, xfrm.top, xfrm.cx, xfrm.cy)
			existing.left, existing.top, existing.cx, existing.cy = l, t, cx, cy
		}

		if txBody := n.child("txBody"); txBody != nil {
			if lstStyle := txBody.child("lstStyle"); lstStyle != nil {
				pt, clr, bold := defaultRunStyleFromLstStyle(lstStyle, theme)
				if pt > 0 {
					existing.defaultFontPt = pt
				}
				if clr != "" {
					existing.defaultColor = clr
				}
				if bold {
					existing.defaultBold = bold
				}
			}
		}
		out[key] = existing
	})
}

// defaultRunStyleFromLstStyle extracts default font size, color, and bold from
// an OOXML lstStyle element (lvl1pPr/defRPr).
func defaultRunStyleFromLstStyle(lstStyle *pptxXMLNode, theme *pptxTheme) (fontPt float64, color string, bold bool) {
	if lstStyle == nil {
		return
	}
	for i := range lstStyle.Children {
		lvl := &lstStyle.Children[i]
		if !strings.HasPrefix(lvl.XMLName.Local, "lvl") {
			continue
		}
		defRPr := lvl.child("defRPr")
		if defRPr == nil {
			continue
		}
		if sz := defRPr.attr("sz"); sz != "" {
			if v := parseEMU(sz); v > 0 {
				fontPt = float64(v) / 100
			}
		}
		bold = defRPr.attr("b") == "1" || strings.EqualFold(defRPr.attr("b"), "true")
		if fill := defRPr.child("solidFill"); fill != nil {
			color = resolveColorNode(fill, theme)
		}
		// Also check direct color child at rPr level (no solidFill wrapper)
		if color == "" {
			if clrNode := defRPr.findDeep("srgbClr"); clrNode == nil {
				if clrNode = defRPr.findDeep("schemeClr"); clrNode != nil {
					// Wrap in a fake solidFill context — resolve directly
					color = resolveColorNodeDirect(clrNode, theme)
				}
			} else {
				color = "#" + strings.ToUpper(clrNode.attr("val"))
			}
		}
		if fontPt > 0 || color != "" {
			break // only use first level
		}
	}
	return fontPt, color, bold
}

// resolveColorNodeDirect resolves a srgbClr/schemeClr/sysClr node directly (no parent solidFill).
func resolveColorNodeDirect(node *pptxXMLNode, theme *pptxTheme) string {
	switch node.XMLName.Local {
	case "srgbClr":
		hex := strings.ToUpper(node.attr("val"))
		if hex != "" {
			return "#" + applyColorTransforms(hex, node)
		}
	case "schemeClr":
		hex := theme.resolveScheme(node.attr("val"))
		if hex != "" {
			return "#" + applyColorTransforms(hex, node)
		}
	case "sysClr":
		if val := node.attr("lastClr"); val != "" {
			return "#" + strings.ToUpper(val)
		}
	}
	return ""
}

// shapeStyleFill reads the shape's fill color from p:style/a:fillRef,
// which is how layout/master background shapes get their theme color.
// idx=1 → first fillStyleLst entry (solidFill), idx>=2 → gradients, idx>=1001 → bgFillStyleLst.
func shapeStyleFill(sp *pptxXMLNode, theme *pptxTheme) string {
	styleEl := sp.child("style")
	if styleEl == nil {
		return ""
	}
	fillRef := styleEl.child("fillRef")
	if fillRef == nil {
		return ""
	}
	idxStr := fillRef.attr("idx")
	if idxStr == "" || idxStr == "0" {
		return ""
	}
	idx := parseEMU(idxStr)

	// Resolve the placeholder color (the schemeClr/srgbClr inside fillRef itself).
	phClrHex := ""
	if clr := resolveColorNode(fillRef, theme); clr != "" {
		phClrHex = strings.TrimPrefix(clr, "#")
	}

	// Pick the fill style list entry: idx 1..N → fillStyleLst, idx 1001+ → bgFillStyleLst.
	var fillNode *pptxXMLNode
	if idx >= 1001 {
		i := int(idx - 1001)
		if i < len(theme.bgFillStyles) {
			fillNode = theme.bgFillStyles[i]
		}
	} else {
		i := int(idx - 1)
		if i >= 0 && i < len(theme.fillStyles) {
			fillNode = theme.fillStyles[i]
		}
	}

	if fillNode != nil && phClrHex != "" {
		if css := renderThemeFillCSS(fillNode, phClrHex, theme); css != "" {
			return "background:" + css + ";"
		}
	}
	// Fall back to flat color when the fill style isn't gradient-renderable.
	if phClrHex != "" {
		return "background-color:#" + phClrHex + ";"
	}
	return ""
}

// parseShapeBorder reads a shape's outline from spPr/ln.
func parseShapeBorder(spPr *pptxXMLNode, theme *pptxTheme) string {
	if spPr == nil {
		return ""
	}
	ln := spPr.child("ln")
	if ln == nil {
		return ""
	}
	if ln.child("noFill") != nil {
		return ""
	}
	fill := ln.child("solidFill")
	if fill == nil {
		return ""
	}
	clr := resolveColorNode(fill, theme)
	if clr == "" {
		return ""
	}
	widthEMU := parseEMU(ln.attr("w"))
	if widthEMU <= 0 {
		widthEMU = 12700
	}
	widthPx := emuToPx(widthEMU)
	if widthPx < 0.5 {
		widthPx = 0.5
	}
	return fmt.Sprintf("border:%.1fpx solid %s;box-sizing:border-box;", widthPx, clr)
}

// ----- Text extraction -----

func collectText(n *pptxXMLNode) string {
	if strings.TrimSpace(n.Content) != "" {
		return n.Content
	}
	var parts []string
	for i := range n.Children {
		if t := collectText(&n.Children[i]); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "")
}

func firstRunStyle(node *pptxXMLNode, theme *pptxTheme) (fontPt float64, colorHex string, bold bool) {
	rPr := node.findDeep("rPr")
	if rPr == nil {
		return 0, "", false
	}
	if sz := rPr.attr("sz"); sz != "" {
		if v := parseEMU(sz); v > 0 {
			fontPt = float64(v) / 100
		}
	}
	bold = rPr.attr("b") == "1" || strings.EqualFold(rPr.attr("b"), "true")
	if fill := rPr.findDeep("solidFill"); fill != nil {
		colorHex = resolveColorNode(fill, theme)
	}
	return fontPt, colorHex, bold
}

func shapeIsTitle(node *pptxXMLNode) bool {
	ph := node.findDeep("ph")
	if ph == nil {
		return false
	}
	t := ph.attr("type")
	return t == "title" || t == "ctrTitle" || t == "subTitle"
}

func findBlipEmbedInTree(node *pptxXMLNode) string {
	if node.XMLName.Local == "blip" {
		if id := node.relAttr("embed"); id != "" {
			return id
		}
	}
	for i := range node.Children {
		if id := findBlipEmbedInTree(&node.Children[i]); id != "" {
			return id
		}
	}
	return ""
}

func shapeFillBackground(spPr *pptxXMLNode, theme *pptxTheme) string {
	if spPr == nil {
		return ""
	}
	fill := spPr.findDeep("solidFill")
	if fill == nil {
		return ""
	}
	clr := resolveColorNode(fill, theme)
	if clr == "" {
		return ""
	}
	hex := strings.TrimPrefix(clr, "#")
	// Check for alpha transforms within the fill node's color child
	alpha := int64(100000)
	for _, child := range fill.Children {
		if a := child.findDeep("alpha"); a != nil {
			if v := parseEMU(a.attr("val")); v > 0 {
				alpha = v
			}
		}
	}
	opacity := float64(alpha) / 100000
	if opacity > 1 {
		opacity = 1
	}
	if len(hex) == 6 && opacity < 1 {
		r := hexByte(hex, 0)
		g := hexByte(hex, 2)
		b := hexByte(hex, 4)
		return fmt.Sprintf("background-color:rgba(%s,%s,%s,%.2f);", r, g, b, opacity)
	}
	return "background-color:" + clr + ";"
}

func hexByte(hex string, offset int) string {
	if len(hex) < offset+2 {
		return "0"
	}
	i, err := strconv.ParseInt(hex[offset:offset+2], 16, 64)
	if err != nil {
		return "0"
	}
	return strconv.FormatInt(i, 10)
}

// ----- Per-paragraph / per-run styled HTML -----

type pptxParaHTML struct {
	html  string
	style string // full CSS for the <p> element (alignment, line-height, spacing, etc.)
}

// extractTxBodyHTML returns per-paragraph HTML with inline styles for each run.
func extractTxBodyHTML(txBody *pptxXMLNode, theme *pptxTheme) []pptxParaHTML {
	var result []pptxParaHTML
	for i := range txBody.Children {
		child := &txBody.Children[i]
		if child.XMLName.Local != "p" {
			continue
		}
		html, style := extractParagraphHTML(child, theme)
		if strings.TrimSpace(html) != "" {
			result = append(result, pptxParaHTML{html: html, style: style})
		}
	}
	return result
}

func extractParagraphHTML(p *pptxXMLNode, theme *pptxTheme) (string, string) {
	var styleProps []string
	marL := int64(0)

	if pPr := p.child("pPr"); pPr != nil {
		switch pPr.attr("algn") {
		case "ctr", "center":
			styleProps = append(styleProps, "text-align:center")
		case "r", "right":
			styleProps = append(styleProps, "text-align:right")
		case "just", "dist":
			styleProps = append(styleProps, "text-align:justify")
		case "l", "left":
			styleProps = append(styleProps, "text-align:left")
		}
		marL = parseEMU(pPr.attr("marL"))
		if marL > 0 {
			styleProps = append(styleProps, fmt.Sprintf("padding-left:%.2fpx", emuToPx(marL)))
		}
		// Line spacing
		if lnSpc := pPr.child("lnSpc"); lnSpc != nil {
			if pt := parseSpcPts(lnSpc); pt > 0 {
				styleProps = append(styleProps, fmt.Sprintf("line-height:%.2fpx", ptToPx(pt)))
			} else if pct := parseSpcPct(lnSpc); pct > 0 {
				styleProps = append(styleProps, fmt.Sprintf("line-height:%.0f%%", pct*100))
			}
		}
		// Space before paragraph
		if spcBef := pPr.child("spcBef"); spcBef != nil {
			if pt := parseSpcPts(spcBef); pt > 0 {
				styleProps = append(styleProps, fmt.Sprintf("margin-top:%.2fpx", ptToPx(pt)))
			}
		}
		// Space after paragraph
		if spcAft := pPr.child("spcAft"); spcAft != nil {
			if pt := parseSpcPts(spcAft); pt > 0 {
				styleProps = append(styleProps, fmt.Sprintf("margin-bottom:%.2fpx", ptToPx(pt)))
			}
		}
	}

	var buf strings.Builder
	for i := range p.Children {
		child := &p.Children[i]
		switch child.XMLName.Local {
		case "r":
			buf.WriteString(extractRunHTML(child, theme))
		case "br":
			buf.WriteString("<br/>")
		case "fld":
			for _, t := range child.findAllDeep("t") {
				if text := collectText(t); text != "" {
					buf.WriteString(escapeHTMLText(text))
				}
			}
		}
	}
	return buf.String(), strings.Join(styleProps, ";")
}

// parseSpcPts returns spacing in points from a spcBef/spcAft/lnSpc node
// that contains a spcPts child (val is in hundredths of a point).
func parseSpcPts(node *pptxXMLNode) float64 {
	if node == nil {
		return 0
	}
	if spcPts := node.child("spcPts"); spcPts != nil {
		if v := parseEMU(spcPts.attr("val")); v > 0 {
			return float64(v) / 100
		}
	}
	return 0
}

// parseSpcPct returns spacing as a multiplier (1.0 = 100%) from a spcPct child
// (val is in thousandths of a percent, 100000 = 100%).
func parseSpcPct(node *pptxXMLNode) float64 {
	if node == nil {
		return 0
	}
	if spcPct := node.child("spcPct"); spcPct != nil {
		if v := parseEMU(spcPct.attr("val")); v > 0 {
			return float64(v) / 100000
		}
	}
	return 0
}

func extractRunHTML(r *pptxXMLNode, theme *pptxTheme) string {
	tNode := r.child("t")
	if tNode == nil {
		return ""
	}
	text := collectText(tNode)
	if text == "" {
		return ""
	}

	var styles []string
	isLink := false
	rPr := r.child("rPr")
	if rPr != nil {
		if sz := rPr.attr("sz"); sz != "" {
			if v := parseEMU(sz); v > 0 {
				styles = append(styles, fmt.Sprintf("font-size:%.2fpx", ptToPx(float64(v)/100)))
			}
		}
		if b := rPr.attr("b"); b == "1" || strings.EqualFold(b, "true") {
			styles = append(styles, "font-weight:700")
		}
		if it := rPr.attr("i"); it == "1" || strings.EqualFold(it, "true") {
			styles = append(styles, "font-style:italic")
		}
		var decorations []string
		if u := rPr.attr("u"); u != "" && u != "none" {
			decorations = append(decorations, "underline")
		}
		if sk := rPr.attr("strike"); sk != "" && sk != "noStrike" {
			decorations = append(decorations, "line-through")
		}
		// Hyperlink: apply hlink theme color + underline
		if rPr.child("hlinkClick") != nil || rPr.child("hlinkMouseOver") != nil {
			isLink = true
			if !contains(decorations, "underline") {
				decorations = append(decorations, "underline")
			}
			if hlinkClr := theme.resolveScheme("hlink"); hlinkClr != "" {
				styles = append(styles, "color:#"+hlinkClr)
			}
		}
		if len(decorations) > 0 {
			styles = append(styles, "text-decoration:"+strings.Join(decorations, " "))
		}
		if !isLink {
			if fill := rPr.child("solidFill"); fill != nil {
				if clr := resolveColorNode(fill, theme); clr != "" {
					styles = append(styles, "color:"+clr)
				}
			}
		}
		if latin := rPr.child("latin"); latin != nil {
			if tf := safeFontFamily(latin.attr("typeface")); tf != "" {
				styles = append(styles, "font-family:'"+tf+"'")
			}
		}
	}

	escaped := escapeHTMLText(text)
	if len(styles) == 0 {
		return escaped
	}
	return `<span style="` + strings.Join(styles, ";") + `">` + escaped + `</span>`
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func safeFontFamily(tf string) string {
	if strings.HasPrefix(tf, "+") || tf == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range tf {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == ' ' || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// parseSlideBgCSS returns a CSS color value for the slide's background.
func parseSlideBgCSS(cSld *pptxXMLNode, theme *pptxTheme) string {
	if cSld == nil {
		return ""
	}
	bg := cSld.child("bg")
	if bg == nil {
		return ""
	}
	if bgPr := bg.child("bgPr"); bgPr != nil {
		if fill := bgPr.child("solidFill"); fill != nil {
			if clr := resolveColorNode(fill, theme); clr != "" {
				return clr
			}
		}
		if gradFill := bgPr.child("gradFill"); gradFill != nil {
			if gsLst := gradFill.child("gsLst"); gsLst != nil {
				for i := range gsLst.Children {
					gs := &gsLst.Children[i]
					if gs.XMLName.Local != "gs" {
						continue
					}
					if clr := resolveColorNode(gs, theme); clr != "" {
						return clr
					}
				}
			}
		}
	}
	if bgRef := bg.child("bgRef"); bgRef != nil {
		if clr := resolveColorNode(bgRef, theme); clr != "" {
			return clr
		}
	}
	return ""
}

// ----- Theme color resolution -----

type pptxTheme struct {
	colors        map[string]string // scheme name → RRGGBB hex (no #)
	clrMap        map[string]string // bg1/tx1/bg2/tx2/etc → dk1/lt1/... per slide master clrMap
	fillStyles    []*pptxXMLNode    // fmtScheme/fillStyleLst entries (1-indexed by fillRef idx)
	bgFillStyles  []*pptxXMLNode    // fmtScheme/bgFillStyleLst entries (1001+ idx)
}

// defaultClrMap is the standard mapping used when a master doesn't define its own.
var defaultClrMap = map[string]string{
	"bg1": "lt1",
	"bg2": "lt2",
	"tx1": "dk1",
	"tx2": "dk2",
}

func loadPptxTheme(zr *zip.Reader) *pptxTheme {
	theme := &pptxTheme{colors: make(map[string]string), clrMap: defaultClrMap}
	for _, name := range []string{"ppt/theme/theme1.xml", "ppt/theme/theme2.xml", "ppt/theme/theme3.xml"} {
		data, err := readZipFile(zr, name)
		if err != nil {
			continue
		}
		root, err := parsePptxXML(data)
		if err != nil {
			continue
		}
		clrScheme := root.findDeep("clrScheme")
		if clrScheme == nil {
			continue
		}
		colors := make(map[string]string)
		for _, child := range clrScheme.Children {
			schemeName := child.XMLName.Local
			hex := extractSingleColorHex(&child)
			if hex != "" {
				colors[schemeName] = hex
			}
		}
		theme.colors = colors
		// Also cache fillStyleLst / bgFillStyleLst entries for fillRef idx resolution.
		if fmtScheme := root.findDeep("fmtScheme"); fmtScheme != nil {
			if fsl := fmtScheme.child("fillStyleLst"); fsl != nil {
				for i := range fsl.Children {
					c := &fsl.Children[i]
					theme.fillStyles = append(theme.fillStyles, c)
				}
			}
			if bfsl := fmtScheme.child("bgFillStyleLst"); bfsl != nil {
				for i := range bfsl.Children {
					c := &bfsl.Children[i]
					theme.bgFillStyles = append(theme.bgFillStyles, c)
				}
			}
		}
		break
	}
	// Read the first slide master's clrMap, if present. Most decks have one master,
	// and its clrMap can swap the standard bg/tx mappings (e.g., dark-themed templates).
	if data, err := readZipFile(zr, "ppt/slideMasters/slideMaster1.xml"); err == nil {
		if root, err := parsePptxXML(data); err == nil {
			if cm := root.findDeep("clrMap"); cm != nil {
				m := make(map[string]string, len(cm.Attrs))
				for _, a := range cm.Attrs {
					if a.Value != "" {
						m[a.Name.Local] = a.Value
					}
				}
				if len(m) > 0 {
					theme.clrMap = m
				}
			}
		}
	}
	return theme
}

func extractSingleColorHex(node *pptxXMLNode) string {
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

func (t *pptxTheme) resolveScheme(name string) string {
	if t.clrMap != nil {
		if alias, ok := t.clrMap[name]; ok && alias != "" {
			name = alias
		}
	}
	return t.colors[name]
}

func resolveColorNode(node *pptxXMLNode, theme *pptxTheme) string {
	if node == nil {
		return ""
	}
	if clr := node.child("srgbClr"); clr != nil {
		hex := strings.ToUpper(clr.attr("val"))
		if hex != "" {
			return "#" + applyColorTransforms(hex, clr)
		}
	}
	if clr := node.child("schemeClr"); clr != nil {
		hex := theme.resolveScheme(clr.attr("val"))
		if hex != "" {
			return "#" + applyColorTransforms(hex, clr)
		}
	}
	if clr := node.child("sysClr"); clr != nil {
		if val := clr.attr("lastClr"); val != "" {
			return "#" + strings.ToUpper(val)
		}
	}
	if clr := node.child("prstClr"); clr != nil {
		if hex := prstColorHex(clr.attr("val")); hex != "" {
			return "#" + hex
		}
	}
	return ""
}

var prstColors = map[string]string{
	"black":   "000000",
	"white":   "FFFFFF",
	"red":     "FF0000",
	"green":   "008000",
	"blue":    "0000FF",
	"yellow":  "FFFF00",
	"cyan":    "00FFFF",
	"magenta": "FF00FF",
	"orange":  "FFA500",
	"purple":  "800080",
	"gray":    "808080",
	"grey":    "808080",
	"silver":  "C0C0C0",
	"navy":    "000080",
	"teal":    "008080",
	"maroon":  "800000",
	"lime":    "00FF00",
	"aqua":    "00FFFF",
	"fuchsia": "FF00FF",
}

func prstColorHex(name string) string {
	return prstColors[strings.ToLower(name)]
}

func applyColorTransforms(hex string, node *pptxXMLNode) string {
	rgba := applyColorTransformsRGBA(hex, node)
	return fmt.Sprintf("%02X%02X%02X", rgba[0], rgba[1], rgba[2])
}

// applyColorTransformsRGBA applies OOXML color transforms and returns RGBA with
// alpha in [0,255]. Alpha defaults to 255 unless an <a:alpha> transform is present.
func applyColorTransformsRGBA(hex string, node *pptxXMLNode) [4]uint8 {
	if len(hex) != 6 {
		return [4]uint8{0, 0, 0, 255}
	}
	r := hexByteU8(hex, 0)
	g := hexByteU8(hex, 2)
	b := hexByteU8(hex, 4)
	alpha := uint8(255)

	for _, child := range node.Children {
		val := parseEMU(child.attr("val"))
		switch child.XMLName.Local {
		case "tint":
			f := float64(val) / 100000
			r = uint8(float64(r) + float64(255-r)*f)
			g = uint8(float64(g) + float64(255-g)*f)
			b = uint8(float64(b) + float64(255-b)*f)
		case "shade":
			f := float64(val) / 100000
			r = uint8(float64(r) * f)
			g = uint8(float64(g) * f)
			b = uint8(float64(b) * f)
		case "lumMod":
			h, l, s := rgbToHLS(r, g, b)
			l = clampF(l * float64(val) / 100000)
			r, g, b = hlsToRGB(h, l, s)
		case "lumOff":
			h, l, s := rgbToHLS(r, g, b)
			l = clampF(l + float64(val)/100000)
			r, g, b = hlsToRGB(h, l, s)
		case "satMod":
			h, l, s := rgbToHLS(r, g, b)
			s = clampF(s * float64(val) / 100000)
			r, g, b = hlsToRGB(h, l, s)
		case "satOff":
			h, l, s := rgbToHLS(r, g, b)
			s = clampF(s + float64(val)/100000)
			r, g, b = hlsToRGB(h, l, s)
		case "alpha":
			f := float64(val) / 100000
			alpha = uint8(clampF(f) * 255)
		}
	}
	return [4]uint8{r, g, b, alpha}
}

// renderThemeFillCSS renders a theme fillStyleLst entry (solidFill or gradFill)
// as a CSS `background:` declaration value (without trailing semicolon), with
// `phClr` substituted by the supplied placeholder color (hex without `#`).
// Returns "" if the fill kind isn't supported.
func renderThemeFillCSS(fillNode *pptxXMLNode, phClrHex string, theme *pptxTheme) string {
	if fillNode == nil || phClrHex == "" {
		return ""
	}
	switch fillNode.XMLName.Local {
	case "solidFill":
		clr := fillNode.child("schemeClr")
		if clr == nil {
			return ""
		}
		rgba := applyColorTransformsRGBA(phClrHex, clr)
		return formatColorCSS(rgba)
	case "gradFill":
		gsLst := fillNode.child("gsLst")
		if gsLst == nil {
			return ""
		}
		type stop struct {
			pos  float64 // 0..100
			rgba [4]uint8
		}
		var stops []stop
		for i := range gsLst.Children {
			gs := &gsLst.Children[i]
			if gs.XMLName.Local != "gs" {
				continue
			}
			pos := float64(parseEMU(gs.attr("pos"))) / 1000
			var clrNode *pptxXMLNode
			if c := gs.child("schemeClr"); c != nil {
				clrNode = c
			} else if c := gs.child("srgbClr"); c != nil {
				clrNode = c
			}
			if clrNode == nil {
				continue
			}
			base := phClrHex
			if clrNode.XMLName.Local == "srgbClr" {
				if v := clrNode.attr("val"); v != "" {
					base = strings.ToUpper(v)
				}
			}
			stops = append(stops, stop{pos: pos, rgba: applyColorTransformsRGBA(base, clrNode)})
		}
		if len(stops) < 2 {
			if len(stops) == 1 {
				return formatColorCSS(stops[0].rgba)
			}
			return ""
		}
		// Angle: OOXML lin ang is in 60000ths of a degree, 0 = pointing right, CW.
		// CSS linear-gradient angle: 0deg = pointing up, CW. So CSS = ang/60000 + 90.
		cssAngle := 90.0
		if lin := fillNode.child("lin"); lin != nil {
			cssAngle = float64(parseEMU(lin.attr("ang")))/60000 + 90
		}
		var b strings.Builder
		fmt.Fprintf(&b, "linear-gradient(%.2fdeg", cssAngle)
		for _, s := range stops {
			fmt.Fprintf(&b, ", %s %.2f%%", formatColorCSS(s.rgba), s.pos)
		}
		b.WriteString(")")
		return b.String()
	}
	return ""
}

func formatColorCSS(rgba [4]uint8) string {
	if rgba[3] == 255 {
		return fmt.Sprintf("#%02X%02X%02X", rgba[0], rgba[1], rgba[2])
	}
	return fmt.Sprintf("rgba(%d,%d,%d,%.3f)", rgba[0], rgba[1], rgba[2], float64(rgba[3])/255)
}

func hexByteU8(hex string, offset int) uint8 {
	if len(hex) < offset+2 {
		return 0
	}
	i, err := strconv.ParseInt(hex[offset:offset+2], 16, 64)
	if err != nil {
		return 0
	}
	return uint8(i)
}

func clampF(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func rgbToHLS(r, g, b uint8) (h, l, s float64) {
	rf := float64(r) / 255
	gf := float64(g) / 255
	bf := float64(b) / 255
	mx := rf
	if gf > mx {
		mx = gf
	}
	if bf > mx {
		mx = bf
	}
	mn := rf
	if gf < mn {
		mn = gf
	}
	if bf < mn {
		mn = bf
	}
	l = (mx + mn) / 2
	if mx == mn {
		return 0, l, 0
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case rf:
		h = (gf - bf) / d
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/d + 2
	case bf:
		h = (rf-gf)/d + 4
	}
	h /= 6
	return h, l, s
}

func hlsToRGB(h, l, s float64) (r, g, b uint8) {
	if s == 0 {
		v := uint8(l * 255)
		return v, v, v
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	return uint8(hue2rgb(p, q, h+1.0/3) * 255),
		uint8(hue2rgb(p, q, h) * 255),
		uint8(hue2rgb(p, q, h-1.0/3) * 255)
}

func hue2rgb(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6 {
		return p + (q-p)*6*t
	}
	if t < 0.5 {
		return q
	}
	if t < 2.0/3 {
		return p + (q-p)*(2.0/3-t)*6
	}
	return p
}
