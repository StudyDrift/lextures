package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
