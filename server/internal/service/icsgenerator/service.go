// Package icsgenerator builds RFC 5545 iCalendar payloads (plan 13.12).
package icsgenerator

import (
	"fmt"
	"strings"
	"time"
)

// Event describes a calendar event for ICS generation.
type Event struct {
	UID       string
	Summary   string
	Location  string
	Start     time.Time
	End       time.Time
	Organizer string
}

// BuildEvent returns a VCALENDAR document for a single VEVENT.
func BuildEvent(ev Event) string {
	dtstamp := ev.Start.UTC().Format("20060102T150405Z")
	dtstart := ev.Start.UTC().Format("20060102T150405Z")
	dtend := ev.End.UTC().Format("20060102T150405Z")

	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Lextures//Conference Scheduling//EN",
		"CALSCALE:GREGORIAN",
		"METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:" + EscapeText(ev.UID),
		"DTSTAMP:" + dtstamp,
		"DTSTART:" + dtstart,
		"DTEND:" + dtend,
		"SUMMARY:" + EscapeText(ev.Summary),
	}
	if ev.Location != "" {
		lines = append(lines, "LOCATION:"+EscapeText(ev.Location))
	}
	if ev.Organizer != "" {
		lines = append(lines, "ORGANIZER;CN="+EscapeText(ev.Organizer)+":MAILTO:noreply@lextures.local")
	}
	lines = append(lines, "END:VEVENT", "END:VCALENDAR")
	return strings.Join(lines, "\r\n") + "\r\n"
}

// EscapeText escapes special characters per RFC 5545 TEXT value rules.
func EscapeText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// ConferenceUID builds a stable UID for a conference slot.
func ConferenceUID(slotID string) string {
	return fmt.Sprintf("conference-%s@lextures", slotID)
}
