package media_checkpoints

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
	var overlay Config
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return cfg
	}
	cfg = overlay
	if cfg.Media.Source == "" {
		cfg.Media.Source = MediaSourceCourseFile
	}
	if cfg.Media.Kind == "" {
		cfg.Media.Kind = MediaKindVideo
	}
	if cfg.TranscriptSource == "" {
		cfg.TranscriptSource = TranscriptInline
	}
	if cfg.Checkpoints == nil {
		cfg.Checkpoints = []Checkpoint{}
	}
	if len(cfg.Checkpoints) > 40 {
		cfg.Checkpoints = cfg.Checkpoints[:40]
	}
	return cfg
}

// ParseState unmarshals learner state with defaults and segment sanitization.
func ParseState(raw json.RawMessage) State {
	st := EmptyState()
	if len(raw) == 0 {
		return st
	}
	_ = json.Unmarshal(raw, &st)
	if st.V == 0 {
		st.V = 1
	}
	if st.Answers == nil {
		st.Answers = map[string]CheckpointAnswer{}
	}
	st.WatchedSegments = NormalizeSegments(st.WatchedSegments, 5)
	if st.FurthestSec < 0 {
		st.FurthestSec = 0
	}
	for _, seg := range st.WatchedSegments {
		if seg[1] > st.FurthestSec {
			st.FurthestSec = seg[1]
		}
	}
	return st
}

// FindCheckpoint returns a checkpoint by id.
func FindCheckpoint(cfg Config, id string) *Checkpoint {
	id = strings.TrimSpace(id)
	for i := range cfg.Checkpoints {
		if cfg.Checkpoints[i].ID == id {
			return &cfg.Checkpoints[i]
		}
	}
	return nil
}

// AttemptsUsed returns how many attempts the learner has for a checkpoint.
func AttemptsUsed(st State, checkpointID string) int {
	ans, ok := st.Answers[checkpointID]
	if !ok {
		return 0
	}
	return len(ans.Attempts)
}

// AttemptsRemaining returns remaining attempts for a checkpoint.
func AttemptsRemaining(cfg Config, st State, checkpointID string) int {
	cp := FindCheckpoint(cfg, checkpointID)
	if cp == nil {
		return 0
	}
	max := CheckpointAttempts(*cp)
	left := max - AttemptsUsed(st, checkpointID)
	if left < 0 {
		return 0
	}
	return left
}

// IsCheckpointDone reports whether the learner finished a checkpoint
// (marked done, or last attempt correct, or attempts exhausted).
func IsCheckpointDone(cfg Config, st State, checkpointID string) bool {
	cp := FindCheckpoint(cfg, checkpointID)
	if cp == nil {
		return false
	}
	ans, ok := st.Answers[checkpointID]
	if !ok {
		return false
	}
	if ans.Done {
		return true
	}
	if len(ans.Attempts) == 0 {
		return false
	}
	last := ans.Attempts[len(ans.Attempts)-1]
	if last.Correct {
		return true
	}
	return AttemptsRemaining(cfg, st, checkpointID) == 0
}

// LastAttemptCorrect reports whether the latest attempt for a checkpoint was correct.
func LastAttemptCorrect(st State, checkpointID string) bool {
	ans, ok := st.Answers[checkpointID]
	if !ok || len(ans.Attempts) == 0 {
		return false
	}
	return ans.Attempts[len(ans.Attempts)-1].Correct
}

// ComputeScore returns correct/total over checkpoints (last attempt by default).
func ComputeScore(cfg Config, st State) (raw, max float64) {
	max = float64(len(cfg.Checkpoints))
	if max == 0 {
		return 0, 0
	}
	for _, cp := range cfg.Checkpoints {
		if LastAttemptCorrect(st, cp.ID) {
			raw++
		}
	}
	return raw, max
}

// AllRequiredComplete reports whether every required checkpoint is done.
func AllRequiredComplete(cfg Config, st State) bool {
	if len(cfg.Checkpoints) == 0 {
		return false
	}
	for _, cp := range cfg.Checkpoints {
		if !CheckpointRequired(cp) {
			continue
		}
		if !IsCheckpointDone(cfg, st, cp.ID) {
			return false
		}
	}
	// At least one checkpoint answered for completion semantics.
	any := false
	for _, cp := range cfg.Checkpoints {
		if AttemptsUsed(st, cp.ID) > 0 {
			any = true
			break
		}
	}
	return any
}

// MergeWatchProgress merges new watched segments and updates furthest.
func MergeWatchProgress(st *State, segments [][2]float64, furthest float64) {
	combined := append(append([][2]float64{}, st.WatchedSegments...), segments...)
	st.WatchedSegments = NormalizeSegments(combined, 5)
	if furthest > st.FurthestSec {
		st.FurthestSec = furthest
	}
	for _, seg := range st.WatchedSegments {
		if seg[1] > st.FurthestSec {
			st.FurthestSec = seg[1]
		}
	}
}

// NowRFC3339 returns UTC now for attempt timestamps.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
