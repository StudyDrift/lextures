package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/diagram_hotspot"
)

func sampleDiagramConfig() (json.RawMessage, diagram_hotspot.Config) {
	cfg := diagram_hotspot.Config{
		Mode:   diagram_hotspot.ModeLabel,
		Prompt: "Label the cell parts",
		Image: diagram_hotspot.ImageRef{
			URL: "/files/cell.png", Alt: "Animal cell diagram", NaturalWidth: 800, NaturalHeight: 600,
		},
		Regions: []diagram_hotspot.Region{
			{ID: "nuc", Label: "Nucleus", Description: "Dense round center containing DNA", Shape: diagram_hotspot.Shape{Kind: "circle", CX: 0.5, CY: 0.5, R: 0.12}},
			{ID: "mit", Label: "Mitochondrion", Description: "Oval organelle that produces energy", Shape: diagram_hotspot.Shape{Kind: "rect", X: 0.7, Y: 0.3, W: 0.15, H: 0.1}},
		},
		Labels: []diagram_hotspot.LabelChip{
			{ID: "l_nuc", Text: "Nucleus"},
			{ID: "l_mit", Text: "Mitochondrion"},
		},
		CorrectRegionByLabel: map[string]string{
			"l_nuc": "nuc",
			"l_mit": "mit",
		},
		FeedbackByRegion: map[string]string{
			"nuc": "Look for the dense center.",
		},
		Attempts:               2,
		LockCorrect:            true,
		ShowPerItemCorrectness: true,
		ShowRegionOutlines:     diagram_hotspot.OutlineOnFocus,
	}
	raw, _ := json.Marshal(cfg)
	return raw, cfg
}

func TestDiagramHotspotCheckLockReset(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	m := reg.Get(diagram_hotspot.ID)
	if m == nil {
		t.Fatal("diagram_hotspot missing from registry")
	}
	cfgJSON, _ := sampleDiagramConfig()
	stJSON, _ := json.Marshal(diagram_hotspot.EmptyState())

	nuc, mit := "nuc", "mit"
	wrongRaw, _ := json.Marshal(map[string]*string{
		"l_nuc": &mit,
		"l_mit": &mit,
	})
	in, _ := json.Marshal(map[string]any{"assignments": json.RawMessage(wrongRaw)})
	res, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      in,
	})
	if err != nil {
		t.Fatal(err)
	}
	score, _ := res.Result["scorePct"].(float64)
	if score != 50 {
		t.Fatalf("score %v want 50", score)
	}
	var st diagram_hotspot.State
	if err := json.Unmarshal(res.StatePatch, &st); err != nil {
		t.Fatal(err)
	}
	if len(st.LockedIDs) != 1 || st.LockedIDs[0] != "l_mit" {
		t.Fatalf("locked %+v", st.LockedIDs)
	}

	reset, err := contenttools.DispatchAction(m, "reset_attempt", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res.StatePatch,
		Input:      json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var st2 diagram_hotspot.State
	_ = json.Unmarshal(reset.StatePatch, &st2)
	if st2.Assignments["l_mit"] == nil || *st2.Assignments["l_mit"] != "mit" {
		t.Fatal("locked assignment should remain")
	}
	if st2.Assignments["l_nuc"] != nil {
		t.Fatal("unlocked should clear")
	}

	rightRaw, _ := json.Marshal(map[string]*string{
		"l_nuc": &nuc,
		"l_mit": &mit,
	})
	in2, _ := json.Marshal(map[string]any{"assignments": json.RawMessage(rightRaw), "usedListMode": true})
	res2, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  reset.StatePatch,
		Input:      in2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Result["scorePct"].(float64) != 100 {
		t.Fatalf("score %+v", res2.Result)
	}
	if res2.Status != contenttools.StatusCompleted {
		t.Fatalf("status %s", res2.Status)
	}
	var st3 diagram_hotspot.State
	_ = json.Unmarshal(res2.StatePatch, &st3)
	if !st3.UsedListMode {
		t.Fatal("expected usedListMode")
	}

	in3, _ := json.Marshal(map[string]any{"assignments": json.RawMessage(rightRaw)})
	exhausted, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res2.StatePatch,
		Input:      in3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if exhausted.Result["error"] != "max_attempts" {
		t.Fatalf("expected max_attempts got %+v", exhausted.Result)
	}
}

func TestDiagramHotspotHotspotModeAndTamper(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	m := reg.Get(diagram_hotspot.ID)
	cfg := diagram_hotspot.Config{
		Mode:   diagram_hotspot.ModeHotspot,
		Prompt: "Find the nucleus",
		Image:  diagram_hotspot.ImageRef{URL: "/img.png", Alt: "Cell", NaturalWidth: 400, NaturalHeight: 400},
		Regions: []diagram_hotspot.Region{
			{ID: "nuc", Label: "Nucleus", Description: "Center", Shape: diagram_hotspot.Shape{Kind: "circle", CX: 0.5, CY: 0.5, R: 0.2}},
			{ID: "mit", Label: "Mito", Description: "Energy", Shape: diagram_hotspot.Shape{Kind: "rect", X: 0.1, Y: 0.1, W: 0.2, H: 0.2}},
		},
		Prompts:                []diagram_hotspot.Prompt{{ID: "p1", Text: "Where is DNA stored?"}},
		CorrectRegionByPrompt:  map[string]string{"p1": "nuc"},
		Attempts:               3,
		ShowPerItemCorrectness: true,
	}
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(diagram_hotspot.EmptyState())

	ghost := "ghost"
	tamperedRaw, _ := json.Marshal(map[string]*string{
		"p1": &ghost,
	})
	inBad, _ := json.Marshal(map[string]any{"assignments": json.RawMessage(tamperedRaw)})
	bad, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      inBad,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bad.Result["error"] != "invalid_placement" {
		t.Fatalf("got %+v", bad.Result)
	}

	nuc := "nuc"
	okRaw, _ := json.Marshal(map[string]*string{"p1": &nuc})
	inOk, _ := json.Marshal(map[string]any{"assignments": json.RawMessage(okRaw)})
	good, err := contenttools.DispatchAction(m, "check", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      inOk,
	})
	if err != nil {
		t.Fatal(err)
	}
	if good.Result["scorePct"].(float64) != 100 {
		t.Fatalf("%+v", good.Result)
	}
}
