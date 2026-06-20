package calendar_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	calendarsvc "github.com/lextures/lextures/server/internal/service/calendar"
)

func TestBuildICalendar_VEVENTFields(t *testing.T) {
	itemID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	due := time.Date(2026, 4, 15, 23, 59, 0, 0, time.UTC)
	events := []calendarsvc.Event{{
		ItemID:      itemID,
		CourseCode:  "BIO101",
		CourseTitle: "Biology",
		Kind:        "assignment",
		Title:       "Lab report",
		Description: "Submit PDF",
		Start:       due,
		End:         due.Add(time.Hour),
	}}
	body := string(calendarsvc.BuildICalendar(events, "https://app.lextures.io", "America/New_York", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)))
	if !strings.Contains(body, "BEGIN:VCALENDAR") {
		t.Fatal("missing VCALENDAR")
	}
	if !strings.Contains(body, itemID.String()+"@lextures.io") {
		t.Fatal("missing stable UID")
	}
	if !strings.Contains(body, "SUMMARY:Biology: Lab report") {
		t.Fatal("missing summary")
	}
	if !strings.Contains(body, "URL:https://app.lextures.io/courses/BIO101/assignments/"+itemID.String()) {
		t.Fatal("missing deep link URL")
	}
	if !strings.Contains(body, "DESCRIPTION:Submit PDF") {
		t.Fatal("missing description")
	}
}

func TestBuildICalendar_DateRangeAllDay(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	events := []calendarsvc.Event{{
		ItemID:       uuid.New(),
		CourseCode:   "MATH",
		Kind:         "quiz",
		Title:        "Midterm",
		Description:  "Quiz window.",
		Start:        start,
		End:          end,
		AllDay:       true,
		IsQuizWindow: true,
	}}
	body := string(calendarsvc.BuildICalendar(events, "", "UTC", time.Now()))
	if !strings.Contains(body, "DTSTART;VALUE=DATE:20260401") {
		t.Fatal("expected all-day DTSTART")
	}
	if !strings.Contains(body, "DTEND;VALUE=DATE:20260408") {
		t.Fatal("expected all-day DTEND")
	}
}

func TestEscapeText(t *testing.T) {
	events := []calendarsvc.Event{{
		ItemID: uuid.New(), CourseCode: "X", Kind: "assignment", Title: "A; B, C",
		Start: time.Now(), End: time.Now().Add(time.Hour),
	}}
	body := string(calendarsvc.BuildICalendar(events, "", "UTC", time.Now()))
	if !strings.Contains(body, "SUMMARY:A\\; B\\, C") {
		t.Fatalf("expected escaped summary, got: %q", body)
	}
}
