package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildAudienceJSON(t *testing.T) {
	students := string(buildAudienceJSON("students"))
	if !strings.Contains(students, "student") {
		t.Fatalf("students = %s", students)
	}
	all := string(buildAudienceJSON("all"))
	if !strings.Contains(all, "all") {
		t.Fatalf("all = %s", all)
	}
}

func TestMessagesList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/communication/messages" && r.URL.Query().Get("folder") == "inbox" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"messages": []any{map[string]any{
					"id": "m1", "subject": "Hello", "fromEmail": "a@example.com", "createdAt": "2026-06-01T00:00:00Z",
				}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	messagesListCmd.SetOut(&out)
	if err := messagesListCmd.RunE(messagesListCmd, nil); err != nil {
		t.Fatalf("messages list: %v", err)
	}
	if !strings.Contains(out.String(), "Hello") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestBroadcastsSend_RequiresYes(t *testing.T) {
	broadcastsSendFlags.org = "org-1"
	broadcastsSendFlags.subject = "Notice"
	broadcastsSendFlags.body = "Body"
	broadcastsSendFlags.yes = false
	defer func() {
		broadcastsSendFlags.org = ""
		broadcastsSendFlags.subject = ""
		broadcastsSendFlags.body = ""
		broadcastsSendFlags.yes = false
	}()
	err := broadcastsSendCmd.RunE(broadcastsSendCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v", err)
	}
}

func TestBroadcastsSend_Success(t *testing.T) {
	var saved map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/orgs/org-1/broadcasts" {
			_ = json.NewDecoder(r.Body).Decode(&saved)
			_ = json.NewEncoder(w).Encode(map[string]any{"broadcast": map[string]any{"id": "b1"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	broadcastsSendFlags.org = "org-1"
	broadcastsSendFlags.subject = "Notice"
	broadcastsSendFlags.body = "All clear"
	broadcastsSendFlags.yes = true
	broadcastsSendFlags.idempotencyKey = "key-1"
	defer func() {
		broadcastsSendFlags.org = ""
		broadcastsSendFlags.subject = ""
		broadcastsSendFlags.body = ""
		broadcastsSendFlags.yes = false
		broadcastsSendFlags.idempotencyKey = ""
	}()

	setCfg(srv.URL, "test-key")
	if err := broadcastsSendCmd.RunE(broadcastsSendCmd, nil); err != nil {
		t.Fatalf("broadcast send: %v", err)
	}
	if saved["subject"] != "Notice" {
		t.Fatalf("saved = %+v", saved)
	}
}

func TestNotificationsList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/notifications" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"notifications": []any{map[string]any{
					"id": "n1", "eventType": "grade_posted", "title": "Grade", "read": false,
				}},
				"unreadCount": 1,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	notificationsListCmd.SetOut(&out)
	if err := notificationsListCmd.RunE(notificationsListCmd, nil); err != nil {
		t.Fatalf("notifications: %v", err)
	}
	if !strings.Contains(out.String(), "grade_posted") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestNotificationPrefsGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me/notification-preferences" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"preferences": []any{map[string]any{"eventType": "grade_posted", "emailEnabled": true}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	globalFlags.jsonOut = true
	setCfg(srv.URL, "test-key")
	var out bytes.Buffer
	notificationPrefsGetCmd.SetOut(&out)
	if err := notificationPrefsGetCmd.RunE(notificationPrefsGetCmd, nil); err != nil {
		t.Fatalf("prefs get: %v", err)
	}
	if !strings.Contains(out.String(), "grade_posted") {
		t.Fatalf("output = %q", out.String())
	}
}
