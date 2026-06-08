package officepreview

import "testing"

func TestParsePptxAnimEndStateExitAndMotion(t *testing.T) {
	slideXML := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:timing>
    <p:tnLst>
      <p:par>
        <p:cTn>
          <p:childTnLst>
            <p:animEffect transition="out">
              <p:cBhvr>
                <p:tgtEl><p:spTgt spid="4"/></p:tgtEl>
              </p:cBhvr>
            </p:animEffect>
            <p:animMotion>
              <p:cBhvr>
                <p:tgtEl><p:spTgt spid="2"/></p:tgtEl>
              </p:cBhvr>
              <p:by><p:pos x="100000" y="200000"/></p:by>
            </p:animMotion>
            <p:set>
              <p:cBhvr>
                <p:tgtEl><p:spTgt spid="3"/></p:tgtEl>
                <p:attrNameLst><p:attrName>style.visibility</p:attrName></p:attrNameLst>
              </p:cBhvr>
              <p:to><p:strVal val="hidden"/></p:to>
            </p:set>
          </p:childTnLst>
        </p:cTn>
      </p:par>
    </p:tnLst>
  </p:timing>
</p:sld>`)
	root, err := parsePptxXML(slideXML)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	state := parsePptxAnimEndState(root)
	if !state.hiddenIDs["4"] {
		t.Fatal("expected spid 4 hidden after exit animation")
	}
	if !state.hiddenIDs["3"] {
		t.Fatal("expected spid 3 hidden after visibility set")
	}
	off := state.offsets["2"]
	if off.dx != 100000 || off.dy != 200000 {
		t.Fatalf("motion offset = %+v, want dx=100000 dy=200000", off)
	}
}

func TestApplyPptxAnimEndState(t *testing.T) {
	layers := []pptxVisualLayer{
		{spID: "2", left: 10, top: 20, cx: 100, cy: 100, kind: "text"},
		{spID: "3", left: 0, top: 0, cx: 50, cy: 50, kind: "text"},
		{spID: "5", left: 0, top: 0, cx: 50, cy: 50, kind: "text"},
	}
	state := pptxAnimEndState{
		hiddenIDs: map[string]bool{"3": true},
		offsets:   map[string]pptxMotionOffset{"2": {dx: 1000, dy: 2000}},
	}
	out := applyPptxAnimEndState(layers, state)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].left != 1010 || out[0].top != 2020 {
		t.Fatalf("offset layer = %+v", out[0])
	}
}
