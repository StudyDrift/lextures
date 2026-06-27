// Package scormscos provides DB access for content.scorm_scos (plan 2.14).
package scormscos

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SCO is a row from content.scorm_scos.
type SCO struct {
	ID             uuid.UUID
	PackageID      uuid.UUID
	Identifier     string
	Title          string
	LaunchHref     string
	SequencingJSON json.RawMessage
	MasteryScore   *float64
}

// Insert creates a SCO row.
func Insert(ctx context.Context, pool *pgxpool.Pool, id, packageID uuid.UUID, identifier, title, launchHref string, mastery *float64) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO content.scorm_scos (id, package_id, identifier, title, launch_href, mastery_score)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, packageID, identifier, title, launchHref, mastery,
	)
	return err
}

// ListForPackage returns all SCOs for a package.
func ListForPackage(ctx context.Context, pool *pgxpool.Pool, packageID uuid.UUID) ([]SCO, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, package_id, identifier, title, launch_href, sequencing_json, mastery_score
		FROM content.scorm_scos WHERE package_id = $1 ORDER BY title`, packageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SCO
	for rows.Next() {
		var s SCO
		if err := rows.Scan(&s.ID, &s.PackageID, &s.Identifier, &s.Title, &s.LaunchHref, &s.SequencingJSON, &s.MasteryScore); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// LoadByID loads a SCO by id.
func LoadByID(ctx context.Context, pool *pgxpool.Pool, scoID uuid.UUID) (*SCO, error) {
	var s SCO
	err := pool.QueryRow(ctx, `
		SELECT id, package_id, identifier, title, launch_href, sequencing_json, mastery_score
		FROM content.scorm_scos WHERE id = $1`, scoID).Scan(
		&s.ID, &s.PackageID, &s.Identifier, &s.Title, &s.LaunchHref, &s.SequencingJSON, &s.MasteryScore,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
