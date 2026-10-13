package parentprogress

import (
	"strings"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/models/gradecomment"
	repo "github.com/lextures/lextures/server/internal/repos/parentprogress"
)

func TestBuildSectionActivityAndEmptyWeek(t *testing.T) {
	start := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	raw, err := gradecomment.MarshalList([]gradecomment.Comment{{
		Body:      "Nice work on the essay.",
		CreatedAt: "2026-10-08T15:00:00Z",
	}})
	if err != nil {
		t.Fatal(err)
	}
	section := BuildSection("Ada",
		[]repo.Item{{Course: "Writing", Title: "Essay"}},
		[]repo.Item{{Course: "Math", Title: "Quiz 2", When: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)}},
		[]repo.GradeChange{{Course: "Math", Title: "Quiz 1", Previous: "7", Next: "9"}},
		[]repo.Comment{{Course: "Writing", Title: "Essay", Body: string(raw)}},
		90*60,
		start,
	)
	if !section.HasActivity() {
		t.Fatal("expected activity")
	}
	if section.TimeSpent != "1 h 30 min" {
		t.Fatalf("time %q", section.TimeSpent)
	}
	if len(section.Comments) != 1 || !strings.Contains(section.Comments[0], "Nice work") {
		t.Fatalf("comments %#v", section.Comments)
	}
	idle := BuildSection("Bea", nil, nil, nil, nil, 0, start)
	if idle.HasActivity() || AnyActivity([]Section{idle}) {
		t.Fatal("idle week should have no activity")
	}
	note := Compose("Pat", nil, true)
	if !note.NoteOnly || !strings.Contains(note.BodyText, "No learning activity") {
		t.Fatalf("note %#v", note)
	}
	full := Compose("Pat", []Section{section}, false)
	for _, want := range []string{"Ada", "Essay", "Quiz 2", "7 → 9", "1 h 30 min", "Nice work"} {
		if !strings.Contains(full.BodyText, want) {
			t.Fatalf("body missing %q:\n%s", want, full.BodyText)
		}
	}
}

func TestOldCommentIsNotNew(t *testing.T) {
	start := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	raw, err := gradecomment.MarshalList([]gradecomment.Comment{{
		Body:      "Last month",
		CreatedAt: "2026-09-01T00:00:00Z",
	}})
	if err != nil {
		t.Fatal(err)
	}
	section := BuildSection("Ada", nil, nil, nil, []repo.Comment{{
		Course: "Writing", Title: "Essay", Body: string(raw),
	}}, 0, start)
	if len(section.Comments) != 0 || section.HasActivity() {
		t.Fatalf("old comment counted: %#v", section.Comments)
	}
}
