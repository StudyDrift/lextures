package quizgame

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
)

// Player is one session_players row.
type Player struct {
	ID                 string
	SessionID          string
	UserID             *string
	Nickname           string
	TeamID             *string
	TotalScore         int
	Streak             int
	Connected          bool
	JoinedAt           time.Time
	RemovedAt          *time.Time
	LastSeenAt         *time.Time
	CurrentIndex       int
	CurrentPhase       string
	QuestionOpenedAt   *time.Time
	QuestionDeadlineAt *time.Time
	QuestionOrder      []int
	TimeBudgetEndsAt   *time.Time
	FinishedAt         *time.Time
	Banned             bool
	RenamedByHost      bool
	JoinIPHash         string
}

const playerSelectCols = `
	id, session_id, user_id, nickname, team_id, total_score, streak, connected,
	joined_at, removed_at, last_seen_at, current_index, current_phase,
	question_opened_at, question_deadline_at, question_order, time_budget_ends_at, finished_at,
	banned, renamed_by_host, COALESCE(join_ip_hash, '')`

// AddPlayerInput joins a player to a lobby/running session.
type AddPlayerInput struct {
	SessionID  string
	UserID     *uuid.UUID
	Nickname   string
	ClientMeta json.RawMessage
	RemoteIP   string // for join-limit / ban integrity (salted hash stored)
	AllowGuest bool   // caller: platform + session allow guests
}

// AddPlayerResult includes the raw reconnect token (shown once).
type AddPlayerResult struct {
	Player      Player
	PlayerToken string // raw; store hashed in DB
	Rejoined    bool
}

// AddPlayer creates a player with a reconnect token.
// Enrolled users: one-session rule (takeover|refuse|off). Guests require AllowGuest + session.AllowGuests.
// Nickname moderation and join IP caps are enforced here (IQ.9).
func AddPlayer(ctx context.Context, pool *pgxpool.Pool, in AddPlayerInput) (*AddPlayerResult, error) {
	nick, err := ValidateNickname(in.Nickname)
	if err != nil {
		return nil, ErrNicknameInvalid
	}
	if err := ScreenNickname(nick); err != nil {
		return nil, ErrNicknameDenied
	}
	sess, err := GetSession(ctx, pool, in.SessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == "ended" || sess.Status == "abandoned" {
		return nil, ErrSessionEnded
	}
	if sess.LobbyLocked {
		return nil, ErrLobbyLocked
	}
	if in.UserID == nil {
		if !in.AllowGuest || !sess.AllowGuests {
			return nil, ErrGuestsNotAllowed
		}
	}

	ipHash := HashJoinIP(in.SessionID, in.RemoteIP)
	banned, err := IsIdentityBanned(ctx, pool, in.SessionID, in.UserID, ipHash)
	if err != nil {
		return nil, err
	}
	if banned {
		return nil, ErrPlayerBanned
	}

	rule := NormalizeOneSessionRule(sess.OneSessionRule)
	if in.UserID != nil && rule != OneSessionOff {
		existing, err := GetPlayerByUser(ctx, pool, in.SessionID, *in.UserID)
		if err == nil && existing != nil {
			if rule == OneSessionRefuse && existing.Connected {
				return nil, ErrOneSessionRefuse
			}
			// takeover (default): rotate token — legitimate reconnect / second-tab takeover
			raw, err := RotatePlayerToken(ctx, pool, existing.ID)
			if err != nil {
				return nil, err
			}
			_ = TouchPlayerSeen(ctx, pool, existing.ID, in.ClientMeta)
			if ipHash != "" {
				_ = setPlayerJoinIPHash(ctx, pool, existing.ID, ipHash)
			}
			p, err := GetPlayer(ctx, pool, existing.ID)
			if err != nil {
				return nil, err
			}
			return &AddPlayerResult{Player: *p, PlayerToken: raw, Rejoined: true}, nil
		}
		if err != nil && !errors.Is(err, ErrPlayerNotFound) {
			return nil, err
		}
	}

	if ipHash != "" && sess.MaxJoinsPerIP > 0 {
		n, err := CountJoinsByIPHash(ctx, pool, in.SessionID, ipHash)
		if err != nil {
			return nil, err
		}
		if n >= sess.MaxJoinsPerIP {
			return nil, ErrJoinLimitExceeded
		}
	}

	raw, err := NewPlayerToken()
	if err != nil {
		return nil, err
	}
	hashed := HashPlayerToken(raw)
	sid, err := uuid.Parse(in.SessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	var user any
	if in.UserID != nil {
		user = *in.UserID
	}
	meta := in.ClientMeta
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	var joinHash any
	if ipHash != "" {
		joinHash = ipHash
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO quizgame.session_players
			(session_id, user_id, nickname, player_token, connected, last_seen_at, client_meta, join_ip_hash)
		VALUES ($1, $2, $3, $4, TRUE, NOW(), $5::jsonb, $6)
		RETURNING id`, sid, user, nick, hashed, []byte(meta), joinHash).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPlayerExists
		}
		return nil, err
	}
	p, err := GetPlayer(ctx, pool, id.String())
	if err != nil {
		return nil, err
	}
	return &AddPlayerResult{Player: *p, PlayerToken: raw}, nil
}

func setPlayerJoinIPHash(ctx context.Context, pool *pgxpool.Pool, playerID, ipHash string) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `UPDATE quizgame.session_players SET join_ip_hash = $2 WHERE id = $1`, id, ipHash)
	return err
}

// GetPlayerByUser finds a non-removed player for an enrolled user in a session.
func GetPlayerByUser(ctx context.Context, pool *pgxpool.Pool, sessionID string, userID uuid.UUID) (*Player, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT `+playerSelectCols+`
		FROM quizgame.session_players
		WHERE session_id = $1 AND user_id = $2 AND removed_at IS NULL
		ORDER BY joined_at ASC
		LIMIT 1`, sid, userID)
	return scanPlayer(row)
}

// RotatePlayerToken issues a new reconnect secret for an existing player.
func RotatePlayerToken(ctx context.Context, pool *pgxpool.Pool, playerID string) (string, error) {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return "", ErrPlayerNotFound
	}
	raw, err := NewPlayerToken()
	if err != nil {
		return "", err
	}
	hashed := HashPlayerToken(raw)
	tag, err := pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET player_token = $2, connected = TRUE, last_seen_at = NOW()
		WHERE id = $1 AND removed_at IS NULL`, id, hashed)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", ErrPlayerNotFound
	}
	return raw, nil
}

// TouchPlayerSeen updates last_seen_at and optional coarse client_meta.
func TouchPlayerSeen(ctx context.Context, pool *pgxpool.Pool, playerID string, clientMeta json.RawMessage) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return ErrPlayerNotFound
	}
	if len(clientMeta) == 0 {
		_, err = pool.Exec(ctx, `UPDATE quizgame.session_players SET last_seen_at = NOW() WHERE id = $1`, id)
		return err
	}
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET last_seen_at = NOW(), client_meta = $2::jsonb
		WHERE id = $1`, id, []byte(clientMeta))
	return err
}

// GetPlayer loads a player by id.
func GetPlayer(ctx context.Context, pool *pgxpool.Pool, playerID string) (*Player, error) {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT `+playerSelectCols+`
		FROM quizgame.session_players WHERE id = $1`, id)
	return scanPlayer(row)
}

// GetPlayerByToken finds a player by raw reconnect token.
func GetPlayerByToken(ctx context.Context, pool *pgxpool.Pool, sessionID, rawToken string) (*Player, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	hashed := HashPlayerToken(rawToken)
	row := pool.QueryRow(ctx, `
		SELECT `+playerSelectCols+`
		FROM quizgame.session_players
		WHERE session_id = $1 AND player_token = $2 AND removed_at IS NULL`, sid, hashed)
	return scanPlayer(row)
}

func scanPlayer(row pgx.Row) (*Player, error) {
	var p Player
	var id, sid uuid.UUID
	var uid uuid.NullUUID
	var teamID uuid.NullUUID
	var lastSeen *time.Time
	var orderRaw []byte
	err := row.Scan(
		&id, &sid, &uid, &p.Nickname, &teamID, &p.TotalScore, &p.Streak, &p.Connected,
		&p.JoinedAt, &p.RemovedAt, &lastSeen, &p.CurrentIndex, &p.CurrentPhase,
		&p.QuestionOpenedAt, &p.QuestionDeadlineAt, &orderRaw, &p.TimeBudgetEndsAt, &p.FinishedAt,
		&p.Banned, &p.RenamedByHost, &p.JoinIPHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPlayerNotFound
	}
	if err != nil {
		return nil, err
	}
	p.ID = id.String()
	p.SessionID = sid.String()
	p.LastSeenAt = lastSeen
	if uid.Valid {
		u := uid.UUID.String()
		p.UserID = &u
	}
	if teamID.Valid {
		t := teamID.UUID.String()
		p.TeamID = &t
	}
	if len(orderRaw) > 0 {
		_ = json.Unmarshal(orderRaw, &p.QuestionOrder)
	}
	if p.CurrentPhase == "" {
		p.CurrentPhase = "lobby"
	}
	return &p, nil
}

// ListPlayers returns non-removed players for a session.
func ListPlayers(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]Player, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT `+playerSelectCols+`
		FROM quizgame.session_players
		WHERE session_id = $1 AND removed_at IS NULL
		ORDER BY joined_at ASC`, sid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Player
	for rows.Next() {
		p, err := scanPlayer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// InitPlayerPacedProgress sets question order and opens Q0 for student_paced/homework.
func InitPlayerPacedProgress(ctx context.Context, pool *pgxpool.Pool, playerID string, order []int, budgetEnds *time.Time) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return ErrPlayerNotFound
	}
	p, err := GetPlayer(ctx, pool, playerID)
	if err != nil {
		return err
	}
	sess, err := GetSession(ctx, pool, p.SessionID)
	if err != nil {
		return err
	}
	if len(order) == 0 {
		order = engine.SequentialQuestionOrder(len(sess.KitSnapshot.Questions))
	}
	orderJSON, _ := json.Marshal(order)
	now := time.Now().UTC()
	kitIdx := 0
	if len(order) > 0 {
		kitIdx = order[0]
	}
	var deadline *time.Time
	ms := engine.ParseModeSettings(sess.Settings)
	pc := engine.NormalizePacedConfig(ms.Paced)
	if kitIdx >= 0 && kitIdx < len(sess.KitSnapshot.Questions) {
		deadline = engine.PacedDeadline(now, sess.KitSnapshot.Questions[kitIdx].TimeLimitSeconds, pc.PerQuestionTimers)
	}
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.session_players SET
			question_order = $2::jsonb,
			current_index = 0,
			current_phase = 'question_open',
			question_opened_at = $3,
			question_deadline_at = $4,
			time_budget_ends_at = $5,
			finished_at = NULL
		WHERE id = $1`, id, orderJSON, now, deadline, budgetEnds)
	return err
}

// AdvancePlayerPaced moves a player to the next question or finishes (AC-3).
func AdvancePlayerPaced(ctx context.Context, pool *pgxpool.Pool, playerID string, now time.Time) (*Player, error) {
	p, err := GetPlayer(ctx, pool, playerID)
	if err != nil {
		return nil, err
	}
	if p.FinishedAt != nil {
		return p, nil
	}
	sess, err := GetSession(ctx, pool, p.SessionID)
	if err != nil {
		return nil, err
	}
	if engine.TimeBudgetExpired(p.TimeBudgetEndsAt, now) {
		return FinalizePlayerPaced(ctx, pool, playerID, now)
	}
	next := p.CurrentIndex + 1
	if next >= len(p.QuestionOrder) {
		return FinalizePlayerPaced(ctx, pool, playerID, now)
	}
	kitIdx, ok := engine.ResolveQuestionIndex(p.QuestionOrder, next)
	if !ok {
		return FinalizePlayerPaced(ctx, pool, playerID, now)
	}
	ms := engine.ParseModeSettings(sess.Settings)
	pc := engine.NormalizePacedConfig(ms.Paced)
	deadline := engine.PacedDeadline(now, sess.KitSnapshot.Questions[kitIdx].TimeLimitSeconds, pc.PerQuestionTimers)
	id, _ := uuid.Parse(playerID)
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.session_players SET
			current_index = $2,
			current_phase = 'question_open',
			question_opened_at = $3,
			question_deadline_at = $4
		WHERE id = $1`, id, next, now, deadline)
	if err != nil {
		return nil, err
	}
	return GetPlayer(ctx, pool, playerID)
}

// FinalizePlayerPaced locks remaining questions and marks finished (AC-4).
func FinalizePlayerPaced(ctx context.Context, pool *pgxpool.Pool, playerID string, now time.Time) (*Player, error) {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return nil, ErrPlayerNotFound
	}
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.session_players SET
			current_phase = 'ended',
			finished_at = COALESCE(finished_at, $2),
			question_deadline_at = NULL
		WHERE id = $1`, id, now)
	if err != nil {
		return nil, err
	}
	return GetPlayer(ctx, pool, playerID)
}

// StartPacedGameForAll initialises every player when host starts student-paced.
func StartPacedGameForAll(ctx context.Context, pool *pgxpool.Pool, sessionID string, now time.Time) error {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	if engine.NormalizeMode(sess.Mode) != engine.ModeStudentPaced {
		return ErrWrongMode
	}
	ms := engine.ParseModeSettings(sess.Settings)
	pc := engine.NormalizePacedConfig(ms.Paced)
	var budgetEnds *time.Time
	if pc.TimeBudgetSeconds > 0 {
		t := now.Add(time.Duration(pc.TimeBudgetSeconds) * time.Second)
		budgetEnds = &t
	}
	players, err := ListPlayers(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	n := len(sess.KitSnapshot.Questions)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		UPDATE quizgame.sessions SET status = 'running', started_at = COALESCE(started_at, $2),
			current_phase = 'running', current_index = 0
		WHERE id = $1::uuid`, sessionID, now)
	if err != nil {
		return err
	}
	for _, p := range players {
		order := engine.SequentialQuestionOrder(n)
		if pc.Shuffle {
			order = engine.ShuffleQuestionOrder(n, nil)
		}
		orderJSON, _ := json.Marshal(order)
		kitIdx := 0
		if len(order) > 0 {
			kitIdx = order[0]
		}
		deadline := (*time.Time)(nil)
		if kitIdx < len(sess.KitSnapshot.Questions) {
			deadline = engine.PacedDeadline(now, sess.KitSnapshot.Questions[kitIdx].TimeLimitSeconds, pc.PerQuestionTimers)
		}
		pid, _ := uuid.Parse(p.ID)
		_, err = tx.Exec(ctx, `
			UPDATE quizgame.session_players SET
				question_order = $2::jsonb, current_index = 0, current_phase = 'question_open',
				question_opened_at = $3, question_deadline_at = $4, time_budget_ends_at = $5, finished_at = NULL
			WHERE id = $1`, pid, orderJSON, now, deadline, budgetEnds)
		if err != nil {
			return err
		}
	}
	if _, err := AppendEvent(ctx, tx, sessionID, "paced_start", map[string]any{"playerCount": len(players)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PacedHostProgress returns aggregate progress for the host view (AC-3).
func PacedHostProgress(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]engine.ProgressBucket, int, int, error) {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return nil, 0, 0, err
	}
	players, err := ListPlayers(ctx, pool, sessionID)
	if err != nil {
		return nil, 0, 0, err
	}
	indices := make([]int, len(players))
	finished := make([]bool, len(players))
	finCount := 0
	for i, p := range players {
		indices[i] = p.CurrentIndex
		finished[i] = p.FinishedAt != nil
		if finished[i] {
			finCount++
		}
	}
	buckets := engine.AggregatePacedProgress(len(sess.KitSnapshot.Questions), indices, finished)
	return buckets, len(players), finCount, nil
}

// SetPlayerConnected updates the connected flag.
func SetPlayerConnected(ctx context.Context, pool *pgxpool.Pool, playerID string, connected bool) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET connected = $2, last_seen_at = CASE WHEN $2 THEN NOW() ELSE last_seen_at END
		WHERE id = $1`, id, connected)
	return err
}

// KickPlayer soft-removes and bans a player so they cannot rejoin this game (IQ.9).
func KickPlayer(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string) error {
	return BanPlayer(ctx, pool, sessionID, playerID)
}

// AddPlayerScore increments total_score and optionally streak.
func AddPlayerScore(ctx context.Context, tx pgx.Tx, playerID string, points int, correct bool) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	if correct {
		_, err = tx.Exec(ctx, `
			UPDATE quizgame.session_players
			SET total_score = total_score + $2, streak = streak + 1
			WHERE id = $1`, id, points)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE quizgame.session_players
			SET total_score = total_score + $2, streak = 0
			WHERE id = $1`, id, points)
	}
	return err
}

// SetPlayerScoreAndStreak adds points and sets the absolute streak (IQ.5 / shield).
func SetPlayerScoreAndStreak(ctx context.Context, tx pgx.Tx, playerID string, points, streakAfter int) error {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	if streakAfter < 0 {
		streakAfter = 0
	}
	_, err = tx.Exec(ctx, `
		UPDATE quizgame.session_players
		SET total_score = total_score + $2, streak = $3
		WHERE id = $1`, id, points, streakAfter)
	return err
}
