package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UpdateLocale sets the user locale preference.
func UpdateLocale(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, locale string) (*Row, error) {
	const q = `UPDATE "user".users
SET locale = $2
WHERE id = $1
RETURNING id::text, email, password_hash, display_name, first_name, last_name, avatar_url, ui_theme, show_help_popover, locale, sid,
  login_blocked, deactivated_at, account_type`
	var r Row
	var dn, fn, ln, av, sid sql.NullString
	var deactivatedAt sql.NullTime
	err := pool.QueryRow(ctx, q, userID, locale).Scan(
		&r.ID, &r.Email, &r.PasswordHash, &dn, &fn, &ln, &av, &r.UITheme, &r.ShowHelpPopover, &r.Locale, &sid,
		&r.LoginBlocked, &deactivatedAt, &r.AccountType,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	r.DisplayName = strPtr(dn)
	r.FirstName = strPtr(fn)
	r.LastName = strPtr(ln)
	r.AvatarURL = strPtr(av)
	r.Sid = strPtr(sid)
	if r.AccountType == "" {
		r.AccountType = AccountTypeStandard
	}
	if deactivatedAt.Valid {
		t := deactivatedAt.Time
		r.DeactivatedAt = &t
	}
	return &r, nil
}
