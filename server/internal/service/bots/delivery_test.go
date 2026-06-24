package bots

import (
	"testing"

	"github.com/lextures/lextures/server/internal/webhooks"
)

func TestParseEnvelope_AssignmentCreated(t *testing.T) {
	payload := []byte(`{
		"event_type": "assignment.created",
		"data": {
			"courseId": "550e8400-e29b-41d4-a716-446655440000",
			"courseCode": "ENG-101",
			"title": "Essay 1",
			"dueAt": "2026-06-01T17:00:00Z",
			"url": "https://app.example/courses/550e8400-e29b-41d4-a716-446655440000"
		}
	}`)
	ep, err := parseEnvelope("assignment.created", payload, "http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	if ep.EventType != string(webhooks.EventAssignmentCreated) {
		t.Fatalf("event_type=%q", ep.EventType)
	}
	if ep.Title != "Essay 1" || ep.CourseCode != "ENG-101" {
		t.Fatalf("parsed fields: %+v", ep)
	}
	if ep.URL == "" {
		t.Fatal("expected url")
	}
}

func TestParseEnvelope_FillsCourseURLFromOrigin(t *testing.T) {
	payload := []byte(`{
		"event_type": "announcement.created",
		"data": {
			"courseId": "550e8400-e29b-41d4-a716-446655440000",
			"title": "Office hours moved"
		}
	}`)
	ep, err := parseEnvelope("announcement.created", payload, "https://lextures.test")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://lextures.test/courses/550e8400-e29b-41d4-a716-446655440000"
	if ep.URL != want {
		t.Fatalf("url=%q want %q", ep.URL, want)
	}
}

func TestParseEnvelope_GradeReleasedIncludesStudent(t *testing.T) {
	payload := []byte(`{
		"event_type": "grade.released",
		"data": {
			"studentUserId": "660e8400-e29b-41d4-a716-446655440001",
			"title": "Quiz 2",
			"pointsEarned": 18.5
		}
	}`)
	ep, err := parseEnvelope("grade.released", payload, "http://localhost:5173")
	if err != nil {
		t.Fatal(err)
	}
	if ep.StudentUserID != "660e8400-e29b-41d4-a716-446655440001" {
		t.Fatalf("studentUserId=%q", ep.StudentUserID)
	}
	if ep.PointsEarned != 18.5 {
		t.Fatalf("pointsEarned=%v", ep.PointsEarned)
	}
}

func TestGradeChannelBlocked_DefaultSettings(t *testing.T) {
	if !gradeChannelBlocked(string(webhooks.EventGradeReleased), false) {
		t.Fatal("grade.released must be channel-blocked by default")
	}
	if gradeChannelBlocked(string(webhooks.EventAssignmentCreated), false) {
		t.Fatal("assignment.created should not be blocked")
	}
}

func TestGradeChannelBlocked_ExplicitOptIn(t *testing.T) {
	if gradeChannelBlocked(string(webhooks.EventGradeReleased), true) {
		t.Fatal("grade.released should post to channel when explicitly enabled")
	}
}