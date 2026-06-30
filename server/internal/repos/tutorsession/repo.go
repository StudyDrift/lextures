// Package tutorsession persists named AI tutor sessions and messages (plan 19.1).
package tutorsession

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultRetentionDays = 365

// Citation is a structured source reference stored on assistant messages.
type Citation struct {
	SourceID string `json:"sourceId"`
	ChunkID  string `json:"chunkId"`
	Excerpt  string `json:"excerpt"`
	Title    string `json:"title,omitempty"`
}

// Session is one named tutor conversation.
type Session struct {
	ID         uuid.UUID `json:"id"`
	StudentID  uuid.UUID `json:"-"`
	CourseID   uuid.UUID `json:"-"`
	Title      *string   `json:"title,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	LastActive time.Time `json:"lastActive"`
}

// Message is one chat turn within a session.
type Message struct {
	ID          uuid.UUID  `json:"id"`
	Role        string     `json:"role"`
	Content     string     `json:"content"`
	Citations   []Citation `json:"citations,omitempty"`
	ConceptTags []string   `json:"conceptTags,omitempty"`
	TokenCount  *int       `json:"tokenCount,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// ConceptConfusionSummary aggregates concept tags from tutor messages (instructor view).
type ConceptConfusionSummary struct {
	ConceptID   string `json:"conceptId"`
	ConceptName string `json:"conceptName"`
	Count       int    `json:"count"`
}

// GetAITutorOptOut returns whether the student has opted out of the AI tutor.
func GetAITutorOptOut(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (bool, error) {
	var optedOut bool
	err := pool.QueryRow(ctx, `
SELECT ai_tutor_opt_out FROM "user".users WHERE id = $1
`, userID).Scan(&optedOut)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return optedOut, err
}

// SetAITutorOptOut updates ai_tutor_opt_out for the user.
func SetAITutorOptOut(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, optedOut bool) error {
	tag, err := pool.Exec(ctx, `
UPDATE "user".users SET ai_tutor_opt_out = $2 WHERE id = $1
`, userID, optedOut)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListSessions returns sessions for a student in a course, most recent first.
func ListSessions(ctx context.Context, pool *pgxpool.Pool, studentID, courseID uuid.UUID) ([]Session, error) {
	rows, err := pool.Query(ctx, `
SELECT id, student_id, course_id, title, created_at, last_active
FROM course.tutor_sessions
WHERE student_id = $1 AND course_id = $2
ORDER BY last_active DESC
`, studentID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []Session{}
	}
	return out, rows.Err()
}

// CreateSession inserts a new tutor session.
func CreateSession(ctx context.Context, pool *pgxpool.Pool, studentID, courseID uuid.UUID, title *string) (Session, error) {
	row := pool.QueryRow(ctx, `
INSERT INTO course.tutor_sessions (student_id, course_id, title)
VALUES ($1, $2, $3)
RETURNING id, student_id, course_id, title, created_at, last_active
`, studentID, courseID, title)
	return scanSession(row)
}

// GetSession loads a session owned by the student.
func GetSession(ctx context.Context, pool *pgxpool.Pool, sessionID, studentID, courseID uuid.UUID) (*Session, error) {
	row := pool.QueryRow(ctx, `
SELECT id, student_id, course_id, title, created_at, last_active
FROM course.tutor_sessions
WHERE id = $1 AND student_id = $2 AND course_id = $3
`, sessionID, studentID, courseID)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteSession removes a session and its messages (CASCADE).
func DeleteSession(ctx context.Context, pool *pgxpool.Pool, sessionID, studentID, courseID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
DELETE FROM course.tutor_sessions
WHERE id = $1 AND student_id = $2 AND course_id = $3
`, sessionID, studentID, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// TouchSession updates last_active on a session.
func TouchSession(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
UPDATE course.tutor_sessions SET last_active = NOW() WHERE id = $1
`, sessionID)
	return err
}

// ListRecentMessages returns up to limit messages for context injection (oldest first).
func ListRecentMessages(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := pool.Query(ctx, `
SELECT id, role, content, citations, concept_tags, token_count, created_at
FROM (
    SELECT id, role, content, citations, concept_tags, token_count, created_at
    FROM course.tutor_messages
    WHERE session_id = $1
    ORDER BY created_at DESC
    LIMIT $2
) sub
ORDER BY created_at ASC
`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

// ListAllMessages returns all messages in a session (oldest first).
func ListAllMessages(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) ([]Message, error) {
	rows, err := pool.Query(ctx, `
SELECT id, role, content, citations, concept_tags, token_count, created_at
FROM course.tutor_messages
WHERE session_id = $1
ORDER BY created_at ASC
`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

// AppendMessage inserts one message and returns it.
func AppendMessage(
	ctx context.Context,
	pool *pgxpool.Pool,
	sessionID uuid.UUID,
	role, content string,
	citations []Citation,
	conceptTagIDs []uuid.UUID,
	tokenCount int,
) (Message, error) {
	var citRaw []byte
	if len(citations) > 0 {
		b, err := json.Marshal(citations)
		if err != nil {
			return Message{}, err
		}
		citRaw = b
	}
	var tok *int
	if tokenCount > 0 {
		tok = &tokenCount
	}
	row := pool.QueryRow(ctx, `
INSERT INTO course.tutor_messages (session_id, role, content, citations, concept_tags, token_count)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, role, content, citations, concept_tags, token_count, created_at
`, sessionID, role, content, citRaw, conceptTagIDs, tok)
	return scanMessage(row)
}

// HasSystemDisclosure reports whether the session already has the disclosure system message.
func HasSystemDisclosure(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS(
    SELECT 1 FROM course.tutor_messages
    WHERE session_id = $1 AND role = 'system'
)
`, sessionID).Scan(&exists)
	return exists, err
}

// PurgeExpiredSessions deletes sessions older than retentionDays for an org.
func PurgeExpiredSessions(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = defaultRetentionDays
	}
	tag, err := pool.Exec(ctx, `
DELETE FROM course.tutor_sessions ts
USING course.courses c
WHERE ts.course_id = c.id
  AND c.org_id = $1
  AND ts.last_active < NOW() - ($2 || ' days')::interval
`, orgID, retentionDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListOrgRetentionDays returns tutor_session_retention_days for each org.
func ListOrgRetentionDays(ctx context.Context, pool *pgxpool.Pool) ([]struct {
	OrgID         uuid.UUID
	RetentionDays int
}, error) {
	rows, err := pool.Query(ctx, `
SELECT id, tutor_session_retention_days FROM tenant.organizations
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		OrgID         uuid.UUID
		RetentionDays int
	}
	for rows.Next() {
		var row struct {
			OrgID         uuid.UUID
			RetentionDays int
		}
		if err := rows.Scan(&row.OrgID, &row.RetentionDays); err != nil {
			return nil, err
		}
		if row.RetentionDays <= 0 {
			row.RetentionDays = defaultRetentionDays
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListConceptConfusion returns aggregate concept confusion counts for instructors.
func ListConceptConfusion(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID, since time.Time) ([]ConceptConfusionSummary, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id::text, c.name, COUNT(*)::int
FROM course.tutor_messages tm
INNER JOIN course.tutor_sessions ts ON ts.id = tm.session_id
CROSS JOIN LATERAL unnest(tm.concept_tags) AS tag(concept_id)
INNER JOIN course.concepts c ON c.id = tag.concept_id
WHERE ts.course_id = $1
  AND tm.created_at >= $2
  AND tm.role = 'user'
GROUP BY c.id, c.name
ORDER BY COUNT(*) DESC
LIMIT 20
`, courseID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ConceptConfusionSummary
	for rows.Next() {
		var s ConceptConfusionSummary
		if err := rows.Scan(&s.ConceptID, &s.ConceptName, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []ConceptConfusionSummary{}
	}
	return out, rows.Err()
}

func scanSession(row pgx.Row) (Session, error) {
	var s Session
	var title *string
	if err := row.Scan(&s.ID, &s.StudentID, &s.CourseID, &title, &s.CreatedAt, &s.LastActive); err != nil {
		return Session{}, err
	}
	s.Title = title
	return s, nil
}

func scanMessages(rows pgx.Rows) ([]Message, error) {
	var out []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if out == nil {
		out = []Message{}
	}
	return out, rows.Err()
}

func scanMessage(row pgx.Row) (Message, error) {
	var m Message
	var citRaw []byte
	var conceptIDs []uuid.UUID
	var tok *int
	if err := row.Scan(&m.ID, &m.Role, &m.Content, &citRaw, &conceptIDs, &tok, &m.CreatedAt); err != nil {
		return Message{}, err
	}
	if len(citRaw) > 0 && string(citRaw) != "null" {
		_ = json.Unmarshal(citRaw, &m.Citations)
	}
	if m.Citations == nil {
		m.Citations = []Citation{}
	}
	for _, id := range conceptIDs {
		m.ConceptTags = append(m.ConceptTags, id.String())
	}
	if m.ConceptTags == nil {
		m.ConceptTags = []string{}
	}
	m.TokenCount = tok
	return m, nil
}
