package officepreview

import "strings"

// pptxAnimEndState captures slide appearance after all click-sequence animations finish.
type pptxAnimEndState struct {
	hiddenIDs map[string]bool
	offsets   map[string]pptxMotionOffset
}

type pptxMotionOffset struct {
	dx, dy int64 // EMU delta applied to shape position
}

func parsePptxAnimEndState(slideRoot *pptxXMLNode) pptxAnimEndState {
	state := pptxAnimEndState{
		hiddenIDs: make(map[string]bool),
		offsets:   make(map[string]pptxMotionOffset),
	}
	timing := slideRoot.child("timing")
	if timing == nil {
		return state
	}
	visibility := make(map[string]bool)
	walkPptxTimingNodes(timing, func(n *pptxXMLNode) {
		switch n.XMLName.Local {
		case "animEffect":
			spid := pptxAnimTargetSpID(n)
			if spid == "" {
				return
			}
			switch n.attr("transition") {
			case "out":
				visibility[spid] = false
			case "in":
				visibility[spid] = true
			}
		case "set":
			spid := pptxAnimTargetSpID(n)
			if spid == "" {
				return
			}
			if pptxSetAnimVisibility(n, false) {
				visibility[spid] = false
			} else if pptxSetAnimVisibility(n, true) {
				visibility[spid] = true
			}
		case "animMotion":
			spid := pptxAnimTargetSpID(n)
			if spid == "" {
				return
			}
			if off := pptxAnimMotionOffset(n); off.dx != 0 || off.dy != 0 {
				prev := state.offsets[spid]
				state.offsets[spid] = pptxMotionOffset{
					dx: prev.dx + off.dx,
					dy: prev.dy + off.dy,
				}
			}
		}
	})
	for spid, visible := range visibility {
		if !visible {
			state.hiddenIDs[spid] = true
		}
	}
	return state
}

func walkPptxTimingNodes(node *pptxXMLNode, fn func(*pptxXMLNode)) {
	if node == nil {
		return
	}
	fn(node)
	for i := range node.Children {
		walkPptxTimingNodes(&node.Children[i], fn)
	}
}

func pptxAnimTargetSpID(node *pptxXMLNode) string {
	spTgt := node.findDeep("spTgt")
	if spTgt == nil {
		return ""
	}
	return spTgt.attr("spid")
}

func pptxSetAnimVisibility(node *pptxXMLNode, wantVisible bool) bool {
	cBhvr := node.child("cBhvr")
	if cBhvr == nil {
		return false
	}
	attrName := cBhvr.findDeep("attrName")
	if attrName == nil || attrName.Content == "" {
		if lst := cBhvr.child("attrNameLst"); lst != nil {
			for i := range lst.Children {
				if lst.Children[i].XMLName.Local == "attrName" {
					attrName = &lst.Children[i]
					break
				}
			}
		}
	}
	name := ""
	if attrName != nil {
		name = strings.TrimSpace(attrName.Content)
		if name == "" {
			name = attrName.attr("val")
		}
	}
	if name != "style.visibility" && name != "visibility" {
		return false
	}
	to := node.child("to")
	if to == nil {
		return false
	}
	if strVal := to.child("strVal"); strVal != nil {
		val := strings.ToLower(strings.TrimSpace(strVal.attr("val")))
		if wantVisible {
			return val == "visible"
		}
		return val == "hidden" || val == "none"
	}
	return false
}

func pptxAnimMotionOffset(node *pptxXMLNode) pptxMotionOffset {
	var off pptxMotionOffset
	if by := node.child("by"); by != nil {
		if pos := by.child("pos"); pos != nil {
			off.dx = parseEMU(pos.attr("x"))
			off.dy = parseEMU(pos.attr("y"))
			return off
		}
	}
	if to := node.child("to"); to != nil {
		if pos := to.child("pos"); pos != nil {
			off.dx = parseEMU(pos.attr("x"))
			off.dy = parseEMU(pos.attr("y"))
		}
	}
	return off
}

func applyPptxAnimEndState(layers []pptxVisualLayer, state pptxAnimEndState) []pptxVisualLayer {
	if len(state.hiddenIDs) == 0 && len(state.offsets) == 0 {
		return layers
	}
	out := make([]pptxVisualLayer, 0, len(layers))
	for _, layer := range layers {
		if layer.spID != "" && state.hiddenIDs[layer.spID] {
			continue
		}
		if layer.spID != "" {
			if off, ok := state.offsets[layer.spID]; ok {
				layer.left += off.dx
				layer.top += off.dy
			}
		}
		out = append(out, layer)
	}
	return out
}

func shapeSpID(node *pptxXMLNode) string {
	if cNvPr := node.findDeep("cNvPr"); cNvPr != nil {
		return cNvPr.attr("id")
	}
	return ""
}
