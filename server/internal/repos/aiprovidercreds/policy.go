package aiprovidercreds

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantBYOKPolicy controls whether orgs may store BYOK credentials (AP.2 FR-9).
type TenantBYOKPolicy struct {
	Allowed           bool     // false only when explicitly disabled
	AllowedProviders  []string // empty / nil = all providers
}

// GetTenantBYOKPolicy loads platform BYOK policy. Missing row → allow all.
func GetTenantBYOKPolicy(ctx context.Context, pool *pgxpool.Pool) (TenantBYOKPolicy, error) {
	out := TenantBYOKPolicy{Allowed: true}
	if pool == nil {
		return out, nil
	}
	var allowed *bool
	var providers []string
	err := pool.QueryRow(ctx, `
SELECT ai_tenant_byok_allowed, ai_tenant_allowed_providers
  FROM settings.platform_app_settings
 LIMIT 1
`).Scan(&allowed, &providers)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if allowed != nil {
		out.Allowed = *allowed
	}
	out.AllowedProviders = providers
	return out, nil
}

// SetTenantBYOKPolicy updates platform BYOK policy columns.
func SetTenantBYOKPolicy(ctx context.Context, pool *pgxpool.Pool, allowed *bool, providers *[]string) error {
	if pool == nil {
		return errors.New("aiprovidercreds: nil pool")
	}
	_, err := pool.Exec(ctx, `
INSERT INTO settings.platform_app_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING
`)
	if err != nil {
		return err
	}
	if allowed != nil {
		_, err = pool.Exec(ctx, `
UPDATE settings.platform_app_settings
   SET ai_tenant_byok_allowed = $1, updated_at = now()
`, *allowed)
		if err != nil {
			return err
		}
	}
	if providers != nil {
		_, err = pool.Exec(ctx, `
UPDATE settings.platform_app_settings
   SET ai_tenant_allowed_providers = $1, updated_at = now()
`, *providers)
		if err != nil {
			return err
		}
	}
	return nil
}

// AllowTenantProvider reports whether an org may configure the given provider.
func (p TenantBYOKPolicy) AllowTenantProvider(provider string) bool {
	if !p.Allowed {
		return false
	}
	provider = strings.TrimSpace(provider)
	if len(p.AllowedProviders) == 0 {
		return true
	}
	for _, a := range p.AllowedProviders {
		if strings.EqualFold(strings.TrimSpace(a), provider) {
			return true
		}
	}
	return false
}
