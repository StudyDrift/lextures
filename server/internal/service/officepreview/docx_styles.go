package officepreview

import (
	"archive/zip"
	"fmt"
	"strings"
)

type docxResolvedPPr struct {
	tag        string // p, h1..h6
	style      string
	listMarker string
}

type docxResolvedRPr struct {
	style string
}

type docxStyleDef struct {
	styleType   string
	styleID     string
	name        string
	basedOn     string
	link        string
	pPr         *docxXMLNode
	rPr         *docxXMLNode
	tblPr       *docxXMLNode
	tblStylePrs map[string]*docxXMLNode // tblStylePr@type → node
	outlineLvl  int                     // -1 if unset
}

type docxStyleSheet struct {
	docPPr *docxXMLNode
	docRPr *docxXMLNode
	styles map[string]*docxStyleDef
}

type docxNumLevel struct {
	numFmt  string
	lvlText string
	start   int64
	jc      string
	pPr     *docxXMLNode
	rPr     *docxXMLNode
}

type docxNumbering struct {
	abstractNums map[string][]docxNumLevel // abstractNumId → levels
	nums         map[string]string         // numId → abstractNumId
}

func loadDocxStyleSheet(zr *zip.Reader) *docxStyleSheet {
	sheet := &docxStyleSheet{styles: make(map[string]*docxStyleDef)}
	data, err := readZipFile(zr, "word/styles.xml")
	if err != nil {
		return sheet
	}
	root, err := parseDocxXML(data)
	if err != nil {
		return sheet
	}
	if dd := root.child("docDefaults"); dd != nil {
		if rpd := dd.child("rPrDefault"); rpd != nil {
			sheet.docRPr = rpd.child("rPr")
		}
		if ppd := dd.child("pPrDefault"); ppd != nil {
			sheet.docPPr = ppd.child("pPr")
		}
	}
	for i := range root.Children {
		st := &root.Children[i]
		if st.XMLName.Local != "style" {
			continue
		}
		id := st.attr("styleId")
		if id == "" {
			continue
		}
		def := &docxStyleDef{
			styleType:   st.attr("type"),
			styleID:     id,
			pPr:         st.child("pPr"),
			rPr:         st.child("rPr"),
			tblPr:       st.child("tblPr"),
			tblStylePrs: make(map[string]*docxXMLNode),
			outlineLvl:  -1,
		}
		for j := range st.Children {
			pr := &st.Children[j]
			if pr.XMLName.Local == "tblStylePr" {
				if typ := pr.attr("type"); typ != "" {
					def.tblStylePrs[typ] = pr
				}
			}
		}
		if bo := st.child("basedOn"); bo != nil {
			def.basedOn = bo.attr("val")
		}
		if lk := st.child("link"); lk != nil {
			def.link = lk.attr("val")
		}
		if name := st.child("name"); name != nil {
			def.name = name.attr("val")
		}
		if ol := def.pPr; ol != nil {
			if lvl := ol.child("outlineLvl"); lvl != nil {
				def.outlineLvl = int(parseDocxInt(lvl.attr("val")))
			}
		}
		sheet.styles[id] = def
	}
	return sheet
}

func loadDocxNumbering(zr *zip.Reader) *docxNumbering {
	num := &docxNumbering{
		abstractNums: make(map[string][]docxNumLevel),
		nums:         make(map[string]string),
	}
	data, err := readZipFile(zr, "word/numbering.xml")
	if err != nil {
		return num
	}
	root, err := parseDocxXML(data)
	if err != nil {
		return num
	}
	for i := range root.Children {
		ch := &root.Children[i]
		switch ch.XMLName.Local {
		case "abstractNum":
			id := ch.attr("abstractNumId")
			if id == "" {
				continue
			}
			var levels []docxNumLevel
			for j := range ch.Children {
				lvl := &ch.Children[j]
				if lvl.XMLName.Local != "lvl" {
					continue
				}
				ilvl := int(parseDocxInt(lvl.attr("ilvl")))
				for len(levels) <= ilvl {
					levels = append(levels, docxNumLevel{start: 1})
				}
				levels[ilvl] = docxNumLevel{
					numFmt:  lvl.childAttr("numFmt", "val"),
					lvlText: lvl.childAttr("lvlText", "val"),
					start:   parseDocxInt(lvl.childAttr("start", "val")),
					jc:      lvl.childAttr("lvlJc", "val"),
					pPr:     lvl.child("pPr"),
					rPr:     lvl.child("rPr"),
				}
				if levels[ilvl].start <= 0 {
					levels[ilvl].start = 1
				}
			}
			num.abstractNums[id] = levels
		case "num":
			id := ch.attr("numId")
			absNode := ch.child("abstractNumId")
			if id == "" || absNode == nil {
				continue
			}
			if abs := absNode.attr("val"); abs != "" {
				num.nums[id] = abs
			}
		}
	}
	return num
}

func (s *docxStyleSheet) resolveParagraph(p *docxXMLNode, theme *docxTheme, numbering *docxNumbering, listState map[string][]int64) docxResolvedPPr {
	var styleIDs []string
	if pPr := p.child("pPr"); pPr != nil {
		if ps := pPr.child("pStyle"); ps != nil {
			if id := ps.attr("val"); id != "" {
				styleIDs = append(styleIDs, id)
			}
		}
	}
	chain := s.styleChain(styleIDs)
	var props []string
	tag := "p"

	for _, def := range chain {
		if css := docxPPrCSS(def.pPr, theme); css != "" {
			props = append(props, css)
		}
		if css := docxRPrCSS(def.rPr, theme); css != "" {
			props = append(props, css)
		}
		if h := docxHeadingTag(def); h != "" {
			tag = h
		}
	}
	if pPr := p.child("pPr"); pPr != nil {
		if css := docxPPrCSS(pPr, theme); css != "" {
			props = append(props, css)
		}
		if ps := pPr.child("pStyle"); ps != nil {
			if def := s.styles[ps.attr("val")]; def != nil {
				if h := docxHeadingTag(def); h != "" {
					tag = h
				}
			}
		}
	}
	if s.docPPr != nil {
		if css := docxPPrCSS(s.docPPr, theme); css != "" {
			props = append(props, css)
		}
	}

	var marker string
	if pPr := p.child("pPr"); pPr != nil {
		if np := pPr.child("numPr"); np != nil && numbering != nil {
			marker = numbering.listMarker(np, listState)
			if npPr := numbering.levelPPr(np); npPr != nil {
				if css := docxPPrCSS(npPr, theme); css != "" {
					props = append(props, css)
				}
			}
		}
	}

	return docxResolvedPPr{
		tag:        tag,
		style:      strings.Join(dedupCSSProps(props), ";"),
		listMarker: marker,
	}
}

func (s *docxStyleSheet) resolveRun(r *docxXMLNode, paraStyleIDs []string, paraDefaultRPr *docxXMLNode, theme *docxTheme) docxResolvedRPr {
	var props []string
	// Apply in ascending priority so later (higher-priority) values win during
	// dedup: document defaults, then the style chain from base to derived, then
	// direct paragraph and run formatting.
	if s.docRPr != nil {
		if css := docxRPrCSS(s.docRPr, theme); css != "" {
			props = append(props, css)
		}
	}
	chain := s.styleChain(paraStyleIDs)
	for i := len(chain) - 1; i >= 0; i-- {
		def := chain[i]
		if css := docxRPrCSS(def.rPr, theme); css != "" {
			props = append(props, css)
		}
		if def.link != "" {
			if linked := s.styles[def.link]; linked != nil && linked.rPr != nil {
				if css := docxRPrCSS(linked.rPr, theme); css != "" {
					props = append(props, css)
				}
			}
		}
	}
	if paraDefaultRPr != nil {
		if css := docxRPrCSS(paraDefaultRPr, theme); css != "" {
			props = append(props, css)
		}
	}
	rPr := r.child("rPr")
	if rPr != nil {
		if rs := rPr.child("rStyle"); rs != nil {
			if id := rs.attr("val"); id != "" {
				if def := s.styles[id]; def != nil && def.rPr != nil {
					if css := docxRPrCSS(def.rPr, theme); css != "" {
						props = append(props, css)
					}
				}
			}
		}
		if css := docxRPrCSS(rPr, theme); css != "" {
			props = append(props, css)
		}
	}
	return docxResolvedRPr{style: strings.Join(dedupCSSProps(props), ";")}
}

func (s *docxStyleSheet) styleChain(styleIDs []string) []*docxStyleDef {
	seen := make(map[string]bool)
	var chain []*docxStyleDef
	for _, id := range styleIDs {
		for cur := id; cur != "" && !seen[cur]; {
			seen[cur] = true
			def, ok := s.styles[cur]
			if !ok {
				break
			}
			chain = append(chain, def)
			cur = def.basedOn
		}
	}
	return chain
}

func docxHeadingTag(def *docxStyleDef) string {
	if def == nil {
		return ""
	}
	switch {
	case strings.EqualFold(def.styleID, "Title"):
		return "h1"
	case strings.EqualFold(def.styleID, "Subtitle"):
		return "h2"
	case strings.HasPrefix(strings.ToLower(def.styleID), "heading"):
		n := def.styleID[len("Heading"):]
		if n >= "1" && n <= "9" {
			return "h" + n
		}
	}
	if def.outlineLvl >= 0 && def.outlineLvl <= 5 {
		return fmt.Sprintf("h%d", def.outlineLvl+1)
	}
	name := strings.ToLower(def.name)
	if strings.HasPrefix(name, "heading ") {
		switch name {
		case "heading 1":
			return "h1"
		case "heading 2":
			return "h2"
		case "heading 3":
			return "h3"
		case "heading 4":
			return "h4"
		case "heading 5":
			return "h5"
		case "heading 6":
			return "h6"
		}
	}
	return ""
}

func docxPPrCSS(pPr *docxXMLNode, theme *docxTheme) string {
	if pPr == nil {
		return ""
	}
	var props []string
	switch pPr.childAttr("jc", "val") {
	case "center":
		props = append(props, "text-align:center")
	case "right":
		props = append(props, "text-align:right")
	case "both", "distribute":
		props = append(props, "text-align:justify")
	case "left":
		props = append(props, "text-align:left")
	}
	if sp := pPr.child("spacing"); sp != nil {
		if before := parseDocxInt(sp.attr("before")); before > 0 {
			props = append(props, fmt.Sprintf("margin-top:%.2fpx", twipsToPx(before)))
		}
		if after := parseDocxInt(sp.attr("after")); after > 0 {
			props = append(props, fmt.Sprintf("margin-bottom:%.2fpx", twipsToPx(after)))
		}
		if line := parseDocxInt(sp.attr("line")); line > 0 {
			rule := sp.attr("lineRule")
			if rule == "auto" || rule == "" {
				props = append(props, fmt.Sprintf("line-height:%.2f", float64(line)/240.0))
			} else {
				props = append(props, fmt.Sprintf("line-height:%.2fpx", twipsToPx(line)))
			}
		}
	}
	if ind := pPr.child("ind"); ind != nil {
		if left := parseDocxInt(ind.attr("left")); left != 0 {
			props = append(props, fmt.Sprintf("margin-left:%.2fpx", twipsToPx(left)))
		}
		if right := parseDocxInt(ind.attr("right")); right != 0 {
			props = append(props, fmt.Sprintf("margin-right:%.2fpx", twipsToPx(right)))
		}
		if first := parseDocxInt(ind.attr("firstLine")); first != 0 {
			props = append(props, fmt.Sprintf("text-indent:%.2fpx", twipsToPx(first)))
		} else if hanging := parseDocxInt(ind.attr("hanging")); hanging != 0 {
			props = append(props, fmt.Sprintf("text-indent:%.2fpx", -twipsToPx(hanging)))
			props = append(props, fmt.Sprintf("padding-left:%.2fpx", twipsToPx(hanging)))
		}
	}
	if shd := pPr.child("shd"); shd != nil {
		if css := docxShdCSS(shd, theme); css != "" {
			props = append(props, strings.TrimSuffix(css, ";"))
		}
	}
	if pBdr := pPr.child("pBdr"); pBdr != nil {
		if css := docxPBdrCSS(pBdr); css != "" {
			props = append(props, strings.TrimSuffix(css, ";"))
		}
	}
	return strings.Join(props, ";")
}

func (s *docxStyleSheet) tblStyleConditionalCSS(styleID, cnfType string, theme *docxTheme) string {
	if s == nil || styleID == "" || cnfType == "" {
		return ""
	}
	def := s.styles[styleID]
	if def == nil {
		return ""
	}
	pr, ok := def.tblStylePrs[cnfType]
	if !ok || pr == nil {
		return ""
	}
	var props []string
	if tcPr := pr.child("tcPr"); tcPr != nil {
		if shd := tcPr.child("shd"); shd != nil {
			if css := docxShdCSS(shd, theme); css != "" {
				props = append(props, strings.TrimSuffix(css, ";"))
			}
		}
		if css := docxBordersCSS(tcPr.child("tcBorders"), theme); css != "" {
			props = append(props, strings.TrimSuffix(css, ";"))
		}
	}
	return strings.Join(props, ";")
}

func (s *docxStyleSheet) defaultFontCSS(theme *docxTheme) string {
	if s == nil || s.docRPr == nil {
		return ""
	}
	return docxRPrCSS(s.docRPr, theme)
}

func docxRPrCSS(rPr *docxXMLNode, theme *docxTheme) string {
	if rPr == nil {
		return ""
	}
	var props []string
	if docxRPrFlag(rPr, "b") {
		props = append(props, "font-weight:700")
	}
	if docxRPrFlag(rPr, "i") {
		props = append(props, "font-style:italic")
	}
	u := rPr.childAttr("u", "val")
	if u == "" {
		u = rPr.attr("u")
	}
	if u != "" && u != "none" {
		props = append(props, "text-decoration:underline")
	}
	if strike := rPr.attr("strike"); strike != "" && strike != "noStrike" {
		props = append(props, "text-decoration:line-through")
	}
	sz := parseDocxInt(rPr.childAttr("sz", "val"))
	if sz <= 0 {
		sz = parseDocxInt(rPr.attr("sz"))
	}
	if sz > 0 {
		props = append(props, fmt.Sprintf("font-size:%.2fpx", halfPtToPx(sz)))
	}
	if clr := resolveDocxColor(rPr, theme); clr != "" {
		props = append(props, "color:"+clr)
	}
	if rf := docxFontFamily(rPr.child("rFonts")); rf != "" {
		props = append(props, "font-family:'"+rf+"'")
	}
	va := rPr.childAttr("vertAlign", "val")
	if va == "" {
		va = rPr.attr("vertAlign")
	}
	switch va {
	case "superscript":
		props = append(props, "vertical-align:super", "font-size:0.75em")
	case "subscript":
		props = append(props, "vertical-align:sub", "font-size:0.75em")
	}
	if hl := rPr.child("highlight"); hl != nil {
		if val := strings.ToUpper(hl.attr("val")); val != "" && val != "NONE" {
			props = append(props, "background-color:#"+docxHighlightHex(val))
		}
	} else if shd := rPr.child("shd"); shd != nil {
		// Run-level shading is how Word fills text with a background color
		// (e.g. highlighted form placeholders) when not using w:highlight.
		if css := docxShdCSS(shd, theme); css != "" {
			props = append(props, strings.TrimSuffix(css, ";"))
		}
	}
	return strings.Join(props, ";")
}

func docxHighlightHex(name string) string {
	switch name {
	case "YELLOW":
		return "FFFF00"
	case "GREEN":
		return "00FF00"
	case "CYAN":
		return "00FFFF"
	case "MAGENTA":
		return "FF00FF"
	case "BLUE":
		return "0000FF"
	case "RED":
		return "FF0000"
	case "DARKBLUE":
		return "000080"
	case "DARKCYAN":
		return "008080"
	case "DARKGREEN":
		return "008000"
	case "DARKMAGENTA":
		return "800080"
	case "DARKRED":
		return "800000"
	case "DARKYELLOW":
		return "808000"
	case "LIGHTGRAY":
		return "D3D3D3"
	case "DARKGRAY":
		return "A9A9A9"
	case "BLACK":
		return "000000"
	case "WHITE":
		return "FFFFFF"
	default:
		return "FFFF00"
	}
}

func docxRPrFlag(rPr *docxXMLNode, local string) bool {
	if ch := rPr.child(local); ch != nil {
		v := ch.attr("val")
		return v == "" || v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "on")
	}
	v := rPr.attr(local)
	return v == "1" || strings.EqualFold(v, "true")
}

func dedupCSSProps(props []string) []string {
	seen := make(map[string]string)
	order := []string{}
	for _, p := range props {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := p
		if idx := strings.Index(p, ":"); idx > 0 {
			key = p[:idx]
		}
		if _, ok := seen[key]; !ok {
			order = append(order, key)
		}
		seen[key] = p
	}
	out := make([]string, 0, len(order))
	for _, k := range order {
		out = append(out, seen[k])
	}
	return out
}

func (n *docxNumbering) levelPPr(numPr *docxXMLNode) *docxXMLNode {
	lvl := n.level(numPr)
	if lvl == nil {
		return nil
	}
	return lvl.pPr
}

func (n *docxNumbering) level(numPr *docxXMLNode) *docxNumLevel {
	if n == nil || numPr == nil {
		return nil
	}
	numID := numPr.childAttr("numId", "val")
	ilvl := int(parseDocxInt(numPr.childAttr("ilvl", "val")))
	absID, ok := n.nums[numID]
	if !ok {
		return nil
	}
	levels, ok := n.abstractNums[absID]
	if !ok || ilvl < 0 || ilvl >= len(levels) {
		return nil
	}
	lvl := levels[ilvl]
	return &lvl
}

func (n *docxNumbering) listMarker(numPr *docxXMLNode, state map[string][]int64) string {
	lvl := n.level(numPr)
	if lvl == nil {
		return ""
	}
	numID := numPr.childAttr("numId", "val")
	ilvl := int(parseDocxInt(numPr.childAttr("ilvl", "val")))
	key := numID
	if state[key] == nil {
		state[key] = []int64{}
	}
	for len(state[key]) <= ilvl {
		state[key] = append(state[key], 0)
	}
	for i := ilvl + 1; i < len(state[key]); i++ {
		state[key][i] = 0
	}
	if state[key][ilvl] == 0 {
		state[key][ilvl] = lvl.start
	} else if lvl.numFmt != "bullet" {
		state[key][ilvl]++
	}
	val := state[key][ilvl]
	text := lvl.lvlText
	if text == "" {
		text = "%1."
	}
	text = strings.ReplaceAll(text, "%"+fmt.Sprint(ilvl+1), fmt.Sprint(val))
	text = strings.ReplaceAll(text, "%1", fmt.Sprint(val))
	if lvl.numFmt == "bullet" {
		return docxBulletGlyph(text)
	}
	return text
}

// docxBulletGlyph maps a bullet's lvlText to a renderable Unicode glyph. Bullet
// characters are usually code points from a symbol font (Symbol/Wingdings) that
// show as tofu (□) in normal fonts, so translate the common ones and fall back
// to a round bullet for any other private-use-area glyph.
func docxBulletGlyph(text string) string {
	switch text {
	case "":
		return "•"
	case "o":
		return "○"
	}
	runes := []rune(text)
	if len(runes) == 1 {
		switch runes[0] {
		case 0xF0B7, 0x00B7, 0x2022: // Symbol bullet / middle dot
			return "\u2022"
		case 0xF0A7, 0xF06E, 0x00A7: // Wingdings filled square
			return "\u25AA"
		case 0xF06F, 0xF0A8: // open circle / square
			return "\u25E6"
		case 0xF0D8, 0xF0E8: // arrowheads
			return "\u2023"
		}
		if runes[0] >= 0xF000 && runes[0] <= 0xF0FF {
			return "\u2022"
		}
	}
	return text
}
