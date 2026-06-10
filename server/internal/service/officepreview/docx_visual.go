package officepreview

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path"
	"strings"
)

type docxPageLayout struct {
	widthPx      float64
	heightPx     float64
	marginTop    float64
	marginRight  float64
	marginBottom float64
	marginLeft   float64
}

type docxRenderCtx struct {
	zr                *zip.Reader
	docPath           string
	rels              map[string]packageRel
	theme             *docxTheme
	styles            *docxStyleSheet
	numbering         *docxNumbering
	listState         map[string][]int64
	textBoxOriginX    float64
	textBoxOriginY    float64
	textBoxHostActive bool
	page              docxPageLayout
}

type docxTextBoxSpec struct {
	drawing *docxXMLNode
	left    float64
	top     float64
	width   float64
	height  float64
}

func docxTextBoxSpecs(node *docxXMLNode) []docxTextBoxSpec {
	if node == nil {
		return nil
	}
	var specs []docxTextBoxSpec
	for _, drawing := range node.findAllDeep("drawing") {
		if drawing.findDeep("txbxContent") == nil {
			continue
		}
		anchor := drawing.findDeep("anchor")
		if anchor == nil {
			anchor = drawing.findDeep("inline")
		}
		left := docxAnchorOffsetPx(anchor, "positionH")
		top := docxAnchorOffsetPx(anchor, "positionV")
		width, height := docxAnchorExtentPx(anchor)
		specs = append(specs, docxTextBoxSpec{
			drawing: drawing,
			left:    left,
			top:     top,
			width:   width,
			height:  height,
		})
	}
	return specs
}

func docxTextBoxBounds(specs []docxTextBoxSpec) (minLeft, minTop, maxBottom float64) {
	if len(specs) == 0 {
		return 0, 0, 0
	}
	minLeft = specs[0].left
	minTop = specs[0].top
	for _, spec := range specs {
		if spec.left < minLeft {
			minLeft = spec.left
		}
		if spec.top < minTop {
			minTop = spec.top
		}
		if bottom := spec.top + spec.height; bottom > maxBottom {
			maxBottom = bottom
		}
	}
	return minLeft, minTop, maxBottom
}

func (ctx *docxRenderCtx) pushTextBoxHost(node *docxXMLNode) (pop func(), hostMinHeight float64, ok bool) {
	specs := docxTextBoxSpecs(node)
	if len(specs) == 0 || ctx.textBoxHostActive {
		return func() {}, 0, false
	}
	minLeft, minTop, maxBottom := docxTextBoxBounds(specs)
	prevX, prevY, prevActive := ctx.textBoxOriginX, ctx.textBoxOriginY, ctx.textBoxHostActive
	ctx.textBoxOriginX = minLeft
	ctx.textBoxOriginY = minTop
	ctx.textBoxHostActive = true
	return func() {
		ctx.textBoxOriginX = prevX
		ctx.textBoxOriginY = prevY
		ctx.textBoxHostActive = prevActive
	}, maxBottom - minTop + 4, true
}

func convertDocxToVisualHTML(data []byte, filename, mimeType string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}
	docData, err := readZipFile(zr, "word/document.xml")
	if err != nil {
		return "", fmt.Errorf("read document.xml: %w", err)
	}
	root, err := parseDocxXML(docData)
	if err != nil {
		return "", fmt.Errorf("parse document.xml: %w", err)
	}
	body := root.findDeep("body")
	if body == nil {
		return "", fmt.Errorf("no document body")
	}
	rels, _ := parsePackageRels(zr, "word/_rels/document.xml.rels")
	ctx := &docxRenderCtx{
		zr:        zr,
		docPath:   "word/document.xml",
		rels:      rels,
		theme:     loadDocxTheme(zr),
		styles:    loadDocxStyleSheet(zr),
		numbering: loadDocxNumbering(zr),
		listState: make(map[string][]int64),
	}
	page := docxParsePageLayout(body)
	ctx.page = page
	footerRoot := loadDocxFooter(ctx, body.child("sectPr"))
	pageGroups := docxSplitBodyPages(body)
	if len(pageGroups) == 0 {
		return "", fmt.Errorf("no visual content rendered")
	}
	totalPages := len(pageGroups)
	var pageFragments []string
	for pageIdx, nodes := range pageGroups {
		var b strings.Builder
		for i := range nodes {
			b.WriteString(renderDocxBlock(&nodes[i], ctx))
		}
		fragment := strings.TrimSpace(b.String())
		if fragment == "" {
			continue
		}
		fragment += renderDocxPageFooter(footerRoot, ctx, pageIdx+1, totalPages)
		pageFragments = append(pageFragments, fragment)
	}
	if len(pageFragments) == 0 {
		return "", fmt.Errorf("no visual content rendered")
	}
	return wrapDocxHTMLDocument(pageFragments, page, ctx.styles.defaultFontCSS(ctx.theme)), nil
}

func docxSplitBodyPages(body *docxXMLNode) [][]docxXMLNode {
	var pages [][]docxXMLNode
	var current []docxXMLNode
	for i := range body.Children {
		ch := body.Children[i]
		if ch.XMLName.Local == "sectPr" {
			continue
		}
		if ch.XMLName.Local == "p" && docxParagraphStartsNewPage(&ch) {
			if len(current) > 0 {
				pages = append(pages, current)
				current = nil
			}
			if docxIsPageBreakOnlyParagraph(&ch) {
				continue
			}
		}
		current = append(current, ch)
	}
	if len(current) > 0 {
		pages = append(pages, current)
	}
	return pages
}

func docxParagraphStartsNewPage(p *docxXMLNode) bool {
	if docxIsPageBreakOnlyParagraph(p) {
		return true
	}
	for i := range p.Children {
		if p.Children[i].XMLName.Local != "r" {
			continue
		}
		r := &p.Children[i]
		for j := range r.Children {
			switch r.Children[j].XMLName.Local {
			case "lastRenderedPageBreak":
				return true
			case "br":
				if r.Children[j].attr("type") == "page" {
					return true
				}
			case "t":
				if strings.TrimSpace(docxCollectText(&r.Children[j])) != "" {
					return false
				}
			}
		}
	}
	return false
}

func docxIsPageBreakOnlyParagraph(p *docxXMLNode) bool {
	hasPageBreak := false
	hasOther := false
	for i := range p.Children {
		ch := &p.Children[i]
		switch ch.XMLName.Local {
		case "pPr", "bookmarkStart", "bookmarkEnd", "proofErr":
			continue
		case "r":
			for j := range ch.Children {
				rc := &ch.Children[j]
				switch rc.XMLName.Local {
				case "br":
					if rc.attr("type") == "page" {
						hasPageBreak = true
					} else {
						hasOther = true
					}
				case "t":
					if strings.TrimSpace(docxCollectText(rc)) != "" {
						hasOther = true
					}
				case "rPr":
					continue
				default:
					hasOther = true
				}
			}
		default:
			hasOther = true
		}
	}
	return hasPageBreak && !hasOther
}

func docxParagraphHasText(p *docxXMLNode) bool {
	for i := range p.Children {
		ch := &p.Children[i]
		switch ch.XMLName.Local {
		case "r":
			for j := range ch.Children {
				if ch.Children[j].XMLName.Local == "t" && strings.TrimSpace(docxCollectText(&ch.Children[j])) != "" {
					return true
				}
			}
		case "hyperlink":
			for j := range ch.Children {
				if ch.Children[j].XMLName.Local == "r" && docxRunHasText(&ch.Children[j]) {
					return true
				}
			}
		case "sdt":
			if content := ch.child("sdtContent"); content != nil {
				for j := range content.Children {
					if content.Children[j].XMLName.Local == "p" && docxParagraphHasText(&content.Children[j]) {
						return true
					}
				}
			}
		case "AlternateContent", "drawing":
			if docxNodeHasTextBox(ch) {
				return true
			}
		}
	}
	return false
}

func docxNodeHasTextBox(n *docxXMLNode) bool {
	if n == nil {
		return false
	}
	if n.XMLName.Local == "drawing" && n.findDeep("txbxContent") != nil {
		return true
	}
	for _, drawing := range n.findAllDeep("drawing") {
		if drawing.findDeep("txbxContent") != nil {
			return true
		}
	}
	return false
}

func docxRunHasText(r *docxXMLNode) bool {
	for i := range r.Children {
		if r.Children[i].XMLName.Local == "t" && strings.TrimSpace(docxCollectText(&r.Children[i])) != "" {
			return true
		}
	}
	return false
}

func renderDocxBlock(ch *docxXMLNode, ctx *docxRenderCtx) string {
	switch ch.XMLName.Local {
	case "p":
		return renderDocxParagraph(ch, ctx)
	case "tbl":
		return renderDocxTable(ch, ctx)
	case "sdt":
		return renderDocxSDT(ch, ctx, nil)
	default:
		return ""
	}
}

func loadDocxFooter(ctx *docxRenderCtx, sectPr *docxXMLNode) *docxXMLNode {
	if sectPr == nil {
		return nil
	}
	var footerID string
	for i := range sectPr.Children {
		fr := &sectPr.Children[i]
		if fr.XMLName.Local != "footerReference" {
			continue
		}
		id := fr.relAttr("id")
		if id == "" {
			continue
		}
		ftype := fr.attr("type")
		if ftype == "default" || ftype == "" {
			footerID = id
			break
		}
		if footerID == "" {
			footerID = id
		}
	}
	if footerID == "" {
		return nil
	}
	rel, ok := ctx.rels[footerID]
	if !ok {
		return nil
	}
	footerPath := resolveOOXMLPath(ctx.docPath, rel.Target)
	data, err := readZipFile(ctx.zr, footerPath)
	if err != nil {
		return nil
	}
	root, err := parseDocxXML(data)
	if err != nil {
		return nil
	}
	return root
}

func renderDocxPageFooter(footerRoot *docxXMLNode, ctx *docxRenderCtx, pageNum, totalPages int) string {
	if footerRoot == nil {
		return ""
	}
	body := footerRoot.findDeep("ftr")
	if body == nil {
		body = footerRoot
	}
	var b strings.Builder
	for i := range body.Children {
		ch := &body.Children[i]
		switch ch.XMLName.Local {
		case "p":
			b.WriteString(renderDocxFooterParagraph(ch, ctx, pageNum, totalPages))
		case "sdt":
			if content := ch.child("sdtContent"); content != nil {
				for j := range content.Children {
					if content.Children[j].XMLName.Local == "p" {
						b.WriteString(renderDocxFooterParagraph(&content.Children[j], ctx, pageNum, totalPages))
					}
				}
			}
		}
	}
	inner := strings.TrimSpace(b.String())
	if inner == "" {
		return ""
	}
	return fmt.Sprintf(`<div class="docx-footer">%s</div>`, inner)
}

func renderDocxFooterParagraph(p *docxXMLNode, ctx *docxRenderCtx, pageNum, totalPages int) string {
	pPr := p.child("pPr")
	var paraStyleIDs []string
	var paraDefaultRPr *docxXMLNode
	if pPr != nil {
		if ps := pPr.child("pStyle"); ps != nil {
			if id := ps.attr("val"); id != "" {
				paraStyleIDs = append(paraStyleIDs, id)
			}
		}
		paraDefaultRPr = pPr.child("rPr")
	}
	paraCSS := ""
	if pPr != nil {
		paraCSS = docxPPrCSS(pPr, ctx.theme)
	}
	inner := renderDocxRunsWithFields(p, ctx, paraStyleIDs, paraDefaultRPr, pageNum, totalPages)
	if inner == "" {
		return ""
	}
	return fmt.Sprintf(`<p class="docx-footer-p"%s>%s</p>`, docxAttrStyle(paraCSS), inner)
}

func renderDocxRunsWithFields(p *docxXMLNode, ctx *docxRenderCtx, paraStyleIDs []string, paraDefaultRPr *docxXMLNode, pageNum, totalPages int) string {
	var b strings.Builder
	var activeField string
	skipCachedFieldText := false
	for i := range p.Children {
		ch := &p.Children[i]
		switch ch.XMLName.Local {
		case "r":
			for j := range ch.Children {
				rc := &ch.Children[j]
				switch rc.XMLName.Local {
				case "fldChar":
					switch rc.attr("fldCharType") {
					case "begin":
						activeField = ""
						skipCachedFieldText = false
					case "separate":
						if activeField != "" {
							skipCachedFieldText = true
							b.WriteString(docxFieldValueSpan(ch, ctx, paraStyleIDs, paraDefaultRPr, activeField, pageNum, totalPages))
						}
					case "end":
						activeField = ""
						skipCachedFieldText = false
					}
				case "instrText":
					instr := strings.ToUpper(strings.TrimSpace(docxCollectText(rc)))
					switch {
					case strings.Contains(instr, "NUMPAGES"):
						activeField = "numpages"
					case strings.Contains(instr, "PAGE"):
						activeField = "page"
					}
				case "t":
					if skipCachedFieldText {
						continue
					}
					text := docxCollectText(rc)
					resolved := ctx.styles.resolveRun(ch, paraStyleIDs, paraDefaultRPr, ctx.theme)
					if resolved.style != "" {
						fmt.Fprintf(&b, `<span style="%s">%s</span>`, resolved.style, escapeHTMLText(text))
					} else {
						b.WriteString(escapeHTMLText(text))
					}
				default:
					continue
				}
			}
		case "hyperlink":
			b.WriteString(renderDocxHyperlink(ch, ctx, paraStyleIDs, paraDefaultRPr))
		}
	}
	return b.String()
}

func docxFieldValueSpan(r *docxXMLNode, ctx *docxRenderCtx, paraStyleIDs []string, paraDefaultRPr *docxXMLNode, field string, pageNum, totalPages int) string {
	text := ""
	switch field {
	case "page":
		text = fmt.Sprintf("%d", pageNum)
	case "numpages":
		text = fmt.Sprintf("%d", totalPages)
	default:
		return ""
	}
	resolved := ctx.styles.resolveRun(r, paraStyleIDs, paraDefaultRPr, ctx.theme)
	if resolved.style != "" {
		return fmt.Sprintf(`<span style="%s">%s</span>`, resolved.style, escapeHTMLText(text))
	}
	return escapeHTMLText(text)
}

func docxParsePageLayout(body *docxXMLNode) docxPageLayout {
	layout := docxPageLayout{
		widthPx:      twipsToPx(12240),
		heightPx:     twipsToPx(15840),
		marginTop:    twipsToPx(1440),
		marginRight:  twipsToPx(1440),
		marginBottom: twipsToPx(1440),
		marginLeft:   twipsToPx(1440),
	}
	sectPr := body.child("sectPr")
	if sectPr == nil {
		return layout
	}
	if pgSz := sectPr.child("pgSz"); pgSz != nil {
		if w := parseDocxInt(pgSz.attr("w")); w > 0 {
			layout.widthPx = twipsToPx(w)
		}
		if h := parseDocxInt(pgSz.attr("h")); h > 0 {
			layout.heightPx = twipsToPx(h)
		}
	}
	if pgMar := sectPr.child("pgMar"); pgMar != nil {
		if v := parseDocxInt(pgMar.attr("top")); v > 0 {
			layout.marginTop = twipsToPx(v)
		}
		if v := parseDocxInt(pgMar.attr("right")); v > 0 {
			layout.marginRight = twipsToPx(v)
		}
		if v := parseDocxInt(pgMar.attr("bottom")); v > 0 {
			layout.marginBottom = twipsToPx(v)
		}
		if v := parseDocxInt(pgMar.attr("left")); v > 0 {
			layout.marginLeft = twipsToPx(v)
		}
	}
	return layout
}

func renderDocxSDT(sdt *docxXMLNode, ctx *docxRenderCtx, paraStyleIDs []string) string {
	content := sdt.child("sdtContent")
	if content == nil {
		return ""
	}
	var sdtDefaultRPr *docxXMLNode
	if sdtPr := sdt.child("sdtPr"); sdtPr != nil {
		sdtDefaultRPr = sdtPr.child("rPr")
	}
	var b strings.Builder
	for i := range content.Children {
		ch := &content.Children[i]
		switch ch.XMLName.Local {
		case "p":
			b.WriteString(renderDocxParagraph(ch, ctx))
		case "tbl":
			b.WriteString(renderDocxTable(ch, ctx))
		case "sdt":
			b.WriteString(renderDocxSDT(ch, ctx, paraStyleIDs))
		case "r":
			b.WriteString(renderDocxRun(ch, ctx, paraStyleIDs, sdtDefaultRPr))
		case "hyperlink":
			b.WriteString(renderDocxHyperlink(ch, ctx, paraStyleIDs, sdtDefaultRPr))
		}
	}
	rendered := b.String()
	if strings.TrimSpace(rendered) != "" {
		return rendered
	}
	if sdtPr := sdt.child("sdtPr"); sdtPr != nil {
		if sdtPr.child("showingPlcHdr") != nil {
			return ""
		}
	}
	return rendered
}

func renderDocxParagraph(p *docxXMLNode, ctx *docxRenderCtx) string {
	pPr := p.child("pPr")
	var paraStyleIDs []string
	var paraDefaultRPr *docxXMLNode
	if pPr != nil {
		if ps := pPr.child("pStyle"); ps != nil {
			if id := ps.attr("val"); id != "" {
				paraStyleIDs = append(paraStyleIDs, id)
			}
		}
		paraDefaultRPr = pPr.child("rPr")
	}
	resolved := ctx.styles.resolveParagraph(p, ctx.theme, ctx.numbering, ctx.listState)
	style := resolved.style
	if extra := docxEmptyParagraphExtraCSS(p, pPr); extra != "" {
		if style != "" {
			style += ";" + extra
		} else {
			style = extra
		}
	}

	var runs strings.Builder
	pop, hostMinH, hasHost := ctx.pushTextBoxHost(p)
	if hasHost {
		defer pop()
	}
	for i := range p.Children {
		ch := &p.Children[i]
		switch ch.XMLName.Local {
		case "r":
			runs.WriteString(renderDocxRun(ch, ctx, paraStyleIDs, paraDefaultRPr))
		case "hyperlink":
			runs.WriteString(renderDocxHyperlink(ch, ctx, paraStyleIDs, paraDefaultRPr))
		case "sdt":
			runs.WriteString(renderDocxSDT(ch, ctx, paraStyleIDs))
		}
	}
	inner := runs.String()
	if hasHost {
		inner = fmt.Sprintf(
			`<div class="docx-textbox-host"%s>%s</div>`,
			docxAttrStyle(fmt.Sprintf("position:relative;min-height:%.2fpx", hostMinH)),
			inner,
		)
	}
	if strings.TrimSpace(inner) == "" && resolved.listMarker == "" && len(docxTextBoxSpecs(p)) == 0 {
		return fmt.Sprintf(`<%s class="docx-p"%s></%s>`, resolved.tag, docxAttrStyle(style), resolved.tag)
	}
	if resolved.listMarker != "" {
		markerStyle := "display:inline-block;min-width:1.5em"
		inner = fmt.Sprintf(`<span class="docx-list-marker" style="%s">%s</span>%s`, markerStyle, escapeHTMLText(resolved.listMarker), inner)
	}
	return fmt.Sprintf(`<%s class="docx-p"%s>%s</%s>`, resolved.tag, docxAttrStyle(style), inner, resolved.tag)
}

func docxEmptyParagraphExtraCSS(p *docxXMLNode, pPr *docxXMLNode) string {
	// An empty paragraph occupies one line in Word, which is how documents
	// create vertical spacing between blocks. Without a min-height the empty
	// <p> collapses to nothing and those gaps disappear. Skip paragraphs that
	// carry a drawing/picture so anchored shapes don't gain stray height.
	if docxParagraphHasText(p) || len(p.findAllDeep("drawing")) > 0 || len(p.findAllDeep("pict")) > 0 {
		return ""
	}
	if pPr == nil {
		return "min-height:1.15em"
	}
	if sp := pPr.child("spacing"); sp != nil {
		if line := parseDocxInt(sp.attr("line")); line > 0 {
			rule := sp.attr("lineRule")
			if rule == "auto" || rule == "" {
				return fmt.Sprintf("min-height:%.2fem", float64(line)/240.0)
			}
			return fmt.Sprintf("min-height:%.2fpx", twipsToPx(line))
		}
	}
	return "min-height:1.15em"
}

func renderDocxHyperlink(hl *docxXMLNode, ctx *docxRenderCtx, paraStyleIDs []string, paraDefaultRPr *docxXMLNode) string {
	href := ""
	if id := hl.relAttr("id"); id != "" {
		if rel, ok := ctx.rels[id]; ok {
			href = rel.Target
		}
	}
	var inner strings.Builder
	for i := range hl.Children {
		if hl.Children[i].XMLName.Local == "r" {
			inner.WriteString(renderDocxRun(&hl.Children[i], ctx, paraStyleIDs, paraDefaultRPr))
		}
	}
	content := inner.String()
	if content == "" {
		return ""
	}
	if href == "" {
		return content
	}
	linkStyle := ""
	if def := ctx.styles.styles["Hyperlink"]; def != nil {
		linkStyle = docxRPrCSS(def.rPr, ctx.theme)
	}
	return fmt.Sprintf(`<a href="%s"%s>%s</a>`, escapeAttr(href), docxAttrStyle(linkStyle), content)
}

func renderDocxRun(r *docxXMLNode, ctx *docxRenderCtx, paraStyleIDs []string, paraDefaultRPr *docxXMLNode) string {
	resolved := ctx.styles.resolveRun(r, paraStyleIDs, paraDefaultRPr, ctx.theme)
	var b strings.Builder
	for i := range r.Children {
		ch := &r.Children[i]
		switch ch.XMLName.Local {
		case "t":
			b.WriteString(escapeHTMLText(docxCollectText(ch)))
		case "tab":
			b.WriteString("&emsp;")
		case "br":
			if ch.attr("type") != "page" {
				b.WriteString("<br/>")
			}
		case "lastRenderedPageBreak":
			// Page split is handled when grouping body children.
		case "drawing":
			if ch.findDeep("txbxContent") != nil {
				b.WriteString(renderDocxDrawingTextBox(ch, ctx, paraStyleIDs, paraDefaultRPr))
			} else {
				b.WriteString(renderDocxDrawing(ch, ctx))
			}
		case "AlternateContent":
			b.WriteString(renderDocxAlternateContent(ch, ctx, paraStyleIDs, paraDefaultRPr))
		case "pict":
			if ch.findDeep("txbxContent") != nil {
				b.WriteString(renderDocxPictTextBox(ch, ctx, paraStyleIDs, paraDefaultRPr))
			} else {
				b.WriteString(renderDocxPict(ch, ctx))
			}
		case "fldSimple":
			b.WriteString(escapeHTMLText(docxCollectText(ch)))
		}
	}
	text := b.String()
	if text == "" {
		return ""
	}
	if (strings.Contains(text, `class="docx-image"`) || strings.Contains(text, `class="docx-textbox"`)) && !strings.Contains(text, "<span") {
		return text
	}
	if resolved.style == "" {
		return text
	}
	return fmt.Sprintf(`<span style="%s">%s</span>`, resolved.style, text)
}

func renderDocxAlternateContent(ac *docxXMLNode, ctx *docxRenderCtx, paraStyleIDs []string, paraDefaultRPr *docxXMLNode) string {
	branch := ac.child("Choice")
	if branch == nil {
		branch = ac.child("Fallback")
	}
	if branch == nil {
		return ""
	}
	var b strings.Builder
	for _, drawing := range branch.findAllDeep("drawing") {
		if drawing.findDeep("txbxContent") != nil {
			b.WriteString(renderDocxDrawingTextBox(drawing, ctx, paraStyleIDs, paraDefaultRPr))
		} else {
			b.WriteString(renderDocxDrawing(drawing, ctx))
		}
	}
	for _, pict := range branch.findAllDeep("pict") {
		if pict.findDeep("txbxContent") != nil {
			b.WriteString(renderDocxPictTextBox(pict, ctx, paraStyleIDs, paraDefaultRPr))
		} else {
			b.WriteString(renderDocxPict(pict, ctx))
		}
	}
	return b.String()
}

// renderDocxTextBoxParagraphs renders the block content of a text box (a text
// box can hold tables and nested content, not just paragraphs), dropping
// trailing empty paragraphs. Authors commonly leave blank lines after the
// caption; in a fixed-height, vertically-centered box those blank lines push
// the real content past the clip boundary and hide the first line.
func renderDocxTextBoxParagraphs(txbxContent *docxXMLNode, ctx *docxRenderCtx) string {
	var blocks []*docxXMLNode
	for i := range txbxContent.Children {
		switch txbxContent.Children[i].XMLName.Local {
		case "p", "tbl", "sdt":
			blocks = append(blocks, &txbxContent.Children[i])
		}
	}
	for len(blocks) > 0 {
		last := blocks[len(blocks)-1]
		if last.XMLName.Local != "p" || docxParagraphHasText(last) || len(docxTextBoxSpecs(last)) > 0 {
			break
		}
		blocks = blocks[:len(blocks)-1]
	}
	var inner strings.Builder
	for _, blk := range blocks {
		switch blk.XMLName.Local {
		case "p":
			inner.WriteString(renderDocxParagraph(blk, ctx))
		case "tbl":
			inner.WriteString(renderDocxTable(blk, ctx))
		case "sdt":
			inner.WriteString(renderDocxSDT(blk, ctx, nil))
		}
	}
	return inner.String()
}

func renderDocxDrawingTextBox(drawing *docxXMLNode, ctx *docxRenderCtx, paraStyleIDs []string, paraDefaultRPr *docxXMLNode) string {
	txbxContent := drawing.findDeep("txbxContent")
	if txbxContent == nil {
		return renderDocxDrawing(drawing, ctx)
	}
	anchor := drawing.findDeep("anchor")
	if anchor == nil {
		anchor = drawing.findDeep("inline")
	}
	left := docxAnchorOffsetPx(anchor, "positionH") - ctx.textBoxOriginX
	top := docxAnchorOffsetPx(anchor, "positionV") - ctx.textBoxOriginY
	width, height := docxAnchorExtentPx(anchor)
	boxStyle := docxTextBoxStyle(drawing, ctx.theme)
	content := strings.TrimSpace(renderDocxTextBoxParagraphs(txbxContent, ctx))
	if content == "" {
		return ""
	}
	posStyle := "position:absolute;"
	if ctx.textBoxHostActive || left != 0 || top != 0 {
		posStyle += fmt.Sprintf("left:%.2fpx;top:%.2fpx;", left, top)
	}
	if width > 0 {
		posStyle += fmt.Sprintf("width:%.2fpx;", width)
	}
	if height > 0 {
		// Word text boxes don't clip overflowing text (unless set to a fixed
		// autofit); use min-height so longer content grows the box instead of
		// being cut off.
		posStyle += fmt.Sprintf("min-height:%.2fpx;", height)
	}
	return fmt.Sprintf(
		`<div class="docx-textbox" style="%s%s">%s</div>`,
		posStyle,
		boxStyle,
		content,
	)
}

func docxTextBoxStyle(drawing *docxXMLNode, theme *docxTheme) string {
	wsp := drawing.findDeep("wsp")
	if wsp == nil {
		return "box-sizing:border-box;"
	}
	var parts []string
	parts = append(parts, "box-sizing:border-box")
	spPr := wsp.child("spPr")
	if spPr == nil {
		return strings.Join(parts, ";") + ";"
	}
	if fill := spPr.child("solidFill"); fill != nil {
		if clr := resolveDocxSolidFillColor(fill, theme); clr != "" {
			parts = append(parts, "background-color:"+clr)
		}
	}
	// Only draw a border when the shape actually defines a line fill. A
	// <a:ln> carrying <a:noFill/> means "no outline"; drawing one anyway
	// boxes every text box (e.g. an entire résumé laid out in text boxes).
	if ln := spPr.child("ln"); ln != nil && ln.child("noFill") == nil {
		widthPx := docxDrawingLnWidthPx(ln)
		borderClr := "#000000"
		if fill := ln.child("solidFill"); fill != nil {
			if clr := resolveDocxSolidFillColor(fill, theme); clr != "" {
				borderClr = clr
			}
		}
		parts = append(parts, fmt.Sprintf("border:%.2fpx solid %s", widthPx, borderClr))
	}
	if bodyPr := wsp.child("bodyPr"); bodyPr != nil {
		switch bodyPr.attr("anchor") {
		case "ctr":
			parts = append(parts, "display:flex", "flex-direction:column", "justify-content:center")
		case "b":
			parts = append(parts, "display:flex", "flex-direction:column", "justify-content:flex-end")
		default:
			parts = append(parts, "display:flex", "flex-direction:column", "justify-content:flex-start")
		}
	}
	return strings.Join(parts, ";") + ";"
}

func renderDocxPictTextBox(pict *docxXMLNode, ctx *docxRenderCtx, paraStyleIDs []string, paraDefaultRPr *docxXMLNode) string {
	txbxContent := pict.findDeep("txbxContent")
	if txbxContent == nil {
		return renderDocxPict(pict, ctx)
	}
	content := strings.TrimSpace(renderDocxTextBoxParagraphs(txbxContent, ctx))
	if content == "" {
		return ""
	}
	return fmt.Sprintf(`<div class="docx-textbox" style="display:inline-block;vertical-align:top;%s">%s</div>`, docxVMLShapeStyle(pict), content)
}

func docxVMLShapeStyle(pict *docxXMLNode) string {
	for _, shape := range pict.findAllDeep("shape") {
		style := shape.attr("style")
		if style == "" {
			continue
		}
		var parts []string
		for _, decl := range strings.Split(style, ";") {
			decl = strings.TrimSpace(decl)
			if decl == "" {
				continue
			}
			kv := strings.SplitN(decl, ":", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			switch key {
			case "margin-left", "margin-top", "width", "height":
				parts = append(parts, key+":"+val)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, ";") + ";"
		}
	}
	return ""
}

func renderDocxDrawing(drawing *docxXMLNode, ctx *docxRenderCtx) string {
	embed := findDocxBlipEmbed(drawing)
	if embed == "" {
		return renderDocxAnchoredShape(drawing, ctx)
	}
	uri := resolveDocxEmbed(ctx, embed)
	if uri == "" {
		return ""
	}
	widthPx, heightPx := docxDrawingSizePx(drawing)
	style := "max-width:100%;height:auto;display:block;margin:0 auto;"
	if widthPx > 0 {
		style += fmt.Sprintf("width:%.2fpx;", widthPx)
	}
	if heightPx > 0 {
		style += fmt.Sprintf("height:%.2fpx;", heightPx)
	}
	return fmt.Sprintf(`<img class="docx-image" src="%s" alt="" style="%s"/>`, uri, style)
}

// renderDocxAnchoredShape renders an anchored DrawingML shape that carries a
// solid fill but no image or text (e.g. a decorative side bar) as an absolutely
// positioned colored box on the page. Shapes without an anchor or a fill render
// as nothing, matching the prior behavior.
func renderDocxAnchoredShape(drawing *docxXMLNode, ctx *docxRenderCtx) string {
	anchor := drawing.findDeep("anchor")
	if anchor == nil {
		return ""
	}
	wsp := drawing.findDeep("wsp")
	if wsp == nil {
		return ""
	}
	spPr := wsp.child("spPr")
	if spPr == nil {
		return ""
	}
	fill := resolveDocxSolidFillColor(spPr.child("solidFill"), ctx.theme)
	if fill == "" {
		return ""
	}
	width, height := docxAnchorExtentPx(anchor)
	if width <= 0 || height <= 0 {
		return ""
	}
	var left, top float64
	if ctx.textBoxHostActive {
		// The shape shares a text-box host's coordinate system with sibling
		// anchored boxes, so position it relative to that host origin (the same
		// way text boxes are placed) instead of the page.
		left = docxAnchorOffsetPxSigned(anchor, "positionH") - ctx.textBoxOriginX
		top = docxAnchorOffsetPxSigned(anchor, "positionV") - ctx.textBoxOriginY
	} else {
		left = docxAnchorBasePx(anchor, "positionH", ctx.page) + docxAnchorOffsetPxSigned(anchor, "positionH")
		top = docxAnchorBasePx(anchor, "positionV", ctx.page) + docxAnchorOffsetPxSigned(anchor, "positionV")
	}
	z := "z-index:0;"
	if anchor.attr("behindDoc") == "1" {
		z = "z-index:-1;"
	}
	return fmt.Sprintf(
		`<div class="docx-shape" style="position:absolute;left:%.2fpx;top:%.2fpx;width:%.2fpx;height:%.2fpx;background-color:%s;%s"></div>`,
		left, top, width, height, fill, z,
	)
}

// docxAnchorBasePx returns the page-relative origin (in px) that an anchor's
// offset is measured from, based on its relativeFrom reference frame.
func docxAnchorBasePx(anchor *docxXMLNode, posLocal string, page docxPageLayout) float64 {
	pos := anchor.child(posLocal)
	if pos == nil {
		return 0
	}
	switch pos.attr("relativeFrom") {
	case "page", "leftMargin", "topMargin":
		return 0
	case "margin":
		if posLocal == "positionV" {
			return page.marginTop
		}
		return page.marginLeft
	default: // column/character/text (H) or paragraph/line/text (V)
		if posLocal == "positionV" {
			return page.marginTop
		}
		return page.marginLeft
	}
}

// docxAnchorOffsetPxSigned reads a posOffset allowing negative values, used for
// shapes that extend into the page margins.
func docxAnchorOffsetPxSigned(anchor *docxXMLNode, posLocal string) float64 {
	if anchor == nil {
		return 0
	}
	pos := anchor.child(posLocal)
	if pos == nil {
		return 0
	}
	if off := pos.child("posOffset"); off != nil {
		return emuToPx(parseDocxInt(strings.TrimSpace(docxCollectText(off))))
	}
	return 0
}

func renderDocxPict(pict *docxXMLNode, ctx *docxRenderCtx) string {
	for _, im := range pict.findAllDeep("imagedata") {
		if id := im.relAttr("id"); id != "" {
			uri := resolveDocxEmbed(ctx, id)
			if uri != "" {
				return fmt.Sprintf(`<img class="docx-image" src="%s" alt="" style="max-width:100%%;height:auto;"/>`, uri)
			}
		}
	}
	return ""
}

func findDocxBlipEmbed(node *docxXMLNode) string {
	if node.XMLName.Local == "blip" {
		if id := node.relAttr("embed"); id != "" {
			return id
		}
	}
	for i := range node.Children {
		if id := findDocxBlipEmbed(&node.Children[i]); id != "" {
			return id
		}
	}
	return ""
}

func docxDrawingSizePx(drawing *docxXMLNode) (float64, float64) {
	for _, local := range []string{"extent", "ext"} {
		if ext := drawing.findDeep(local); ext != nil {
			cx := parseDocxInt(ext.attr("cx"))
			cy := parseDocxInt(ext.attr("cy"))
			if cx > 0 && cy > 0 {
				return emuToPx(cx), emuToPx(cy)
			}
		}
	}
	return 0, 0
}

func resolveDocxEmbed(ctx *docxRenderCtx, embedID string) string {
	rel, ok := ctx.rels[embedID]
	if !ok {
		return ""
	}
	mediaPath := resolveOOXMLPath(ctx.docPath, rel.Target)
	raw, err := readZipFile(ctx.zr, mediaPath)
	if err != nil {
		return ""
	}
	ext := strings.ToLower(path.Ext(mediaPath))
	if ext == ".emf" || ext == ".wmf" {
		return ""
	}
	return dataURIForPath(mediaPath, raw)
}

type docxTableRenderState struct {
	styleID     string
	rowCnfType  string
	mergeActive []int
}

func docxDirectTableRows(tbl *docxXMLNode) []*docxXMLNode {
	var rows []*docxXMLNode
	for i := range tbl.Children {
		if tbl.Children[i].XMLName.Local == "tr" {
			rows = append(rows, &tbl.Children[i])
		}
	}
	return rows
}

func docxTableStyleID(tbl *docxXMLNode) string {
	if tblPr := tbl.child("tblPr"); tblPr != nil {
		if ts := tblPr.child("tblStyle"); ts != nil {
			return ts.attr("val")
		}
	}
	return ""
}

func docxTcGridSpan(tc *docxXMLNode) int {
	if tcPr := tc.child("tcPr"); tcPr != nil {
		if gs := tcPr.child("gridSpan"); gs != nil {
			if n := parseDocxInt(gs.attr("val")); n > 0 {
				return int(n)
			}
		}
	}
	return 1
}

func docxTcVMergeVal(tc *docxXMLNode) string {
	if tcPr := tc.child("tcPr"); tcPr != nil {
		if vm := tcPr.child("vMerge"); vm != nil {
			if v := vm.attr("val"); v != "" {
				return v
			}
			return "continue"
		}
	}
	return ""
}

func docxCountVMergeSpan(rows []*docxXMLNode, rowIdx, cellIdx int) int {
	if rowIdx >= len(rows) {
		return 1
	}
	cells := docxDirectRowCells(rows[rowIdx])
	if cellIdx >= len(cells) {
		return 1
	}
	if docxTcVMergeVal(cells[cellIdx]) != "restart" {
		return 1
	}
	span := 1
	for ri := rowIdx + 1; ri < len(rows); ri++ {
		nextCells := docxDirectRowCells(rows[ri])
		if cellIdx >= len(nextCells) {
			break
		}
		if docxTcVMergeVal(nextCells[cellIdx]) == "continue" {
			span++
			continue
		}
		break
	}
	return span
}

func docxDirectRowCells(tr *docxXMLNode) []*docxXMLNode {
	var cells []*docxXMLNode
	for i := range tr.Children {
		if tr.Children[i].XMLName.Local == "tc" {
			cells = append(cells, &tr.Children[i])
		}
	}
	return cells
}

func docxTrCnfStyleType(tr *docxXMLNode) string {
	if trPr := tr.child("trPr"); trPr != nil {
		return docxCnfStylePrType(trPr.child("cnfStyle"))
	}
	return ""
}

func docxTcCnfStyleType(tc *docxXMLNode) string {
	if tcPr := tc.child("tcPr"); tcPr != nil {
		return docxCnfStylePrType(tcPr.child("cnfStyle"))
	}
	return ""
}

func renderDocxTable(tbl *docxXMLNode, ctx *docxRenderCtx) string {
	var tableStyle string
	tblStyleID := docxTableStyleID(tbl)
	if tblPr := tbl.child("tblPr"); tblPr != nil {
		if def := ctx.styles.styles[tblStyleID]; def != nil && def.tblPr != nil {
			tableStyle = docxBordersCSS(def.tblPr.child("tblBorders"), ctx.theme)
		}
		tableStyle += docxBordersCSS(tblPr.child("tblBorders"), ctx.theme)
		if tw := tblPr.child("tblW"); tw != nil && tw.attr("type") == "dxa" {
			if w := parseDocxInt(tw.attr("w")); w > 0 {
				tableStyle += fmt.Sprintf("width:%.2fpx;", twipsToPx(w))
			}
		}
	}
	rows := docxDirectTableRows(tbl)
	var b strings.Builder
	fmt.Fprintf(&b, `<table class="docx-table" style="border-collapse:collapse;%s">`, tableStyle)
	mergeActive := []int{}
	for rowIdx, tr := range rows {
		var rowHTML string
		rowHTML, mergeActive = renderDocxTableRow(tr, ctx, docxTableRenderState{
			styleID:     tblStyleID,
			rowCnfType:  docxTrCnfStyleType(tr),
			mergeActive: mergeActive,
		}, rows, rowIdx)
		b.WriteString(rowHTML)
	}
	b.WriteString("</table>")
	return b.String()
}

func docxTrHeightCSS(tr *docxXMLNode) string {
	if trPr := tr.child("trPr"); trPr != nil {
		if th := trPr.child("trHeight"); th != nil {
			if h := parseDocxInt(th.attr("val")); h > 0 {
				return fmt.Sprintf("height:%.2fpx", twipsToPx(h))
			}
		}
	}
	return ""
}

func renderDocxTableRow(tr *docxXMLNode, ctx *docxRenderCtx, tstate docxTableRenderState, rows []*docxXMLNode, rowIdx int) (string, []int) {
	var b strings.Builder
	rowStyle := docxTrHeightCSS(tr)
	if rowStyle != "" {
		fmt.Fprintf(&b, `<tr style="%s">`, rowStyle)
	} else {
		b.WriteString("<tr>")
	}
	colIdx := 0
	cellIdx := 0
	mergeActive := append([]int(nil), tstate.mergeActive...)
	for i := range tr.Children {
		if tr.Children[i].XMLName.Local != "tc" {
			continue
		}
		tc := &tr.Children[i]
		for colIdx < len(mergeActive) && mergeActive[colIdx] > 0 {
			mergeActive[colIdx]--
			colIdx++
		}
		vm := docxTcVMergeVal(tc)
		if vm == "continue" {
			colIdx += docxTcGridSpan(tc)
			cellIdx++
			continue
		}
		rowSpan := 1
		if vm == "restart" {
			rowSpan = docxCountVMergeSpan(rows, rowIdx, cellIdx)
			for len(mergeActive) <= colIdx {
				mergeActive = append(mergeActive, 0)
			}
			mergeActive[colIdx] = rowSpan - 1
		}
		cnfType := docxTcCnfStyleType(tc)
		if cnfType == "" {
			cnfType = tstate.rowCnfType
		}
		b.WriteString(renderDocxTableCell(tc, ctx, tstate.styleID, cnfType, rowSpan))
		colIdx += docxTcGridSpan(tc)
		cellIdx++
	}
	b.WriteString("</tr>")
	return b.String(), mergeActive
}

func renderDocxTableCell(tc *docxXMLNode, ctx *docxRenderCtx, tblStyleID, cnfType string, rowSpan int) string {
	var cellStyle strings.Builder
	if tblStyleID != "" && cnfType != "" {
		cellStyle.WriteString(ctx.styles.tblStyleConditionalCSS(tblStyleID, cnfType, ctx.theme))
	}
	if tcPr := tc.child("tcPr"); tcPr != nil {
		if tw := tcPr.child("tcW"); tw != nil && tw.attr("type") == "dxa" {
			if w := parseDocxInt(tw.attr("w")); w > 0 {
				fmt.Fprintf(&cellStyle, "width:%.2fpx;", twipsToPx(w))
			}
		}
		if shd := tcPr.child("shd"); shd != nil {
			cellStyle.WriteString(docxShdCSS(shd, ctx.theme))
		}
		if va := tcPr.child("vAlign"); va != nil {
			switch va.attr("val") {
			case "center":
				cellStyle.WriteString("vertical-align:middle;")
			case "bottom":
				cellStyle.WriteString("vertical-align:bottom;")
			}
		}
		cellStyle.WriteString(docxBordersCSS(tcPr.child("tcBorders"), ctx.theme))
		if mar := docxTcMarCSS(tcPr.child("tcMar")); mar != "" {
			cellStyle.WriteString(mar)
			cellStyle.WriteString(";")
		}
	}
	var inner strings.Builder
	// Floating text boxes are hosted by the paragraph that anchors them
	// (see renderDocxParagraph), which keeps them in normal flow below any
	// preceding content in the cell. A cell-wide host would collapse every
	// box to the cell-top origin and overlap that content.
	for i := range tc.Children {
		ch := &tc.Children[i]
		switch ch.XMLName.Local {
		case "p":
			inner.WriteString(renderDocxParagraph(ch, ctx))
		case "tbl":
			inner.WriteString(renderDocxTable(ch, ctx))
		case "sdt":
			inner.WriteString(renderDocxSDT(ch, ctx, nil))
		}
	}
	innerHTML := inner.String()
	spanAttr := ""
	if rowSpan > 1 {
		spanAttr = fmt.Sprintf(` rowspan="%d"`, rowSpan)
	}
	return fmt.Sprintf(`<td class="docx-td"%s%s>%s</td>`, spanAttr, docxAttrStyle(cellStyle.String()), innerHTML)
}

func docxAttrStyle(css string) string {
	css = strings.TrimSpace(css)
	if css == "" {
		return ""
	}
	if !strings.HasSuffix(css, ";") {
		css += ";"
	}
	return ` style="` + css + `"`
}

func wrapDocxHTMLDocument(pages []string, page docxPageLayout, defaultFontCSS string) string {
	contentWidth := page.widthPx - page.marginLeft - page.marginRight
	if contentWidth <= 0 {
		contentWidth = page.widthPx
	}
	pageWidth := contentWidth + page.marginLeft + page.marginRight
	var pageHTML strings.Builder
	for i, fragment := range pages {
		if i > 0 {
			pageHTML.WriteString("\n")
		}
		pageStyle := fmt.Sprintf(
			"min-height:%.2fpx;%s",
			page.heightPx,
			defaultFontCSS,
		)
		fmt.Fprintf(&pageHTML, `<div class="docx-page"%s>%s</div>`, docxAttrStyle(pageStyle), fragment)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
  :root { color-scheme: light; }
  body {
    margin: 0;
    padding: 1.25rem 1rem 2.5rem;
    background: #e2e8f0;
    color: #0f172a;
  }
  .docx-pages, body { display: flex; flex-direction: column; gap: 1.5rem; align-items: center; }
  .docx-page {
    margin: 0;
    width: %.2fpx;
    max-width: 100%%;
    background: #fff;
    box-shadow: 0 4px 24px rgba(15,23,42,0.12);
    border-radius: 0.25rem;
    padding: %.2fpx %.2fpx %.2fpx %.2fpx;
    box-sizing: border-box;
    word-wrap: break-word;
    overflow-wrap: break-word;
    overflow: hidden;
    position: relative;
  }
  .docx-shape { pointer-events: none; }
  .docx-p { margin: 0; }
  .docx-p + .docx-p { margin-top: 0; }
  .docx-table { width: 100%%; max-width: 100%%; margin: 0.5rem 0; table-layout: fixed; }
  /* OOXML cell widths (tcW) are the full cell width including padding and
     borders, so use border-box; content-box would inflate every column and
     push nested tables past their container. */
  .docx-td { vertical-align: top; box-sizing: border-box; }
  .docx-image { margin: 0.15rem auto; display: block; max-width: 100%%; height: auto; }
  .docx-textbox { margin: 0; }
  .docx-textbox-host { position: relative; }
  .docx-textbox-host > .docx-p { margin: 0; }
  .docx-textbox .docx-p { margin: 0; }
  h1.docx-p, h2.docx-p, h3.docx-p, h4.docx-p, h5.docx-p, h6.docx-p { font-weight: inherit; font-size: inherit; }
  .docx-footer {
    position: absolute;
    left: %.2fpx;
    right: %.2fpx;
    bottom: %.2fpx;
    text-align: right;
    color: #808080;
    font-size: 9pt;
  }
  .docx-footer-p { margin: 0; }
  .docx-p h1, .docx-p h2, .docx-p h3, .docx-p h4, .docx-p h5, .docx-p h6 { margin: 0; font: inherit; color: inherit; }
  a { color: inherit; }
</style>
</head>
<body>
%s
</body>
</html>`, pageWidth,
		page.marginTop, page.marginRight, page.marginBottom, page.marginLeft,
		page.marginLeft, page.marginRight, page.marginBottom,
		pageHTML.String())
}
