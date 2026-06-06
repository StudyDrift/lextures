package icsgenerator

import (
	"strings"
	"testing"
	"time"
)

func TestBuildEvent_RFC5545Fields(t *testing.T) {
	start := time.Date(2025, 11, 18, 16, 0, 0, 0, time.UTC)
	end := start.Add(15 * time.Minute)
	ics := BuildEvent(Event{
		UID:       "conference-test@lextures",
		Summary:   "Parent-Teacher Conference with Ms. Smith",
		Location:  "Room 204",
		Start:     start,
		End:       end,
		Organizer: "Ms. Smith",
	})
	for _, want := range []string{
		"BEGIN:VCALENDAR",
		"BEGIN:VEVENT",
		"UID:conference-test@lextures",
		"DTSTART:20251118T160000Z",
		"DTEND:20251118T161500Z",
		"SUMMARY:Parent-Teacher Conference with Ms. Smith",
		"LOCATION:Room 204",
		"END:VEVENT",
		"END:VCALENDAR",
	} {
		if !strings.Contains(ics, want) {
			t.Fatalf("ICS missing %q\n%s", want, ics)
		}
	}
}

func TestEscapeText(t *testing.T) {
	got := EscapeText("Hello; world,\nline two")
	want := `Hello\; world\,` + `\n` + `line two`
	if got != want {
		t.Fatalf("EscapeText = %q, want %q", got, want)
	}
}
