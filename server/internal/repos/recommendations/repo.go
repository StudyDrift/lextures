package recommendations

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecommendationOverrideRow struct {
	ID              uuid.UUID
	CourseID        uuid.UUID
	StructureItemID uuid.UUID
	OverrideType    string
	Surface         *string
	CreatedBy       uuid.UUID
	CreatedAt       time.Time
}

type CachedRecommendations struct {
	Recommendations []json.RawMessage `json:"recommendations"`
	Degraded        bool              `json:"degraded"`
}

func GetCache(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, surface string) (*CachedRecommendations, bool, error) {
	var payload json.RawMessage
	var expiresAt time.Time
	err := pool.QueryRow(ctx, `
SELECT recommendations, expires_at
FROM course.recommendation_cache
WHERE user_id = $1 AND course_id = $2 AND surface = $3
`, userID, courseID, surface).Scan(&payload, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var parsed CachedRecommendations
	if err := json.Unmarshal(payload, &parsed); err != nil {
		parsed = CachedRecommendations{}
	}
	expired := !expiresAt.After(time.Now().UTC())
	return &parsed, expired, nil
}

func InsertEvent(ctx context.Context, pool *pgxpool.Pool, userID, courseID uuid.UUID, itemID *uuid.UUID, surface, eventType string, rank *int16) error {
	_, err := pool.Exec(ctx, `
INSERT INTO course.recommendation_events (user_id, course_id, item_id, surface, event_type, rank)
VALUES ($1, $2, $3, $4, $5, $6)
`, userID, courseID, itemID, surface, eventType, rank)
	return err
}

type ConceptQuizItemRow struct {
	ConceptID       uuid.UUID
	StructureItemID uuid.UUID
	Title           string
}
