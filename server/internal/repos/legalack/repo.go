// Package legalack persists user acknowledgements of privacy policy and terms updates.
package legalack

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DocumentPrivacyPolicy  = "privacy_policy"
	DocumentTermsOfService = "terms_of_service"
)

// PendingDocument is a legal document the user has not yet acknowledged at the current version.
type PendingDocument struct {
	Document      string
	Version       string
	EffectiveDate string
}

// RecordAck inserts an acknowledgement for the given user, document, and version.
func RecordAck(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, document, version string) error {
	document = strings.TrimSpace(document)
	version = strings.TrimSpace(version)
	if document == "" || version == "" {
		return errors.New("legalack: document and version required")
	}
	const q = `
INSERT INTO settings.user_legal_acknowledgements (user_id, document, version)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, document, version) DO NOTHING`
	_, err := pool.Exec(ctx, q, userID, document, version)
	return err
}

// HasAck returns whether the user has acknowledged the given document at the given version.
func HasAck(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, document, version string) (bool, error) {
	const q = `
SELECT 1 FROM settings.user_legal_acknowledgements
WHERE user_id = $1 AND document = $2 AND version = $3
LIMIT 1`
	var one int
	err := pool.QueryRow(ctx, q, userID, document, version).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Pending returns documents from currentVersions that the user has not acknowledged.
func Pending(
	ctx context.Context,
	pool *pgxpool.Pool,
	userID uuid.UUID,
	currentVersions map[string]struct {
		Version       string
		EffectiveDate string
	},
) ([]PendingDocument, error) {
	var out []PendingDocument
	for doc, cur := range currentVersions {
		ok, err := HasAck(ctx, pool, userID, doc, cur.Version)
		if err != nil {
			return nil, err
		}
		if !ok {
			out = append(out, PendingDocument{
				Document:      doc,
				Version:       cur.Version,
				EffectiveDate: cur.EffectiveDate,
			})
		}
	}
	return out, nil
}
