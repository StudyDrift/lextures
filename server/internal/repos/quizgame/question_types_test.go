package quizgame

import (
	"encoding/json"
	"testing"
)

func TestNormalizeCreateInput_DefaultsAndBounds(t *testing.T) {
	in := CreateQuestionInput{Prompt: "  Hello <b>world</b>  "}
	if err := NormalizeCreateInput(&in); err != nil {
		t.Fatal(err)
	}
	if in.QuestionType != QTypeMCSingle {
		t.Fatalf("type=%s", in.QuestionType)
	}
	if in.TimeLimitSeconds != defTimeLimit {
		t.Fatalf("timer=%d", in.TimeLimitSeconds)
	}
	if in.Prompt != "Hello world" {
		t.Fatalf("prompt=%q", in.Prompt)
	}
	if len(in.Options) != 4 {
		t.Fatalf("options=%d", len(in.Options))
	}

	in.TimeLimitSeconds = 3
	if err := NormalizeCreateInput(&in); err == nil {
		t.Fatal("expected timer bound error")
	}
}

func TestValidateQuestionReady_MCAndPoll(t *testing.T) {
	opts, _ := json.Marshal([]Option{
		{ID: "a", Text: "A", IsCorrect: false},
		{ID: "b", Text: "B", IsCorrect: false},
		{ID: "c", Text: "C", IsCorrect: false},
		{ID: "d", Text: "D", IsCorrect: false},
	})
	q := Question{
		ID: "q1", QuestionType: QTypeMCSingle, Prompt: "Pick one",
		Options: opts, TimeLimitSeconds: 15, PointsStyle: PointsStandard,
	}
	issues := ValidateQuestionReady(q)
	if len(issues) == 0 {
		t.Fatal("expected missing_correct")
	}
	found := false
	for _, i := range issues {
		if i.Code == "missing_correct" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues=%+v", issues)
	}

	optsOK, _ := json.Marshal([]Option{
		{ID: "a", Text: "A", IsCorrect: true},
		{ID: "b", Text: "B", IsCorrect: false},
		{ID: "c", Text: "C", IsCorrect: false},
		{ID: "d", Text: "D", IsCorrect: false},
	})
	q.Options = optsOK
	if issues = ValidateQuestionReady(q); len(issues) != 0 {
		t.Fatalf("mc should be ready: %+v", issues)
	}

	pollOpts, _ := json.Marshal([]Option{
		{ID: "a", Text: "Yes", IsCorrect: false},
		{ID: "b", Text: "No", IsCorrect: false},
	})
	poll := Question{
		ID: "p1", QuestionType: QTypePoll, Prompt: "Opinion?",
		Options: pollOpts, TimeLimitSeconds: 20, PointsStyle: PointsNone,
	}
	if issues = ValidateQuestionReady(poll); len(issues) != 0 {
		t.Fatalf("poll with 0 correct should be ready: %+v", issues)
	}
}

func TestValidateQuestionReady_MediaAlt(t *testing.T) {
	ref := "obj/key"
	opts, _ := json.Marshal([]Option{
		{ID: "a", Text: "A", IsCorrect: true},
		{ID: "b", Text: "B", IsCorrect: false},
	})
	q := Question{
		ID: "q1", QuestionType: QTypeMCSingle, Prompt: "Img?",
		PromptMediaRef: &ref, Options: opts, TimeLimitSeconds: 20,
	}
	issues := ValidateQuestionReady(q)
	found := false
	for _, i := range issues {
		if i.Code == "missing_media_alt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing_media_alt, got %+v", issues)
	}
}

func TestMapBankQuestionType(t *testing.T) {
	got, ok := MapBankQuestionType("short_answer")
	if !ok || got != QTypeTypeAnswer {
		t.Fatalf("got %s ok=%v", got, ok)
	}
	if _, ok := MapBankQuestionType("hotspot"); ok {
		t.Fatal("hotspot should not map")
	}
}

func TestSanitizeStripsHTML(t *testing.T) {
	got := sanitizePlainText("<script>alert(1)</script>Hi", 100)
	if got != "alert(1)Hi" && got != "Hi" {
		// tags stripped; script contents may remain as text — that's fine for projector safety
		if len(got) == 0 {
			t.Fatal("empty")
		}
	}
	if got != "alert(1)Hi" {
		t.Fatalf("got %q", got)
	}
}
