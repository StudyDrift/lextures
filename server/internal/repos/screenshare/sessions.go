package screenshare

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/screenshare/engine"
)

var (
	ErrSessionNotFound = errors.New("screenshare: session not found")
	ErrSessionEnded    = errors.New("screenshare: session ended")
	ErrViewerCap       = errors.New("screenshare: viewer cap exceeded")
	ErrBadJoinToken    = errors.New("screenshare: invalid join token")
)

// Session is one screenshare.sessions row.
type Session struct {
	ID                string
	CourseID          string
	HostID            *string
	Title             *string
	Status            string
	Policy            string
	PresentAudio      bool
	ViewerCap         int
	ActivePresenterID *string
	Settings          json.RawMessage
	JoinTokenHash     string
	StartedAt         *time.Time
	EndedAt           *time.Time
	CreatedAt         time.Time
}

const sessionCols = `
	id::text, course_id::text, host_id::text, title, status::text, policy::text, present_audio, viewer_cap,
	active_presenter_id::text, settings, join_token_hash, started_at, ended_at, created_at`

func scanSession(row pgx.Row) (*Session, error) {
	var s Session
	var hostID, title, presenter *string
	var settings []byte
	err := row.Scan(
		&s.ID, &s.CourseID, &hostID, &title, &s.Status, &s.Policy, &s.PresentAudio, &s.ViewerCap,
		&presenter, &settings, &s.JoinTokenHash, &s.StartedAt, &s.EndedAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.HostID = hostID
	s.Title = title
	s.ActivePresenterID = presenter
	if len(settings) > 0 {
		s.Settings = settings
	} else {
		s.Settings = json.RawMessage(`{}`)
	}
	return &s, nil
}

// HashJoinToken returns a hex SHA-256 of the raw token.
func HashJoinToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NewJoinToken returns a raw join token and its hash.
func NewJoinToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, HashJoinToken(raw), nil
}

// CreateInput starts an open session.
type CreateInput struct {
	CourseID     uuid.UUID
	HostID       uuid.UUID
	Title        string
	Policy       string
	PresentAudio bool
	ViewerCap    int
	Settings     json.RawMessage
}

// CreateSession inserts an open session and returns it plus the raw join token (once).
func CreateSession(ctx context.Context, pool *pgxpool.Pool, in CreateInput) (*Session, string, error) {
	if pool == nil {
		return nil, "", fmt.Errorf("screenshare: nil pool")
	}
	policy := in.Policy
	switch engine.Policy(policy) {
	case engine.PolicyHostOnly, engine.PolicyRequest, engine.PolicyFreeForAll:
	default:
		policy = string(engine.PolicyRequest)
	}
	capN := in.ViewerCap
	if capN <= 0 {
		capN = engine.DefaultViewerCap
	}
	raw, hash, err := NewJoinToken()
	if err != nil {
		return nil, "", err
	}
	settings := in.Settings
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	var title *string
	if t := in.Title; t != "" {
		title = &t
	}
	const q = `
		INSERT INTO screenshare.sessions (
			course_id, host_id, title, status, policy, present_audio, viewer_cap,
			settings, join_token_hash, started_at
		) VALUES ($1,$2,$3,'open',$4::screenshare.present_policy,$5,$6,$7,$8,NOW())
		RETURNING ` + sessionCols
	row := pool.QueryRow(ctx, q,
		in.CourseID, in.HostID, title, policy, in.PresentAudio, capN, settings, hash,
	)
	s, err := scanSession(row)
	if err != nil {
		return nil, "", err
	}
	_ = AppendEvent(ctx, pool, s.ID, "session_open", in.HostID.String(), map[string]any{
		"policy": policy,
	})
	return s, raw, nil
}

// GetSession loads a session by id.
func GetSession(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Session, error) {
	row := pool.QueryRow(ctx, `SELECT `+sessionCols+` FROM screenshare.sessions WHERE id = $1`, id)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	return s, err
}

// GetActiveForCourse returns the most recent open/presenting session for a course, if any.
func GetActiveForCourse(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) (*Session, error) {
	row := pool.QueryRow(ctx, `
		SELECT `+sessionCols+`
		FROM screenshare.sessions
		WHERE course_id = $1 AND status IN ('open','presenting')
		ORDER BY created_at DESC
		LIMIT 1`, courseID)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

// VerifyJoinToken checks the raw token against the stored hash.
func VerifyJoinToken(s *Session, raw string) bool {
	if s == nil || raw == "" {
		return false
	}
	return s.JoinTokenHash == HashJoinToken(raw)
}

// EndSession marks a session ended.
func EndSession(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, actorID string) error {
	tag, err := pool.Exec(ctx, `
		UPDATE screenshare.sessions
		SET status = 'ended', active_presenter_id = NULL, ended_at = NOW()
		WHERE id = $1 AND status IN ('open','presenting')`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionEnded
	}
	_ = AppendEvent(ctx, pool, id.String(), "session_end", actorID, map[string]any{"by": "host"})
	return nil
}

// SetPresenter updates active presenter and status.
func SetPresenter(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, presenterID *uuid.UUID, status string) error {
	_, err := pool.Exec(ctx, `
		UPDATE screenshare.sessions
		SET active_presenter_id = $2, status = $3::screenshare.session_status
		WHERE id = $1 AND status IN ('open','presenting')`, id, presenterID, status)
	return err
}

// SetPolicy updates the present policy.
func SetPolicy(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, policy string) error {
	_, err := pool.Exec(ctx, `
		UPDATE screenshare.sessions
		SET policy = $2::screenshare.present_policy
		WHERE id = $1 AND status IN ('open','presenting')`, id, policy)
	return err
}

// AppendEvent inserts the next seq event. Returns the seq.
func AppendEvent(ctx context.Context, pool *pgxpool.Pool, sessionID, typ, actorID string, payload map[string]any) int {
	if pool == nil {
		return 0
	}
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return 0
	}
	if payload == nil {
		payload = map[string]any{}
	}
	b, _ := json.Marshal(payload)
	var actor *uuid.UUID
	if actorID != "" {
		if u, err := uuid.Parse(actorID); err == nil {
			actor = &u
		}
	}
	var seq int
	_ = pool.QueryRow(ctx, `
		INSERT INTO screenshare.events (session_id, seq, type, actor_id, payload)
		SELECT $1, COALESCE(MAX(seq),0)+1, $2, $3, $4::jsonb
		FROM screenshare.events WHERE session_id = $1
		RETURNING seq`, sid, typ, actor, b).Scan(&seq)
	return seq
}

// UpsertParticipant records/updates a participant row.
func UpsertParticipant(ctx context.Context, pool *pgxpool.Pool, sessionID, userID uuid.UUID, role string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO screenshare.participants (session_id, user_id, role, connected, joined_at)
		VALUES ($1, $2, $3::screenshare.participant_role, TRUE, NOW())
		ON CONFLICT (session_id, user_id, role) DO UPDATE
		SET connected = TRUE, left_at = NULL, joined_at = COALESCE(screenshare.participants.joined_at, NOW())`,
		sessionID, userID, role)
	return err
}

// MarkParticipantLeft marks a participant disconnected.
func MarkParticipantLeft(ctx context.Context, pool *pgxpool.Pool, sessionID, userID uuid.UUID, role string) error {
	_, err := pool.Exec(ctx, `
		UPDATE screenshare.participants
		SET connected = FALSE, left_at = NOW()
		WHERE session_id = $1 AND user_id = $2 AND role = $3::screenshare.participant_role`,
		sessionID, userID, role)
	return err
}

// CountConnectedViewers counts connected viewer+display participants.
func CountConnectedViewers(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM screenshare.participants
		WHERE session_id = $1 AND connected = TRUE AND role IN ('viewer','display')`, sessionID).Scan(&n)
	return n, err
}

// ListAbandonedSessions returns idle open/presenting sessions older than maxAge.
func ListAbandonedSessions(ctx context.Context, pool *pgxpool.Pool, now time.Time, maxAge time.Duration, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 50
	}
	cutoff := now.Add(-maxAge)
	rows, err := pool.Query(ctx, `
		SELECT id FROM screenshare.sessions
		WHERE status IN ('open','presenting') AND created_at < $1
		ORDER BY created_at ASC
		LIMIT $2`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// FinaliseAbandoned marks a session abandoned.
func FinaliseAbandoned(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID, now time.Time) error {
	tag, err := pool.Exec(ctx, `
		UPDATE screenshare.sessions
		SET status = 'abandoned', active_presenter_id = NULL, ended_at = $2
		WHERE id = $1 AND status IN ('open','presenting')`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_ = AppendEvent(ctx, pool, id.String(), "session_end", "", map[string]any{"reason": "abandoned"})
	return nil
}

// EngineState builds an engine.State snapshot from a session row + viewer count.
func EngineState(s *Session, viewerCount int, pending []string) engine.State {
	st := engine.State{
		Status:          engine.Status(s.Status),
		Policy:          engine.Policy(s.Policy),
		ViewerCount:     viewerCount,
		ViewerCap:       s.ViewerCap,
		PendingRequests: pending,
	}
	if s.ActivePresenterID != nil {
		st.ActivePresenterID = *s.ActivePresenterID
	}
	return st
}
