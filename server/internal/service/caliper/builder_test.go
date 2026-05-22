package caliper

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildEvent_gradeEvent(t *testing.T) {
	score := 92.0
	ev := BuildEvent(BuildInput{
		EventID:    uuid.New(),
		EventType:  "GradeEvent",
		Action:     ActionGraded,
		ActorIRI:   "https://lextures.test/users/inst",
		ObjectIRI:  "https://lextures.test/assignments/a1",
		ObjectName: "Essay 1",
		CourseIRI:  "https://lextures.test/courses/demo",
		Score:      &score,
	})
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), ActionGraded) {
		t.Fatalf("missing action: %s", raw)
	}
	if ev.Context != "http://purl.imsglobal.org/ctx/caliper/v1p2" {
		t.Fatalf("unexpected context: %s", ev.Context)
	}
}
