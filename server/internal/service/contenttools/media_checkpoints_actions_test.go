package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/media_checkpoints"
)

func sampleMediaConfig(practiceOnly bool, preventSkip bool) media_checkpoints.Config {
	req := true
	attempts := 2
	show := true
	return media_checkpoints.Config{
		Media: media_checkpoints.MediaRef{
			Source:      media_checkpoints.MediaSourceCourseFile,
			FileID:      "file-1",
			Kind:        media_checkpoints.MediaKindVideo,
			DurationSec: 120,
			URL:         "https://example.com/clip.webm",
		},
		TranscriptSource:          media_checkpoints.TranscriptInline,
		TranscriptMarkdown:        "0:00 Intro\n0:30 Concept\n1:30 Wrap",
		PreventSkipPastUnanswered: preventSkip,
		PracticeOnly:              practiceOnly,
		Checkpoints: []media_checkpoints.Checkpoint{
			{
				ID: "c1", AtSec: 15, Required: &req, Attempts: &attempts, ShowFeedback: &show,
				Question: media_checkpoints.Question{
					Type:   media_checkpoints.TypeSingle,
					Prompt: "Main idea?",
					Options: []media_checkpoints.Option{
						{ID: "a", Text: "Wrong", Correct: false, Feedback: "Not quite"},
						{ID: "b", Text: "Right", Correct: true, Feedback: "Yes"},
					},
				},
			},
			{
				ID: "c2", AtSec: 45, Required: &req, Attempts: &attempts, ShowFeedback: &show,
				Question: media_checkpoints.Question{
					Type:            media_checkpoints.TypeShortText,
					Prompt:          "Keyword?",
					AcceptedAnswers: []string{"photosynthesis"},
				},
			},
		},
	}
}

func TestMediaCheckpointsAnswerFlow(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(media_checkpoints.ID)
	if m == nil {
		t.Fatal("missing manifest")
	}

	cfg := sampleMediaConfig(true, false)
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(media_checkpoints.EmptyState())

	in, _ := json.Marshal(map[string]any{"checkpointId": "c1", "value": "a"})
	res, err := contenttools.DispatchAction(m, "answer_checkpoint", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      in,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["correct"] != false {
		t.Fatalf("want incorrect: %#v", res.Result)
	}
	if res.Result["feedback"] != "Not quite" {
		t.Fatalf("feedback: %#v", res.Result["feedback"])
	}
	if res.ScoreRaw != nil {
		t.Fatal("practiceOnly must not expose score to framework")
	}

	in2, _ := json.Marshal(map[string]any{"checkpointId": "c1", "value": "b"})
	res2, err := contenttools.DispatchAction(m, "answer_checkpoint", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res.StatePatch,
		Input:      in2,
		Status:     contenttools.StatusInProgress,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.Result["correct"] != true || res2.Result["done"] != true {
		t.Fatalf("want correct+done: %#v", res2.Result)
	}

	// Sensitive keys must not appear in student-facing redacted config.
	redacted, err := contenttools.RedactSensitiveConfig(m.ConfigSchema, cfgJSON)
	if err != nil {
		t.Fatal(err)
	}
	var red map[string]any
	_ = json.Unmarshal(redacted, &red)
	cps, _ := red["checkpoints"].([]any)
	if len(cps) == 0 {
		t.Fatal("checkpoints missing after redaction")
	}
	cp0, _ := cps[0].(map[string]any)
	q, _ := cp0["question"].(map[string]any)
	opts, _ := q["options"].([]any)
	opt0, _ := opts[0].(map[string]any)
	if _, ok := opt0["correct"]; ok {
		t.Fatal("correct must be redacted")
	}
	if _, ok := opt0["feedback"]; ok {
		t.Fatal("feedback must be redacted")
	}
}

func TestMediaCheckpointsProgressDoesNotAffectScore(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(media_checkpoints.ID)
	cfg := sampleMediaConfig(false, false)
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(media_checkpoints.EmptyState())

	in, _ := json.Marshal(map[string]any{
		"watchedSegments": [][2]float64{{0, 180}, {300, 360}},
		"furthestSec":     360,
	})
	res, err := contenttools.DispatchAction(m, "record_progress", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  stJSON,
		Input:      in,
	})
	if err != nil {
		t.Fatal(err)
	}
	var st media_checkpoints.State
	_ = json.Unmarshal(res.StatePatch, &st)
	if st.FurthestSec != 360 {
		t.Fatalf("furthest=%v", st.FurthestSec)
	}
	if st.ScoreRaw == nil || *st.ScoreRaw != 0 {
		t.Fatalf("forged watch data must not create a score: %#v", st.ScoreRaw)
	}

	// Answer one correctly — score becomes 1/2.
	ansIn, _ := json.Marshal(map[string]any{"checkpointId": "c1", "value": "b", "transcriptOnly": true})
	ans, err := contenttools.DispatchAction(m, "answer_checkpoint", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  res.StatePatch,
		Input:      ansIn,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ans.ScoreRaw == nil || *ans.ScoreRaw != 1 || ans.ScoreMax == nil || *ans.ScoreMax != 2 {
		t.Fatalf("score=%v/%v", ans.ScoreRaw, ans.ScoreMax)
	}
	var st2 media_checkpoints.State
	_ = json.Unmarshal(ans.StatePatch, &st2)
	if !st2.UsedTranscriptOnly {
		t.Fatal("expected transcript-only flag")
	}
}

func TestMediaCheckpointsMaxAttempts(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(media_checkpoints.ID)
	cfg := sampleMediaConfig(true, false)
	cfgJSON, _ := json.Marshal(cfg)
	stJSON, _ := json.Marshal(media_checkpoints.EmptyState())

	var patch json.RawMessage = stJSON
	for i := 0; i < 2; i++ {
		in, _ := json.Marshal(map[string]any{"checkpointId": "c1", "value": "a"})
		res, err := contenttools.DispatchAction(m, "answer_checkpoint", contenttools.ActionContext{
			ConfigJSON: cfgJSON,
			StateJSON:  patch,
			Input:      in,
		})
		if err != nil {
			t.Fatal(err)
		}
		patch = res.StatePatch
	}
	in, _ := json.Marshal(map[string]any{"checkpointId": "c1", "value": "b"})
	res, err := contenttools.DispatchAction(m, "answer_checkpoint", contenttools.ActionContext{
		ConfigJSON: cfgJSON,
		StateJSON:  patch,
		Input:      in,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "max_attempts" {
		t.Fatalf("want max_attempts: %#v", res.Result)
	}
}
