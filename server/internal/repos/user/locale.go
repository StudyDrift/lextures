package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultLocale is the fallback BCP 47 tag when none is stored.
const DefaultLocale = "en"

// UpdateLocale sets the user's UI locale (BCP 47 tag).
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
	if r.Locale == "" {
		r.Locale = DefaultLocale
	}
	if r.AccountType == "" {
		r.AccountType = AccountTypeStandard
	}
	if deactivatedAt.Valid {
		t := deactivatedAt.Time
		r.DeactivatedAt = &t
	}
	return &r, nil
}

// NormalizeLocalePrimary returns the primary BCP 47 subtag lowercased, or empty if invalid.
func NormalizeLocalePrimary(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, "-_"); i >= 0 {
		s = s[:i]
	}
	if len(s) < 2 || len(s) > 8 {
		return ""
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return s
}
