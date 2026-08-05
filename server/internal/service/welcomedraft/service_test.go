package welcomedraft

import "testing"

func TestParseDraftJSON(t *testing.T) {
	d, err := ParseDraftJSON(`{"subject":"Welcome to BIO 101","body":"Hello everyone…"}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Subject != "Welcome to BIO 101" || d.Body == "" {
		t.Fatalf("unexpected draft: %+v", d)
	}
}

func TestLooksLikeLearnerField(t *testing.T) {
	if !looksLikeLearnerField("Student name list") {
		t.Fatal("expected reject")
	}
	if looksLikeLearnerField("Introduction to Biology") {
		t.Fatal("should allow title")
	}
}
