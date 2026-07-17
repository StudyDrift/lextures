package board

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserBoardExport is the DSAR export slice for collaboration boards (VC.10 FR-6).
type UserBoardExport struct {
	Posts       []map[string]any `json:"posts"`
	Comments    []map[string]any `json:"comments"`
	Reactions   []map[string]any `json:"reactions"`
	Attachments []map[string]any `json:"attachments"`
}

// ExportUserContent collects a user's board posts, comments, reactions, and attachments.
func ExportUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (UserBoardExport, error) {
	out := UserBoardExport{
		Posts:       []map[string]any{},
		Comments:    []map[string]any{},
		Reactions:   []map[string]any{},
		Attachments: []map[string]any{},
	}

	prows, err := pool.Query(ctx, `
		SELECT p.id, p.board_id, c.course_code, p.content_type, p.title, p.body, p.link_url,
		       p.status, p.created_at, p.updated_at
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE p.author_id = $1
		ORDER BY p.created_at ASC
	`, userID)
	if err != nil {
		return out, err
	}
	defer prows.Close()
	for prows.Next() {
		var id, boardID uuid.UUID
		var courseCode, contentType, title, status string
		var body []byte
		var linkURL *string
		var createdAt, updatedAt time.Time
		if err := prows.Scan(&id, &boardID, &courseCode, &contentType, &title, &body, &linkURL, &status, &createdAt, &updatedAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":          id.String(),
			"boardId":     boardID.String(),
			"courseCode":  courseCode,
			"contentType": contentType,
			"title":       title,
			"status":      status,
			"createdAt":   createdAt.UTC().Format(time.RFC3339),
			"updatedAt":   updatedAt.UTC().Format(time.RFC3339),
		}
		if len(body) > 0 {
			var parsed any
			if json.Unmarshal(body, &parsed) == nil {
				row["body"] = parsed
			}
		}
		if linkURL != nil {
			row["linkUrl"] = *linkURL
		}
		out.Posts = append(out.Posts, row)
	}
	if err := prows.Err(); err != nil {
		return out, err
	}

	crows, err := pool.Query(ctx, `
		SELECT c.id, c.post_id, p.board_id, co.course_code, c.body, c.created_at
		FROM board.post_comments c
		INNER JOIN board.posts p ON p.id = c.post_id
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses co ON co.id = b.course_id
		WHERE c.author_id = $1
		ORDER BY c.created_at ASC
	`, userID)
	if err != nil {
		return out, err
	}
	defer crows.Close()
	for crows.Next() {
		var id, postID, boardID uuid.UUID
		var courseCode string
		var body []byte
		var createdAt time.Time
		if err := crows.Scan(&id, &postID, &boardID, &courseCode, &body, &createdAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"id":         id.String(),
			"postId":     postID.String(),
			"boardId":    boardID.String(),
			"courseCode": courseCode,
			"createdAt":  createdAt.UTC().Format(time.RFC3339),
		}
		if len(body) > 0 {
			var parsed any
			if json.Unmarshal(body, &parsed) == nil {
				row["body"] = parsed
			}
		}
		out.Comments = append(out.Comments, row)
	}
	if err := crows.Err(); err != nil {
		return out, err
	}

	rrows, err := pool.Query(ctx, `
		SELECT r.post_id, p.board_id, c.course_code, r.kind, r.value, r.created_at
		FROM board.post_reactions r
		INNER JOIN board.posts p ON p.id = r.post_id
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE r.user_id = $1
		ORDER BY r.created_at ASC
	`, userID)
	if err != nil {
		return out, err
	}
	defer rrows.Close()
	for rrows.Next() {
		var postID, boardID uuid.UUID
		var courseCode, kind string
		var value *float64
		var createdAt time.Time
		if err := rrows.Scan(&postID, &boardID, &courseCode, &kind, &value, &createdAt); err != nil {
			return out, err
		}
		row := map[string]any{
			"postId":     postID.String(),
			"boardId":    boardID.String(),
			"courseCode": courseCode,
			"kind":       kind,
			"createdAt":  createdAt.UTC().Format(time.RFC3339),
		}
		if value != nil {
			row["value"] = *value
		}
		out.Reactions = append(out.Reactions, row)
	}
	if err := rrows.Err(); err != nil {
		return out, err
	}

	arows, err := pool.Query(ctx, `
		SELECT a.id, a.board_id, c.course_code, a.storage_key, a.file_name, a.mime_type,
		       a.size_bytes, a.alt_text, a.created_at
		FROM board.post_attachments a
		INNER JOIN board.boards b ON b.id = a.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE a.uploaded_by = $1
		ORDER BY a.created_at ASC
	`, userID)
	if err != nil {
		return out, err
	}
	defer arows.Close()
	for arows.Next() {
		var id, boardID uuid.UUID
		var courseCode, storageKey, fileName, mimeType, altText string
		var sizeBytes int64
		var createdAt time.Time
		if err := arows.Scan(&id, &boardID, &courseCode, &storageKey, &fileName, &mimeType, &sizeBytes, &altText, &createdAt); err != nil {
			return out, err
		}
		out.Attachments = append(out.Attachments, map[string]any{
			"id":         id.String(),
			"boardId":    boardID.String(),
			"courseCode": courseCode,
			"storageKey": storageKey,
			"fileName":   fileName,
			"mimeType":   mimeType,
			"sizeBytes":  sizeBytes,
			"altText":    altText,
			"createdAt":  createdAt.UTC().Format(time.RFC3339),
		})
	}
	return out, arows.Err()
}

// ErasureAttachment is a storage key to delete during DSAR erasure.
type ErasureAttachment struct {
	StorageKey string
	SizeBytes  int64
	CourseID   uuid.UUID
	TenantID   uuid.UUID
	UploadedBy uuid.UUID
}

// ListUserAttachmentsForErasure returns attachment blobs owned by the user.
func ListUserAttachmentsForErasure(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]ErasureAttachment, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.storage_key, a.size_bytes, b.course_id, c.org_id, a.uploaded_by
		FROM board.post_attachments a
		INNER JOIN board.boards b ON b.id = a.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE a.uploaded_by = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ErasureAttachment, 0)
	for rows.Next() {
		var e ErasureAttachment
		if err := rows.Scan(&e.StorageKey, &e.SizeBytes, &e.CourseID, &e.TenantID, &e.UploadedBy); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EraseUserContent removes or anonymises a user's board contributions (VC.10 FR-6 / AC-7).
// Attachment rows are deleted; callers must delete object-store blobs beforehand or after.
func EraseUserContent(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	// Detach posts from attachments we are about to delete.
	if _, err := pool.Exec(ctx, `
		UPDATE board.posts SET attachment_id = NULL
		WHERE attachment_id IN (
			SELECT id FROM board.post_attachments WHERE uploaded_by = $1
		)
	`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM board.post_attachments WHERE uploaded_by = $1`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM board.post_reactions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM board.post_comments WHERE author_id = $1`, userID); err != nil {
		return err
	}
	// Anonymise posts rather than hard-delete so board structure remains intact for peers.
	if _, err := pool.Exec(ctx, `
		UPDATE board.posts
		SET author_id = NULL,
		    title = CASE WHEN title = '' THEN title ELSE '[redacted]' END,
		    body = CASE WHEN body IS NULL THEN NULL ELSE '"[redacted]"'::jsonb END,
		    link_url = NULL,
		    link_preview = NULL,
		    drawing_data = NULL,
		    guest_display_name = '',
		    updated_at = NOW()
		WHERE author_id = $1
	`, userID); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `DELETE FROM board.board_members WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return nil
}

// CountUserContentRows returns remaining identifiable rows for verification after erasure.
func CountUserContentRows(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int64, error) {
	var n int64
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM board.posts WHERE author_id = $1) +
			(SELECT COUNT(*) FROM board.post_comments WHERE author_id = $1) +
			(SELECT COUNT(*) FROM board.post_reactions WHERE user_id = $1) +
			(SELECT COUNT(*) FROM board.post_attachments WHERE uploaded_by = $1)
	`, userID).Scan(&n)
	return n, err
}
