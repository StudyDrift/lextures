package xapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestActorHash_anonymize(t *testing.T) {
	h := ActorHash("alice@example.com", true)
	if h == "alice@example.com" {
		t.Fatal("expected hash, got plaintext email")
	}
	if len(h) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(h))
	}
	mbox := ActorMbox("alice@example.com", true)
	if strings.Contains(mbox, "alice@example.com") {
		t.Fatalf("mbox leaked email: %s", mbox)
	}
}

func TestBuildStatement_passedQuiz(t *testing.T) {
	score := 0.85
	ok := true
	stmt := BuildStatement(BuildInput{
		StatementID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		ActorEmail:  "student@test.invalid",
		VerbID:      VerbPassed,
		ObjectID:    "https://lextures.test/courses/demo/quizzes/q1",
		ObjectTitle: "Quiz 1",
		CourseIRI:   "https://lextures.test/courses/demo",
		Score:       &score,
		Success:     &ok,
	})
	raw, err := json.Marshal(stmt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), VerbPassed) {
		t.Fatalf("missing verb: %s", raw)
	}
	if stmt.Result == nil || stmt.Result.Score == nil || *stmt.Result.Score.Scaled != 0.85 {
		t.Fatalf("unexpected result: %+v", stmt.Result)
	}
}
