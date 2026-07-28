package inline_discussion_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_discussion"
)

func TestParseConfigDefaults(t *testing.T) {
	cfg := inline_discussion.ParseConfig(nil)
	if !cfg.PostBeforeYouSee || !cfg.AllowReplies || cfg.RequiredPosts != 1 {
		t.Fatalf("defaults: %+v", cfg)
	}
	if cfg.Anonymity != inline_discussion.AnonymityNamed || cfg.EditWindowMinutes != 5 {
		t.Fatalf("defaults anonymity/edit: %+v", cfg)
	}
	raw, _ := json.Marshal(map[string]any{
		"prompt":           "What stood out?",
		"postBeforeYouSee": false,
		"anonymity":        "anonymous_to_peers",
		"requiredReplies":  2,
		"sort":             "newest",
		"pageSize":         10,
	})
	cfg = inline_discussion.ParseConfig(raw)
	if cfg.Prompt != "What stood out?" || cfg.PostBeforeYouSee || cfg.RequiredReplies != 2 {
		t.Fatalf("overlay: %+v", cfg)
	}
	if cfg.Anonymity != inline_discussion.AnonymityAnonymousToPeers || cfg.Sort != inline_discussion.SortNewest {
		t.Fatalf("overlay enums: %+v", cfg)
	}
}

func TestCanSeePeersAndCompletion(t *testing.T) {
	cfg := inline_discussion.DefaultConfig()
	cfg.PostBeforeYouSee = true
	cfg.RequiredPosts = 1
	cfg.RequiredReplies = 2
	st := inline_discussion.EmptyState()
	if inline_discussion.CanSeePeers(cfg, st, false) {
		t.Fatal("student should not see peers before posting")
	}
	if !inline_discussion.CanSeePeers(cfg, st, true) {
		t.Fatal("staff always sees peers")
	}
	st.MyPostIDs = []string{"p1"}
	if !inline_discussion.CanSeePeers(cfg, st, false) {
		t.Fatal("after root post peers should be visible")
	}
	if inline_discussion.IsComplete(cfg, st) {
		t.Fatal("need replies")
	}
	st.MyReplyIDs = []string{"r1", "r2"}
	if !inline_discussion.IsComplete(cfg, st) {
		t.Fatal("should complete")
	}
}

func TestEditWindow(t *testing.T) {
	cfg := inline_discussion.DefaultConfig()
	cfg.EditWindowMinutes = 5
	created := time.Now().UTC().Add(-2 * time.Minute)
	if !inline_discussion.WithinEditWindow(cfg, created, time.Now().UTC()) {
		t.Fatal("should be editable")
	}
	created = time.Now().UTC().Add(-10 * time.Minute)
	if inline_discussion.WithinEditWindow(cfg, created, time.Now().UTC()) {
		t.Fatal("window closed")
	}
}

func TestTipTapRoundTripAndMeta(t *testing.T) {
	meta := &inline_discussion.PostMeta{Endorsed: true, EndorsedAt: "t"}
	body := inline_discussion.TipTapDocFromText("Hello peers", meta)
	if got := inline_discussion.TextFromTipTap(body); got != "Hello peers" {
		t.Fatalf("text=%q", got)
	}
	gotMeta := inline_discussion.MetaFromTipTap(body)
	if !gotMeta.Endorsed {
		t.Fatalf("meta=%+v", gotMeta)
	}
	gotMeta.Removed = true
	tomb := inline_discussion.WithMeta(body, gotMeta)
	if !inline_discussion.MetaFromTipTap(tomb).Removed {
		t.Fatal("expected removed")
	}
}

func TestGuardStatePut(t *testing.T) {
	cur, _ := json.Marshal(inline_discussion.State{V: 1, MyPostIDs: []string{"a"}, ThreadID: "t"})
	next, _ := json.Marshal(inline_discussion.State{V: 1, Draft: "x"})
	blocked, msg := inline_discussion.GuardStatePut(cur, next)
	if blocked {
		t.Fatalf("draft-only should be allowed: %s", msg)
	}
	next2, _ := json.Marshal(inline_discussion.State{V: 1, MyPostIDs: []string{"a", "b"}, ThreadID: "t"})
	blocked, msg = inline_discussion.GuardStatePut(cur, next2)
	if !blocked || msg == "" {
		t.Fatal("mutating post ids must block")
	}
}

func TestThreadTitle(t *testing.T) {
	got := inline_discussion.ThreadTitleForInstance("abc")
	if got != "ct.inline:abc" {
		t.Fatalf("title=%q", got)
	}
}
