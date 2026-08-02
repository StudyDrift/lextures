package inline_questions

import (
	"encoding/json"
	"testing"
)

func TestParseConfig_QuestionsAtATime(t *testing.T) {
	cfg := ParseConfig(json.RawMessage(`{"questionsAtATime":"all"}`))
	if QuestionsAtATimeCount(cfg) != 0 {
		t.Fatalf("all => 0, got %d", QuestionsAtATimeCount(cfg))
	}
	cfg = ParseConfig(json.RawMessage(`{"questionsAtATime":1}`))
	if QuestionsAtATimeCount(cfg) != 1 {
		t.Fatalf("1 => 1, got %d", QuestionsAtATimeCount(cfg))
	}
	cfg = ParseConfig(json.RawMessage(`{"questionsAtATime":2}`))
	if QuestionsAtATimeCount(cfg) != 2 {
		t.Fatalf("2 => 2, got %d", QuestionsAtATimeCount(cfg))
	}
	cfg = ParseConfig(json.RawMessage(`{}`))
	if QuestionsAtATimeCount(cfg) != 0 {
		t.Fatalf("default => 0, got %d", QuestionsAtATimeCount(cfg))
	}
	// Invalid values fall back to default all
	cfg = ParseConfig(json.RawMessage(`{"questionsAtATime":9}`))
	if QuestionsAtATimeCount(cfg) != 0 {
		t.Fatalf("invalid 9 => 0, got %d", QuestionsAtATimeCount(cfg))
	}
}
