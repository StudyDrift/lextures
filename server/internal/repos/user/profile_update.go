package user

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetAvatarURLIfEmptyTx stores an avatar URL when the user does not already have one.
func SetAvatarURLIfEmptyTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, avatarURL string) (bool, error) {
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		return false, nil
	}
	tag, err := tx.Exec(ctx, `
UPDATE "user".users
SET avatar_url = $2
WHERE id = $1
  AND (avatar_url IS NULL OR TRIM(avatar_url) = '')
`, userID, avatarURL)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// UpdateProfile patches account profile fields for one user.
func UpdateProfile(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	firstName, lastName, avatarURL, uiTheme *string,
	showHelpPopover *bool,
	timezone *string,
) (*Row, error) {
	const q = `UPDATE "user".users
SET
	first_name = $2,
	last_name = $3,
	avatar_url = $4,
	ui_theme = COALESCE($5, ui_theme),
	show_help_popover = COALESCE($6, show_help_popover),
	timezone = COALESCE($7, timezone)
WHERE id = $1
RETURNING ` + userRowColumns
	row := pool.QueryRow(ctx, q, userID, firstName, lastName, avatarURL, uiTheme, showHelpPopover, timezone)
	r, err := scanInsertedUserRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r, nil
}
