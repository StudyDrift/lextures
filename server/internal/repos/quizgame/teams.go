package quizgame

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
)

var (
	ErrTeamNotFound      = errors.New("quizgame: team not found")
	ErrModeImmutable     = errors.New("quizgame: mode config immutable after join")
	ErrWrongMode         = errors.New("quizgame: wrong session mode")
	ErrOneDeviceAnswered = errors.New("quizgame: team already answered on shared device")
)

// Team is one quizgame.teams row.
type Team struct {
	ID         string
	SessionID  string
	Name       string
	Color      *string
	TotalScore int
}

// CreateTeams creates named teams for a team-mode lobby session.
func CreateTeams(ctx context.Context, pool *pgxpool.Pool, sessionID string, names []string, colors []string) ([]Team, error) {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	if engine.NormalizeMode(sess.Mode) != engine.ModeTeam {
		return nil, ErrWrongMode
	}
	if sess.Status != "lobby" {
		n, _ := CountActivePlayers(ctx, pool, sessionID)
		if n > 0 {
			return nil, ErrModeImmutable
		}
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	if len(names) == 0 {
		ms := engine.ParseModeSettings(sess.Settings)
		tc := engine.NormalizeTeamConfig(ms.Team)
		names = engine.DefaultTeamNames(tc.TeamCount)
		colors = engine.DefaultTeamColors(tc.TeamCount)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, _ = tx.Exec(ctx, `DELETE FROM quizgame.teams WHERE session_id = $1`, sid)
	out := make([]Team, 0, len(names))
	for i, name := range names {
		var color any
		if i < len(colors) && colors[i] != "" {
			color = colors[i]
		}
		var id uuid.UUID
		err := tx.QueryRow(ctx, `
			INSERT INTO quizgame.teams (session_id, name, color)
			VALUES ($1, $2, $3) RETURNING id`, sid, name, color).Scan(&id)
		if err != nil {
			return nil, err
		}
		t := Team{ID: id.String(), SessionID: sessionID, Name: name, TotalScore: 0}
		if c, ok := color.(string); ok {
			t.Color = &c
		}
		out = append(out, t)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

// ListTeams returns teams for a session.
func ListTeams(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]Team, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	rows, err := pool.Query(ctx, `
		SELECT id, session_id, name, color, total_score
		FROM quizgame.teams WHERE session_id = $1 ORDER BY name ASC`, sid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Team
	for rows.Next() {
		var t Team
		var id, sess uuid.UUID
		var color *string
		if err := rows.Scan(&id, &sess, &t.Name, &color, &t.TotalScore); err != nil {
			return nil, err
		}
		t.ID = id.String()
		t.SessionID = sess.String()
		t.Color = color
		out = append(out, t)
	}
	return out, rows.Err()
}

// AssignPlayerToTeam sets session_players.team_id.
func AssignPlayerToTeam(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID, teamID string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return ErrPlayerNotFound
	}
	tid, err := uuid.Parse(teamID)
	if err != nil {
		return ErrTeamNotFound
	}
	var ok bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM quizgame.teams WHERE id = $1 AND session_id = $2)`, tid, sid).Scan(&ok)
	if err != nil || !ok {
		return ErrTeamNotFound
	}
	tag, err := pool.Exec(ctx, `
		UPDATE quizgame.session_players SET team_id = $3
		WHERE id = $1 AND session_id = $2 AND removed_at IS NULL`, pid, sid, tid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPlayerNotFound
	}
	return nil
}

// AssignPlayersInput is a bulk assign map playerId → teamId.
type AssignPlayersInput struct {
	Assignments map[string]string // playerID → teamID
	AutoBalance bool
}

// AssignPlayers assigns players to teams (host-controlled). AutoBalance spreads evenly.
func AssignPlayers(ctx context.Context, pool *pgxpool.Pool, sessionID string, in AssignPlayersInput) error {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	if engine.NormalizeMode(sess.Mode) != engine.ModeTeam {
		return ErrWrongMode
	}
	teams, err := ListTeams(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	if len(teams) == 0 {
		return fmt.Errorf("quizgame: no teams")
	}
	players, err := ListPlayers(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	assign := in.Assignments
	if in.AutoBalance || len(assign) == 0 {
		pids := make([]string, 0, len(players))
		for _, p := range players {
			pids = append(pids, p.ID)
		}
		tids := make([]string, 0, len(teams))
		for _, t := range teams {
			tids = append(tids, t.ID)
		}
		assign = engine.AutoBalanceAssign(pids, tids)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for pid, tid := range assign {
		playerUUID, err := uuid.Parse(pid)
		if err != nil {
			continue
		}
		teamUUID, err := uuid.Parse(tid)
		if err != nil {
			continue
		}
		_, err = tx.Exec(ctx, `
			UPDATE quizgame.session_players SET team_id = $3
			WHERE id = $1 AND session_id = $2 AND removed_at IS NULL`,
			playerUUID, sess.ID, teamUUID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// RefreshTeamScores recomputes teams.total_score from member aggregation and returns leaderboard.
func RefreshTeamScores(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]engine.TeamLeaderboardEntry, error) {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	ms := engine.ParseModeSettings(sess.Settings)
	agg := engine.NormalizeTeamConfig(ms.Team).Aggregate

	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	rows, err := pool.Query(ctx, `
		SELECT p.id, p.team_id, p.total_score,
			COALESCE((
				SELECT SUM(r.response_ms) FROM quizgame.session_responses r
				WHERE r.session_id = p.session_id AND r.player_id = p.id
			), 0)
		FROM quizgame.session_players p
		WHERE p.session_id = $1 AND p.removed_at IS NULL AND p.team_id IS NOT NULL`, sid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []engine.TeamMemberScore
	for rows.Next() {
		var pid, tid uuid.UUID
		var score, msSum int
		if err := rows.Scan(&pid, &tid, &score, &msSum); err != nil {
			return nil, err
		}
		members = append(members, engine.TeamMemberScore{
			PlayerID: pid.String(), TeamID: tid.String(), TotalScore: score, ResponseMs: msSum,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	teams, err := ListTeams(ctx, pool, sessionID)
	if err != nil {
		return nil, err
	}
	meta := map[string]struct{ Name, Color string }{}
	for _, t := range teams {
		c := ""
		if t.Color != nil {
			c = *t.Color
		}
		meta[t.ID] = struct{ Name, Color string }{t.Name, c}
	}
	board := engine.AggregateTeamScores(members, meta, agg)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, e := range board {
		tid, _ := uuid.Parse(e.TeamID)
		_, err = tx.Exec(ctx, `UPDATE quizgame.teams SET total_score = $2 WHERE id = $1`, tid, e.Score)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return board, nil
}

// TeamAlreadyAnswered reports whether any teammate answered this question (one_device_per_team).
func TeamAlreadyAnswered(ctx context.Context, pool *pgxpool.Pool, sessionID, teamID string, questionIndex int) (bool, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return false, ErrSessionNotFound
	}
	tid, err := uuid.Parse(teamID)
	if err != nil {
		return false, ErrTeamNotFound
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quizgame.session_responses r
		JOIN quizgame.session_players p ON p.id = r.player_id
		WHERE r.session_id = $1 AND r.question_index = $2
		  AND p.team_id = $3 AND p.removed_at IS NULL`, sid, questionIndex, tid).Scan(&n)
	return n > 0, err
}

// GetTeam returns a team by id.
func GetTeam(ctx context.Context, pool *pgxpool.Pool, teamID string) (*Team, error) {
	id, err := uuid.Parse(teamID)
	if err != nil {
		return nil, ErrTeamNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, session_id, name, color, total_score FROM quizgame.teams WHERE id = $1`, id)
	var t Team
	var tid, sid uuid.UUID
	var color *string
	err = row.Scan(&tid, &sid, &t.Name, &color, &t.TotalScore)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTeamNotFound
	}
	if err != nil {
		return nil, err
	}
	t.ID = tid.String()
	t.SessionID = sid.String()
	t.Color = color
	return &t, nil
}
