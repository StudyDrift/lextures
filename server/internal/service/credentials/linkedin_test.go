package credentials

import (
	"net/url"
	"testing"
	"time"
)

func TestBuildLinkedInCertificationURL(t *testing.T) {
	issued := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	params := BuildLinkedInParams(
		"Intro to Data Science",
		"Lextures",
		"https://app.example.com/verify/abc",
		"abc",
		issued,
	)
	parsed, err := url.Parse(params.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	q := parsed.Query()
	if q.Get("startTask") != "CERTIFICATION_NAME" {
		t.Fatalf("startTask: %q", q.Get("startTask"))
	}
	if q.Get("name") != "Intro to Data Science" {
		t.Fatalf("name: %q", q.Get("name"))
	}
	if q.Get("organizationName") != "Lextures" {
		t.Fatalf("organizationName: %q", q.Get("organizationName"))
	}
	if q.Get("issueYear") != "2026" {
		t.Fatalf("issueYear: %q", q.Get("issueYear"))
	}
	if q.Get("issueMonth") != "6" {
		t.Fatalf("issueMonth: %q", q.Get("issueMonth"))
	}
	if q.Get("certUrl") != "https://app.example.com/verify/abc" {
		t.Fatalf("certUrl: %q", q.Get("certUrl"))
	}
	if q.Get("certId") != "abc" {
		t.Fatalf("certId: %q", q.Get("certId"))
	}
}