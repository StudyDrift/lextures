package h5p

import (
	"encoding/json"
	"testing"
)

func TestCompletionStatus_completed(t *testing.T) {
	stmt := Statement{Verb: VerbRef{ID: "http://adlnet.gov/expapi/verbs/completed"}}
	status, _, _ := CompletionStatus(stmt)
	if status != "completed" {
		t.Fatalf("got %q want completed", status)
	}
}

func TestCompletionStatus_passedWithScore(t *testing.T) {
	success := true
	stmt := Statement{
		Verb: VerbRef{ID: "http://adlnet.gov/expapi/verbs/completed"},
		Result: &StatementResult{
			Success: &success,
			Score:   &ScoreResult{Raw: 8, Max: 10},
		},
	}
	status, raw, max := CompletionStatus(stmt)
	if status != "passed" {
		t.Fatalf("got %q want passed", status)
	}
	if raw == nil || *raw != 8 || max == nil || *max != 10 {
		t.Fatalf("unexpected scores %v %v", raw, max)
	}
}

func TestCompletionStatus_inProgress(t *testing.T) {
	stmt := Statement{Verb: VerbRef{ID: "http://adlnet.gov/expapi/verbs/attempted"}}
	status, _, _ := CompletionStatus(stmt)
	if status != "in_progress" {
		t.Fatalf("got %q want in_progress", status)
	}
}

func TestParseStatement(t *testing.T) {
	raw := json.RawMessage(`{"verb":{"id":"http://adlnet.gov/expapi/verbs/completed"}}`)
	stmt, err := ParseStatement(raw)
	if err != nil {
		t.Fatal(err)
	}
	if stmt.Verb.ID == "" {
		t.Fatal("expected verb id")
	}
}

func TestDisplayLabel(t *testing.T) {
	if DisplayLabel("completed") != "Completed" {
		t.Fatal("label mismatch")
	}
}
