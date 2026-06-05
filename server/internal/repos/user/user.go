// Package user is the Go port of server/src/repos/user.rs (subset for auth).
package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	AccountTypeStandard = "standard"
	AccountTypeParent   = "parent"
)

// userRowColumns is the canonical SELECT/RETURNING column list for user rows.
const userRowColumns = `id::text, email, password_hash, display_name, first_name, last_name, avatar_url, ui_theme, show_help_popover, locale, timezone, sid,
       login_blocked, deactivated_at, account_type`

// Row is a users table row for authentication and profile in API responses.
type Row struct {
	ID              string
	Email           string
	PasswordHash    string
	DisplayName     *string
	FirstName       *string
	LastName        *string
	AvatarURL       *string
	UITheme         string
	ShowHelpPopover bool
	Locale          string
	Timezone        *string
	Sid             *string
	LoginBlocked    bool
	DeactivatedAt   *time.Time
	// AccountType is "standard" (default) or "parent" (plan 5.10).
	AccountType string
}

func strPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func scanUserRow(ctx context.Context, pool *pgxpool.Pool, query string, arg any) (*Row, error) {
	var r Row
	var displayName, firstName, lastName, avatar, timezone, sid sql.NullString
	var deactivatedAt sql.NullTime
	err := pool.QueryRow(ctx, query, arg).Scan(
		&r.ID, &r.Email, &r.PasswordHash, &displayName, &firstName, &lastName, &avatar, &r.UITheme, &r.ShowHelpPopover, &r.Locale, &timezone, &sid,
		&r.LoginBlocked, &deactivatedAt, &r.AccountType,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	r.DisplayName = strPtr(displayName)
	r.FirstName = strPtr(firstName)
	r.LastName = strPtr(lastName)
	r.AvatarURL = strPtr(avatar)
	r.Timezone = strPtr(timezone)
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

func scanInsertedUserRow(row pgx.Row) (*Row, error) {
	var r Row
	var dn, fn, ln, av, timezone, sid sql.NullString
	var deactivatedAt sql.NullTime
	err := row.Scan(
		&r.ID, &r.Email, &r.PasswordHash, &dn, &fn, &ln, &av, &r.UITheme, &r.ShowHelpPopover, &r.Locale, &timezone, &sid,
		&r.LoginBlocked, &deactivatedAt, &r.AccountType,
	)
	if err != nil {
		return nil, err
	}
	r.DisplayName = strPtr(dn)
	r.FirstName = strPtr(fn)
	r.LastName = strPtr(ln)
	r.AvatarURL = strPtr(av)
	r.Timezone = strPtr(timezone)
	r.Sid = strPtr(sid)
	if r.AccountType == "" {
		r.AccountType = AccountTypeStandard
	}
	if r.Locale == "" {
		r.Locale = DefaultLocale
	}
	if deactivatedAt.Valid {
		t := deactivatedAt.Time
		r.DeactivatedAt = &t
	}
	return &r, nil
}

// GetGradeLevel returns the user's grade_level or nil if unset (plan 13.11).
func GetGradeLevel(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*string, error) {
	var gl sql.NullString
	err := pool.QueryRow(ctx, `SELECT grade_level FROM "user".users WHERE id = $1`, userID).Scan(&gl)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return strPtr(gl), nil
}

// FindByEmail returns a user by exact email (already normalized) or nil if missing.
func FindByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*Row, error) {
	const q = `SELECT ` + userRowColumns + `
FROM "user".users WHERE email = $1`
	return scanUserRow(ctx, pool, q, email)
}

// FindByID returns a user by primary key or nil.
func FindByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Row, error) {
	const q = `SELECT ` + userRowColumns + `
FROM "user".users WHERE id = $1`
	return scanUserRow(ctx, pool, q, id)
}

// InsertUser creates a new user; email must be normalized. Returns the full row.
func InsertUser(ctx context.Context, pool *pgxpool.Pool, email, passwordHash string, displayName *string) (*Row, error) {
	const q = `INSERT INTO "user".users (email, password_hash, display_name, org_id)
VALUES ($1, $2, $3, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1))
RETURNING ` + userRowColumns
	row := pool.QueryRow(ctx, q, email, passwordHash, displayName)
	return scanInsertedUserRow(row)
}

// InsertUserTx is InsertUser within an existing transaction.
func InsertUserTx(ctx context.Context, tx pgx.Tx, email, passwordHash string, displayName *string) (*Row, error) {
	const q = `INSERT INTO "user".users (email, password_hash, display_name, org_id)
VALUES ($1, $2, $3, (SELECT id FROM tenant.organizations WHERE slug = 'default' LIMIT 1))
RETURNING ` + userRowColumns
	row := tx.QueryRow(ctx, q, email, passwordHash, displayName)
	return scanInsertedUserRow(row)
}

// InsertUserInOrgTx creates a user in a specific organization (Canvas import provisioning).
func InsertUserInOrgTx(ctx context.Context, tx pgx.Tx, orgID uuid.UUID, email, passwordHash string, displayName *string) (*Row, error) {
	const q = `INSERT INTO "user".users (email, password_hash, display_name, org_id)
VALUES ($1, $2, $3, $4)
RETURNING ` + userRowColumns
	row := tx.QueryRow(ctx, q, email, passwordHash, displayName, orgID)
	return scanInsertedUserRow(row)
}

// SetPasswordHash updates the user's password hash (Argon2id PHC string).
func SetPasswordHash(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, passwordHash string) error {
	const q = `UPDATE "user".users SET password_hash = $2 WHERE id = $1::uuid`
	tag, err := pool.Exec(ctx, q, userID.String(), passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user: not found for password update")
	}
	return nil
}

// NormalizeEmail trims and lowercases an email (parity with services/auth/credentials.rs).
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// FindByEmailCI finds a user by case-insensitive email or nil.
func FindByEmailCI(ctx context.Context, pool *pgxpool.Pool, email string) (*Row, error) {
	em := NormalizeEmail(email)
	if em == "" {
		return nil, nil
	}
	const q = `SELECT ` + userRowColumns + `
FROM "user".users WHERE lower(email) = lower($1)`
	return scanUserRow(ctx, pool, q, em)
}
