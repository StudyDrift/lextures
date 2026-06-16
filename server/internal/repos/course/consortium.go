package course

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConsortiumSettings holds consortium-shareable flag for a course.
type ConsortiumSettings struct {
	ConsortiumShareable bool `json:"consortiumShareable"`
}

// GetConsortiumSettings returns consortium settings for a course, or nil if not found.
func GetConsortiumSettings(ctx context.Context, pool *pgxpool.Pool, courseCode string) (*ConsortiumSettings, error) {
	var shareable bool
	err := pool.QueryRow(ctx, `
SELECT consortium_shareable FROM course.courses WHERE course_code = $1
`, courseCode).Scan(&shareable)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &ConsortiumSettings{ConsortiumShareable: shareable}, nil
}

// SetConsortiumShareable updates the consortium-shareable flag.
func SetConsortiumShareable(ctx context.Context, pool *pgxpool.Pool, courseCode string, shareable bool) (*ConsortiumSettings, error) {
	tag, err := pool.Exec(ctx, `
UPDATE course.courses SET consortium_shareable = $2, updated_at = NOW()
WHERE course_code = $1
`, courseCode, shareable)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return &ConsortiumSettings{ConsortiumShareable: shareable}, nil
}
