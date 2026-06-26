package lrsconfig

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
)

// Endpoint is an external LRS target (credentials omitted in list responses).
type Endpoint struct {
	ID             uuid.UUID
	OrgID          uuid.UUID
	Label          string
	EndpointURL    string
	AuthType       string
	Username       *string
	Enabled        bool
	HasPassword    bool
	HasOAuthSecret bool
	OAuthClientID  *string
	OAuthTokenURL  *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type endpointSecrets struct {
	PasswordCiphertext      []byte
	OAuthClientSecretCipher []byte
}

// ListByOrg returns endpoints for an organization (no secrets).
func ListByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Endpoint, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, label, endpoint_url, auth_type, username, enabled,
       (password_ciphertext IS NOT NULL), (oauth_client_secret_ciphertext IS NOT NULL),
       oauth_client_id, oauth_token_url, created_at, updated_at
FROM analytics.lrs_endpoints
WHERE org_id = $1
ORDER BY created_at
`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Endpoint
	for rows.Next() {
		var e Endpoint
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.Label, &e.EndpointURL, &e.AuthType, &e.Username, &e.Enabled,
			&e.HasPassword, &e.HasOAuthSecret, &e.OAuthClientID, &e.OAuthTokenURL, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListEnabled returns enabled endpoints for forwarding (includes secrets).
func ListEnabled(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Endpoint, []endpointSecrets, error) {
	rows, err := pool.Query(ctx, `
SELECT id, org_id, label, endpoint_url, auth_type, username, enabled,
       password_ciphertext, oauth_client_secret_ciphertext,
       oauth_client_id, oauth_token_url, created_at, updated_at
FROM analytics.lrs_endpoints
WHERE org_id = $1 AND enabled = true
ORDER BY created_at
`, orgID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var eps []Endpoint
	var secs []endpointSecrets
	for rows.Next() {
		var e Endpoint
		var sec endpointSecrets
		if err := rows.Scan(
			&e.ID, &e.OrgID, &e.Label, &e.EndpointURL, &e.AuthType, &e.Username, &e.Enabled,
			&sec.PasswordCiphertext, &sec.OAuthClientSecretCipher,
			&e.OAuthClientID, &e.OAuthTokenURL, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, nil, err
		}
		e.HasPassword = len(sec.PasswordCiphertext) > 0
		e.HasOAuthSecret = len(sec.OAuthClientSecretCipher) > 0
		eps = append(eps, e)
		secs = append(secs, sec)
	}
	return eps, secs, rows.Err()
}

type CreateInput struct {
	OrgID             uuid.UUID
	Label             string
	EndpointURL       string
	AuthType          string
	Username          string
	Password          []byte // plaintext; encrypted before store
	OAuthClientID     string
	OAuthClientSecret []byte
	OAuthTokenURL     string
	Enabled           bool
}

// Create inserts a new LRS endpoint.
func Create(ctx context.Context, pool *pgxpool.Pool, key []byte, in CreateInput) (uuid.UUID, error) {
	var pwCipher, oauthCipher []byte
	var err error
	if len(in.Password) > 0 && len(key) == 32 {
		pwCipher, err = appsecrets.Encrypt(in.Password, key)
		if err != nil {
			return uuid.Nil, err
		}
	}
	if len(in.OAuthClientSecret) > 0 && len(key) == 32 {
		oauthCipher, err = appsecrets.Encrypt(in.OAuthClientSecret, key)
		if err != nil {
			return uuid.Nil, err
		}
	}
	var id uuid.UUID
	err = pool.QueryRow(ctx, `
INSERT INTO analytics.lrs_endpoints (
  org_id, label, endpoint_url, auth_type, username, password_ciphertext,
  oauth_client_id, oauth_client_secret_ciphertext, oauth_token_url, enabled
) VALUES ($1, $2, $3, $4, NULLIF($5,''), $6, NULLIF($7,''), $8, NULLIF($9,''), $10)
RETURNING id
`, in.OrgID, in.Label, in.EndpointURL, in.AuthType, in.Username, pwCipher,
		in.OAuthClientID, oauthCipher, in.OAuthTokenURL, in.Enabled).Scan(&id)
	return id, err
}

type UpdateInput struct {
	Label             *string
	EndpointURL       *string
	AuthType          *string
	Username          *string
	Password          []byte // nil = unchanged; empty = clear
	OAuthClientID     *string
	OAuthClientSecret []byte
	OAuthTokenURL     *string
	Enabled           *bool
}

// Update patches an endpoint; returns not found when missing.
func Update(ctx context.Context, pool *pgxpool.Pool, key []byte, id uuid.UUID, in UpdateInput) (bool, error) {
	cur, sec, err := getSecrets(ctx, pool, id)
	if err != nil {
		return false, err
	}
	if cur == nil {
		return false, nil
	}
	pwCipher := sec.PasswordCiphertext
	if in.Password != nil {
		if len(in.Password) == 0 {
			pwCipher = nil
		} else if len(key) == 32 {
			pwCipher, err = appsecrets.Encrypt(in.Password, key)
			if err != nil {
				return false, err
			}
		}
	}
	oauthCipher := sec.OAuthClientSecretCipher
	if in.OAuthClientSecret != nil {
		if len(in.OAuthClientSecret) == 0 {
			oauthCipher = nil
		} else if len(key) == 32 {
			oauthCipher, err = appsecrets.Encrypt(in.OAuthClientSecret, key)
			if err != nil {
				return false, err
			}
		}
	}
	tag, err := pool.Exec(ctx, `
UPDATE analytics.lrs_endpoints SET
  label = COALESCE($2, label),
  endpoint_url = COALESCE($3, endpoint_url),
  auth_type = COALESCE($4, auth_type),
  username = COALESCE($5, username),
  password_ciphertext = $6,
  oauth_client_id = COALESCE($7, oauth_client_id),
  oauth_client_secret_ciphertext = $8,
  oauth_token_url = COALESCE($9, oauth_token_url),
  enabled = COALESCE($10, enabled),
  updated_at = now()
WHERE id = $1
`, id, in.Label, in.EndpointURL, in.AuthType, in.Username, pwCipher,
		in.OAuthClientID, oauthCipher, in.OAuthTokenURL, in.Enabled)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func getSecrets(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Endpoint, *endpointSecrets, error) {
	var e Endpoint
	var sec endpointSecrets
	err := pool.QueryRow(ctx, `
SELECT id, org_id, label, endpoint_url, auth_type, username, enabled,
       password_ciphertext, oauth_client_secret_ciphertext,
       oauth_client_id, oauth_token_url, created_at, updated_at
FROM analytics.lrs_endpoints WHERE id = $1
`, id).Scan(
		&e.ID, &e.OrgID, &e.Label, &e.EndpointURL, &e.AuthType, &e.Username, &e.Enabled,
		&sec.PasswordCiphertext, &sec.OAuthClientSecretCipher,
		&e.OAuthClientID, &e.OAuthTokenURL, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return &e, &sec, nil
}

// GetForForward loads one enabled endpoint with secrets.
func GetForForward(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Endpoint, *endpointSecrets, error) {
	return getSecrets(ctx, pool, id)
}
