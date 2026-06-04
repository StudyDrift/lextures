package course

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WhiteboardRow struct {
	ID         string          `json:"id"`
	CourseID   string          `json:"courseId"`
	Title      string          `json:"title"`
	CanvasData json.RawMessage `json:"canvasData"`
	CreatedBy  *string         `json:"createdBy,omitempty"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

func scanWhiteboardRow(row pgx.Row) (WhiteboardRow, error) {
	var wb WhiteboardRow
	var id, courseID uuid.UUID
	var createdBy uuid.NullUUID
	var canvasData []byte
	if err := row.Scan(&id, &courseID, &wb.Title, &canvasData, &createdBy, &wb.CreatedAt, &wb.UpdatedAt); err != nil {
		return WhiteboardRow{}, err
	}
	wb.ID = id.String()
	wb.CourseID = courseID.String()
	if len(canvasData) > 0 {
		wb.CanvasData = json.RawMessage(canvasData)
	} else {
		wb.CanvasData = json.RawMessage("[]")
	}
	if createdBy.Valid {
		s := createdBy.UUID.String()
		wb.CreatedBy = &s
	}
	return wb, nil
}

func ListWhiteboards(ctx context.Context, pool *pgxpool.Pool, courseCode string) ([]WhiteboardRow, error) {
	rows, err := pool.Query(ctx, `
		SELECT w.id, w.course_id, w.title, w.canvas_data, w.created_by, w.created_at, w.updated_at
		FROM course.whiteboards w
		INNER JOIN course.courses c ON c.id = w.course_id
		WHERE c.course_code = $1
		ORDER BY w.updated_at DESC
	`, courseCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WhiteboardRow
	for rows.Next() {
		wb, err := scanWhiteboardRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wb)
	}
	return out, rows.Err()
}

func CreateWhiteboard(ctx context.Context, pool *pgxpool.Pool, courseCode string, createdBy uuid.UUID, title string, canvasData json.RawMessage) (*WhiteboardRow, error) {
	if len(canvasData) == 0 {
		canvasData = json.RawMessage("[]")
	}
	row := pool.QueryRow(ctx, `
		INSERT INTO course.whiteboards (course_id, title, canvas_data, created_by)
		SELECT c.id, $2, $3, $4
		FROM course.courses c
		WHERE c.course_code = $1
		RETURNING id, course_id, title, canvas_data, created_by, created_at, updated_at
	`, courseCode, title, []byte(canvasData), createdBy)
	wb, err := scanWhiteboardRow(row)
	if err != nil {
		return nil, err
	}
	return &wb, nil
}

func GetWhiteboard(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) (*WhiteboardRow, error) {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	row := pool.QueryRow(ctx, `
		SELECT w.id, w.course_id, w.title, w.canvas_data, w.created_by, w.created_at, w.updated_at
		FROM course.whiteboards w
		INNER JOIN course.courses c ON c.id = w.course_id
		WHERE c.course_code = $1 AND w.id = $2
	`, courseCode, id)
	wb, err := scanWhiteboardRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &wb, nil
}

func UpdateWhiteboard(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID, title string, canvasData json.RawMessage) (*WhiteboardRow, error) {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return nil, nil
	}
	if len(canvasData) == 0 {
		canvasData = json.RawMessage("[]")
	}
	row := pool.QueryRow(ctx, `
		UPDATE course.whiteboards w
		SET title = $3, canvas_data = $4, updated_at = NOW()
		FROM course.courses c
		WHERE c.id = w.course_id AND c.course_code = $1 AND w.id = $2
		RETURNING w.id, w.course_id, w.title, w.canvas_data, w.created_by, w.created_at, w.updated_at
	`, courseCode, id, title, []byte(canvasData))
	wb, err := scanWhiteboardRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &wb, nil
}

func DeleteWhiteboard(ctx context.Context, pool *pgxpool.Pool, courseCode, boardID string) error {
	id, err := uuid.Parse(boardID)
	if err != nil {
		return nil
	}
	_, err = pool.Exec(ctx, `
		DELETE FROM course.whiteboards w
		USING course.courses c
		WHERE c.id = w.course_id AND c.course_code = $1 AND w.id = $2
	`, courseCode, id)
	return err
}
