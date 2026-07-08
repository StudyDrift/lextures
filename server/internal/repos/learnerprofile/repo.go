package learnerprofile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ValidFacetKeys is the registry of facet keys LP02–LP06 extend.
var ValidFacetKeys = map[string]struct{}{
	"study_rhythm":      {},
	"content_modality":  {},
	"strengths_growth":  {},
	"interests":         {},
	"learning_approach": {},
}

// ErrUnknownFacet is returned when a facet key is not in the registry.
var ErrUnknownFacet = errors.New("learnerprofile: unknown facet key")

// EnsureProfile lazily creates a learner.profiles row for userID.
func EnsureProfile(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, `
INSERT INTO learner.profiles (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET updated_at = now()
RETURNING id
`, userID).Scan(&id)
	return id, err
}

// GetProfileByUserID returns the profile for userID, or nil when absent.
func GetProfileByUserID(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (*Profile, error) {
	row := pool.QueryRow(ctx, `
SELECT id, user_id, status, last_computed_at, created_at, updated_at
FROM learner.profiles
WHERE user_id = $1
`, userID)
	var p Profile
	if err := row.Scan(&p.ID, &p.UserID, &p.Status, &p.LastComputedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// ListFacets returns all facets for a profile ordered by facet_key.
func ListFacets(ctx context.Context, pool *pgxpool.Pool, profileID uuid.UUID) ([]Facet, error) {
	rows, err := pool.Query(ctx, `
SELECT id, profile_id, facet_key, state, summary, confidence, computed_version, updated_at
FROM learner.profile_facets
WHERE profile_id = $1
ORDER BY facet_key
`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Facet
	for rows.Next() {
		var f Facet
		if err := rows.Scan(&f.ID, &f.ProfileID, &f.FacetKey, &f.State, &f.Summary, &f.Confidence, &f.ComputedVersion, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// GetFacet returns one facet for profileID and facetKey.
func GetFacet(ctx context.Context, pool *pgxpool.Pool, profileID uuid.UUID, facetKey string) (*Facet, error) {
	if _, ok := ValidFacetKeys[facetKey]; !ok {
		return nil, ErrUnknownFacet
	}
	row := pool.QueryRow(ctx, `
SELECT id, profile_id, facet_key, state, summary, confidence, computed_version, updated_at
FROM learner.profile_facets
WHERE profile_id = $1 AND facet_key = $2
`, profileID, facetKey)
	var f Facet
	if err := row.Scan(&f.ID, &f.ProfileID, &f.FacetKey, &f.State, &f.Summary, &f.Confidence, &f.ComputedVersion, &f.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

// ListInsights returns insights for a facet ordered by salience desc.
func ListInsights(ctx context.Context, pool *pgxpool.Pool, facetID uuid.UUID) ([]Insight, error) {
	rows, err := pool.Query(ctx, `
SELECT id, facet_id, insight_key, label_i18n_key, value, confidence, salience, created_at
FROM learner.profile_insights
WHERE facet_id = $1
ORDER BY salience DESC, insight_key
`, facetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Insight
	for rows.Next() {
		var ins Insight
		if err := rows.Scan(&ins.ID, &ins.FacetID, &ins.InsightKey, &ins.LabelI18nKey, &ins.Value, &ins.Confidence, &ins.Salience, &ins.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, ins)
	}
	return out, rows.Err()
}

// ListEvidenceForInsights returns evidence grouped by insight id.
func ListEvidenceForInsights(ctx context.Context, pool *pgxpool.Pool, insightIDs []uuid.UUID) (map[uuid.UUID][]Evidence, error) {
	if len(insightIDs) == 0 {
		return map[uuid.UUID][]Evidence{}, nil
	}
	rows, err := pool.Query(ctx, `
SELECT id, insight_id, source_kind, source_table, course_id, observation_count,
       window_start, window_end, contribution, sample_refs, created_at
FROM learner.profile_evidence
WHERE insight_id = ANY($1)
ORDER BY insight_id, created_at
`, insightIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[uuid.UUID][]Evidence)
	for rows.Next() {
		var ev Evidence
		if err := rows.Scan(
			&ev.ID, &ev.InsightID, &ev.SourceKind, &ev.SourceTable, &ev.CourseID,
			&ev.ObservationCount, &ev.WindowStart, &ev.WindowEnd, &ev.Contribution,
			&ev.SampleRefs, &ev.CreatedAt,
		); err != nil {
			return nil, err
		}
		out[ev.InsightID] = append(out[ev.InsightID], ev)
	}
	return out, rows.Err()
}

// WriteFacet atomically upserts a facet and replaces its insights and evidence.
func WriteFacet(ctx context.Context, pool *pgxpool.Pool, profileID uuid.UUID, facetKey string, w FacetWrite) error {
	if _, ok := ValidFacetKeys[facetKey]; !ok {
		return ErrUnknownFacet
	}
	if w.Summary == nil {
		w.Summary = json.RawMessage(`{}`)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var facetID uuid.UUID
	err = tx.QueryRow(ctx, `
INSERT INTO learner.profile_facets (profile_id, facet_key, state, summary, confidence, computed_version, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now())
ON CONFLICT (profile_id, facet_key) DO UPDATE SET
    state = EXCLUDED.state,
    summary = EXCLUDED.summary,
    confidence = EXCLUDED.confidence,
    computed_version = EXCLUDED.computed_version,
    updated_at = now()
RETURNING id
`, profileID, facetKey, w.State, w.Summary, w.Confidence, w.ComputedVersion).Scan(&facetID)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM learner.profile_insights WHERE facet_id = $1`, facetID); err != nil {
		return err
	}

	for _, ins := range w.Insights {
		if len(ins.Evidence) == 0 {
			return fmt.Errorf("learnerprofile: insight %q requires at least one evidence row", ins.InsightKey)
		}
		value := ins.Value
		if value == nil {
			value = json.RawMessage(`{}`)
		}
		var insightID uuid.UUID
		err = tx.QueryRow(ctx, `
INSERT INTO learner.profile_insights (facet_id, insight_key, label_i18n_key, value, confidence, salience)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`, facetID, ins.InsightKey, ins.LabelI18nKey, value, ins.Confidence, ins.Salience).Scan(&insightID)
		if err != nil {
			return err
		}
		for _, ev := range ins.Evidence {
			if _, err := tx.Exec(ctx, `
INSERT INTO learner.profile_evidence (
    insight_id, source_kind, source_table, course_id, observation_count,
    window_start, window_end, contribution, sample_refs
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`, insightID, ev.SourceKind, ev.SourceTable, ev.CourseID, ev.ObservationCount,
				ev.WindowStart, ev.WindowEnd, ev.Contribution, ev.SampleRefs); err != nil {
				return err
			}
		}
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
UPDATE learner.profiles SET last_computed_at = $2, updated_at = $2 WHERE id = $1
`, profileID, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SetProfileStatus updates profile status (active or paused).
func SetProfileStatus(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, status string) error {
	if status != "active" && status != "paused" {
		return fmt.Errorf("learnerprofile: invalid status %q", status)
	}
	tag, err := pool.Exec(ctx, `
UPDATE learner.profiles SET status = $2, updated_at = now() WHERE user_id = $1
`, userID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		_, err = EnsureProfile(ctx, pool, userID)
		if err != nil {
			return err
		}
		_, err = pool.Exec(ctx, `
UPDATE learner.profiles SET status = $2, updated_at = now() WHERE user_id = $1
`, userID, status)
	}
	return err
}

// EraseUser deletes all learner.* rows for userID via ON DELETE CASCADE.
func EraseUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) error {
	_, err := pool.Exec(ctx, `DELETE FROM learner.profiles WHERE user_id = $1`, userID)
	return err
}

// ListActiveUserIDs returns user ids with an active profile or recent learning signals.
func ListActiveUserIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 10000
	}
	rows, err := pool.Query(ctx, `
SELECT user_id FROM (
    SELECT user_id FROM learner.profiles WHERE status = 'active'
    UNION
    SELECT DISTINCT user_id FROM analytics.engagement_events
    WHERE occurred_at > now() - interval '90 days'
) AS active_users
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// CountEngagementEvents counts engagement events for a user (signal probe for derivers).
func CountEngagementEvents(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT count(*)::int FROM analytics.engagement_events WHERE user_id = $1
`, userID).Scan(&n)
	return n, err
}

// PurgeInactiveProfiles deletes learner profiles for users with no engagement
// events within the retention window (LP08 / S02 alignment).
func PurgeInactiveProfiles(ctx context.Context, pool *pgxpool.Pool, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 365
	}
	tag, err := pool.Exec(ctx, `
DELETE FROM learner.profiles p
WHERE p.status = 'active'
  AND NOT EXISTS (
    SELECT 1 FROM analytics.engagement_events e
    WHERE e.user_id = p.user_id
      AND e.occurred_at > now() - ($1::int * interval '1 day')
  )
  AND p.updated_at < now() - ($1::int * interval '1 day')
`, retentionDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// CountProfileRows returns the number of learner.* rows for a user (erasure verification).
func CountProfileRows(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) (int, error) {
	var n int
	err := pool.QueryRow(ctx, `
SELECT (
  (SELECT count(*)::int FROM learner.profiles WHERE user_id = $1) +
  (SELECT count(*)::int FROM learner.profile_facets f
     JOIN learner.profiles p ON p.id = f.profile_id WHERE p.user_id = $1) +
  (SELECT count(*)::int FROM learner.profile_insights i
     JOIN learner.profile_facets f ON f.id = i.facet_id
     JOIN learner.profiles p ON p.id = f.profile_id WHERE p.user_id = $1) +
  (SELECT count(*)::int FROM learner.profile_evidence ev
     JOIN learner.profile_insights i ON i.id = ev.insight_id
     JOIN learner.profile_facets f ON f.id = i.facet_id
     JOIN learner.profiles p ON p.id = f.profile_id WHERE p.user_id = $1)
)::int
`, userID).Scan(&n)
	return n, err
}