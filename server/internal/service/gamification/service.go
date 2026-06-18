package gamification

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Row is the aggregate gamification state for one user.
type Row struct {
	UserID             uuid.UUID
	XPTotal            int
	CurrentStreak      int
	LongestStreak      int
	LastActivityDate   *time.Time
	StreakFreezes      int
	FreezeCoverDate    *time.Time
	LeaderboardVisible bool
}

// Badge is an awarded milestone badge.
type Badge struct {
	BadgeType string    `json:"badgeType"`
	AwardedAt time.Time `json:"awardedAt"`
}

// Profile is the GET /me/gamification payload.
type Profile struct {
	XPTotal            int     `json:"xpTotal"`
	Level              int     `json:"level"`
	XPToNextLevel      int     `json:"xpToNextLevel"`
	LevelProgressPct   int     `json:"levelProgressPct"`
	CurrentStreak      int     `json:"currentStreak"`
	LongestStreak      int     `json:"longestStreak"`
	StreakFreezes      int     `json:"streakFreezes"`
	StreakAtRisk       bool    `json:"streakAtRisk"`
	StreakHoursLeft    float64 `json:"streakHoursLeft,omitempty"`
	StreakEnded        bool    `json:"streakEnded,omitempty"`
	LeaderboardVisible bool    `json:"leaderboardVisible"`
	Badges             []Badge `json:"badges"`
	RecentBadges       []Badge `json:"recentBadges"`
}

// LeaderboardEntry is one row on a course leaderboard.
type LeaderboardEntry struct {
	Rank          int    `json:"rank"`
	UserID        string `json:"userId"`
	DisplayName   string `json:"displayName"`
	XPEarned      int    `json:"xpEarned"`
	IsCurrentUser bool   `json:"isCurrentUser,omitempty"`
}

// LeaderboardResponse is GET /courses/:id/leaderboard.
type LeaderboardResponse struct {
	TopEntries  []LeaderboardEntry `json:"topEntries"`
	CurrentUser *LeaderboardEntry  `json:"currentUser,omitempty"`
}

// AwardResult is returned after a successful XP award.
type AwardResult struct {
	AwardedXP     int
	NewXPTotal    int
	NewBadges     []string
	StreakChanged bool
}

var (
	ErrNoFreezes      = errors.New("gamification: no streak freezes available")
	ErrNoActiveStreak = errors.New("gamification: no active streak to protect")
)

func ensureRow(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
INSERT INTO gamification.user_gamification (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO NOTHING
`, userID)
	return err
}

// LoadProfile loads gamification stats, reconciling streak on read.
func LoadProfile(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, now time.Time, timezone *string) (Profile, error) {
	if err := ensureRow(ctx, pool, userID); err != nil {
		return Profile{}, err
	}
	row, err := loadRow(ctx, pool, userID)
	if err != nil {
		return Profile{}, err
	}
	today := UserLocalDate(now, timezone)
	newStreak, streakEnded, freezeConsumed := ReconcileStreakOnLogin(row.LastActivityDate, row.CurrentStreak, row.FreezeCoverDate, today)
	if newStreak != row.CurrentStreak || freezeConsumed {
		if streakEnded {
			RecordStreakReset()
		}
		if err := persistStreakReconcile(ctx, pool, userID, newStreak, freezeConsumed); err != nil {
			return Profile{}, err
		}
		row.CurrentStreak = newStreak
	}
	badges, err := listBadges(ctx, pool, userID)
	if err != nil {
		return Profile{}, err
	}
	level := LevelFromXP(row.XPTotal)
	nextXP := XPForNextLevel(level)
	prevXP := 0
	if level > 0 {
		prevXP = XPForNextLevel(level - 1)
	}
	span := nextXP - prevXP
	progress := 0
	if span > 0 {
		progress = int(float64(row.XPTotal-prevXP) / float64(span) * 100)
		if progress > 100 {
			progress = 100
		}
	}
	atRisk, hoursLeft := StreakAtRisk(row.CurrentStreak, row.LastActivityDate, today, now, timezone)
	recent := badges
	if len(recent) > 5 {
		recent = recent[:5]
	}
	return Profile{
		XPTotal:            row.XPTotal,
		Level:              level,
		XPToNextLevel:      nextXP - row.XPTotal,
		LevelProgressPct:   progress,
		CurrentStreak:      row.CurrentStreak,
		LongestStreak:      row.LongestStreak,
		StreakFreezes:      row.StreakFreezes,
		StreakAtRisk:       atRisk,
		StreakHoursLeft:    hoursLeft,
		StreakEnded:        streakEnded,
		LeaderboardVisible: row.LeaderboardVisible,
		Badges:             badges,
		RecentBadges:       recent,
	}, nil
}

func loadRow(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (Row, error) {
	var row Row
	var last, freeze sql.NullTime
	err := pool.QueryRow(ctx, `
SELECT user_id, xp_total, current_streak, longest_streak, last_activity_date,
       streak_freezes, freeze_cover_date, leaderboard_visible
FROM gamification.user_gamification
WHERE user_id = $1
`, userID).Scan(
		&row.UserID, &row.XPTotal, &row.CurrentStreak, &row.LongestStreak, &last,
		&row.StreakFreezes, &freeze, &row.LeaderboardVisible,
	)
	if err != nil {
		return Row{}, err
	}
	if last.Valid {
		t := last.Time
		row.LastActivityDate = &t
	}
	if freeze.Valid {
		t := freeze.Time
		row.FreezeCoverDate = &t
	}
	return row, nil
}

func loadRowTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (Row, error) {
	var row Row
	var last, freeze sql.NullTime
	err := tx.QueryRow(ctx, `
SELECT user_id, xp_total, current_streak, longest_streak, last_activity_date,
       streak_freezes, freeze_cover_date, leaderboard_visible
FROM gamification.user_gamification
WHERE user_id = $1
`, userID).Scan(
		&row.UserID, &row.XPTotal, &row.CurrentStreak, &row.LongestStreak, &last,
		&row.StreakFreezes, &freeze, &row.LeaderboardVisible,
	)
	if err != nil {
		return Row{}, err
	}
	if last.Valid {
		t := last.Time
		row.LastActivityDate = &t
	}
	if freeze.Valid {
		t := freeze.Time
		row.FreezeCoverDate = &t
	}
	return row, nil
}

func listBadges(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Badge, error) {
	rows, err := pool.Query(ctx, `
SELECT badge_type, awarded_at
FROM gamification.user_badges
WHERE user_id = $1
ORDER BY awarded_at DESC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Badge
	for rows.Next() {
		var b Badge
		if err := rows.Scan(&b.BadgeType, &b.AwardedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	if out == nil {
		out = []Badge{}
	}
	return out, rows.Err()
}

func persistStreakReconcile(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, streak int, clearFreeze bool) error {
	if clearFreeze {
		_, err := pool.Exec(ctx, `
UPDATE gamification.user_gamification
SET current_streak = $2, freeze_cover_date = NULL, updated_at = NOW()
WHERE user_id = $1
`, userID, streak)
		return err
	}
	_, err := pool.Exec(ctx, `
UPDATE gamification.user_gamification
SET current_streak = $2, updated_at = NOW()
WHERE user_id = $1
`, userID, streak)
	return err
}

// AwardXP records an idempotent XP event and updates streak/badges.
func AwardXP(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	activityType string,
	sourceID *uuid.UUID,
	courseID *uuid.UUID,
	idempotencyKey string,
	now time.Time,
	timezone *string,
) (AwardResult, error) {
	xp := XPAward(activityType)
	if xp <= 0 {
		return AwardResult{}, nil
	}
	if err := ensureRow(ctx, pool, userID); err != nil {
		return AwardResult{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AwardResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
INSERT INTO gamification.xp_events (user_id, course_id, activity_type, source_id, xp_awarded, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (idempotency_key) DO NOTHING
`, userID, courseID, activityType, sourceID, xp, idempotencyKey)
	if err != nil {
		return AwardResult{}, err
	}
	if tag.RowsAffected() == 0 {
		return AwardResult{}, nil
	}
	RecordXPAwarded(activityType, xp)

	var newTotal int
	err = tx.QueryRow(ctx, `
UPDATE gamification.user_gamification
SET xp_total = xp_total + $2, updated_at = NOW()
WHERE user_id = $1
RETURNING xp_total
`, userID, xp).Scan(&newTotal)
	if err != nil {
		return AwardResult{}, err
	}

	row, err := loadRowTx(ctx, tx, userID)
	if err != nil {
		return AwardResult{}, err
	}
	today := UserLocalDate(now, timezone)
	newStreak, _, freezeUsed := ComputeStreakAfterActivity(
		row.LastActivityDate, row.CurrentStreak, row.LongestStreak, row.FreezeCoverDate, today,
	)
	freezes := row.StreakFreezes
	if freezeUsed && freezes > 0 {
		freezes--
	}
	for _, m := range StreakFreezeMilestones {
		if newStreak == m && row.CurrentStreak < m {
			freezes++
		}
	}
	_, err = tx.Exec(ctx, `
UPDATE gamification.user_gamification
SET current_streak = $2,
    longest_streak = GREATEST(longest_streak, $2),
    last_activity_date = $3,
    streak_freezes = $4,
    freeze_cover_date = CASE WHEN $5 THEN NULL ELSE freeze_cover_date END,
    updated_at = NOW()
WHERE user_id = $1
`, userID, newStreak, today, freezes, freezeUsed)
	if err != nil {
		return AwardResult{}, err
	}

	newBadges, err := checkAndAwardBadgesTx(ctx, tx, userID, newStreak, newTotal, activityType)
	if err != nil {
		return AwardResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AwardResult{}, err
	}
	return AwardResult{
		AwardedXP:     xp,
		NewXPTotal:    newTotal,
		NewBadges:     newBadges,
		StreakChanged: newStreak != row.CurrentStreak || freezeUsed,
	}, nil
}

func checkAndAwardBadgesTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, streak, xpTotal int, activityType string) ([]string, error) {
	candidates := badgeCandidates(streak, xpTotal, activityType)
	var awarded []string
	for _, b := range candidates {
		tag, err := tx.Exec(ctx, `
INSERT INTO gamification.user_badges (user_id, badge_type)
VALUES ($1, $2)
ON CONFLICT (user_id, badge_type) DO NOTHING
`, userID, b)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() > 0 {
			awarded = append(awarded, b)
		}
	}
	return awarded, nil
}

func badgeCandidates(streak, xpTotal int, activityType string) []string {
	var out []string
	if streak >= 7 {
		out = append(out, BadgeStreak7)
	}
	if streak >= 30 {
		out = append(out, BadgeStreak30)
	}
	if xpTotal >= 100 {
		out = append(out, BadgeXP100)
	}
	if xpTotal >= 1000 {
		out = append(out, BadgeXP1000)
	}
	if activityType == ActivityCourseCompleted {
		out = append(out, BadgeFirstCourseComplete)
	}
	return out
}

// SpendStreakFreeze applies a freeze item to protect the streak through today.
func SpendStreakFreeze(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, now time.Time, timezone *string) error {
	if err := ensureRow(ctx, pool, userID); err != nil {
		return err
	}
	row, err := loadRow(ctx, pool, userID)
	if err != nil {
		return err
	}
	if row.StreakFreezes <= 0 {
		return ErrNoFreezes
	}
	if row.CurrentStreak <= 0 {
		return ErrNoActiveStreak
	}
	today := UserLocalDate(now, timezone)
	tag, err := pool.Exec(ctx, `
UPDATE gamification.user_gamification
SET streak_freezes = streak_freezes - 1,
    freeze_cover_date = $2,
    updated_at = NOW()
WHERE user_id = $1 AND streak_freezes > 0
`, userID, today)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoFreezes
	}
	return nil
}

// CourseGamificationEnabled reports whether a course participates in gamification.
func CourseGamificationEnabled(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (bool, error) {
	var enabled bool
	err := pool.QueryRow(ctx, `
SELECT gamification_enabled FROM course.courses WHERE id = $1
`, courseID).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return enabled, nil
}

// LoadCourseLeaderboard returns top 10 learners by course XP plus the viewer's rank.
func LoadCourseLeaderboard(ctx context.Context, pool *pgxpool.Pool, courseID, viewerID uuid.UUID) (LeaderboardResponse, error) {
	rows, err := pool.Query(ctx, `
SELECT u.id::text,
       COALESCE(NULLIF(TRIM(u.display_name), ''), split_part(u.email, '@', 1)) AS display_name,
       COALESCE(SUM(x.xp_awarded), 0)::int AS xp
FROM gamification.xp_events x
INNER JOIN "user".users u ON u.id = x.user_id
INNER JOIN gamification.user_gamification g ON g.user_id = u.id AND g.leaderboard_visible = TRUE
WHERE x.course_id = $1
GROUP BY u.id, display_name
ORDER BY xp DESC, display_name ASC
LIMIT 10
`, courseID)
	if err != nil {
		return LeaderboardResponse{}, err
	}
	defer rows.Close()
	var top []LeaderboardEntry
	rank := 0
	for rows.Next() {
		rank++
		var e LeaderboardEntry
		var uid string
		if err := rows.Scan(&uid, &e.DisplayName, &e.XPEarned); err != nil {
			return LeaderboardResponse{}, err
		}
		e.Rank = rank
		e.UserID = uid
		if uid == viewerID.String() {
			e.IsCurrentUser = true
		}
		top = append(top, e)
	}
	if err := rows.Err(); err != nil {
		return LeaderboardResponse{}, err
	}
	if top == nil {
		top = []LeaderboardEntry{}
	}
	resp := LeaderboardResponse{TopEntries: top}
	for _, e := range top {
		if e.IsCurrentUser {
			return resp, nil
		}
	}
	var viewerRank int
	var viewerXP int
	var viewerName string
	err = pool.QueryRow(ctx, `
WITH ranked AS (
  SELECT u.id::text AS user_id,
         COALESCE(NULLIF(TRIM(u.display_name), ''), split_part(u.email, '@', 1)) AS display_name,
         COALESCE(SUM(x.xp_awarded), 0)::int AS xp,
         RANK() OVER (ORDER BY COALESCE(SUM(x.xp_awarded), 0) DESC, display_name ASC) AS rnk
  FROM gamification.xp_events x
  INNER JOIN "user".users u ON u.id = x.user_id
  INNER JOIN gamification.user_gamification g ON g.user_id = u.id AND g.leaderboard_visible = TRUE
  WHERE x.course_id = $1
  GROUP BY u.id, display_name
)
SELECT rnk, xp, display_name FROM ranked WHERE user_id = $2::text
`, courseID, viewerID.String()).Scan(&viewerRank, &viewerXP, &viewerName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return resp, nil
		}
		return LeaderboardResponse{}, err
	}
	resp.CurrentUser = &LeaderboardEntry{
		Rank:          viewerRank,
		UserID:        viewerID.String(),
		DisplayName:   viewerName,
		XPEarned:      viewerXP,
		IsCurrentUser: true,
	}
	return resp, nil
}
