package readinglevel

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	readingsvc "github.com/lextures/lextures/server/internal/service/readinglevel"
)

// ScoreAndPersist analyzes markdown and stores FKGL/FRE on the module row.
func ScoreAndPersist(ctx context.Context, pool *pgxpool.Pool, itemID uuid.UUID, itemType ItemType, markdown string) error {
	if pool == nil {
		return nil
	}
	plain := readingsvc.PlainTextFromMarkdown(markdown)
	sc := readingsvc.Analyze(plain)
	return UpdateScoreForItem(ctx, pool, itemID, itemType, sc)
}
