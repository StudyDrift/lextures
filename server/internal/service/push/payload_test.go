package push

import (
	"strings"
	"testing"
)

func TestBuildNativePayloadJSON(t *testing.T) {
	raw := BuildNativePayloadJSON("Grade posted", "Your essay was graded.", "/courses/cs101/grades", "grade_posted")
	if !strings.Contains(string(raw), `"action_url":"/courses/cs101/grades"`) {
		t.Fatalf("expected action_url in payload: %s", raw)
	}
}

func TestParseAPNSP8KeyRejectsEmpty(t *testing.T) {
	if _, err := parseAPNSP8Key(""); err == nil {
		t.Fatal("expected error for empty key")
	}
}
