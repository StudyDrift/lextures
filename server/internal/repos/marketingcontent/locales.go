package marketingcontent

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func ListLocales(ctx context.Context, q querier, enabledOnly bool) ([]Locale, error) {
	rows, err := q.Query(ctx, `SELECT code,label,is_default,rtl,ts_config,enabled,sort_order,created_at,updated_at
 FROM marketing.content_locales WHERE (NOT $1 OR enabled) ORDER BY sort_order,code`, enabledOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Locale, 0)
	for rows.Next() {
		var l Locale
		if err := rows.Scan(&l.Code, &l.Label, &l.IsDefault, &l.RTL, &l.TSConfig, &l.Enabled, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func GetLocale(ctx context.Context, q querier, code string) (*Locale, error) {
	var l Locale
	err := q.QueryRow(ctx, `SELECT code,label,is_default,rtl,ts_config,enabled,sort_order,created_at,updated_at
 FROM marketing.content_locales WHERE code=$1`, code).Scan(&l.Code, &l.Label, &l.IsDefault, &l.RTL, &l.TSConfig, &l.Enabled, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func UpsertLocale(ctx context.Context, tx pgx.Tx, l Locale) (*Locale, error) {
	if l.TSConfig == "" {
		l.TSConfig = "simple"
	}
	if l.Label == "" {
		l.Label = l.Code
	}
	err := tx.QueryRow(ctx, `INSERT INTO marketing.content_locales (code,label,is_default,rtl,ts_config,enabled,sort_order)
 VALUES ($1,$2,$3,$4,$5,$6,$7)
 ON CONFLICT (code) DO UPDATE SET label=EXCLUDED.label,rtl=EXCLUDED.rtl,ts_config=EXCLUDED.ts_config,enabled=EXCLUDED.enabled,sort_order=EXCLUDED.sort_order
 RETURNING code,label,is_default,rtl,ts_config,enabled,sort_order,created_at,updated_at`,
		l.Code, l.Label, l.IsDefault, l.RTL, l.TSConfig, l.Enabled, l.SortOrder).Scan(
		&l.Code, &l.Label, &l.IsDefault, &l.RTL, &l.TSConfig, &l.Enabled, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt)
	return &l, err
}

func PatchLocale(ctx context.Context, tx pgx.Tx, code string, enabled *bool, sortOrder *int, label *string) (*Locale, error) {
	var l Locale
	err := tx.QueryRow(ctx, `UPDATE marketing.content_locales SET
 enabled=COALESCE($2,enabled), sort_order=COALESCE($3,sort_order), label=COALESCE($4,label)
 WHERE code=$1 RETURNING code,label,is_default,rtl,ts_config,enabled,sort_order,created_at,updated_at`,
		code, enabled, sortOrder, label).Scan(&l.Code, &l.Label, &l.IsDefault, &l.RTL, &l.TSConfig, &l.Enabled, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt)
	return &l, err
}

func LocaleTSConfig(ctx context.Context, q querier, code string) (string, error) {
	var cfg string
	err := q.QueryRow(ctx, `SELECT COALESCE(ts_config,'simple') FROM marketing.content_locales WHERE code=$1`, code).Scan(&cfg)
	if err == pgx.ErrNoRows || strings.TrimSpace(cfg) == "" {
		return "simple", nil
	}
	return cfg, err
}

func LocalesEnabled(ctx context.Context, q querier) (bool, error) {
	var enabled bool
	err := q.QueryRow(ctx, `SELECT COALESCE(locales_enabled,false) FROM marketing.content_editorial_settings WHERE singleton`).Scan(&enabled)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return enabled, err
}

func ResolvePublicLocale(ctx context.Context, q querier, raw string) (string, error) {
	code := NormalizeLocaleCode(raw)
	if code == "" {
		return "", fmt.Errorf("unsupported locale")
	}
	if code == DefaultLocale {
		return DefaultLocale, nil
	}
	on, err := LocalesEnabled(ctx, q)
	if err != nil {
		return "", err
	}
	if !on {
		return DefaultLocale, nil
	}
	l, err := GetLocale(ctx, q, code)
	if err == pgx.ErrNoRows || (err == nil && (!l.Enabled || l.Code != code)) {
		return "", fmt.Errorf("unsupported locale")
	}
	if err != nil {
		return "", err
	}
	return l.Code, nil
}
