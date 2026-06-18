package course

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PermanentDeleteOutcome lists on-disk or object-store keys to remove after the DB row is deleted.
type PermanentDeleteOutcome struct {
	CourseID                      uuid.UUID
	CourseCode                    string
	RemovedCourseFileStorageKeys  []string
	RemovedFileManagerStorageKeys []string
	RemovedStorageObjectKeys      []string
	RemovedFeedbackMediaKeys      []string
	RemovedFeedbackCaptionKeys    []string
}

// PermanentlyDeleteCourse removes an archived course and all related rows (via CASCADE).
// The course must already be archived. Returns nil,nil when the course is missing or not archived.
func PermanentlyDeleteCourse(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, courseCode string) (*PermanentDeleteOutcome, error) {
	courseID, archived, err := archivedCourseInOrg(ctx, pool, orgID, courseCode)
	if err != nil {
		return nil, err
	}
	if courseID == nil {
		return nil, nil
	}
	if !archived {
		return nil, fmt.Errorf("course is not archived")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fileKeys, err := listStorageKeysForCourseFiles(ctx, tx, *courseID)
	if err != nil {
		return nil, err
	}
	fileManagerKeys, err := listStorageKeysForFileManager(ctx, tx, *courseID)
	if err != nil {
		return nil, err
	}
	storageObjectKeys, err := listStorageObjectKeysForCourse(ctx, tx, *courseID)
	if err != nil {
		return nil, err
	}
	feedbackKeys, captionKeys, err := listFeedbackMediaKeysForCourse(ctx, tx, *courseID)
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM course.courses WHERE id = $1 AND org_id = $2`, *courseID, orgID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &PermanentDeleteOutcome{
		CourseID:                      *courseID,
		CourseCode:                    courseCode,
		RemovedCourseFileStorageKeys:  fileKeys,
		RemovedFileManagerStorageKeys: fileManagerKeys,
		RemovedStorageObjectKeys:      storageObjectKeys,
		RemovedFeedbackMediaKeys:      feedbackKeys,
		RemovedFeedbackCaptionKeys:    captionKeys,
	}, nil
}

func archivedCourseInOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID, courseCode string) (*uuid.UUID, bool, error) {
	var id uuid.UUID
	var archived bool
	err := pool.QueryRow(ctx, `
SELECT id, archived FROM course.courses WHERE course_code = $1 AND org_id = $2
`, courseCode, orgID).Scan(&id, &archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &id, archived, nil
}

func listStorageObjectKeysForCourse(ctx context.Context, tx pgx.Tx, courseID uuid.UUID) ([]string, error) {
	rows, err := tx.Query(ctx, `
SELECT object_key FROM storage.objects
WHERE course_id = $1 AND deleted_at IS NULL
`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStringColumn(rows)
}

func listFeedbackMediaKeysForCourse(ctx context.Context, tx pgx.Tx, courseID uuid.UUID) (mediaKeys []string, captionKeys []string, err error) {
	rows, err := tx.Query(ctx, `
SELECT storage_key, caption_key FROM course.submission_feedback_media
WHERE course_id = $1
`, courseID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var caption *string
		if err := rows.Scan(&key, &caption); err != nil {
			return nil, nil, err
		}
		mediaKeys = append(mediaKeys, key)
		if caption != nil && *caption != "" {
			captionKeys = append(captionKeys, *caption)
		}
	}
	return mediaKeys, captionKeys, rows.Err()
}

func scanStringColumn(rows pgx.Rows) ([]string, error) {
	out := make([]string, 0)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}