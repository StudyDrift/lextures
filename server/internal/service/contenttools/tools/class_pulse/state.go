package class_pulse

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
		Question        *string        `json:"question"`
		Options         []Option       `json:"options"`
		CorrectOptionID *string        `json:"correctOptionId"`
		Explanation     *string        `json:"explanation"`
		AllowSecondVote *bool          `json:"allowSecondVote"`
		RevealCorrect   *RevealCorrect `json:"revealCorrect"`
		MinRespondents  *int           `json:"minRespondents"`
		ScopeToSection  *bool          `json:"scopeToSection"`
		ShowPercentages *bool          `json:"showPercentages"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	if overlay.Question != nil {
		cfg.Question = *overlay.Question
	}
	if overlay.Options != nil {
		cfg.Options = overlay.Options
	}
	if overlay.CorrectOptionID != nil {
		cfg.CorrectOptionID = strings.TrimSpace(*overlay.CorrectOptionID)
	}
	if overlay.Explanation != nil {
		cfg.Explanation = *overlay.Explanation
	}
	if overlay.AllowSecondVote != nil {
		cfg.AllowSecondVote = *overlay.AllowSecondVote
	}
	if overlay.RevealCorrect != nil {
		switch *overlay.RevealCorrect {
		case RevealAfterVote, RevealAfterRevote, RevealNever:
			cfg.RevealCorrect = *overlay.RevealCorrect
		}
	}
	if overlay.MinRespondents != nil {
		cfg.MinRespondents = *overlay.MinRespondents
	}
	if overlay.ScopeToSection != nil {
		cfg.ScopeToSection = *overlay.ScopeToSection
	}
	if overlay.ShowPercentages != nil {
		cfg.ShowPercentages = *overlay.ShowPercentages
	}
	return cfg
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
	return st
}

// FindOption returns an option by id.
func FindOption(cfg Config, id string) *Option {
	id = strings.TrimSpace(id)
	for i := range cfg.Options {
		if cfg.Options[i].ID == id {
			return &cfg.Options[i]
		}
	}
	return nil
}

// VoteForRound returns the vote for a round, if any.
func (s State) VoteForRound(round int) *Vote {
	for i := range s.Votes {
		if s.Votes[i].Round == round {
			return &s.Votes[i]
		}
	}
	return nil
}

// HasVotedRound reports whether the learner has submitted a vote for round.
func (s State) HasVotedRound(round int) bool {
	return s.VoteForRound(round) != nil
}

// MaxVotedRound returns the highest committed round (0 if none).
func (s State) MaxVotedRound() int {
	max := 0
	for _, v := range s.Votes {
		if v.Round > max {
			max = v.Round
		}
	}
	return max
}

// NowRFC3339 returns UTC now for vote timestamps.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ShouldRevealCorrect reports whether correctness may be shown to the learner.
func ShouldRevealCorrect(cfg Config, st State) bool {
	if strings.TrimSpace(cfg.CorrectOptionID) == "" {
		return false
	}
	switch cfg.RevealCorrect {
	case RevealNever:
		return false
	case RevealAfterVote:
		return st.HasVotedRound(1)
	case RevealAfterRevote:
		if !cfg.AllowSecondVote {
			return st.HasVotedRound(1)
		}
		return st.HasVotedRound(2)
	default:
		return false
	}
}

// GuardStatePut refuses mutation of committed votes via PUT (votes go through action).
func GuardStatePut(current, next json.RawMessage) (blocked bool, message string) {
	cur := ParseState(current)
	if len(cur.Votes) == 0 {
		return false, ""
	}
	if len(next) == 0 {
		return false, ""
	}
	nxt := ParseState(next)
	if votesChanged(cur.Votes, nxt.Votes) {
		return true, "Votes are locked after submit; use reset to start over."
	}
	return false, ""
}

func votesChanged(a, b []Vote) bool {
	if len(a) != len(b) {
		// Allow omitting votes in a draft-only PUT by treating empty next votes as unchanged
		// when the client only sends draft fields — but ParseState would leave Votes nil/empty.
		// If next has fewer votes, that is a mutation attempt.
		if len(b) == 0 {
			return false
		}
		return true
	}
	for i := range a {
		if a[i].Round != b[i].Round || a[i].OptionID != b[i].OptionID || a[i].At != b[i].At {
			return true
		}
	}
	return false
}
