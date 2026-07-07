package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildMeetingPayload_WithDuration(t *testing.T) {
	payload, err := buildMeetingPayload("Sync", "2026-07-01T10:00:00Z", "60", "", "jitsi")
	if err != nil {
		t.Fatal(err)
	}
	if payload["title"] != "Sync" {
		t.Fatalf("payload=%v", payload)
	}
	if payload["scheduledEnd"] == "" {
		t.Fatalf("missing end: %v", payload)
	}
}

func TestMeetingsList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/courses/CS101/meetings" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"meetings": []any{map[string]any{"id": "m1", "title": "Office Hours", "status": "scheduled"}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	meetingsListFlags.course = "CS101"
	defer func() { meetingsListFlags.course = "" }()
	setCfg(srv.URL, "tok")
	if err := meetingsListCmd.RunE(meetingsListCmd, nil); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestOfficeHoursSet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/courses/CS101/availability" {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"window": map[string]any{"id": "w1"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	officeHoursSetFlags.course = "CS101"
	officeHoursSetFlags.file = writeTempJSON(t, map[string]any{
		"dayOfWeek": 1, "startTime": "09:00", "endTime": "11:00",
	})
	defer func() {
		officeHoursSetFlags.course = ""
		officeHoursSetFlags.file = ""
	}()
	setCfg(srv.URL, "tok")
	if err := officeHoursSetCmd.RunE(officeHoursSetCmd, nil); err != nil {
		t.Fatalf("set: %v", err)
	}
}

func TestCalendarTokenRotate_RedactsInJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/me/calendar-token" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token": "secret-token", "feedUrl": "http://x/cal.ics?token=secret-token",
				"expiresAt": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	globalFlags.jsonOut = true
	defer func() { globalFlags.jsonOut = false }()
	setCfg(srv.URL, "tok")
	var out strings.Builder
	calendarTokenRotateCmd.SetOut(&out)
	if err := calendarTokenRotateCmd.RunE(calendarTokenRotateCmd, nil); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if strings.Contains(out.String(), "secret-token") {
		t.Fatalf("token leaked in json: %s", out.String())
	}
}