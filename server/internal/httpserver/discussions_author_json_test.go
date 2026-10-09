package httpserver

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/repos/discussions"
)

func TestThreadAndPostJSONIncludeAuthor(t *testing.T) {
	name := "Chase Willden"
	avatar := "https://cdn.example/chase.png"
	now := time.Date(2026, 10, 9, 15, 4, 5, 0, time.UTC)
	thread := &discussions.ThreadDetail{
		ThreadRow: discussions.ThreadRow{
			ID:                uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			ForumID:           uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			AuthorID:          uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			AuthorDisplayName: &name,
			AuthorAvatarURL:   &avatar,
			Title:             "Welcome",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		Body: json.RawMessage(`{"type":"doc"}`),
	}
	raw, err := json.Marshal(threadDetailJSON(thread))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["authorDisplayName"] != name {
		t.Fatalf("authorDisplayName = %v", got["authorDisplayName"])
	}
	if got["authorAvatarUrl"] != avatar {
		t.Fatalf("authorAvatarUrl = %v", got["authorAvatarUrl"])
	}

	post := &discussions.PostRow{
		ID:                uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		ThreadID:          thread.ID,
		AuthorID:          thread.AuthorID,
		AuthorDisplayName: &name,
		Body:              json.RawMessage(`{"type":"doc"}`),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	raw, err = json.Marshal(postJSON(post))
	if err != nil {
		t.Fatal(err)
	}
	got = map[string]any{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["authorDisplayName"] != name {
		t.Fatalf("post authorDisplayName = %v", got["authorDisplayName"])
	}
	if _, ok := got["authorAvatarUrl"]; !ok {
		t.Fatal("post authorAvatarUrl missing")
	}
	if got["authorAvatarUrl"] != nil {
		t.Fatalf("post authorAvatarUrl = %v, want null", got["authorAvatarUrl"])
	}
}
