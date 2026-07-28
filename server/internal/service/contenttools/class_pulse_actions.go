package contenttools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/service/contenttools/analytics"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/class_pulse"
)

var classPulseAggCache = analytics.NewAggregateCache(30 * time.Second)

func init() {
	RegisterActionHandler(class_pulse.ID, "vote", handleClassPulseVote)
	RegisterActionHandler(class_pulse.ID, "aggregate", handleClassPulseAggregate)
}

func handleClassPulseVote(ctx ActionContext) (*ActionResult, error) {
	cfg := class_pulse.ParseConfig(ctx.ConfigJSON)
	st := class_pulse.ParseState(ctx.StateJSON)

	var in struct {
		OptionID string `json:"optionId"`
		Round    int    `json:"round"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid vote input: %w", err)
		}
	}
	round := in.Round
	if round == 0 {
		round = 1
	}
	if round != 1 && round != 2 {
		ObserveClassPulseVote(round, "invalid_round")
		return &ActionResult{
			Result: map[string]any{
				"error":   "invalid_round",
				"message": "Round must be 1 or 2.",
			},
		}, nil
	}
	if round == 2 && !cfg.AllowSecondVote {
		ObserveClassPulseVote(round, "revote_disabled")
		return &ActionResult{
			Result: map[string]any{
				"error":   "revote_disabled",
				"message": "Second vote is not enabled for this poll.",
			},
		}, nil
	}
	if round == 2 && !st.HasVotedRound(1) {
		ObserveClassPulseVote(round, "round1_required")
		return &ActionResult{
			Result: map[string]any{
				"error":   "round1_required",
				"message": "Submit your first vote before revoting.",
			},
		}, nil
	}

	optionID := strings.TrimSpace(in.OptionID)
	if optionID == "" {
		return nil, fmt.Errorf("optionId is required")
	}
	if class_pulse.FindOption(cfg, optionID) == nil {
		return nil, fmt.Errorf("unknown optionId")
	}

	if existing := st.VoteForRound(round); existing != nil {
		ObserveClassPulseVote(round, "already_voted")
		return &ActionResult{
			Result: map[string]any{
				"error":   "already_voted",
				"message": "You already voted in this round. Your vote cannot be changed.",
				"state":   classPulseStateView(st),
			},
		}, nil
	}

	now := class_pulse.NowRFC3339()
	st.Votes = append(st.Votes, class_pulse.Vote{Round: round, OptionID: optionID, At: now})
	st.Draft = nil
	st.V = 1
	if st.SawAggregateAt == "" {
		st.SawAggregateAt = now
	}
	finalRound := 1
	if cfg.AllowSecondVote {
		finalRound = 2
	}
	if round >= finalRound {
		st.CompletedAt = now
	}
	if class_pulse.ShouldRevealCorrect(cfg, st) && cfg.CorrectOptionID != "" {
		ok := optionID == cfg.CorrectOptionID
		st.Correct = &ok
	}

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}

	invalidateClassPulseCache(ctx.InstanceID)
	aggPayload := classPulseBuildAggregatePayload(ctx, cfg, &st, false)

	ObserveClassPulseVote(round, "ok")
	if round == 2 {
		if v1 := st.VoteForRound(1); v1 != nil && v1.OptionID != optionID {
			ObserveClassPulseRevoteShift(1)
		} else {
			ObserveClassPulseRevoteShift(0)
		}
	}

	result := map[string]any{
		"state": classPulseStateView(st),
	}
	for k, v := range aggPayload {
		result[k] = v
	}
	if reveal := classPulseRevealPayload(cfg, st); reveal != nil {
		result["reveal"] = reveal
	}

	status := StatusInProgress
	if st.CompletedAt != "" {
		status = StatusCompleted
	}

	return &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     status,
	}, nil
}

func handleClassPulseAggregate(ctx ActionContext) (*ActionResult, error) {
	cfg := class_pulse.ParseConfig(ctx.ConfigJSON)
	st := class_pulse.ParseState(ctx.StateJSON)
	staff := ctx.InteractRole == "instructor" || ctx.InteractRole == "ta"
	if !staff && st.MaxVotedRound() == 0 {
		ObserveClassPulseVote(0, "aggregate_denied")
		return &ActionResult{
			Result: map[string]any{
				"error":   "vote_required",
				"message": "Vote before viewing the class distribution.",
			},
		}, nil
	}

	result := classPulseBuildAggregatePayload(ctx, cfg, &st, staff)
	if reveal := classPulseRevealPayload(cfg, st); reveal != nil {
		result["reveal"] = reveal
	}
	result["state"] = classPulseStateView(st)
	return &ActionResult{Result: result}, nil
}

func classPulseStateView(st class_pulse.State) map[string]any {
	out := map[string]any{"v": st.V}
	if len(st.Votes) > 0 {
		out["votes"] = st.Votes
	}
	if st.SawAggregateAt != "" {
		out["sawAggregateAt"] = st.SawAggregateAt
	}
	if st.CompletedAt != "" {
		out["completedAt"] = st.CompletedAt
	}
	if st.Correct != nil {
		out["correct"] = *st.Correct
	}
	return out
}

func classPulseRevealPayload(cfg class_pulse.Config, st class_pulse.State) map[string]any {
	if !class_pulse.ShouldRevealCorrect(cfg, st) {
		return nil
	}
	out := map[string]any{"correctOptionId": cfg.CorrectOptionID}
	if cfg.Explanation != "" {
		out["explanation"] = cfg.Explanation
	}
	return out
}

func invalidateClassPulseCache(instanceID uuid.UUID) {
	// Always invalidate — unit tests use uuid.Nil and still share the process cache.
	classPulseAggCache.InvalidatePrefix(instanceID.String())
	if instanceID != uuid.Nil {
		analytics.InvalidateForInstance(instanceID.String(), "", "")
	}
}

func classPulseSectionFilter(ctx ActionContext, cfg class_pulse.Config) *uuid.UUID {
	if !cfg.ScopeToSection || ctx.Pool == nil || ctx.Ctx == nil {
		return nil
	}
	sec, err := enrollment.GetStudentSectionID(ctx.Ctx, ctx.Pool, ctx.CourseID, ctx.PrincipalID)
	if err != nil || sec == nil {
		return nil
	}
	return sec
}

func classPulseBuildAggregatePayload(ctx ActionContext, cfg class_pulse.Config, self *class_pulse.State, staff bool) map[string]any {
	sectionFilter := classPulseSectionFilter(ctx, cfg)
	cacheKey := classPulseCacheKey(ctx.InstanceID, sectionFilter, staff)
	useCache := ctx.InstanceID != uuid.Nil

	if useCache {
		if cached, ok := classPulseAggCache.Get(cacheKey); ok {
			ObserveClassPulseAggregateCache(true)
			if m, ok := cached.(map[string]any); ok {
				return cloneMap(m)
			}
		}
		ObserveClassPulseAggregateCache(false)
	}

	rows := classPulseLoadRows(ctx, self)
	out := map[string]any{}

	r1 := class_pulse.BuildRoundAggregate(rows, 1, cfg, sectionFilter)
	if r1.Suppressed {
		ObserveClassPulseSuppressionHit()
	}
	out["aggregate"] = r1
	out["suppressed"] = r1.Suppressed
	if r1.Reason != "" {
		out["reason"] = r1.Reason
	}

	showRound2 := cfg.AllowSecondVote && (staff || (self != nil && self.HasVotedRound(2)))
	if showRound2 {
		r2 := class_pulse.BuildRoundAggregate(rows, 2, cfg, sectionFilter)
		out["aggregateRound2"] = r2
		if staff {
			out["shift"] = class_pulse.BuildShift(rows, sectionFilter)
		}
	}

	if useCache {
		classPulseAggCache.Set(cacheKey, cloneMap(out))
	}
	return out
}

func classPulseCacheKey(instanceID uuid.UUID, section *uuid.UUID, staff bool) string {
	sec := "all"
	if section != nil {
		sec = section.String()
	}
	role := "student"
	if staff {
		role = "staff"
	}
	return instanceID.String() + ":class_pulse:" + sec + ":" + role
}

func classPulseCallerRole(ctx ActionContext) string {
	switch ctx.InteractRole {
	case "instructor", "ta", "teacher", "teaching_assistant":
		return ctx.InteractRole
	default:
		if ctx.InteractRole != "" {
			return ctx.InteractRole
		}
		return "student"
	}
}

func classPulseLoadRows(ctx ActionContext, self *class_pulse.State) []class_pulse.VoteRow {
	if ctx.Pool == nil || ctx.Ctx == nil {
		if self == nil {
			return nil
		}
		return []class_pulse.VoteRow{{
			EnrollmentID: ctx.EnrollmentID,
			Role:         classPulseCallerRole(ctx),
			State:        *self,
		}}
	}
	raws, err := ctrepo.ListEnrollmentStatesForAggregate(ctx.Ctx, ctx.Pool, ctx.InstanceID)
	if err != nil {
		if self == nil {
			return nil
		}
		return []class_pulse.VoteRow{{
			EnrollmentID: ctx.EnrollmentID,
			Role:         classPulseCallerRole(ctx),
			State:        *self,
		}}
	}
	out := make([]class_pulse.VoteRow, 0, len(raws)+1)
	seenSelf := false
	for _, r := range raws {
		row := class_pulse.VoteRow{
			EnrollmentID: r.EnrollmentID,
			Role:         r.Role,
			SectionID:    r.SectionID,
			State:        class_pulse.ParseState(r.StateJSON),
		}
		if r.EnrollmentID == ctx.EnrollmentID && self != nil {
			row.State = *self
			seenSelf = true
		}
		out = append(out, row)
	}
	if self != nil && !seenSelf && self.MaxVotedRound() > 0 {
		out = append(out, class_pulse.VoteRow{
			EnrollmentID: ctx.EnrollmentID,
			Role:         classPulseCallerRole(ctx),
			SectionID:    classPulseSectionFilter(ctx, class_pulse.ParseConfig(ctx.ConfigJSON)),
			State:        *self,
		})
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// GuardStatePut refuses vote mutations via PUT (AC-4 / FR-8).
func GuardClassPulseStatePut(toolID string, current, next json.RawMessage) (blocked bool, message string) {
	if toolID != class_pulse.ID {
		return false, ""
	}
	return class_pulse.GuardStatePut(current, next)
}

var (
	classPulseMetricsOnce sync.Once

	classPulseVotesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_votes_total",
		Help:      "Class Pulse vote outcomes by round and outcome (CT.21).",
	}, []string{"round", "outcome"})

	classPulseSuppressionHitsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_class_pulse_suppression_hits_total",
		Help:      "Class Pulse aggregate responses withheld for small-n (CT.21).",
	})

	classPulseAggregateCacheHitsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_class_pulse_aggregate_cache_total",
		Help:      "Class Pulse aggregate cache hits/misses (CT.21).",
	}, []string{"result"})

	classPulseRevoteShiftMagnitude = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "lextures",
		Name:      "content_tool_class_pulse_revote_shift_magnitude",
		Help:      "Fraction of round-2 voters who changed option (CT.21).",
		Buckets:   []float64{0, 0.1, 0.25, 0.5, 0.75, 1},
	})
)

func registerClassPulseMetrics() {
	classPulseMetricsOnce.Do(func() {
		prometheus.MustRegister(
			classPulseVotesTotal,
			classPulseSuppressionHitsTotal,
			classPulseAggregateCacheHitsTotal,
			classPulseRevoteShiftMagnitude,
		)
		classPulseVotesTotal.WithLabelValues("1", "_reserved").Add(0)
		classPulseAggregateCacheHitsTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveClassPulseVote increments lextures_content_tool_votes_total{round,outcome}.
func ObserveClassPulseVote(round int, outcome string) {
	registerClassPulseMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	classPulseVotesTotal.WithLabelValues(strconv.Itoa(round), outcome).Inc()
}

// ObserveClassPulseSuppressionHit increments small-n suppression counter.
func ObserveClassPulseSuppressionHit() {
	registerClassPulseMetrics()
	classPulseSuppressionHitsTotal.Inc()
}

// ObserveClassPulseAggregateCache records cache hit/miss.
func ObserveClassPulseAggregateCache(hit bool) {
	registerClassPulseMetrics()
	if hit {
		classPulseAggregateCacheHitsTotal.WithLabelValues("hit").Inc()
	} else {
		classPulseAggregateCacheHitsTotal.WithLabelValues("miss").Inc()
	}
}

// ObserveClassPulseRevoteShift records the fraction of revoters who changed option.
func ObserveClassPulseRevoteShift(magnitude float64) {
	registerClassPulseMetrics()
	if magnitude < 0 {
		magnitude = 0
	}
	if magnitude > 1 {
		magnitude = 1
	}
	classPulseRevoteShiftMagnitude.Observe(magnitude)
}
