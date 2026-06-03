package officepreview

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path"
	"strings"
)

const (
	defaultPptxCX = 9144000
	defaultPptxCY = 6858000
)

type pptxSlideSize struct {
	cx int64
	cy int64
}

type pptxVisualLayer struct {
	left, top, cx, cy int64
	rotDeg            float64
	flipH             bool
	zIndex            int
	kind              string // text | image
	// text layers
	paraHTML   []pptxParaHTML
	fontPx     float64
	color      string
	background string
	border     string
	vertAlign  string // top | middle | bottom
	vert       string // "" | "vert" | "vert270" — text direction
	noWrap     bool   // white-space: nowrap
	bold       bool
	isTitle    bool
	// image layers
	dataURI string
	alt     string
}

func convertPptxToVisualHTML(data []byte, filename, mimeType string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pptx zip: %w", err)
	}
	slidePaths, err := pptxSlidePaths(zr)
	if err != nil {
		return "", err
	}
	if len(slidePaths) == 0 {
		return "", fmt.Errorf("no slides")
	}
	size := pptxPresentationSize(zr)
	theme := loadPptxTheme(zr)

	var slides strings.Builder
	rendered := 0
	for i, slidePath := range slidePaths {
		slideHTML, ok := renderPptxVisualSlide(zr, slidePath, i+1, size, theme)
		if !ok {
			continue
		}
		slides.WriteString(slideHTML)
		rendered++
	}
	if rendered == 0 {
		return "", fmt.Errorf("no visual slides rendered")
	}
	slideWpx := emuToPx(size.cx)
	return wrapPptxHTMLDocument(slides.String(), slideWpx), nil
}

func pptxPresentationSize(zr *zip.Reader) pptxSlideSize {
	size := pptxSlideSize{cx: defaultPptxCX, cy: defaultPptxCY}
	data, err := readZipFile(zr, "ppt/presentation.xml")
	if err != nil {
		return size
	}
	root, err := parsePptxXML(data)
	if err != nil {
		return size
	}
	sldSz := root.findDeep("sldSz")
	if sldSz == nil {
		return size
	}
	if cx := parseEMU(sldSz.attr("cx")); cx > 0 {
		size.cx = cx
	}
	if cy := parseEMU(sldSz.attr("cy")); cy > 0 {
		size.cy = cy
	}
	return size
}

func renderPptxVisualSlide(zr *zip.Reader, slidePath string, slideNum int, size pptxSlideSize, theme *pptxTheme) (string, bool) {
	slideData, err := readZipFile(zr, slidePath)
	if err != nil {
		return "", false
	}
	slideRoot, err := parsePptxXML(slideData)
	if err != nil {
		return "", false
	}
	slideRels, _ := parsePackageRels(zr, pptxSlideRelsPath(slidePath))

	// Resolve layout and master for background rendering and placeholder info.
	layoutPath := pptxRelatedPartPath(slideRels, slidePath, "slideLayout")
	var layoutRoot, masterRoot *pptxXMLNode
	var layoutRels, masterRels map[string]packageRel
	var masterPath string
	if layoutPath != "" {
		if data, err := readZipFile(zr, layoutPath); err == nil {
			layoutRoot, _ = parsePptxXML(data)
		}
		layoutRels, _ = parsePackageRels(zr, pptxPartRelsPath(layoutPath))
		masterPath = pptxRelatedPartPath(layoutRels, layoutPath, "slideMaster")
		if masterPath != "" {
			if data, err := readZipFile(zr, masterPath); err == nil {
				masterRoot, _ = parsePptxXML(data)
			}
			masterRels, _ = parsePackageRels(zr, pptxPartRelsPath(masterPath))
		}
	}

	// Build placeholder geometry + default style map from layout and master.
	phMap := buildPhMap(masterRoot, layoutRoot, theme)

	// Collect background (image or solid color).
	bgURI := ""
	bgColor := ""
	for _, pair := range []struct {
		root *pptxXMLNode
		pth  string
		rels map[string]packageRel
	}{
		{slideRoot, slidePath, slideRels},
		{layoutRoot, layoutPath, layoutRels},
		{masterRoot, masterPath, masterRels},
	} {
		if pair.root == nil || (bgURI != "" || bgColor != "") {
			continue
		}
		if cSld := pair.root.findDeep("cSld"); cSld != nil {
			bgColor = parseSlideBgCSS(cSld, theme)
			if bg := cSld.child("bg"); bg != nil {
				if embed := findBlipEmbedInTree(bg); embed != "" {
					bgURI = resolvePptxEmbed(zr, pair.pth, pair.rels, embed)
				}
			}
		}
	}

	var layers []pptxVisualLayer
	z := 0

	// 1. Background (non-placeholder) shapes from master.
	if masterRoot != nil {
		bgLayers := extractBackgroundLayers(masterRoot, zr, masterPath, masterRels, theme)
		for i := range bgLayers {
			bgLayers[i].zIndex = z
			z++
		}
		layers = append(layers, bgLayers...)
	}

	// 2. Background (non-placeholder) shapes from layout.
	if layoutRoot != nil {
		bgLayers := extractBackgroundLayers(layoutRoot, zr, layoutPath, layoutRels, theme)
		for i := range bgLayers {
			bgLayers[i].zIndex = z
			z++
		}
		layers = append(layers, bgLayers...)
	}

	// 3. All shapes from the slide, with placeholder position/style inheritance.
	slideLayers := extractSlideLayers(slideRoot, zr, slidePath, slideRels, size, theme, phMap)
	for i := range slideLayers {
		slideLayers[i].zIndex = z
		z++
	}
	layers = append(layers, slideLayers...)

	// Promote a large background image if no explicit background was found.
	if bgURI == "" && bgColor == "" {
		var best *pptxVisualLayer
		var bestArea int64
		for i := range layers {
			if layers[i].kind != "image" || layers[i].dataURI == "" {
				continue
			}
			area := layers[i].cx * layers[i].cy
			if area > bestArea {
				bestArea = area
				best = &layers[i]
			}
		}
		if best != nil && bestArea > (size.cx*size.cy)/3 {
			bgURI = best.dataURI
			filtered := layers[:0]
			for i := range layers {
				if &layers[i] == best {
					continue
				}
				filtered = append(filtered, layers[i])
			}
			layers = filtered
		}
	}

	if len(layers) == 0 && bgURI == "" && bgColor == "" {
		return "", false
	}

	slideWpx := emuToPx(size.cx)
	slideHpx := emuToPx(size.cy)

	canvasStyle := fmt.Sprintf("aspect-ratio: %.4f / %.4f", float64(size.cx), float64(size.cy))
	if bgColor != "" && bgURI == "" {
		canvasStyle += "; background:" + bgColor
	}

	var b strings.Builder
	b.WriteString(`<article class="pptx-slide-wrap">`)
	b.WriteString(fmt.Sprintf(`<p class="pptx-slide-label">Slide %d</p>`, slideNum))
	b.WriteString(fmt.Sprintf(`<div class="pptx-canvas" style="%s">`, canvasStyle))
	if bgURI != "" {
		b.WriteString(fmt.Sprintf(`<img class="pptx-canvas-bg" src="%s" alt="" role="presentation"/>`, bgURI))
	}
	b.WriteString(fmt.Sprintf(`<div class="pptx-inner" data-w="%.2f" style="width:%.2fpx;height:%.2fpx">`, slideWpx, slideWpx, slideHpx))
	for _, layer := range layers {
		b.WriteString(renderPptxVisualLayer(layer))
	}
	b.WriteString(`</div></div></article>`)
	return b.String(), true
}

// extractBackgroundLayers renders only non-placeholder shapes (backgrounds,
// logos, decorative elements) from a master or layout part.
func extractBackgroundLayers(root *pptxXMLNode, zr *zip.Reader, partPath string, rels map[string]packageRel, theme *pptxTheme) []pptxVisualLayer {
	var layers []pptxVisualLayer
	z := 0
	walkPptxShapes(root, identityTransform, func(local string, n *pptxXMLNode, gt pptxGroupTransform) {
		// Skip placeholder shapes — the slide provides the actual content.
		if n.findDeep("ph") != nil {
			return
		}
		layer, ok := extractShapeLayer(local, n, gt, zr, partPath, rels, theme, nil)
		if ok {
			layer.zIndex = z
			z++
			layers = append(layers, layer)
		}
	})
	return layers
}

// extractSlideLayers renders all shapes from the slide. Placeholder shapes look
// up their position and default style from phMap when not set on the shape itself.
func extractSlideLayers(root *pptxXMLNode, zr *zip.Reader, slidePath string, rels map[string]packageRel, size pptxSlideSize, theme *pptxTheme, phMap map[phKey]phInfo) []pptxVisualLayer {
	var layers []pptxVisualLayer
	z := 0
	walkPptxShapes(root, identityTransform, func(local string, n *pptxXMLNode, gt pptxGroupTransform) {
		layer, ok := extractShapeLayer(local, n, gt, zr, slidePath, rels, theme, phMap)
		if ok {
			layer.zIndex = z
			z++
			layers = append(layers, layer)
		}
	})
	return layers
}

// extractShapeLayer extracts a single visual layer from a sp, pic, or cxnSp shape.
// phMap is nil for master/layout background rendering.
func extractShapeLayer(local string, n *pptxXMLNode, gt pptxGroupTransform, zr *zip.Reader, partPath string, rels map[string]packageRel, theme *pptxTheme, phMap map[phKey]phInfo) (pptxVisualLayer, bool) {
	switch local {
	case "pic":
		return extractPicLayer(n, gt, zr, partPath, rels)
	case "sp":
		return extractSpLayer(n, gt, zr, partPath, rels, theme, phMap)
	case "cxnSp":
		return extractCxnSpLayer(n, gt, theme)
	}
	return pptxVisualLayer{}, false
}

// extractCxnSpLayer renders a connector shape (line) as a thin colored rectangle.
func extractCxnSpLayer(n *pptxXMLNode, gt pptxGroupTransform, theme *pptxTheme) (pptxVisualLayer, bool) {
	xfrm := readShapeXfrm(n)
	left, top, cx, cy := gt.apply(xfrm.left, xfrm.top, xfrm.cx, xfrm.cy)

	spPr := n.child("spPr")
	var clr string
	lineWidthEMU := int64(12700) // 1pt default

	if spPr != nil {
		if ln := spPr.child("ln"); ln != nil {
			if w := parseEMU(ln.attr("w")); w > 0 {
				lineWidthEMU = w
			}
			if fill := ln.child("solidFill"); fill != nil {
				clr = resolveColorNode(fill, theme)
			}
		}
	}
	// Fall back to p:style/lnRef for color
	if clr == "" {
		if style := n.child("style"); style != nil {
			if lnRef := style.child("lnRef"); lnRef != nil && lnRef.attr("idx") != "0" {
				clr = resolveColorNode(lnRef, theme)
			}
		}
	}
	if clr == "" {
		return pptxVisualLayer{}, false
	}

	// For zero-dimension lines, use the stroke width as the thin dimension.
	// The group transform scale applies to positions but not to stroke width.
	lineWidthPx := emuToPx(lineWidthEMU)
	if lineWidthPx < 0.5 {
		lineWidthPx = 0.5
	}

	if cx == 0 {
		cx = int64(lineWidthPx * 914400 / 96) // convert px back to EMU for consistent handling
	} else if cy == 0 {
		cy = int64(lineWidthPx * 914400 / 96)
	}
	if cx <= 0 || cy <= 0 {
		return pptxVisualLayer{}, false
	}

	return pptxVisualLayer{
		left: left, top: top, cx: cx, cy: cy,
		rotDeg: xfrm.rotDeg,
		kind:   "text",
		background: "background-color:" + clr + ";",
	}, true
}

func extractPicLayer(n *pptxXMLNode, gt pptxGroupTransform, zr *zip.Reader, partPath string, rels map[string]packageRel) (pptxVisualLayer, bool) {
	xfrm := readShapeXfrm(n)
	embed := findBlipEmbedInTree(n)
	if embed == "" || xfrm.cx <= 0 || xfrm.cy <= 0 {
		return pptxVisualLayer{}, false
	}
	left, top, cx, cy := gt.apply(xfrm.left, xfrm.top, xfrm.cx, xfrm.cy)
	uri := resolvePptxEmbed(zr, partPath, rels, embed)
	if uri == "" {
		return pptxVisualLayer{}, false
	}
	alt := "Slide image"
	if cNvPr := n.findDeep("cNvPr"); cNvPr != nil {
		if a := cNvPr.attr("descr"); a != "" {
			alt = a
		} else if a := cNvPr.attr("name"); a != "" {
			alt = a
		}
	}
	return pptxVisualLayer{
		left: left, top: top, cx: cx, cy: cy,
		rotDeg: xfrm.rotDeg, flipH: xfrm.flipH,
		kind: "image", dataURI: uri, alt: alt,
	}, true
}

func extractSpLayer(n *pptxXMLNode, gt pptxGroupTransform, zr *zip.Reader, partPath string, rels map[string]packageRel, theme *pptxTheme, phMap map[phKey]phInfo) (pptxVisualLayer, bool) {
	xfrm := readShapeXfrm(n)
	left, top, cx, cy := gt.apply(xfrm.left, xfrm.top, xfrm.cx, xfrm.cy)

	// Placeholder position inheritance: if this shape has no geometry of its own,
	// look it up from the layout/master via phMap.
	var phk *phKey
	if ph := n.findDeep("ph"); ph != nil {
		k := phKey{typ: ph.attr("type"), idx: ph.attr("idx")}
		phk = &k
	}
	if (cx <= 0 || cy <= 0) && phk != nil && phMap != nil {
		if info, ok := phMap[*phk]; ok {
			left, top, cx, cy = info.left, info.top, info.cx, info.cy
		}
	}
	if cx <= 0 || cy <= 0 {
		return pptxVisualLayer{}, false
	}

	spPr := n.child("spPr")
	txBody := n.child("txBody")

	// If there's no text body but a blip fill, treat as an image.
	var paraHTML []pptxParaHTML
	if txBody != nil {
		paraHTML = extractTxBodyHTML(txBody, theme)
	}
	if len(paraHTML) == 0 && spPr != nil {
		if embed := findBlipEmbedInTree(spPr); embed != "" {
			uri := resolvePptxEmbed(zr, partPath, rels, embed)
			if uri != "" {
				return pptxVisualLayer{
					left: left, top: top, cx: cx, cy: cy,
					rotDeg: xfrm.rotDeg, flipH: xfrm.flipH,
					kind: "image", dataURI: uri, alt: "Slide graphic",
				}, true
			}
		}
	}
	if len(paraHTML) == 0 {
		// No text content — still render if the shape has a visible fill or border
		// (e.g., a colored rectangle used as a decorative element).
		bg := resolveShapeFill(n, spPr, theme)
		border := parseShapeBorder(spPr, theme)
		if bg == "" && border == "" {
			return pptxVisualLayer{}, false
		}
		return pptxVisualLayer{
			left: left, top: top, cx: cx, cy: cy,
			rotDeg: xfrm.rotDeg, flipH: xfrm.flipH,
			kind: "text", background: bg, border: border,
		}, true
	}

	// Resolve default text style (own shape → placeholder map → fallback).
	fontPt, color, bold := firstRunStyle(n, theme)
	if phk != nil && phMap != nil {
		if info, ok := phMap[*phk]; ok {
			if fontPt == 0 && info.defaultFontPt > 0 {
				fontPt = info.defaultFontPt
			}
			if color == "" && info.defaultColor != "" {
				color = info.defaultColor
			}
			if !bold {
				bold = info.defaultBold
			}
		}
	}
	if fontPt == 0 {
		fontPt = 14
	}
	if color == "" {
		color = "#1e3a5f"
	}
	title := shapeIsTitle(n)
	if title && fontPt < 20 {
		fontPt = 28
	}

	bg := resolveShapeFill(n, spPr, theme)
	border := parseShapeBorder(spPr, theme)

	vertAlign := "top"
	vert := ""
	noWrap := false
	if txBody != nil {
		if bodyPr := txBody.child("bodyPr"); bodyPr != nil {
			switch bodyPr.attr("anchor") {
			case "ctr", "center":
				vertAlign = "middle"
			case "b", "bottom":
				vertAlign = "bottom"
			}
			switch bodyPr.attr("vert") {
			case "vert", "vert90", "eaVert", "mongolianVert", "wordArtVert":
				vert = "vert"
			case "vert270", "wordArtVertRtl":
				vert = "vert270"
			}
			noWrap = bodyPr.attr("wrap") == "none"
		}
	}

	return pptxVisualLayer{
		left: left, top: top, cx: cx, cy: cy,
		rotDeg: xfrm.rotDeg, flipH: xfrm.flipH,
		kind: "text", paraHTML: paraHTML,
		fontPx: ptToPx(fontPt), color: color, bold: bold, isTitle: title,
		background: bg, border: border, vertAlign: vertAlign,
		vert: vert, noWrap: noWrap,
	}, true
}

// resolveShapeFill returns the background CSS for a shape, checking both
// spPr solidFill and the shape's p:style/fillRef reference.
func resolveShapeFill(sp, spPr *pptxXMLNode, theme *pptxTheme) string {
	if bg := shapeFillBackground(spPr, theme); bg != "" {
		return bg
	}
	return shapeStyleFill(sp, theme)
}

type pptxPartRef struct {
	path     string
	relsPath string
	isLayout bool
}

func pptxPartRelsPath(partPath string) string {
	dir, base := path.Split(partPath)
	if dir == "" {
		return "_rels/" + base + ".rels"
	}
	return dir + "_rels/" + base + ".rels"
}

func pptxSlidePartChain(zr *zip.Reader, slidePath string, slideRels map[string]packageRel) []pptxPartRef {
	var chain []pptxPartRef
	layoutPath := pptxRelatedPartPath(slideRels, slidePath, "slideLayout")
	if layoutPath != "" {
		layoutRels, _ := parsePackageRels(zr, pptxPartRelsPath(layoutPath))
		masterPath := pptxRelatedPartPath(layoutRels, layoutPath, "slideMaster")
		if masterPath != "" {
			chain = append(chain, pptxPartRef{path: masterPath, relsPath: pptxPartRelsPath(masterPath)})
		}
		chain = append(chain, pptxPartRef{path: layoutPath, relsPath: pptxPartRelsPath(layoutPath), isLayout: true})
	}
	chain = append(chain, pptxPartRef{path: slidePath, relsPath: pptxSlideRelsPath(slidePath)})
	return chain
}

func resolvePptxEmbed(zr *zip.Reader, slidePath string, rels map[string]packageRel, embedID string) string {
	rel, ok := rels[embedID]
	if !ok {
		return ""
	}
	mediaPath := resolveOOXMLPath(slidePath, rel.Target)
	raw, err := readZipFile(zr, mediaPath)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(mediaPath))
	if ext == ".emf" || ext == ".wmf" {
		return ""
	}
	return dataURIForPath(mediaPath, raw)
}

func renderPptxVisualLayer(layer pptxVisualLayer) string {
	left := emuToPx(layer.left)
	top := emuToPx(layer.top)
	width := emuToPx(layer.cx)
	height := emuToPx(layer.cy)
	z := layer.zIndex + 1

	var xformParts []string
	if layer.rotDeg != 0 {
		xformParts = append(xformParts, fmt.Sprintf("rotate(%.3fdeg)", layer.rotDeg))
	}
	if layer.flipH {
		xformParts = append(xformParts, "scaleX(-1)")
	}
	xformCSS := ""
	if len(xformParts) > 0 {
		xformCSS = "transform:" + strings.Join(xformParts, " ") + ";"
	}

	posCSS := fmt.Sprintf("left:%.2fpx;top:%.2fpx;width:%.2fpx;height:%.2fpx;z-index:%d;", left, top, width, height, z)

	if layer.kind == "image" {
		return fmt.Sprintf(
			`<img class="pptx-layer pptx-layer-image" style="%s%s" src="%s" alt="%s" loading="lazy"/>`,
			posCSS, xformCSS, layer.dataURI, escapeAttr(layer.alt),
		)
	}

	// Pure-fill shape with no text.
	if len(layer.paraHTML) == 0 {
		return fmt.Sprintf(
			`<div class="pptx-layer" style="%s%s%s%s"></div>`,
			posCSS, xformCSS, layer.background, layer.border,
		)
	}

	var textHTML strings.Builder
	for _, para := range layer.paraHTML {
		if strings.TrimSpace(para.html) == "" {
			continue
		}
		if para.style != "" {
			textHTML.WriteString(fmt.Sprintf(`<p style="%s">`, para.style))
		} else {
			textHTML.WriteString("<p>")
		}
		textHTML.WriteString(para.html)
		textHTML.WriteString("</p>")
	}
	if textHTML.Len() == 0 {
		return ""
	}

	weight := "400"
	if layer.bold || layer.isTitle {
		weight = "700"
	}
	defaultAlign := "left"
	if layer.isTitle {
		defaultAlign = "center"
	}

	var extraCSS strings.Builder
	extraCSS.WriteString(layer.background)
	extraCSS.WriteString(layer.border)

	// Vertical text: wrap content in a writing-mode div instead of using flex.
	if layer.vert != "" {
		var wmCSS string
		switch layer.vert {
		case "vert270":
			wmCSS = "writing-mode:vertical-lr;transform:rotate(180deg);transform-origin:center;"
		default:
			wmCSS = "writing-mode:vertical-lr;"
		}
		noWrapCSS := ""
		if layer.noWrap {
			noWrapCSS = "white-space:nowrap;"
		}
		return fmt.Sprintf(
			`<div class="pptx-layer" style="%s%sfont-size:%.2fpx;color:%s;font-weight:%s;%s"><div style="%sheight:100%%;overflow:hidden;">%s</div></div>`,
			posCSS, xformCSS,
			layer.fontPx, layer.color, weight,
			extraCSS.String(),
			wmCSS+noWrapCSS,
			textHTML.String(),
		)
	}

	vertJustify := "flex-start"
	switch layer.vertAlign {
	case "middle":
		vertJustify = "center"
	case "bottom":
		vertJustify = "flex-end"
	}
	noWrapCSS := ""
	if layer.noWrap {
		noWrapCSS = "white-space:nowrap;"
	}

	return fmt.Sprintf(
		`<div class="pptx-layer pptx-layer-text%s" style="%s%sfont-size:%.2fpx;color:%s;font-weight:%s;text-align:%s;justify-content:%s;%s%s">%s</div>`,
		titleClass(layer.isTitle),
		posCSS, xformCSS,
		layer.fontPx, layer.color, weight, defaultAlign, vertJustify,
		extraCSS.String(), noWrapCSS,
		textHTML.String(),
	)
}

func titleClass(isTitle bool) string {
	if isTitle {
		return " pptx-layer-title"
	}
	return ""
}

func escapeHTMLText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func wrapPptxHTMLDocument(fragment string, slideWpx float64) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  :root { color-scheme: light; }
  body {
    font-family: "Segoe UI", system-ui, -apple-system, Roboto, sans-serif;
    margin: 0;
    padding: 1.25rem 1rem 2.5rem;
    background: #e2e8f0;
    color: #0f172a;
  }
  .pptx-slide-wrap { margin: 0 auto 2rem; max-width: 960px; }
  .pptx-slide-label {
    margin: 0 0 0.5rem;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: #64748b;
  }
  .pptx-canvas {
    position: relative;
    width: 100%%;
    background: #fff;
    border-radius: 0.5rem;
    overflow: hidden;
    box-shadow: 0 4px 24px rgba(15,23,42,0.18);
  }
  .pptx-canvas-bg {
    position: absolute;
    inset: 0;
    width: 100%%;
    height: 100%%;
    object-fit: cover;
    z-index: 0;
  }
  .pptx-inner {
    position: absolute;
    top: 0;
    left: 0;
    transform-origin: top left;
  }
  .pptx-layer {
    position: absolute;
    box-sizing: border-box;
    margin: 0;
    padding: 0;
    overflow: hidden;
    line-height: 1.25;
  }
  .pptx-layer-image { object-fit: contain; }
  .pptx-layer-text {
    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    word-wrap: break-word;
    overflow-wrap: break-word;
    padding: 0.1em 0.3em;
  }
  .pptx-layer-text p { margin: 0.1em 0; }
  .pptx-layer-title p { margin: 0.06em 0; }
  .pptx-empty { color: #64748b; font-style: italic; padding: 2rem; }
</style>
</head>
<body>
%s
<script>
(function(){
  var W=%.2f;
  function scale(){
    document.querySelectorAll('.pptx-canvas').forEach(function(c){
      var inner=c.querySelector('.pptx-inner');
      if(inner) inner.style.transform='scale('+(c.clientWidth/W)+')';
    });
  }
  if(typeof ResizeObserver!=='undefined'){
    var ro=new ResizeObserver(scale);
    document.querySelectorAll('.pptx-canvas').forEach(function(c){ro.observe(c);});
  } else {
    window.addEventListener('resize',scale);
  }
  scale();
})();
</script>
</body>
</html>`, fragment, slideWpx)
}
