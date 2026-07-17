package quizgame

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageDaily is one precomputed day bucket for Live Quiz analytics.
type UsageDaily struct {
	Day             time.Time `json:"day"`
	OrgID           string    `json:"orgId"`
	CourseID        string    `json:"courseId"`
	Games           int       `json:"games"`
	Players         int       `json:"players"`
	Answers         int       `json:"answers"`
	GuestPlayers    int       `json:"guestPlayers"`
	EnrolledPlayers int       `json:"enrolledPlayers"`
	AICostCents     int       `json:"aiCostCents"`
}

// AnalyticsSummary is the admin Live Quizzes analytics payload (IQ.11 FR-3).
type AnalyticsSummary struct {
	Games              int                    `json:"games"`
	GamesByMode        map[string]int         `json:"gamesByMode"`
	UniqueHosts        int                    `json:"uniqueHosts"`
	UniquePlayers      int                    `json:"uniquePlayers"`
	AnswersSubmitted   int                    `json:"answersSubmitted"`
	AvgParticipation   float64                `json:"avgParticipation"`
	GuestPlayers       int                    `json:"guestPlayers"`
	EnrolledPlayers    int                    `json:"enrolledPlayers"`
	AICostCents        int                    `json:"aiCostCents"`
	CoursesUsing       int                    `json:"coursesUsing"`
	PendingReviewCount int                    `json:"pendingReviewCount"`
	LiveGamesNow       int                    `json:"liveGamesNow"`
	Daily              []UsageDaily           `json:"daily"`
	From               string                 `json:"from"`
	To                 string                 `json:"to"`
	OrgID              string                 `json:"orgId,omitempty"`
}

// LiveGameOps is one live/abandoned game for the admin ops view.
type LiveGameOps struct {
	ID        string     `json:"id"`
	CourseCode string    `json:"courseCode"`
	JoinCode  string     `json:"joinCode,omitempty"`
	Status    string     `json:"status"`
	Mode      string     `json:"mode"`
	HostID    *string    `json:"hostId,omitempty"`
	Players   int        `json:"players"`
	CreatedAt time.Time  `json:"createdAt"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
}

// GetAnalytics returns adoption/usage/AI cost aggregates for an org and time range.
func GetAnalytics(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, from, to time.Time) (AnalyticsSummary, error) {
	if to.Before(from) {
		from, to = to, from
	}
	sum := AnalyticsSummary{
		GamesByMode: map[string]int{},
		Daily:       []UsageDaily{},
		From:        from.UTC().Format("2006-01-02"),
		To:          to.UTC().Format("2006-01-02"),
		OrgID:       orgID.String(),
	}

	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int,
		       COUNT(DISTINCT s.host_id)::int,
		       COUNT(DISTINCT s.course_id)::int
		FROM quizgame.sessions s
		INNER JOIN course.courses c ON c.id = s.course_id
		WHERE c.org_id = $1
		  AND s.created_at >= $2 AND s.created_at < $3
	`, orgID, from, to).Scan(&sum.Games, &sum.UniqueHosts, &sum.CoursesUsing)
	if err != nil {
		return sum, err
	}

	rows, err := pool.Query(ctx, `
		SELECT COALESCE(s.mode, 'live_classic'), COUNT(*)::int
		FROM quizgame.sessions s
		INNER JOIN course.courses c ON c.id = s.course_id
		WHERE c.org_id = $1
		  AND s.created_at >= $2 AND s.created_at < $3
		GROUP BY 1
	`, orgID, from, to)
	if err != nil {
		return sum, err
	}
	for rows.Next() {
		var mode string
		var n int
		if err := rows.Scan(&mode, &n); err != nil {
			rows.Close()
			return sum, err
		}
		sum.GamesByMode[mode] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return sum, err
	}

	err = pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT p.id)::int,
		       COUNT(DISTINCT p.id) FILTER (WHERE p.user_id IS NULL)::int,
		       COUNT(DISTINCT p.id) FILTER (WHERE p.user_id IS NOT NULL)::int
		FROM quizgame.session_players p
		INNER JOIN quizgame.sessions s ON s.id = p.session_id
		INNER JOIN course.courses c ON c.id = s.course_id
		WHERE c.org_id = $1
		  AND p.joined_at >= $2 AND p.joined_at < $3
		  AND p.removed_at IS NULL
	`, orgID, from, to).Scan(&sum.UniquePlayers, &sum.GuestPlayers, &sum.EnrolledPlayers)
	if err != nil {
		return sum, err
	}

	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM quizgame.session_responses r
		INNER JOIN quizgame.sessions s ON s.id = r.session_id
		INNER JOIN course.courses c ON c.id = s.course_id
		WHERE c.org_id = $1
		  AND r.answered_at >= $2 AND r.answered_at < $3
	`, orgID, from, to).Scan(&sum.AnswersSubmitted)
	if err != nil {
		return sum, err
	}

	if sum.Games > 0 {
		sum.AvgParticipation = float64(sum.UniquePlayers) / float64(sum.Games)
	}

	// AI cost from aiusage for live quiz kit generation (USD → cents).
	var costUSD float64
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(u.cost_usd), 0)
		FROM analytics.ai_usage_log u
		INNER JOIN course.courses c ON c.id = u.course_id
		WHERE c.org_id = $1
		  AND u.feature = 'live_quiz_kit_generation'
		  AND u.created_at >= $2 AND u.created_at < $3
	`, orgID, from, to).Scan(&costUSD)
	sum.AICostCents = int(costUSD*100 + 0.5)

	pending, _ := CountPendingReviews(ctx, pool)
	sum.PendingReviewCount = pending
	live, _ := CountConcurrentLiveGames(ctx, pool, &orgID)
	sum.LiveGamesNow = live

	drows, err := pool.Query(ctx, `
		SELECT day, org_id, course_id, games, players, answers, guest_players, enrolled_players, ai_cost_cents
		FROM quizgame.usage_daily
		WHERE org_id = $1 AND day >= $2::date AND day <= $3::date
		ORDER BY day ASC
	`, orgID, from, to)
	if err != nil {
		return sum, err
	}
	defer drows.Close()
	for drows.Next() {
		var u UsageDaily
		var oid, cid uuid.UUID
		if err := drows.Scan(&u.Day, &oid, &cid, &u.Games, &u.Players, &u.Answers,
			&u.GuestPlayers, &u.EnrolledPlayers, &u.AICostCents); err != nil {
			return sum, err
		}
		u.OrgID = oid.String()
		u.CourseID = cid.String()
		sum.Daily = append(sum.Daily, u)
	}
	return sum, drows.Err()
}

// RefreshUsageDaily upserts daily rollups for the given day (UTC). When orgID is nil, all orgs.
func RefreshUsageDaily(ctx context.Context, pool *pgxpool.Pool, orgID *uuid.UUID, day time.Time) (int, error) {
	day = time.Date(day.UTC().Year(), day.UTC().Month(), day.UTC().Day(), 0, 0, 0, 0, time.UTC)
	next := day.AddDate(0, 0, 1)

	q := `
		INSERT INTO quizgame.usage_daily (
			day, org_id, course_id, games, players, answers, guest_players, enrolled_players, ai_cost_cents
		)
		SELECT
			$1::date,
			c.org_id,
			c.id,
			COALESCE(g.games, 0),
			COALESCE(p.players, 0),
			COALESCE(a.answers, 0),
			COALESCE(p.guest_players, 0),
			COALESCE(p.enrolled_players, 0),
			COALESCE(ai.ai_cost_cents, 0)
		FROM course.courses c
		LEFT JOIN (
			SELECT course_id, COUNT(*)::int AS games
			FROM quizgame.sessions
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY course_id
		) g ON g.course_id = c.id
		LEFT JOIN (
			SELECT s.course_id,
			       COUNT(DISTINCT sp.id)::int AS players,
			       COUNT(DISTINCT sp.id) FILTER (WHERE sp.user_id IS NULL)::int AS guest_players,
			       COUNT(DISTINCT sp.id) FILTER (WHERE sp.user_id IS NOT NULL)::int AS enrolled_players
			FROM quizgame.session_players sp
			INNER JOIN quizgame.sessions s ON s.id = sp.session_id
			WHERE sp.joined_at >= $1 AND sp.joined_at < $2 AND sp.removed_at IS NULL
			GROUP BY s.course_id
		) p ON p.course_id = c.id
		LEFT JOIN (
			SELECT s.course_id, COUNT(*)::int AS answers
			FROM quizgame.session_responses r
			INNER JOIN quizgame.sessions s ON s.id = r.session_id
			WHERE r.answered_at >= $1 AND r.answered_at < $2
			GROUP BY s.course_id
		) a ON a.course_id = c.id
		LEFT JOIN (
			SELECT course_id, (COALESCE(SUM(cost_usd), 0) * 100)::int AS ai_cost_cents
			FROM analytics.ai_usage_log
			WHERE feature = 'live_quiz_kit_generation'
			  AND created_at >= $1 AND created_at < $2
			GROUP BY course_id
		) ai ON ai.course_id = c.id
		WHERE (g.games IS NOT NULL OR p.players IS NOT NULL OR a.answers IS NOT NULL OR ai.ai_cost_cents IS NOT NULL)
	`
	args := []any{day, next}
	if orgID != nil {
		q += ` AND c.org_id = $3`
		args = append(args, *orgID)
	}
	q += `
		ON CONFLICT (day, org_id, course_id) DO UPDATE SET
			games = EXCLUDED.games,
			players = EXCLUDED.players,
			answers = EXCLUDED.answers,
			guest_players = EXCLUDED.guest_players,
			enrolled_players = EXCLUDED.enrolled_players,
			ai_cost_cents = EXCLUDED.ai_cost_cents
	`
	tag, err := pool.Exec(ctx, q, args...)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ListLiveGames returns active (and optionally recently ended) games for ops.
func ListLiveGames(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, limit int) ([]LiveGameOps, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := pool.Query(ctx, `
		SELECT s.id, c.course_code, COALESCE(s.join_code, ''), s.status, s.mode, s.host_id,
		       (SELECT COUNT(*)::int FROM quizgame.session_players p
		         WHERE p.session_id = s.id AND p.removed_at IS NULL),
		       s.created_at, s.started_at
		FROM quizgame.sessions s
		INNER JOIN course.courses c ON c.id = s.course_id
		WHERE c.org_id = $1
		  AND s.status IN ('lobby', 'running', 'paused')
		ORDER BY s.created_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LiveGameOps, 0)
	for rows.Next() {
		var g LiveGameOps
		var id uuid.UUID
		var host *uuid.UUID
		if err := rows.Scan(&id, &g.CourseCode, &g.JoinCode, &g.Status, &g.Mode, &host,
			&g.Players, &g.CreatedAt, &g.StartedAt); err != nil {
			return nil, err
		}
		g.ID = id.String()
		if host != nil {
			s := host.String()
			g.HostID = &s
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
