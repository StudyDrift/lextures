package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpdateManagedLearnerProfile updates the few profile fields a managed learner may change
// themselves (#775): display name, avatar, and theme. Nil leaves a field unchanged;
// clearAvatar removes the avatar. Returns nil when the user is not a managed learner.
func UpdateManagedLearnerProfile(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	displayName, avatarURL, uiTheme *string,
	clearAvatar bool,
) (*Row, error) {
	const q = `UPDATE "user".users
SET
	display_name = COALESCE($2, display_name),
	avatar_url = CASE WHEN $5 THEN NULL ELSE COALESCE($3, avatar_url) END,
	ui_theme = COALESCE($4, ui_theme)
WHERE id = $1 AND account_type = 'managed'
RETURNING ` + userRowColumns
	row := pool.QueryRow(ctx, q, userID, displayName, avatarURL, uiTheme, clearAvatar)
	r, err := scanInsertedUserRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}
