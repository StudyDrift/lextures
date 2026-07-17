package quizgame

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/service/boardfilter"
)

var (
	ErrLobbyLocked       = errors.New("quizgame: lobby locked")
	ErrPlayerBanned      = errors.New("quizgame: player banned")
	ErrNicknameDenied    = errors.New("quizgame: nickname denied")
	ErrJoinLimitExceeded = errors.New("quizgame: join limit exceeded")
	ErrOneSessionRefuse  = errors.New("quizgame: one session refuse")
	ErrContentDenied     = errors.New("quizgame: content denied")
)

const (
	OneSessionTakeover = "takeover"
	OneSessionRefuse   = "refuse"
	OneSessionOff      = "off"

	SafetyNicknameDenied = "nickname_denied"
	SafetyKicked         = "kicked"
	SafetyBanned         = "banned"
	SafetyRenamed        = "renamed"
	SafetyMuted          = "muted"
	SafetyLobbyLocked    = "lobby_locked"
	SafetyIntegrityFlag  = "integrity_flag"
	SafetyContentFlag    = "content_flag"
	SafetyContentDenied  = "content_denied"
)

// NormalizeOneSessionRule returns takeover|refuse|off.
func NormalizeOneSessionRule(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case OneSessionRefuse:
		return OneSessionRefuse
	case OneSessionOff:
		return OneSessionOff
	default:
		return OneSessionTakeover
	}
}

// ScreenNickname runs the shared blocklist against a nickname (fail-closed on match).
func ScreenNickname(nickname string) error {
	res := boardfilter.Match(nickname, nil)
	if res.Matched {
		return ErrNicknameDenied
	}
	return nil
}

// ScreenOpenText screens free-text answers for projector display. Matched → deny.
func ScreenOpenText(text string) (denied bool, term string) {
	res := boardfilter.Match(text, nil)
	return res.Matched, res.Term
}

// HashJoinIP salts the remote IP with the session id (transient integrity signal).
func HashJoinIP(sessionID, remoteIP string) string {
	remoteIP = strings.TrimSpace(remoteIP)
	if remoteIP == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(sessionID + "|" + remoteIP))
	return hex.EncodeToString(sum[:16])
}

// DisplayNickname returns the public label honouring mute-names.
func DisplayNickname(namesMuted bool, playerIndex int, nickname string) string {
	if namesMuted {
		n := playerIndex + 1
		if n < 1 {
			n = 1
		}
		return fmt.Sprintf("Player %d", n)
	}
	return nickname
}

// SafetyEvent is one audit row.
type SafetyEvent struct {
	ID        int64
	SessionID string
	PlayerID  *string
	ActorID   *string
	Kind      string
	Detail    json.RawMessage
	CreatedAt time.Time
}

// RecordSafetyEvent appends an audited safety action.
func RecordSafetyEvent(ctx context.Context, pool *pgxpool.Pool, sessionID string, playerID, actorID *uuid.UUID, kind string, detail any) error {
	if pool == nil {
		return fmt.Errorf("quizgame: nil pool")
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	raw, err := json.Marshal(detail)
	if err != nil || len(raw) == 0 {
		raw = []byte(`{}`)
	}
	var pid, aid any
	if playerID != nil {
		pid = *playerID
	}
	if actorID != nil {
		aid = *actorID
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO quizgame.safety_events (session_id, player_id, actor_id, kind, detail)
		VALUES ($1, $2, $3, $4, $5::jsonb)`, sid, pid, aid, kind, raw)
	return err
}

// ListSafetyEvents returns recent safety events for a session (newest last).
func ListSafetyEvents(ctx context.Context, pool *pgxpool.Pool, sessionID string, limit int) ([]SafetyEvent, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := pool.Query(ctx, `
		SELECT id, session_id, player_id, actor_id, kind, detail, created_at
		FROM quizgame.safety_events
		WHERE session_id = $1
		ORDER BY created_at ASC
		LIMIT $2`, sid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SafetyEvent
	for rows.Next() {
		var e SafetyEvent
		var id uuid.UUID
		var pid, aid uuid.NullUUID
		var detail []byte
		if err := rows.Scan(&e.ID, &id, &pid, &aid, &e.Kind, &detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.SessionID = id.String()
		if pid.Valid {
			s := pid.UUID.String()
			e.PlayerID = &s
		}
		if aid.Valid {
			s := aid.UUID.String()
			e.ActorID = &s
		}
		if len(detail) == 0 {
			detail = []byte(`{}`)
		}
		e.Detail = detail
		out = append(out, e)
	}
	return out, rows.Err()
}

// SetLobbyLocked toggles new-join lock.
func SetLobbyLocked(ctx context.Context, pool *pgxpool.Pool, sessionID string, locked bool) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	tag, err := pool.Exec(ctx, `UPDATE quizgame.sessions SET lobby_locked = $2 WHERE id = $1`, sid, locked)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// SetNamesMuted toggles mute-names for projector/public labels.
func SetNamesMuted(ctx context.Context, pool *pgxpool.Pool, sessionID string, muted bool) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	tag, err := pool.Exec(ctx, `UPDATE quizgame.sessions SET names_muted = $2 WHERE id = $1`, sid, muted)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// PatchSessionSafety updates allow_guests / one_session_rule / max_joins_per_ip (host at create or lobby).
type SessionSafetyPatch struct {
	AllowGuests     *bool
	OneSessionRule  *string
	MaxJoinsPerIP   *int
	LobbyLocked     *bool
	NamesMuted      *bool
}

// PatchSessionSafety applies optional safety settings.
func PatchSessionSafety(ctx context.Context, pool *pgxpool.Pool, sessionID string, patch SessionSafetyPatch) error {
	sess, err := GetSession(ctx, pool, sessionID)
	if err != nil {
		return err
	}
	allow := sess.AllowGuests
	if patch.AllowGuests != nil {
		allow = *patch.AllowGuests
	}
	rule := sess.OneSessionRule
	if patch.OneSessionRule != nil {
		rule = NormalizeOneSessionRule(*patch.OneSessionRule)
	}
	maxIP := sess.MaxJoinsPerIP
	if patch.MaxJoinsPerIP != nil {
		maxIP = *patch.MaxJoinsPerIP
		if maxIP < 1 {
			maxIP = 1
		}
		if maxIP > 100 {
			maxIP = 100
		}
	}
	locked := sess.LobbyLocked
	if patch.LobbyLocked != nil {
		locked = *patch.LobbyLocked
	}
	muted := sess.NamesMuted
	if patch.NamesMuted != nil {
		muted = *patch.NamesMuted
	}
	sid, _ := uuid.Parse(sessionID)
	_, err = pool.Exec(ctx, `
		UPDATE quizgame.sessions SET
			allow_guests = $2,
			one_session_rule = $3,
			max_joins_per_ip = $4,
			lobby_locked = $5,
			names_muted = $6
		WHERE id = $1`, sid, allow, rule, maxIP, locked, muted)
	return err
}

// IsIdentityBanned reports whether an enrolled user or IP hash is banned from the session.
func IsIdentityBanned(ctx context.Context, pool *pgxpool.Pool, sessionID string, userID *uuid.UUID, ipHash string) (bool, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return false, ErrSessionNotFound
	}
	if userID != nil {
		var banned bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM quizgame.session_players
				WHERE session_id = $1 AND user_id = $2 AND banned = TRUE
			)`, sid, *userID).Scan(&banned)
		if err != nil {
			return false, err
		}
		if banned {
			return true, nil
		}
	}
	if ipHash != "" {
		var banned bool
		err = pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM quizgame.session_players
				WHERE session_id = $1 AND join_ip_hash = $2 AND banned = TRUE
			)`, sid, ipHash).Scan(&banned)
		if err != nil {
			return false, err
		}
		return banned, nil
	}
	return false, nil
}

// CountJoinsByIPHash counts non-removed player rows with the given IP hash.
func CountJoinsByIPHash(ctx context.Context, pool *pgxpool.Pool, sessionID, ipHash string) (int, error) {
	if ipHash == "" {
		return 0, nil
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return 0, ErrSessionNotFound
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quizgame.session_players
		WHERE session_id = $1 AND join_ip_hash = $2 AND removed_at IS NULL`, sid, ipHash).Scan(&n)
	return n, err
}

// BanPlayer kicks and bans a player (blocks rejoin).
func BanPlayer(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string) error {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return ErrPlayerNotFound
	}
	tag, err := pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET removed_at = COALESCE(removed_at, NOW()), connected = FALSE, banned = TRUE
		WHERE id = $1 AND session_id = $2`, pid, sid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPlayerNotFound
	}
	return nil
}

// RenamePlayer forces a neutral nickname (host control).
func RenamePlayer(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID, nickname string) error {
	nick, err := ValidateNickname(nickname)
	if err != nil {
		return ErrNicknameInvalid
	}
	if err := ScreenNickname(nick); err != nil {
		return ErrNicknameDenied
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	pid, err := uuid.Parse(playerID)
	if err != nil {
		return ErrPlayerNotFound
	}
	tag, err := pool.Exec(ctx, `
		UPDATE quizgame.session_players
		SET nickname = $3, renamed_by_host = TRUE
		WHERE id = $1 AND session_id = $2 AND removed_at IS NULL`, pid, sid, nick)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrPlayerExists
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrPlayerNotFound
	}
	return nil
}

// NeutralPlayerName builds "Player N" for host rename.
func NeutralPlayerName(ctx context.Context, pool *pgxpool.Pool, sessionID, playerID string) (string, error) {
	players, err := ListPlayers(ctx, pool, sessionID)
	if err != nil {
		return "", err
	}
	for i, p := range players {
		if p.ID == playerID {
			return fmt.Sprintf("Player %d", i+1), nil
		}
	}
	// Fallback if already removed from active list.
	return "Player", nil
}

// IntegrityFlag is an advisory post-game signal (never auto-punitive).
type IntegrityFlag struct {
	Kind     string `json:"kind"`
	PlayerID string `json:"playerId,omitempty"`
	Detail   string `json:"detail"`
}

// ComputeIntegrityFlags returns advisory signals for the host report.
func ComputeIntegrityFlags(ctx context.Context, pool *pgxpool.Pool, sessionID string) ([]IntegrityFlag, error) {
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	var flags []IntegrityFlag

	// Duplicate join_ip_hash across multiple active/historical player rows.
	rows, err := pool.Query(ctx, `
		SELECT join_ip_hash, COUNT(*)::int
		FROM quizgame.session_players
		WHERE session_id = $1 AND join_ip_hash IS NOT NULL AND join_ip_hash <> ''
		GROUP BY join_ip_hash
		HAVING COUNT(*) > 1`, sid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		var n int
		if err := rows.Scan(&hash, &n); err != nil {
			return nil, err
		}
		flags = append(flags, IntegrityFlag{
			Kind:   "duplicate_device",
			Detail: fmt.Sprintf("%d players shared a device signal", n),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Improbably fast answers: median response under 400ms across ≥3 answers for a player.
	fastRows, err := pool.Query(ctx, `
		SELECT player_id::text, COUNT(*)::int,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY response_ms)::float8 AS med
		FROM quizgame.session_responses
		WHERE session_id = $1
		GROUP BY player_id
		HAVING COUNT(*) >= 3 AND PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY response_ms) < 400`, sid)
	if err != nil {
		// PERCENTILE_CONT requires ordered-set agg; if unavailable, skip silently.
		if !errors.Is(err, pgx.ErrNoRows) {
			return flags, nil
		}
		return flags, nil
	}
	defer fastRows.Close()
	for fastRows.Next() {
		var pid string
		var n int
		var med float64
		if err := fastRows.Scan(&pid, &n, &med); err != nil {
			return flags, nil
		}
		flags = append(flags, IntegrityFlag{
			Kind:     "improbably_fast",
			PlayerID: pid,
			Detail:   fmt.Sprintf("median response %.0fms across %d answers", med, n),
		})
	}
	return flags, nil
}
