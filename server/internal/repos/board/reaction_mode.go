package board

import (
	"fmt"
	"strings"
)

// Reaction modes for a board (VC.5).
const (
	ReactionModeNone  = "none"
	ReactionModeLike  = "like"
	ReactionModeVote  = "vote"
	ReactionModeStar  = "star"
	ReactionModeGrade = "grade"
)

// Reaction kinds stored on post_reactions rows.
const (
	ReactionKindLike  = "like"
	ReactionKindVote  = "vote"
	ReactionKindStar  = "star"
	ReactionKindGrade = "grade"
)

// NormalizeReactionMode returns a canonical mode or an error.
func NormalizeReactionMode(raw string) (string, error) {
	m := strings.TrimSpace(strings.ToLower(raw))
	switch m {
	case ReactionModeNone, ReactionModeLike, ReactionModeVote, ReactionModeStar, ReactionModeGrade:
		return m, nil
	default:
		return "", fmt.Errorf("board: invalid reaction_mode %q", raw)
	}
}

// ModeToKind maps board reaction_mode to the reaction kind used in post_reactions.
// Returns empty for none.
func ModeToKind(mode string) string {
	switch mode {
	case ReactionModeLike:
		return ReactionKindLike
	case ReactionModeVote:
		return ReactionKindVote
	case ReactionModeStar:
		return ReactionKindStar
	case ReactionModeGrade:
		return ReactionKindGrade
	default:
		return ""
	}
}
