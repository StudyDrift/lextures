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
)

// FullCopyAsyncThreshold: full copies with at least this many attachments return 202.
const FullCopyAsyncThreshold = 10

// BlobCopier copies object-store bytes so full copies do not share mutable objects (AC-3).
type BlobCopier interface {
	CopyBlob(ctx context.Context, srcKey, destKey string) error
}

// InstantiateOpts controls board creation from a definition.
type InstantiateOpts struct {
	Title       string
	Description string
	Locale      string
	// AuthorID is used for seed posts (FR-3: attributed to creating instructor).
	AuthorID uuid.UUID
	// BlobCopier is required when definition seed posts include attachments.
	BlobCopier BlobCopier
	// OnProgress reports 0–100 for long copies (optional).
	OnProgress func(pct int)
}

// InstantiateFromDefinition creates a board from a template definition (FR-3).
func InstantiateFromDefinition(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode string,
	createdBy uuid.UUID,
	def TemplateDefinition,
	opts InstantiateOpts,
) (*Board, error) {
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		return nil, fmt.Errorf("board: title is required")
	}
	created, err := Create(ctx, pool, courseCode, createdBy, title, opts.Description)
	if err != nil || created == nil {
		return created, err
	}
	report := opts.OnProgress
	if report == nil {
		report = func(int) {}
	}
	report(5)

	// Apply settings atomically; on failure hard-delete the shell board.
	cleanup := func() {
		_, _ = HardDelete(ctx, pool, courseCode, created.ID)
	}

	patch := PatchBoardInput{
		Layout: &def.Layout,
	}
	if len(def.Settings) > 0 {
		patch.Settings = def.Settings
	}
	if def.ReactionMode != "" {
		patch.ReactionMode = &def.ReactionMode
	}
	if def.Attribution != "" {
		patch.Attribution = &def.Attribution
	}
	if def.ModerationMode != "" {
		patch.ModerationMode = &def.ModerationMode
	}
	if def.FilterAction != "" {
		patch.FilterAction = &def.FilterAction
	}
	if def.CanPost != nil {
		patch.CanPost = def.CanPost
	}
	if def.CanInteract != nil {
		patch.CanInteract = def.CanInteract
	}
	if def.CanArrange != nil {
		patch.CanArrange = def.CanArrange
	}
	updated, err := Patch(ctx, pool, courseCode, created.ID, patch)
	if err != nil {
		cleanup()
		return nil, err
	}
	if updated != nil {
		created = updated
	}
	report(20)

	secIDByKey := map[string]string{}
	for i, sec := range def.Sections {
		idx := sec.SortIndex
		if idx == 0 && i > 0 {
			idx = float64(i)
		}
		s, err := CreateSection(ctx, pool, courseCode, created.ID, sec.Title, &idx)
		if err != nil {
			cleanup()
			return nil, err
		}
		if s == nil {
			cleanup()
			return nil, fmt.Errorf("board: could not create section %q", sec.Title)
		}
		secIDByKey[sec.Key] = s.ID
	}
	report(40)

	totalPosts := len(def.SeedPosts)
	for i, sp := range def.SeedPosts {
		if err := materializeSeedPost(ctx, pool, courseCode, created.ID, opts.AuthorID, sp, secIDByKey, opts.BlobCopier); err != nil {
			cleanup()
			return nil, err
		}
		if totalPosts > 0 {
			report(40 + (50 * (i + 1) / totalPosts))
		}
	}
	report(100)
	return Get(ctx, pool, courseCode, created.ID)
}

func materializeSeedPost(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseCode, boardID string,
	authorID uuid.UUID,
	sp DefinitionPost,
	secIDByKey map[string]string,
	copier BlobCopier,
) error {
	in := CreatePostInput{
		ContentType: sp.ContentType,
		Title:       sp.Title,
		Body:        sp.Body,
		LinkURL:     sp.LinkURL,
		DrawingData: sp.DrawingData,
		Status:      PostStatusApproved,
	}
	if sp.Attachment != nil {
		if FileBackedContentType(sp.ContentType) {
			if copier == nil {
				return fmt.Errorf("board: attachment copy requires storage")
			}
			destKey := fmt.Sprintf("boards/%s/attachments/%s/%s", courseCode, boardID, uuid.New().String())
			if err := copier.CopyBlob(ctx, sp.Attachment.StorageKey, destKey); err != nil {
				return fmt.Errorf("board: copy attachment: %w", err)
			}
			scan := sp.Attachment.ScanStatus
			if scan == "" {
				scan = ScanClean
			}
			att, err := CreateAttachment(
				ctx, pool, courseCode, boardID, authorID,
				destKey, sp.Attachment.FileName, sp.Attachment.MimeType, sp.Attachment.AltText, scan,
				sp.Attachment.SizeBytes,
			)
			if err != nil {
				return err
			}
			if att == nil {
				return fmt.Errorf("board: could not create attachment")
			}
			id := att.ID
			in.AttachmentID = &id
		}
	}

	post, err := CreatePost(ctx, pool, courseCode, boardID, authorID, in, nil)
	if err != nil {
		return err
	}
	if post == nil {
		return fmt.Errorf("board: could not create seed post")
	}

	arr := ArrangePostInput{}
	needArrange := false
	if sk := strings.TrimSpace(sp.SectionKey); sk != "" {
		if sid, ok := secIDByKey[sk]; ok {
			arr.SectionID = &sid
			needArrange = true
		}
	}
	if sp.SortIndex != 0 {
		idx := sp.SortIndex
		arr.SortIndex = &idx
		needArrange = true
	}
	if len(sp.Position) > 0 && string(sp.Position) != "null" {
		var pos PostPosition
		if err := json.Unmarshal(sp.Position, &pos); err == nil {
			arr.Position = &pos
			needArrange = true
		}
	}
	if sp.EventDate != nil {
		arr.EventDate = sp.EventDate
		needArrange = true
	}
	if sp.Lat != nil && sp.Lng != nil {
		arr.Lat = sp.Lat
		arr.Lng = sp.Lng
		needArrange = true
	}
	if needArrange {
		_, err = ArrangePost(ctx, pool, courseCode, boardID, post.ID, arr)
		if err != nil {
			return err
		}
	}
	return nil
}

// CopyBoardOpts controls duplication (FR-4 / FR-7 / FR-8).
type CopyBoardOpts struct {
	Mode        string // structure|full
	Title       string
	Description string
	AuthorID    uuid.UUID
	BlobCopier  BlobCopier
	OnProgress  func(pct int)
}

// CountBoardAttachments returns attachment count for async threshold decisions.
func CountBoardAttachments(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) (int, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return 0, nil
	}
	var n int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM board.post_attachments a
		INNER JOIN board.boards b ON b.id = a.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
	`, courseCode, bid).Scan(&n)
	return n, err
}

// CopyBoard duplicates a board into the target course (structure or full).
// Does not copy reactions, comments, reports, moderation log, or share links (FR-8 / AC-6).
func CopyBoard(
	ctx context.Context,
	pool *pgxpool.Pool,
	sourceCourseCode, sourceBoardID, targetCourseCode string,
	createdBy uuid.UUID,
	opts CopyBoardOpts,
) (*Board, error) {
	mode, err := NormalizeCopyMode(opts.Mode)
	if err != nil {
		return nil, err
	}
	src, err := Get(ctx, pool, sourceCourseCode, sourceBoardID)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, nil
	}
	sections, err := ListSections(ctx, pool, sourceCourseCode, sourceBoardID)
	if err != nil {
		return nil, err
	}
	includePosts := mode == CopyModeFull
	var posts []Post
	if includePosts {
		posts, err = ListPosts(ctx, pool, sourceCourseCode, sourceBoardID)
		if err != nil {
			return nil, err
		}
	}
	def := BoardToDefinition(*src, sections, posts, includePosts)
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = src.Title + " (copy)"
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen]
		}
	}
	desc := opts.Description
	if desc == "" {
		desc = src.Description
	}
	author := opts.AuthorID
	if author == uuid.Nil {
		author = createdBy
	}
	return InstantiateFromDefinition(ctx, pool, targetCourseCode, createdBy, def, InstantiateOpts{
		Title:       title,
		Description: desc,
		AuthorID:    author,
		BlobCopier:  opts.BlobCopier,
		OnProgress:  opts.OnProgress,
	})
}

// CopyJob tracks async full-copy progress.
type CopyJob struct {
	ID            string     `json:"id"`
	TargetCourseID string    `json:"targetCourseId"`
	SourceBoardID string     `json:"sourceBoardId"`
	CreatedBy     *string    `json:"createdBy,omitempty"`
	Mode          string     `json:"mode"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	Progress      int        `json:"progress"`
	ResultBoardID *string    `json:"resultBoardId,omitempty"`
	Error         string     `json:"error"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// CreateCopyJob inserts a pending copy job row.
func CreateCopyJob(
	ctx context.Context,
	pool *pgxpool.Pool,
	targetCourseID, sourceBoardID uuid.UUID,
	createdBy uuid.UUID,
	mode, title string,
) (*CopyJob, error) {
	mode, err := NormalizeCopyMode(mode)
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO board.board_copy_jobs (
			target_course_id, source_board_id, created_by, mode, title, status, progress
		) VALUES ($1, $2, $3, $4, $5, 'pending', 0)
		RETURNING id
	`, targetCourseID, sourceBoardID, createdBy, mode, strings.TrimSpace(title)).Scan(&id)
	if err != nil {
		return nil, err
	}
	return GetCopyJob(ctx, pool, id.String())
}

// GetCopyJob loads a copy job by id.
func GetCopyJob(ctx context.Context, pool *pgxpool.Pool, jobID string) (*CopyJob, error) {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return nil, nil
	}
	var j CopyJob
	var tid, sid uuid.UUID
	var createdBy, resultID uuid.NullUUID
	err = pool.QueryRow(ctx, `
		SELECT id, target_course_id, source_board_id, created_by, mode, title, status, progress,
			result_board_id, error, created_at, updated_at
		FROM board.board_copy_jobs
		WHERE id = $1
	`, id).Scan(
		&id, &tid, &sid, &createdBy, &j.Mode, &j.Title, &j.Status, &j.Progress,
		&resultID, &j.Error, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	j.ID = id.String()
	j.TargetCourseID = tid.String()
	j.SourceBoardID = sid.String()
	if createdBy.Valid {
		s := createdBy.UUID.String()
		j.CreatedBy = &s
	}
	if resultID.Valid {
		s := resultID.UUID.String()
		j.ResultBoardID = &s
	}
	return &j, nil
}

// UpdateCopyJobProgress updates progress/status for a copy job.
func UpdateCopyJobProgress(ctx context.Context, pool *pgxpool.Pool, jobID string, status string, progress int, resultBoardID *uuid.UUID, errMsg string) error {
	id, err := uuid.Parse(jobID)
	if err != nil {
		return fmt.Errorf("board: invalid job id")
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	_, err = pool.Exec(ctx, `
		UPDATE board.board_copy_jobs
		SET status = $2, progress = $3, result_board_id = COALESCE($4, result_board_id),
			error = $5, updated_at = NOW()
		WHERE id = $1
	`, id, status, progress, resultBoardID, errMsg)
	return err
}

// GetCopyJobForCourse ensures the job belongs to the given course code.
func GetCopyJobForCourse(ctx context.Context, pool *pgxpool.Pool, courseCode, jobID string) (*CopyJob, error) {
	j, err := GetCopyJob(ctx, pool, jobID)
	if err != nil || j == nil {
		return j, err
	}
	var code string
	err = pool.QueryRow(ctx, `
		SELECT course_code FROM course.courses WHERE id = $1::uuid
	`, j.TargetCourseID).Scan(&code)
	if err != nil || code != courseCode {
		return nil, nil
	}
	return j, nil
}
