package flashcards

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
		Title            *string `json:"title"`
		Cards            []Card  `json:"cards"`
		ReversePractice  *bool   `json:"reversePractice"`
		SessionCap       *int    `json:"sessionCap"`
		Shuffle          *bool   `json:"shuffle"`
		RequireFirstPass *bool   `json:"requireFirstPass"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Title != nil {
		cfg.Title = strings.TrimSpace(*overlay.Title)
	}
	if overlay.Cards != nil {
		cfg.Cards = normalizeCards(overlay.Cards)
	}
	if overlay.ReversePractice != nil {
		cfg.ReversePractice = *overlay.ReversePractice
	}
	if overlay.SessionCap != nil {
		cap := *overlay.SessionCap
		if cap < 1 {
			cap = 1
		}
		if cap > 20 {
			cap = 20
		}
		cfg.SessionCap = cap
	}
	if overlay.Shuffle != nil {
		cfg.Shuffle = *overlay.Shuffle
	}
	if overlay.RequireFirstPass != nil {
		cfg.RequireFirstPass = *overlay.RequireFirstPass
	}
	return cfg
}

func normalizeCards(in []Card) []Card {
	out := make([]Card, 0, len(in))
	seen := map[string]struct{}{}
	for _, c := range in {
		id := strings.TrimSpace(c.ID)
		front := strings.TrimSpace(c.Front)
		back := strings.TrimSpace(c.Back)
		if id == "" || front == "" || back == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, Card{
			ID:        id,
			Front:     front,
			Back:      back,
			FrontLang: strings.TrimSpace(c.FrontLang),
			BackLang:  strings.TrimSpace(c.BackLang),
			ImageURL:  strings.TrimSpace(c.ImageURL),
			ImageAlt:  strings.TrimSpace(c.ImageAlt),
			Hint:      strings.TrimSpace(c.Hint),
		})
	}
	if len(out) > 20 {
		out = out[:20]
	}
	return out
}

// ParseState unmarshals learner state with defaults.
func ParseState(raw json.RawMessage) State {
	st := EmptyState()
	if len(raw) == 0 {
		return st
	}
	_ = json.Unmarshal(raw, &st)
	if st.V == 0 {
		st.V = 1
	}
	if st.Cards == nil {
		st.Cards = map[string]CardProgress{}
	}
	if st.Sessions == nil {
		st.Sessions = []SessionRecord{}
	}
	return st
}

// FindCard returns a card by id.
func FindCard(cfg Config, id string) *Card {
	id = strings.TrimSpace(id)
	for i := range cfg.Cards {
		if cfg.Cards[i].ID == id {
			return &cfg.Cards[i]
		}
	}
	return nil
}

// NowRFC3339 returns UTC now.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// FirstPassComplete reports whether every config card has been rated at least once.
func FirstPassComplete(cfg Config, st State) bool {
	if len(cfg.Cards) == 0 {
		return false
	}
	for _, c := range cfg.Cards {
		prog, ok := st.Cards[c.ID]
		if !ok || prog.Seen < 1 || prog.FirstRating == nil {
			return false
		}
	}
	return true
}

// ApplyRating updates card progress and session counters for one rating.
func ApplyRating(st *State, cardID string, rating Rating, now string) {
	if st.Cards == nil {
		st.Cards = map[string]CardProgress{}
	}
	prog := st.Cards[cardID]
	prog.Seen++
	prog.LastRating = &rating
	prog.LastSeenAt = now
	if prog.FirstRating == nil {
		r := rating
		prog.FirstRating = &r
	}
	st.Cards[cardID] = prog
	if st.ActiveSession != nil {
		st.ActiveSession.Reviewed++
		st.ActiveSession.Revealed = false
		st.ActiveSession.Index++
	}
}

// EndActiveSession closes the active session into sessions history.
func EndActiveSession(st *State, now string) SessionRecord {
	rec := SessionRecord{StartedAt: now, EndedAt: now, Reviewed: 0}
	if st.ActiveSession == nil {
		return rec
	}
	rec = SessionRecord{
		StartedAt: st.ActiveSession.StartedAt,
		EndedAt:   now,
		Reviewed:  st.ActiveSession.Reviewed,
	}
	st.Sessions = append(st.Sessions, rec)
	st.ActiveSession = nil
	return rec
}

// MergeStates merges concurrent flashcard states (max seen, last-write ratings).
func MergeStates(a, b State) State {
	out := EmptyState()
	if a.V != 0 {
		out.V = a.V
	}
	if b.V != 0 {
		out.V = b.V
	}
	out.Cards = map[string]CardProgress{}
	keys := map[string]struct{}{}
	for k := range a.Cards {
		keys[k] = struct{}{}
	}
	for k := range b.Cards {
		keys[k] = struct{}{}
	}
	for k := range keys {
		pa, okA := a.Cards[k]
		pb, okB := b.Cards[k]
		switch {
		case okA && !okB:
			out.Cards[k] = pa
		case okB && !okA:
			out.Cards[k] = pb
		default:
			merged := pa
			if pb.Seen > merged.Seen {
				merged.Seen = pb.Seen
			}
			if pb.LastSeenAt > merged.LastSeenAt {
				merged.LastSeenAt = pb.LastSeenAt
				merged.LastRating = pb.LastRating
			}
			if merged.FirstRating == nil {
				merged.FirstRating = pb.FirstRating
			}
			out.Cards[k] = merged
		}
	}
	// Prefer longer session history.
	if len(b.Sessions) > len(a.Sessions) {
		out.Sessions = append([]SessionRecord{}, b.Sessions...)
	} else {
		out.Sessions = append([]SessionRecord{}, a.Sessions...)
	}
	if a.FirstPassCompletedAt != "" {
		out.FirstPassCompletedAt = a.FirstPassCompletedAt
	}
	if b.FirstPassCompletedAt != "" && (out.FirstPassCompletedAt == "" || b.FirstPassCompletedAt < out.FirstPassCompletedAt) {
		out.FirstPassCompletedAt = b.FirstPassCompletedAt
	}
	// Keep the more advanced active session if any.
	if a.ActiveSession != nil && (b.ActiveSession == nil || a.ActiveSession.Reviewed >= b.ActiveSession.Reviewed) {
		cp := *a.ActiveSession
		out.ActiveSession = &cp
	} else if b.ActiveSession != nil {
		cp := *b.ActiveSession
		out.ActiveSession = &cp
	}
	return out
}
