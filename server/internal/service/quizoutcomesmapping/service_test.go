package quizoutcomesmapping

import (
	"strings"
	"testing"
)

func TestParseSuggestionsJSON_Success(t *testing.T) {
	t.Parallel()
	raw := "```json\n{\"suggestions\":[" +
		`{"targetKind":"quiz","quizQuestionId":"","outcomeId":"o1","measurementLevel":"summative","intensityLevel":"high","rationale":"Covers the unit"},` +
		`{"targetKind":"quiz_question","quizQuestionId":"q1","outcomeId":"o1","measurementLevel":"formative","intensityLevel":"medium","rationale":"Direct prompt match"},` +
		`{"targetKind":"quiz_question","quizQuestionId":"missing","outcomeId":"o1","measurementLevel":"formative","intensityLevel":"medium","rationale":"drop"},` +
		`{"targetKind":"quiz","quizQuestionId":"","outcomeId":"nope","measurementLevel":"formative","intensityLevel":"medium","rationale":"drop"}` +
		"]}\n```"
	got, err := ParseSuggestionsJSON(raw, map[string]struct{}{"o1": {}}, map[string]struct{}{"q1": {}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2: %#v", len(got), got)
	}
	if got[0].TargetKind != "quiz" || got[0].OutcomeID != "o1" || got[0].MeasurementLevel != "summative" {
		t.Fatalf("got %#v", got[0])
	}
	if got[1].TargetKind != "quiz_question" || got[1].QuizQuestionID != "q1" {
		t.Fatalf("got %#v", got[1])
	}
}

func TestParseSuggestionsJSON_DefaultsLevels(t *testing.T) {
	t.Parallel()
	raw := `{"suggestions":[{"targetKind":"quiz","outcomeId":"o1","measurementLevel":"nope","intensityLevel":"nope"}]}`
	got, err := ParseSuggestionsJSON(raw, map[string]struct{}{"o1": {}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].MeasurementLevel != defaultMeasurement || got[0].IntensityLevel != defaultIntensity {
		t.Fatalf("got %#v", got[0])
	}
}

func TestParseSuggestionsJSON_Invalid(t *testing.T) {
	t.Parallel()
	if _, err := ParseSuggestionsJSON("not json", nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseSuggestionsJSON_CapsCount(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	b.WriteString(`{"suggestions":[`)
	for i := 0; i < MaxSuggestions+5; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"targetKind":"quiz","outcomeId":"o`)
		b.WriteString(strings.Repeat("x", i+1))
		b.WriteString(`","measurementLevel":"formative","intensityLevel":"medium"}`)
	}
	b.WriteString(`]}`)
	valid := make(map[string]struct{}, MaxSuggestions+5)
	for i := 0; i < MaxSuggestions+5; i++ {
		valid["o"+strings.Repeat("x", i+1)] = struct{}{}
	}
	got, err := ParseSuggestionsJSON(b.String(), valid, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxSuggestions {
		t.Fatalf("len=%d want %d", len(got), MaxSuggestions)
	}
}
