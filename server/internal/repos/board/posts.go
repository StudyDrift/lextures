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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxPostTitleLen = 500

// Post is one card on a collaboration board.
type Post struct {
	ID               string          `json:"id"`
	BoardID          string          `json:"boardId"`
	AuthorID         *string         `json:"authorId"`
	GuestDisplayName string          `json:"guestDisplayName,omitempty"`
	ContentType      string          `json:"contentType"`
	Title            string          `json:"title"`
	Body             json.RawMessage `json:"body,omitempty"`
	LinkURL          *string         `json:"linkUrl,omitempty"`
	LinkPreview      json.RawMessage `json:"linkPreview,omitempty"`
	DrawingData      json.RawMessage `json:"drawingData,omitempty"`
	AttachmentID     *string         `json:"attachmentId,omitempty"`
	Attachment       *Attachment     `json:"attachment,omitempty"`
	SectionID        *string         `json:"sectionId,omitempty"`
	SortIndex        float64         `json:"sortIndex"`
	Position         json.RawMessage `json:"position,omitempty"`
	EventDate        *time.Time      `json:"eventDate,omitempty"`
	Lat              *float64        `json:"lat,omitempty"`
	Lng              *float64        `json:"lng,omitempty"`
	Status           string          `json:"status"`
	Hidden           bool            `json:"hidden"`
	Removed          bool            `json:"removed"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

func scanPost(row pgx.Row) (Post, error) {
	var (
		id, boardID                       uuid.UUID
		authorID, attachmentID, sectionID uuid.NullUUID
		p                                 Post
		body, preview, drawing, position  []byte
		linkURL                           *string
		eventDate                         *time.Time
		lat, lng                          *float64
	)
	if err := row.Scan(
		&id, &boardID, &authorID, &p.GuestDisplayName, &p.ContentType, &p.Title,
		&body, &linkURL, &preview, &drawing, &attachmentID,
		&sectionID, &p.SortIndex, &position, &eventDate, &lat, &lng,
		&p.Status, &p.Hidden, &p.Removed,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return Post{}, err
	}
	p.ID = id.String()
	p.BoardID = boardID.String()
	if authorID.Valid {
		s := authorID.UUID.String()
		p.AuthorID = &s
	}
	if attachmentID.Valid {
		s := attachmentID.UUID.String()
		p.AttachmentID = &s
	}
	if sectionID.Valid {
		s := sectionID.UUID.String()
		p.SectionID = &s
	}
	p.LinkURL = linkURL
	p.EventDate = eventDate
	p.Lat = lat
	p.Lng = lng
	if p.Status == "" {
		p.Status = PostStatusApproved
	}
	if len(body) > 0 {
		p.Body = json.RawMessage(body)
	}
	if len(preview) > 0 {
		p.LinkPreview = json.RawMessage(preview)
	}
	if len(drawing) > 0 {
		p.DrawingData = json.RawMessage(drawing)
	}
	if len(position) > 0 {
		p.Position = json.RawMessage(position)
	}
	return p, nil
}

func selectPostCols() string {
	return `p.id, p.board_id, p.author_id, COALESCE(p.guest_display_name, ''), p.content_type, p.title, p.body, p.link_url, p.link_preview,
		p.drawing_data, p.attachment_id, p.section_id, p.sort_index, p.position,
		p.event_date, p.lat, p.lng, p.status, p.hidden, p.removed, p.created_at, p.updated_at`
}

// ListPosts returns posts for a board, newest first.
func ListPosts(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) ([]Post, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT `+selectPostCols()+`
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		ORDER BY p.sort_index ASC, p.created_at DESC
	`, courseCode, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Post, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := attachAttachments(ctx, pool, courseCode, boardID, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPost returns a single post scoped to course + board.
func GetPost(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, postID string) (*Post, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	pid, err := uuid.Parse(postID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT `+selectPostCols()+`
		FROM board.posts p
		INNER JOIN board.boards b ON b.id = p.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2 AND p.id = $3
	`, courseCode, bid, pid)
	p, err := scanPost(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	list := []Post{p}
	if err := attachAttachments(ctx, pool, courseCode, boardID, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

func attachAttachments(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string, posts []Post) error {
	for i := range posts {
		if posts[i].AttachmentID == nil {
			continue
		}
		a, err := GetAttachment(ctx, pool, courseCode, boardID, *posts[i].AttachmentID)
		if err != nil {
			return err
		}
		posts[i].Attachment = a
	}
	return nil
}

// CreatePost inserts a validated post.
func CreatePost(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	authorID uuid.UUID,
	in CreatePostInput,
	linkPreview *LinkPreview,
) (*Post, error) {
	in.ContentType = strings.TrimSpace(strings.ToLower(in.ContentType))
	if err := ValidateCreatePost(in); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if len(title) > maxPostTitleLen {
		return nil, fmt.Errorf("board: title must be at most %d characters", maxPostTitleLen)
	}
	body, err := NormalizeBody(in.Body)
	if err != nil {
		return nil, fmt.Errorf("board: invalid body: %w", err)
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}

	var attID *uuid.UUID
	if in.AttachmentID != nil && strings.TrimSpace(*in.AttachmentID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*in.AttachmentID))
		if err != nil {
			return nil, fmt.Errorf("board: invalid attachment_id")
		}
		att, err := GetAttachment(ctx, pool, courseCode, boardID, parsed.String())
		if err != nil {
			return nil, err
		}
		if att == nil {
			return nil, fmt.Errorf("board: attachment not found")
		}
		attID = &parsed
	}

	var linkURL *string
	if u := strings.TrimSpace(in.LinkURL); u != "" {
		linkURL = &u
	}
	var previewJSON []byte
	if linkPreview != nil {
		previewJSON, err = json.Marshal(linkPreview)
		if err != nil {
			return nil, err
		}
	}
	var drawing json.RawMessage
	if len(in.DrawingData) > 0 && string(in.DrawingData) != "null" {
		drawing = in.DrawingData
	}

	status := strings.TrimSpace(strings.ToLower(in.Status))
	if status == "" {
		status = PostStatusApproved
	}
	switch status {
	case PostStatusApproved, PostStatusPending, PostStatusRejected:
	default:
		return nil, fmt.Errorf("board: invalid post status")
	}

	var insertedID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO board.posts (
			board_id, author_id, content_type, title, body, link_url, link_preview, drawing_data, attachment_id, status
		)
		SELECT b.id, $3, $4, $5, $6, $7, $8, $9, $10, $11
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		RETURNING id
	`, courseCode, bid, authorID, in.ContentType, title, nullableJSON(body), linkURL, nullableJSON(previewJSON), nullableJSON(drawing), attID, status).Scan(&insertedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, fmt.Errorf("board: attachment not found")
		}
		return nil, err
	}
	return GetPost(ctx, pool, courseCode, boardID, insertedID.String())
}

// CreateGuestPost inserts a post with no author_id (contribute share links).
func CreateGuestPost(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, guestDisplayName string,
	in CreatePostInput,
	linkPreview *LinkPreview,
) (*Post, error) {
	in.ContentType = strings.TrimSpace(strings.ToLower(in.ContentType))
	if err := ValidateCreatePost(in); err != nil {
		return nil, err
	}
	guestDisplayName = strings.TrimSpace(guestDisplayName)
	if guestDisplayName == "" {
		return nil, fmt.Errorf("board: display name is required")
	}
	if len(guestDisplayName) > 80 {
		return nil, fmt.Errorf("board: display name must be at most 80 characters")
	}
	title := strings.TrimSpace(in.Title)
	if len(title) > maxPostTitleLen {
		return nil, fmt.Errorf("board: title must be at most %d characters", maxPostTitleLen)
	}
	body, err := NormalizeBody(in.Body)
	if err != nil {
		return nil, fmt.Errorf("board: invalid body: %w", err)
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	var linkURL *string
	if u := strings.TrimSpace(in.LinkURL); u != "" {
		linkURL = &u
	}
	var previewJSON []byte
	if linkPreview != nil {
		previewJSON, err = json.Marshal(linkPreview)
		if err != nil {
			return nil, err
		}
	}
	var drawing json.RawMessage
	if len(in.DrawingData) > 0 && string(in.DrawingData) != "null" {
		drawing = in.DrawingData
	}
	status := strings.TrimSpace(strings.ToLower(in.Status))
	if status == "" {
		status = PostStatusApproved
	}
	switch status {
	case PostStatusApproved, PostStatusPending, PostStatusRejected:
	default:
		return nil, fmt.Errorf("board: invalid post status")
	}

	var insertedID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO board.posts (
			board_id, author_id, guest_display_name, content_type, title, body, link_url, link_preview, drawing_data, status
		)
		SELECT b.id, NULL, $3, $4, $5, $6, $7, $8, $9, $10
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		RETURNING id
	`, courseCode, bid, guestDisplayName, in.ContentType, title, nullableJSON(body), linkURL, nullableJSON(previewJSON), nullableJSON(drawing), status).Scan(&insertedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return GetPost(ctx, pool, courseCode, boardID, insertedID.String())
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// PatchPostInput is a partial update.
type PatchPostInput struct {
	Title       *string
	Body        json.RawMessage
	LinkURL     *string
	DrawingData json.RawMessage
	LinkPreview *LinkPreview
}

// PatchPost updates mutable fields on a post.
func PatchPost(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID, postID string,
	in PatchPostInput,
) (*Post, error) {
	existing, err := GetPost(ctx, pool, courseCode, boardID, postID)
	if err != nil || existing == nil {
		return existing, err
	}
	title := existing.Title
	if in.Title != nil {
		title = strings.TrimSpace(*in.Title)
		if len(title) > maxPostTitleLen {
			return nil, fmt.Errorf("board: title must be at most %d characters", maxPostTitleLen)
		}
	}
	body := existing.Body
	if in.Body != nil {
		body, err = NormalizeBody(in.Body)
		if err != nil {
			return nil, fmt.Errorf("board: invalid body: %w", err)
		}
	}
	linkURL := existing.LinkURL
	if in.LinkURL != nil {
		u := strings.TrimSpace(*in.LinkURL)
		if u == "" {
			linkURL = nil
		} else {
			if err := validateHTTPURL(u); err != nil {
				return nil, err
			}
			linkURL = &u
		}
	}
	drawing := existing.DrawingData
	if in.DrawingData != nil {
		drawing = in.DrawingData
	}
	preview := existing.LinkPreview
	if in.LinkPreview != nil {
		preview, err = json.Marshal(in.LinkPreview)
		if err != nil {
			return nil, err
		}
	}

	pid, _ := uuid.Parse(postID)
	bid, _ := uuid.Parse(boardID)
	row := pool.QueryRow(ctx, `
		UPDATE board.posts p
		SET
			title = $4,
			body = $5,
			link_url = $6,
			link_preview = COALESCE($7, p.link_preview),
			drawing_data = COALESCE($8, p.drawing_data),
			updated_at = NOW()
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE p.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND p.id = $3
		RETURNING `+selectPostCols()+`
	`, courseCode, bid, pid, title, nullableJSON(body), linkURL, nullableJSON(preview), nullableJSON(drawing))
	p, err := scanPost(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	list := []Post{p}
	if err := attachAttachments(ctx, pool, courseCode, boardID, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// DeletePost permanently removes a post. Returns true if deleted.
func DeletePost(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, postID string) (bool, error) {
	pid, err := uuid.Parse(postID)
	if err != nil {
		return false, nil
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return false, nil
	}
	tag, err := pool.Exec(ctx, `
		DELETE FROM board.posts p
		USING board.boards b, course.courses c
		WHERE p.board_id = b.id AND c.id = b.course_id
		  AND c.course_code = $1 AND b.id = $2 AND p.id = $3
	`, courseCode, bid, pid)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
