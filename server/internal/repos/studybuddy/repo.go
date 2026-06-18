// Package studybuddy persists AI study buddy memory and session context (plan 15.12).
package studybuddy

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionTTL = 7 * 24 * time.Hour

// MemoryRow is the persisted study buddy memory for one user and course.
type MemoryRow struct {
	ID                 uuid.UUID  `json:"id"`
	UserID             uuid.UUID  `json:"userId"`
	CourseID           uuid.UUID  `json:"courseId"`
	GoalsSummary       *string    `json:"goalsSummary,omitempty"`
	StruggleConcepts   []string   `json:"struggleConcepts"`
	LastSessionSummary *string    `json:"lastSessionSummary,omitempty"`
	LastActiveAt       *time.Time `json:"lastActiveAt,omitempty"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// Message is one chat turn in a session.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SessionRow holds current-session messages with expiry.
type SessionRow struct {
	ID        uuid.UUID `json:"id"`
	Messages  []Message `json:"messages"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// CourseContentPage is one published content page used for RAG retrieval.
type CourseContentPage struct {
	ItemID uuid.UUID
	Title  string
	Body   string
}

const memorySelect = `
SELECT id, user_id, course_id, goals_summary, struggle_concepts,
       last_session_summary, last_active_at, updated_at
FROM "user".study_buddy_memory
`

func scanMemory(row pgx.Row) (*MemoryRow, error) {
	var r MemoryRow
	var goals, summary *string
	err := row.Scan(
		&r.ID, &r.UserID, &r.CourseID, &goals, &r.StruggleConcepts,
		&summary, &r.LastActiveAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.GoalsSummary = goals
	r.LastSessionSummary = summary
	if r.StruggleConcepts == nil {
		r.StruggleConcepts = []string{}
	}
	return &r, nil
}

// GetMemory returns memory for a user/course pair, or nil when absent.
func GetMemory(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) (*MemoryRow, error) {
	row, err := scanMemory(pool.QueryRow(ctx, memorySelect+` WHERE user_id = $1 AND course_id = $2`, userID, courseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// UpsertMemory inserts or updates the memory row.
func UpsertMemory(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, goals *string, struggles []string, sessionSummary *string) (*MemoryRow, error) {
	if struggles == nil {
		struggles = []string{}
	}
	now := time.Now().UTC()
	row, err := scanMemory(pool.QueryRow(ctx, `
INSERT INTO "user".study_buddy_memory (
    user_id, course_id, goals_summary, struggle_concepts, last_session_summary, last_active_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $6)
ON CONFLICT (user_id, course_id) DO UPDATE SET
    goals_summary = EXCLUDED.goals_summary,
    struggle_concepts = EXCLUDED.struggle_concepts,
    last_session_summary = COALESCE(EXCLUDED.last_session_summary, "user".study_buddy_memory.last_session_summary),
    last_active_at = EXCLUDED.last_active_at,
    updated_at = EXCLUDED.updated_at
RETURNING id, user_id, course_id, goals_summary, struggle_concepts,
          last_session_summary, last_active_at, updated_at
`, userID, courseID, goals, struggles, sessionSummary, now))
	if err != nil {
		return nil, err
	}
	return row, nil
}

// UpdateSessionSummary stores a rolling session summary on the memory row.
func UpdateSessionSummary(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, summary string) error {
	_, err := pool.Exec(ctx, `
UPDATE "user".study_buddy_memory
SET last_session_summary = $3, last_active_at = NOW(), updated_at = NOW()
WHERE user_id = $1 AND course_id = $2
`, userID, courseID, summary)
	return err
}

// DeleteMemory removes memory for GDPR erasure.
func DeleteMemory(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID) error {
	_, err := pool.Exec(ctx, `
DELETE FROM "user".study_buddy_memory WHERE user_id = $1 AND course_id = $2
`, userID, courseID)
	return err
}

// GetOrCreateSession loads an existing session or creates a new one.
func GetOrCreateSession(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, sessionID uuid.UUID) (*SessionRow, error) {
	if sessionID != uuid.Nil {
		var raw []byte
		var expires time.Time
		err := pool.QueryRow(ctx, `
SELECT messages, expires_at FROM studybuddy.sessions
WHERE id = $1 AND user_id = $2 AND course_id = $3 AND expires_at > NOW()
`, sessionID, userID, courseID).Scan(&raw, &expires)
		if err == nil {
			msgs, err := decodeMessages(raw)
			if err != nil {
				return nil, err
			}
			return &SessionRow{ID: sessionID, Messages: msgs, ExpiresAt: expires}, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	newID := uuid.New()
	expires := time.Now().UTC().Add(sessionTTL)
	_, err := pool.Exec(ctx, `
INSERT INTO studybuddy.sessions (id, user_id, course_id, messages, expires_at)
VALUES ($1, $2, $3, '[]', $4)
`, newID, userID, courseID, expires)
	if err != nil {
		return nil, err
	}
	return &SessionRow{ID: newID, Messages: []Message{}, ExpiresAt: expires}, nil
}

// AppendSessionMessage appends a message to a session and extends expiry.
func AppendSessionMessage(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID, msg Message) error {
	raw, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	expires := time.Now().UTC().Add(sessionTTL)
	_, err = pool.Exec(ctx, `
UPDATE studybuddy.sessions
SET messages = COALESCE(messages, '[]'::jsonb) || jsonb_build_array($2::jsonb),
    expires_at = $3,
    updated_at = NOW()
WHERE id = $1
`, sessionID, raw, expires)
	return err
}

// ListSessionMessages returns decoded messages for a session.
func ListSessionMessages(ctx context.Context, pool *pgxpool.Pool, sessionID uuid.UUID) ([]Message, error) {
	var raw []byte
	err := pool.QueryRow(ctx, `SELECT messages FROM studybuddy.sessions WHERE id = $1`, sessionID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return decodeMessages(raw)
}

func decodeMessages(raw []byte) ([]Message, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []Message{}, nil
	}
	var msgs []Message
	if err := json.Unmarshal(raw, &msgs); err != nil {
		// jsonb array of objects stored via || append may need wrapping
		var arr []json.RawMessage
		if err2 := json.Unmarshal(raw, &arr); err2 != nil {
			return nil, err
		}
		msgs = make([]Message, 0, len(arr))
		for _, item := range arr {
			var m Message
			if err := json.Unmarshal(item, &m); err != nil {
				return nil, err
			}
			msgs = append(msgs, m)
		}
	}
	if msgs == nil {
		msgs = []Message{}
	}
	return msgs, nil
}

// ListCourseContentPages returns published content pages for RAG indexing.
func ListCourseContentPages(ctx context.Context, pool *pgxpool.Pool, courseID uuid.UUID) ([]CourseContentPage, error) {
	rows, err := pool.Query(ctx, `
SELECT c.id, c.title, COALESCE(m.markdown, '')
FROM course.course_structure_items c
INNER JOIN course.module_content_pages m ON m.structure_item_id = c.id
WHERE c.course_id = $1
  AND c.kind = 'content_page'
  AND c.published
  AND NOT c.archived
  AND TRIM(COALESCE(m.markdown, '')) <> ''
ORDER BY c.sort_order
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CourseContentPage
	for rows.Next() {
		var p CourseContentPage
		if err := rows.Scan(&p.ItemID, &p.Title, &p.Body); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LowQuizStruggles returns quiz item titles where the learner scored below thresholdPct.
func LowQuizStruggles(ctx context.Context, pool *pgxpool.Pool, courseID, userID uuid.UUID, thresholdPct float64, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := pool.Query(ctx, `
SELECT DISTINCT ON (qa.structure_item_id) csi.title
FROM course.quiz_attempts qa
INNER JOIN course.course_structure_items csi ON csi.id = qa.structure_item_id
WHERE qa.course_id = $1
  AND qa.student_user_id = $2
  AND qa.status = 'submitted'
  AND qa.score_percent IS NOT NULL
  AND qa.score_percent < $3
  AND qa.submitted_at > NOW() - INTERVAL '14 days'
ORDER BY qa.structure_item_id, qa.submitted_at DESC
LIMIT $4
`, courseID, userID, thresholdPct, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return nil, err
		}
		out = append(out, title)
	}
	return out, rows.Err()
}

// StaleVisitedModules returns module titles not visited in staleDays.
func StaleVisitedModules(ctx context.Context, pool *pgxpool.Pool, enrollmentID, courseID uuid.UUID, staleDays int, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 3
	}
	if staleDays <= 0 {
		staleDays = 5
	}
	rows, err := pool.Query(ctx, `
WITH RECURSIVE leaf AS (
    SELECT id AS leaf_id, id AS cur, parent_id, kind
    FROM course.course_structure_items
    WHERE course_id = $2 AND kind NOT IN ('module', 'heading') AND published AND NOT archived
  UNION ALL
    SELECT l.leaf_id, p.id, p.parent_id, p.kind
    FROM leaf l
    JOIN course.course_structure_items p ON p.id = l.parent_id
    WHERE l.kind <> 'module'
),
leaf_module AS (
    SELECT leaf_id, cur AS module_id FROM leaf WHERE kind = 'module'
)
SELECT DISTINCT m.title
FROM course.learner_item_progress lip
INNER JOIN leaf_module lm ON lm.leaf_id = lip.item_id
INNER JOIN course.course_structure_items m ON m.id = lm.module_id
WHERE lip.enrollment_id = $1
  AND lip.last_visited_at IS NOT NULL
  AND lip.last_visited_at < NOW() - ($3 || ' days')::interval
ORDER BY m.title
LIMIT $4
`, enrollmentID, courseID, staleDays, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			return nil, err
		}
		out = append(out, title)
	}
	return out, rows.Err()
}

// PurgeExpiredSessions removes expired session rows (maintenance hook).
func PurgeExpiredSessions(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `DELETE FROM studybuddy.sessions WHERE expires_at < NOW()`)
	return err
}
