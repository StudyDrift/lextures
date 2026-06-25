package webhooksvc

import (
	"encoding/json"
	"testing"
)

func TestParseSubscriptionSettings(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"source":"zapier","includePII":true}`)
	got := ParseSubscriptionSettings(raw)
	if got.Source != "zapier" || !got.IncludePII {
		t.Fatalf("got %+v", got)
	}
}

func TestStripPIIFields(t *testing.T) {
	t.Parallel()
	data := map[string]any{"studentUserId": "abc", "studentEmail": "x@y.com", "email": "z@y.com"}
	stripPIIFields(data)
	if _, ok := data["studentEmail"]; ok {
		t.Fatal("expected studentEmail stripped")
	}
	if data["studentUserId"] != "abc" {
		t.Fatal("expected studentUserId preserved")
	}
}

func TestAdaptPayloadForSubscription_NoPII(t *testing.T) {
	t.Parallel()
	env := map[string]any{
		"event_id":    "00000000-0000-0000-0000-000000000001",
		"event_type":  "enrollment.created",
		"api_version": "2026-04-17",
		"created_at":  "2026-01-01T00:00:00Z",
		"data": map[string]any{
			"studentUserId": "00000000-0000-0000-0000-000000000002",
			"studentEmail":  "secret@example.com",
		},
	}
	payload, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	out, err := AdaptPayloadForSubscription(nil, nil, payload, json.RawMessage(`{"includePII":false}`))
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	data := parsed["data"].(map[string]any)
	if _, ok := data["studentEmail"]; ok {
		t.Fatal("email should be stripped without includePII")
	}
}
