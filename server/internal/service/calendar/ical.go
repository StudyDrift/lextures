package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildICalendar serializes events to RFC 5545 text/calendar bytes.
func BuildICalendar(events []Event, webOrigin, timezone string, now time.Time) []byte {
	origin := strings.TrimRight(strings.TrimSpace(webOrigin), "/")
	if timezone == "" {
		timezone = "UTC"
	}
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Lextures//Calendar Feeds//EN\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")
	writeVTimezone(&b, timezone)

	for _, ev := range events {
		writeVEvent(&b, ev, origin, now)
	}
	b.WriteString("END:VCALENDAR\r\n")
	return []byte(b.String())
}

func writeVTimezone(b *strings.Builder, tz string) {
	// Minimal VTIMEZONE stub; clients treat DTSTART as floating or UTC when TZID omitted.
	b.WriteString("BEGIN:VTIMEZONE\r\n")
	b.WriteString("TZID:" + escapeText(tz) + "\r\n")
	b.WriteString("X-LIC-LOCATION:" + escapeText(tz) + "\r\n")
	b.WriteString("END:VTIMEZONE\r\n")
}

func writeVEvent(b *strings.Builder, ev Event, origin string, now time.Time) {
	uid := fmt.Sprintf("%s@lextures.io", ev.ItemID.String())
	summary := ev.Title
	if ev.CourseTitle != "" {
		summary = ev.CourseTitle + ": " + ev.Title
	}
	url := itemURL(origin, ev.CourseCode, ev.Kind, ev.ItemID)

	b.WriteString("BEGIN:VEVENT\r\n")
	b.WriteString("UID:" + escapeText(uid) + "\r\n")
	b.WriteString("DTSTAMP:" + now.UTC().Format("20060102T150405Z") + "\r\n")
	if ev.AllDay {
		b.WriteString("DTSTART;VALUE=DATE:" + formatDate(ev.Start) + "\r\n")
		b.WriteString("DTEND;VALUE=DATE:" + formatDate(ev.End) + "\r\n")
	} else {
		b.WriteString("DTSTART:" + ev.Start.UTC().Format("20060102T150405Z") + "\r\n")
		b.WriteString("DTEND:" + ev.End.UTC().Format("20060102T150405Z") + "\r\n")
	}
	b.WriteString("SUMMARY:" + escapeText(summary) + "\r\n")
	if ev.Description != "" {
		b.WriteString("DESCRIPTION:" + escapeText(ev.Description) + "\r\n")
	}
	if url != "" {
		b.WriteString("URL:" + escapeText(url) + "\r\n")
	}
	b.WriteString("END:VEVENT\r\n")
}

func itemURL(origin, courseCode, kind string, itemID uuid.UUID) string {
	if origin == "" || courseCode == "" {
		return ""
	}
	pathKind := "assignments"
	if kind == "quiz" {
		pathKind = "quizzes"
	}
	return fmt.Sprintf("%s/courses/%s/%s/%s", origin, courseCode, pathKind, itemID.String())
}

func formatDate(t time.Time) string {
	return t.UTC().Format("20060102")
}

func escapeText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
