package inline_discussion

import (
	"encoding/json"
	"strings"
	"time"
)

// ParseConfig unmarshals instructor config with defaults applied.
func ParseConfig(raw json.RawMessage) Config {
	cfg := DefaultConfig()
	if len(raw) == 0 {
		return cfg
	}
	var overlay struct {
		Prompt            *string        `json:"prompt"`
		PostBeforeYouSee  *bool          `json:"postBeforeYouSee"`
		AllowReplies      *bool          `json:"allowReplies"`
		RequiredPosts     *int           `json:"requiredPosts"`
		RequiredReplies   *int           `json:"requiredReplies"`
		Anonymity         *AnonymityMode `json:"anonymity"`
		EditWindowMinutes *int           `json:"editWindowMinutes"`
		AllowDelete       *bool          `json:"allowDelete"`
		Sort              *SortOrder     `json:"sort"`
		PageSize          *int           `json:"pageSize"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Prompt != nil {
		cfg.Prompt = strings.TrimSpace(*overlay.Prompt)
	}
	if overlay.PostBeforeYouSee != nil {
		cfg.PostBeforeYouSee = *overlay.PostBeforeYouSee
	}
	if overlay.AllowReplies != nil {
		cfg.AllowReplies = *overlay.AllowReplies
	}
	if overlay.RequiredPosts != nil && *overlay.RequiredPosts >= 0 {
		cfg.RequiredPosts = *overlay.RequiredPosts
	}
	if overlay.RequiredReplies != nil && *overlay.RequiredReplies >= 0 {
		cfg.RequiredReplies = *overlay.RequiredReplies
	}
	if overlay.Anonymity != nil {
		switch *overlay.Anonymity {
		case AnonymityNamed, AnonymityAnonymousToPeers:
			cfg.Anonymity = *overlay.Anonymity
		}
	}
	if overlay.EditWindowMinutes != nil && *overlay.EditWindowMinutes >= 0 {
		cfg.EditWindowMinutes = *overlay.EditWindowMinutes
	}
	if overlay.AllowDelete != nil {
		cfg.AllowDelete = *overlay.AllowDelete
	}
	if overlay.Sort != nil {
		switch *overlay.Sort {
		case SortOldest, SortNewest:
			cfg.Sort = *overlay.Sort
		}
	}
	if overlay.PageSize != nil {
		ps := *overlay.PageSize
		if ps < 1 {
			ps = 1
		}
		if ps > 100 {
			ps = 100
		}
		cfg.PageSize = ps
	}
	return cfg
}

// ParseState unmarshals learner participation state with defaults.
func ParseState(raw json.RawMessage) State {
	st := EmptyState()
	if len(raw) == 0 {
		return st
	}
	_ = json.Unmarshal(raw, &st)
	if st.V == 0 {
		st.V = 1
	}
	if st.MyPostIDs == nil {
		st.MyPostIDs = []string{}
	}
	if st.MyReplyIDs == nil {
		st.MyReplyIDs = []string{}
	}
	return st
}

// NowRFC3339 returns UTC now.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// HasRootPost reports whether the learner has posted at least one top-level post.
func (s State) HasRootPost() bool {
	return len(s.MyPostIDs) > 0
}

// CanSeePeers reports whether peer posts may be returned for this learner.
func CanSeePeers(cfg Config, st State, staff bool) bool {
	if staff {
		return true
	}
	if !cfg.PostBeforeYouSee {
		return true
	}
	return st.HasRootPost()
}

// IsComplete reports whether participation requirements are met.
func IsComplete(cfg Config, st State) bool {
	if len(st.MyPostIDs) < cfg.RequiredPosts {
		return false
	}
	if len(st.MyReplyIDs) < cfg.RequiredReplies {
		return false
	}
	return true
}

// WithinEditWindow reports whether createdAt is still editable.
func WithinEditWindow(cfg Config, createdAt time.Time, now time.Time) bool {
	if cfg.EditWindowMinutes <= 0 {
		return false
	}
	deadline := createdAt.Add(time.Duration(cfg.EditWindowMinutes) * time.Minute)
	return !now.After(deadline)
}

// ContainsID reports whether id is in the slice.
func ContainsID(ids []string, id string) bool {
	id = strings.TrimSpace(id)
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// AppendUnique appends id when absent.
func AppendUnique(ids []string, id string) []string {
	if ContainsID(ids, id) {
		return ids
	}
	return append(ids, id)
}

// GuardStatePut refuses mutation of participation ids via PUT (posts go through actions).
func GuardStatePut(current, next json.RawMessage) (blocked bool, message string) {
	cur := ParseState(current)
	if len(cur.MyPostIDs) == 0 && len(cur.MyReplyIDs) == 0 && cur.ThreadID == "" && cur.CompletedAt == "" {
		return false, ""
	}
	if len(next) == 0 {
		return false, ""
	}
	nxt := ParseState(next)
	if !sameIDs(cur.MyPostIDs, nxt.MyPostIDs) ||
		!sameIDs(cur.MyReplyIDs, nxt.MyReplyIDs) {
		return true, "Participation records are locked after posting; use reset to start over."
	}
	// Draft-only puts may omit threadId / completedAt; block explicit clears/changes.
	if nxt.ThreadID != "" && cur.ThreadID != "" && nxt.ThreadID != cur.ThreadID {
		return true, "Participation records are locked after posting; use reset to start over."
	}
	if cur.CompletedAt != "" && nxt.CompletedAt != "" && nxt.CompletedAt != cur.CompletedAt {
		return true, "Participation records are locked after posting; use reset to start over."
	}
	if cur.CompletedAt != "" && nxt.CompletedAt == "" && len(nxt.MyPostIDs) > 0 {
		return true, "Participation records are locked after posting; use reset to start over."
	}
	return false, ""
}

func sameIDs(a, b []string) bool {
	if len(a) != len(b) {
		if len(b) == 0 {
			// Draft-only PUT may omit id arrays.
			return true
		}
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
