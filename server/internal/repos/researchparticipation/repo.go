// Package researchparticipation stores explicit tenant research decisions.
package researchparticipation

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Participation string

const (
	OptIn  Participation = "opt_in"
	OptOut Participation = "opt_out"
)

func (p Participation) Valid() bool { return p == OptIn || p == OptOut }

type Setting struct {
	OrgID         uuid.UUID
	Participation Participation
	UpdatedBy     uuid.UUID
	UpdatedAt     time.Time
}

func Get(ctx context.Context, pool *pgxpool.Pool, orgID uuid.UUID) (*Setting, error) {
	var s Setting
	err := pool.QueryRow(ctx, `
SELECT org_id, participation, updated_by, updated_at
FROM tenant.research_participation_settings WHERE org_id = $1`, orgID).
		Scan(&s.OrgID, &s.Participation, &s.UpdatedBy, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &s, err
}

func Upsert(ctx context.Context, pool *pgxpool.Pool, orgID, actorID uuid.UUID, p Participation) (*Setting, error) {
	if !p.Valid() {
		return nil, errors.New("invalid research participation value")
	}
	var s Setting
	err := pool.QueryRow(ctx, `
INSERT INTO tenant.research_participation_settings (org_id, participation, updated_by)
VALUES ($1, $2, $3)
ON CONFLICT (org_id) DO UPDATE SET participation = EXCLUDED.participation,
  updated_by = EXCLUDED.updated_by, updated_at = NOW()
RETURNING org_id, participation, updated_by, updated_at`, orgID, p, actorID).
		Scan(&s.OrgID, &s.Participation, &s.UpdatedBy, &s.UpdatedAt)
	return &s, err
}
