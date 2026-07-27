package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/predict_reveal"
)

func samplePredictRevealConfig() (json.RawMessage, predict_reveal.Config) {
	cfg := predict_reveal.Config{
		Question:           "What will happen when we heat the balloon?",
		Mode:               predict_reveal.ModeChoice,
		ConfidenceScale:    predict_reveal.ScaleThree,
		ConfidenceRequired: true,
		ShowPeerResults:    true,
		Outcomes: []predict_reveal.Outcome{
			{ID: "expand", Text: "It expands", Correct: true},
			{ID: "shrink", Text: "It shrinks", Correct: false},
		},
		Reveal: predict_reveal.Reveal{Markdown: "The balloon expands as air molecules move faster."},
	}
	raw, _ := json.Marshal(cfg)
	return raw, cfg
}

func TestPredictRevealCommitGateAndIdempotency(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(predict_reveal.ID)
	if m == nil {
		t.Fatal("missing manifest")
	}

	cfgJSON, cfg := samplePredictRevealConfig()
	_ = cfg
	stJSON, _ := json.Marshal(predict_reveal.EmptyState())

	// Redaction: student payload must not include reveal.
	redacted, err := contenttools.RedactSensitiveConfig(m.ConfigSchema, cfgJSON)
	if err != nil {
		t.Fatal(err)
	}
	var red map[string]any
	_ = json.Unmarshal(redacted, &red)
	if _, ok := red["reveal"]; ok {
		t.Fatalf("reveal leaked in redacted config: %s", redacted)
	}
	outcomes, _ := red["outcomes"].([]any)
	if len(outcomes) == 0 {
		t.Fatal("outcomes should remain")
	}
	first, _ := outcomes[0].(map[string]any)
	if _, ok := first["correct"]; ok {
		t.Fatal("correct flag leaked")
	}

	// Commit without confidence → soft error.
	in, _ := json.Marshal(map[string]any{
		"prediction": map[string]any{"outcomeId": "expand"},
	})
	res, err := contenttools.DispatchAction(m, "commit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      in,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "confidence_required" {
		t.Fatalf("want confidence_required, got %#v", res.Result)
	}

	conf := 3.0
	in2, _ := json.Marshal(map[string]any{
		"prediction": map[string]any{"outcomeId": "shrink"},
		"confidence": conf,
	})
	res2, err := contenttools.DispatchAction(m, "commit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      in2,
	})
	if err != nil {
		t.Fatal(err)
	}
	reveal, ok := res2.Result["reveal"].(map[string]any)
	if !ok || reveal["markdown"] == nil {
		t.Fatalf("reveal missing: %#v", res2.Result)
	}
	var st predict_reveal.State
	if err := json.Unmarshal(res2.StatePatch, &st); err != nil {
		t.Fatal(err)
	}
	if st.CommittedAt == "" || st.RevealedAt == "" {
		t.Fatalf("timestamps missing: %#v", st)
	}
	if st.ConfidenceBucket != "certain" {
		t.Fatalf("bucket: %q", st.ConfidenceBucket)
	}
	if st.Correct == nil || *st.Correct {
		t.Fatalf("want correct=false for shrink, got %#v", st.Correct)
	}
	peer, _ := res2.Result["peerResults"].(predict_reveal.PeerResults)
	// Without pool, peerResults is a pointer in map — check via JSON.
	if raw, err := json.Marshal(res2.Result["peerResults"]); err == nil {
		var pr predict_reveal.PeerResults
		_ = json.Unmarshal(raw, &pr)
		if !pr.Suppressed {
			t.Fatalf("expected suppressed peer results without pool/small n: %#v", pr)
		}
		_ = peer
	}

	// Attempt to change prediction after commit.
	in3, _ := json.Marshal(map[string]any{
		"prediction": map[string]any{"outcomeId": "expand"},
		"confidence": 2.0,
	})
	res3, err := contenttools.DispatchAction(m, "commit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res2.StatePatch,
		Input:      in3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res3.Result["error"] != "already_committed" {
		t.Fatalf("want already_committed, got %#v", res3.Result)
	}

	// Idempotent empty recommit returns reveal.
	res4, err := contenttools.DispatchAction(m, "commit", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res2.StatePatch,
		Input:      json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res4.Result["reveal"]; !ok {
		t.Fatalf("idempotent reveal missing: %#v", res4.Result)
	}

	// Reflect.
	inR, _ := json.Marshal(map[string]any{"text": "I thought heat made it shrink."})
	resR, err := contenttools.DispatchAction(m, "reflect", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res2.StatePatch,
		Input:      inR,
	})
	if err != nil {
		t.Fatal(err)
	}
	var stR predict_reveal.State
	_ = json.Unmarshal(resR.StatePatch, &stR)
	if stR.Reflection == "" {
		t.Fatal("reflection not stored")
	}

	// Open mode + filter.
	openCfg := cfg
	openCfg.Mode = predict_reveal.ModeOpen
	openCfg.Outcomes = nil
	openJSON, _ := json.Marshal(openCfg)
	bad, _ := json.Marshal(map[string]any{
		"prediction": map[string]any{"text": "kill myself now"},
		"confidence": 2.0,
	})
	resF, err := contenttools.DispatchAction(m, "commit", contenttools.ActionContext{
		ConfigJSON: openJSON,
		StateJSON:  stJSON,
		Input:      bad,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resF.Result["error"] != "filtered" {
		t.Fatalf("want filtered, got %#v", resF.Result)
	}
}

func TestPredictRevealRegistryAndProjector(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	if reg.Get(predict_reveal.ID) == nil {
		t.Fatal("predict_reveal not registered")
	}
	blocked, msg := contenttools.GuardPredictRevealStatePut(
		predict_reveal.ID,
		json.RawMessage(`{"v":1,"committedAt":"2026-01-01T00:00:00Z"}`),
	)
	if !blocked || msg == "" {
		t.Fatal("guard should block")
	}
}
