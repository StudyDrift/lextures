package contenttools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/lextures/lextures/server/internal/service/contenttools/tools/inline_questions"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/media_checkpoints"
)

func init() {
	RegisterActionHandler(media_checkpoints.ID, "answer_checkpoint", handleMediaCheckpointsAnswer)
	RegisterActionHandler(media_checkpoints.ID, "record_progress", handleMediaCheckpointsRecordProgress)
}

func mediaCheckpointsStatus(prev string, complete bool, engaged bool) string {
	if complete {
		return StatusCompleted
	}
	switch prev {
	case StatusCompleted:
		return StatusCompleted
	case StatusSubmitted, StatusInProgress:
		if engaged {
			return StatusInProgress
		}
		return prev
	default:
		if engaged {
			return StatusInProgress
		}
		return prev
	}
}

func handleMediaCheckpointsAnswer(ctx ActionContext) (*ActionResult, error) {
	cfg := media_checkpoints.ParseConfig(ctx.ConfigJSON)
	st := media_checkpoints.ParseState(ctx.StateJSON)

	var in struct {
		CheckpointID string `json:"checkpointId"`
		Value        any    `json:"value"`
		TranscriptOnly *bool `json:"transcriptOnly"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid answer_checkpoint input: %w", err)
		}
	}
	in.CheckpointID = strings.TrimSpace(in.CheckpointID)
	if in.CheckpointID == "" {
		return nil, fmt.Errorf("checkpointId is required")
	}
	cp := media_checkpoints.FindCheckpoint(cfg, in.CheckpointID)
	if cp == nil {
		return nil, fmt.Errorf("unknown checkpointId")
	}
	if in.Value == nil {
		return nil, fmt.Errorf("value is required")
	}

	remaining := media_checkpoints.AttemptsRemaining(cfg, st, in.CheckpointID)
	if remaining == 0 {
		ObserveMediaCheckpoints("max_attempts")
		return &ActionResult{
			Result: map[string]any{
				"error":             "max_attempts",
				"message":           "No attempts remaining for this checkpoint.",
				"attemptsRemaining": 0,
				"checkpointId":      in.CheckpointID,
			},
		}, nil
	}

	// CT.8 — screen short_text before storage.
	if cp.Question.Type == inline_questions.TypeShortText {
		text, ok := in.Value.(string)
		if !ok {
			if b, err := json.Marshal(in.Value); err == nil {
				text = string(b)
			}
		}
		screen := ScreenFreeText(text, FilterActionFlag, true)
		if screen.Action == FilterActionBlock || screen.Crisis {
			ObserveMediaCheckpoints("filtered")
			out := map[string]any{
				"error":         "filtered",
				"message":       screen.Guidance,
				"preserveInput": true,
			}
			if screen.Crisis {
				out["crisis"] = true
			}
			return &ActionResult{Result: out}, nil
		}
	}

	grade := media_checkpoints.GradeCheckpoint(*cp, in.Value)
	ans := st.Answers[in.CheckpointID]
	ans.Attempts = append(ans.Attempts, media_checkpoints.Attempt{
		Value:   in.Value,
		Correct: grade.Correct,
		At:      media_checkpoints.NowRFC3339(),
	})
	st.Answers[in.CheckpointID] = ans
	if grade.Correct || media_checkpoints.AttemptsRemaining(cfg, st, in.CheckpointID) == 0 {
		ans.Done = true
		st.Answers[in.CheckpointID] = ans
	}
	if in.TranscriptOnly != nil && *in.TranscriptOnly {
		st.UsedTranscriptOnly = true
		ObserveMediaCheckpointsTranscriptOnly()
	}

	raw, max := media_checkpoints.ComputeScore(cfg, st)
	st.ScoreRaw = &raw
	st.ScoreMax = &max
	complete := media_checkpoints.AllRequiredComplete(cfg, st)
	if complete {
		st.CompletedAt = media_checkpoints.NowRFC3339()
	}

	result := map[string]any{
		"correct":           grade.Correct,
		"attemptsRemaining": media_checkpoints.AttemptsRemaining(cfg, st, in.CheckpointID),
		"checkpointId":      in.CheckpointID,
		"done":              ans.Done,
	}
	if media_checkpoints.CheckpointShowFeedback(*cp) {
		if grade.Feedback != "" {
			result["feedback"] = grade.Feedback
		}
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	outcome := "incorrect"
	if grade.Correct {
		outcome = "correct"
	}
	ObserveMediaCheckpoints(outcome)

	out := &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     mediaCheckpointsStatus(ctx.Status, complete, true),
	}
	if !cfg.PracticeOnly {
		out.ScoreRaw = &raw
		out.ScoreMax = &max
	}
	return out, nil
}

func handleMediaCheckpointsRecordProgress(ctx ActionContext) (*ActionResult, error) {
	cfg := media_checkpoints.ParseConfig(ctx.ConfigJSON)
	st := media_checkpoints.ParseState(ctx.StateJSON)

	var in struct {
		WatchedSegments    [][2]float64 `json:"watchedSegments"`
		FurthestSec        *float64     `json:"furthestSec"`
		UsedTranscriptOnly *bool        `json:"usedTranscriptOnly"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid record_progress input: %w", err)
		}
	}

	furthest := st.FurthestSec
	if in.FurthestSec != nil && *in.FurthestSec > furthest {
		furthest = *in.FurthestSec
	}
	media_checkpoints.MergeWatchProgress(&st, in.WatchedSegments, furthest)
	if in.UsedTranscriptOnly != nil && *in.UsedTranscriptOnly {
		if !st.UsedTranscriptOnly {
			ObserveMediaCheckpointsTranscriptOnly()
		}
		st.UsedTranscriptOnly = true
	}

	// Watch data must never forge a grade — recompute score only from answers.
	raw, max := media_checkpoints.ComputeScore(cfg, st)
	st.ScoreRaw = &raw
	st.ScoreMax = &max

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveMediaCheckpointsProgress()
	engaged := len(st.WatchedSegments) > 0 || st.UsedTranscriptOnly || len(st.Answers) > 0
	complete := media_checkpoints.AllRequiredComplete(cfg, st)
	return &ActionResult{
		Result: map[string]any{
			"furthestSec":     st.FurthestSec,
			"watchedSegments": st.WatchedSegments,
			"segmentCount":    len(st.WatchedSegments),
		},
		StatePatch: patch,
		Status:     mediaCheckpointsStatus(ctx.Status, complete, engaged),
	}, nil
}

var (
	mediaCheckpointsMetricsOnce sync.Once

	mediaCheckpointsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_checkpoints_total",
		Help:      "Media Checkpoints answer outcomes by result (CT.19).",
	}, []string{"result"})

	mediaCheckpointsTranscriptOnlyTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_checkpoints_transcript_only_total",
		Help:      "Media Checkpoints transcript-only usage (CT.19).",
	})

	mediaCheckpointsProgressTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_checkpoints_progress_total",
		Help:      "Media Checkpoints watch-progress writes (CT.19).",
	})
)

func registerMediaCheckpointsMetrics() {
	mediaCheckpointsMetricsOnce.Do(func() {
		prometheus.MustRegister(
			mediaCheckpointsTotal,
			mediaCheckpointsTranscriptOnlyTotal,
			mediaCheckpointsProgressTotal,
		)
		mediaCheckpointsTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveMediaCheckpoints increments lextures_content_tool_checkpoints_total{result}.
func ObserveMediaCheckpoints(result string) {
	registerMediaCheckpointsMetrics()
	if result == "" {
		result = "_unknown"
	}
	mediaCheckpointsTotal.WithLabelValues(result).Inc()
}

// ObserveMediaCheckpointsTranscriptOnly increments transcript-only usage.
func ObserveMediaCheckpointsTranscriptOnly() {
	registerMediaCheckpointsMetrics()
	mediaCheckpointsTranscriptOnlyTotal.Inc()
}

// ObserveMediaCheckpointsProgress increments progress-write counter.
func ObserveMediaCheckpointsProgress() {
	registerMediaCheckpointsMetrics()
	mediaCheckpointsProgressTotal.Inc()
}
