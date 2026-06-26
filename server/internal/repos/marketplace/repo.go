// Package marketplace stores marketplace apps and per-org installations (plan 16.9).
package marketplace

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	appSecretPrefix    = "mcs_"
	tokenPrefixLabel   = "mkt_"
	refreshPrefixLabel = "mkr_"
)

// ErrNotFound is returned when a queried record does not exist.
var ErrNotFound = errors.New("marketplace: not found")

// ErrDuplicateSlug is returned when an app slug is already taken.
var ErrDuplicateSlug = errors.New("marketplace: slug already taken")

// App is a registered marketplace application.
type App struct {
	ID                 uuid.UUID
	DeveloperUserID    *uuid.UUID
	Name               string
	Slug               string
	Description        string
	LogoURL            *string
	RedirectURIs       []string
	RequestedScopes    []string
	ClientID           string
	ClientSecretPrefix string
	Published          bool
	CreatedAt          time.Time
}

// Installation is a per-org app installation.
type Installation struct {
	ID                 uuid.UUID
	AppID              uuid.UUID
	OrgID              uuid.UUID
	AppName            string
	AppSlug            string
	AppLogoURL         *string
	AccessTokenPrefix  string
	GrantedScopes      []string
	InstalledBy        *uuid.UUID
	InstalledAt        time.Time
	RevokedAt          *time.Time
	LastUsedAt         *time.Time
}

// GenerateClientSecret returns a new mcs_ client secret, its SHA-256 hex hash and 8-char prefix.
func GenerateClientSecret() (secret, hashHex, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}
	secret = appSecretPrefix + base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(secret))
	hashHex = hex.EncodeToString(sum[:])
	prefix = secret[:8]
	return secret, hashHex, prefix, nil
}

// GenerateAccessToken returns a new mkt_ bearer token, its hash and prefix.
func GenerateAccessToken() (token, hashHex, prefix string, err error) {
	return generateToken(tokenPrefixLabel)
}

// GenerateRefreshToken returns a new mkr_ refresh token, its hash and prefix.
func GenerateRefreshToken() (token, hashHex, prefix string, err error) {
	return generateToken(refreshPrefixLabel)
}

func generateToken(pfx string) (token, hashHex, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}
	token = pfx + base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	hashHex = hex.EncodeToString(sum[:])
	prefix = token[:8]
	return token, hashHex, prefix, nil
}

// HashToken returns the SHA-256 hex hash of a token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateAppParams is the input for CreateApp.
type CreateAppParams struct {
	DeveloperUserID uuid.UUID
	Name            string
	Slug            string
	Description     string
	LogoURL         *string
	RedirectURIs    []string
	RequestedScopes []string
	ClientSecretHash   string
	ClientSecretPrefix string
}

// CreateApp inserts a new marketplace app and returns it.
func CreateApp(ctx context.Context, pool *pgxpool.Pool, p CreateAppParams) (App, error) {
	var a App
	err := pool.QueryRow(ctx, `
		INSERT INTO marketplace.apps
		    (developer_user_id, name, slug, description, logo_url, redirect_uris,
		     requested_scopes, client_secret_hash, client_secret_prefix)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, developer_user_id, name, slug, description, logo_url,
		          redirect_uris, requested_scopes, client_id, client_secret_prefix,
		          published, created_at
	`, p.DeveloperUserID, p.Name, p.Slug, p.Description, p.LogoURL,
		p.RedirectURIs, p.RequestedScopes, p.ClientSecretHash, p.ClientSecretPrefix,
	).Scan(
		&a.ID, &a.DeveloperUserID, &a.Name, &a.Slug, &a.Description, &a.LogoURL,
		&a.RedirectURIs, &a.RequestedScopes, &a.ClientID, &a.ClientSecretPrefix,
		&a.Published, &a.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return App{}, ErrDuplicateSlug
		}
		return App{}, err
	}
	return a, nil
}

// ListAppsByDeveloper returns all apps registered by a developer.
func ListAppsByDeveloper(ctx context.Context, pool *pgxpool.Pool, developerUserID uuid.UUID) ([]App, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, developer_user_id, name, slug, description, logo_url,
		       redirect_uris, requested_scopes, client_id, client_secret_prefix,
		       published, created_at
		FROM marketplace.apps
		WHERE developer_user_id = $1
		ORDER BY created_at DESC
	`, developerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApps(rows)
}

// ListPublishedApps returns all published marketplace apps ordered by name.
func ListPublishedApps(ctx context.Context, pool *pgxpool.Pool) ([]App, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, developer_user_id, name, slug, description, logo_url,
		       redirect_uris, requested_scopes, client_id, client_secret_prefix,
		       published, created_at
		FROM marketplace.apps
		WHERE published = true
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApps(rows)
}

// GetAppBySlug returns a published app by its slug.
func GetAppBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*App, error) {
	a, err := getApp(ctx, pool, `slug = $1 AND published = true`, slug)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return a, err
}

// GetAppByClientID returns any app by OAuth client_id (used during consent/token exchange).
func GetAppByClientID(ctx context.Context, pool *pgxpool.Pool, clientID string) (*App, error) {
	a, err := getApp(ctx, pool, `client_id = $1`, clientID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return a, err
}

// GetAppByID returns an app by primary key.
func GetAppByID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*App, error) {
	a, err := getApp(ctx, pool, `id = $1`, id)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return a, err
}

func getApp(ctx context.Context, pool *pgxpool.Pool, where string, args ...any) (*App, error) {
	var a App
	err := pool.QueryRow(ctx,
		`SELECT id, developer_user_id, name, slug, description, logo_url,
		        redirect_uris, requested_scopes, client_id, client_secret_prefix,
		        published, created_at
		 FROM marketplace.apps
		 WHERE `+where,
		args...,
	).Scan(
		&a.ID, &a.DeveloperUserID, &a.Name, &a.Slug, &a.Description, &a.LogoURL,
		&a.RedirectURIs, &a.RequestedScopes, &a.ClientID, &a.ClientSecretPrefix,
		&a.Published, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &a, err
}

func scanApps(rows pgx.Rows) ([]App, error) {
	var out []App
	for rows.Next() {
		var a App
		if err := rows.Scan(
			&a.ID, &a.DeveloperUserID, &a.Name, &a.Slug, &a.Description, &a.LogoURL,
			&a.RedirectURIs, &a.RequestedScopes, &a.ClientID, &a.ClientSecretPrefix,
			&a.Published, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ValidateClientSecret verifies a raw client secret against the stored hash.
func ValidateClientSecret(ctx context.Context, pool *pgxpool.Pool, clientID, rawSecret string) (*App, bool, error) {
	var a App
	var storedHash string
	err := pool.QueryRow(ctx, `
		SELECT id, developer_user_id, name, slug, description, logo_url,
		       redirect_uris, requested_scopes, client_id, client_secret_prefix,
		       published, created_at, client_secret_hash
		FROM marketplace.apps
		WHERE client_id = $1
	`, clientID).Scan(
		&a.ID, &a.DeveloperUserID, &a.Name, &a.Slug, &a.Description, &a.LogoURL,
		&a.RedirectURIs, &a.RequestedScopes, &a.ClientID, &a.ClientSecretPrefix,
		&a.Published, &a.CreatedAt, &storedHash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &a, HashToken(rawSecret) == storedHash, nil
}

// CreateInstallationParams is the input for CreateInstallation.
type CreateInstallationParams struct {
	AppID              uuid.UUID
	OrgID              uuid.UUID
	AccessTokenHash    string
	AccessTokenPrefix  string
	RefreshTokenHash   string
	RefreshTokenPrefix string
	GrantedScopes      []string
	InstalledBy        uuid.UUID
}

// CreateInstallation records a new app installation for an org (upsert by app+org).
func CreateInstallation(ctx context.Context, pool *pgxpool.Pool, p CreateInstallationParams) (Installation, error) {
	var ins Installation
	err := pool.QueryRow(ctx, `
		INSERT INTO marketplace.installations
		    (app_id, org_id, access_token_hash, access_token_prefix,
		     refresh_token_hash, refresh_token_prefix, granted_scopes, installed_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (app_id, org_id) DO UPDATE
		    SET access_token_hash    = EXCLUDED.access_token_hash,
		        access_token_prefix  = EXCLUDED.access_token_prefix,
		        refresh_token_hash   = EXCLUDED.refresh_token_hash,
		        refresh_token_prefix = EXCLUDED.refresh_token_prefix,
		        granted_scopes       = EXCLUDED.granted_scopes,
		        installed_by         = EXCLUDED.installed_by,
		        installed_at         = now(),
		        revoked_at           = NULL
		RETURNING id, app_id, org_id,
		          (SELECT name FROM marketplace.apps WHERE id = EXCLUDED.app_id),
		          (SELECT slug FROM marketplace.apps WHERE id = EXCLUDED.app_id),
		          (SELECT logo_url FROM marketplace.apps WHERE id = EXCLUDED.app_id),
		          access_token_prefix, granted_scopes, installed_by,
		          installed_at, revoked_at, last_used_at
	`, p.AppID, p.OrgID, p.AccessTokenHash, p.AccessTokenPrefix,
		p.RefreshTokenHash, p.RefreshTokenPrefix, p.GrantedScopes, p.InstalledBy,
	).Scan(
		&ins.ID, &ins.AppID, &ins.OrgID,
		&ins.AppName, &ins.AppSlug, &ins.AppLogoURL,
		&ins.AccessTokenPrefix, &ins.GrantedScopes,
		&ins.InstalledBy, &ins.InstalledAt, &ins.RevokedAt, &ins.LastUsedAt,
	)
	return ins, err
}

// GetInstallationByAccessToken finds a non-revoked installation by bearer token.
func GetInstallationByAccessToken(ctx context.Context, pool *pgxpool.Pool, rawToken string) (*Installation, error) {
	hash := HashToken(rawToken)
	var ins Installation
	err := pool.QueryRow(ctx, `
		SELECT i.id, i.app_id, i.org_id,
		       a.name, a.slug, a.logo_url,
		       i.access_token_prefix, i.granted_scopes,
		       i.installed_by, i.installed_at, i.revoked_at, i.last_used_at
		FROM marketplace.installations i
		JOIN marketplace.apps a ON a.id = i.app_id
		WHERE i.access_token_hash = $1
		  AND i.revoked_at IS NULL
	`, hash).Scan(
		&ins.ID, &ins.AppID, &ins.OrgID,
		&ins.AppName, &ins.AppSlug, &ins.AppLogoURL,
		&ins.AccessTokenPrefix, &ins.GrantedScopes,
		&ins.InstalledBy, &ins.InstalledAt, &ins.RevokedAt, &ins.LastUsedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &ins, err
}

// GetInstallationByRefreshToken finds a non-revoked installation by refresh token.
func GetInstallationByRefreshToken(ctx context.Context, pool *pgxpool.Pool, rawToken string) (*Installation, error) {
	hash := HashToken(rawToken)
	var ins Installation
	err := pool.QueryRow(ctx, `
		SELECT i.id, i.app_id, i.org_id,
		       a.name, a.slug, a.logo_url,
		       i.access_token_prefix, i.granted_scopes,
		       i.installed_by, i.installed_at, i.revoked_at, i.last_used_at
		FROM marketplace.installations i
		JOIN marketplace.apps a ON a.id = i.app_id
		WHERE i.refresh_token_hash = $1
		  AND i.revoked_at IS NULL
	`, hash).Scan(
		&ins.ID, &ins.AppID, &ins.OrgID,
		&ins.AppName, &ins.AppSlug, &ins.AppLogoURL,
		&ins.AccessTokenPrefix, &ins.GrantedScopes,
		&ins.InstalledBy, &ins.InstalledAt, &ins.RevokedAt, &ins.LastUsedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &ins, err
}

// ListInstallationsByOrg returns all active installations for an org.
func ListInstallationsByOrg(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) ([]Installation, error) {
	rows, err := pool.Query(ctx, `
		SELECT i.id, i.app_id, i.org_id,
		       a.name, a.slug, a.logo_url,
		       i.access_token_prefix, i.granted_scopes,
		       i.installed_by, i.installed_at, i.revoked_at, i.last_used_at
		FROM marketplace.installations i
		JOIN marketplace.apps a ON a.id = i.app_id
		WHERE i.org_id = $1
		  AND i.revoked_at IS NULL
		ORDER BY i.installed_at DESC
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInstallations(rows)
}

func scanInstallations(rows pgx.Rows) ([]Installation, error) {
	var out []Installation
	for rows.Next() {
		var ins Installation
		if err := rows.Scan(
			&ins.ID, &ins.AppID, &ins.OrgID,
			&ins.AppName, &ins.AppSlug, &ins.AppLogoURL,
			&ins.AccessTokenPrefix, &ins.GrantedScopes,
			&ins.InstalledBy, &ins.InstalledAt, &ins.RevokedAt, &ins.LastUsedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, ins)
	}
	return out, rows.Err()
}

// RevokeInstallation marks an installation as revoked by ID, checking org ownership.
func RevokeInstallation(ctx context.Context, pool *pgxpool.Pool, id, orgID uuid.UUID) error {
	tag, err := pool.Exec(ctx, `
		UPDATE marketplace.installations
		SET revoked_at = now()
		WHERE id = $1 AND org_id = $2 AND revoked_at IS NULL
	`, id, orgID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RotateTokens replaces the access and refresh tokens for an installation.
func RotateTokens(ctx context.Context, pool *pgxpool.Pool, installationID uuid.UUID,
	newAccessHash, newAccessPrefix, newRefreshHash, newRefreshPrefix string) error {
	_, err := pool.Exec(ctx, `
		UPDATE marketplace.installations
		SET access_token_hash    = $2,
		    access_token_prefix  = $3,
		    refresh_token_hash   = $4,
		    refresh_token_prefix = $5,
		    last_used_at         = now()
		WHERE id = $1 AND revoked_at IS NULL
	`, installationID, newAccessHash, newAccessPrefix, newRefreshHash, newRefreshPrefix)
	return err
}

// TouchLastUsed updates the last_used_at timestamp for an installation.
func TouchLastUsed(ctx context.Context, pool *pgxpool.Pool, installationID uuid.UUID) {
	_, _ = pool.Exec(ctx, `
		UPDATE marketplace.installations
		SET last_used_at = now()
		WHERE id = $1
	`, installationID)
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "unique") || contains(err.Error(), "duplicate")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
