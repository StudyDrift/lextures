package contenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"

	ctrepo "github.com/lextures/lextures/server/internal/repos/contenttools"
	"github.com/lextures/lextures/server/internal/service/contenttools/tools/flashcards"
)

func init() {
	RegisterActionHandler(flashcards.ID, "start_session", handleFlashcardsStartSession)
	RegisterActionHandler(flashcards.ID, "rate", handleFlashcardsRate)
	RegisterActionHandler(flashcards.ID, "end_session", handleFlashcardsEndSession)
	RegisterActionHandler(flashcards.ID, "status", handleFlashcardsStatus)
}

func flashcardsPlatformSRS(ctx ActionContext) bool {
	if ctx.SRSPracticeEnabled == nil {
		return true // unit tests without the flag treat platform as on
	}
	return *ctx.SRSPracticeEnabled
}

func flashcardsCourseSRS(ctx ActionContext) bool {
	if ctx.Pool == nil || ctx.Ctx == nil {
		return false
	}
	on, err := flashcards.CourseSRSEnabled(ctx.Ctx, ctx.Pool, ctx.CourseID)
	if err != nil {
		return false
	}
	return on
}

func handleFlashcardsStartSession(ctx ActionContext) (*ActionResult, error) {
	cfg := flashcards.ParseConfig(ctx.ConfigJSON)
	if len(cfg.Cards) < 3 {
		return nil, fmt.Errorf("deck must have at least 3 cards")
	}
	st := flashcards.ParseState(ctx.StateJSON)
	platformOn := flashcardsPlatformSRS(ctx)
	courseOn := flashcardsCourseSRS(ctx)
	srsOn := flashcards.SRSActive(platformOn, courseOn)

	due, err := flashcards.LoadDueMap(ctx.Ctx, ctx.Pool, platformOn, courseOn, ctx.InstanceID, ctx.PrincipalID, cfg)
	if err != nil {
		due = map[string]flashcards.CardDueInfo{}
	}
	seed := ctx.EnrollmentID.String() + "|" + time.Now().UTC().Format(time.RFC3339Nano)
	queue := flashcards.SelectSessionQueue(cfg, st, due, time.Now().UTC(), seed)
	status := flashcards.ComputeDeckStatus(cfg, st, due, srsOn)

	if len(queue) == 0 {
		ObserveFlashcardsSession("caught_up")
		return &ActionResult{
			Result: map[string]any{
				"caughtUp":   true,
				"srsEnabled": srsOn,
				"status":     status,
				"state":      flashcardsStateView(st),
				"message":    "all_caught_up",
			},
		}, nil
	}

	now := flashcards.NowRFC3339()
	st.ActiveSession = &flashcards.ActiveSession{
		StartedAt: now,
		Queue:     queue,
		Index:     0,
		Reviewed:  0,
		Revealed:  false,
	}
	st.V = 1
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveFlashcardsSession("started")
	return &ActionResult{
		Result: map[string]any{
			"caughtUp":   false,
			"srsEnabled": srsOn,
			"status":     status,
			"queue":      queue,
			"current":    flashcardsCurrentCard(cfg, st),
			"state":      flashcardsStateView(st),
		},
		StatePatch: patch,
		Status:     StatusInProgress,
	}, nil
}

func handleFlashcardsRate(ctx ActionContext) (*ActionResult, error) {
	cfg := flashcards.ParseConfig(ctx.ConfigJSON)
	st := flashcards.ParseState(ctx.StateJSON)

	var in struct {
		CardID         string            `json:"cardId"`
		Rating         flashcards.Rating `json:"rating"`
		Side           string            `json:"side"`
		IdempotencyKey string            `json:"idempotencyKey"`
	}
	if len(ctx.Input) > 0 {
		if err := json.Unmarshal(ctx.Input, &in); err != nil {
			return nil, fmt.Errorf("invalid rate input: %w", err)
		}
	}
	_ = in.IdempotencyKey
	cardID := strings.TrimSpace(in.CardID)
	side := strings.TrimSpace(in.Side)
	if side == "" {
		side = flashcards.SideForward
	}
	if !flashcards.ValidSide(side) {
		return nil, fmt.Errorf("invalid side")
	}
	if !flashcards.ValidRating(in.Rating) {
		return nil, fmt.Errorf("invalid rating")
	}
	card := flashcards.FindCard(cfg, cardID)
	if card == nil {
		return nil, fmt.Errorf("unknown cardId")
	}
	if st.ActiveSession == nil {
		return &ActionResult{
			Result: map[string]any{
				"error":   "no_session",
				"message": "Start a session before rating cards.",
			},
		}, nil
	}
	if st.ActiveSession.Index < 0 || st.ActiveSession.Index >= len(st.ActiveSession.Queue) {
		return &ActionResult{
			Result: map[string]any{
				"error":   "session_complete",
				"message": "No more cards in this session.",
			},
		}, nil
	}
	cur := st.ActiveSession.Queue[st.ActiveSession.Index]
	if cur.CardID != cardID || cur.Side != side {
		return &ActionResult{
			Result: map[string]any{
				"error":   "card_mismatch",
				"message": "Rate the current card in the session queue.",
			},
		}, nil
	}

	now := flashcards.NowRFC3339()
	flashcards.ApplyRating(&st, cardID, in.Rating, now)

	platformOn := flashcardsPlatformSRS(ctx)
	courseOn := flashcardsCourseSRS(ctx)
	srsOn := flashcards.SRSActive(platformOn, courseOn)

	var nextDue *string
	var srsErr string
	if srsOn {
		dueAt, err := flashcards.SubmitCardRating(
			ctx.Ctx, ctx.Pool, platformOn, courseOn, ctx.CourseID, ctx.InstanceID, ctx.PrincipalID, *card, side, in.Rating,
		)
		if err != nil {
			srsErr = "srs_submit_failed"
			ObserveFlashcardsSRSSubmit("error")
		} else {
			ObserveFlashcardsSRSSubmit("ok")
			if dueAt != nil {
				s := dueAt.UTC().Format(time.RFC3339)
				nextDue = &s
			}
		}
	} else {
		ObserveFlashcardsSRSSubmit("skipped")
	}
	ObserveFlashcardsCardReview(string(in.Rating))

	if cfg.RequireFirstPass && flashcards.FirstPassComplete(cfg, st) && st.FirstPassCompletedAt == "" {
		st.FirstPassCompletedAt = now
	}

	sessionDone := st.ActiveSession.Index >= len(st.ActiveSession.Queue)
	var summary map[string]any
	if sessionDone {
		rec := flashcards.EndActiveSession(&st, now)
		summary = map[string]any{
			"reviewed": rec.Reviewed,
			"endedAt":  rec.EndedAt,
		}
		ObserveFlashcardsSession("completed")
	}

	due, _ := flashcards.LoadDueMap(ctx.Ctx, ctx.Pool, platformOn, courseOn, ctx.InstanceID, ctx.PrincipalID, cfg)
	deckStatus := flashcards.ComputeDeckStatus(cfg, st, due, srsOn)

	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"state":      flashcardsStateView(st),
		"srsEnabled": srsOn,
		"status":     deckStatus,
		"rating":     string(in.Rating),
		"cardId":     cardID,
		"side":       side,
	}
	if nextDue != nil {
		result["nextDueAt"] = *nextDue
	}
	if srsErr != "" {
		result["srsError"] = srsErr
	}
	if summary != nil {
		result["sessionComplete"] = true
		result["summary"] = summary
		result["current"] = nil
	} else {
		result["sessionComplete"] = false
		result["current"] = flashcardsCurrentCard(cfg, st)
	}

	status := StatusInProgress
	if st.FirstPassCompletedAt != "" {
		status = StatusCompleted
	}

	return &ActionResult{
		Result:     result,
		StatePatch: patch,
		Status:     status,
	}, nil
}

func handleFlashcardsEndSession(ctx ActionContext) (*ActionResult, error) {
	cfg := flashcards.ParseConfig(ctx.ConfigJSON)
	st := flashcards.ParseState(ctx.StateJSON)
	now := flashcards.NowRFC3339()
	rec := flashcards.EndActiveSession(&st, now)
	platformOn := flashcardsPlatformSRS(ctx)
	courseOn := flashcardsCourseSRS(ctx)
	srsOn := flashcards.SRSActive(platformOn, courseOn)
	due, _ := flashcards.LoadDueMap(ctx.Ctx, ctx.Pool, platformOn, courseOn, ctx.InstanceID, ctx.PrincipalID, cfg)
	deckStatus := flashcards.ComputeDeckStatus(cfg, st, due, srsOn)
	patch, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	ObserveFlashcardsSession("ended")
	status := StatusInProgress
	if st.FirstPassCompletedAt != "" {
		status = StatusCompleted
	}
	return &ActionResult{
		Result: map[string]any{
			"summary": map[string]any{
				"reviewed": rec.Reviewed,
				"endedAt":  rec.EndedAt,
			},
			"srsEnabled": srsOn,
			"status":     deckStatus,
			"state":      flashcardsStateView(st),
		},
		StatePatch: patch,
		Status:     status,
	}, nil
}

func handleFlashcardsStatus(ctx ActionContext) (*ActionResult, error) {
	cfg := flashcards.ParseConfig(ctx.ConfigJSON)
	st := flashcards.ParseState(ctx.StateJSON)
	platformOn := flashcardsPlatformSRS(ctx)
	courseOn := flashcardsCourseSRS(ctx)
	srsOn := flashcards.SRSActive(platformOn, courseOn)
	due, _ := flashcards.LoadDueMap(ctx.Ctx, ctx.Pool, platformOn, courseOn, ctx.InstanceID, ctx.PrincipalID, cfg)
	deckStatus := flashcards.ComputeDeckStatus(cfg, st, due, srsOn)
	return &ActionResult{
		Result: map[string]any{
			"srsEnabled": srsOn,
			"status":     deckStatus,
			"state":      flashcardsStateView(st),
			"current":    flashcardsCurrentCard(cfg, st),
		},
	}, nil
}

func flashcardsStateView(st flashcards.State) map[string]any {
	out := map[string]any{"v": st.V}
	if len(st.Cards) > 0 {
		out["cards"] = st.Cards
	}
	if len(st.Sessions) > 0 {
		out["sessions"] = st.Sessions
	}
	if st.ActiveSession != nil {
		out["activeSession"] = st.ActiveSession
	}
	if st.FirstPassCompletedAt != "" {
		out["firstPassCompletedAt"] = st.FirstPassCompletedAt
	}
	return out
}

func flashcardsCurrentCard(cfg flashcards.Config, st flashcards.State) map[string]any {
	if st.ActiveSession == nil {
		return nil
	}
	if st.ActiveSession.Index < 0 || st.ActiveSession.Index >= len(st.ActiveSession.Queue) {
		return nil
	}
	item := st.ActiveSession.Queue[st.ActiveSession.Index]
	card := flashcards.FindCard(cfg, item.CardID)
	if card == nil {
		return nil
	}
	prompt, answer := card.Front, card.Back
	promptLang, answerLang := card.FrontLang, card.BackLang
	if item.Side == flashcards.SideReverse {
		prompt, answer = card.Back, card.Front
		promptLang, answerLang = card.BackLang, card.FrontLang
	}
	out := map[string]any{
		"cardId":     card.ID,
		"side":       item.Side,
		"prompt":     prompt,
		"answer":     answer,
		"index":      st.ActiveSession.Index,
		"total":      len(st.ActiveSession.Queue),
		"revealed":   st.ActiveSession.Revealed,
		"hint":       card.Hint,
		"imageUrl":   card.ImageURL,
		"imageAlt":   card.ImageAlt,
		"promptLang": promptLang,
		"answerLang": answerLang,
	}
	return out
}

// ClearFlashcardsSchedulingForReset clears SRS scheduling for a flashcards instance (CT.23 FR-12).
func ClearFlashcardsSchedulingForReset(ctx context.Context, pool *pgxpool.Pool, courseID, instanceID, userID uuid.UUID) error {
	if pool == nil {
		return nil
	}
	inst, err := ctrepo.GetInstance(ctx, pool, courseID, instanceID)
	if err != nil || inst == nil {
		return err
	}
	cfg := flashcards.ParseConfig(inst.ConfigJSON)
	return flashcards.ClearSchedulingForDeck(ctx, pool, instanceID, userID, cfg)
}

var (
	flashcardsMetricsOnce sync.Once

	flashcardsCardReviewsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_card_reviews_total",
		Help:      "Flashcards self-ratings by rating (CT.23).",
	}, []string{"rating"})

	flashcardsSessionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_flashcards_sessions_total",
		Help:      "Flashcards session lifecycle outcomes (CT.23).",
	}, []string{"outcome"})

	flashcardsSRSSubmitTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "lextures",
		Name:      "content_tool_flashcards_srs_submit_total",
		Help:      "Flashcards SRS submission outcomes (CT.23).",
	}, []string{"outcome"})
)

func registerFlashcardsMetrics() {
	flashcardsMetricsOnce.Do(func() {
		prometheus.MustRegister(
			flashcardsCardReviewsTotal,
			flashcardsSessionsTotal,
			flashcardsSRSSubmitTotal,
		)
		flashcardsCardReviewsTotal.WithLabelValues("_reserved").Add(0)
		flashcardsSessionsTotal.WithLabelValues("_reserved").Add(0)
		flashcardsSRSSubmitTotal.WithLabelValues("_reserved").Add(0)
	})
}

// ObserveFlashcardsCardReview increments lextures_content_tool_card_reviews_total{rating}.
func ObserveFlashcardsCardReview(rating string) {
	registerFlashcardsMetrics()
	if rating == "" {
		rating = "_unknown"
	}
	flashcardsCardReviewsTotal.WithLabelValues(rating).Inc()
}

// ObserveFlashcardsSession records session lifecycle.
func ObserveFlashcardsSession(outcome string) {
	registerFlashcardsMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	flashcardsSessionsTotal.WithLabelValues(outcome).Inc()
}

// ObserveFlashcardsSRSSubmit records SRS write outcomes.
func ObserveFlashcardsSRSSubmit(outcome string) {
	registerFlashcardsMetrics()
	if outcome == "" {
		outcome = "_unknown"
	}
	flashcardsSRSSubmitTotal.WithLabelValues(outcome).Inc()
}
