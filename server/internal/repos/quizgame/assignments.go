package quizgame

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/quizgame/engine"
	"github.com/lextures/lextures/server/internal/quizgame/scoring"
	"github.com/lextures/lextures/server/internal/repos/studentaccommodations"
)

var (
	ErrAssignmentNotFound = errors.New("quizgame: assignment not found")
	ErrGuestsNotAllowed   = errors.New("quizgame: guests not allowed")
)

// Assignment is one quizgame.assignments row.
type Assignment struct {
	ID              string
	KitID           string
	CourseID        string
	Title           string
	OpensAt         *time.Time
	DueAt           *time.Time
	ClosesAt        *time.Time
	AttemptsAllowed int
	GradePolicy     string
	Shuffle         bool
	PointsPossible  *float64
	GradebookItemID *string
	ScoringProfile  string
	ScoringConfig   json.RawMessage
	CreatedBy       *string
	CreatedAt       time.Time
}

// CreateAssignmentInput creates an async homework assignment.
type CreateAssignmentInput struct {
	CourseCode      string
	KitID           string
	Title           string
	OpensAt         *time.Time
	DueAt           *time.Time
	ClosesAt        *time.Time
	AttemptsAllowed int
	GradePolicy     string
	Shuffle         *bool
	PointsPossible  *float64
	ScoringProfile  string
	ScoringConfig   scoring.Config
	CreatedBy       uuid.UUID
}

// CreateAssignment binds a ready kit to a course as homework (IQ.6 FR-6).
func CreateAssignment(ctx context.Context, pool *pgxpool.Pool, in CreateAssignmentInput) (*Assignment, error) {
	if pool == nil {
		return nil, fmt.Errorf("quizgame: nil pool")
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
	title := in.Title
	if title == "" {
		title = kit.Title
	}
	attempts := in.AttemptsAllowed
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 100 {
		attempts = 100
	}
	policy := string(engine.NormalizeGradePolicy(in.GradePolicy))
	shuffle := true
	if in.Shuffle != nil {
		shuffle = *in.Shuffle
	}
	profile := scoring.NormalizeProfile(in.ScoringProfile)
	cfg := scoring.ResolveConfig(profile, in.ScoringConfig)
	cfgJSON := scoring.MarshalConfig(cfg)

	courseID, err := uuid.Parse(kit.CourseID)
	if err != nil {
		return nil, err
	}
	kitUUID, err := uuid.Parse(kit.ID)
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO quizgame.assignments (
			kit_id, course_id, title, opens_at, due_at, closes_at,
			attempts_allowed, grade_policy, shuffle, points_possible,
			scoring_profile, scoring_config, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13)
		RETURNING id`,
		kitUUID, courseID, title, in.OpensAt, in.DueAt, in.ClosesAt,
		attempts, policy, shuffle, in.PointsPossible,
		profile, []byte(cfgJSON), in.CreatedBy,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return GetAssignment(ctx, pool, id.String())
}

// GetAssignment loads by id.
func GetAssignment(ctx context.Context, pool *pgxpool.Pool, assignmentID string) (*Assignment, error) {
	id, err := uuid.Parse(assignmentID)
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, kit_id, course_id, title, opens_at, due_at, closes_at,
			attempts_allowed, grade_policy, shuffle, points_possible, gradebook_item_id,
			scoring_profile, scoring_config, created_by, created_at
		FROM quizgame.assignments WHERE id = $1`, id)
	return scanAssignment(row)
}

// GetAssignmentByCourse ensures the assignment belongs to the course code.
func GetAssignmentByCourse(ctx context.Context, pool *pgxpool.Pool, courseCode, assignmentID string) (*Assignment, error) {
	a, err := GetAssignment(ctx, pool, assignmentID)
	if err != nil || a == nil {
		return nil, err
	}
	var ok bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM course.courses c
			WHERE c.id = $1::uuid AND c.course_code = $2
		)`, a.CourseID, courseCode).Scan(&ok)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAssignmentNotFound
	}
	return a, nil
}

// ListAssignmentsForCourse lists homework assignments for a course.
func ListAssignmentsForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode string) ([]Assignment, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.id, a.kit_id, a.course_id, a.title, a.opens_at, a.due_at, a.closes_at,
			a.attempts_allowed, a.grade_policy, a.shuffle, a.points_possible, a.gradebook_item_id,
			a.scoring_profile, a.scoring_config, a.created_by, a.created_at
		FROM quizgame.assignments a
		JOIN course.courses c ON c.id = a.course_id
		WHERE c.course_code = $1
		ORDER BY a.due_at NULLS LAST, a.created_at DESC`, courseCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Assignment
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func scanAssignment(row pgx.Row) (*Assignment, error) {
	var a Assignment
	var id, kitID, courseID uuid.UUID
	var gbItem uuid.NullUUID
	var createdBy uuid.NullUUID
	var points *float64
	var cfg []byte
	err := row.Scan(
		&id, &kitID, &courseID, &a.Title, &a.OpensAt, &a.DueAt, &a.ClosesAt,
		&a.AttemptsAllowed, &a.GradePolicy, &a.Shuffle, &points, &gbItem,
		&a.ScoringProfile, &cfg, &createdBy, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAssignmentNotFound
	}
	if err != nil {
		return nil, err
	}
	a.ID = id.String()
	a.KitID = kitID.String()
	a.CourseID = courseID.String()
	a.PointsPossible = points
	if gbItem.Valid {
		s := gbItem.UUID.String()
		a.GradebookItemID = &s
	}
	if createdBy.Valid {
		s := createdBy.UUID.String()
		a.CreatedBy = &s
	}
	if len(cfg) == 0 {
		cfg = []byte(`{}`)
	}
	a.ScoringConfig = cfg
	return &a, nil
}

// AssignmentWindowForUser resolves opens/due/close with accommodations (IQ.6 FR-8 / AC-7).
func AssignmentWindowForUser(ctx context.Context, pool *pgxpool.Pool, a *Assignment, userID uuid.UUID, now time.Time) (engine.AssignmentWindow, int, error) {
	base := engine.AssignmentWindow{OpensAt: a.OpensAt, DueAt: a.DueAt, ClosesAt: a.ClosesAt}
	mult := 1.0
	extra := 0
	courseID, err := uuid.Parse(a.CourseID)
	if err == nil {
		acc, err := studentaccommodations.FindActiveForCourse(ctx, pool, userID, courseID)
		if err == nil && acc != nil {
			if acc.TimeMultiplier > 0 {
				mult = acc.TimeMultiplier
			}
			extra = int(acc.ExtraAttempts)
		} else {
			g, gerr := studentaccommodations.FindActiveGlobal(ctx, pool, userID)
			if gerr == nil && g != nil {
				if g.TimeMultiplier > 0 {
					mult = g.TimeMultiplier
				}
				extra = int(g.ExtraAttempts)
			}
		}
	}
	eff := engine.EffectiveWindow(base, mult, now)
	allowed := engine.EffectiveAttemptsAllowed(a.AttemptsAllowed, extra)
	return eff, allowed, nil
}

// AssignmentAttempt is one homework run.
type AssignmentAttempt struct {
	ID           string
	AssignmentID string
	UserID       string
	SessionID    string
	AttemptNo    int
	Score        int
	SubmittedAt  *time.Time
	IsLate       bool
}

// CountAttempts returns how many attempts a user has for an assignment.
func CountAttempts(ctx context.Context, pool *pgxpool.Pool, assignmentID string, userID uuid.UUID) (int, error) {
	aid, err := uuid.Parse(assignmentID)
	if err != nil {
		return 0, ErrAssignmentNotFound
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quizgame.assignment_attempts
		WHERE assignment_id = $1 AND user_id = $2`, aid, userID).Scan(&n)
	return n, err
}

// ListAttemptsForUser returns attempts ordered by attempt_no.
func ListAttemptsForUser(ctx context.Context, pool *pgxpool.Pool, assignmentID string, userID uuid.UUID) ([]AssignmentAttempt, error) {
	aid, err := uuid.Parse(assignmentID)
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	rows, err := pool.Query(ctx, `
		SELECT id, assignment_id, user_id, session_id, attempt_no, score, submitted_at, is_late
		FROM quizgame.assignment_attempts
		WHERE assignment_id = $1 AND user_id = $2
		ORDER BY attempt_no ASC`, aid, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AssignmentAttempt
	for rows.Next() {
		var at AssignmentAttempt
		var id, asid, uid, sid uuid.UUID
		if err := rows.Scan(&id, &asid, &uid, &sid, &at.AttemptNo, &at.Score, &at.SubmittedAt, &at.IsLate); err != nil {
			return nil, err
		}
		at.ID = id.String()
		at.AssignmentID = asid.String()
		at.UserID = uid.String()
		at.SessionID = sid.String()
		out = append(out, at)
	}
	return out, rows.Err()
}

// GetAttempt loads an attempt by id.
func GetAttempt(ctx context.Context, pool *pgxpool.Pool, attemptID string) (*AssignmentAttempt, error) {
	id, err := uuid.Parse(attemptID)
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, assignment_id, user_id, session_id, attempt_no, score, submitted_at, is_late
		FROM quizgame.assignment_attempts WHERE id = $1`, id)
	var at AssignmentAttempt
	var aid, asid, uid, sid uuid.UUID
	err = row.Scan(&aid, &asid, &uid, &sid, &at.AttemptNo, &at.Score, &at.SubmittedAt, &at.IsLate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAssignmentNotFound
	}
	if err != nil {
		return nil, err
	}
	at.ID = aid.String()
	at.AssignmentID = asid.String()
	at.UserID = uid.String()
	at.SessionID = sid.String()
	return &at, nil
}

// FindOpenAttempt returns an in-progress (unsubmitted) attempt for resume.
func FindOpenAttempt(ctx context.Context, pool *pgxpool.Pool, assignmentID string, userID uuid.UUID) (*AssignmentAttempt, error) {
	aid, err := uuid.Parse(assignmentID)
	if err != nil {
		return nil, ErrAssignmentNotFound
	}
	row := pool.QueryRow(ctx, `
		SELECT id, assignment_id, user_id, session_id, attempt_no, score, submitted_at, is_late
		FROM quizgame.assignment_attempts
		WHERE assignment_id = $1 AND user_id = $2 AND submitted_at IS NULL
		ORDER BY attempt_no DESC LIMIT 1`, aid, userID)
	var at AssignmentAttempt
	var id, asid, uid, sid uuid.UUID
	err = row.Scan(&id, &asid, &uid, &sid, &at.AttemptNo, &at.Score, &at.SubmittedAt, &at.IsLate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	at.ID = id.String()
	at.AssignmentID = asid.String()
	at.UserID = uid.String()
	at.SessionID = sid.String()
	return &at, nil
}

// StartAssignmentAttempt creates a homework session + attempt (or resumes open) (FR-7, AC-5).
func StartAssignmentAttempt(ctx context.Context, pool *pgxpool.Pool, courseCode string, assignmentID string, userID uuid.UUID, nickname string) (*AssignmentAttempt, *Session, *AddPlayerResult, error) {
	a, err := GetAssignmentByCourse(ctx, pool, courseCode, assignmentID)
	if err != nil {
		return nil, nil, nil, err
	}
	now := time.Now().UTC()
	win, allowed, err := AssignmentWindowForUser(ctx, pool, a, userID, now)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := engine.CheckPlayWindow(win, now); err != nil {
		return nil, nil, nil, err
	}

	if open, err := FindOpenAttempt(ctx, pool, assignmentID, userID); err != nil {
		return nil, nil, nil, err
	} else if open != nil {
		sess, err := GetSession(ctx, pool, open.SessionID)
		if err != nil {
			return nil, nil, nil, err
		}
		p, err := GetPlayerByUser(ctx, pool, open.SessionID, userID)
		if err != nil {
			return nil, nil, nil, err
		}
		raw, err := RotatePlayerToken(ctx, pool, p.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		return open, sess, &AddPlayerResult{Player: *p, PlayerToken: raw, Rejoined: true}, nil
	}

	used, err := CountAttempts(ctx, pool, assignmentID, userID)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := engine.CheckAttempts(used, allowed); err != nil {
		return nil, nil, nil, err
	}

	shuffle := a.Shuffle
	paced := &engine.PacedConfig{Shuffle: shuffle, PerQuestionTimers: true}
	sess, err := CreateGame(ctx, pool, CreateGameInput{
		CourseCode:     courseCode,
		KitID:          a.KitID,
		HostID:         userID,
		Pacing:         string(engine.PacingManual),
		Mode:           string(engine.ModeHomework),
		PacedConfig:    paced,
		ScoringProfile: a.ScoringProfile,
		ScoringConfig:  scoring.ParseConfigJSON(a.ScoringConfig),
		NoJoinCode:     true,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	// Mark session running for homework (single-player).
	_, _ = pool.Exec(ctx, `
		UPDATE quizgame.sessions SET status = 'running', started_at = COALESCE(started_at, NOW()),
			current_phase = 'question_open', current_index = 0
		WHERE id = $1::uuid`, sess.ID)

	nick := nickname
	if nick == "" {
		nick = "Player"
	}
	uid := userID
	join, err := AddPlayer(ctx, pool, AddPlayerInput{
		SessionID: sess.ID,
		UserID:    &uid,
		Nickname:  nick,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	// Init paced progress for the player.
	order := engine.SequentialQuestionOrder(len(sess.KitSnapshot.Questions))
	if shuffle {
		order = engine.ShuffleQuestionOrder(len(sess.KitSnapshot.Questions), nil)
	}
	if err := InitPlayerPacedProgress(ctx, pool, join.Player.ID, order, nil); err != nil {
		return nil, nil, nil, err
	}

	aid, _ := uuid.Parse(assignmentID)
	var attemptID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO quizgame.assignment_attempts (assignment_id, user_id, session_id, attempt_no, is_late)
		VALUES ($1, $2, $3::uuid, $4, $5)
		RETURNING id`,
		aid, userID, sess.ID, used+1, engine.IsLate(win, now),
	).Scan(&attemptID)
	if err != nil {
		return nil, nil, nil, err
	}
	at, err := GetAttempt(ctx, pool, attemptID.String())
	if err != nil {
		return nil, nil, nil, err
	}
	sess, _ = GetSession(ctx, pool, sess.ID)
	return at, sess, join, nil
}

// SubmitAssignmentAttempt finalises a homework run and writes gradebook score (AC-6).
func SubmitAssignmentAttempt(ctx context.Context, pool *pgxpool.Pool, courseCode, assignmentID, attemptID string, userID uuid.UUID) (*AssignmentAttempt, float64, error) {
	a, err := GetAssignmentByCourse(ctx, pool, courseCode, assignmentID)
	if err != nil {
		return nil, 0, err
	}
	at, err := GetAttempt(ctx, pool, attemptID)
	if err != nil || at == nil {
		return nil, 0, ErrAssignmentNotFound
	}
	if at.AssignmentID != a.ID || at.UserID != userID.String() {
		return nil, 0, ErrAssignmentNotFound
	}
	if at.SubmittedAt != nil {
		grade, _ := GetAssignmentGrade(ctx, pool, a.ID, userID)
		return at, grade, nil
	}
	now := time.Now().UTC()
	win, _, _ := AssignmentWindowForUser(ctx, pool, a, userID, now)
	if err := engine.CheckPlayWindow(win, now); err != nil && !errors.Is(err, engine.ErrClosed) {
		// Allow submit if already in progress even near close; refuse if never opened.
		if errors.Is(err, engine.ErrNotYetOpen) {
			return nil, 0, err
		}
	}
	late := engine.IsLate(win, now)

	player, err := GetPlayerByUser(ctx, pool, at.SessionID, userID)
	if err != nil {
		return nil, 0, err
	}
	score := player.TotalScore

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		UPDATE quizgame.assignment_attempts
		SET score = $2, submitted_at = $3, is_late = $4
		WHERE id = $1::uuid`, at.ID, score, now, late)
	if err != nil {
		return nil, 0, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE quizgame.sessions SET status = 'ended', current_phase = 'ended',
			ended_at = COALESCE(ended_at, $2), join_code = NULL
		WHERE id = $1::uuid`, at.SessionID, now)
	if err != nil {
		return nil, 0, err
	}
	_, _ = tx.Exec(ctx, `
		UPDATE quizgame.session_players SET finished_at = COALESCE(finished_at, $2), current_phase = 'ended'
		WHERE id = $1::uuid`, player.ID, now)

	if err := tx.Commit(ctx); err != nil {
		return nil, 0, err
	}

	attempts, err := ListAttemptsForUser(ctx, pool, a.ID, userID)
	if err != nil {
		return nil, 0, err
	}
	scores := make([]int, 0, len(attempts))
	for _, x := range attempts {
		if x.SubmittedAt != nil || x.ID == at.ID {
			s := x.Score
			if x.ID == at.ID {
				s = score
			}
			scores = append(scores, s)
		}
	}
	grade := engine.ApplyGradePolicy(scores, engine.GradePolicy(a.GradePolicy))
	if err := UpsertAssignmentGrade(ctx, pool, a.ID, userID, grade, a.GradePolicy); err != nil {
		return nil, 0, err
	}
	out, err := GetAttempt(ctx, pool, attemptID)
	return out, grade, err
}

// UpsertAssignmentGrade writes the policy-applied gradebook score.
func UpsertAssignmentGrade(ctx context.Context, pool *pgxpool.Pool, assignmentID string, userID uuid.UUID, score float64, policy string) error {
	aid, err := uuid.Parse(assignmentID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO quizgame.assignment_grades (assignment_id, user_id, score, policy, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (assignment_id, user_id) DO UPDATE
		SET score = EXCLUDED.score, policy = EXCLUDED.policy, updated_at = NOW()`,
		aid, userID, score, policy)
	return err
}

// GetAssignmentGrade returns the stored gradebook score (0 if none).
func GetAssignmentGrade(ctx context.Context, pool *pgxpool.Pool, assignmentID string, userID uuid.UUID) (float64, error) {
	aid, err := uuid.Parse(assignmentID)
	if err != nil {
		return 0, err
	}
	var score float64
	err = pool.QueryRow(ctx, `
		SELECT score FROM quizgame.assignment_grades
		WHERE assignment_id = $1 AND user_id = $2`, aid, userID).Scan(&score)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return score, err
}
