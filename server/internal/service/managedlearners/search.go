package managedlearners

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SearchHit is a managed learner the actor can already see on the Learners page.
type SearchHit struct {
	UserID      uuid.UUID
	DisplayName string
	GradeLevel  *string
	Score       float64
}

// Search finds the actor's managed learners by display name.
// Learners are included before they are enrolled in a course.
// A non-empty scopeCourseCode keeps only learners enrolled in that course.
// ErrForbidden and ErrActorBlocked mean the actor has no learner list; callers may treat that as no hits.
func Search(
	ctx context.Context,
	pool *pgxpool.Pool,
	actorID uuid.UUID,
	q string,
	scopeCourseCode *string,
	limit int,
) ([]SearchHit, int, error) {
	q = strings.TrimSpace(q)
	if q == "" || limit <= 0 {
		return nil, 0, nil
	}
	if limit > 50 {
		limit = 50
	}
	_, orgID, err := assertCanManageDependents(ctx, pool, actorID)
	if err != nil {
		return nil, 0, err
	}

	rows, err := pool.Query(ctx, `
SELECT u.id, COALESCE(u.display_name, ''), u.grade_level,
       CASE
           WHEN lower(btrim(u.display_name)) = lower($2) THEN 0.95
           ELSE 0.55
       END AS rank,
       COUNT(*) OVER () AS total
FROM "user".parent_student_links l
INNER JOIN "user".users u ON u.id = l.student_user_id
WHERE l.parent_user_id = $1
  AND l.org_id = $4
  AND l.status IN ('active', 'pending')
  AND u.account_type = 'managed'
  AND btrim(COALESCE(u.display_name, '')) <> ''
  AND u.display_name ILIKE $3 ESCAPE E'\\'
  AND (
      $5::text IS NULL
      OR EXISTS (
          SELECT 1
          FROM course.course_enrollments ce
          INNER JOIN course.courses c ON c.id = ce.course_id AND c.archived = false
          WHERE ce.user_id = u.id
            AND ce.active
            AND lower(c.course_code) = lower($5)
      )
  )
ORDER BY rank DESC, u.display_name ASC
LIMIT $6
`, actorID, q, containsLikePattern(q), orgID, scopeCourseCode, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []SearchHit
	var total int
	for rows.Next() {
		var hit SearchHit
		if err := rows.Scan(&hit.UserID, &hit.DisplayName, &hit.GradeLevel, &hit.Score, &total); err != nil {
			return nil, 0, err
		}
		hit.DisplayName = strings.TrimSpace(hit.DisplayName)
		if hit.DisplayName == "" {
			continue
		}
		out = append(out, hit)
	}
	return out, total, rows.Err()
}

// IgnoreSearchAccessErr reports actor states that simply have no learner directory.
func IgnoreSearchAccessErr(err error) bool {
	return errors.Is(err, ErrForbidden) || errors.Is(err, ErrActorBlocked)
}

func containsLikePattern(q string) string {
	var b strings.Builder
	b.Grow(len(q) + 2)
	b.WriteByte('%')
	for _, r := range q {
		switch r {
		case '\\', '%', '_':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('%')
	return b.String()
}
