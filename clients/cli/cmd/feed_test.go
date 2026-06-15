package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func resetFeedFlags() {
	feedFlags.course = ""
	feedChannelsCreateFlags.name = ""
	feedChannelsUpdateFlags.name = ""
	feedChannelsDeleteFlags.force = false
	feedChannelsDeleteInput = nil
	feedPostFlags.channel = ""
	feedPostFlags.body = ""
	feedRecentFlags.channel = ""
	feedRecentFlags.n = 20
}

func sampleFeedChannel(id, name string) feedChannel {
	return feedChannel{
		ID:        id,
		Name:      name,
		SortOrder: 0,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// ============================================================
// feed channels list
// ============================================================

func TestFeedChannelsList_Success(t *testing.T) {
	channels := []feedChannel{sampleFeedChannel("c1", "general"), sampleFeedChannel("c2", "announcements")}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/courses/CS101/feed/channels" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(feedChannelsBody{Channels: channels})
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"

	var out bytes.Buffer
	feedChannelsListCmd.SetOut(&out)
	if err := runFeedChannelsList(feedChannelsListCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "general") || !strings.Contains(out.String(), "announcements") {
		t.Errorf("output = %q; want both channels", out.String())
	}
}

func TestFeedChannelsList_RequiresCourse(t *testing.T) {
	resetFeedFlags()
	if err := runFeedChannelsList(feedChannelsListCmd, nil); err == nil {
		t.Fatal("expected error when --course missing")
	}
}

// ============================================================
// feed channels create
// ============================================================

func TestFeedChannelsCreate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/courses/CS101/feed/channels" {
			http.NotFound(w, r)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "homework" {
			t.Errorf("name = %q; want homework", body["name"])
		}
		_ = json.NewEncoder(w).Encode(sampleFeedChannel("c9", "homework"))
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedChannelsCreateFlags.name = "homework"

	var out bytes.Buffer
	feedChannelsCreateCmd.SetOut(&out)
	if err := runFeedChannelsCreate(feedChannelsCreateCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "homework") || !strings.Contains(out.String(), "c9") {
		t.Errorf("output = %q; want created channel", out.String())
	}
}

func TestFeedChannelsCreate_RequiresName(t *testing.T) {
	resetFeedFlags()
	feedFlags.course = "CS101"
	if err := runFeedChannelsCreate(feedChannelsCreateCmd, nil); err == nil {
		t.Fatal("expected error when --name missing")
	}
}

// ============================================================
// feed channels update
// ============================================================

func TestFeedChannelsUpdate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/courses/CS101/feed/channels/"+testChannelUUID {
			http.NotFound(w, r)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(sampleFeedChannel(testChannelUUID, body["name"]))
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedChannelsUpdateFlags.name = "renamed"

	var out bytes.Buffer
	feedChannelsUpdateCmd.SetOut(&out)
	if err := runFeedChannelsUpdate(feedChannelsUpdateCmd, []string{testChannelUUID}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Errorf("output = %q; want renamed", out.String())
	}
}

// ============================================================
// feed channels delete
// ============================================================

func TestFeedChannelsDelete_Force(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/courses/CS101/feed/channels/"+testChannelUUID {
			http.NotFound(w, r)
			return
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedChannelsDeleteFlags.force = true

	var out bytes.Buffer
	feedChannelsDeleteCmd.SetOut(&out)
	if err := runFeedChannelsDelete(feedChannelsDeleteCmd, []string{testChannelUUID}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !called {
		t.Error("expected DELETE to be called")
	}
	if !strings.Contains(out.String(), "Deleted channel "+testChannelUUID) {
		t.Errorf("output = %q; want deleted message", out.String())
	}
}

func TestFeedChannelsDelete_AbortsOnNo(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedChannelsDeleteInput = strings.NewReader("n\n")

	var out bytes.Buffer
	feedChannelsDeleteCmd.SetOut(&out)
	if err := runFeedChannelsDelete(feedChannelsDeleteCmd, []string{"c1"}); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if called {
		t.Error("expected DELETE NOT to be called after abort")
	}
	if !strings.Contains(out.String(), "Aborted") {
		t.Errorf("output = %q; want aborted", out.String())
	}
}

// ============================================================
// feed post
// ============================================================

const testChannelUUID = "11111111-1111-1111-1111-111111111111"

func TestFeedPost_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/courses/CS101/feed/channels/"+testChannelUUID+"/messages" {
			http.NotFound(w, r)
			return
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["body"] != "hello world" {
			t.Errorf("body = %q; want hello world", body["body"])
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "m1"})
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedPostFlags.channel = testChannelUUID
	feedPostFlags.body = "hello world"

	var out bytes.Buffer
	feedPostCmd.SetOut(&out)
	if err := runFeedPost(feedPostCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(out.String(), "m1") {
		t.Errorf("output = %q; want message id", out.String())
	}
}

func TestFeedPost_RequiresChannelAndBody(t *testing.T) {
	resetFeedFlags()
	feedFlags.course = "CS101"
	if err := runFeedPost(feedPostCmd, nil); err == nil {
		t.Fatal("expected error when --channel missing")
	}
	feedPostFlags.channel = testChannelUUID
	if err := runFeedPost(feedPostCmd, nil); err == nil {
		t.Fatal("expected error when --body missing")
	}
}

// ============================================================
// feed recent
// ============================================================

func TestFeedRecent_LimitFilter(t *testing.T) {
	var gotLimit string
	msgs := []feedMessage{
		{ID: "m1", AuthorEmail: "a@x.com", Body: "first", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "m2", AuthorEmail: "b@x.com", Body: "second", CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{ID: "m3", AuthorEmail: "c@x.com", Body: "third", CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/courses/CS101/feed/channels/"+testChannelUUID+"/messages" {
			http.NotFound(w, r)
			return
		}
		gotLimit = r.URL.Query().Get("limit")
		_ = json.NewEncoder(w).Encode(feedMessagesBody{Messages: msgs})
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedRecentFlags.channel = testChannelUUID
	feedRecentFlags.n = 2

	var out bytes.Buffer
	feedRecentCmd.SetOut(&out)
	if err := runFeedRecent(feedRecentCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if gotLimit != "2" {
		t.Errorf("limit query = %q; want 2", gotLimit)
	}
	// Newest 2 shown; oldest ("first") dropped.
	if strings.Contains(out.String(), "first") {
		t.Errorf("output = %q; should not include oldest message", out.String())
	}
	if !strings.Contains(out.String(), "second") || !strings.Contains(out.String(), "third") {
		t.Errorf("output = %q; want newest two messages", out.String())
	}
}

// TestFeedPost_ResolvesChannelName covers posting with a channel name (not a
// UUID): the CLI should look up the channel list and resolve to its id.
func TestFeedPost_ResolvesChannelName(t *testing.T) {
	var postedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/courses/CS101/feed/channels":
			_ = json.NewEncoder(w).Encode(feedChannelsBody{Channels: []feedChannel{sampleFeedChannel(testChannelUUID, "general")}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/messages"):
			postedPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "m1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedPostFlags.channel = "general"
	feedPostFlags.body = "@everyone, welcome to class"

	var out bytes.Buffer
	feedPostCmd.SetOut(&out)
	if err := runFeedPost(feedPostCmd, nil); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	want := "/api/v1/courses/CS101/feed/channels/" + testChannelUUID + "/messages"
	if postedPath != want {
		t.Errorf("posted path = %q; want %q (name should resolve to id)", postedPath, want)
	}
}

func TestFeedPost_UnknownChannelName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/courses/CS101/feed/channels" {
			_ = json.NewEncoder(w).Encode(feedChannelsBody{Channels: []feedChannel{sampleFeedChannel(testChannelUUID, "general")}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	setCfg(srv.URL, "test-key")
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedPostFlags.channel = "nonexistent"
	feedPostFlags.body = "hi"

	if err := runFeedPost(feedPostCmd, nil); err == nil {
		t.Fatal("expected error for unknown channel name")
	}
}

func TestFeedRecent_RequiresPositiveN(t *testing.T) {
	resetFeedFlags()
	feedFlags.course = "CS101"
	feedRecentFlags.channel = "c1"
	feedRecentFlags.n = 0
	if err := runFeedRecent(feedRecentCmd, nil); err == nil {
		t.Fatal("expected error when -n <= 0")
	}
}
