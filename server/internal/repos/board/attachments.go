package board

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Attachment scan statuses persisted on board.post_attachments.
const (
	ScanPending = "pending"
	ScanClean   = "clean"
	ScanBlocked = "blocked"
)

// Attachment is a file stored for a board post.
type Attachment struct {
	ID         string    `json:"id"`
	BoardID    string    `json:"boardId"`
	StorageKey string    `json:"storageKey"`
	FileName   string    `json:"fileName"`
	MimeType   string    `json:"mimeType"`
	SizeBytes  int64     `json:"sizeBytes"`
	AltText    string    `json:"altText"`
	ScanStatus string    `json:"scanStatus"`
	UploadedBy *string   `json:"uploadedBy"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Max attachment sizes (align with course image uploads; VC.10 may tighten).
const (
	MaxImageBytes = 10 * 1024 * 1024
	MaxFileBytes  = 50 * 1024 * 1024
	MaxVideoBytes = 200 * 1024 * 1024
	MaxAudioBytes = 50 * 1024 * 1024
)

func scanAttachment(row pgx.Row) (Attachment, error) {
	var a Attachment
	var id, boardID uuid.UUID
	var uploadedBy uuid.NullUUID
	if err := row.Scan(
		&id, &boardID, &a.StorageKey, &a.FileName, &a.MimeType, &a.SizeBytes,
		&a.AltText, &a.ScanStatus, &uploadedBy, &a.CreatedAt,
	); err != nil {
		return Attachment{}, err
	}
	a.ID = id.String()
	a.BoardID = boardID.String()
	if uploadedBy.Valid {
		s := uploadedBy.UUID.String()
		a.UploadedBy = &s
	}
	return a, nil
}

func selectAttachmentCols() string {
	return `a.id, a.board_id, a.storage_key, a.file_name, a.mime_type, a.size_bytes, a.alt_text, a.scan_status, a.uploaded_by, a.created_at`
}

// CreateAttachment inserts a post attachment row for a board in the given course.
func CreateAttachment(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	uploadedBy uuid.UUID,
	storageKey, fileName, mimeType, altText, scanStatus string,
	sizeBytes int64,
) (*Attachment, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil, fmt.Errorf("board: file_name is required")
	}
	mimeType = strings.TrimSpace(mimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	storageKey = strings.TrimSpace(storageKey)
	if storageKey == "" {
		return nil, fmt.Errorf("board: storage_key is required")
	}
	if sizeBytes <= 0 {
		return nil, fmt.Errorf("board: size_bytes must be positive")
	}
	if scanStatus == "" {
		scanStatus = ScanPending
	}
	switch scanStatus {
	case ScanPending, ScanClean, ScanBlocked:
	default:
		return nil, fmt.Errorf("board: invalid scan_status")
	}

	var insertedID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO board.post_attachments (
			board_id, storage_key, file_name, mime_type, size_bytes, alt_text, scan_status, uploaded_by
		)
		SELECT b.id, $3, $4, $5, $6, $7, $8, $9
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		RETURNING id
	`, courseCode, bid, storageKey, fileName, mimeType, sizeBytes, altText, scanStatus, uploadedBy).Scan(&insertedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return GetAttachment(ctx, pool, courseCode, boardID, insertedID.String())
}

// GetAttachment returns an attachment scoped to course + board.
func GetAttachment(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, attachmentID string) (*Attachment, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	aid, err := uuid.Parse(attachmentID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT `+selectAttachmentCols()+`
		FROM board.post_attachments a
		INNER JOIN board.boards b ON b.id = a.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2 AND a.id = $3
	`, courseCode, bid, aid)
	a, err := scanAttachment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// SetAttachmentScanStatus updates scan_status for an attachment.
func SetAttachmentScanStatus(ctx context.Context, pool *pgxpool.Pool, attachmentID, status string) error {
	aid, err := uuid.Parse(attachmentID)
	if err != nil {
		return fmt.Errorf("board: invalid attachment id")
	}
	switch status {
	case ScanPending, ScanClean, ScanBlocked:
	default:
		return fmt.Errorf("board: invalid scan_status")
	}
	_, err = pool.Exec(ctx, `
		UPDATE board.post_attachments SET scan_status = $2 WHERE id = $1
	`, aid, status)
	return err
}

// BlockedAttachmentRef identifies a post whose attachment was blocked (for moderation queue).
type BlockedAttachmentRef struct {
	BoardID    string
	PostID     string
	CourseCode string
}

// SyncAttachmentScanByStorageKey updates board attachments matching a storage key (AV worker hook).
// When status is blocked, returns posts that should enter the moderation queue.
func SyncAttachmentScanByStorageKey(ctx context.Context, pool *pgxpool.Pool, storageKey, status string) ([]BlockedAttachmentRef, error) {
	switch status {
	case ScanPending, ScanClean, ScanBlocked:
	default:
		return nil, fmt.Errorf("board: invalid scan_status")
	}
	_, err := pool.Exec(ctx, `
		UPDATE board.post_attachments SET scan_status = $2 WHERE storage_key = $1
	`, storageKey, status)
	if err != nil || status != ScanBlocked {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT b.id, p.id, c.course_code
		FROM board.post_attachments a
		INNER JOIN board.posts p ON p.attachment_id = a.id
		INNER JOIN board.boards b ON b.id = a.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE a.storage_key = $1 AND a.scan_status = $2
	`, storageKey, ScanBlocked)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BlockedAttachmentRef, 0)
	for rows.Next() {
		var boardID, postID uuid.UUID
		var courseCode string
		if err := rows.Scan(&boardID, &postID, &courseCode); err != nil {
			return nil, err
		}
		out = append(out, BlockedAttachmentRef{
			BoardID: boardID.String(), PostID: postID.String(), CourseCode: courseCode,
		})
	}
	return out, rows.Err()
}

// AllowedMimeForContent reports whether mime is acceptable for a content type family.
func AllowedMimeForContent(contentHint, mimeType string) bool {
	m := strings.ToLower(strings.TrimSpace(mimeType))
	switch strings.ToLower(strings.TrimSpace(contentHint)) {
	case ContentTypeImage:
		return strings.HasPrefix(m, "image/")
	case ContentTypeAudio:
		return strings.HasPrefix(m, "audio/")
	case ContentTypeVideo:
		return strings.HasPrefix(m, "video/")
	case ContentTypeFile:
		// Broad allow-list for documents; reject executable-ish types.
		if strings.HasPrefix(m, "application/x-msdownload") ||
			strings.HasPrefix(m, "application/x-executable") ||
			m == "application/x-sh" {
			return false
		}
		return true
	default:
		return strings.HasPrefix(m, "image/") ||
			strings.HasPrefix(m, "audio/") ||
			strings.HasPrefix(m, "video/") ||
			strings.HasPrefix(m, "application/") ||
			strings.HasPrefix(m, "text/")
	}
}

// MaxBytesForMime returns the size cap for a MIME type.
func MaxBytesForMime(mimeType string) int64 {
	m := strings.ToLower(mimeType)
	switch {
	case strings.HasPrefix(m, "image/"):
		return MaxImageBytes
	case strings.HasPrefix(m, "video/"):
		return MaxVideoBytes
	case strings.HasPrefix(m, "audio/"):
		return MaxAudioBytes
	default:
		return MaxFileBytes
	}
}

// AttachmentAccessible reports whether the file may be served to clients.
func AttachmentAccessible(a Attachment, avEnabled bool) bool {
	if !avEnabled {
		return a.ScanStatus != ScanBlocked
	}
	return a.ScanStatus == ScanClean
}
