// Package trustcenter persists trust-center data (sub-processor notification subscriptions).
package trustcenter

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Subscribe records an email address for sub-processor change notifications.
// Duplicate emails are silently ignored (idempotent).
func Subscribe(ctx context.Context, pool *pgxpool.Pool, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("trustcenter: invalid email address")
	}
	const q = `
INSERT INTO trust.sub_processor_subscriptions (email)
VALUES ($1)
ON CONFLICT (lower(email)) DO NOTHING`
	_, err := pool.Exec(ctx, q, email)
	return err
}
