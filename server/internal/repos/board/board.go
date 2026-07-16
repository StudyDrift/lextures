// Package board persists course-scoped visual collaboration boards (plan VC.1 / VC.3).
package board

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/repos/organization"
)

const maxTitleLen = 200
const maxSlugAttempts = 8

// Board is one collaboration board row.
type Board struct {
	ID               string          `json:"id"`
	CourseID         string          `json:"courseId"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Slug             string          `json:"slug"`
	Archived         bool            `json:"archived"`
	Layout           string          `json:"layout"`
	LayoutLocked     bool            `json:"layoutLocked"`
	Settings         json.RawMessage `json:"settings,omitempty"`
	ReactionMode     string          `json:"reactionMode"`
	AssignmentID     *string         `json:"assignmentId,omitempty"`
	Visibility       string          `json:"visibility"`
	VisibilityTarget *string         `json:"visibilityTarget,omitempty"`
	Attribution      string          `json:"attribution"`
	CanPost          bool            `json:"canPost"`
	CanInteract      bool            `json:"canInteract"`
	CanArrange       bool            `json:"canArrange"`
	ModerationMode   string          `json:"moderationMode"`
	FilterAction     string          `json:"filterAction"`
	Locked           bool            `json:"locked"`
	FrozenUntil      *time.Time      `json:"frozenUntil,omitempty"`
	CreatedBy        *string         `json:"createdBy"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// PatchBoardInput is a partial update for board metadata and layout (VC.3 / VC.5 / VC.6).
type PatchBoardInput struct {
	Title        *string
	Description  *string
	Archived     *bool
	Layout       *string
	LayoutLocked *bool
	Settings     json.RawMessage
	ReactionMode *string
	// AssignmentID updates the optional gradebook link. Empty string clears it.
	AssignmentID *string
	Visibility       *string
	VisibilityTarget *string // empty string clears
	Attribution      *string
	CanPost          *bool
	CanInteract      *bool
	CanArrange       *bool
	ModerationMode   *string
	FilterAction     *string
	Locked           *bool
	// FrozenUntil updates freeze state. Empty string clears; RFC3339 sets; nil leaves unchanged.
	FrozenUntil *string
}

func scanBoard(row pgx.Row) (Board, error) {
	var b Board
	var id, courseID uuid.UUID
	var createdBy, assignmentID, visTarget uuid.NullUUID
	var settings []byte
	var frozenUntil *time.Time
	if err := row.Scan(
		&id, &courseID, &b.Title, &b.Description, &b.Slug, &b.Archived,
		&b.Layout, &b.LayoutLocked, &settings, &b.ReactionMode, &assignmentID,
		&b.Visibility, &visTarget, &b.Attribution, &b.CanPost, &b.CanInteract, &b.CanArrange,
		&b.ModerationMode, &b.FilterAction, &b.Locked, &frozenUntil,
		&createdBy, &b.CreatedAt, &b.UpdatedAt,
	); err != nil {
		return Board{}, err
	}
	b.ID = id.String()
	b.CourseID = courseID.String()
	b.FrozenUntil = frozenUntil
	if createdBy.Valid {
		s := createdBy.UUID.String()
		b.CreatedBy = &s
	}
	if assignmentID.Valid {
		s := assignmentID.UUID.String()
		b.AssignmentID = &s
	}
	if visTarget.Valid {
		s := visTarget.UUID.String()
		b.VisibilityTarget = &s
	}
	if len(settings) > 0 {
		b.Settings = json.RawMessage(settings)
	} else {
		b.Settings = json.RawMessage(`{}`)
	}
	if b.Layout == "" {
		b.Layout = LayoutWall
	}
	if b.ReactionMode == "" {
		b.ReactionMode = ReactionModeNone
	}
	if b.Visibility == "" {
		b.Visibility = VisibilityCourse
	}
	if b.Attribution == "" {
		b.Attribution = AttributionNamed
	}
	if b.ModerationMode == "" {
		b.ModerationMode = ModerationOpen
	}
	if b.FilterAction == "" {
		b.FilterAction = FilterFlag
	}
	return b, nil
}

func selectBoardCols() string {
	return `b.id, b.course_id, b.title, b.description, b.slug, b.archived,
		b.layout, b.layout_locked, b.settings, b.reaction_mode, b.assignment_id,
		b.visibility, b.visibility_target, b.attribution, b.can_post, b.can_interact, b.can_arrange,
		b.moderation_mode, b.filter_action, b.locked, b.frozen_until,
		b.created_by, b.created_at, b.updated_at`
}

// List returns boards for a course, newest updated first.
// When includeArchived is false, archived boards are excluded.
func List(ctx context.Context, pool *pgxpool.Pool, courseCode string, includeArchived bool) ([]Board, error) {
	q := `
		SELECT ` + selectBoardCols() + `
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1`
	if !includeArchived {
		q += ` AND b.archived = FALSE`
	}
	q += ` ORDER BY b.updated_at DESC`

	rows, err := pool.Query(ctx, q, courseCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Board, 0)
	for rows.Next() {
		b, err := scanBoard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Get returns a single board by id within a course, or nil if not found.
func Get(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) (*Board, error) {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT `+selectBoardCols()+`
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
	`, courseCode, id)
	b, err := scanBoard(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

// Create inserts a board with a unique slug derived from title.
func Create(ctx context.Context, pool *pgxpool.Pool, courseCode string, createdBy uuid.UUID, title, description string) (*Board, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("board: title is required")
	}
	if len(title) > maxTitleLen {
		return nil, fmt.Errorf("board: title must be at most %d characters", maxTitleLen)
	}
	if description == "" {
		description = ""
	}

	baseSlug := organization.SuggestSlugFromName(title)
	if baseSlug == "" {
		baseSlug = "board"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var courseID uuid.UUID
	if err := tx.QueryRow(ctx, `
		SELECT id FROM course.courses WHERE course_code = $1
	`, courseCode).Scan(&courseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var created *Board
	for attempt := 0; attempt < maxSlugAttempts; attempt++ {
		slug := baseSlug
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
			if len(slug) > 48 {
				slug = slug[:48]
			}
		}
		row := tx.QueryRow(ctx, `
			INSERT INTO board.boards (course_id, title, description, slug, created_by)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, course_id, title, description, slug, archived,
				layout, layout_locked, settings, reaction_mode, assignment_id,
				visibility, visibility_target, attribution, can_post, can_interact, can_arrange,
				moderation_mode, filter_action, locked, frozen_until,
				created_by, created_at, updated_at
		`, courseID, title, description, slug, createdBy)
		b, err := scanBoard(row)
		if err == nil {
			created = &b
			break
		}
		if !isUniqueViolation(err) {
			return nil, err
		}
	}
	if created == nil {
		return nil, fmt.Errorf("board: could not allocate unique slug")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

// Patch updates title, description, archived, and/or layout fields.
func Patch(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string, in PatchBoardInput) (*Board, error) {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return nil, fmt.Errorf("board: title is required")
		}
		if len(t) > maxTitleLen {
			return nil, fmt.Errorf("board: title must be at most %d characters", maxTitleLen)
		}
		in.Title = &t
	}
	var layout *string
	if in.Layout != nil {
		norm, err := NormalizeLayout(*in.Layout)
		if err != nil {
			return nil, err
		}
		layout = &norm
	}
	var reactionMode *string
	if in.ReactionMode != nil {
		norm, err := NormalizeReactionMode(*in.ReactionMode)
		if err != nil {
			return nil, err
		}
		reactionMode = &norm
	}
	var settings any
	if len(in.Settings) > 0 && string(in.Settings) != "null" {
		if !json.Valid(in.Settings) {
			return nil, fmt.Errorf("board: settings must be valid JSON")
		}
		settings = in.Settings
	}
	clearAssignment := false
	var assignmentUUID *uuid.UUID
	if in.AssignmentID != nil {
		raw := strings.TrimSpace(*in.AssignmentID)
		if raw == "" {
			clearAssignment = true
		} else {
			parsed, err := uuid.Parse(raw)
			if err != nil {
				return nil, fmt.Errorf("board: invalid assignment_id")
			}
			assignmentUUID = &parsed
		}
	}
	var visibility *string
	if in.Visibility != nil {
		norm, err := NormalizeVisibility(*in.Visibility)
		if err != nil {
			return nil, err
		}
		visibility = &norm
	}
	var attribution *string
	if in.Attribution != nil {
		norm, err := NormalizeAttribution(*in.Attribution)
		if err != nil {
			return nil, err
		}
		attribution = &norm
	}
	clearVisTarget := false
	var visTargetUUID *uuid.UUID
	if in.VisibilityTarget != nil {
		raw := strings.TrimSpace(*in.VisibilityTarget)
		if raw == "" {
			clearVisTarget = true
		} else {
			parsed, err := uuid.Parse(raw)
			if err != nil {
				return nil, fmt.Errorf("board: invalid visibility_target")
			}
			visTargetUUID = &parsed
		}
	}
	var moderationMode *string
	if in.ModerationMode != nil {
		norm, err := NormalizeModerationMode(*in.ModerationMode)
		if err != nil {
			return nil, err
		}
		moderationMode = &norm
	}
	var filterAction *string
	if in.FilterAction != nil {
		norm, err := NormalizeFilterAction(*in.FilterAction)
		if err != nil {
			return nil, err
		}
		filterAction = &norm
	}
	clearFrozen := false
	var frozenUntil *time.Time
	if in.FrozenUntil != nil {
		raw := strings.TrimSpace(*in.FrozenUntil)
		if raw == "" {
			clearFrozen = true
		} else {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				return nil, fmt.Errorf("board: frozen_until must be RFC3339")
			}
			frozenUntil = &parsed
		}
	}

	row := pool.QueryRow(ctx, `
		UPDATE board.boards b
		SET
			title = COALESCE($3, b.title),
			description = COALESCE($4, b.description),
			archived = COALESCE($5, b.archived),
			layout = COALESCE($6, b.layout),
			layout_locked = COALESCE($7, b.layout_locked),
			settings = COALESCE($8, b.settings),
			reaction_mode = COALESCE($9, b.reaction_mode),
			assignment_id = CASE
				WHEN $10::boolean THEN NULL
				WHEN $11::uuid IS NOT NULL THEN $11::uuid
				ELSE b.assignment_id
			END,
			visibility = COALESCE($12, b.visibility),
			visibility_target = CASE
				WHEN $13::boolean THEN NULL
				WHEN $14::uuid IS NOT NULL THEN $14::uuid
				ELSE b.visibility_target
			END,
			attribution = COALESCE($15, b.attribution),
			can_post = COALESCE($16, b.can_post),
			can_interact = COALESCE($17, b.can_interact),
			can_arrange = COALESCE($18, b.can_arrange),
			moderation_mode = COALESCE($19, b.moderation_mode),
			filter_action = COALESCE($20, b.filter_action),
			locked = COALESCE($21, b.locked),
			frozen_until = CASE
				WHEN $22::boolean THEN NULL
				WHEN $23::timestamptz IS NOT NULL THEN $23::timestamptz
				ELSE b.frozen_until
			END,
			updated_at = NOW()
		FROM course.courses c
		WHERE c.id = b.course_id AND c.course_code = $1 AND b.id = $2
		RETURNING b.id, b.course_id, b.title, b.description, b.slug, b.archived,
			b.layout, b.layout_locked, b.settings, b.reaction_mode, b.assignment_id,
			b.visibility, b.visibility_target, b.attribution, b.can_post, b.can_interact, b.can_arrange,
			b.moderation_mode, b.filter_action, b.locked, b.frozen_until,
			b.created_by, b.created_at, b.updated_at
	`, courseCode, id, in.Title, in.Description, in.Archived, layout, in.LayoutLocked, settings,
		reactionMode, clearAssignment, assignmentUUID,
		visibility, clearVisTarget, visTargetUUID, attribution, in.CanPost, in.CanInteract, in.CanArrange,
		moderationMode, filterAction, in.Locked, clearFrozen, frozenUntil)
	b, err := scanBoard(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Switching to columns ensures an Unsorted section and assigns unsectioned cards (AC-1).
	if layout != nil && *layout == LayoutColumns {
		if _, err := EnsureUnsortedSection(ctx, pool, courseCode, boardID); err != nil {
			return nil, err
		}
		if err := AssignUnsectionedToUnsorted(ctx, pool, courseCode, boardID); err != nil {
			return nil, err
		}
	}
	return &b, nil
}

// SoftDelete sets archived = true.
func SoftDelete(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) (*Board, error) {
	archived := true
	return Patch(ctx, pool, courseCode, boardID, PatchBoardInput{Archived: &archived})
}

// HardDelete permanently removes a board. Returns true if a row was deleted.
func HardDelete(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) (bool, error) {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return false, nil
	}
	tag, err := pool.Exec(ctx, `
		DELETE FROM board.boards b
		USING course.courses c
		WHERE c.id = b.course_id AND c.course_code = $1 AND b.id = $2
	`, courseCode, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps pgconn.PgError; string match avoids importing pgconn in every path.
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "23505")
}
