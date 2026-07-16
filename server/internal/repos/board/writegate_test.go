package board

import (
	"errors"
	"testing"
	"time"
)

func TestCheckWriteAllowed_LockAndFreeze(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	future := now.Add(5 * time.Minute)
	past := now.Add(-time.Minute)

	locked := &Board{Locked: true}
	if err := CheckWriteAllowed(locked, false, WritePost, now); !errors.Is(err, ErrBoardLocked) {
		t.Fatalf("locked non-manager: got %v", err)
	}
	if err := CheckWriteAllowed(locked, true, WritePost, now); err != nil {
		t.Fatalf("locked manager: %v", err)
	}

	frozen := &Board{FrozenUntil: &future}
	if err := CheckWriteAllowed(frozen, false, WritePost, now); !errors.Is(err, ErrBoardFrozen) {
		t.Fatalf("frozen post: got %v", err)
	}
	if err := CheckWriteAllowed(frozen, false, WriteReact, now); err != nil {
		t.Fatalf("freeze should not block react: %v", err)
	}
	expired := &Board{FrozenUntil: &past}
	if err := CheckWriteAllowed(expired, false, WritePost, now); err != nil {
		t.Fatalf("expired freeze: %v", err)
	}
}

func TestApplyMinorsModerationFloor(t *testing.T) {
	mode, filter := ApplyMinorsModerationFloor(ModerationOpen, FilterFlag, true)
	if mode != ModerationApproval || filter != FilterBlock {
		t.Fatalf("got %s/%s", mode, filter)
	}
	mode, filter = ApplyMinorsModerationFloor(ModerationOpen, FilterFlag, false)
	if mode != ModerationOpen || filter != FilterFlag {
		t.Fatalf("got %s/%s", mode, filter)
	}
}

func TestFilterVisiblePosts(t *testing.T) {
	author := "a1"
	posts := []Post{
		{ID: "1", Status: PostStatusApproved},
		{ID: "2", Status: PostStatusPending, AuthorID: &author},
		{ID: "3", Status: PostStatusPending, AuthorID: strPtr("other")},
		{ID: "4", Status: PostStatusApproved, Hidden: true},
		{ID: "5", Status: PostStatusRejected},
		{ID: "6", Status: PostStatusApproved, Removed: true},
	}
	peer := FilterVisiblePosts(posts, author, false)
	if len(peer) != 2 || peer[0].ID != "1" || peer[1].ID != "2" {
		t.Fatalf("peer visible=%v", ids(peer))
	}
	mgr := FilterVisiblePosts(posts, author, true)
	if len(mgr) != 3 {
		t.Fatalf("manager visible=%v want 3 (approved + both pending)", ids(mgr))
	}
}

func strPtr(s string) *string { return &s }

func ids(posts []Post) []string {
	out := make([]string, len(posts))
	for i, p := range posts {
		out[i] = p.ID
	}
	return out
}
