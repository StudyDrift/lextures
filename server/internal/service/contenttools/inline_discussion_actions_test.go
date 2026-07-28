package contenttools_test

import (
	"encoding/json"
	"testing"

	"github.com/lextures/lextures/server/internal/service/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/analytics"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_discussion"
)

func TestInlineDiscussionRegistryAndProjector(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(inline_discussion.ID)
	if m == nil {
		t.Fatal("missing inline_discussion")
	}
	if !analytics.HasProjector(inline_discussion.ID) {
		t.Fatal("missing projector")
	}
	found := false
	for _, a := range m.Actions {
		if a.Name == "post" || a.Name == "thread" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected post/thread actions")
	}
	caps := map[string]bool{}
	for _, c := range m.Capabilities {
		caps[c] = true
	}
	if !caps["peer_visible"] || !caps["state"] {
		t.Fatalf("capabilities=%v", m.Capabilities)
	}
}

func TestInlineDiscussionThreadLockedWithoutPool(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(inline_discussion.ID)
	cfg, _ := json.Marshal(map[string]any{
		"prompt":           "Share one insight.",
		"postBeforeYouSee": true,
		"requiredPosts":    1,
	})
	st, _ := json.Marshal(inline_discussion.EmptyState())
	res, err := contenttools.DispatchAction(m, "thread", contenttools.ActionContext{
		ConfigJSON:   cfg,
		StateJSON:    st,
		InteractRole: "student",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["canSeePeers"] != false {
		t.Fatalf("want canSeePeers=false: %#v", res.Result)
	}
	if res.Result["locked"] != true {
		t.Fatalf("want locked: %#v", res.Result)
	}
	if res.Result["prompt"] != "Share one insight." {
		t.Fatalf("prompt: %#v", res.Result["prompt"])
	}
}

func TestInlineDiscussionFilterBlockPreservesDraft(t *testing.T) {
	reg, err := contenttools.BuildBuiltinRegistry()
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(inline_discussion.ID)
	cfg, _ := json.Marshal(map[string]any{"prompt": "Discuss."})
	st, _ := json.Marshal(inline_discussion.EmptyState())
	in, _ := json.Marshal(map[string]any{"text": "this is fucking awful", "idempotencyKey": "k1"})
	res, err := contenttools.DispatchAction(m, "post", contenttools.ActionContext{
		ConfigJSON:   cfg,
		StateJSON:    st,
		Input:        in,
		InteractRole: "student",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Result["error"] != "filtered" {
		t.Fatalf("want filtered, got %#v", res.Result)
	}
	if res.Result["preserveInput"] != true {
		t.Fatalf("preserveInput: %#v", res.Result)
	}
	if res.StatePatch != nil {
		t.Fatal("blocked post must not write state")
	}
}

func TestGuardInlineDiscussionStatePut(t *testing.T) {
	cur, _ := json.Marshal(inline_discussion.State{V: 1, MyPostIDs: []string{"p"}, ThreadID: "t"})
	next, _ := json.Marshal(inline_discussion.State{V: 1, MyPostIDs: []string{"p", "q"}, ThreadID: "t"})
	blocked, msg := contenttools.GuardInlineDiscussionStatePut(inline_discussion.ID, cur, next)
	if !blocked || msg == "" {
		t.Fatal("expected block")
	}
	if blocked, _ := contenttools.GuardInlineDiscussionStatePut("other", cur, next); blocked {
		t.Fatal("other tool")
	}
}

func TestInlineDiscussionAnonymityProjectionHelpers(t *testing.T) {
	cfg := inline_discussion.DefaultConfig()
	cfg.Anonymity = inline_discussion.AnonymityAnonymousToPeers
	st := inline_discussion.EmptyState()
	st.MyPostIDs = []string{"mine"}
	if !inline_discussion.CanSeePeers(cfg, st, false) {
		t.Fatal("after post can see")
	}
}
