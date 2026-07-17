package quizgame

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
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/quizgame/scoring"
)

var (
	ErrKitNotReady      = errors.New("quizgame: kit not ready")
	ErrSessionNotFound  = errors.New("quizgame: session not found")
	ErrSessionEnded     = errors.New("quizgame: session ended")
	ErrJoinCodeTaken    = errors.New("quizgame: join code collision")
	ErrPlayerExists     = errors.New("quizgame: nickname taken")
	ErrPlayerNotFound   = errors.New("quizgame: player not found")
	ErrDuplicateAnswer  = errors.New("quizgame: duplicate answer")
	ErrLateAnswer       = errors.New("quizgame: late answer")
	ErrNotAccepting     = errors.New("quizgame: not accepting answers")
	ErrJoinCodeNotFound = errors.New("quizgame: join code not found")
	ErrNicknameInvalid  = errors.New("quizgame: invalid nickname")
)

const sessionSelectCols = `
	id, kit_id, course_id, host_id, join_code, mode::text, status::text, pacing,
	kit_snapshot, current_index, current_phase, question_opened_at, question_deadline_at,
	host_disconnected_at, settings, scoring_profile, scoring_profile_ver, scoring_config,
	leaderboard_privacy, started_at, ended_at, created_at,
	allow_guests, lobby_locked, names_muted, one_session_rule, max_joins_per_ip`

// Session is one quizgame.sessions row.
type Session struct {
	ID                 string
	KitID              string
	CourseID           string
	HostID             *string
	JoinCode           *string
	Mode               string
	Status             string
	Pacing             string
	KitSnapshot        engine.KitSnapshot
	CurrentIndex       int
	CurrentPhase       string
	QuestionOpenedAt   *time.Time
	QuestionDeadlineAt *time.Time
	HostDisconnectedAt *time.Time
	Settings           json.RawMessage
	ScoringProfile     string
	ScoringProfileVer  int
	ScoringConfig      json.RawMessage
	LeaderboardPrivacy string
	StartedAt          *time.Time
	EndedAt            *time.Time
	CreatedAt          time.Time
	AllowGuests        bool
	LobbyLocked        bool
	NamesMuted         bool
	OneSessionRule     string
	MaxJoinsPerIP      int
}

// CreateGameInput starts a lobby session from a ready kit.
type CreateGameInput struct {
	CourseCode         string
	KitID              string
	HostID             uuid.UUID
	Pacing             string // manual | auto
	Mode               string // live_classic | team | student_paced | homework
	Settings           json.RawMessage
	TeamConfig         *engine.TeamConfig
	PacedConfig        *engine.PacedConfig
	ScoringProfile     string
	ScoringConfig      scoring.Config
	LeaderboardPrivacy string
	PowerUpsEnabled    *bool // nil → profile default
	NoJoinCode         bool  // homework attempts: no public join code
	AllowGuests        bool
	OneSessionRule     string
	MaxJoinsPerIP      int // 0 → default 5
}

// CreateGame snapshots the kit, allocates a join code, and inserts a lobby session.
func CreateGame(ctx context.Context, pool *pgxpool.Pool, in CreateGameInput) (*Session, error) {
	if pool == nil {
		return nil, fmt.Errorf("quizgame: nil pool")
	}
	pacing := in.Pacing
	if pacing != string(engine.PacingAuto) {
		pacing = string(engine.PacingManual)
	}
	mode := engine.NormalizeMode(in.Mode)
	settings, err := engine.MergeModeSettingsInto(in.Settings, mode, in.TeamConfig, in.PacedConfig)
	if err != nil {
		return nil, err
	}
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	profile := scoring.NormalizeProfile(in.ScoringProfile)
	cfg := scoring.ResolveConfig(profile, in.ScoringConfig)
	if in.PowerUpsEnabled != nil {
		cfg.PowerUpsEnabled = *in.PowerUpsEnabled
	}
	cfgJSON := scoring.MarshalConfig(cfg)
	privacy := scoring.NormalizePrivacy(in.LeaderboardPrivacy)
	oneSession := NormalizeOneSessionRule(in.OneSessionRule)
	maxJoins := in.MaxJoinsPerIP
	if maxJoins <= 0 {
		maxJoins = 5
	}
	if maxJoins > 100 {
		maxJoins = 100
	}

	kit, err := Get(ctx, pool, in.CourseCode, in.KitID)
	if err != nil {
		return nil, err
	}
	if kit == nil || kit.Archived {
		return nil, ErrSessionNotFound
	}
	vr, err := ValidateKit(ctx, pool, in.CourseCode, in.KitID)
	if err != nil {
		return nil, err
	}
	if vr == nil || !vr.IsReady {
		return nil, ErrKitNotReady
	}
	qs, err := ListQuestions(ctx, pool, in.CourseCode, in.KitID)
	if err != nil {
		return nil, err
	}
	snap, err := buildKitSnapshot(kit, qs)
	if err != nil {
		return nil, err
	}
	snapJSON, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}

	courseID, err := uuid.Parse(kit.CourseID)
	if err != nil {
		return nil, err
	}
	kitUUID, err := uuid.Parse(kit.ID)
	if err != nil {
		return nil, err
	}

	insertOnce := func(joinCode any) (*Session, error) {
		var id uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO quizgame.sessions (
				kit_id, course_id, host_id, join_code, mode, status, pacing,
				kit_snapshot, current_index, current_phase, settings,
				scoring_profile, scoring_profile_ver, scoring_config, leaderboard_privacy,
				allow_guests, one_session_rule, max_joins_per_ip
			) VALUES ($1,$2,$3,$4,$5::quizgame.session_mode,'lobby',$6,$7,-1,'lobby',$8,$9,$10,$11::jsonb,$12,$13,$14,$15)
			RETURNING id`,
			kitUUID, courseID, in.HostID, joinCode, string(mode), pacing, snapJSON, settings,
			profile, scoring.Version, []byte(cfgJSON), privacy,
			in.AllowGuests, oneSession, maxJoins,
		).Scan(&id)
		if err != nil {
			return nil, err
		}
		return GetSession(ctx, pool, id.String())
	}

	if in.NoJoinCode || mode == engine.ModeHomework {
		return insertOnce(nil)
	}

	var sess *Session
	for attempt := 0; attempt < 8; attempt++ {
		code, err := engine.GenerateJoinCode(engine.JoinCodeDigits)
		if err != nil {
			return nil, err
		}
		sess, err = insertOnce(code)
		if err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return nil, err
		}
		return sess, nil
	}
	return nil, ErrJoinCodeTaken
}

func buildKitSnapshot(kit *Kit, qs []Question) (engine.KitSnapshot, error) {
	out := engine.KitSnapshot{KitID: kit.ID, Title: kit.Title}
	for _, q := range qs {
		sq := engine.SnapshotQuestion{
			ID:               q.ID,
			Position:         q.Position,
			QuestionType:     q.QuestionType,
			Prompt:           q.Prompt,
			PromptMediaRef:   q.PromptMediaRef,
			PromptMediaAlt:   q.PromptMediaAlt,
			TimeLimitSeconds: q.TimeLimitSeconds,
			PointsStyle:      q.PointsStyle,
			AnswerShuffle:    q.AnswerShuffle,
			Explanation:      q.Explanation,
			SourceQuestionID: q.SourceQuestionID,
		}
		var opts []Option
		if len(q.Options) > 0 {
			if err := json.Unmarshal(q.Options, &opts); err != nil {
				return out, fmt.Errorf("options: %w", err)
			}
		}
		for _, o := range opts {
			sq.Options = append(sq.Options, engine.Option{
				ID: o.ID, Text: o.Text, MediaRef: o.MediaRef, MediaAlt: o.MediaAlt, IsCorrect: o.IsCorrect,
			})
		}
		if len(q.CorrectAnswer) > 0 && string(q.CorrectAnswer) != "null" {
			var ca map[string]any
			if err := json.Unmarshal(q.CorrectAnswer, &ca); err == nil {
				sq.CorrectAnswer = ca
			}
		}
		out.Questions = append(out.Questions, sq)
	}
	return out, nil
}

// GetSession loads a session by id.
func GetSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) (*Session, error) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT `+sessionSelectCols+`
		FROM quizgame.sessions WHERE id = $1`, id)
	return scanSession(row)
}

// GetSessionByCourse ensures the session belongs to the course code.
func GetSessionByCourse(ctx context.Context, pool *pgxpool.Pool, courseCode, sessionID string) (*Session, error) {
	s, err := GetSession(ctx, pool, sessionID)
	if err != nil || s == nil {
		return nil, err
	}
	var ok bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM course.courses c
			WHERE c.id = $1::uuid AND c.course_code = $2
		)`, s.CourseID, courseCode).Scan(&ok)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrSessionNotFound
	}
	return s, nil
}

// LookupByJoinCode finds an active session by join code.
func LookupByJoinCode(ctx context.Context, pool *pgxpool.Pool, code string) (*Session, error) {
	row := pool.QueryRow(ctx, `
		SELECT `+sessionSelectCols+`
		FROM quizgame.sessions
		WHERE join_code = $1 AND status IN ('lobby','running','paused')
		LIMIT 1`, code)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrJoinCodeNotFound
	}
	return s, err
}

func scanSession(row pgx.Row) (*Session, error) {
	var s Session
	var id, kitID, courseID uuid.UUID
	var hostID uuid.NullUUID
	var joinCode *string
	var snap []byte
	var settings []byte
	var scoringConfig []byte
	err := row.Scan(
		&id, &kitID, &courseID, &hostID, &joinCode, &s.Mode, &s.Status, &s.Pacing,
		&snap, &s.CurrentIndex, &s.CurrentPhase, &s.QuestionOpenedAt, &s.QuestionDeadlineAt,
		&s.HostDisconnectedAt, &settings, &s.ScoringProfile, &s.ScoringProfileVer, &scoringConfig,
		&s.LeaderboardPrivacy, &s.StartedAt, &s.EndedAt, &s.CreatedAt,
		&s.AllowGuests, &s.LobbyLocked, &s.NamesMuted, &s.OneSessionRule, &s.MaxJoinsPerIP,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	s.ID = id.String()
	s.KitID = kitID.String()
	s.CourseID = courseID.String()
	if hostID.Valid {
		h := hostID.UUID.String()
		s.HostID = &h
	}
	s.JoinCode = joinCode
	if err := json.Unmarshal(snap, &s.KitSnapshot); err != nil {
		return nil, err
	}
	if len(settings) == 0 {
		settings = []byte(`{}`)
	}
	s.Settings = settings
	if len(scoringConfig) == 0 {
		scoringConfig = []byte(`{}`)
	}
	s.ScoringConfig = scoringConfig
	if s.ScoringProfile == "" {
		s.ScoringProfile = scoring.ProfileCompetitive
	}
	if s.ScoringProfileVer == 0 {
		s.ScoringProfileVer = scoring.Version
	}
	if s.LeaderboardPrivacy == "" {
		s.LeaderboardPrivacy = scoring.PrivacyNames
	}
	s.OneSessionRule = NormalizeOneSessionRule(s.OneSessionRule)
	if s.MaxJoinsPerIP <= 0 {
		s.MaxJoinsPerIP = 5
	}
	return &s, nil
}

// EngineState projects a session into the pure reducer state.
func (s *Session) EngineState() engine.State {
	st := engine.State{
		SessionID:     s.ID,
		Status:        engine.Status(s.Status),
		Phase:         engine.Phase(s.CurrentPhase),
		Pacing:        engine.Pacing(s.Pacing),
		QuestionIndex: s.CurrentIndex,
		QuestionCount: len(s.KitSnapshot.Questions),
		OpenedAt:      s.QuestionOpenedAt,
		Deadline:      s.QuestionDeadlineAt,
		HostPaused:    s.Status == string(engine.StatusPaused) || s.CurrentPhase == string(engine.PhaseWaitingForHost),
	}
	return st
}

// PersistStateFull writes reducer output back to the session row inside a transaction.
func PersistStateFull(ctx context.Context, tx pgx.Tx, sessionID string, st engine.State, clearJoinCode bool, hostDisconnected bool) error {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	var hostDisc any
	if hostDisconnected {
		hostDisc = time.Now().UTC()
	}
	_, err = tx.Exec(ctx, `
		UPDATE quizgame.sessions SET
			status = $2::quizgame.session_status,
			current_index = $3,
			current_phase = $4,
			question_opened_at = $5,
			question_deadline_at = $6,
			host_disconnected_at = CASE
				WHEN $8::boolean THEN COALESCE(host_disconnected_at, $9::timestamptz)
				WHEN $4 <> 'waiting_for_host' AND $2::text IN ('lobby','running') THEN NULL
				ELSE host_disconnected_at
			END,
			started_at = COALESCE(started_at, CASE WHEN $2::text = 'running' THEN NOW() ELSE NULL END),
			ended_at = CASE WHEN $2::text IN ('ended','abandoned') THEN COALESCE(ended_at, NOW()) ELSE ended_at END,
			join_code = CASE WHEN $7::boolean THEN NULL ELSE join_code END
		WHERE id = $1`,
		id, string(st.Status), st.QuestionIndex, string(st.Phase),
		st.OpenedAt, st.Deadline, clearJoinCode, hostDisconnected, hostDisc,
	)
	return err
}

// AppendEvent writes the next monotonic event seq.
func AppendEvent(ctx context.Context, tx pgx.Tx, sessionID string, typ string, payload map[string]any) (int, error) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return 0, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	var seq int
	err = tx.QueryRow(ctx, `
		INSERT INTO quizgame.session_events (session_id, seq, type, payload)
		VALUES (
			$1,
			COALESCE((SELECT MAX(seq) FROM quizgame.session_events WHERE session_id = $1), 0) + 1,
			$2,
			$3::jsonb
		)
		RETURNING seq`, id, typ, b).Scan(&seq)
	return seq, err
}

// ListEventsAfter returns events with seq > afterSeq.
func ListEventsAfter(ctx context.Context, pool *pgxpool.Pool, sessionID string, afterSeq int) ([]SessionEvent, error) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT seq, type, payload, created_at
		FROM quizgame.session_events
		WHERE session_id = $1 AND seq > $2
		ORDER BY seq ASC`, id, afterSeq)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionEvent
	for rows.Next() {
		var e SessionEvent
		var payload []byte
		if err := rows.Scan(&e.Seq, &e.Type, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Payload = payload
		out = append(out, e)
	}
	return out, rows.Err()
}

// SessionEvent is one append-only log row.
type SessionEvent struct {
	Seq       int
	Type      string
	Payload   json.RawMessage
	CreatedAt time.Time
}

// LatestSeq returns the highest event seq for a session (0 if none).
func LatestSeq(ctx context.Context, pool *pgxpool.Pool, sessionID string) (int, error) {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return 0, err
	}
	var seq *int
	err = pool.QueryRow(ctx, `SELECT MAX(seq) FROM quizgame.session_events WHERE session_id = $1`, id).Scan(&seq)
	if err != nil {
		return 0, err
	}
	if seq == nil {
		return 0, nil
	}
	return *seq, nil
}

// HashPlayerToken returns a hex sha256 of the raw token for storage.
func HashPlayerToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NewPlayerToken generates a raw reconnect secret.
func NewPlayerToken() (raw string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ListAbandonedSessions returns sessions needing finalisation (host grace expired or stale lobby).
func ListAbandonedSessions(ctx context.Context, pool *pgxpool.Pool, now time.Time, hostGrace time.Duration, lobbyMaxAge time.Duration, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	hostCutoff := now.Add(-hostGrace)
	lobbyCutoff := now.Add(-lobbyMaxAge)
	rows, err := pool.Query(ctx, `
		SELECT id::text FROM quizgame.sessions
		WHERE (
			(status = 'paused' AND host_disconnected_at IS NOT NULL AND host_disconnected_at < $1)
			OR (status = 'lobby' AND created_at < $2)
			OR (status = 'running' AND host_disconnected_at IS NOT NULL AND host_disconnected_at < $1)
		)
		ORDER BY created_at ASC
		LIMIT $3`, hostCutoff, lobbyCutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// FinaliseAbandoned marks a session abandoned/ended and clears the join code.
func FinaliseAbandoned(ctx context.Context, pool *pgxpool.Pool, sessionID string, now time.Time) error {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE quizgame.sessions SET
			status = 'abandoned',
			current_phase = 'ended',
			join_code = NULL,
			ended_at = COALESCE(ended_at, $2)
		WHERE id = $1 AND status IN ('lobby','running','paused')`, id, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	if _, err := AppendEvent(ctx, tx, sessionID, "abandoned", map[string]any{"reason": "reaper"}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
