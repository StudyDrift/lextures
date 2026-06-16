// Package seattime implements heartbeat-based seat-time tracking and CEU awards (plan 14.17).
package seattime

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	reposeattime "github.com/lextures/lextures/server/internal/repos/seattime"
)

const (
	// CountedMinuteGap is the minimum interval between counted seat-time minutes.
	CountedMinuteGap = 60 * time.Second
	// RapidHeartbeatGap flags anomaly when heartbeats arrive faster than this (4/min max).
	RapidHeartbeatGap = 15 * time.Second
	// MaxDailyMinutesPerCourse caps plausible daily seat time per course (8 hours).
	MaxDailyMinutesPerCourse = 8 * 60
)

// ErrNotEnrolled is returned when a heartbeat targets a course the user is not enrolled in.
var ErrNotEnrolled = errors.New("not enrolled in course")

// SessionState is the in-memory heartbeat state for one session.
type SessionState struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	ContentItemID   uuid.UUID
	CourseID        uuid.UUID
	SessionToken    string
	SessionStart    time.Time
	SessionEnd      time.Time
	MinutesActive   int
	AnomalyFlag     bool
	LastCountedAt   time.Time
	LastHeartbeatAt time.Time
	Dirty           bool
}

// HeartbeatResult is returned to API callers after processing a heartbeat.
type HeartbeatResult struct {
	MinutesActive int
	Counted       bool
	AnomalyFlag   bool
}

// ProgressSummary is learner CE progress for one course.
type ProgressSummary struct {
	TotalMinutes int
	RequiredHours float64
	CEUCredit     float64
	CEUEarned     float64
	ProgressPct   float64
	Awarded       bool
}

// ApplyHeartbeat updates session state for an incoming heartbeat.
func ApplyHeartbeat(state SessionState, now time.Time, dailyMinutesBefore int) (SessionState, bool) {
	if state.SessionStart.IsZero() {
		state.SessionStart = now
	}
	state.SessionEnd = now
	state.Dirty = true

	if !state.LastHeartbeatAt.IsZero() && now.Sub(state.LastHeartbeatAt) < RapidHeartbeatGap {
		state.AnomalyFlag = true
	}
	state.LastHeartbeatAt = now

	counted := false
	if state.LastCountedAt.IsZero() || now.Sub(state.LastCountedAt) >= CountedMinuteGap {
		if dailyMinutesBefore+state.MinutesActive < MaxDailyMinutesPerCourse {
			state.MinutesActive++
			state.LastCountedAt = now
			counted = true
		}
	}
	return state, counted
}

// ComputeProgress derives CEU progress from totals and configuration.
func ComputeProgress(totalMinutes int, cfg *reposeattime.CEUConfig, awarded bool) ProgressSummary {
	out := ProgressSummary{}
	if cfg == nil || !cfg.Enabled {
		return out
	}
	out.TotalMinutes = totalMinutes
	out.RequiredHours = cfg.RequiredHours
	out.CEUCredit = cfg.CEUCredit
	requiredMinutes := int(cfg.RequiredHours * 60)
	if requiredMinutes > 0 {
		pct := float64(totalMinutes) / float64(requiredMinutes) * 100
		if pct > 100 {
			pct = 100
		}
		out.ProgressPct = pct
	}
	if awarded {
		out.Awarded = true
		out.CEUEarned = cfg.CEUCredit
	} else if requiredMinutes > 0 {
		hours := float64(totalMinutes) / 60.0
		ratio := hours / cfg.RequiredHours
		if ratio > 1 {
			ratio = 1
		}
		out.CEUEarned = ratio * cfg.CEUCredit
	}
	return out
}

// Buffer batches session writes and flushes to the database periodically.
type Buffer struct {
	mu       sync.Mutex
	sessions map[string]SessionState
	pool     *pgxpool.Pool
}

func sessionKey(userID, contentItemID uuid.UUID, token string) string {
	return userID.String() + ":" + contentItemID.String() + ":" + token
}

// NewBuffer creates a seat-time write buffer.
func NewBuffer(pool *pgxpool.Pool) *Buffer {
	return &Buffer{
		sessions: make(map[string]SessionState),
		pool:     pool,
	}
}

// GlobalBuffer is the process-wide seat-time buffer flushed by background jobs.
var GlobalBuffer *Buffer

// InitGlobalBuffer sets up the singleton buffer for the process.
func InitGlobalBuffer(pool *pgxpool.Pool) {
	GlobalBuffer = NewBuffer(pool)
}

// ProcessHeartbeat validates and records a heartbeat, returning updated session totals.
func (b *Buffer) ProcessHeartbeat(ctx context.Context, userID, contentItemID uuid.UUID, sessionToken string, now time.Time) (HeartbeatResult, error) {
	if b == nil || b.pool == nil {
		return HeartbeatResult{}, nil
	}
	meta, err := reposeattime.ResolveContentItemCourse(ctx, b.pool, contentItemID)
	if err != nil || meta == nil {
		return HeartbeatResult{}, err
	}
	enrolled, err := reposeattime.UserEnrolledInCourse(ctx, b.pool, userID, meta.CourseID)
	if err != nil {
		return HeartbeatResult{}, err
	}
	if !enrolled {
		return HeartbeatResult{}, ErrNotEnrolled
	}

	key := sessionKey(userID, contentItemID, sessionToken)
	b.mu.Lock()
	defer b.mu.Unlock()

	state, ok := b.sessions[key]
	if !ok {
		existing, err := reposeattime.GetSession(ctx, b.pool, userID, contentItemID, sessionToken)
		if err != nil {
			return HeartbeatResult{}, err
		}
		if existing != nil {
			state = SessionState{
				ID:            existing.ID,
				UserID:        existing.UserID,
				ContentItemID: existing.ContentItemID,
				CourseID:      existing.CourseID,
				SessionToken:  existing.SessionToken,
				SessionStart:  existing.SessionStart,
				MinutesActive: existing.MinutesActive,
				AnomalyFlag:   existing.AnomalyFlag,
				LastCountedAt: existing.SessionStart.Add(time.Duration(existing.MinutesActive) * time.Minute),
			}
			if existing.SessionEnd != nil {
				state.LastHeartbeatAt = *existing.SessionEnd
			}
		} else {
			state = SessionState{
				ID:            uuid.New(),
				UserID:        userID,
				ContentItemID: contentItemID,
				CourseID:      meta.CourseID,
				SessionToken:  sessionToken,
			}
		}
	}

	dailyMinutes, err := reposeattime.DailyMinutesForCourse(ctx, b.pool, userID, meta.CourseID, now)
	if err != nil {
		return HeartbeatResult{}, err
	}

	var counted bool
	state, counted = ApplyHeartbeat(state, now, dailyMinutes)
	b.sessions[key] = state

	return HeartbeatResult{
		MinutesActive: state.MinutesActive,
		Counted:       counted,
		AnomalyFlag:   state.AnomalyFlag,
	}, nil
}

// Flush persists dirty sessions to the database.
func (b *Buffer) Flush(ctx context.Context) error {
	if b == nil || b.pool == nil {
		return nil
	}
	b.mu.Lock()
	dirty := make([]SessionState, 0, len(b.sessions))
	for _, s := range b.sessions {
		if s.Dirty {
			dirty = append(dirty, s)
			s.Dirty = false
			b.sessions[sessionKey(s.UserID, s.ContentItemID, s.SessionToken)] = s
		}
	}
	b.mu.Unlock()

	for _, s := range dirty {
		end := s.SessionEnd
		if err := reposeattime.UpsertSession(ctx, b.pool, reposeattime.Session{
			ID:            s.ID,
			UserID:        s.UserID,
			ContentItemID: s.ContentItemID,
			CourseID:      s.CourseID,
			SessionToken:  s.SessionToken,
			SessionStart:  s.SessionStart,
			SessionEnd:    &end,
			MinutesActive: s.MinutesActive,
			AnomalyFlag:   s.AnomalyFlag,
		}); err != nil {
			return err
		}
	}
	return nil
}

// MaybeIssueCEUAward checks thresholds and creates an award when eligible.
// The second return value is true only when a new award row was inserted.
func MaybeIssueCEUAward(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, now time.Time) (*reposeattime.CEUAward, bool, error) {
	cfg, err := reposeattime.GetCEUConfig(ctx, pool, courseID)
	if err != nil || cfg == nil || !cfg.Enabled {
		return nil, false, err
	}
	existing, err := reposeattime.GetCEUAward(ctx, pool, userID, courseID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return existing, false, nil
	}
	total, err := reposeattime.TotalMinutesForCourse(ctx, pool, userID, courseID)
	if err != nil {
		return nil, false, err
	}
	requiredMinutes := int(cfg.RequiredHours * 60)
	if total < requiredMinutes {
		return nil, false, nil
	}
	award, err := reposeattime.CreateCEUAward(ctx, pool, userID, courseID, cfg.CEUCredit, cfg.RequiredHours, now)
	if err != nil {
		return nil, false, err
	}
	return award, true, nil
}

// LoadProgress loads CE progress for a learner in one course.
func LoadProgress(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (ProgressSummary, error) {
	cfg, err := reposeattime.GetCEUConfig(ctx, pool, courseID)
	if err != nil {
		return ProgressSummary{}, err
	}
	total, err := reposeattime.TotalMinutesForCourse(ctx, pool, userID, courseID)
	if err != nil {
		return ProgressSummary{}, err
	}
	award, err := reposeattime.GetCEUAward(ctx, pool, userID, courseID)
	if err != nil {
		return ProgressSummary{}, err
	}
	return ComputeProgress(total, cfg, award != nil), nil
}
