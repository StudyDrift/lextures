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

const maxSectionTitleLen = 200

// Section is a named column/shelf on a board.
type Section struct {
	ID        string    `json:"id"`
	BoardID   string    `json:"boardId"`
	Title     string    `json:"title"`
	SortIndex float64   `json:"sortIndex"`
	CreatedAt time.Time `json:"createdAt"`
}

func scanSection(row pgx.Row) (Section, error) {
	var s Section
	var id, boardID uuid.UUID
	if err := row.Scan(&id, &boardID, &s.Title, &s.SortIndex, &s.CreatedAt); err != nil {
		return Section{}, err
	}
	s.ID = id.String()
	s.BoardID = boardID.String()
	return s, nil
}

func selectSectionCols() string {
	return `s.id, s.board_id, s.title, s.sort_index, s.created_at`
}

// ListSections returns sections for a board ordered by sort_index.
func ListSections(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) ([]Section, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		SELECT `+selectSectionCols()+`
		FROM board.sections s
		INNER JOIN board.boards b ON b.id = s.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		ORDER BY s.sort_index ASC, s.created_at ASC
	`, courseCode, bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Section, 0)
	for rows.Next() {
		s, err := scanSection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSection returns one section scoped to course + board.
func GetSection(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, sectionID string) (*Section, error) {
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	sid, err := uuid.Parse(sectionID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT `+selectSectionCols()+`
		FROM board.sections s
		INNER JOIN board.boards b ON b.id = s.board_id
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2 AND s.id = $3
	`, courseCode, bid, sid)
	s, err := scanSection(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// CreateSection inserts a named section; sort_index defaults to append.
func CreateSection(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, title string, sortIndex *float64) (*Section, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("board: section title is required")
	}
	if len(title) > maxSectionTitleLen {
		return nil, fmt.Errorf("board: section title must be at most %d characters", maxSectionTitleLen)
	}
	bid, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}

	idx := 0.0
	if sortIndex != nil {
		idx = *sortIndex
	} else {
		var max *float64
		_ = pool.QueryRow(ctx, `
			SELECT MAX(s.sort_index)
			FROM board.sections s
			INNER JOIN board.boards b ON b.id = s.board_id
			INNER JOIN course.courses c ON c.id = b.course_id
			WHERE c.course_code = $1 AND b.id = $2
		`, courseCode, bid).Scan(&max)
		idx = AppendSortIndex(max)
	}

	var insertedID uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO board.sections (board_id, title, sort_index)
		SELECT b.id, $3, $4
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE c.course_code = $1 AND b.id = $2
		RETURNING id
	`, courseCode, bid, title, idx).Scan(&insertedID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return GetSection(ctx, pool, courseCode, boardID, insertedID.String())
}

// PatchSectionInput is a partial section update.
type PatchSectionInput struct {
	Title     *string
	SortIndex *float64
}

// PatchSection renames and/or reorders a section.
func PatchSection(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, sectionID string, in PatchSectionInput) (*Section, error) {
	existing, err := GetSection(ctx, pool, courseCode, boardID, sectionID)
	if err != nil || existing == nil {
		return existing, err
	}
	title := existing.Title
	if in.Title != nil {
		title = strings.TrimSpace(*in.Title)
		if title == "" {
			return nil, fmt.Errorf("board: section title is required")
		}
		if len(title) > maxSectionTitleLen {
			return nil, fmt.Errorf("board: section title must be at most %d characters", maxSectionTitleLen)
		}
	}
	idx := existing.SortIndex
	if in.SortIndex != nil {
		idx = *in.SortIndex
	}
	sid, _ := uuid.Parse(sectionID)
	bid, _ := uuid.Parse(boardID)
	row := pool.QueryRow(ctx, `
		UPDATE board.sections s
		SET title = $4, sort_index = $5
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE s.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND s.id = $3
		RETURNING s.id, s.board_id, s.title, s.sort_index, s.created_at
	`, courseCode, bid, sid, title, idx)
	s, err := scanSection(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// DeleteSection removes a section and moves its cards to Unsorted (FR-3).
func DeleteSection(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, sectionID string) (bool, error) {
	existing, err := GetSection(ctx, pool, courseCode, boardID, sectionID)
	if err != nil || existing == nil {
		return false, err
	}

	unsorted, err := EnsureUnsortedSection(ctx, pool, courseCode, boardID)
	if err != nil {
		return false, err
	}
	if unsorted == nil {
		return false, fmt.Errorf("board: could not ensure Unsorted section")
	}
	// Do not delete Unsorted itself when it is the only/home section.
	if existing.ID == unsorted.ID {
		return false, fmt.Errorf("board: cannot delete the Unsorted section")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sid, _ := uuid.Parse(sectionID)
	uid, _ := uuid.Parse(unsorted.ID)
	bid, _ := uuid.Parse(boardID)

	if _, err := tx.Exec(ctx, `
		UPDATE board.posts p
		SET section_id = $3, updated_at = NOW()
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE p.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND p.section_id = $4
	`, courseCode, bid, uid, sid); err != nil {
		return false, err
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM board.sections s
		USING board.boards b, course.courses c
		WHERE s.board_id = b.id AND c.id = b.course_id
		  AND c.course_code = $1 AND b.id = $2 AND s.id = $3
	`, courseCode, bid, sid)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// EnsureUnsortedSection returns the Unsorted section, creating it at sort_index 0 if missing.
func EnsureUnsortedSection(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) (*Section, error) {
	sections, err := ListSections(ctx, pool, courseCode, boardID)
	if err != nil {
		return nil, err
	}
	for i := range sections {
		if sections[i].Title == UnsortedSectionTitle {
			return &sections[i], nil
		}
	}
	zero := 0.0
	return CreateSection(ctx, pool, courseCode, boardID, UnsortedSectionTitle, &zero)
}

// AssignUnsectionedToUnsorted moves posts with NULL section_id into Unsorted.
func AssignUnsectionedToUnsorted(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) error {
	unsorted, err := EnsureUnsortedSection(ctx, pool, courseCode, boardID)
	if err != nil || unsorted == nil {
		return err
	}
	uid, _ := uuid.Parse(unsorted.ID)
	bid, _ := uuid.Parse(boardID)
	_, err = pool.Exec(ctx, `
		UPDATE board.posts p
		SET section_id = $3, updated_at = NOW()
		FROM board.boards b
		INNER JOIN course.courses c ON c.id = b.course_id
		WHERE p.board_id = b.id AND c.course_code = $1 AND b.id = $2 AND p.section_id IS NULL
	`, courseCode, bid, uid)
	return err
}
