// Package useraudit maps server/src/repos/user_audit.rs.
package useraudit

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StructureItemIsCourseContentPage is true when the structure row exists, belongs to the course, and is a content page.
func StructureItemIsCourseContentPage(ctx context.Context, pool *pgxpool.Pool, courseID, structureItemID uuid.UUID) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM course.course_structure_items
	WHERE id = $1 AND course_id = $2 AND kind = 'content_page'
)
`, structureItemID, courseID).Scan(&ok)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Insert appends a user_audit row. structureItemID is nil for course_visit.
func Insert(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, structureItemID *uuid.UUID, eventKind string) error {
	_, err := pool.Exec(ctx, `
INSERT INTO "user".user_audit (user_id, course_id, structure_item_id, event_kind, occurred_at)
VALUES ($1, $2, $3, $4, NOW())
`, userID, courseID, structureItemID, eventKind)
	return err
}

// InsertCredentialShare records a credential share analytics event (plan 15.6).
// structure_item_id stores the credential id for share events.
func InsertCredentialShare(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID, courseID, credentialID uuid.UUID,
	channel string,
) error {
	eventKind := "credential_share_" + channel
	credID := credentialID
	_, err := pool.Exec(ctx, `
INSERT INTO "user".user_audit (user_id, course_id, structure_item_id, event_kind, occurred_at)
VALUES ($1, $2, $3, $4, NOW())
`, userID, courseID, &credID, eventKind)
	return err
}
